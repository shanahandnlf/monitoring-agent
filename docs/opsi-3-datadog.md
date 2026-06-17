# Opsi 3 - Datadog (Free Tier)

Opsi ini mengganti paradigma monitoring dari **self-hosted + pull** (Prometheus/ELK pada Opsi 1 & 2) menjadi **SaaS + push**: Datadog Agent berjalan di tiap node, mengumpulkan metrik, lalu mengirimkannya ke backend Datadog di cloud.

Pada POC ini, Datadog Agent **tidak** menggantikan Go agent yang sudah ada. Sebaliknya, Datadog Agent men-scrape endpoint Prometheus (`/metrics`) milik Go agent dan demo app lewat integrasi **OpenMetrics**, lalu mem-push-nya ke Datadog dengan tag `zone` dan `host`. Jadi metrik yang sama (`system_*`, `demo_app_*`) tetap dipakai, hanya tujuan penyimpanannya yang berbeda.

## Arsitektur

```
        zone_a_net                     zone_b_net
  ┌────────────────────┐         ┌────────────────────┐
  │ datadog-agent       │        │ datadog-agent       │
  │   host=node-zone-a  │        │   host=node-zone-b  │
  │   tag zone:zone-a   │        │   tag zone:zone-b   │
  └─────────┬───────────┘        └─────────┬───────────┘
   OpenMetrics scrape lokal        OpenMetrics scrape lokal
   ┌─────────┴─────────┐           ┌────────┴──────────┐
   │ agent-zone-a:9100 │           │ agent-zone-b:9100 │
   │ demo-app-a:8080   │           │ demo-app-b:8080   │
   └───────────────────┘           └───────────────────┘
            │ HTTPS :443 (push)              │
            └───────────────┬────────────────┘
                            ▼
                  ┌───────────────────┐
                  │  Datadog (SaaS)   │  dashboard, metric explorer, alert
                  └───────────────────┘
```

## Prasyarat

1. Akun Datadog gratis: https://www.datadoghq.com/ (pilih site, mis. `datadoghq.com` (US1) atau `ap1.datadoghq.com`).
2. API Key: **Organization Settings > API Keys**.
3. Isi `.env` di root repo:

   ```bash
   cp .env.example .env
   ```

   ```dotenv
   DD_API_KEY=<api-key-kamu>
   DD_SITE=datadoghq.com
   ```

   > `DD_SITE` harus cocok dengan region akunmu, kalau tidak data tidak akan masuk.

## Menjalankan

```bash
make up-datadog      # build + start agent, demo-app, traffic, datadog-agent (2 zona)
make ps-datadog
make logs-datadog
make down-datadog
make reset-datadog   # down + hapus volume
```

> Catatan: target `make` sudah memakai `--env-file .env`. Karena compose file ada di `deploy/`, project directory-nya `deploy/`, sehingga `.env` di root repo **tidak** terbaca otomatis. Kalau menjalankan manual, sertakan flag itu:
>
> ```bash
> docker compose --env-file .env -f deploy/docker-compose.yml -f deploy/docker-compose.datadog.yml up --build
> ```

Cek di Datadog:
- **Infrastructure > Host Map**: muncul `node-zone-a` dan `node-zone-b`.
- **Metrics > Explorer**: cari `poc.system_cpu_usage_percent` atau `poc.demo_app_requests_total`, filter `from zone:zone-a`.
- Segmentasi per zona/host memakai **tag** (`zone:zone-a`, `host:node-zone-a`) — bukan label/federation seperti Prometheus.

Verifikasi agent dari host:

```bash
docker exec monitoring-poc-datadog-datadog-agent-zone-a-1 agent status
```

## Dashboard

Tersedia dashboard siap-import di `deploy/datadog/dashboard.json` yang meniru panel Grafana (CPU, memory, swap, network, application requests) dengan template variable `zone` & `host`.

Cara import di Datadog:
1. **Dashboards → New Dashboard**, beri nama bebas, lalu **Create**.
2. Klik ikon **gear/settings** (kanan atas) → **Import dashboard JSON**.
3. Tempel isi `deploy/datadog/dashboard.json`, lalu konfirmasi replace.

Metrik yang dipakai (namespace `poc`):
- Gauge: `poc.system_cpu_usage_percent` (`scope:overall`), `poc.system_memory_usage_percent`, `poc.system_memory_used_bytes` / `_available_bytes`, `poc.system_swap_usage_percent`, `poc.system_network_*_per_second`.
- Counter: `poc.demo_app_requests.count` (counter `_total` → `.count` di Datadog; dipakai dengan `.as_rate()` / `.as_count()`).

