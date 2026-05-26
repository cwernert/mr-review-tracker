BINARY := mr-review-tracker
APP_NAME := MR Review Tracker
APP_BUNDLE := dist/$(APP_NAME).app
INSTALL_DIR := $(HOME)/Applications

GO ?= go
LDFLAGS := -s -w

.PHONY: all build test clean app install uninstall run

all: build

## build: compile a plain binary in the repo root
build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## test: run unit tests
test:
	$(GO) test ./...

## run: build and execute the binary in-place (foreground, ^C to quit)
run: build
	./$(BINARY)

## app: assemble a macOS .app bundle under ./dist with LSUIElement=1
app: build
	rm -rf "$(APP_BUNDLE)"
	mkdir -p "$(APP_BUNDLE)/Contents/MacOS"
	mkdir -p "$(APP_BUNDLE)/Contents/Resources"
	cp $(BINARY) "$(APP_BUNDLE)/Contents/MacOS/$(BINARY)"
	./scripts/write-info-plist.sh "$(APP_BUNDLE)/Contents/Info.plist"
	@echo "Built $(APP_BUNDLE)"

## install: build the .app and copy it into ~/Applications
install: app
	rm -rf "$(INSTALL_DIR)/$(APP_NAME).app"
	mkdir -p "$(INSTALL_DIR)"
	cp -R "$(APP_BUNDLE)" "$(INSTALL_DIR)/"
	@echo "Installed to $(INSTALL_DIR)/$(APP_NAME).app"

## uninstall: remove the app and config
uninstall:
	rm -rf "$(INSTALL_DIR)/$(APP_NAME).app"
	rm -rf "$(HOME)/Library/Application Support/mr-review-tracker"
	rm -f "$(HOME)/Library/LaunchAgents/com.cwernert.mr-review-tracker.plist"
	-launchctl unload "$(HOME)/Library/LaunchAgents/com.cwernert.mr-review-tracker.plist" 2>/dev/null
	@echo "Removed app, config, and LaunchAgent"

## clean: delete build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/
