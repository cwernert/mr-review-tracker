package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Payload is the relevant data we pull from a Storage by Zapier channel.
type Payload struct {
	Title   string
	Name    string
	Content string
}

// Fetch reads the channel record for the given UUID secret. It accepts either the
// new neutral keys (title / name / content) or the legacy swiftbar-zapier keys
// (swiftbar_title / swiftbar_name / swiftbar_content) so existing Zaps keep working.
func Fetch(ctx context.Context, channelID string) (*Payload, error) {
	if strings.TrimSpace(channelID) == "" {
		return nil, fmt.Errorf("channel id is empty")
	}

	url := fmt.Sprintf(storageURLPattern, channelID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch storage: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("storage returned %s: %s", resp.Status, truncate(string(body), 200))
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode body: %w", err)
	}

	if errVal, ok := raw["error"].(string); ok && errVal != "" {
		return nil, fmt.Errorf("storage error: %s", errVal)
	}

	p := &Payload{
		Title:   pickString(raw, "title", "swiftbar_title"),
		Name:    pickString(raw, "name", "swiftbar_name"),
		Content: pickString(raw, "content", "swiftbar_content"),
	}
	return p, nil
}

func pickString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
