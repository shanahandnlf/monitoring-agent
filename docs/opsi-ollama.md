# Opsi AI - Ollama (Ringkasan Log On-Prem)

Fitur ini menambahkan kemampuan AI sederhana ke stack monitoring memakai **Ollama**, sebuah LLM runtime yang berjalan **100% lokal/offline**. Tujuannya: **ringkasan log otomatis** — secara periodik membaca log error aplikasi dari Elasticsearch, merangkumnya dengan LLM, lalu menyimpan hasilnya kembali ke Elasticsearch untuk ditampilkan di Grafana.

Berbeda dari Datadog (SaaS), seluruh proses ini berjalan di infrastruktur sendiri tanpa mengirim data ke cloud — cocok untuk lingkungan air-gapped, gratis, dan tanpa API key.

## Arsitektur

```
                       core_net
  ┌──────────────┐  query logs   ┌─────────────────────┐  prompt  ┌──────────┐
  │ Elasticsearch│◄──────────────│  insight (Go)       │─────────►│  ollama  │
  │  demo-app-   │               │  - scheduler batch  │   HTTP   │ llama3.2 │
  │  logs-*      │──────────────►│  - POST /summarize  │◄─────────│   :3b    │
  │              │  tulis hasil  │  - /healthz         │ ringkasan└──────────┘
  │ monitoring-  │◄──────────────┘─────────────────────┘
  │ insights-*   │
  └──────┬───────┘
         ▼  datasource "AI Insights" (elasticsearch-insights)
  Grafana panel "AI Log Insights (Ollama)"
```

Dua komponen baru (keduanya di `core_net`):

- **`ollama`** — container LLM runtime, model default `llama3.2:3b` (kecil, CPU-friendly).
- **`insight`** — binary Go (`cmd/insight`) yang menjadwalkan ringkasan, query ES, memanggil Ollama, dan menulis hasil ke ES.

Agent dan demo-app **tidak diubah** sama sekali.

## Cara menjalankan

```bash
make up-ollama       # stack + ollama + insight (pull model otomatis saat pertama)
make ps-ollama
make logs-ollama
make down-ollama
make reset-ollama    # + hapus volume (termasuk model)
```

Saat pertama dijalankan, container `ollama` akan men-download model (`llama3.2:3b`, ~2GB) ke volume. Tunggu sampai selesai (`make logs-ollama` → tunggu baris `success` dari pull). Ganti model lewat env `OLLAMA_MODEL` (mis. `OLLAMA_MODEL=qwen2.5:3b make up-ollama`).

## Cara melihat hasil

1. Hasilkan beberapa error 5xx: `make errors-on` (matikan lagi dengan `make errors-off`).
2. Picu ringkasan langsung (tanpa menunggu interval 5 menit):

   ```bash
   curl -XPOST http://localhost:8090/summarize
   ```

3. Cek dokumen hasil di Elasticsearch:

   ```bash
   curl "http://localhost:9200/monitoring-insights-*/_search?pretty&size=3"
   ```

4. Di Grafana (`http://localhost:3000`) buka dashboard **Segmented Zone Monitoring POC**, panel **"AI Log Insights (Ollama)"** menampilkan ringkasan terbaru.

## Konfigurasi (env pada service `insight`)

| Env | Default | Keterangan |
| --- | --- | --- |
| `INSIGHT_LISTEN_ADDRESS` | `:8090` | Alamat HTTP server |
| `INSIGHT_INTERVAL` | `5m` | Interval scheduler ringkasan otomatis |
| `INSIGHT_LOG_WINDOW` | `15m` | Rentang waktu log yang dianalisis |
| `INSIGHT_MAX_LOG_LINES` | `20` | Batas contoh baris log error dalam prompt |
| `ELASTICSEARCH_URL` | `http://elasticsearch:9200` | Sumber log + tujuan hasil |
| `OLLAMA_URL` | `http://ollama:11434` | Endpoint Ollama |
| `OLLAMA_MODEL` | `llama3.2:3b` | Model LLM |
| `OLLAMA_TIMEOUT` | `60s` | Timeout inferensi |

## Pertimbangan performa

LLM sengaja **tidak pernah berada di hot path**:

- Agent (scrape metrik) dan demo-app (request handler) tidak disentuh.
- Inferensi hanya jalan **batch** (default tiap 5 menit) atau saat endpoint `/summarize` dipanggil — bukan per-request.
- Model **kecil & CPU-friendly** (`llama3.2:3b`), dan di-unload saat idle (`OLLAMA_KEEP_ALIVE=5m`).
- Container `ollama` diberi `mem_limit` agar tidak menstarve Elasticsearch/Prometheus.
- Input ke LLM dibatasi (`INSIGHT_MAX_LOG_LINES` + ringkasan agregat, bukan log mentah).
- Bila Ollama lambat/mati, `insight` mencatat error dan lanjut — tidak crash, tidak mengganggu komponen lain.
- Stack ini **opt-in** (`make up-ollama`); opsi 1/2/3/tracing tidak terpengaruh.

## Air-gap

- Ollama runtime + model berjalan sepenuhnya lokal, tanpa panggilan internet saat runtime.
- Yang perlu internet hanya **download model sekali**. Untuk server terisolasi:
  1. Jalankan `make up-ollama` di mesin yang punya internet agar model ter-pull ke volume `ollama_models`.
  2. Arsipkan volume model:

     ```bash
     docker run --rm -v monitoring-poc-ollama_ollama_models:/m -v "$PWD":/out alpine \
       tar czf /out/ollama-models.tar.gz -C /m .
     ```

  3. Di server tujuan, `docker load` bundle image (`scripts/offline-bundle.sh`) lalu pulihkan arsip model ke volume yang sama sebelum `make up-ollama`.
- Binary `insight` adalah static Go binary (seperti agent), aman dijalankan offline.
