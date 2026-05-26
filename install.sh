#!/usr/bin/env bash
# install.sh — fetch source, build, and install MR Review Tracker into ~/Applications.
# Optionally registers a LaunchAgent so it starts at login.
#
# Usage:
#   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/cwernert/mr-review-tracker/main/install.sh)"
set -euo pipefail

clear

RED='\033[0;31m'
GRN='\033[0;32m'
YEL='\033[0;33m'
NC='\033[0m'

REPO_URL="https://github.com/cwernert/mr-review-tracker.git"
REPO_BRANCH="main"
APP_NAME="MR Review Tracker"
BUNDLE_ID="com.cwernert.mr-review-tracker"
WORK_DIR="$(mktemp -d -t mr-review-tracker-install.XXXXXX)"
INSTALL_DIR="$HOME/Applications"
APP_PATH="$INSTALL_DIR/${APP_NAME}.app"
LAUNCH_AGENT="$HOME/Library/LaunchAgents/${BUNDLE_ID}.plist"

trap 'rm -rf "$WORK_DIR"' EXIT

echo "This script will install ${APP_NAME} into ${INSTALL_DIR}."
echo "If a previous version is running it will be quit and replaced."
read -p "Are you ready to continue? (y/n) " yn
case $yn in
    [yY]*) echo "Installing ${APP_NAME}..." ;;
    *)     echo "Installation aborted; exiting."; exit 0 ;;
esac

# Quit any running instance so we can replace the binary.
pkill -x "mr-review-tracker" 2>/dev/null || true
osascript -e "tell application \"${APP_NAME}\" to quit" 2>/dev/null || true

echo "Checking dependencies..."
installed="Installed:"
required="Required:"
reqCount=0

# Homebrew
if which -s brew >/dev/null 2>&1; then
    installed="${installed} homebrew"
else
    required="${required} homebrew"
    reqCount=$((reqCount+1))
fi

# Go
if which go >/dev/null 2>&1; then
    goversion="$(go version | awk '{print $3}' | sed 's/^go//')"
    goMajor="$(echo "$goversion" | cut -d. -f1)"
    goMinor="$(echo "$goversion" | cut -d. -f2)"
    if [[ "$goMajor" -gt 1 ]] || { [[ "$goMajor" -eq 1 ]] && [[ "$goMinor" -ge 20 ]]; }; then
        installed="${installed} go(${goversion})"
    else
        echo -e "${RED}Go ${goversion} detected; need >= 1.20.${NC}"
        required="${required} go"
        reqCount=$((reqCount+1))
    fi
else
    required="${required} go"
    reqCount=$((reqCount+1))
fi

# Xcode CLT (needed for Cgo against Cocoa)
if xcode-select -p >/dev/null 2>&1; then
    installed="${installed} xcode-clt"
else
    required="${required} xcode-clt"
    reqCount=$((reqCount+1))
fi

if [[ "$reqCount" -gt 0 ]]; then
    echo -e "${YEL}Some dependencies are missing.${NC}"
    echo "  ${installed}"
    echo "  ${required}"
    read -p "Install the missing dependencies now? (y/n) " yn
    case $yn in
        [yY]*) ;;
        *) echo "Installation aborted; exiting."; exit 0 ;;
    esac

    if echo "$required" | grep -q 'homebrew'; then
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install.sh)"
    fi
    if echo "$required" | grep -q 'xcode-clt'; then
        echo "Triggering Xcode Command Line Tools install (a GUI prompt will appear)..."
        xcode-select --install || true
        echo "Press <Return> once the installer has finished."
        read -r _
    fi
    if echo "$required" | grep -q 'go'; then
        brew install go
    fi
else
    echo -e "${GRN}All dependencies are installed.${NC}"
    echo "  ${installed}"
fi

echo "Cloning ${REPO_URL}..."
git clone --depth 1 --branch "${REPO_BRANCH}" "${REPO_URL}" "${WORK_DIR}/src"

echo "Building..."
( cd "${WORK_DIR}/src" && make app )

echo "Copying to ${APP_PATH}..."
mkdir -p "${INSTALL_DIR}"
rm -rf "${APP_PATH}"
cp -R "${WORK_DIR}/src/dist/${APP_NAME}.app" "${INSTALL_DIR}/"

read -p "Start ${APP_NAME} at login? (y/n) " yn
if [[ "$yn" =~ ^[yY] ]]; then
    mkdir -p "$(dirname "${LAUNCH_AGENT}")"
    cat >"${LAUNCH_AGENT}" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${BUNDLE_ID}</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/bin/open</string>
        <string>-a</string>
        <string>${APP_PATH}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>
PLIST
    launchctl unload "${LAUNCH_AGENT}" 2>/dev/null || true
    launchctl load "${LAUNCH_AGENT}"
    echo -e "${GRN}LaunchAgent installed at ${LAUNCH_AGENT}${NC}"
fi

echo -e "${GRN}Launching ${APP_NAME}...${NC}"
open "${APP_PATH}"

echo "Done. Look for the title in your menu bar."
echo "If you don't see it, you can change the channel UUID and polling interval"
echo "by clicking the title once it appears -> Settings."
