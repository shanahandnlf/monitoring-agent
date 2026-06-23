//go:build !windows
// +build !windows

package main

// Stub so the package compiles and formatter tests run on non-Windows dev machines.

type collector struct {
	speedMbps float64
}

func newCollector(speedMbps float64) *collector {
	return &collector{speedMbps: speedMbps}
}

func (c *collector) collect() []metric {
	return nil
}
