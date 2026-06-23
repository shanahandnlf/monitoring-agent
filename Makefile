COMPOSE := docker compose
# Build agents with Go 1.20 (last toolchain that runs on Server 2008 R2 / Win7).
# Install: go install golang.org/dl/go1.20@latest && go1.20 download
GO ?= go1.20
export GOTOOLCHAIN := local
DOCKER ?= docker
AGENT_URL ?= http://host.docker.internal:9100/metrics
DEMO_URL ?= http://host.docker.internal:8080/api/demo
LOG_RATE ?= 200
SCALE ?= 20
BENCH_PROJECT := monitoring-bench
BENCH_FILE := deploy/docker-compose.bench.yml
EVENTLOG_FILE := deploy/docker-compose.eventlog.yml
DYNAMIC_FILE := deploy/docker-compose.dynamic.yml
COMPOSE_FILE := deploy/docker-compose.yml
SINGLE_FILE := deploy/docker-compose.single.yml
ERRORS_FILE := deploy/docker-compose.errors.yml
DATADOG_FILE := deploy/docker-compose.datadog.yml
TRACING_FILE := deploy/docker-compose.tracing.yml
OLLAMA_FILE := deploy/docker-compose.ollama.yml
FEDERATED_PROJECT := monitoring-poc-federated
SINGLE_PROJECT := monitoring-poc-single
DATADOG_PROJECT := monitoring-poc-datadog
TRACING_PROJECT := monitoring-poc-tracing
OLLAMA_PROJECT := monitoring-poc-ollama
ALL_PROJECT := monitoring-poc-all
DEMO_SERVICES := demo-app-zone-a demo-app-zone-b

DATADOG_SERVICES := \
	agent-zone-a \
	agent-zone-b \
	demo-app-zone-a \
	demo-app-zone-b \
	traffic-zone-a \
	traffic-zone-b \
	datadog-agent-zone-a \
	datadog-agent-zone-b

SINGLE_SERVICES := \
	agent-zone-a \
	agent-zone-b \
	demo-app-zone-a \
	demo-app-zone-b \
	traffic-zone-a \
	traffic-zone-b \
	elasticsearch \
	logstash \
	kibana \
	bigip-mock \
	blackbox-central \
	prometheus-central \
	grafana

.PHONY: help build build-linux build-windows build-agent-2008 bench-agent bench-k6-agent bench-k6-logs bench-fleet bench-fleet-down bench-scale eventlog-demo eventlog-demo-down up-single up-federated up-datadog up-tracing up-ollama up-all down-single down-federated down-datadog down-tracing down-ollama down-all down reset-federated reset-single reset-datadog reset-tracing reset-ollama reset-all ps-single ps-federated ps-datadog ps-tracing ps-ollama ps-all logs-single logs-federated logs-datadog logs-tracing logs-ollama logs-all errors-on errors-off errors-on-federated errors-off-federated errors-on-single errors-off-single errors-on-all errors-off-all test

