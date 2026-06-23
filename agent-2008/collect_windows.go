//go:build windows
// +build windows

package main

// Win32 metric collection (stdlib only, builds on Go 1.10.8 for Server 2008
// non-R2). Per-core CPU is omitted; only overall CPU is reported.

import (
	"sort"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	iphlpapi            = syscall.NewLazyDLL("iphlpapi.dll")
	procGlobalMemStatus = kernel32.NewProc("GlobalMemoryStatusEx")
	procGetSystemTimes  = kernel32.NewProc("GetSystemTimes")
	procGetIfTable      = iphlpapi.NewProc("GetIfTable")
)

type memoryStatusEx struct {
	cbSize                  uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

type filetime struct {
	low  uint32
	high uint32
}

func (f filetime) uint64() uint64 {
	return uint64(f.high)<<32 | uint64(f.low)
}

// Win32 MIB_IFROW. Counters are 32-bit and wrap at 4 GiB (Prometheus rate() handles it).
type mibIfRow struct {
	wszName           [256]uint16
	dwIndex           uint32
	dwType            uint32
	dwMtu             uint32
	dwSpeed           uint32
	dwPhysAddrLen     uint32
	bPhysAddr         [8]byte
	dwAdminStatus     uint32
	dwOperStatus      uint32
	dwLastChange      uint32
	dwInOctets        uint32
	dwInUcastPkts     uint32
	dwInNUcastPkts    uint32
	dwInDiscards      uint32
	dwInErrors        uint32
	dwInUnknownProtos uint32
	dwOutOctets       uint32
	dwOutUcastPkts    uint32
	dwOutNUcastPkts   uint32
	dwOutDiscards     uint32
	dwOutErrors       uint32
	dwOutQLen         uint32
	dwDescrLen        uint32
	bDescr            [256]byte
}

type cpuSample struct {
	busy  uint64
	total uint64
	valid bool
}

type netSample struct {
	rxBytes   uint64
	txBytes   uint64
	rxPackets uint64
	txPackets uint64
	rxErrors  uint64
	txErrors  uint64
	at        time.Time
}

type collector struct {
	speedMbps float64
	now       func() time.Time
	prevCPU   cpuSample
	prevNet   map[string]netSample
}

func newCollector(speedMbps float64) *collector {
	return &collector{
		speedMbps: speedMbps,
		now:       time.Now,
		prevNet:   make(map[string]netSample),
	}
}

func (c *collector) collect() []metric {
	metrics := make([]metric, 0, 64)
	metrics = append(metrics, c.memoryMetrics()...)
	metrics = append(metrics, c.cpuMetrics()...)
	metrics = append(metrics, c.networkMetrics()...)
	return metrics
}

func (c *collector) memoryMetrics() []metric {
	var m memoryStatusEx
	m.cbSize = uint32(unsafe.Sizeof(m))
	r, _, _ := procGlobalMemStatus.Call(uintptr(unsafe.Pointer(&m)))
	if r == 0 {
		return nil
	}

	memUsed := m.ullTotalPhys - m.ullAvailPhys
	swapTotal := m.ullTotalPageFile
	swapFree := m.ullAvailPageFile
	var swapUsed uint64
	if swapTotal >= swapFree {
		swapUsed = swapTotal - swapFree
	}
	swapPct := 0.0
	if swapTotal > 0 {
		swapPct = float64(swapUsed) / float64(swapTotal) * 100
	}

	virt := map[string]string{"scope": "virtual"}
	swap := map[string]string{"scope": "swap"}
	return []metric{
		{"system_memory_usage_percent", "Virtual memory usage percentage.", gauge, float64(m.dwMemoryLoad), virt},
		{"system_memory_total_bytes", "Total virtual memory in bytes.", gauge, float64(m.ullTotalPhys), virt},
		{"system_memory_used_bytes", "Used virtual memory in bytes.", gauge, float64(memUsed), virt},
		{"system_memory_available_bytes", "Available virtual memory in bytes.", gauge, float64(m.ullAvailPhys), virt},
		{"system_swap_usage_percent", "Swap memory usage percentage.", gauge, swapPct, swap},
		{"system_swap_total_bytes", "Total swap memory in bytes.", gauge, float64(swapTotal), swap},
		{"system_swap_used_bytes", "Used swap memory in bytes.", gauge, float64(swapUsed), swap},
		{"system_swap_free_bytes", "Free swap memory in bytes.", gauge, float64(swapFree), swap},
	}
}

func (c *collector) cpuMetrics() []metric {
	var idle, kernel, user filetime
	r, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if r == 0 {
		return nil
	}

	// On Windows the kernel time already includes idle time.
	total := kernel.uint64() + user.uint64()
	busy := total - idle.uint64()

	current := cpuSample{busy: busy, total: total, valid: true}
	usage := 0.0
	if c.prevCPU.valid {
		dTotal := total - c.prevCPU.total
		dBusy := busy - c.prevCPU.busy
		if dTotal > 0 {
			usage = float64(dBusy) / float64(dTotal) * 100
		}
	}
	c.prevCPU = current

	return []metric{
		{"system_cpu_usage_percent", "CPU usage percentage.", gauge, usage,
			map[string]string{"scope": "overall", "cpu": "all"}},
	}
}

func (c *collector) networkMetrics() []metric {
	rows, ok := ifTable()
	if !ok {
		return nil
	}

	sort.Slice(rows, func(i, j int) bool {
		return ifName(&rows[i]) < ifName(&rows[j])
	})

	now := c.now()
	metrics := make([]metric, 0, len(rows)*11)
	seen := make(map[string]struct{}, len(rows))

	for i := range rows {
		row := &rows[i]
		name := ifName(row)
		if name == "" {
			continue
		}
		seen[name] = struct{}{}
		labels := map[string]string{"interface": name}

		rxBytes := uint64(row.dwInOctets)
		txBytes := uint64(row.dwOutOctets)
		rxPackets := uint64(row.dwInUcastPkts) + uint64(row.dwInNUcastPkts)
		txPackets := uint64(row.dwOutUcastPkts) + uint64(row.dwOutNUcastPkts)
		rxErrors := uint64(row.dwInErrors)
		txErrors := uint64(row.dwOutErrors)

		metrics = append(metrics,
			metric{"system_network_receive_bytes_total", "Total bytes received by network interface.", counter, float64(rxBytes), labels},
			metric{"system_network_transmit_bytes_total", "Total bytes transmitted by network interface.", counter, float64(txBytes), labels},
			metric{"system_network_receive_packets_total", "Total packets received by network interface.", counter, float64(rxPackets), labels},
			metric{"system_network_transmit_packets_total", "Total packets transmitted by network interface.", counter, float64(txPackets), labels},
			metric{"system_network_receive_errors_total", "Total receive errors reported by network interface.", counter, float64(rxErrors), labels},
			metric{"system_network_transmit_errors_total", "Total transmit errors reported by network interface.", counter, float64(txErrors), labels},
		)

		current := netSample{
			rxBytes: rxBytes, txBytes: txBytes,
			rxPackets: rxPackets, txPackets: txPackets,
			rxErrors: rxErrors, txErrors: txErrors,
			at: now,
		}

		util, rxPps, txPps, rxEps, txEps := 0.0, 0.0, 0.0, 0.0, 0.0
		if prev, exists := c.prevNet[name]; exists {
			util = utilizationPercent(prev, current, c.speedMbps)
			rxPps = counterRate(prev.rxPackets, current.rxPackets, prev.at, current.at)
			txPps = counterRate(prev.txPackets, current.txPackets, prev.at, current.at)
			rxEps = counterRate(prev.rxErrors, current.rxErrors, prev.at, current.at)
			txEps = counterRate(prev.txErrors, current.txErrors, prev.at, current.at)
		}
		c.prevNet[name] = current

		metrics = append(metrics,
			metric{"system_network_utilization_percent", "Estimated network utilization percentage based on RX and TX byte deltas.", gauge, util, labels},
			metric{"system_network_receive_packets_per_second", "Estimated receive packet rate by network interface.", gauge, rxPps, labels},
			metric{"system_network_transmit_packets_per_second", "Estimated transmit packet rate by network interface.", gauge, txPps, labels},
			metric{"system_network_receive_errors_per_second", "Estimated receive error rate by network interface.", gauge, rxEps, labels},
			metric{"system_network_transmit_errors_per_second", "Estimated transmit error rate by network interface.", gauge, txEps, labels},
		)
	}

	for name := range c.prevNet {
		if _, ok := seen[name]; !ok {
			delete(c.prevNet, name)
		}
	}

	return metrics
}

// ifTable calls GetIfTable twice: once to learn the buffer size, then to fill it.
func ifTable() ([]mibIfRow, bool) {
	var size uint32
	procGetIfTable.Call(0, uintptr(unsafe.Pointer(&size)), 0)
	if size == 0 {
		return nil, false
	}

	buf := make([]byte, size)
	r, _, _ := procGetIfTable.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 0)
	if r != 0 { // NO_ERROR == 0
		return nil, false
	}

	n := *(*uint32)(unsafe.Pointer(&buf[0]))
	rowSize := unsafe.Sizeof(mibIfRow{})
	base := uintptr(unsafe.Pointer(&buf[0])) + unsafe.Sizeof(uint32(0))

	rows := make([]mibIfRow, 0, int(n))
	for i := uint32(0); i < n; i++ {
		row := *(*mibIfRow)(unsafe.Pointer(base + uintptr(i)*rowSize))
		rows = append(rows, row)
	}
	return rows, true
}

