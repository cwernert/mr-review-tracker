# mr-review-tracker

A tiny native macOS menu-bar app, written in Go, that periodically polls a [Storage by Zapier](https://zapier.com/apps/storage/integrations) channel and surfaces the result in your menu bar.

It is the spiritual successor to [`swiftbar-zapier`](https://github.com/cwernert/swiftbar-zapier) — same Storage-driven workflow, no SwiftBar / Node dependency.

1. [Why?](#why)
2. [Install](#install)
    - [First launch on macOS](#first-launch-on-macos)
    - [Start at login](#start-at-login)
3. [Usage](#usage)
    - [Changing the Channel ID](#changing-the-channel-id)
    - [Setting the polling interval](#setting-the-polling-interval)
    - [Populating the Channel with content](#populating-the-channel-with-content)
4. [How it works](#how-it-works)
5. [Uninstall](#uninstall)
6. [Development](#development)

## Why?

The original `swiftbar-zapier` plugin works great, but it requires SwiftBar, Node, and a small constellation of shell scripts. `mr-review-tracker` is a single Go binary packaged as a regular `.app` that lives in your menu bar. The default channel UUID it ships with is the "Integrations MR Reviews" channel — hence the name — but it works with any Storage by Zapier UUID.

## Install

1. Head to the [Releases page](https://github.com/cwernert/mr-review-tracker/releases/latest) and download the zip for your Mac:
    - **Apple Silicon** (M1/M2/M3/M4): `mr-review-tracker-arm64.zip`
    - **Intel**: `mr-review-tracker-amd64.zip`
2. Unzip it. You'll get `MR Review Tracker.app`.
3. Drag the `.app` into `~/Applications` (or `/Applications`).
4. Launch it — see [First launch on macOS](#first-launch-on-macos) below.

### First launch on macOS

Because the app is **ad-hoc signed** (no $99/yr Apple Developer cert), macOS Gatekeeper will refuse to open it the first time with something like _"MR Review Tracker can't be opened because Apple cannot check it for malicious software"_. This is normal for open-source Mac apps.

Three ways to get past it (any one works, you only need to do it once):

| Method | How |
|---|---|
| **Right-click → Open** | In Finder, right-click `MR Review Tracker.app` → **Open** → **Open** in the dialog |
| **System Settings** | After macOS blocks it, go to **System Settings → Privacy & Security**, scroll down, click **Open Anyway** |
| **Terminal** | `xattr -d com.apple.quarantine "$HOME/Applications/MR Review Tracker.app"` then launch normally |

After the first launch, double-clicking works forever.

### Start at login

To have it launch automatically when you sign in:

1. **System Settings → General → Login Items & Extensions**
2. Under **Open at Login**, click **+**, pick `MR Review Tracker.app`, then **Open**.

## Usage

Once running you should see the title (default: `MR Review Tracker`) in your macOS menu bar. Click it to open the submenu. If you have a MacBook with a notch and lots of menu-bar items, the title may be hidden behind the notch — apps like [Ice](https://github.com/jordanbaird/Ice) or **Bartender** can surface it.

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
- Release builds are produced per architecture (`arm64`, `amd64`) on GitHub-hosted macOS runners, ad-hoc codesigned, and packaged with `ditto` so resource forks + extended attributes survive the round trip.

## Uninstall

If you cloned the repo:

```sh
make uninstall
```

…or, manually:

```sh
pkill -x mr-review-tracker 2>/dev/null
rm -rf "$HOME/Applications/MR Review Tracker.app"
rm -rf "$HOME/Library/Application Support/mr-review-tracker"
launchctl unload "$HOME/Library/LaunchAgents/com.cwernert.mr-review-tracker.plist" 2>/dev/null
rm -f "$HOME/Library/LaunchAgents/com.cwernert.mr-review-tracker.plist"
```

(The LaunchAgent paths only matter if you set one up yourself — the app doesn't ship one.)

## Development

```sh
git clone git@github.com:cwernert/mr-review-tracker.git
cd mr-review-tracker
make test          # run unit tests
make run           # build + run the binary in foreground (^C to quit)
make app           # produce dist/<arch>/MR Review Tracker.app (ad-hoc signed)
make zip           # produce dist/mr-review-tracker-<arch>.zip (ditto-packed)
make install       # build + replace ~/Applications/MR Review Tracker.app + launch
make stop          # kill any running copy of the app
make uninstall     # remove app, config, and any LaunchAgent
```

Requires Go ≥ 1.20 and the Xcode Command Line Tools (cgo links against Cocoa).

`make app` defaults to the host architecture; override with `ARCH=amd64` (or `ARCH=arm64`) to target the other. Cross-arch builds need a native toolchain, so CI uses arch-matched runners (`macos-15` for arm64, `macos-15-intel` for amd64) rather than cross-compiling.

### Cutting a release

```sh
git tag v0.1.0
git push origin v0.1.0
```

The [`Release` workflow](.github/workflows/release.yml) takes over from there:

1. Builds `MR Review Tracker.app` on `macos-15` (arm64) and `macos-15-intel` (amd64) in parallel.
2. Ad-hoc codesigns each bundle.
3. Packages each into `mr-review-tracker-<arch>.zip` with `ditto`.
4. Creates / updates the GitHub Release for that tag and attaches both zips.

You can also trigger it manually from the **Actions** tab via **workflow_dispatch** with an explicit tag name.