> Filter per zona/host memakai tag (`zone:`, `host:`) — bukan label/federation seperti Prometheus.

## Logs

Datadog Agent mengumpulkan **log aplikasi** dengan men-*tail* stdout container demo-app (mount `docker.sock` + `/var/lib/docker/containers`), lalu push ke Datadog. demo-app menulis access log sebagai **JSON murni** (tanpa prefix), sehingga Datadog otomatis mem-parse field `zone`, `host`, `host_ip`, `service`, `status`, `latency_ms`, `message`.

Model per-zona: tiap Datadog Agent hanya men-tail container demo-app di zonanya (filter `DD_CONTAINER_INCLUDE_LOGS`), supaya tidak dobel karena dua agent berbagi satu Docker host.

Lihat di Datadog: **Logs → Search**, filter `service:payments-api` atau `@zone:zone-a`.

> Catatan deployment: di POC ini satu agent per zona membaca log container lewat `docker.sock` (semua container ada di satu Docker host). Di produksi multi-VM, satu agent per zona **tidak** bisa membaca log VM lain — pilih: aplikasi mengirim log via jaringan ke agent zona, atau pasang agent di tiap VM.

> **Penting (free tier):** Log Management **tidak** termasuk free tier. Ini hanya jalan selama **free trial** / paket berbayar. Setelah trial habis, `DD_LOGS_ENABLED=true` tidak akan menyimpan log (dan dihitung biaya per-GB di paket berbayar).

## Limitasi Free Tier

> Ketentuan Datadog bisa berubah — selalu cek halaman pricing terbaru. Ringkasan umum free tier:

| Aspek | Free Tier |
| --- | --- |
| Jumlah host | **maksimal 5 host** (POC ini pakai 2: `node-zone-a`, `node-zone-b`) |
| Retensi metrik | **1 hari** saja |
| Infrastructure metrics + integrasi (OpenMetrics, dll) | ✅ termasuk |
| Dashboard, Host Map, Metric Explorer | ✅ termasuk |
| **Log Management (pengganti ELK)** | ❌ tidak termasuk free tier (berbayar per-GB; aktif saat free trial) |
| **APM / tracing** | ❌ tidak termasuk free tier (aktif saat free trial; lihat [opsi-tracing.md](opsi-tracing.md)) |
| **Monitors / alerting** | ❌ umumnya butuh paket Pro |
| Anomaly/forecast, lookback panjang | ❌ tidak termasuk |

**Implikasi untuk use-case ini:**

- POC ini mengaktifkan `DD_LOGS_ENABLED=true` dan `DD_APM_ENABLED=true` karena memakai **free trial** (membuka fitur Pro sementara). Setelah trial habis dan turun ke free tier, Logs & APM **berhenti** — hanya metrik infrastruktur yang tetap jalan.
- **Alerting 5xx** seperti di Opsi 1/2 tidak bisa direplikasi penuh di free tier (monitors butuh Pro).
- Retensi 1 hari membuatnya hanya cocok untuk demo/eksplorasi, bukan analisis historis.

## Apakah penyimpanan data bisa on-premise?

**Tidak untuk Datadog SaaS.** Datadog adalah layanan cloud; metrik dan log disimpan di backend Datadog (region yang dipilih lewat `DD_SITE`, mis. US/EU/AP), **bukan** di server sendiri. Tidak ada deployment Datadog self-hosted/on-prem untuk pelanggan umum (berbeda dari Prometheus/Grafana/Elastic yang memang bisa self-hosted).

Yang bisa dilakukan terkait kontrol data:

- **Pemilihan region (`DD_SITE`)** untuk data residency — tapi tetap di cloud Datadog.
- **Observability Pipelines / Datadog Agent + Vector**: memproses & memfilter data **di on-prem** sebelum dikirim, dan bisa **menggandakan rute** sebagian data ke tujuan on-prem sendiri (mis. Elasticsearch/S3) sambil mengirim sebagian ke Datadog. Ini fitur lanjutan/berbayar, dan untuk metrik penyimpanannya tetap di cloud Datadog.

**Kesimpulan:** kalau syaratnya data **harus tersimpan on-premise** (umum di lingkungan perbankan/regulasi ketat), Datadog SaaS tidak memenuhinya, dan ini justru argumen kuat memilih Opsi 1/2 (Prometheus + ELK, sepenuhnya on-prem). Datadog unggul di sisi *managed* dan kelengkapan fitur, tetapi menukar kontrol data dan biaya.
