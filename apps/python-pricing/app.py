"""python-pricing — the tail of the chain.

A tiny FastAPI service that returns a price for an order. OpenTelemetry is wired
explicitly (TracerProvider + OTLP exporter + W3C propagator + FastAPIInstrumentor)
so the incoming trace context from java-orders is reliably extracted and this
service's spans join the same distributed trace. Logs are also exported over OTLP.
"""

import logging
import random
import time

from fastapi import FastAPI, HTTPException

from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc._log_exporter import OTLPLogExporter
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.propagate import set_global_textmap
from opentelemetry.sdk._logs import LoggerProvider, LoggingHandler
from opentelemetry.sdk._logs.export import BatchLogRecordProcessor
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator

_resource = Resource.create()  # reads OTEL_SERVICE_NAME / OTEL_RESOURCE_ATTRIBUTES

_tracer_provider = TracerProvider(resource=_resource)
_tracer_provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter()))
trace.set_tracer_provider(_tracer_provider)
set_global_textmap(TraceContextTextMapPropagator())

_logger_provider = LoggerProvider(resource=_resource)
_logger_provider.add_log_record_processor(BatchLogRecordProcessor(OTLPLogExporter()))
logging.basicConfig(level=logging.INFO)
logging.getLogger().addHandler(LoggingHandler(logger_provider=_logger_provider))
logger = logging.getLogger("python-pricing")

app = FastAPI(title="python-pricing")
FastAPIInstrumentor.instrument_app(app)  # SERVER span + remote-context extraction


@app.get("/healthz")
def healthz():
    return {"status": "ok"}


@app.get("/price")
def price(sku: str = "SKU-1"):
    # Simulate a pricing computation.
    time.sleep(random.uniform(0.01, 0.08))

    # ~8% synthetic failures so error traces span go-demo -> java-orders -> here.
    if random.random() < 0.08:
        logger.warning("pricing engine unavailable for sku=%s", sku)
        raise HTTPException(status_code=500, detail="pricing engine unavailable")

    amount = round(random.uniform(10, 500), 2)
    logger.info("priced sku=%s amount=%s", sku, amount)
    return {"sku": sku, "amount": amount, "currency": "USD"}
