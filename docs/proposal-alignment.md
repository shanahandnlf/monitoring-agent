# Proposal Alignment

Dokumen ini mencatat posisi implementasi PoC terhadap Proposal Design System Revisi 2.0.

## Sudah Disimulasikan

- Dua opsi arsitektur:
  - Single Prometheus + Agent + ELK.
  - Federated Prometheus + Agent + ELK.
- Segmentasi zona lokal:
  - `zone_a_net`
  - `zone_b_net`
  - `core_net`
- HAProxy sebagai mock F5 BIG-IP untuk lintas zona dan federation endpoint.
- Go agent berbasis `gopsutil` sebagai static binary.
- Endpoint agent:
  - `/metrics`
  - `/healthz`
- Infrastructure metrics:
  - CPU overall dan per core.
  - Memory usage, total, used, available.
  - Swap usage, total, used, free.
  - Network bytes in/out.
  - Network packets in/out.
  - Network errors in/out.
  - Estimated network utilization.
  - Estimated packet/error rate per interface.
- Availability:
  - HTTP probe.
  - TCP probe.
  - ICMP probe.
- Application observability:
  - Demo app menghasilkan structured JSON access log.
  - Log dikirim ke Logstash dan disimpan di Elasticsearch.
  - Grafana dashboard membaca application request, 5xx error, latency p95, dan logs dari Elasticsearch.
  - Demo app tetap expose Prometheus metrics sebagai pelengkap/debug.
- Grafana provisioning:
  - Prometheus datasource.
  - Elasticsearch datasource.
  - Dashboard terpadu.
  - Alerting rule PoC untuk CPU, memory, availability, dan application 5xx error.
- Build/deployment preparation:
  - Linux static binary.
  - Windows static binary.
  - Offline image/binary bundle script.

## Limitasi PoC Lokal

- Jumlah server masih disimulasikan dengan dua agent dan dua demo app, bukan 50-200 server.
- F5 BIG-IP asli belum digunakan; HAProxy hanya mock lokal untuk menunjukkan pola traffic dan port.
- Rule network masih representasi lokal, belum berupa request resmi ke tim network.
- ELK masih dijalankan lokal, bukan Elasticsearch existing read-only milik tim aplikasi.
- Grafana security masih mode PoC:
  - Default admin user/password.
  - Anonymous viewer aktif.
  - Belum HTTPS, RBAC enterprise, SSO, atau identity integration.
- Agent belum dipaketkan sebagai Linux systemd service atau Windows Service.
- Belum ada signed binary, antivirus scan, jump host flow, atau SOP transfer file production.
- Prometheus target masih static config, belum service discovery atau generated inventory untuk skala puluhan/ratusan server.

## Gap Production Berikutnya

- Tambah deployment unit:
  - `systemd` unit untuk Linux.
  - Windows Service wrapper atau installer.
- Tambah inventory/template target agar onboarding server baru tidak perlu edit YAML manual.
- Validasi Grafana alerting terhadap instance Grafana live.
- Hubungkan Grafana ke Elasticsearch existing secara read-only.
- Ganti HAProxy mock dengan detail rule F5 BIG-IP aktual.
- Tambah hardening:
  - HTTPS.
  - Secret management.
  - RBAC/SSO.
  - Audit trail.
  - Firewall local agent.
