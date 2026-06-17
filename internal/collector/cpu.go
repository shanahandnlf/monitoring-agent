package collector

import (
	"fmt"
	"strconv"

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

	perCore, err := cpu.Percent(0, true)
	if err != nil {
		return nil, fmt.Errorf("cpu percent per core: %w", err)
	}

	metrics := []Metric{
		{
			Name:  "system_cpu_usage_percent",
			Help:  "CPU usage percentage.",
			Type:  GaugeMetric,
			Value: overall[0],
			Labels: map[string]string{
				"scope": "overall",
				"cpu":   "all",
			},
		},
	}

	for index, value := range perCore {
		metrics = append(metrics, Metric{
			Name:  "system_cpu_usage_percent",
			Help:  "CPU usage percentage.",
			Type:  GaugeMetric,
			Value: value,
			Labels: map[string]string{
				"scope": "core",
				"cpu":   strconv.Itoa(index),
			},
		})
	}

	return metrics, nil
}
