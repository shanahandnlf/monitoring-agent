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

Project ini punya tiga mode sesuai opsi arsitektur:

- Opsi 1: `Single Prometheus + Agent + ELK`
- Opsi 2: `Federated Prometheus + Agent + ELK`
- Opsi 3: `Datadog (SaaS)` — metrics + logs + APM ke cloud Datadog (butuh `DD_API_KEY`)

Semua memakai port host yang sama (`3000`, `9090`, `9200`, dan lainnya), jadi jalankan salah satu opsi saja dalam satu waktu.

Pilih salah satu cara jalan:

| Cara jalan | Command start | Command stop | Toggle error |
| --- | --- | --- | --- |
| Federated via Makefile | `make up-federated` | `make down-federated` | `make errors-on-federated` / `make errors-off-federated` |
| Single via Makefile | `make up-single` | `make down-single` | `make errors-on-single` / `make errors-off-single` |
| Datadog via Makefile | `make up-datadog` | `make down-datadog` | - |
| Manual Docker Compose | `docker compose -f deploy/docker-compose.yml up --build` | `docker compose -f deploy/docker-compose.yml down` | `make errors-on` / `make errors-off` |

### Opsi 1 - Single Prometheus

Mode ini menjalankan satu Prometheus central yang scrape agent dan demo app lintas zona melalui HAProxy mock BIG-IP.

```bash
make up-single
```

Stop Opsi 1:

```bash
make down-single
```

### Opsi 2 - Federated Prometheus

Mode ini menjalankan Prometheus per-zone, lalu Prometheus central mengambil metrik dari setiap zone Prometheus melalui endpoint federation.

```bash
make up-federated
```

Stop Opsi 2:

```bash
make down-federated
```

### Opsi 3 - Datadog (SaaS)

Mode ini mengganti paradigma dari self-hosted (Opsi 1/2) menjadi SaaS: Datadog Agent berjalan per-zona, mengumpulkan metrik (via OpenMetrics), log aplikasi, dan trace (APM via OTLP), lalu mem-push semuanya ke cloud Datadog. Tidak ada Prometheus/ELK pada opsi ini.

Beda mendasar dengan Opsi 1/2:

- Data **tersimpan di cloud Datadog** (region lewat `DD_SITE`), bukan on-prem. Tidak ada opsi penyimpanan on-premise, jadi opsi ini **tidak cocok untuk server tanpa internet**.
- **Logs dan APM** hanya aktif saat **free trial** atau **Paid Plan**.

Prasyarat: akun Datadog + API key. Isi di `.env`:

```bash
cp .env.example .env
```

```dotenv
DD_API_KEY=<api-key-kamu>
DD_SITE=us5.datadoghq.com
```

> `DD_SITE` harus cocok dengan region akunmu (mis. `datadoghq.com`, `us5.datadoghq.com`, `ap1.datadoghq.com`), kalau tidak data tidak masuk.

Jalankan:

```bash
make up-datadog
make ps-datadog
make logs-datadog
make down-datadog
```

Cek di Datadog (sesuai region `DD_SITE`):

- **Infrastructure > Host Map**: `node-zone-a`, `node-zone-b`
- **Metrics > Explorer**: `poc.system_cpu_usage_percent`, filter `zone:zone-a`
- **Logs > Search**: `service:payments-api`
- **APM > Traces / Services**: filter `env:poc`

Detail lengkap (limitasi free tier, on-premise, setup APM/tracing) ada di [docs/opsi-3-datadog.md](docs/opsi-3-datadog.md) dan [docs/opsi-tracing.md](docs/opsi-tracing.md).

Stop semua project opsi:

```bash
make down
```

### Manual Docker Compose Opsi 2

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

## Screenshots

### Grafana Dashboard

URL:

```text
http://localhost:3000
```

File screenshot:

```text
docs/screenshots/grafana-dashboard.png
```

![alt text](docs/screenshots/grafana-dashboard.png)

Dashboard Grafana menjadi tampilan utama untuk monitoring. Screenshot ini menunjukkan status target per zone, CPU, memory, availability probe, network throughput, application request dari Elasticsearch, latency p95, dan log aplikasi.

### Zone A Prometheus Targets

URL:

```text
http://localhost:9091/targets
```

File screenshot:

```text
docs/screenshots/prometheus-zone-a-targets.png
```

![Zone A Prometheus Targets](docs/screenshots/prometheus-zone-a-targets.png)

Prometheus Zone A melakukan scrape target lokal di `zone-a`. Screenshot ini memperlihatkan agent, demo app, dan Blackbox Exporter probe dalam status `UP`.

### Zone B Prometheus Targets

URL:

```text
http://localhost:9092/targets
```

File screenshot:

```text
docs/screenshots/prometheus-zone-b-targets.png
```

![Zone B Prometheus Targets](docs/screenshots/prometheus-zone-b-targets.png)

Prometheus Zone B melakukan scrape target lokal di `zone-b`. Ini menunjukkan pola yang sama seperti Zone A, tetapi untuk agent dan demo app di zona berbeda.

### HAProxy Status

URL:

```text
http://localhost:8404
```

File screenshot:

```text
docs/screenshots/haproxy-status.png
```

![HAProxy Status](docs/screenshots/haproxy-status.png)