// ifName uses the ANSI adapter description from GetIfTable as the interface label.
func ifName(row *mibIfRow) string {
	n := int(row.dwDescrLen)
	if n <= 0 || n > len(row.bDescr) {
		n = len(row.bDescr)
	}
	b := row.bDescr[:n]
	for len(b) > 0 && (b[len(b)-1] == 0 || b[len(b)-1] == ' ') {
		b = b[:len(b)-1]
	}
	return string(b)
}

func utilizationPercent(prev, current netSample, speedMbps float64) float64 {
	elapsed := current.at.Sub(prev.at).Seconds()
	if elapsed <= 0 || speedMbps <= 0 {
		return 0
	}
	var delta uint64
	if current.rxBytes >= prev.rxBytes {
		delta += current.rxBytes - prev.rxBytes
	}
	if current.txBytes >= prev.txBytes {
		delta += current.txBytes - prev.txBytes
	}
	bitsPerSecond := float64(delta) * 8 / elapsed
	return bitsPerSecond / (speedMbps * 1000000) * 100
}

func counterRate(prev, current uint64, prevAt, currentAt time.Time) float64 {
	elapsed := currentAt.Sub(prevAt).Seconds()
	if elapsed <= 0 || current < prev {
		return 0
	}
	return float64(current-prev) / elapsed
}
