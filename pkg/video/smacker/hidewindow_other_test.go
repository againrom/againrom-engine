//go:build !windows

package smacker_test

import "os/exec"

func hideProcessWindow(_ *exec.Cmd) {}
