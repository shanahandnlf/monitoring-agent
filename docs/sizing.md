# Sizing 500 Server — Single & Federated

Kebutuhan CPU/RAM/disk untuk menampung **500 server** pada dua jalur: **metrik**
(Go agent → Prometheus) dan **log** (demo-app/Windows Event Log → ELK).

## Asumsi

| Parameter | Nilai | Sumber |
| --- | --- | --- |
| Jumlah server | 500 | target |
| Series metrik per server | ~200 | kontrak metrik agent |
| Retensi (metrik & log) | **14 hari** | ditetapkan |
| Replika Elasticsearch | **0** | ditetapkan (PoC/hemat) |
| Scrape interval | 15s | config zona |
| Ukuran 1 dokumen log di ES | **~2,2 KB** (store) | **terukur** (`demo-app-logs`: 278 dok / 599,6 KB) |
| Request/detik per server | **parametrik** | realitas produksi (isi sendiri) |

> Angka metrik & ukuran-per-dokumen di bawah **terukur** dari PoC ini; hanya
> request-rate per server yang harus kamu isi (tak bisa diukur di laptop).

## Jalur Metrik (Prometheus) — kecil

- Series: 500 × 200 = **~100k** (+ blackbox/app ≈ **~150k**).
- RAM: terukur **~422 MB untuk 100k series** → ~650 MB untuk 150k; sediakan **2–4 GB**
  (untuk query + compaction).
- Ingest: 150k / 15s ≈ **10k samples/s**.
- Disk: `10k × 2 byte × 86400 × 14` ≈ **~24 GB** → sediakan **~50 GB SSD**.
- CPU: terukur <1% untuk scrape 100k → **2–4 core** lebih dari cukup.

**Metrik untuk 500 server itu ringan** (~0,5 GB RAM, ~50 GB disk).

## Jalur Log (ELK) — penentu utama

Volume didorong **request-rate**, bukan jumlah server. Dengan ~2,2 KB/dokumen,
1 dokumen per request, replika 0:

```
GB/hari = req_per_detik_per_server × 0,19 GB × 500
disk_14h = GB/hari × 14 × 1,15 (overhead index)   [replika 0]
```

| Rate/server | GB/hari (500 svr) | Disk 14 hari (usable) |
| --- | --- | --- |
| 1 req/s | ~95 GB | **~1,5 TB** |
| 5 req/s | ~475 GB | **~7,6 TB** |
| 10 req/s | ~950 GB | **~15 TB** |
| 20 req/s | ~1,9 TB | **~30 TB** |
| 50 req/s | ~4,75 TB | **~76 TB** |

Sediakan **disk fisik ~1,5–2× usable** (Elasticsearch menjaga disk < 85% + ruang
merge). **Log mendominasi total disk** — metrik (~50 GB) tak signifikan dibanding ini.

ES cluster:
- Heap ≤ 31 GB (= ½ RAM → node **64 GB RAM**); target ukuran shard 10–50 GB.
- **Data node**: 3–5 node tergantung volume (mis. 10 req/s/14h ≈ 15 TB → ~3 node ×
  ~6–8 TB disk).
- Indexing CPU sebanding docs/s (500 × rate); beberapa core per data node.

## Rekomendasi spesifikasi

### Opsi Single (1 Prometheus untuk semua 500)

| Komponen | CPU | RAM | Disk |
| --- | --- | --- | --- |
| Prometheus | 4 vCPU | 8 GB | 50 GB SSD |
| ES data node ×3 | 8 vCPU | 64 GB | sesuai tabel log (mis. 6–8 TB/node) |
| Logstash | 4 vCPU | 8 GB | kecil |
| Kibana/Grafana | 2 vCPU | 4 GB | kecil |

SPOF di Prometheus; satu node men-scrape 500 target (~33 scrape/s — aman).

### Opsi Federated (mis. 5 zona × 100 server)

| Komponen | CPU | RAM | Disk |
| --- | --- | --- | --- |
| Prometheus zona ×5 | 2 vCPU | 4 GB | 20 GB SSD |
| Prometheus central | 2 vCPU | 4 GB | 20 GB SSD |
| ELK (data/Logstash/Kibana) | **sama dengan Single** | | log tetap terpusat |

Federation membagi beban scrape + isolasi kegagalan per zona; **biaya ELK tidak
berubah** karena log tetap terpusat.

## Cara hitung ulang

Ganti `req_per_detik_per_server` dengan angka dari traffic aplikasimu (ukur dengan
`make bench-k6-logs` lalu lihat pertumbuhan `_cat/indices`), atau ganti retensi/replika
lalu terapkan rumus di atas. Replika 1 → **disk ×2**.
