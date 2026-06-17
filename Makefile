COMPOSE := docker compose
COMPOSE_FILE := deploy/docker-compose.yml
SINGLE_FILE := deploy/docker-compose.single.yml
ERRORS_FILE := deploy/docker-compose.errors.yml
FEDERATED_PROJECT := monitoring-poc-federated
SINGLE_PROJECT := monitoring-poc-single
DEMO_SERVICES := demo-app-zone-a demo-app-zone-b

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

.PHONY: help build build-linux build-windows up-single up-federated down-single down-federated down reset-federated reset-single ps-single ps-federated logs-single logs-federated errors-on errors-off errors-on-federated errors-off-federated errors-on-single errors-off-single test

help:
	@echo "Available targets:"
	@echo "  make build          Build local host binaries"
	@echo "  make build-linux    Cross-compile static linux/amd64 binaries"
	@echo "  make build-windows  Cross-compile windows/amd64 binaries"
	@echo "  make up-single      Run Opsi 1: Single Prometheus + Agent + ELK"
	@echo "  make up-federated   Run Opsi 2: Federated Prometheus + Agent + ELK"
	@echo "  make down-single    Stop Opsi 1 containers"
	@echo "  make down-federated Stop Opsi 2 containers"
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
	@echo "  make test           Run Go tests"

build:
	mkdir -p bin
	go build -o bin/agent ./cmd/agent
	go build -o bin/demo-app ./cmd/demo-app

build-linux:
	mkdir -p bin/linux-amd64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/linux-amd64/agent ./cmd/agent
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/linux-amd64/demo-app ./cmd/demo-app

build-windows:
	mkdir -p bin/windows-amd64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/windows-amd64/agent.exe ./cmd/agent
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/windows-amd64/demo-app.exe ./cmd/demo-app

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

test:
	go test ./...
