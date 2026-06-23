package collector

import (
	"fmt"

	"github.com/shirou/gopsutil/v3/mem"
)

type MemoryCollector struct{}

func NewMemoryCollector() *MemoryCollector {
	return &MemoryCollector{}
}

func (c *MemoryCollector) Name() string {
	return "memory"
}

func (c *MemoryCollector) Collect() ([]Metric, error) {
	virtualMemory, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("virtual memory: %w", err)
	}

	swapMemory, err := mem.SwapMemory()
	if err != nil {
		return nil, fmt.Errorf("swap memory: %w", err)
	}

	return []Metric{
		{
			Name:  "system_memory_usage_percent",
			Help:  "Virtual memory usage percentage.",
			Type:  GaugeMetric,
			Value: virtualMemory.UsedPercent,
			Labels: map[string]string{
				"scope": "virtual",
			},
		},
		{
			Name:  "system_memory_total_bytes",
			Help:  "Total virtual memory in bytes.",
			Type:  GaugeMetric,
			Value: float64(virtualMemory.Total),
			Labels: map[string]string{
				"scope": "virtual",
			},
		},
		{
			Name:  "system_memory_used_bytes",
			Help:  "Used virtual memory in bytes.",
			Type:  GaugeMetric,
			Value: float64(virtualMemory.Used),
			Labels: map[string]string{
				"scope": "virtual",
			},
		},
		{
			Name:  "system_memory_available_bytes",
			Help:  "Available virtual memory in bytes.",
			Type:  GaugeMetric,
			Value: float64(virtualMemory.Available),
			Labels: map[string]string{
				"scope": "virtual",
			},
		},
		{
			Name:  "system_swap_usage_percent",
			Help:  "Swap memory usage percentage.",
			Type:  GaugeMetric,
			Value: swapMemory.UsedPercent,
			Labels: map[string]string{
				"scope": "swap",
			},
		},
		{
			Name:  "system_swap_total_bytes",
			Help:  "Total swap memory in bytes.",
			Type:  GaugeMetric,
			Value: float64(swapMemory.Total),
			Labels: map[string]string{
				"scope": "swap",
			},
		},
		{
			Name:  "system_swap_used_bytes",
			Help:  "Used swap memory in bytes.",
			Type:  GaugeMetric,
			Value: float64(swapMemory.Used),
			Labels: map[string]string{
				"scope": "swap",
			},
		},
		{
			Name:  "system_swap_free_bytes",
			Help:  "Free swap memory in bytes.",
			Type:  GaugeMetric,
			Value: float64(swapMemory.Free),
			Labels: map[string]string{
				"scope": "swap",
			},
		},
	}, nil
}
