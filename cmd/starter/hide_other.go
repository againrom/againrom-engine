//go:build !windows

package main

import "os/exec"

func hideConsole(*exec.Cmd) {}
