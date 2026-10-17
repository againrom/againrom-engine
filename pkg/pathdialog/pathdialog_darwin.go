//go:build darwin

package pathdialog

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
)

func show(request Request) (Result, error) {
	return showOsascript(request, runOsascript)
}

func runOsascript(args []string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	command := exec.Command("/usr/bin/osascript", args...)
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		err = fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return stdout.String(), stderr.String(), err
}
