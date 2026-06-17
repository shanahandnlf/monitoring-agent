package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type config struct {
	ListenAddress string
	Zone          string
	ServiceName   string
	LogstashURL   string
	ErrorRate     float64
	Hostname      string
	HostIP        string
}

type demoApp struct {
	cfg        config
	registry   *prometheus.Registry
	requests   *prometheus.CounterVec
	errors     *prometheus.CounterVec
	duration   *prometheus.HistogramVec
	httpClient *http.Client
	hostname   string
	rng        *rand.Rand
	rngMu      sync.Mutex
	nextID     atomic.Uint64
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

type accessLog struct {
	Timestamp string  `json:"@timestamp"`
	Service   string  `json:"service"`
	Zone      string  `json:"zone"`
	Hostname  string  `json:"hostname"`
	HostIP    string  `json:"host_ip"`
	RequestID string  `json:"request_id"`
	Method    string  `json:"method"`
	Path      string  `json:"path"`
	Status    int     `json:"status"`
	LatencyMS float64 `json:"latency_ms"`
	RemoteIP  string  `json:"remote_ip"`
	Message   string  `json:"message"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)

	cfg, err := loadConfig(os.Args[1:])
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	app := newDemoApp(cfg)

	shutdownTracing, err := initTracing(context.Background(), cfg, app.hostname)
	if err != nil {
		log.Printf("init tracing: %v", err)
	}
	defer func() { _ = shutdownTracing(context.Background()) }()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(app.registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", app.withAccessLog(app.handleHealthz))
	mux.HandleFunc("/api/demo", app.withAccessLog(app.handleDemo))

	server := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           instrumentHandler(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("demo app %s listening on %s in zone %s", cfg.ServiceName, cfg.ListenAddress, cfg.Zone)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http server: %v", err)
	}
}

func newDemoApp(cfg config) *demoApp {
	hostname := cfg.Hostname
	if hostname == "" {
		resolved, err := os.Hostname()
		if err != nil || resolved == "" {
			resolved = "unknown"
		}
		hostname = resolved
	}

	constLabels := prometheus.Labels{
		"zone":    cfg.Zone,
		"service": cfg.ServiceName,
		"host":    hostname,
	}

	requests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "demo_app_requests_total",
			Help:        "Total number of demo application HTTP requests.",
			ConstLabels: constLabels,
		},
		[]string{"method", "path", "status"},
	)
	errors := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "demo_app_errors_total",
			Help:        "Total number of demo application HTTP error responses.",
			ConstLabels: constLabels,
		},
		[]string{"method", "path", "status"},
	)
	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "demo_app_request_duration_seconds",
			Help:        "Demo application HTTP request duration in seconds.",
			Buckets:     prometheus.DefBuckets,
			ConstLabels: constLabels,
		},
		[]string{"method", "path", "status"},
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(requests, errors, duration)

	return &demoApp{
		cfg:      cfg,
		registry: registry,
		requests: requests,
		errors:   errors,
		duration: duration,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
		hostname: hostname,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func loadConfig(args []string) (config, error) {
	cfg := config{
		ListenAddress: envString("DEMO_LISTEN_ADDRESS", ":8080"),
		Zone:          envString("DEMO_ZONE", "local"),
		ServiceName:   envString("DEMO_SERVICE_NAME", "demo-app"),
		LogstashURL:   envString("DEMO_LOGSTASH_URL", ""),
		ErrorRate:     envFloat64("DEMO_ERROR_RATE", 0),
		Hostname:      envString("DEMO_HOSTNAME", ""),
		HostIP:        envString("DEMO_HOST_IP", ""),
	}

	flags := flag.NewFlagSet("demo-app", flag.ContinueOnError)
	flags.StringVar(&cfg.ListenAddress, "listen-address", cfg.ListenAddress, "HTTP listen address.")
	flags.StringVar(&cfg.Zone, "zone", cfg.Zone, "Logical zone label.")
	flags.StringVar(&cfg.ServiceName, "service-name", cfg.ServiceName, "Service name label for metrics and logs.")
	flags.StringVar(&cfg.LogstashURL, "logstash-url", cfg.LogstashURL, "Optional Logstash HTTP input URL for JSON access logs.")
	flags.Float64Var(&cfg.ErrorRate, "error-rate", cfg.ErrorRate, "Synthetic error rate between 0 and 1 for /api/demo.")
	flags.StringVar(&cfg.Hostname, "hostname", cfg.Hostname, "Host identity for the metrics host label and log hostname field (defaults to OS hostname).")
	flags.StringVar(&cfg.HostIP, "host-ip", cfg.HostIP, "Host IP address recorded in access logs for display.")

	if err := flags.Parse(args); err != nil {
		return config{}, err
	}

	if cfg.ListenAddress == "" {
		return config{}, fmt.Errorf("listen address cannot be empty")
	}
	if cfg.Zone == "" {
		return config{}, fmt.Errorf("zone cannot be empty")
	}
	if cfg.ServiceName == "" {
		return config{}, fmt.Errorf("service name cannot be empty")
	}
	if cfg.ErrorRate < 0 || cfg.ErrorRate > 1 {
		return config{}, fmt.Errorf("error rate must be between 0 and 1")
	}

	return cfg, nil
}

func (a *demoApp) withAccessLog(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next(recorder, r)

		latency := time.Since(started)
		status := strconv.Itoa(recorder.status)
		path := r.URL.Path

		a.requests.WithLabelValues(r.Method, path, status).Inc()
		a.duration.WithLabelValues(r.Method, path, status).Observe(latency.Seconds())
		if recorder.status >= http.StatusInternalServerError {
			a.errors.WithLabelValues(r.Method, path, status).Inc()
		}

		a.emitAccessLog(accessLog{
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Service:   a.cfg.ServiceName,
			Zone:      a.cfg.Zone,
			Hostname:  a.hostname,
			HostIP:    a.cfg.HostIP,
			RequestID: fmt.Sprintf("%s-%d", a.cfg.Zone, a.nextID.Add(1)),
			Method:    r.Method,
			Path:      path,
			Status:    recorder.status,
			LatencyMS: float64(latency.Microseconds()) / 1000,
			RemoteIP:  r.RemoteAddr,
			Message:   fmt.Sprintf("%s %s returned %d", r.Method, path, recorder.status),
		})
	}
}

func (a *demoApp) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (a *demoApp) handleDemo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	a.rngMu.Lock()
	isError := a.rng.Float64() < a.cfg.ErrorRate
	sleepFor := time.Duration(50+a.rng.Intn(250)) * time.Millisecond
	a.rngMu.Unlock()

	_, span := otel.Tracer("demo-app").Start(r.Context(), "process-demo")
	span.SetAttributes(attribute.Bool("demo.synthetic_error", isError))
	time.Sleep(sleepFor)
	span.End()

	w.Header().Set("Content-Type", "application/json")
	if isError {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":      false,
			"zone":    a.cfg.Zone,
			"service": a.cfg.ServiceName,
			"error":   "synthetic demo failure",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"zone":    a.cfg.Zone,
		"service": a.cfg.ServiceName,
	})
}

func (a *demoApp) emitAccessLog(entry accessLog) {
	payload, err := json.Marshal(entry)
	if err != nil {
		log.Printf("marshal access log: %v", err)
		return
	}

	fmt.Println(string(payload))

	if a.cfg.LogstashURL == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.LogstashURL, bytes.NewReader(payload))
	if err != nil {
		log.Printf("create logstash request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		log.Printf("post logstash event: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		log.Printf("logstash returned status %d", resp.StatusCode)
	}
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func envFloat64(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}

	return parsed
}
