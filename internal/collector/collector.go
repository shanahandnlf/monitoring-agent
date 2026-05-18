package collector

import "fmt"

type MetricType string

const (
	GaugeMetric   MetricType = "gauge"
	CounterMetric MetricType = "counter"
)

type Metric struct {
	Name   string
	Help   string
	Type   MetricType
	Value  float64
	Labels map[string]string
}

type Collector interface {
	Name() string
	Collect() ([]Metric, error)
}

func CollectAll(collectors ...Collector) ([]Metric, error) {
	metrics := make([]Metric, 0)

	for _, collector := range collectors {
		collected, err := collector.Collect()
		if err != nil {
			return nil, fmt.Errorf("collector %q: %w", collector.Name(), err)
		}

		metrics = append(metrics, collected...)
	}

	return metrics, nil
}
