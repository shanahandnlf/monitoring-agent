# Opsi Kompatibilitas Windows Server 2008 (Linux + Windows)

Fleet target berisi Windows dan Linux dengan versi campur — ada yang terbaru, ada
yang setua **Windows Server 2008**. Agent harus bisa jalan di semuanya. Kendalanya
ada di **toolchain Go**, bukan di gopsutil: Go menghentikan dukungan OS lama, dan
gopsutil/client_golang modern menuntut Go baru.

| Target | Kernel | Go terakhir yang jalan | Bisa kompilasi gopsutil/client_golang? |
| --- | --- | --- | --- |
| Win10 / Server 2016+ & Linux baru | NT 10 | terbaru | ✅ |
| **Server 2008 R2** / Win7 / 2012 & Linux lama | NT 6.1 | **Go 1.20** | ✅ (gopsutil v3 + client_golang v1.19) |
| **Server 2008 non-R2** / Vista | NT 6.0 | **Go 1.10.8** | ❌ butuh Go ≥ 1.18 |

Sumber: <https://go.dev/wiki/MinimumRequirements>, golang/go#57003 (Go 1.21 menaikkan
minimum ke Win10/Server2016; Go 1.11 sudah drop Vista/2008-non-R2).

Karena tidak ada satu toolchain yang **jalan di non-R2** sekaligus bisa
**mengkompilasi gopsutil**, dipakai **2 artefak biner** dengan **kontrak metrik &
endpoint identik** (`/metrics`, `/healthz`, nama metrik `system_*`, label
`host`/`os`/`zone`) — Prometheus dan Grafana tidak perlu dibedakan.

## Agent A — utama (gopsutil), Go 1.20

Seluruh repo diturunkan ke `go 1.20` (lihat `go.mod`). Dependency diturunkan ke
baris yang masih kompatibel Go 1.20: gopsutil `v3.24.5`, client_golang `v1.19.1`,
OpenTelemetry `v1.21.0` (semconv `v1.21.0`), gRPC `v1.59.0`, `x/sys v0.20.0`.

- **Meng-cover**: semua Linux (lama→baru) + **Windows Server 2008 R2 sampai terbaru**.
- **Metrik**: lengkap, termasuk CPU per-core.
- Biner Go 1.20 *forward-compatible* — satu biner Windows ini juga jalan di Win10/11
  dan Server 2016/2019/2022/2025.

```bash
make build-windows   # bin/windows-amd64/agent.exe  (Server 2008 R2+)
make build-linux     # bin/linux-amd64/agent        (Linux lama→baru)
```

> Penting: target build memakai **Go 1.20** (variabel `GO ?= go1.20` di Makefile).
> Biner yang dibangun dengan Go ≥ 1.21 **tidak akan start** di Server 2008 R2.
> Install toolchain sekali:
> `go install golang.org/dl/go1.20@latest && go1.20 download`.

## Agent B — legacy untuk Server 2008 non-R2, Go 1.10.8

Program terpisah di `agent-2008/`, **hanya pustaka standar** (Go 1.10 belum punya
modules), mengumpulkan metrik via **syscall Win32 langsung**. Build mode
`GO111MODULE=off` di Docker `golang:1.10.8` (toolchain itu tidak punya biner
darwin/arm64, jadi lewat container).

```bash
make build-agent-2008
# -> bin/windows-2008/agent-2008-win32.exe   (GOARCH=386, untuk 2008 32-bit)
# -> bin/windows-2008/agent-2008-win64.exe   (GOARCH=amd64)
```

Sumber metrik (semua API ada sejak Vista/2008):

| Metrik | Win32 API |
| --- | --- |
| `system_memory_*`, `system_swap_*` | `GlobalMemoryStatusEx` (kernel32) |
| `system_cpu_usage_percent` (overall) | `GetSystemTimes` (kernel32), delta 2 scrape |
| `system_network_*` per interface | `GetIfTable` (iphlpapi) |

### Batasan Agent B (didokumentasikan, bukan bug)

- **CPU hanya overall** — tidak ada `scope="core"` (per-core butuh
  `NtQuerySystemInformation`, sengaja di-skip demi kesederhanaan & keamanan di OS tua).
- **Counter network 32-bit** (`GetIfTable` memakai DWORD) — bisa wrap di 4 GiB;
  Prometheus `rate()` menoleransinya sebagai counter reset.
- **Label `interface`** memakai *deskripsi adapter* (mis. nama model NIC), bukan nama
  ramah seperti gopsutil — penamaan bisa berbeda dari Agent A.
- `agent_collect_errors_total` tidak diekspor di Agent B.

## Konfigurasi (sama untuk kedua agent)

Env var (atau flag dengan nama sama) — lihat `internal/config/config.go`:

| Env | Flag | Default | Arti |
| --- | --- | --- | --- |
| `AGENT_LISTEN_ADDRESS` | `-listen-address` | `:9100` | alamat HTTP |
| `AGENT_ZONE` | `-zone` | `local` | label `zone` |
| `AGENT_NETWORK_SPEED_MBPS` | `-network-speed-mbps` | `1000` | speed link untuk utilization |
| `AGENT_HOSTNAME` | — | hostname OS | label `host` |
| `AGENT_NETWORK_INTERFACES` | `-network-interfaces` | semua | filter interface (Agent A) |

## Install sebagai Windows Service

Pakai **NSSM** (atau `sc.exe`) — menghindari ketergantungan tambahan, seragam untuk
kedua agent. Contoh:

```bat
nssm install MonitoringAgent C:\agent\agent.exe -listen-address=:9100 -zone=dc-jkt
nssm set MonitoringAgent AppEnvironmentExtra AGENT_HOSTNAME=srv-app-01
nssm start MonitoringAgent
```

Untuk Server 2008 non-R2 ganti `agent.exe` dengan `agent-2008-win32.exe` /
`agent-2008-win64.exe`.

## Verifikasi

1. **Kompilasi**
   - `make test` (Go 1.20) — unit test collector + config + formatter `agent-2008`.
   - `make build-windows && make build-linux && make build-agent-2008` semua sukses.
2. **Smoke test (host/Linux)** Agent A: jalankan, lalu `curl localhost:9100/metrics`
   harus memuat `system_cpu_usage_percent`, `system_memory_*`, `system_network_*`
   dengan label `host`/`os`/`zone`; `curl localhost:9100/healthz` → `ok`.
3. **Acceptance di VM nyata** (wajib sebelum rollout):
   - Agent A → VM **Server 2008 R2** + satu **Linux lama**.
   - Agent B → VM **Server 2008 non-R2**.
   - Biner start tanpa error *"not a valid Win32 application"* / missing DLL /
     entry-point; `curl http://<host>:9100/metrics` dari Prometheus; target tampil
     **`up=1`** di `/targets` (inilah heartbeat di model pull).

## Risiko (sadar dan diterima)

- Server 2008 (R2 & non-R2) sudah **EOL** (tanpa patch keamanan); Go 1.20 dan Go
  1.10 juga EOL. Ini konsekuensi melekat pada target OS lama, bukan pilihan
  implementasi.
- Mitigasi Agent B: server HTTP internal di balik F5, tidak mem-parse input
  tak-tepercaya selain HTTP header (`ReadHeaderTimeout` aktif).
- Jika ke depan stack tracing/insight sulit dipertahankan di Go 1.20, alternatif:
  isolasi agent ke modul Go 1.20 sendiri dan biarkan demo-app/insight di Go modern.