help:
	@echo "Available targets:"
	@echo "  make build           Build local host binaries (Go 1.20)"
	@echo "  make build-linux     Cross-compile static linux/amd64 binaries (Go 1.20)"
	@echo "  make build-windows   Cross-compile windows/amd64 binaries (Go 1.20; runs on Server 2008 R2+)"
	@echo "  make build-agent-2008 Build legacy agent for Windows Server 2008 non-R2 (Go 1.10.8 via Docker)"
	@echo "  make bench-agent     Go micro-benchmarks: per-scrape cost (ns/op, allocs/op)"
	@echo "  make bench-k6-agent  k6 load test agent /metrics endpoint (latency p95/p99, RPS)"
	@echo "  make bench-k6-logs   k6 traffic to demo-app /api/demo to load the ELK log pipeline"
	@echo "  make bench-fleet     Avalanche: simulate ~100k series (500 servers) into a bench Prometheus"
	@echo "  make bench-fleet-down Stop+clean the Avalanche bench stack"
	@echo "  make bench-scale     Scale real demo-app replicas as a cross-check (SCALE=20)"
	@echo "  make eventlog-demo   Ship synthetic Windows Event Log records into ELK (needs a running stack)"
	@echo "  make eventlog-demo-down Stop the Windows Event Log demo shipper"
	@echo "  make up-single      Run Opsi 1: Single Prometheus + Agent + ELK"
	@echo "  make up-federated   Run Opsi 2: Federated Prometheus + Agent + ELK"
	@echo "  make up-datadog     Run Opsi 3: Datadog Agent (free tier, butuh DD_API_KEY di .env)"
	@echo "  make up-tracing     Run Opsi 1/2 + Grafana Tempo (distributed tracing on-prem)"
	@echo "  make up-ollama      Run Opsi AI: stack + Ollama + insight (ringkasan log on-prem)"
	@echo "  make up-all         Run semua: stack + Tempo (tracing) + Ollama + insight"
	@echo "  make down-single    Stop Opsi 1 containers"
	@echo "  make down-federated Stop Opsi 2 containers"
	@echo "  make down-all       Stop stack + Tempo + Ollama containers"
	@echo "  make down           Stop both option projects"
	@echo "  make reset-federated Stop Opsi 2 + hapus semua volumes (fresh start)"
	@echo "  make reset-single   Stop Opsi 1 + hapus semua volumes (fresh start)"
	@echo "  make ps-single      Show Opsi 1 container status"
	@echo "  make ps-federated   Show Opsi 2 container status"
	@echo "  make logs-single    Follow Opsi 1 logs"
	@echo "  make logs-federated Follow Opsi 2 logs"
	@echo "  make errors-on      Enable synthetic 5xx errors for manual docker compose project"
	@echo "  make errors-off     Disable synthetic 5xx errors for manual docker compose project"
	@echo "  make errors-on-federated / errors-off-federated"
	@echo "  make errors-on-single / errors-off-single"
	@echo "  make errors-on-all / errors-off-all"
	@echo "  make test           Run Go tests"

build:
	mkdir -p bin
	$(GO) build -o bin/agent ./cmd/agent
	$(GO) build -o bin/demo-app ./cmd/demo-app
	$(GO) build -o bin/insight ./cmd/insight

build-linux:
	mkdir -p bin/linux-amd64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -trimpath -ldflags="-s -w" -o bin/linux-amd64/agent ./cmd/agent
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -trimpath -ldflags="-s -w" -o bin/linux-amd64/demo-app ./cmd/demo-app
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -trimpath -ldflags="-s -w" -o bin/linux-amd64/insight ./cmd/insight

build-windows:
	mkdir -p bin/windows-amd64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -trimpath -ldflags="-s -w" -o bin/windows-amd64/agent.exe ./cmd/agent
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -trimpath -ldflags="-s -w" -o bin/windows-amd64/demo-app.exe ./cmd/demo-app

# Legacy agent for Windows Server 2008 non-R2 (Go 1.10.8 via Docker, stdlib-only).
build-agent-2008:
	mkdir -p bin/windows-2008
	$(DOCKER) run --rm --platform linux/amd64 -v "$(CURDIR)/agent-2008":/src -w /src \
		-e GO111MODULE=off -e GOOS=windows -e GOARCH=386 \
		golang:1.10.8 go build -ldflags="-s -w" -o /src/build/agent-2008-win32.exe .
	$(DOCKER) run --rm --platform linux/amd64 -v "$(CURDIR)/agent-2008":/src -w /src \
		-e GO111MODULE=off -e GOOS=windows -e GOARCH=amd64 \
		golang:1.10.8 go build -ldflags="-s -w" -o /src/build/agent-2008-win64.exe .
	mv agent-2008/build/agent-2008-win32.exe bin/windows-2008/
	mv agent-2008/build/agent-2008-win64.exe bin/windows-2008/
	rmdir agent-2008/build 2>/dev/null || true

bench-agent:
	$(GO) test ./internal/collector/ -bench . -benchmem -run '^$$'

bench-k6-agent:
	$(DOCKER) run --rm -i --add-host=host.docker.internal:host-gateway \
		-v "$(CURDIR)/bench/k6":/scripts -e AGENT_URL=$(AGENT_URL) \
		grafana/k6 run /scripts/agent_metrics.js

bench-k6-logs:
	$(DOCKER) run --rm -i --add-host=host.docker.internal:host-gateway \
		-v "$(CURDIR)/bench/k6":/scripts -e DEMO_URL=$(DEMO_URL) -e RATE=$(LOG_RATE) \
		grafana/k6 run /scripts/demo_app_traffic.js

