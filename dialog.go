package main

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrDialogCancelled signals the user pressed Cancel in an osascript dialog.
var ErrDialogCancelled = errors.New("dialog cancelled")

// PromptForChannelID shows a native macOS dialog asking for a new channel UUID.
// If the user clicks Cancel it returns ErrDialogCancelled.
func PromptForChannelID(current string) (string, error) {
	script := fmt.Sprintf(`set defaultID to %q
try
  set theResult to display dialog "Paste a Storage by Zapier channel UUID:" default answer defaultID with title "MR Review Tracker" buttons {"Cancel", "Save"} default button "Save"
  return text returned of theResult
on error number -128
  return "__CANCELLED__"
end try`, current)

	out, err := osascript(script)
	if err != nil {
		return "", err
	}
	if out == "__CANCELLED__" {
		return "", ErrDialogCancelled
	}
	return strings.TrimSpace(out), nil
}

// ShowMessage displays a small informational dialog. Failures are swallowed - it's
// best-effort UX.
func ShowMessage(title, body string) {
	script := fmt.Sprintf(`display dialog %q with title %q buttons {"OK"} default button "OK"`, body, title)
	_, _ = osascript(script)
}

func osascript(body string) (string, error) {
	cmd := exec.Command("osascript", "-e", body)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("osascript: %w", err)
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}
