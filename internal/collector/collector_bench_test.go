package collector

import "testing"

func benchCollectors() []Collector {
	return []Collector{
		NewCPUCollector(),
		NewMemoryCollector(),
		NewNetworkCollector(nil, 1000),
	}
}

func BenchmarkCollectAll(b *testing.B) {
	collectors := benchCollectors()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := CollectAll(collectors...); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCPUCollect(b *testing.B) {
	c := NewCPUCollector()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.Collect(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryCollect(b *testing.B) {
	c := NewMemoryCollector()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.Collect(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNetworkCollect(b *testing.B) {
	c := NewNetworkCollector(nil, 1000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.Collect(); err != nil {
			b.Fatal(err)
		}
	}
}