bench-fleet:
	$(COMPOSE) -p $(BENCH_PROJECT) -f $(BENCH_FILE) up -d
	@echo "Bench Prometheus: http://localhost:9094 (query: prometheus_tsdb_head_series)"

bench-fleet-down:
	$(COMPOSE) -p $(BENCH_PROJECT) -f $(BENCH_FILE) down -v

bench-scale:
	$(COMPOSE) -p monitoring-poc-dynamic -f $(COMPOSE_FILE) -f $(DYNAMIC_FILE) \
		up -d --scale demo-app-zone-a=$(SCALE) --no-recreate

eventlog-demo:
	$(COMPOSE) -p $(FEDERATED_PROJECT) -f $(COMPOSE_FILE) -f $(EVENTLOG_FILE) up -d filebeat-eventlog-demo
	@echo "Synthetic Windows events shipped. Check: curl 'localhost:9200/_cat/indices/windows-eventlog-*?v'"

eventlog-demo-down:
	$(COMPOSE) -p $(FEDERATED_PROJECT) -f $(COMPOSE_FILE) -f $(EVENTLOG_FILE) rm -sf filebeat-eventlog-demo

up-single:
	$(COMPOSE) -p $(SINGLE_PROJECT) -f $(COMPOSE_FILE) -f $(SINGLE_FILE) up --build --no-deps $(SINGLE_SERVICES)

up-federated:
	$(COMPOSE) -p $(FEDERATED_PROJECT) -f $(COMPOSE_FILE) up --build

down-single:
	$(COMPOSE) -p $(SINGLE_PROJECT) -f $(COMPOSE_FILE) -f $(SINGLE_FILE) down

down-federated:
	$(COMPOSE) -p $(FEDERATED_PROJECT) -f $(COMPOSE_FILE) down

down:
	$(COMPOSE) -p $(SINGLE_PROJECT) -f $(COMPOSE_FILE) -f $(SINGLE_FILE) down
	$(COMPOSE) -p $(FEDERATED_PROJECT) -f $(COMPOSE_FILE) down

reset-federated:
	$(COMPOSE) -p $(FEDERATED_PROJECT) -f $(COMPOSE_FILE) down -v

reset-single:
	$(COMPOSE) -p $(SINGLE_PROJECT) -f $(COMPOSE_FILE) -f $(SINGLE_FILE) down -v

ps-single:
	$(COMPOSE) -p $(SINGLE_PROJECT) -f $(COMPOSE_FILE) -f $(SINGLE_FILE) ps

ps-federated:
	$(COMPOSE) -p $(FEDERATED_PROJECT) -f $(COMPOSE_FILE) ps

logs-single:
	$(COMPOSE) -p $(SINGLE_PROJECT) -f $(COMPOSE_FILE) -f $(SINGLE_FILE) logs -f

logs-federated:
	$(COMPOSE) -p $(FEDERATED_PROJECT) -f $(COMPOSE_FILE) logs -f

errors-on:
	$(COMPOSE) -f $(COMPOSE_FILE) -f $(ERRORS_FILE) up -d --no-deps $(DEMO_SERVICES)

errors-off:
	$(COMPOSE) -f $(COMPOSE_FILE) up -d --no-deps $(DEMO_SERVICES)

errors-on-federated:
	$(COMPOSE) -p $(FEDERATED_PROJECT) -f $(COMPOSE_FILE) -f $(ERRORS_FILE) up -d --no-deps $(DEMO_SERVICES)

errors-off-federated:
	$(COMPOSE) -p $(FEDERATED_PROJECT) -f $(COMPOSE_FILE) up -d --no-deps $(DEMO_SERVICES)

errors-on-single:
	$(COMPOSE) -p $(SINGLE_PROJECT) -f $(COMPOSE_FILE) -f $(SINGLE_FILE) -f $(ERRORS_FILE) up -d --no-deps $(DEMO_SERVICES)

errors-off-single:
	$(COMPOSE) -p $(SINGLE_PROJECT) -f $(COMPOSE_FILE) -f $(SINGLE_FILE) up -d --no-deps $(DEMO_SERVICES)

