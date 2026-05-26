BINARY := mr-review-tracker
APP_NAME := MR Review Tracker

# ARCH defaults to the host architecture (arm64 on Apple Silicon, amd64 on Intel).
# Override to build a bundle for a different arch, e.g. `make app ARCH=amd64`.
# Cross-arch builds require native toolchain support; CI uses arch-matched runners.
HOST_ARCH := $(shell uname -m | sed 's/x86_64/amd64/')
ARCH ?= $(HOST_ARCH)

APP_BUNDLE_DIR := dist/$(ARCH)
APP_BUNDLE := $(APP_BUNDLE_DIR)/$(APP_NAME).app
APP_ZIP := dist/$(BINARY)-$(ARCH).zip
INSTALL_DIR := $(HOME)/Applications
INSTALLED_APP := $(INSTALL_DIR)/$(APP_NAME).app

GO ?= go
LDFLAGS := -s -w

.PHONY: all build test clean app zip install uninstall run stop

all: build

## build: compile a plain binary in the repo root (for local dev / `make run`)
build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## test: run unit tests
test:
	$(GO) test ./...

## stop: kill any running copy of the app (no-op if nothing is running)
stop:
	-@pkill -x $(BINARY) 2>/dev/null && echo "Stopped running $(BINARY) instance(s)" || echo "No running $(BINARY) to stop"
	@sleep 1

## run: build and execute the binary in-place (foreground, ^C to quit).
## Kills any other running copy first so you never end up with two icons.
run: build stop
	./$(BINARY)

## app: assemble a macOS .app bundle for $(ARCH) under ./dist/$(ARCH)/.
## Compiles fresh into the bundle and ad-hoc codesigns it so Gatekeeper
## treats it as a real (locally-trusted) app on first launch.
app:
	rm -rf "$(APP_BUNDLE)"
	mkdir -p "$(APP_BUNDLE)/Contents/MacOS"
	mkdir -p "$(APP_BUNDLE)/Contents/Resources"
	GOOS=darwin GOARCH=$(ARCH) CGO_ENABLED=1 $(GO) build -ldflags "$(LDFLAGS)" -o "$(APP_BUNDLE)/Contents/MacOS/$(BINARY)" .
	./scripts/write-info-plist.sh "$(APP_BUNDLE)/Contents/Info.plist"
	codesign --force --deep --sign - "$(APP_BUNDLE)"
	@echo "Built $(APP_BUNDLE)"

## zip: package the .app for $(ARCH) into a release-ready zip using ditto
## (preserves resource forks + extended attributes that plain `zip` strips).
zip: app
	rm -f "$(APP_ZIP)"
	ditto -c -k --keepParent "$(APP_BUNDLE)" "$(APP_ZIP)"
	@echo "Packaged $(APP_ZIP)"

## install: build the .app for the host arch, replace any existing copy in
## ~/Applications, and launch it. Always uses the native arch regardless of
## any inherited ARCH value.
install: stop
	$(MAKE) app ARCH=$(HOST_ARCH)
	rm -rf "$(INSTALLED_APP)"
	mkdir -p "$(INSTALL_DIR)"
	cp -R "dist/$(HOST_ARCH)/$(APP_NAME).app" "$(INSTALL_DIR)/"
	@echo "Installed to $(INSTALLED_APP)"
	open "$(INSTALLED_APP)"

## uninstall: remove the app, config, and any LaunchAgent the user may have added
uninstall: stop
	rm -rf "$(INSTALLED_APP)"
	rm -rf "$(HOME)/Library/Application Support/mr-review-tracker"
	-launchctl unload "$(HOME)/Library/LaunchAgents/com.cwernert.mr-review-tracker.plist" 2>/dev/null
	rm -f "$(HOME)/Library/LaunchAgents/com.cwernert.mr-review-tracker.plist"
	@echo "Removed app, config, and LaunchAgent"

## clean: delete build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/
