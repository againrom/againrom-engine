package ui

import "os"

type editorClipboard interface {
	read() (string, error)
	write(string) error
}

type systemEditorClipboard struct{}

type memoryEditorClipboard struct{ value string }

func (c *memoryEditorClipboard) read() (string, error) { return c.value, nil }
func (c *memoryEditorClipboard) write(s string) error  { c.value = s; return nil }

func newTextClipboard(inMemory bool) editorClipboard {
	if inMemory {
		return &memoryEditorClipboard{}
	}
	return systemEditorClipboard{}
}

var textClipboard = newTextClipboard(os.Getenv("AGAINROM_HEADLESS_ONLY") == "1")

func ReadTextClipboard() (string, error) { return textClipboard.read() }

func UseMemoryTextClipboard() { textClipboard = newTextClipboard(true) }