HAProxy dipakai sebagai mock BIG-IP. Screenshot ini menunjukkan frontend dan backend yang menjembatani traffic antara `core_net`, `zone_a_net`, dan `zone_b_net`.

### Elasticsearch Logs

URL:

```text
http://localhost:9200/demo-app-logs-*/_search?pretty&size=5
```

File screenshot:

```text
docs/screenshots/elasticsearch-logs.png
```

![Elasticsearch Logs](docs/screenshots/elasticsearch-logs.png)

Endpoint Elasticsearch dipakai untuk mengecek raw log yang sudah masuk dari demo app. Data ini menjadi sumber application logs dan application metrics di Grafana.

### Kibana Discover

URL:

```text
http://localhost:5601/app/discover
```

File screenshot:

```text
docs/screenshots/kibana-discover.png
```

![Kibana Discover](docs/screenshots/kibana-discover.png)

Kibana bersifat opsional untuk eksplorasi log. Screenshot ini memperlihatkan index log aplikasi dan field seperti `service`, `zone`, `status`, `latency_ms`, dan `message`.

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
- Application metrics dari Elasticsearch logs: request count, 5xx errors, latency p95
- Application logs: log aplikasi dari Elasticsearch
- Grafana alerting untuk CPU, memory, availability probe, dan application 5xx errors

Dashboard punya filter:

- `Zone`: pilih `All`, `zone-a`, atau `zone-b`.
- `Host`: isi regex hostname, default `.*` untuk semua host.
- `Log Status`: pilih `All logs` atau `5xx errors only`.

Contoh filter satu host:

```text
Host = 18064c220071
```

Catatan untuk PoC Docker: hostname agent dan hostname demo app bisa berbeda karena berjalan sebagai container terpisah. Di production, nilai ini bisa disamakan dengan hostname VM/server.

## Toggle Synthetic Error

Secara default demo app tidak membuat error palsu:

```text
DEMO_ERROR_RATE=0
```

Pakai command sesuai cara menjalankan stack.

Kalau start manual dengan `docker compose -f deploy/docker-compose.yml up`:

```bash
make errors-on
make errors-off
```

Kalau start dengan `make up-federated`:

```bash
make errors-on-federated
make errors-off-federated
```

Kalau start dengan `make up-single`:

```bash
make errors-on-single
make errors-off-single
```

Saat error dinyalakan, default rate-nya:

```text
zone-a payments-api: 15%
zone-b lending-api: 10%
```

Rate bisa diubah saat menyalakan:

```bash
DEMO_ERROR_RATE_ZONE_A=0.05 DEMO_ERROR_RATE_ZONE_B=0.02 make errors-on
```

Nilai local bisa disimpan di `.env`:

```bash
cp .env.example .env
```

Untuk memicu in-app alert Grafana dengan cepat:

```bash
DEMO_ERROR_RATE_ZONE_A=1 DEMO_ERROR_RATE_ZONE_B=1 make errors-on
```

Tunggu 1-2 menit, lalu buka:

```text
http://localhost:3000
```

Masuk ke menu:

```text
Alerting > Alert rules
```

Rule yang harus berubah menjadi alerting:

```text
Application 5xx errors detected
```

Dashboard juga punya panel:

```text
Alert Notifications
```

Panel ini berfungsi seperti badge notifikasi. Nilainya menunjukkan jumlah 5xx error dalam 5 menit terakhir. Panel berubah merah jika nilainya lebih dari 0.

Setelah selesai demo alert:

```bash
make errors-off
```

Setelah error dimatikan, data 5xx lama masih bisa muncul di Grafana sampai keluar dari time range. Pakai `Last 5 minutes` untuk melihat kondisi terbaru.

## Kumpulan Contoh Query Prometheus

Buka Central Prometheus:

```text
http://localhost:9090
```

Coba query berikut:

```promql
system_cpu_usage_percent
system_memory_usage_percent
system_memory_used_bytes
system_memory_available_bytes
system_swap_usage_percent
system_network_receive_bytes_total
system_network_transmit_bytes_total
system_network_receive_packets_total
system_network_transmit_packets_total
system_network_receive_errors_total
system_network_transmit_errors_total
system_network_receive_packets_per_second
system_network_transmit_packets_per_second
system_network_receive_errors_per_second
system_network_transmit_errors_per_second
probe_success
demo_app_requests_total
demo_app_errors_total
demo_app_request_duration_seconds_count
```

Query per zone:

```promql
sum by (zone) (up)
sum by (zone, probe_type) (probe_success)
system_cpu_usage_percent{scope="overall"}
system_cpu_usage_percent{scope="core"}
sum by (zone, service) (rate(demo_app_requests_total[1m]))
```

## Cara Cek Logs

Cek logs aplikasi dari Elasticsearch:

```bash
curl "http://localhost:9200/demo-app-logs-*/_search?pretty&size=5"
```

Query application error dari Elasticsearch:

```bash
curl "http://localhost:9200/demo-app-logs-*/_search?pretty&q=status:%5B500%20TO%20599%5D&size=5"
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
make build
make build-linux
make build-windows
```

Untuk membuat bundle offline berisi image Docker dan binary static Linux/Windows:

```bash
scripts/offline-bundle.sh
```
