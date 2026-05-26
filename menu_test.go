package main

import "testing"

func TestParseLine(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		wantText  string
		wantURL   string
		wantDiv   bool
		wantSkip  bool
	}{
		{name: "plain", in: "Hello world", wantText: "Hello world"},
		{name: "empty", in: "   ", wantSkip: true},
		{name: "divider", in: "---", wantDiv: true},
		{name: "long divider", in: "------", wantDiv: true},
		{name: "bold stripped", in: "**MRs:** 0", wantText: "MRs: 0"},
		{
			name:     "with href",
			in:       "Open Cursor | href=https://cursor.com",
			wantText: "Open Cursor",
			wantURL:  "https://cursor.com",
		},
		{
			name:     "with url alias",
			in:       "Open issue | url=https://example.com/1",
			wantText: "Open issue",
			wantURL:  "https://example.com/1",
		},
		{
			name:     "with extra params",
			in:       "Status | href=https://x.io color=red font=Menlo",
			wantText: "Status",
			wantURL:  "https://x.io",
		},
		{
			name:     "submenu indent stripped",
			in:       "-- Sub item",
			wantText: "Sub item",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text, url, div, skip := parseLine(tc.in)
			if text != tc.wantText || url != tc.wantURL || div != tc.wantDiv || skip != tc.wantSkip {
				t.Fatalf("parseLine(%q) = (%q, %q, %v, %v); want (%q, %q, %v, %v)",
					tc.in, text, url, div, skip, tc.wantText, tc.wantURL, tc.wantDiv, tc.wantSkip)
			}
		})
	}
}

func TestStripMarkdownEmphasis(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"**bold**", "bold"},
		{"_italic_", "italic"},
		{"__under__", "under"},
		{"plain", "plain"},
		{"  **MRs:** 0  ", "MRs: 0"},
	} {
		got := stripMarkdownEmphasis(tc.in)
		if got != tc.want {
			t.Errorf("stripMarkdownEmphasis(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

func TestPickStringFallback(t *testing.T) {
	m := map[string]any{
		"swiftbar_title": "Old key wins when new key absent",
	}
	got := pickString(m, "title", "swiftbar_title")
	if got != "Old key wins when new key absent" {
		t.Fatalf("got %q", got)
	}

	m2 := map[string]any{
		"title":          "New key wins",
		"swiftbar_title": "Should be ignored",
	}
	if got := pickString(m2, "title", "swiftbar_title"); got != "New key wins" {
		t.Fatalf("got %q", got)
	}

	if got := pickString(map[string]any{}, "title"); got != "" {
		t.Fatalf("got %q for empty map", got)
	}
}
