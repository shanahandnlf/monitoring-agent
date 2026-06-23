# Opsi Dinamis - Service Discovery (Prometheus docker_sd)

Varian ini menjalankan stack federated, tetapi mengganti **service discovery statik** (target ditulis manual di `static_configs`) menjadi **dinamis** memakai `docker_sd_config`. Prometheus zona menemukan target otomatis dengan menanyakan Docker daemon, lalu memfilter berdasarkan label container. Target baru (mis. hasil scaling) langsung ter-discover tanpa mengedit config.

Opsi statik (single/federated/datadog/tracing/ollama) tidak diubah — varian ini terpisah untuk perbandingan.

## Statik vs Dinamis

| | Statik (`zone-a.yml`) | Dinamis (`zone-a-dynamic.yml`) |
| --- | --- | --- |
| Target | Ditulis manual (`static_configs: targets: [...]`) | Ditemukan dari Docker daemon |
| Tambah/scale instance | Edit config + reload | Otomatis (refresh tiap 15s) |
| Sumber | Daftar di file | Label container + `docker.sock` |

## Cara kerja

Tiap Prometheus zona mount `docker.sock` dan menjalankan job `docker-dynamic`:

1. **Keep** hanya container ber-label `prometheus.scrape=true`.
2. **Keep** hanya target di network zonanya (`__meta_docker_network_name =~ .*zone_a_net`) → segmentasi zona tetap terjaga.
3. **Set `__address__`** = IP container : port dari label `prometheus.port` (karena image mengekspos 8080 & 9100, langkah ini memilih port yang benar; target duplikat per-port otomatis ter-dedup).
4. **Set label** `zone`, `role`, `service` dari label container.

Label dipasang di service (overlay `docker-compose.dynamic.yml`), contoh:

```yaml
agent-zone-a:    { prometheus.scrape: "true", prometheus.port: "9100", zone: zone-a, role: infrastructure }
demo-app-zone-a: { prometheus.scrape: "true", prometheus.port: "8080", zone: zone-a, role: application, service: payments-api }
```

Central Prometheus dan blackbox (probe availability) tetap statik — federation dan probing tidak butuh service discovery.

## Menjalankan

```bash
make up-dynamic
make ps-dynamic
make logs-dynamic
make down-dynamic
```

Cek hasil discovery:

- `http://localhost:9091/targets` (zone-a) & `http://localhost:9092/targets` (zone-b) → target `docker-dynamic` muncul sebagai **discovered**, status UP, dengan label `zone/role/service`.
- `http://localhost:9091/service-discovery` → lihat label `__meta_docker_*` mentah sebelum relabel.
- `http://localhost:9090` (central) → metrik kedua zona tetap masuk via federation.
- `http://localhost:3000` (Grafana) → dashboard terisi seperti biasa (metrik sama, hanya cara discover beda).

## Demo scaling (otomatis ter-discover)

```bash
docker compose -p monitoring-poc-dynamic \
  -f deploy/docker-compose.yml -f deploy/docker-compose.dynamic.yml \
  up -d --scale demo-app-zone-a=3 --no-recreate
```

Tunggu ~15-20 detik, buka `http://localhost:9091/targets` → muncul **3 target** `demo-app` zone-a (tiap replika punya `instance` berbeda, jadi series tidak bentrok). Scale balik:

```bash
docker compose -p monitoring-poc-dynamic \
  -f deploy/docker-compose.yml -f deploy/docker-compose.dynamic.yml \
  up -d --scale demo-app-zone-a=1 --no-recreate
```

Target berkurang otomatis tanpa edit config — inti dari service discovery dinamis.

## Catatan

- Prometheus zona dijalankan sebagai `user: "0:0"` (root) agar bisa membaca `docker.sock`. Untuk produksi, lebih aman memakai docker-socket-proxy (mengekspos Docker API terbatas via TCP) daripada mount socket langsung sebagai root.
- Pendekatan ini khusus lingkungan Docker. Di produksi nyata, padanannya: `kubernetes_sd` (Kubernetes), `consul_sd`, `dns_sd`, atau `file_sd`.
