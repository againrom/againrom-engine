//go:build !windows

package launch

import "os/exec"

// HideConsole does nothing where a child program opens no console window.
func HideConsole(*exec.Cmd) {}
