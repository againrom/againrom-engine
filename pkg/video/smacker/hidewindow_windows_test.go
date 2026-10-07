package smacker_test

import (
	"os/exec"
	"syscall"
)

// hideProcessWindow keeps the oracle helper processes (ffmpeg, the
// windows/386 cutscenehelper) from flashing a console window during the
// gated release test, mirroring pkg/game's own quietWitnessProcess.
func hideProcessWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}
