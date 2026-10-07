package game

import (
	"os/exec"
	"syscall"
)

func quietWitnessProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}
