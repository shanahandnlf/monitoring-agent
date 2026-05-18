package collector

import (
	"fmt"

	"github.com/shirou/gopsutil/v4/cpu"
)

type CPUCollector struct{}

func NewCPUCollector() *CPUCollector {
	return &CPUCollector{}
}

func (c *CPUCollector) Name() string {
	return "cpu"
}

func (c *CPUCollector) Collect() ([]Metric, error) {
	overall, err := cpu.Percent(0, false)
	if err != nil {
		return nil, fmt.Errorf("cpu percent overall: %w", err)
	}

	if len(overall) == 0 {
		return nil, fmt.Errorf("cpu percent overall: no samples returned")
	}

	return []Metric{
		{
			Name:  "system_cpu_usage_percent",
			Help:  "Overall CPU usage percentage.",
			Type:  GaugeMetric,
			Value: overall[0],
			Labels: map[string]string{
				"scope": "overall",
			},
		},
	}, nil
}
