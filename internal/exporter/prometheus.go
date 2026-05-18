package exporter

import (
	"sort"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/shanahandnlf/monitoring-agent.git/internal/collector"
)

type PrometheusExporter struct {
	collectors  []collector.Collector
	constLabels prometheus.Labels

	mu            sync.Mutex
	collectErrors map[string]float64
}

func NewPrometheusExporter(collectors []collector.Collector, constLabels map[string]string) *PrometheusExporter {
	labels := prometheus.Labels{}
	for key, value := range constLabels {
		if value == "" {
			continue
		}
		labels[key] = value
	}

	collectErrors := make(map[string]float64, len(collectors)+1)
	for _, collector := range collectors {
		collectErrors[collector.Name()] = 0
	}
	collectErrors["exporter"] = 0

	return &PrometheusExporter{
		collectors:    collectors,
		constLabels:   labels,
		collectErrors: collectErrors,
	}
}

func (e *PrometheusExporter) Describe(ch chan<- *prometheus.Desc) {
	// Dynamic collectors are intentionally unchecked. The exporter creates
	// descriptors from the metric contract returned by each gopsutil collector.
}

func (e *PrometheusExporter) Collect(ch chan<- prometheus.Metric) {
	for _, collector := range e.collectors {
		metrics, err := collector.Collect()
		if err != nil {
			e.incrementCollectError(collector.Name())
			continue
		}

		for _, metric := range metrics {
			e.emitMetric(ch, metric)
		}
	}

	e.emitCollectErrors(ch)
}

func (e *PrometheusExporter) emitMetric(ch chan<- prometheus.Metric, metric collector.Metric) {
	labelNames, labelValues := variableLabels(metric.Labels)
	desc := prometheus.NewDesc(metric.Name, metric.Help, labelNames, e.constLabels)

	prometheusMetric, err := prometheus.NewConstMetric(
		desc,
		prometheusValueType(metric.Type),
		metric.Value,
		labelValues...,
	)
	if err != nil {
		e.incrementCollectError("exporter")
		return
	}

	ch <- prometheusMetric
}

func (e *PrometheusExporter) emitCollectErrors(ch chan<- prometheus.Metric) {
	desc := prometheus.NewDesc(
		"agent_collect_errors_total",
		"Total number of collector errors observed by the agent.",
		[]string{"collector"},
		e.constLabels,
	)

	e.mu.Lock()
	defer e.mu.Unlock()

	collectorNames := make([]string, 0, len(e.collectErrors))
	for collectorName := range e.collectErrors {
		collectorNames = append(collectorNames, collectorName)
	}
	sort.Strings(collectorNames)

	for _, collectorName := range collectorNames {
		ch <- prometheus.MustNewConstMetric(
			desc,
			prometheus.CounterValue,
			e.collectErrors[collectorName],
			collectorName,
		)
	}
}

func (e *PrometheusExporter) incrementCollectError(collectorName string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.collectErrors[collectorName]++
}

func variableLabels(labels map[string]string) ([]string, []string) {
	if len(labels) == 0 {
		return nil, nil
	}

	labelNames := make([]string, 0, len(labels))
	for labelName := range labels {
		labelNames = append(labelNames, labelName)
	}
	sort.Strings(labelNames)

	labelValues := make([]string, 0, len(labelNames))
	for _, labelName := range labelNames {
		labelValues = append(labelValues, labels[labelName])
	}

	return labelNames, labelValues
}

func prometheusValueType(metricType collector.MetricType) prometheus.ValueType {
	if metricType == collector.CounterMetric {
		return prometheus.CounterValue
	}

	return prometheus.GaugeValue
}
