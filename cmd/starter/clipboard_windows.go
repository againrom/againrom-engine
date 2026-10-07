package main

import "os/exec"

// clipboardText reads the clipboard through PowerShell. An error is an empty
// paste.
func clipboardText() string {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", "Get-Clipboard -Raw")
	hideConsole(cmd)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}
