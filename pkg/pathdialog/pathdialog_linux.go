package pathdialog

import "os/exec"

func show(request Request) (Result, error) {
	return showLinuxDialog(request, exec.LookPath, func(executable string, args []string) (string, int, error) {
		command := exec.Command(executable, args...)
		stdout, err := command.Output()
		code := -1
		if command.ProcessState != nil {
			code = command.ProcessState.ExitCode()
		}
		return string(stdout), code, err
	})
}
