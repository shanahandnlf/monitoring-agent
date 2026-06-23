// Command agent-2008 is the legacy monitoring agent for Windows Server 2008
// non-R2: stdlib-only (Go 1.10.8), metrics via raw Win32 syscalls, same
// /metrics and /healthz contract as the main gopsutil agent (cmd/agent).
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type config struct {
	listenAddress string
	zone          string
	hostname      string
	speedMbps     float64
}

func loadConfig() config {
	c := config{
		listenAddress: envOr("AGENT_LISTEN_ADDRESS", ":9100"),
		zone:          envOr("AGENT_ZONE", "local"),
		speedMbps:     envFloat("AGENT_NETWORK_SPEED_MBPS", 1000),
	}

	if host := os.Getenv("AGENT_HOSTNAME"); host != "" {
		c.hostname = host
	} else if resolved, err := os.Hostname(); err == nil && resolved != "" {
		c.hostname = resolved
	} else {
		c.hostname = "unknown"
	}

	// Flags override env, matching the main agent's precedence and names.
	flag.StringVar(&c.listenAddress, "listen-address", c.listenAddress, "HTTP listen address.")
	flag.StringVar(&c.zone, "zone", c.zone, "Logical zone label for exported metrics.")
	flag.Float64Var(&c.speedMbps, "network-speed-mbps", c.speedMbps, "Assumed link speed in Mbps for network utilization calculation.")
	flag.Parse()

	if c.speedMbps <= 0 {
		c.speedMbps = 1000
	}
	return c
}

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	cfg := loadConfig()

	constLabels := map[string]string{"host": cfg.hostname, "os": "windows"}
	if cfg.zone != "" {
		constLabels["zone"] = cfg.zone
	}

	col := newCollector(cfg.speedMbps)

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		fmt.Fprint(w, formatMetrics(col.collect(), constLabels))
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok\n"))
	})

	server := &http.Server{
		Addr:              cfg.listenAddress,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("legacy monitoring agent (win2008) listening on %s", cfg.listenAddress)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("http server: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed
		}
	}
	return fallback
}
