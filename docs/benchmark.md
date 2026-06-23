# Benchmark & Stress Test (target 500 server)

Dua klaim yang harus dibuktikan objektif: **(1) agent ringan** dan **(2) backend
sanggup 500 server**. Stack-nya *pull-based* (Prometheus men-scrape `/metrics`;
demo-app menulis log JSON → Logstash → Elasticsearch), dan itu menentukan tool.

## Tool per lapisan (jawaban "pakai k6?")

| Lapisan | Beban | Tool | k6? |
| --- | --- | --- | --- |
| Footprint agent | CPU/RAM/alloc per agent | Go `testing.B` + self-metrics | bukan k6 |
| Endpoint agent | HTTP `/metrics` di-hit | **k6** | ✅ ya |
| Metrics 500 server | Prometheus ingest ~100k series | **prometheus/avalanche** + `docker --scale` | ❌ k6 salah tool |
| Log 500 server | demo-app → Logstash → ES | **k6** ke `/api/demo` | ✅ ya |

**k6 dipakai untuk yang HTTP-driven** (endpoint agent + traffic log). Untuk
mensimulasikan 500 server yang **di-scrape** Prometheus, k6 tidak bisa — Prometheus
*menarik* metrik, jadi pakai **Avalanche** (exporter seri sintetis).

---

## Lapisan 1 — Footprint agent (bukti "ringan")

### Biaya per-scrape (Go benchmark)
```bash
make bench-agent      # = go1.20 test ./internal/collector/ -bench . -benchmem
```
Membaca **ns/op** (waktu sekali collect), **B/op**, **allocs/op**. Biaya steady-state
agent ≈ ns/op × frekuensi scrape (mis. sekali per 15s) → praktis nol.

Contoh hasil (laptop, macOS — jalankan di OS target untuk angka representatif):

| Benchmark | ns/op | allocs/op |
| --- | --- | --- |
| CPUCollect | ~5.900 (5.9 µs) | 63 |
| MemoryCollect | ~2.400 (2.4 µs) | 26 |
| NetworkCollect | ~1.680.000 (1.7 ms) | 331 |
| CollectAll | ~1.620.000 (1.6 ms) | 423 |

> Network dominan karena `gopsutil net.IOCounters` di macOS mahal; di **Linux** (baca
> `/proc/net/dev`) jauh lebih murah. Walau 1.6 ms sekali per 15s = ~0,01% CPU.

### Footprint berjalan (self-metrics)
Agent kini mengekspor metrik dirinya sendiri (`cmd/agent/main.go`): `go_goroutines`,
`go_memstats_*` (heap), dan di **Linux** `process_resident_memory_bytes` /
`process_cpu_seconds_total`. Amati di Grafana sepanjang waktu (deteksi leak).

```bash
make build && AGENT_ZONE=bench ./bin/agent &      # atau di dalam stack
make bench-k6-agent                                # k6 ramp ke /metrics, p95/p99 + RPS
```
- `go_*` tersedia di **semua OS** (termasuk Windows) → proxy memori utama.
- `process_resident_memory_bytes` hanya **Linux** (procfs). Di **Windows** ukur RSS
  via Task Manager / `Get-Counter`; di Docker via `docker stats`.
- **Agent legacy (agent-2008)** tak punya client_golang → footprint diukur eksternal.

---

## Lapisan 2 — Kapasitas metrics 500 server (Prometheus)

```bash
make bench-fleet                       # Avalanche ~100k series + Prometheus bench
open http://localhost:9094
# skala fan-out (5 target, series x5):
docker compose -p monitoring-bench -f deploy/docker-compose.bench.yml up -d --scale avalanche=5
make bench-fleet-down                  # stop + bersih
```
Atur jumlah series via env: `BENCH_METRIC_COUNT` (default 200) ×
`BENCH_SERIES_COUNT` (default 500) = ~100k series (≈ 500 server × 200 series).

PromQL untuk dibaca (di `:9094`):

| Apa | Query |
| --- | --- |
| Total series aktif | `prometheus_tsdb_head_series` |
| Laju ingest | `rate(prometheus_tsdb_head_samples_appended_total[5m])` |
| RAM Prometheus | `process_resident_memory_bytes{job="prometheus-bench"}` |
| CPU Prometheus | `rate(process_cpu_seconds_total{job="prometheus-bench"}[5m])` |
| Durasi scrape | `scrape_duration_seconds` |
| Ukuran disk TSDB | `prometheus_tsdb_storage_blocks_bytes` |

Contoh hasil terukur (laptop, ~100k series, scrape 15s):

| Metrik | Nilai |
| --- | --- |
| `prometheus_tsdb_head_series` | **100.482** |
| samples appended/s | **~6.700** |
| RAM Prometheus | **~422 MB** |
| CPU Prometheus | **~0,5% (0,005 core)** |

**Kesimpulan**: metrics untuk 500 server itu **ringan** (~0,4 GB RAM, <1% CPU di
laptop). Ini mengonfirmasi sizing: jalur metrics murah, **ELK (log) yang berat**.

### Cross-check agent nyata
```bash
make bench-scale SCALE=20              # 20 replika demo-app, ter-discover via docker_sd
```
Memastikan angka sintetis Avalanche sejalan dengan target asli pada skala kecil.

---

## Lapisan 3 — Kapasitas log 500 server (ELK)

```bash
make up-federated                      # stack + ELK
make bench-k6-logs LOG_RATE=200        # k6 pacu /api/demo (200 req/s) → log ke ES
```
Ukur saat beban jalan:

| Apa | Cara |
| --- | --- |
| Jumlah & ukuran index | `curl 'localhost:9200/_cat/indices?v'` |
| Laju indexing | `curl 'localhost:9200/_nodes/stats/indices' \| jq ...indexing.index_total` (delta/detik) |
| CPU/RAM ES & Logstash | `docker stats` |

### Ekstrapolasi ke 500 server
Log didorong oleh **request rate**, bukan jumlah server. Ukur per-server lalu proyeksi:

```
raw_GB/hari/server = req/s × byte_per_log × 86400 / 1e9
disk_ES = raw_GB/hari/server × 500 × (1 + replica) × retensi_hari × 1.15(overhead)
```
Sediakan ~2× ruang usable (ES butuh disk <85%; ada merge). RAM node: heap ≤ 31 GB
(= ½ RAM). **Inilah komponen termahal & penentu sizing untuk 500 server** — angka
pastinya harus dari pengukuran `raw_GB/hari` kamu, bukan tebakan.

---

## Catatan / limitasi (dicatat di laporan)

- Uji di **laptop lokal** → skala 500 untuk metrics disimulasikan sintetis
  (Avalanche) dan untuk log diekstrapolasi dari pengukuran per-server. Bukan 500
  agent nyata.
- Benchmark network di macOS lebih mahal dari Linux — jalankan `make bench-agent`
  di OS target untuk angka final.
- k6 dijalankan via image `grafana/k6` (Docker), mencapai host lewat
  `host.docker.internal`. Override target dengan `AGENT_URL` / `DEMO_URL`.
