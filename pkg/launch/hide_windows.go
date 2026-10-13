package launch

import (
	"os/exec"
	"syscall"
)

// createNoWindow is the process creation flag that gives a console program no
// console window. It does not touch the program's own windows.
const createNoWindow = 0x08000000

// HideConsole keeps a child console program from opening a console window.
func HideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
