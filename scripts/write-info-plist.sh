#!/usr/bin/env bash
# Writes the Info.plist for the macOS app bundle.
# Usage: write-info-plist.sh <path/to/Info.plist>
set -euo pipefail

OUT="${1:?usage: $0 <path/to/Info.plist>}"

cat >"$OUT" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>
    <string>MR Review Tracker</string>
    <key>CFBundleDisplayName</key>
    <string>MR Review Tracker</string>
    <key>CFBundleIdentifier</key>
    <string>com.cwernert.mr-review-tracker</string>
    <key>CFBundleVersion</key>
    <string>1.0.0</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0.0</string>
    <key>CFBundleExecutable</key>
    <string>mr-review-tracker</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>LSMinimumSystemVersion</key>
    <string>11.0</string>
    <!-- Hide the dock icon; only show in the menu bar -->
    <key>LSUIElement</key>
    <true/>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
PLIST
