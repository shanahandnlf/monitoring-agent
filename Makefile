COMPOSE := docker compose
COMPOSE_FILE := deploy/docker-compose.yml
SINGLE_FILE := deploy/docker-compose.single.yml
FEDERATED_PROJECT := monitoring-poc-federated
SINGLE_PROJECT := monitoring-poc-single

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

.PHONY: help up-single up-federated down-single down-federated down ps-single ps-federated logs-single logs-federated test

help:
	@echo "Available targets:"
	@echo "  make up-single      Run Opsi 1: Single Prometheus + Agent + ELK"
	@echo "  make up-federated   Run Opsi 2: Federated Prometheus + Agent + ELK"
	@echo "  make down-single    Stop Opsi 1 containers"
	@echo "  make down-federated Stop Opsi 2 containers"
	@echo "  make down           Stop both option projects"
	@echo "  make ps-single      Show Opsi 1 container status"
	@echo "  make ps-federated   Show Opsi 2 container status"
	@echo "  make logs-single    Follow Opsi 1 logs"
	@echo "  make logs-federated Follow Opsi 2 logs"
	@echo "  make test           Run Go tests"

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

ps-single:
	$(COMPOSE) -p $(SINGLE_PROJECT) -f $(COMPOSE_FILE) -f $(SINGLE_FILE) ps

ps-federated:
	$(COMPOSE) -p $(FEDERATED_PROJECT) -f $(COMPOSE_FILE) ps

logs-single:
	$(COMPOSE) -p $(SINGLE_PROJECT) -f $(COMPOSE_FILE) -f $(SINGLE_FILE) logs -f

logs-federated:
	$(COMPOSE) -p $(FEDERATED_PROJECT) -f $(COMPOSE_FILE) logs -f

test:
	go test ./...
