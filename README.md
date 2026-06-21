# go-probe

A lightweight HTTP endpoint health checker written in Go. Concurrently probes a list of target URLs on a configurable interval and exposes results as Prometheus metrics and a JSON status endpoint.

Built as a learning project while exploring Go — directly motivated by observability work at WebMD involving Grafana dashboards and Kubernetes-based pipelines.

## Live Demo

Deployed on Hugging Face Spaces: **https://kareemadesola-go-probe.hf.space**

- `/metrics` — Prometheus output
- `/status` — JSON health status

## Features

- Concurrent probing via goroutines — all targets are checked simultaneously
- Prometheus `/metrics` endpoint (`probe_up`, `probe_latency_ms`)
- JSON `/status` endpoint with per-target up/down state and latency
- Configurable probe interval, HTTP timeout, and listen address
- Zero external dependencies — pure Go standard library

## Usage

```bash
go build -o go-probe .

# Probe two endpoints every 15 seconds
./go-probe \
  --targets=https://example.com,https://google.com \
  --interval=15s \
  --timeout=5s \
  --addr=:8080
```

### Endpoints

| Path | Description |
|---|---|
| `/metrics` | Prometheus text format — scrape with Prometheus or curl |
| `/status` | JSON summary of all probe results |

### Example `/metrics` output

```
# HELP probe_up Whether the HTTP probe succeeded (1 = up, 0 = down)
# TYPE probe_up gauge
probe_up{url="https://example.com"} 1
probe_up{url="https://google.com"} 1

# HELP probe_latency_ms HTTP probe round-trip latency in milliseconds
# TYPE probe_latency_ms gauge
probe_latency_ms{url="https://example.com"} 142.00
probe_latency_ms{url="https://google.com"} 89.00
```

### Example `/status` output

```json
{
  "all_up": true,
  "targets": [
    {
      "url": "https://example.com",
      "up": true,
      "status_code": 200,
      "latency_ms": 142000000,
      "checked_at": "2026-06-21T10:00:00Z"
    }
  ]
}
```

## What I learned building this

- **Goroutines and `sync.WaitGroup`** — launching all probes concurrently and waiting for all to finish before serving results
- **`sync.RWMutex`** — protecting the shared results map from concurrent reads/writes (multiple goroutines write results; HTTP handlers read them)
- **Go interfaces and structs** — modelling probe results as typed structs rather than raw maps
- **Standard `net/http`** — both as an HTTP client (for probing) and server (for /metrics and /status)
- **Prometheus exposition format** — hand-writing the text format clarified what libraries like `prometheus/client_golang` abstract away

## Planned improvements

- [ ] YAML config file support for more complex target definitions
- [ ] Alert webhook (POST to Slack/PagerDuty when a target goes down)
- [ ] TLS certificate expiry check alongside HTTP health
- [ ] Proper Prometheus client library integration (`prometheus/client_golang`)
- [ ] Docker image + Kubernetes deployment manifest
