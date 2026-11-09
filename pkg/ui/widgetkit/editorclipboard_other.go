//go:build !windows && !darwin

package widgetkit

var processEditorClipboard MemoryClipboard

func (systemEditorClipboard) Read() (string, error) { return processEditorClipboard.Read() }
func (systemEditorClipboard) Write(s string) error  { return processEditorClipboard.Write(s) }
