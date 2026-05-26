# mr-review-tracker

A tiny native macOS menu-bar app, written in Go, that periodically polls a [Storage by Zapier](https://zapier.com/apps/storage/integrations) channel and surfaces the result in your menu bar.

It is the spiritual successor to [`swiftbar-zapier`](https://github.com/cwernert/swiftbar-zapier) — same Storage-driven workflow, no SwiftBar / Node dependency.

1. [Why?](#why)
2. [Installation](#installation)
3. [Usage](#usage)
    - [Changing the Channel ID](#changing-the-channel-id)
    - [Setting the polling interval](#setting-the-polling-interval)
    - [Populating the Channel with content](#populating-the-channel-with-content)
4. [How it works](#how-it-works)
5. [Uninstallation](#uninstallation)
6. [Development](#development)

## Why?

The original `swiftbar-zapier` plugin works great, but it requires SwiftBar, Node, and a small constellation of shell scripts. `mr-review-tracker` is a single Go binary packaged as a regular `.app` that lives in your menu bar. The default channel UUID it ships with is the "Integrations MR Reviews" channel — hence the name — but it works with any Storage by Zapier UUID.

## Installation

Run this in Terminal:

> `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/cwernert/mr-review-tracker/main/install.sh)"`

The script will:

1. Confirm you're ready to proceed
2. Quit any running copy of the app
3. Check / install dependencies via Homebrew (`go`, Xcode Command Line Tools)
4. Clone the repo, run `make app` to produce `MR Review Tracker.app`
5. Copy the bundle into `~/Applications`
6. Optionally register a `~/Library/LaunchAgents` plist so it starts at login
7. Launch the app

You can review the script before running it: [install.sh](install.sh).

## Usage

Once installed you should see the title (default: `MR Review Tracker`) in your macOS menu bar. Click it to open the submenu.

The default Channel ID is `fbdee107-eb6b-47d5-ba86-0aaa8a41b813` (the "Integrations MR Reviews" channel). You can change it at any time — see below.

### Changing the Channel ID

1. Click the title in your menu bar
2. Settings → Change Channel ID…
3. Paste a new UUID and hit Save

The config is stored at:

> `~/Library/Application Support/mr-review-tracker/config.json`

You can also click **Settings → Open config file** to edit it directly.

### Setting the polling interval

1. Click the title in your menu bar
2. Settings → Set polling interval
3. Pick one of `5s / 10s / 30s / 1m / 5m / 15m`

The current interval is shown with a checkmark. Click "Refresh now" to force an immediate fetch.

### Populating the Channel with content

The app reads from `https://store.zapier.com/api/records?secret=<UUID>` and expects a JSON object with these keys:

```json
{
  "title":   "MRs: 0",
  "name":    "Integrations MR Reviews",
  "content": "Last updated: 2026-05-26T00:51:43+00:00\n---\nOpen dashboard | href=https://gitlab.com/dashboard/merge_requests"
}
```

For backwards compatibility with existing SwiftBar Zaps, the legacy `swiftbar_title`, `swiftbar_name`, and `swiftbar_content` keys are also supported.

`content` is split on newlines:

| Pattern in a content line | Effect |
|---|---|
| `text` | plain disabled menu item |
| `text \| href=URL` (or `url=`) | clickable item that opens the URL in your default browser |
| `---` (or longer) | visual divider |
| `**bold**`, `_italic_` markers | stripped (the menu bar renders plain text) |
| any other ` key=value ` parameters | ignored |

A "Set Channel Title & Content in SwiftBar" Zap action remains the simplest way to feed the channel; just use the same Channel UUID here.

## How it works

```
┌──────────────────────────┐    every N seconds   ┌────────────────────────────┐
│ mr-review-tracker (menu) │ ───────────────────► │ store.zapier.com/api/records│
└──────────────┬───────────┘                      └──────────────┬─────────────┘
               │                                                 │
               ▼                                                 ▼
      menubar + submenu                              JSON {title, name, content}
```

- Single static binary, ~7 MB.
- Menu bar UI is provided by [`getlantern/systray`](https://github.com/getlantern/systray).
- Polling lives in a single goroutine and is interrupted by an explicit "Refresh now" or by an interval change — no busy-waiting.
- Settings dialogs are native macOS dialogs via `osascript`.

## Uninstallation

```sh
make uninstall
```

…or, manually:

```sh
rm -rf "$HOME/Applications/MR Review Tracker.app"
rm -rf "$HOME/Library/Application Support/mr-review-tracker"
launchctl unload "$HOME/Library/LaunchAgents/com.cwernert.mr-review-tracker.plist" 2>/dev/null
rm -f "$HOME/Library/LaunchAgents/com.cwernert.mr-review-tracker.plist"
```

## Development

```sh
git clone git@github.com:cwernert/mr-review-tracker.git
cd mr-review-tracker
make test         # run unit tests
make run          # build + run the binary in foreground (^C to quit)
make app          # produce dist/MR Review Tracker.app
make install      # copy the .app into ~/Applications
```

Requires Go ≥ 1.20 and the Xcode Command Line Tools (Cgo links against Cocoa).
