#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
DIST_DIR="${ROOT_DIR}/dist"
LINUX_BIN_DIR="${DIST_DIR}/bin/linux-${GOARCH:-amd64}"
WINDOWS_BIN_DIR="${DIST_DIR}/bin/windows-${GOARCH:-amd64}"
IMAGE_TAR="${1:-${DIST_DIR}/monitoring-poc-images.tar}"
GOARCH_VALUE="${GOARCH:-amd64}"

mkdir -p "${LINUX_BIN_DIR}" "${WINDOWS_BIN_DIR}"

echo "Building static Linux binaries..."
CGO_ENABLED=0 GOOS=linux GOARCH="${GOARCH_VALUE}" go build -trimpath -ldflags="-s -w" -o "${LINUX_BIN_DIR}/agent" "${ROOT_DIR}/cmd/agent"
CGO_ENABLED=0 GOOS=linux GOARCH="${GOARCH_VALUE}" go build -trimpath -ldflags="-s -w" -o "${LINUX_BIN_DIR}/demo-app" "${ROOT_DIR}/cmd/demo-app"

echo "Building static Windows binaries..."
CGO_ENABLED=0 GOOS=windows GOARCH="${GOARCH_VALUE}" go build -trimpath -ldflags="-s -w" -o "${WINDOWS_BIN_DIR}/agent.exe" "${ROOT_DIR}/cmd/agent"
CGO_ENABLED=0 GOOS=windows GOARCH="${GOARCH_VALUE}" go build -trimpath -ldflags="-s -w" -o "${WINDOWS_BIN_DIR}/demo-app.exe" "${ROOT_DIR}/cmd/demo-app"

echo "Building local Docker images..."
docker compose -f "${ROOT_DIR}/deploy/docker-compose.yml" build agent-zone-a demo-app-zone-a

echo "Pulling third-party images..."
for image in \
  prom/prometheus:v2.52.0 \
  prom/blackbox-exporter:v0.25.0 \
  grafana/grafana:11.1.0 \
  docker.elastic.co/elasticsearch/elasticsearch:8.15.0 \
  docker.elastic.co/logstash/logstash:8.15.0 \
  docker.elastic.co/kibana/kibana:8.15.0 \
  haproxy:2.9-alpine \
  curlimages/curl:8.9.1
do
  docker pull "${image}"
done

echo "Saving image bundle to ${IMAGE_TAR}..."
docker save \
  monitoring-agent:local \
  monitoring-demo-app:local \
  prom/prometheus:v2.52.0 \
  prom/blackbox-exporter:v0.25.0 \
  grafana/grafana:11.1.0 \
  docker.elastic.co/elasticsearch/elasticsearch:8.15.0 \
  docker.elastic.co/logstash/logstash:8.15.0 \
  docker.elastic.co/kibana/kibana:8.15.0 \
  haproxy:2.9-alpine \
  curlimages/curl:8.9.1 \
  -o "${IMAGE_TAR}"

echo "Offline bundle ready:"
echo "  ${LINUX_BIN_DIR}/agent"
echo "  ${LINUX_BIN_DIR}/demo-app"
echo "  ${WINDOWS_BIN_DIR}/agent.exe"
echo "  ${WINDOWS_BIN_DIR}/demo-app.exe"
echo "  ${IMAGE_TAR}"
