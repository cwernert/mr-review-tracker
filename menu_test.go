package main

import (
	"strings"
	"testing"
	"time"
)

var refNow = time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)

func TestHasAsapLabel(t *testing.T) {
	cases := []struct {
		name   string
		labels []string
		want   bool
	}{
		{"empty", nil, false},
		{"no match", []string{"backend", "bug"}, false},
		{"match", []string{"asap-review"}, true},
		{"match mixed case", []string{"ASAP-Review"}, true},
		{"match with whitespace", []string{" asap-review "}, true},
		{"match among others", []string{"backend", "asap-review", "p1"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MR{Labels: tc.labels}.HasAsapLabel()
			if got != tc.want {
				t.Fatalf("want %v got %v", tc.want, got)
			}
		})
	}
}

func TestSeverity(t *testing.T) {
	cases := []struct {
		name    string
		updated time.Time
		want    Severity
	}{
		{"zero", time.Time{}, SevNormal},
		{"fresh", refNow.Add(-30 * time.Minute), SevNormal},
		{"just under stale", refNow.Add(-7*time.Hour - 59*time.Minute), SevNormal},
		{"exactly stale", refNow.Add(-8 * time.Hour), SevStale},
		{"borderline very stale", refNow.Add(-11*time.Hour - 59*time.Minute), SevStale},
		{"exactly very stale", refNow.Add(-12 * time.Hour), SevVeryStale},
		{"ancient", refNow.Add(-3 * 24 * time.Hour), SevVeryStale},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MR{UpdatedAt: tc.updated}.Severity(refNow)
			if got != tc.want {
				t.Fatalf("want %v got %v", tc.want, got)
			}
		})
	}
}

func TestMenuTitle(t *testing.T) {
	cases := []struct {
		name string
		mr   MR
		want string
	}{
		{
			"plain fresh",
			MR{Title: "Fix bug", UpdatedAt: refNow.Add(-1 * time.Hour)},
			"Fix bug",
		},
		{
			"asap fresh",
			MR{Title: "Fix bug", Labels: []string{"asap-review"}, UpdatedAt: refNow.Add(-1 * time.Hour)},
			"🚨 Fix bug",
		},
		{
			"stale only",
			MR{Title: "Old bug", UpdatedAt: refNow.Add(-9 * time.Hour)},
			"⏰ Old bug",
		},
		{
			"very stale only",
			MR{Title: "Ancient bug", UpdatedAt: refNow.Add(-24 * time.Hour)},
			"🔴 Ancient bug",
		},
		{
			"asap and very stale",
			MR{Title: "Critical", Labels: []string{"asap-review"}, UpdatedAt: refNow.Add(-24 * time.Hour)},
			"🚨 🔴 Critical",
		},
		{
			"draft",
			MR{Title: "WIP feature", Draft: true, UpdatedAt: refNow.Add(-1 * time.Hour)},
			"[Draft] WIP feature",
		},
		{
			"asap draft",
			MR{Title: "Urgent WIP", Labels: []string{"asap-review"}, Draft: true, UpdatedAt: refNow.Add(-1 * time.Hour)},
			"🚨 [Draft] Urgent WIP",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.mr.MenuTitle(refNow)
			if got != tc.want {
				t.Fatalf("want %q got %q", tc.want, got)
			}
		})
	}
}

func TestBarTitle(t *testing.T) {
	fresh := MR{UpdatedAt: refNow.Add(-1 * time.Hour)}
	stale := MR{UpdatedAt: refNow.Add(-9 * time.Hour)}
	asap := MR{Labels: []string{"asap-review"}, UpdatedAt: refNow.Add(-1 * time.Hour)}

	cases := []struct {
		name string
		mrs  []MR
		want string
	}{
		{"none", []MR{}, "MRs: 0"},
		{"all fresh", []MR{fresh, fresh}, "MRs: 2"},
		{"one stale", []MR{fresh, stale}, "⚠️ MRs: 2"},
		{"any asap", []MR{fresh, asap}, "🚨 MRs: 2"},
		{"asap beats stale", []MR{stale, asap}, "🚨 MRs: 2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := barTitle(tc.mrs, refNow)
			if got != tc.want {
				t.Fatalf("want %q got %q", tc.want, got)
			}
		})
	}
}

func TestTooltipContains(t *testing.T) {
	mr := MR{
		Title:     "Fix bug",
		Author:    "alice",
		Project:   "grp/proj!42",
		CreatedAt: refNow.Add(-3 * 24 * time.Hour),
		UpdatedAt: refNow.Add(-2 * time.Hour),
		Labels:    []string{"backend", "asap-review"},
		Draft:     true,
	}
	tip := mr.Tooltip(refNow)
	for _, want := range []string{
		"Author: @alice",
		"Project: grp/proj!42",
		"Updated: ",
		"2h ago",
		"Opened: ",
		"Labels: backend, asap-review",
		"Draft",
	} {
		if !strings.Contains(tip, want) {
			t.Errorf("tooltip missing %q\nfull tooltip:\n%s", want, tip)
		}
	}
}

func TestHumanizeDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "<1m"},
		{5 * time.Minute, "5m"},
		{59 * time.Minute, "59m"},
		{1 * time.Hour, "1h"},
		{23 * time.Hour, "23h"},
		{24 * time.Hour, "1d"},
		{48 * time.Hour, "2d"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := humanizeDuration(tc.d); got != tc.want {
				t.Errorf("humanizeDuration(%v) = %q; want %q", tc.d, got, tc.want)
			}
		})
	}
}
