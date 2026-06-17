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

func TestCalculateCounterRate(t *testing.T) {
	got := calculateCounterRate(100, 250, time.Unix(0, 0), time.Unix(3, 0))
	if got != 50 {
		t.Fatalf("calculateCounterRate() = %v, want 50", got)
	}
}

func TestCalculateCounterRateHandlesCounterReset(t *testing.T) {
	got := calculateCounterRate(250, 100, time.Unix(0, 0), time.Unix(3, 0))
	if got != 0 {
		t.Fatalf("calculateCounterRate() = %v, want 0", got)
	}
}
