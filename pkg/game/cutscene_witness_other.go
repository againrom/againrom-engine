//go:build !windows

package game

import "os/exec"

func quietWitnessProcess(_ *exec.Cmd) {}
