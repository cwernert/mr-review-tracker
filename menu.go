package main

import (
	"os/exec"
	"strings"
	"sync"

	"github.com/getlantern/systray"
)

const (
	defaultBarTitle  = "MR Review Tracker"
	maxContentSlots  = 50 // pre-allocated submenu lines; extras are hidden until needed
	separatorPattern = "─────────"
)

// contentSlot represents one pre-allocated submenu item we can repurpose at runtime.
type contentSlot struct {
	item *systray.MenuItem
	url  string
}

// Menu owns the systray menu items and the goroutine plumbing to keep them in sync
// with config + storage.
type Menu struct {
	cfg *Config

	statusItem *systray.MenuItem
	slots      []*contentSlot

	intervalParent *systray.MenuItem
	intervalItems  []intervalChoice

	mu          sync.Mutex
	currentBar  string
	pollSeconds int
}

type intervalChoice struct {
	label string
	secs  int
	item  *systray.MenuItem
}

// pollIntervalChoices is shown as a submenu under "Set polling interval".
var pollIntervalChoices = []struct {
	label string
	secs  int
}{
	{"5 seconds", 5},
	{"10 seconds", 10},
	{"30 seconds", 30},
	{"1 minute", 60},
	{"5 minutes", 300},
	{"15 minutes", 900},
}

// BuildMenu creates the static menu structure once and wires up the action
// channels. The returned Menu can then be updated with ApplyPayload / SetError.
func BuildMenu(cfg *Config, refreshNow func(), onChangeChannel func(), onSetInterval func(int), onQuit func()) *Menu {
	m := &Menu{cfg: cfg, slots: make([]*contentSlot, 0, maxContentSlots)}

	systray.SetTitle(defaultBarTitle)
	systray.SetTooltip("Reviews waiting on you, courtesy of Zapier Storage")

	m.statusItem = systray.AddMenuItem("Loading…", "Current status")
	m.statusItem.Disable()

	systray.AddSeparator()

	for i := 0; i < maxContentSlots; i++ {
		it := systray.AddMenuItem("", "")
		it.Hide()
		slot := &contentSlot{item: it}
		m.slots = append(m.slots, slot)
		go func(s *contentSlot) {
			for range s.item.ClickedCh {
				if s.url != "" {
					_ = exec.Command("open", s.url).Start()
				}
			}
		}(slot)
	}

	systray.AddSeparator()

	refresh := systray.AddMenuItem("Refresh now", "Fetch the latest from Storage by Zapier")
	go func() {
		for range refresh.ClickedCh {
			refreshNow()
		}
	}()

	settings := systray.AddMenuItem("Settings", "")

	changeChannel := settings.AddSubMenuItem("Change Channel ID…", "Paste a different Storage by Zapier UUID")
	go func() {
		for range changeChannel.ClickedCh {
			onChangeChannel()
		}
	}()

	m.intervalParent = settings.AddSubMenuItem("Set polling interval", "How often to poll Storage by Zapier")
	for _, choice := range pollIntervalChoices {
		c := choice
		it := m.intervalParent.AddSubMenuItem(c.label, "")
		m.intervalItems = append(m.intervalItems, intervalChoice{label: c.label, secs: c.secs, item: it})
		go func(secs int) {
			for range it.ClickedCh {
				onSetInterval(secs)
			}
		}(c.secs)
	}

	openConfig := settings.AddSubMenuItem("Open config file", "Open the JSON config in your default editor")
	go func() {
		for range openConfig.ClickedCh {
			_ = exec.Command("open", cfg.Path()).Start()
		}
	}()

	systray.AddSeparator()

	about := systray.AddMenuItem("About MR Review Tracker", "Open the project page on GitHub")
	go func() {
		for range about.ClickedCh {
			_ = exec.Command("open", "https://github.com/cwernert/mr-review-tracker").Start()
		}
	}()

	quit := systray.AddMenuItem("Quit", "Stop MR Review Tracker")
	go func() {
		<-quit.ClickedCh
		onQuit()
	}()

	_, secs := cfg.Snapshot()
	m.SyncIntervalCheckmarks(secs)

	return m
}

