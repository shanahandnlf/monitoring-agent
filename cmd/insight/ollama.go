package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (a *insight) queryLogs(ctx context.Context) (logStats, []string, error) {
	body := map[string]any{
		"size": a.cfg.MaxLogLines,
		"sort": []any{map[string]any{"@timestamp": map[string]any{"order": "desc"}}},
		"query": map[string]any{
			"range": map[string]any{
				"@timestamp": map[string]any{"gte": "now-" + durationToES(a.cfg.LogWindow)},
			},
		},
		"aggs": map[string]any{
			"by_zone_service": map[string]any{
				"multi_terms": map[string]any{
					"terms": []any{
						map[string]any{"field": "zone.keyword"},
						map[string]any{"field": "service.keyword"},
					},
					"size": 20,
				},
				"aggs": map[string]any{
					"errors": map[string]any{
						"filter": map[string]any{"range": map[string]any{"status": map[string]any{"gte": 500}}},
					},
				},
			},
			"errors_total": map[string]any{
				"filter": map[string]any{"range": map[string]any{"status": map[string]any{"gte": 500}}},
			},
		},
		"post_filter": map[string]any{"range": map[string]any{"status": map[string]any{"gte": 500}}},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return logStats{}, nil, err
	}

	url := strings.TrimRight(a.cfg.ElasticsearchURL, "/") + "/demo-app-logs-*/_search"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return logStats{}, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return logStats{}, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return logStats{}, nil, fmt.Errorf("elasticsearch status %d: %s", resp.StatusCode, string(b))
	}

	var sr struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source struct {
					Zone    string  `json:"zone"`
					Service string  `json:"service"`
					Status  int     `json:"status"`
					Path    string  `json:"path"`
					Latency float64 `json:"latency_ms"`
					Message string  `json:"message"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
		Aggregations struct {
			ErrorsTotal struct {
				DocCount int `json:"doc_count"`
			} `json:"errors_total"`
			ByZoneService struct {
				Buckets []struct {
					Key      []string `json:"key"`
					DocCount int      `json:"doc_count"`
					Errors   struct {
						DocCount int `json:"doc_count"`
					} `json:"errors"`
				} `json:"buckets"`
			} `json:"by_zone_service"`
		} `json:"aggregations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return logStats{}, nil, err
	}

	stats := logStats{byGroup: map[string]*groupCount{}}
	stats.totalError = sr.Aggregations.ErrorsTotal.DocCount
	for _, b := range sr.Aggregations.ByZoneService.Buckets {
		key := strings.Join(b.Key, "/")
		stats.byGroup[key] = &groupCount{total: b.DocCount, errors: b.Errors.DocCount}
		stats.totalDocs += b.DocCount
	}

	samples := make([]string, 0, len(sr.Hits.Hits))
	for _, h := range sr.Hits.Hits {
		s := h.Source
		samples = append(samples, fmt.Sprintf("[%s/%s] %d %s %.0fms - %s", s.Zone, s.Service, s.Status, s.Path, s.Latency, s.Message))
	}

	return stats, samples, nil
}

func (a *insight) writeInsight(ctx context.Context, doc map[string]any) error {
	payload, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	index := "monitoring-insights-" + time.Now().UTC().Format("2006.01.02")
	url := strings.TrimRight(a.cfg.ElasticsearchURL, "/") + "/" + index + "/_doc"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("elasticsearch status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (a *insight) generate(ctx context.Context, prompt string) (string, error) {
	body := map[string]any{
		"model":  a.cfg.OllamaModel,
		"prompt": prompt,
		"stream": false,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	url := strings.TrimRight(a.cfg.OllamaURL, "/") + "/api/generate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("ollama status %d: %s", resp.StatusCode, string(b))
	}

	var gr struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return "", err
	}
	return gr.Response, nil
}

func buildPrompt(window time.Duration, stats logStats, samples []string) string {
	var b strings.Builder
	b.WriteString("Kamu adalah asisten monitoring. Berdasarkan ringkasan log error aplikasi berikut, ")
	b.WriteString("buat ringkasan singkat (maksimal 5 kalimat) dalam Bahasa Indonesia: ")
	b.WriteString("sebutkan zona/service yang paling bermasalah, pola error yang terlihat, dan kemungkinan penyebabnya.\n\n")
	b.WriteString(fmt.Sprintf("Window waktu: %s terakhir.\n", window))
	b.WriteString(fmt.Sprintf("Total error 5xx: %d.\n", stats.errorCount()))

	if len(stats.byGroup) > 0 {
		b.WriteString("\nError per zona/service:\n")
		for _, g := range stats.groups() {
			c := stats.byGroup[g]
			b.WriteString(fmt.Sprintf("- %s: %d error dari %d request\n", g, c.errors, c.total))
		}
	}

	if len(samples) > 0 {
		b.WriteString("\nContoh baris log error terbaru:\n")
		for _, s := range samples {
			b.WriteString("- " + s + "\n")
		}
	}

	if stats.errorCount() == 0 {
		b.WriteString("\nTidak ada error 5xx pada window ini. Nyatakan bahwa kondisi sistem sehat.\n")
	}

	return b.String()
}

func durationToES(d time.Duration) string {
	if d%time.Hour == 0 {
		return fmt.Sprintf("%dh", int(d/time.Hour))
	}
	return fmt.Sprintf("%dm", int(d/time.Minute))
}
