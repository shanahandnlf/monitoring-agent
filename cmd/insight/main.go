package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type config struct {
	ListenAddress    string
	Interval         time.Duration
	LogWindow        time.Duration
	MaxLogLines      int
	ElasticsearchURL string
	OllamaURL        string
	OllamaModel      string
	OllamaTimeout    time.Duration
	DatadogAPIKey    string
	DatadogSite      string
}

type insight struct {
	cfg        config
	httpClient *http.Client
}

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)

	cfg := loadConfig()
	app := &insight{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.OllamaTimeout + 10*time.Second},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", app.handleHealthz)
	mux.HandleFunc("/summarize", app.handleSummarize)

	server := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("insight service listening on %s (model=%s, interval=%s)", cfg.ListenAddress, cfg.OllamaModel, cfg.Interval)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go app.scheduler(ctx)

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown http server: %v", err)
	}
	log.Println("insight service stopped")
}

func (a *insight) scheduler(ctx context.Context) {
	ticker := time.NewTicker(a.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.runSummary(ctx); err != nil {
				log.Printf("scheduled summary: %v", err)
			}
		}
	}
}

func (a *insight) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (a *insight) handleSummarize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := a.runSummary(r.Context()); err != nil {
		log.Printf("manual summary: %v", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func (a *insight) runSummary(ctx context.Context) error {
	started := time.Now()

	stats, samples, err := a.queryLogs(ctx)
	if err != nil {
		return fmt.Errorf("query logs: %w", err)
	}

	prompt := buildPrompt(a.cfg.LogWindow, stats, samples)

	summary, err := a.generate(ctx, prompt)
	if err != nil {
		return fmt.Errorf("ollama generate: %w", err)
	}

	doc := map[string]any{
		"@timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"type":        "log_summary",
		"model":       a.cfg.OllamaModel,
		"summary":     strings.TrimSpace(summary),
		"window":      a.cfg.LogWindow.String(),
		"error_count": stats.errorCount(),
		"duration_ms": float64(time.Since(started).Microseconds()) / 1000,
	}

	if err := a.writeInsight(ctx, doc); err != nil {
		return fmt.Errorf("write insight: %w", err)
	}

	if a.cfg.DatadogAPIKey != "" {
		if err := a.forwardToDatadog(ctx, doc); err != nil {
			log.Printf("forward to datadog: %v", err)
		} else {
			log.Printf("log summary forwarded to datadog (%s)", a.cfg.DatadogSite)
		}
	}

	log.Printf("log summary written (errors=%d, took=%s)", stats.errorCount(), time.Since(started).Round(time.Millisecond))
	return nil
}

func loadConfig() config {
	return config{
		ListenAddress:    envString("INSIGHT_LISTEN_ADDRESS", ":8090"),
		Interval:         envDuration("INSIGHT_INTERVAL", 5*time.Minute),
		LogWindow:        envDuration("INSIGHT_LOG_WINDOW", 15*time.Minute),
		MaxLogLines:      envInt("INSIGHT_MAX_LOG_LINES", 20),
		ElasticsearchURL: envString("ELASTICSEARCH_URL", "http://elasticsearch:9200"),
		OllamaURL:        envString("OLLAMA_URL", "http://ollama:11434"),
		OllamaModel:      envString("OLLAMA_MODEL", "llama3.2:3b"),
		OllamaTimeout:    envDuration("OLLAMA_TIMEOUT", 60*time.Second),
		DatadogAPIKey:    envString("DD_API_KEY", ""),
		DatadogSite:      envString("DD_SITE", "datadoghq.com"),
	}
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

type logStats struct {
	byGroup    map[string]*groupCount
	totalDocs  int
	totalError int
}

type groupCount struct {
	total  int
	errors int
}

func (s logStats) errorCount() int { return s.totalError }

func (s logStats) groups() []string {
	keys := make([]string, 0, len(s.byGroup))
	for k := range s.byGroup {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
