package config

import "testing"

func TestLoadUsesFlags(t *testing.T) {
	cfg, err := load(
		[]string{
			"-listen-address", ":9200",
			"-zone", "dev",
			"-network-interfaces", "en0, eth0",
			"-network-speed-mbps", "2500",
		},
		func(string) (string, bool) { return "", false },
		func() (string, error) { return "test-host", nil },
	)
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}

	if cfg.ListenAddress != ":9200" {
		t.Fatalf("ListenAddress = %q, want :9200", cfg.ListenAddress)
	}
	if cfg.Zone != "dev" {
		t.Fatalf("Zone = %q, want dev", cfg.Zone)
	}
	if cfg.NetworkSpeedMbps != 2500 {
		t.Fatalf("NetworkSpeedMbps = %v, want 2500", cfg.NetworkSpeedMbps)
	}
	if len(cfg.NetworkInterfaces) != 2 || cfg.NetworkInterfaces[0] != "en0" || cfg.NetworkInterfaces[1] != "eth0" {
		t.Fatalf("NetworkInterfaces = %#v, want [en0 eth0]", cfg.NetworkInterfaces)
	}
}

func TestLoadUsesEnvironmentDefaults(t *testing.T) {
	env := map[string]string{
		"AGENT_LISTEN_ADDRESS":     ":9300",
		"AGENT_ZONE":               "local-dev",
		"AGENT_NETWORK_INTERFACES": "lo0",
		"AGENT_NETWORK_SPEED_MBPS": "100",
	}

	cfg, err := load(
		nil,
		func(key string) (string, bool) {
			value, ok := env[key]
			return value, ok
		},
		func() (string, error) { return "env-host", nil },
	)
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}

	if cfg.ListenAddress != ":9300" {
		t.Fatalf("ListenAddress = %q, want :9300", cfg.ListenAddress)
	}
	if cfg.Zone != "local-dev" {
		t.Fatalf("Zone = %q, want local-dev", cfg.Zone)
	}
	if cfg.NetworkSpeedMbps != 100 {
		t.Fatalf("NetworkSpeedMbps = %v, want 100", cfg.NetworkSpeedMbps)
	}
	if len(cfg.NetworkInterfaces) != 1 || cfg.NetworkInterfaces[0] != "lo0" {
		t.Fatalf("NetworkInterfaces = %#v, want [lo0]", cfg.NetworkInterfaces)
	}
}
