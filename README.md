# Segmented Zone Monitoring POC

POC lokal untuk monitoring multi-zone dengan Docker Compose.

Stack ini mensimulasikan:

- `Zone A` dan `Zone B`
- `Core Monitoring`
- HAProxy sebagai mock BIG-IP
- Prometheus per-zone dan central Prometheus
- Grafana dashboard
- Blackbox exporter untuk availability check
- Elasticsearch, Logstash, Kibana untuk demo application logs
- Go agent untuk infrastructure metrics
- Go demo app untuk application metrics dan logs

## Prerequisites

Pastikan sudah tersedia:

- Go
- Docker Desktop
- Docker Compose


## Cara Menjalankan

Dari root repo, jalankan:

```bash
docker compose -f deploy/docker-compose.yml up --build
```

Tunggu beberapa menit sampai semua service up

Cek status container:

```bash
docker compose -f deploy/docker-compose.yml ps
```

Jalan di background:

```bash
docker compose -f deploy/docker-compose.yml up --build -d
```

## URL Akses

| Komponen | URL | Keterangan |
| --- | --- | --- |
| Grafana | http://localhost:3000 | Dashboard utama |
| Central Prometheus | http://localhost:9090 | Metrics hasil federation |
| Zone A Prometheus | http://localhost:9091 | Metrics lokal Zone A |
| Zone B Prometheus | http://localhost:9092 | Metrics lokal Zone B |
| HAProxy mock BIG-IP | http://localhost:8404 | Status proxy antar-zone |
| Elasticsearch | http://localhost:9200 | Storage demo application logs |
| Kibana | http://localhost:5601 | Opsional untuk cek logs |

Login Grafana:

```text
username: admin
password: admin
```

## Cara Melihat Dashboard

1. Buka Grafana:

   ```text
   http://localhost:3000
   ```

2. Login dengan `admin` / `admin`.

3. Buka menu `Dashboards`.

4. Pilih folder:

   ```text
   Monitoring POC
   ```

5. Buka dashboard:

   ```text
   Segmented Zone Monitoring POC
   ```

Dashboard ini menampilkan:

- Infrastructure metrics: CPU, memory, network
- Availability monitoring: HTTP, TCP, ICMP probe status
- Application metrics: request rate, error rate, latency
- Application logs: log aplikasi dari Elasticsearch

## Query Prometheus Cepat

Buka Central Prometheus:

```text
http://localhost:9090
```

Coba query berikut:

```promql
system_cpu_usage_percent
system_memory_usage_percent
probe_success
demo_app_requests_total
demo_app_errors_total
demo_app_request_duration_seconds_count
```

Query ringkas per zone:

```promql
sum by (zone) (up)
sum by (zone, probe_type) (probe_success)
sum by (zone, service) (rate(demo_app_requests_total[1m]))
```

## Cara Cek Logs

Cek logs aplikasi dari Elasticsearch:

```bash
curl "http://localhost:9200/demo-app-logs-*/_search?pretty&size=5"
```

Kalau data belum muncul, tunggu 1-2 menit. Demo app akan membuat traffic otomatis lewat service `traffic-zone-a` dan `traffic-zone-b`.


## Cara Stop

Stop semua service:

```bash
docker compose -f deploy/docker-compose.yml down
```

Stop dan hapus volume data lokal:

```bash
docker compose -f deploy/docker-compose.yml down -v
```

## Build Binary

Kalau hanya ingin build binary Go:

```bash
go test ./...
CGO_ENABLED=0 GOOS=linux go build -o bin/agent ./cmd/agent
CGO_ENABLED=0 GOOS=linux go build -o bin/demo-app ./cmd/demo-app
```

