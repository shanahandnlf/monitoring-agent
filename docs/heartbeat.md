# Indikator Heartbeat

Stack ini **pull-based**: agent tidak "mengirim" heartbeat. Sebaliknya, setiap kali
Prometheus berhasil men-scrape `/metrics` sebuah target, ia membuat metrik sintetis
**`up`**:

- `up == 1` → scrape berhasil → host hidup + agent jalan + jaringan oke (**heartbeat**).
- `up == 0` → scrape gagal.

Ini lebih andal daripada heartbeat *push*: kalau agent nge-hang tapi prosesnya masih
hidup, push palsu bisa bilang "alive", sedangkan scrape yang gagal langsung ketahuan.

## Alert

Lihat `deploy/grafana/provisioning/alerting/rules.yml` → **Agent heartbeat lost**:

```
expr: min by (zone, instance) (up{role="infrastructure"})
condition: < 1        # yaitu up == 0
for: 2m               # ~2-8 scrape terlewat, hindari false positive
noDataState: Alerting # target hilang sama sekali = heartbeat hilang
severity: critical
```

Filter `role="infrastructure"` menyasar target agent (di mode statik `job=agent`
maupun dinamis lewat label container), bukan target federation/self Prometheus.

## Menentukan penyebab (gabungkan dengan blackbox)

`up == 0` saja tidak memberi tahu *kenapa*. Kombinasikan dengan probe blackbox
(`probe_success`) yang sudah ada:

| `up` (scrape agent) | `probe_success` (ICMP/TCP ke host) | Kesimpulan |
| --- | --- | --- |
| 0 | 1 | Host hidup, **agent** bermasalah (crash/stop/port) |
| 0 | 0 | **Host atau jaringan** mati |
| 1 | — | Sehat (heartbeat normal) |
| 1 (tapi `agent_collect_errors_total` naik) | — | Agent hidup, satu collector gagal |

## Query berguna

| Apa | Query (Prometheus) |
| --- | --- |
| Target agent yang down | `up{role="infrastructure"} == 0` |
| Lama sejak scrape sukses terakhir | `time() - timestamp(up{role="infrastructure"})` |
| Jumlah agent hidup per zona | `sum by (zone) (up{role="infrastructure"})` |
| Error collector | `agent_collect_errors_total` |

## Catatan untuk skala 500 server

Heartbeat per-target ini otomatis terbentuk untuk setiap server yang di-scrape —
tidak perlu konfigurasi tambahan per server. `for: 2m` menyeimbangkan deteksi cepat
vs false positive saat scrape sesekali meleset.
