package widgetkit

import "os"

// Clipboard is a text clipboard, the system's or an in-memory one.
type Clipboard interface {
	Read() (string, error)
	Write(string) error
}

type systemEditorClipboard struct{}

// MemoryClipboard is an in-memory clipboard: tests and headless runs.
type MemoryClipboard struct{ Value string }

func (c *MemoryClipboard) Read() (string, error) { return c.Value, nil }
func (c *MemoryClipboard) Write(s string) error  { c.Value = s; return nil }

// NewTextClipboard is the system clipboard, or an in-memory one.
func NewTextClipboard(inMemory bool) Clipboard {
	if inMemory {
		return &MemoryClipboard{}
	}
	return systemEditorClipboard{}
}

// TextClipboard is the clipboard every program reads and writes.
var TextClipboard = NewTextClipboard(os.Getenv("AGAINROM_HEADLESS_ONLY") == "1")

func ReadTextClipboard() (string, error) { return TextClipboard.Read() }

func UseMemoryTextClipboard() { TextClipboard = NewTextClipboard(true) }
