package main

import (
	"os/exec"
	"sync"
	"time"

	"github.com/getlantern/systray"
)

const (
	defaultBarTitle = "MR Review Tracker"
	maxMRSlots      = 50 // pre-allocated slots; extras are hidden when there are fewer open MRs
)

// mrSlot is one pre-allocated menu item we repurpose on each refresh to render
// an MR. We can't add or remove items from the systray menu at runtime, so we
// allocate up front and hide/show as needed.
type mrSlot struct {
	item *systray.MenuItem
	url  string
}

// Menu owns the systray menu items and the synchronisation around updating
// them from the polling goroutine.
type Menu struct {
	cfg *Config

	mrSlots     []*mrSlot
	emptyItem   *systray.MenuItem // shown for empty list, loading, or error
	fetchedItem *systray.MenuItem // disabled "Last fetched: ..." line

	intervalParent *systray.MenuItem
	intervalItems  []intervalChoice

	mu sync.Mutex
}

type intervalChoice struct {
	secs int
	item *systray.MenuItem
}

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

// BuildMenu builds the static menu skeleton and wires up the click handlers.
// The resulting Menu can then be updated via ApplyPayload / SetError as new
// data arrives.
func BuildMenu(cfg *Config, refreshNow, onChangeChannel func(), onSetInterval func(int), onQuit func()) *Menu {
	m := &Menu{cfg: cfg, mrSlots: make([]*mrSlot, 0, maxMRSlots)}

	systray.SetTitle(defaultBarTitle)
	systray.SetTooltip("Open merge requests, courtesy of Storage by Zapier")

	for i := 0; i < maxMRSlots; i++ {
		it := systray.AddMenuItem("", "")
		it.Hide()
		slot := &mrSlot{item: it}
		m.mrSlots = append(m.mrSlots, slot)
		go func(s *mrSlot) {
			for range s.item.ClickedCh {
				if s.url != "" {
					_ = exec.Command("open", s.url).Start()
				}
			}
		}(slot)
	}

	m.emptyItem = systray.AddMenuItem("Loading…", "")
	m.emptyItem.Disable()

	m.fetchedItem = systray.AddMenuItem("Last fetched: never", "")
	m.fetchedItem.Disable()

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
		m.intervalItems = append(m.intervalItems, intervalChoice{secs: c.secs, item: it})
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

// ApplyPayload renders the MR list, updates the bar title with severity hints,
// and refreshes the "Last fetched" footer.
func (m *Menu) ApplyPayload(p *Payload) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	systray.SetTitle(barTitle(p.MRs, now))

	used := 0
	for _, mr := range p.MRs {
		if used >= len(m.mrSlots) {
			break
		}
		slot := m.mrSlots[used]
		used++
		slot.url = mr.URL
		slot.item.SetTitle(mr.MenuTitle(now))
		slot.item.SetTooltip(mr.Tooltip(now))
		if mr.URL != "" {
			slot.item.Enable()
		} else {
			slot.item.Disable()
		}
		slot.item.Show()
	}
	for i := used; i < len(m.mrSlots); i++ {
		m.mrSlots[i].item.Hide()
		m.mrSlots[i].url = ""
	}

	if used == 0 {
		m.emptyItem.SetTitle("No open MRs 🎉")
		m.emptyItem.Show()
	} else {
		m.emptyItem.Hide()
	}

	m.fetchedItem.SetTitle("Last fetched: " + formatFetched(p.FetchedAt))
}

// SetError hides the MR list and surfaces the error in the empty-state slot.
// Leaves "Last fetched" untouched so the user can still see when the most
// recent successful fetch happened.
func (m *Menu) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	systray.SetTitle("⚠️ MR Review Tracker")
	m.emptyItem.SetTitle("Error: " + truncate(err.Error(), 80))
	m.emptyItem.Show()
	for _, s := range m.mrSlots {
		s.item.Hide()
		s.url = ""
	}
}

// SyncIntervalCheckmarks ticks whichever interval matches the current poll seconds.
func (m *Menu) SyncIntervalCheckmarks(secs int) {
	for _, c := range m.intervalItems {
		if c.secs == secs {
			c.item.Check()
		} else {
			c.item.Uncheck()
		}
	}
}

func formatFetched(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Local().Format("Mon 15:04:05 MST")
}
