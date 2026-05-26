// MR Review Tracker is a tiny macOS menu-bar app that polls a Storage by Zapier
// channel and surfaces the result in your menu bar. Spiritual successor to the
// swiftbar-zapier plugin, but standalone (no SwiftBar / Node dependency).
package main

import (
	"context"
	"errors"
	"log"
	"sync/atomic"
	"time"

	"github.com/getlantern/systray"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	app := newApp(cfg)
	systray.Run(app.onReady, app.onExit)
}

type app struct {
	cfg     *Config
	menu    *Menu
	tickReq chan struct{}
	stop    chan struct{}

	// pollSeconds is read by the ticker loop without holding a lock.
	pollSeconds atomic.Int64
}

func newApp(cfg *Config) *app {
	a := &app{
		cfg:     cfg,
		tickReq: make(chan struct{}, 1),
		stop:    make(chan struct{}),
	}
	_, secs := cfg.Snapshot()
	a.pollSeconds.Store(int64(secs))
	return a
}

func (a *app) onReady() {
	a.menu = BuildMenu(a.cfg, a.requestRefresh, a.handleChangeChannel, a.handleSetInterval, a.quit)
	go a.pollLoop()
	a.requestRefresh()
}

func (a *app) onExit() {
	close(a.stop)
}

func (a *app) quit() {
	systray.Quit()
}

func (a *app) requestRefresh() {
	select {
	case a.tickReq <- struct{}{}:
	default:
	}
}

// pollLoop refreshes on demand and on a timer. The timer's interval can be
// updated at any time via setPollSeconds.
func (a *app) pollLoop() {
	a.refreshOnce()

	for {
		secs := a.pollSeconds.Load()
		if secs < int64(minPollSecs) {
			secs = int64(minPollSecs)
		}
		t := time.NewTimer(time.Duration(secs) * time.Second)

		select {
		case <-a.stop:
			t.Stop()
			return
		case <-a.tickReq:
			t.Stop()
			a.refreshOnce()
		case <-t.C:
			a.refreshOnce()
		}
	}
}

func (a *app) refreshOnce() {
	channelID, _ := a.cfg.Snapshot()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	payload, err := Fetch(ctx, channelID)
	if err != nil {
		log.Printf("fetch: %v", err)
		a.menu.SetError(err)
		return
	}
	a.menu.ApplyPayload(payload)
}

func (a *app) handleChangeChannel() {
	current, _ := a.cfg.Snapshot()
	id, err := PromptForChannelID(current)
	if err != nil {
		if errors.Is(err, ErrDialogCancelled) {
			return
		}
		log.Printf("prompt: %v", err)
		return
	}
	if id == "" || id == current {
		return
	}
	if err := a.cfg.SetChannelID(id); err != nil {
		log.Printf("save channel id: %v", err)
		ShowMessage("MR Review Tracker", "Could not save channel ID: "+err.Error())
		return
	}
	a.requestRefresh()
}

func (a *app) handleSetInterval(secs int) {
	if err := a.cfg.SetPollSeconds(secs); err != nil {
		log.Printf("save interval: %v", err)
		return
	}
	a.pollSeconds.Store(int64(secs))
	a.menu.SyncIntervalCheckmarks(secs)
	a.requestRefresh()
}
