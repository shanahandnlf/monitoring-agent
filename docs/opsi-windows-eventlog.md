# Opsi Windows Event Log → ELK

Menangkap **kenapa** sebuah aplikasi/service Windows mati — bukan sekadar "mati".

## Konteks

Tiga sinyal saling melengkapi:

| Sinyal | Lihat | Status |
| --- | --- | --- |
| Heartbeat (`up`) | host/agent masih di-scrape? | sudah ada |
| Blackbox (`probe_success`) | endpoint reachable dari luar? | sudah ada |
| **Windows Event Log** | **alasan** dari dalam OS (service crash, app exception) | **opsi ini** |

Blackbox & heartbeat bilang *mati*; Event Log bilang *mati karena apa* (mis. Event ID
7034 *"service terminated unexpectedly"*, 1000 *application error*).

## Cara kerja

Event Log **tidak** dibaca Go agent. Sebuah log shipper (**Winlogbeat**) dipasang di
tiap host Windows, membaca channel System/Application/Security, lalu mengirim ke input
**`beats`** Logstash (port 5044) — masuk ke ELK yang sama dengan log aplikasi.

```
Windows host: Event Log ──Winlogbeat──▶ Logstash :5044 (beats) ──▶ Elasticsearch (windows-eventlog-*)
demo-app:     JSON access log ─HTTP──▶ Logstash :8080 (http)  ──▶ Elasticsearch (demo-app-logs-*)
```

Routing di `deploy/elk/logstash/pipeline.conf`: event ber-field `winlog` → index
`windows-eventlog-*`; selain itu (log aplikasi) tetap `demo-app-logs-*`.

## Di host Windows asli

Pakai `deploy/elk/winlogbeat/winlogbeat.example.yml` sebagai dasar:
- Install Winlogbeat sebagai service, arahkan `output.logstash.hosts` ke VIP F5 / Logstash.
- **Server 2008**: pakai Winlogbeat **7.x** (8.x butuh Windows lebih baru) — sejalan
  dengan matriks OS di [`opsi-windows-2008.md`](opsi-windows-2008.md).

## Demo lokal (tanpa host Windows)

PoC jalan di Docker/Linux, jadi Winlogbeat asli tidak bisa. Sebagai gantinya sebuah
container **Filebeat** mengirim record sintetis berbentuk event Windows
(`deploy/elk/winlogbeat/sample-events.ndjson`) ke input `beats` yang sama:

```bash
make up-federated
make eventlog-demo
curl 'localhost:9200/_cat/indices/windows-eventlog-*?v'
curl 'localhost:9200/windows-eventlog-*/_search?q=event.code:7034&pretty'
make eventlog-demo-down
```

Di Grafana/Kibana, datasource Elasticsearch bisa diarahkan ke index
`windows-eventlog-*` untuk panel "Windows service/app errors", dikorelasikan dengan
alert heartbeat & blackbox.

## Pelengkap (opsional)

`windows_exporter` punya collector `service` → metrik state service (running/stopped)
untuk alert cepat di Prometheus, sementara isi/alasan tetap di ELK lewat Event Log.
