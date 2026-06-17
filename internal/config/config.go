package config

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type Config struct {
	ListenAddress     string
	Zone              string
	NetworkInterfaces []string
	NetworkSpeedMbps  float64
	Hostname          string
	OperatingSystem   string
}

func Load(args []string) (Config, error) {
	return load(args, os.LookupEnv, os.Hostname)
}

func (c Config) BaseLabels() map[string]string {
	labels := map[string]string{
		"host": c.Hostname,
		"os":       c.OperatingSystem,
	}

	if c.Zone != "" {
		labels["zone"] = c.Zone
	}

	return labels
}

func load(args []string, lookupEnv func(string) (string, bool), hostname func() (string, error)) (Config, error) {
	cfg := Config{
		ListenAddress:    envString(lookupEnv, "AGENT_LISTEN_ADDRESS", ":9100"),
		Zone:             envString(lookupEnv, "AGENT_ZONE", "local"),
		NetworkSpeedMbps: envFloat64(lookupEnv, "AGENT_NETWORK_SPEED_MBPS", 1000),
		OperatingSystem:  runtime.GOOS,
	}

	cfg.NetworkInterfaces = splitCSV(envString(lookupEnv, "AGENT_NETWORK_INTERFACES", ""))

	if override := envString(lookupEnv, "AGENT_HOSTNAME", ""); override != "" {
		cfg.Hostname = override
	} else {
		resolvedHostname, err := hostname()
		if err != nil || resolvedHostname == "" {
			resolvedHostname = "unknown"
		}
		cfg.Hostname = resolvedHostname
	}

	networkInterfaces := strings.Join(cfg.NetworkInterfaces, ",")

	flags := flag.NewFlagSet("agent", flag.ContinueOnError)
	flags.StringVar(&cfg.ListenAddress, "listen-address", cfg.ListenAddress, "HTTP listen address.")
	flags.StringVar(&cfg.Zone, "zone", cfg.Zone, "Logical zone label for exported metrics.")
	flags.StringVar(&networkInterfaces, "network-interfaces", networkInterfaces, "Comma-separated network interfaces to collect. Empty means all interfaces.")
	flags.Float64Var(&cfg.NetworkSpeedMbps, "network-speed-mbps", cfg.NetworkSpeedMbps, "Assumed link speed in Mbps for network utilization calculation.")

	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}

	cfg.NetworkInterfaces = splitCSV(networkInterfaces)

	if cfg.ListenAddress == "" {
		return Config{}, fmt.Errorf("listen address cannot be empty")
	}
	if cfg.NetworkSpeedMbps <= 0 {
		return Config{}, fmt.Errorf("network speed must be greater than zero")
	}

	return cfg, nil
}

func envString(lookupEnv func(string) (string, bool), key string, fallback string) string {
	if value, ok := lookupEnv(key); ok && value != "" {
		return value
	}

	return fallback
}

func envFloat64(lookupEnv func(string) (string, bool), key string, fallback float64) float64 {
	value, ok := lookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}

	return parsed
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		values = append(values, part)
	}

	return values
}
