package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// supportedSchemaVersion is the wire-format version this binary understands.
// Bump this in lock-step with the `v` field in zap/produce-storage.js whenever
// the on-the-wire shape changes.
const supportedSchemaVersion = 1

// Payload is the slim shape produced by zap/produce-storage.js and stored in
// the Storage by Zapier channel that backs the app.
type Payload struct {
	Version   int       `json:"v"`
	FetchedAt time.Time `json:"fetched_at"`
	MRs       []MR      `json:"mrs"`
}

// MR is one merge request as the menu needs to render it. Anything the menu
// does not display is intentionally absent so we don't bloat the Storage
// record.
type MR struct {
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Project   string    `json:"project"` // e.g. "group/project!123"
	Author    string    `json:"author"`
	AuthorURL string    `json:"author_url"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
	Labels    []string  `json:"labels"`
	Draft     bool      `json:"draft"`
}

// Severity buckets an MR by how long it has been idle.
type Severity int

const (
	SevNormal Severity = iota
	SevStale            // updated_at >= 8h ago
	SevVeryStale        // updated_at >= 12h ago
)

const (
	asapLabel            = "asap-review"
	staleThresholdHours  = 8.0
	veryStaleThresholdHr = 12.0
)

// Fetch reads the channel record and decodes it into Payload. Returns an
// actionable error if the schema version is missing or wrong, so the menu
// can prompt the user to update the producer Zap.
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("storage returned %s", resp.Status)
	}

	var p Payload
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, fmt.Errorf("decode body: %w", err)
	}

	if p.Version != supportedSchemaVersion {
		return nil, fmt.Errorf("storage schema v%d not supported — update the producer Zap (need v%d)", p.Version, supportedSchemaVersion)
	}

	return &p, nil
}

// HasAsapLabel reports whether the MR carries the "asap-review" label
// (case-insensitive, whitespace-tolerant).
func (m MR) HasAsapLabel() bool {
	for _, l := range m.Labels {
		if strings.EqualFold(strings.TrimSpace(l), asapLabel) {
			return true
		}
	}
	return false
}

// Severity returns the staleness bucket of the MR based on UpdatedAt.
// A zero UpdatedAt is treated as Normal so missing data never looks urgent.
func (m MR) Severity(now time.Time) Severity {
	if m.UpdatedAt.IsZero() {
		return SevNormal
	}
	hours := now.Sub(m.UpdatedAt).Hours()
	switch {
	case hours >= veryStaleThresholdHr:
		return SevVeryStale
	case hours >= staleThresholdHours:
		return SevStale
	default:
		return SevNormal
	}
}

// MenuTitle is the line shown in the menu for this MR. Marker order:
//
//	🚨   asap-review label present
//	🔴   updated_at >= 12h ago (very stale)
//	⏰   updated_at >= 8h ago  (stale)
//	[Draft]  WIP / draft MR
//
// Multiple markers compose, e.g. "🚨 🔴 [Draft] Critical fix".
func (m MR) MenuTitle(now time.Time) string {
	var b strings.Builder
	if m.HasAsapLabel() {
		b.WriteString("🚨 ")
	}
	switch m.Severity(now) {
	case SevVeryStale:
		b.WriteString("🔴 ")
	case SevStale:
		b.WriteString("⏰ ")
	}
	if m.Draft {
		b.WriteString("[Draft] ")
	}
	b.WriteString(m.Title)
	return b.String()
}

// Tooltip is the multi-line metadata blob shown on hover.
func (m MR) Tooltip(now time.Time) string {
	lines := []string{
		"Author: @" + m.Author,
		"Project: " + m.Project,
		"Updated: " + humanizeUpdated(m.UpdatedAt, now),
	}
	if !m.CreatedAt.IsZero() {
		lines = append(lines, "Opened: "+m.CreatedAt.Local().Format("2006-01-02"))
	}
	if len(m.Labels) > 0 {
		lines = append(lines, "Labels: "+strings.Join(m.Labels, ", "))
	}
	if m.Draft {
		lines = append(lines, "Draft")
	}
	return strings.Join(lines, "\n")
}

// barTitle computes the menu-bar title for a list of MRs, prefixing with
// urgency icons if anything is asap-flagged or stale. asap beats stale.
func barTitle(mrs []MR, now time.Time) string {
	hasAsap, hasStale := false, false
	for _, mr := range mrs {
		if mr.HasAsapLabel() {
			hasAsap = true
		}
		if mr.Severity(now) != SevNormal {
			hasStale = true
		}
	}
	title := fmt.Sprintf("MRs: %d", len(mrs))
	switch {
	case hasAsap:
		return "🚨 " + title
	case hasStale:
		return "⚠️ " + title
	default:
		return title
	}
}

func humanizeUpdated(t, now time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	stamp := t.Local().Format("2006-01-02 15:04")
	return fmt.Sprintf("%s (%s ago)", stamp, humanizeDuration(now.Sub(t)))
}

func humanizeDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "<1m"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours())/24)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
