# Observability Lab

A self-hosted, Docker-based observability platform built as a **learnable journey**.
Each level is a self-contained, runnable milestone — stop at any one and you already
have a strong project. The foundation is **OpenTelemetry**: a single instrumentation
standard whose telemetry fans out to open-source and commercial backends without
touching application code.

> **N0 (this level) is done and runs locally with one command.** Higher levels are the
> roadmap — see [The journey](#the-journey).

---

## N0 — Core (current)

A minimal Go service emits **traces, metrics, and logs** over OTLP to an OpenTelemetry
Collector, which routes each signal to its backend. Grafana ships pre-provisioned with
all three datasources wired together (trace ↔ log ↔ metric correlation).

```
                         ┌──────────────► Prometheus  (metrics)
 go-demo ──OTLP──► OTel Collector ───────► Tempo       (traces)
 (Go SDK)                 └──────────────► Loki        (logs)
                                                  │
 (optional) Datadog agent ◄── same OTLP ──┘   Grafana (single pane of glass)
```

The app drives itself with synthetic load, so dashboards populate on boot — no manual
`curl` needed. Tempo's metrics-generator derives RED metrics and service graphs from the
spans, written back to Prometheus.

### Run it

```bash
docker compose up --build
```

| UI / endpoint     | URL                              | Notes                                  |
|-------------------|----------------------------------|----------------------------------------|
| Grafana           | http://localhost:3000            | Anonymous admin; dashboard auto-loaded |
| Prometheus        | http://localhost:9090            | Targets under Status → Targets         |
| Tempo             | http://localhost:3200            | Query via Grafana Explore (TraceQL)    |
| Loki              | http://localhost:3100            | Query via Grafana Explore (LogQL)      |
| Demo app          | http://localhost:8080/work       | Manual request; `/healthz` for probe   |

Open **Grafana → Dashboards → "Observability Lab — N0 Overview"** for request rate,
p95 span latency, and live logs. In **Explore**, jump from a Tempo trace straight to its
logs in Loki (correlation is provisioned).

### Optional: Datadog

Prove the same OTLP telemetry reaches a commercial backend with zero app changes:

```bash
cp .env.example .env   # set DD_API_KEY
```

Then add a `datadog` exporter to `otel-collector/config.yaml` and uncomment the
`datadog-agent` service in `docker-compose.yml`.

---

## The journey

| Level | Theme              | Adds                                                                 |
|-------|--------------------|----------------------------------------------------------------------|
| **N0** | Core              | OTel Collector · Prometheus · Grafana · Loki · Tempo · (Datadog)     |
| N1     | Tracing & alerting | Jaeger/Zipkin · AlertManager + PagerDuty/OpsGenie · Blackbox exporter |
| N2     | Distributed system | Kafka/Redpanda · RabbitMQ · Redis · Postgres exporters · resilience4j |
| N3     | Logs & data        | Elastic/Kibana · Vector · Neo4j (service dependency graph)            |
| N4     | K8s & SRE          | k3s · kube-state-metrics/node-exporter/cAdvisor · Helm · Sloth (SLOs) |
| N5     | Advanced           | Service mesh (Istio/Linkerd) · eBPF (Beyla) · Pyroscope · chaos eng.  |
| N6     | GitOps & IaC       | Terraform · Ansible · ArgoCD · Vault · Keycloak (OIDC on Grafana)     |

Planned: extend the demo into a **polyglot** trio (Go + Java + Python) to showcase
OpenTelemetry's cross-language story.

Progress is tracked as a board in **[ROADMAP.md](ROADMAP.md)** — per-level checklists
for everything above.

---

## Layout

```
docker-compose.yml          # the whole N0 stack
otel-collector/config.yaml  # OTLP in; Prometheus/Tempo/Loki out
prometheus/prometheus.yml    # scrape config + remote-write receiver
tempo/tempo.yaml             # traces + metrics-generator (RED, service graph)
loki/loki-config.yaml        # single-binary Loki with native OTLP ingest
grafana/provisioning/        # datasources (correlated) + dashboard provider
grafana/dashboards/          # N0 overview dashboard
apps/go-demo/                # instrumented Go service (traces + metrics + logs)
```

## Tech proven at N0

OpenTelemetry · OTel Collector · Prometheus · Grafana · Loki · Tempo · Docker Compose ·
Go · (optional) Datadog.
