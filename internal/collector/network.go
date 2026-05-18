package collector

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	gopsutilnet "github.com/shirou/gopsutil/v4/net"
)

type NetworkCollector struct {
	interfaces map[string]struct{}
	speedMbps  float64
	now        func() time.Time

	mu       sync.Mutex
	previous map[string]networkSample
}

type networkSample struct {
	receivedBytes    uint64
	transmittedBytes uint64
	collectedAt      time.Time
}

func NewNetworkCollector(interfaces []string, speedMbps float64) *NetworkCollector {
	if speedMbps <= 0 {
		speedMbps = 1000
	}

	return &NetworkCollector{
		interfaces: interfaceSet(interfaces),
		speedMbps:  speedMbps,
		now:        time.Now,
		previous:   make(map[string]networkSample),
	}
}

func (c *NetworkCollector) Name() string {
	return "network"
}

func (c *NetworkCollector) Collect() ([]Metric, error) {
	counters, err := gopsutilnet.IOCounters(true)
	if err != nil {
		return nil, fmt.Errorf("network io counters: %w", err)
	}

	sort.Slice(counters, func(i, j int) bool {
		return counters[i].Name < counters[j].Name
	})

	now := c.now()
	metrics := make([]Metric, 0, len(counters)*3)
	seen := make(map[string]struct{}, len(counters))

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, counter := range counters {
		if !c.shouldCollect(counter.Name) {
			continue
		}

		seen[counter.Name] = struct{}{}

		labels := map[string]string{
			"interface": counter.Name,
		}

		metrics = append(metrics,
			Metric{
				Name:   "system_network_receive_bytes_total",
				Help:   "Total bytes received by network interface.",
				Type:   CounterMetric,
				Value:  float64(counter.BytesRecv),
				Labels: labels,
			},
			Metric{
				Name:   "system_network_transmit_bytes_total",
				Help:   "Total bytes transmitted by network interface.",
				Type:   CounterMetric,
				Value:  float64(counter.BytesSent),
				Labels: labels,
			},
		)

		current := networkSample{
			receivedBytes:    counter.BytesRecv,
			transmittedBytes: counter.BytesSent,
			collectedAt:      now,
		}

		utilizationPercent := 0.0
		if previous, ok := c.previous[counter.Name]; ok {
			utilizationPercent = calculateUtilizationPercent(previous, current, c.speedMbps)
		}
		c.previous[counter.Name] = current

		metrics = append(metrics, Metric{
			Name:   "system_network_utilization_percent",
			Help:   "Estimated network utilization percentage based on RX and TX byte deltas.",
			Type:   GaugeMetric,
			Value:  utilizationPercent,
			Labels: labels,
		})
	}

	for interfaceName := range c.previous {
		if _, ok := seen[interfaceName]; !ok {
			delete(c.previous, interfaceName)
		}
	}

	return metrics, nil
}

func (c *NetworkCollector) shouldCollect(interfaceName string) bool {
	if len(c.interfaces) == 0 {
		return true
	}

	_, ok := c.interfaces[interfaceName]
	return ok
}

func interfaceSet(interfaces []string) map[string]struct{} {
	if len(interfaces) == 0 {
		return nil
	}

	set := make(map[string]struct{}, len(interfaces))
	for _, interfaceName := range interfaces {
		interfaceName = strings.TrimSpace(interfaceName)
		if interfaceName == "" {
			continue
		}
		set[interfaceName] = struct{}{}
	}

	if len(set) == 0 {
		return nil
	}

	return set
}

func calculateUtilizationPercent(previous, current networkSample, speedMbps float64) float64 {
	elapsedSeconds := current.collectedAt.Sub(previous.collectedAt).Seconds()
	if elapsedSeconds <= 0 || speedMbps <= 0 {
		return 0
	}

	var byteDelta uint64
	if current.receivedBytes >= previous.receivedBytes {
		byteDelta += current.receivedBytes - previous.receivedBytes
	}
	if current.transmittedBytes >= previous.transmittedBytes {
		byteDelta += current.transmittedBytes - previous.transmittedBytes
	}

	bitsPerSecond := float64(byteDelta) * 8 / elapsedSeconds
	linkSpeedBitsPerSecond := speedMbps * 1_000_000

	return bitsPerSecond / linkSpeedBitsPerSecond * 100
}