// ApplyPayload fills the bar title + content slots from a fetched payload.
// Lines starting with `---` become visual dividers; lines containing
// ` | href=URL ` (or `url=URL`) become clickable items that open in the browser.
func (m *Menu) ApplyPayload(p *Payload) {
	m.mu.Lock()
	defer m.mu.Unlock()

	bar, _, _, _ := parseLine(p.Title)
	bar = stripMarkdownEmphasis(bar)
	if bar == "" {
		bar = defaultBarTitle
	}
	systray.SetTitle(bar)
	m.currentBar = bar

	if p.Name != "" {
		m.statusItem.SetTitle(p.Name)
	} else {
		m.statusItem.SetTitle(bar)
	}

	lines := splitLines(p.Content)
	used := 0
	for _, line := range lines {
		if used >= len(m.slots) {
			break
		}
		text, url, isDivider, skip := parseLine(line)
		if skip {
			continue
		}
		slot := m.slots[used]
		used++
		slot.url = url
		if isDivider {
			slot.item.SetTitle(separatorPattern)
			slot.item.Disable()
		} else {
			slot.item.SetTitle(text)
			if url != "" {
				slot.item.Enable()
				slot.item.SetTooltip(url)
			} else {
				slot.item.Disable()
				slot.item.SetTooltip("")
			}
		}
		slot.item.Show()
	}
	for i := used; i < len(m.slots); i++ {
		m.slots[i].item.Hide()
		m.slots[i].url = ""
	}
}

// SetError surfaces an error in the menu without crashing the app.
func (m *Menu) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	systray.SetTitle("⚠️ MR Review Tracker")
	m.statusItem.SetTitle("Error: " + truncate(err.Error(), 80))
	for _, s := range m.slots {
		s.item.Hide()
		s.url = ""
	}
}

// SyncIntervalCheckmarks ticks whichever interval matches the current poll seconds.
func (m *Menu) SyncIntervalCheckmarks(secs int) {
	m.pollSeconds = secs
	for _, c := range m.intervalItems {
		if c.secs == secs {
			c.item.Check()
		} else {
			c.item.Uncheck()
		}
	}
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	raw := strings.Split(s, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		out = append(out, strings.TrimRight(line, "\r"))
	}
	return out
}

// stripMarkdownEmphasis removes `**bold**` / `*italic*` / `__underline__` markers
// since the systray library renders titles as plain text.
func stripMarkdownEmphasis(s string) string {
	for _, marker := range []string{"**", "__", "*", "_"} {
		s = strings.ReplaceAll(s, marker, "")
	}
	return strings.TrimSpace(s)
}

// parseLine pulls a clickable URL out of a SwiftBar-flavoured line.
//
// Examples accepted:
//
//	"Open Cursor | href=https://cursor.com" -> ("Open Cursor", "https://cursor.com", false, false)
//	"---"                                   -> ("", "", true, false)
//	""                                      -> ("", "", false, true) // skip
//
// Any unknown ` key=value ` parameters are stripped from the visible label.
func parseLine(line string) (text, url string, divider, skip bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", "", false, true
	}
	if strings.HasPrefix(trimmed, "---") {
		return "", "", true, false
	}

	// Strip leading SwiftBar indentation markers ("--", "----", etc.).
	for strings.HasPrefix(trimmed, "--") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "--"))
	}

	parts := strings.SplitN(trimmed, "|", 2)
	text = stripMarkdownEmphasis(strings.TrimSpace(parts[0]))
	if len(parts) == 2 {
		for _, kv := range strings.Fields(parts[1]) {
			eq := strings.IndexByte(kv, '=')
			if eq <= 0 {
				continue
			}
			key := strings.ToLower(kv[:eq])
			val := strings.Trim(kv[eq+1:], `"'`)
			if key == "href" || key == "url" {
				url = val
			}
		}
	}
	if text == "" {
		text = url
	}
	if text == "" {
		return "", "", false, true
	}
	return text, url, false, false
}
