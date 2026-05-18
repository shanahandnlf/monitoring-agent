package collector

import (
	"fmt"

	"github.com/shirou/gopsutil/v4/mem"
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
	}, nil
}
