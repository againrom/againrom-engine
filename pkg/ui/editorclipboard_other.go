//go:build !windows && !darwin

package ui

var processEditorClipboard memoryEditorClipboard

func (systemEditorClipboard) read() (string, error) { return processEditorClipboard.read() }
func (systemEditorClipboard) write(s string) error  { return processEditorClipboard.write(s) }
