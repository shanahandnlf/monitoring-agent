package main

import (
	"strings"
	"testing"
	"time"
)

func TestBuildPromptWithErrors(t *testing.T) {
	stats := logStats{
		byGroup: map[string]*groupCount{
			"zone-a/payments-api": {total: 100, errors: 15},
			"zone-b/lending-api":  {total: 80, errors: 2},
		},
		totalError: 17,
	}
	samples := []string{"[zone-a/payments-api] 500 /api/demo 210ms - GET /api/demo returned 500"}

	prompt := buildPrompt(15*time.Minute, stats, samples)

	for _, want := range []string{"zone-a/payments-api", "zone-b/lending-api", "17", "GET /api/demo returned 500"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q\n%s", want, prompt)
		}
	}
	if strings.Index(prompt, "zone-a/payments-api") > strings.Index(prompt, "zone-b/lending-api") {
		t.Error("groups not sorted ascending")
	}
}

func TestBuildPromptHealthy(t *testing.T) {
	prompt := buildPrompt(15*time.Minute, logStats{byGroup: map[string]*groupCount{}}, nil)
	if !strings.Contains(prompt, "sehat") {
		t.Errorf("healthy prompt should mention sehat, got:\n%s", prompt)
	}
}

func TestDurationToES(t *testing.T) {
	cases := map[time.Duration]string{
		15 * time.Minute: "15m",
		2 * time.Hour:    "2h",
		90 * time.Minute: "90m",
	}
	for d, want := range cases {
		if got := durationToES(d); got != want {
			t.Errorf("durationToES(%s) = %s, want %s", d, got, want)
		}
	}
}
