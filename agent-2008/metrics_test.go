package main

import (
	"strings"
	"testing"
)

func TestFormatMetricsMergesAndSortsLabels(t *testing.T) {
	metrics := []metric{
		{"system_cpu_usage_percent", "CPU usage percentage.", gauge, 12.5,
			map[string]string{"scope": "overall", "cpu": "all"}},
	}
	constLabels := map[string]string{"host": "h1", "os": "windows", "zone": "z1"}

	out := formatMetrics(metrics, constLabels)

	wantLine := `system_cpu_usage_percent{cpu="all",host="h1",os="windows",scope="overall",zone="z1"} 12.5`
	if !strings.Contains(out, wantLine) {
		t.Fatalf("missing expected series line.\ngot:\n%s\nwant line:\n%s", out, wantLine)
	}
	if !strings.Contains(out, "# HELP system_cpu_usage_percent CPU usage percentage.") {
		t.Fatalf("missing HELP line.\ngot:\n%s", out)
	}
	if !strings.Contains(out, "# TYPE system_cpu_usage_percent gauge") {
		t.Fatalf("missing TYPE line.\ngot:\n%s", out)
	}
}

func TestFormatMetricsSingleHelpPerName(t *testing.T) {
	metrics := []metric{
		{"system_network_receive_bytes_total", "Total bytes received by network interface.", counter, 100,
			map[string]string{"interface": "eth0"}},
		{"system_network_receive_bytes_total", "Total bytes received by network interface.", counter, 200,
			map[string]string{"interface": "eth1"}},
	}

	out := formatMetrics(metrics, nil)

	if got := strings.Count(out, "# HELP system_network_receive_bytes_total"); got != 1 {
		t.Fatalf("expected exactly 1 HELP line, got %d:\n%s", got, out)
	}
	if got := strings.Count(out, "# TYPE system_network_receive_bytes_total counter"); got != 1 {
		t.Fatalf("expected exactly 1 TYPE line, got %d:\n%s", got, out)
	}
}

func TestEscapeLabelValue(t *testing.T) {
	got := escapeLabelValue(`a"b\c` + "\n")
	want := `a\"b\\c\n`
	if got != want {
		t.Fatalf("escapeLabelValue = %q, want %q", got, want)
	}
}