up-datadog:
	$(COMPOSE) --env-file .env -p $(DATADOG_PROJECT) -f $(COMPOSE_FILE) -f $(DATADOG_FILE) up --build --no-deps $(DATADOG_SERVICES)

down-datadog:
	$(COMPOSE) --env-file .env -p $(DATADOG_PROJECT) -f $(COMPOSE_FILE) -f $(DATADOG_FILE) down

reset-datadog:
	$(COMPOSE) --env-file .env -p $(DATADOG_PROJECT) -f $(COMPOSE_FILE) -f $(DATADOG_FILE) down -v

ps-datadog:
	$(COMPOSE) --env-file .env -p $(DATADOG_PROJECT) -f $(COMPOSE_FILE) -f $(DATADOG_FILE) ps

logs-datadog:
	$(COMPOSE) --env-file .env -p $(DATADOG_PROJECT) -f $(COMPOSE_FILE) -f $(DATADOG_FILE) logs -f

up-tracing:
	$(COMPOSE) -p $(TRACING_PROJECT) -f $(COMPOSE_FILE) -f $(TRACING_FILE) up --build

down-tracing:
	$(COMPOSE) -p $(TRACING_PROJECT) -f $(COMPOSE_FILE) -f $(TRACING_FILE) down

reset-tracing:
	$(COMPOSE) -p $(TRACING_PROJECT) -f $(COMPOSE_FILE) -f $(TRACING_FILE) down -v

ps-tracing:
	$(COMPOSE) -p $(TRACING_PROJECT) -f $(COMPOSE_FILE) -f $(TRACING_FILE) ps

logs-tracing:
	$(COMPOSE) -p $(TRACING_PROJECT) -f $(COMPOSE_FILE) -f $(TRACING_FILE) logs -f

up-ollama:
	$(COMPOSE) --env-file .env -p $(OLLAMA_PROJECT) -f $(COMPOSE_FILE) -f $(OLLAMA_FILE) up --build

up-all:
	$(COMPOSE) --env-file .env -p $(ALL_PROJECT) -f $(COMPOSE_FILE) -f $(TRACING_FILE) -f $(OLLAMA_FILE) up --build

down-ollama:
	$(COMPOSE) --env-file .env -p $(OLLAMA_PROJECT) -f $(COMPOSE_FILE) -f $(OLLAMA_FILE) down

down-all:
	$(COMPOSE) --env-file .env -p $(ALL_PROJECT) -f $(COMPOSE_FILE) -f $(TRACING_FILE) -f $(OLLAMA_FILE) down

reset-ollama:
	$(COMPOSE) --env-file .env -p $(OLLAMA_PROJECT) -f $(COMPOSE_FILE) -f $(OLLAMA_FILE) down -v

reset-all:
	$(COMPOSE) --env-file .env -p $(ALL_PROJECT) -f $(COMPOSE_FILE) -f $(TRACING_FILE) -f $(OLLAMA_FILE) down -v

ps-ollama:
	$(COMPOSE) --env-file .env -p $(OLLAMA_PROJECT) -f $(COMPOSE_FILE) -f $(OLLAMA_FILE) ps

ps-all:
	$(COMPOSE) --env-file .env -p $(ALL_PROJECT) -f $(COMPOSE_FILE) -f $(TRACING_FILE) -f $(OLLAMA_FILE) ps

logs-ollama:
	$(COMPOSE) --env-file .env -p $(OLLAMA_PROJECT) -f $(COMPOSE_FILE) -f $(OLLAMA_FILE) logs -f

logs-all:
	$(COMPOSE) --env-file .env -p $(ALL_PROJECT) -f $(COMPOSE_FILE) -f $(TRACING_FILE) -f $(OLLAMA_FILE) logs -f

errors-on-all:
	$(COMPOSE) --env-file .env -p $(ALL_PROJECT) -f $(COMPOSE_FILE) -f $(TRACING_FILE) -f $(OLLAMA_FILE) -f $(ERRORS_FILE) up -d --no-deps $(DEMO_SERVICES)

errors-off-all:
	$(COMPOSE) --env-file .env -p $(ALL_PROJECT) -f $(COMPOSE_FILE) -f $(TRACING_FILE) -f $(OLLAMA_FILE) up -d --no-deps $(DEMO_SERVICES)

test:
	$(GO) test ./...
