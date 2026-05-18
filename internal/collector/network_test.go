package collector

import (
	"testing"
	"time"
)

func TestCalculateUtilizationPercent(t *testing.T) {
	previous := networkSample{
		receivedBytes:    0,
		transmittedBytes: 0,
		collectedAt:      time.Unix(0, 0),
	}
	current := networkSample{
		receivedBytes:    50_000_000,
		transmittedBytes: 50_000_000,
		collectedAt:      time.Unix(1, 0),
	}

	got := calculateUtilizationPercent(previous, current, 1000)
	if got != 80 {
		t.Fatalf("calculateUtilizationPercent() = %v, want 80", got)
	}
}

func TestCalculateUtilizationPercentHandlesCounterReset(t *testing.T) {
	previous := networkSample{
		receivedBytes:    100,
		transmittedBytes: 100,
		collectedAt:      time.Unix(0, 0),
	}
	current := networkSample{
		receivedBytes:    50,
		transmittedBytes: 25,
		collectedAt:      time.Unix(1, 0),
	}

	got := calculateUtilizationPercent(previous, current, 1000)
	if got != 0 {
		t.Fatalf("calculateUtilizationPercent() = %v, want 0", got)
	}
}
