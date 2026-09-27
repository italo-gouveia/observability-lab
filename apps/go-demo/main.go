// Command go-demo is a minimal HTTP service fully instrumented with
// OpenTelemetry (traces, metrics, logs) exported over OTLP to the collector.
// It also drives itself with synthetic load so dashboards have data on boot.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const serviceName = "go-demo"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	shutdown, err := setupOTel(ctx)
	if err != nil {
		slog.Error("otel setup failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			slog.Error("otel shutdown failed", "error", err)
		}
	}()

	logger := otelslog.NewLogger(serviceName)
	srv := newServer(logger)

	go generateLoad(ctx, logger)

	go func() {
		logger.Info("server starting", "addr", ":8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func newServer(logger *slog.Logger) *http.Server {
	meter := otel.Meter(serviceName)
	requests, _ := meter.Int64Counter(
		"demo.requests",
		metric.WithDescription("Total demo requests handled"),
	)

	ordersURL := getenv("ORDERS_URL", "http://java-orders:8081")
	// otelhttp transport injects the W3C traceparent so the trace continues
	// into java-orders (and on to python-pricing).
	ordersClient := &http.Client{
		Timeout:   5 * time.Second,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	mux := http.NewServeMux()
	mux.Handle("/work", otelhttp.NewHandler(
		http.HandlerFunc(workHandler(ordersClient, ordersURL, requests, logger)), "work",
	))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return &http.Server{Addr: ":8080", Handler: mux}
}

func workHandler(orders *http.Client, ordersURL string, requests metric.Int64Counter, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context() // otelhttp already opened the "work" server span

		status := callOrders(ctx, orders, ordersURL, logger)

		requests.Add(ctx, 1, metric.WithAttributes(
			attribute.String("route", "/work"),
			attribute.Int("status", status),
		))
		logger.InfoContext(ctx, "handled work request", "status", status)
		w.WriteHeader(status)
		fmt.Fprintf(w, "done with status %d\n", status)
	}
}

// callOrders calls java-orders and maps the outcome to a status code. A downstream
// failure surfaces as 502 so error traces span all three services.
func callOrders(ctx context.Context, client *http.Client, baseURL string, logger *slog.Logger) int {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/orders", nil)
	if err != nil {
		return http.StatusInternalServerError
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.WarnContext(ctx, "orders call failed", "error", err)
		return http.StatusBadGateway
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 500 {
		return http.StatusBadGateway
	}
	return http.StatusOK
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func generateLoad(ctx context.Context, logger *slog.Logger) {
	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			resp, err := client.Get("http://localhost:8080/work")
			if err != nil {
				logger.Warn("self-load request failed", "error", err)
				continue
			}
			resp.Body.Close()
		}
	}
}

func setupOTel(ctx context.Context) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithAttributes(),
	)
	if err != nil {
		return nil, err
	}

	traceExp, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, err
	}
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)
	// W3C propagation so outbound calls carry the trace to java-orders/python-pricing.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	metricExp, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		return nil, err
	}
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp,
			sdkmetric.WithInterval(10*time.Second))),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)

	logExp, err := otlploggrpc.New(ctx)
	if err != nil {
		return nil, err
	}
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(loggerProvider)

	return func(c context.Context) error {
		return errors.Join(
			tracerProvider.Shutdown(c),
			meterProvider.Shutdown(c),
			loggerProvider.Shutdown(c),
		)
	}, nil
}
