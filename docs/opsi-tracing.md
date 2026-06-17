# Distributed Tracing (Grafana Tempo & Datadog APM)

demo-app diinstrumentasi **sekali** dengan **OpenTelemetry** (`cmd/demo-app/tracing.go`), lalu trace-nya bisa diarahkan ke dua backend berbeda hanya dengan mengganti endpoint OTLP:

- **Opsi 1/2 (on-prem):** Grafana **Tempo**
- **Opsi 3 (cloud):** **Datadog APM** (lewat OTLP receiver di Datadog Agent)

Instrumentasi bersifat **opt-in**: kalau `OTEL_EXPORTER_OTLP_ENDPOINT` kosong, tracing mati total dan demo-app jalan persis seperti semula (span jadi no-op, tanpa overhead). Jadi stack non-tracing tidak terpengaruh.

## Apa yang di-trace

- Span server otomatis per request (`otelhttp`) untuk `/api/demo`, `/healthz`.
- Child span manual `process-demo` di sekitar pemrosesan, dengan atribut `demo.synthetic_error`.
- Resource attributes: `service.name`, `service.instance.id`, `zone`, `host`, `env=poc`.

```
GET /api/demo  (root span, otelhttp)
└─ process-demo  (child span)   ← di sinilah waktu kerja terlihat
```

## A. Grafana Tempo (Opsi 1/2 — on-prem)

Trace dikirim ke Tempo via OTLP gRPC, disimpan lokal di Tempo, dilihat di Grafana.

```
demo-app ─OTLP:4317→ Tempo ─→ Grafana (Explore / Tempo datasource)
```

Menjalankan (full stack federated + Tempo):

```bash
make up-tracing
make ps-tracing
make logs-tracing
make down-tracing
make reset-tracing   # + hapus volume
```

Melihat trace:
1. Buka Grafana `http://localhost:3000` → **Explore**.
2. Pilih datasource **Tempo**.
3. **Search** → pilih service `payments-api` / `lending-api`, atau cari oleh tag `zone`.
4. Klik satu trace → muncul **waterfall** span (`http.server` → `process-demo`).

Penyimpanan trace: **lokal di container Tempo** (on-prem), retention 1 jam (lihat `deploy/tempo/tempo.yaml`).

## B. Datadog APM (Opsi 3 — cloud)

Datadog Agent mengaktifkan **OTLP receiver** (`DD_OTLP_CONFIG_RECEIVER_PROTOCOLS_GRPC_ENDPOINT=0.0.0.0:4317`, `DD_APM_ENABLED=true`). demo-app push OTLP ke agent zonanya, agent meneruskan ke Datadog.

```
demo-app ─OTLP:4317→ Datadog Agent (per zona) ─HTTPS→ Datadog APM (US5)
```

Sudah otomatis aktif saat `make up-datadog` (endpoint OTLP di-set di `docker-compose.datadog.yml`).

Melihat trace: Datadog → **APM → Traces / Services**, filter `service:payments-api` atau `env:poc`.

Verifikasi dari agent:

```bash
docker exec monitoring-poc-datadog-datadog-agent-zone-a-1 agent status | sed -n '/APM Agent/,/Autodiscovery/p'
```

Tanda sukses: muncul `Priority sampling rate for 'service:payments-api,env:poc'` dan `Stats: N payloads` yang bertambah (artinya trace diterima & diproses).

> **Free tier:** APM **tidak** termasuk free tier — hanya jalan saat **free trial** / paket berbayar (per-host). Ini kontras dengan Tempo yang open-source & on-prem.

## Perbandingan

| | Tempo (Opsi 1/2) | Datadog APM (Opsi 3) |
| --- | --- | --- |
| Penyimpanan trace | On-prem (Tempo) | Cloud Datadog |
| Tampilan | Grafana | Datadog APM UI |
| Biaya | Open-source | Berbayar (trial) |
| Instrumentasi | Sama: OpenTelemetry (1 kode, beda endpoint) | |
