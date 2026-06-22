package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (a *insight) forwardToDatadog(ctx context.Context, doc map[string]any) error {
	event := map[string]any{
		"ddsource":    "insight",
		"service":     "monitoring-insight",
		"ddtags":      "env:poc,type:log_summary",
		"message":     doc["summary"],
		"model":       doc["model"],
		"window":      doc["window"],
		"error_count": doc["error_count"],
	}

	payload, err := json.Marshal([]any{event})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://http-intake.logs.%s/api/v2/logs", a.cfg.DatadogSite)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", a.cfg.DatadogAPIKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("datadog intake status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
