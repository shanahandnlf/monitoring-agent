package main

// Hand-rolled Prometheus text formatter (stdlib only, Go 1.10.8). Metric names
// and labels mirror the main agent so Prometheus/Grafana need no branching.

import (
	"sort"
	"strconv"
	"strings"
)

type metricType int

const (
	gauge metricType = iota
	counter
)

type metric struct {
	name   string
	help   string
	mtype  metricType
	value  float64
	labels map[string]string
}

func typeName(t metricType) string {
	if t == counter {
		return "counter"
	}
	return "gauge"
}

// formatMetrics renders metrics in the Prometheus text format. constLabels are
// merged into every series (host/os/zone). One HELP/TYPE block is emitted per
// metric name; series are sorted for stable, diffable output.
func formatMetrics(metrics []metric, constLabels map[string]string) string {
	sort.SliceStable(metrics, func(i, j int) bool {
		if metrics[i].name != metrics[j].name {
			return metrics[i].name < metrics[j].name
		}
		return labelString(metrics[i].labels, constLabels) < labelString(metrics[j].labels, constLabels)
	})

	var b strings.Builder
	help := make(map[string]bool)
	for _, m := range metrics {
		if !help[m.name] {
			b.WriteString("# HELP " + m.name + " " + m.help + "\n")
			b.WriteString("# TYPE " + m.name + " " + typeName(m.mtype) + "\n")
			help[m.name] = true
		}
		b.WriteString(m.name)
		b.WriteString(labelString(m.labels, constLabels))
		b.WriteString(" ")
		b.WriteString(strconv.FormatFloat(m.value, 'g', -1, 64))
		b.WriteString("\n")
	}
	return b.String()
}

// labelString merges per-metric labels with constLabels and renders the sorted
// {k="v",...} block. constLabels win on key collision is not expected; series
// labels and const labels use disjoint keys by construction.
func labelString(labels, constLabels map[string]string) string {
	merged := make(map[string]string, len(labels)+len(constLabels))
	for k, v := range constLabels {
		if v != "" {
			merged[k] = v
		}
	}
	for k, v := range labels {
		merged[k] = v
	}
	if len(merged) == 0 {
		return ""
	}

	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("{")
	for i, k := range keys {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(k)
		b.WriteString(`="`)
		b.WriteString(escapeLabelValue(merged[k]))
		b.WriteString(`"`)
	}
	b.WriteString("}")
	return b.String()
}

func escapeLabelValue(v string) string {
	v = strings.Replace(v, `\`, `\\`, -1)
	v = strings.Replace(v, `"`, `\"`, -1)
	v = strings.Replace(v, "\n", `\n`, -1)
	return v
}
