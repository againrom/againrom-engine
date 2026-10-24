package ui

import (
	"os/exec"
	"strconv"
	"strings"
)

func (systemEditorClipboard) read() (string, error) {
	b, err := exec.Command("/usr/bin/osascript", "-e", "get the clipboard as text").Output()
	return strings.TrimSuffix(string(b), "\n"), err
}

func (systemEditorClipboard) write(s string) error {
	return exec.Command("/usr/bin/osascript", "-e", "set the clipboard to "+strconv.Quote(s)).Run()
}
