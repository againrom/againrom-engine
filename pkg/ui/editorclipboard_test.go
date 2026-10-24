package ui

import "testing"

func init() { UseMemoryTextClipboard() }

func TestClipboardDefaultsToMemoryDuringTests(t *testing.T) {
	c, ok := textClipboard.(*memoryEditorClipboard)
	if !ok {
		t.Fatalf("test clipboard is %T", textClipboard)
	}
	before := c.value
	t.Cleanup(func() { c.value = before })
	if err := c.write("Привет"); err != nil {
		t.Fatal(err)
	}
	if got, err := c.read(); err != nil || got != "Привет" {
		t.Fatal(got, err)
	}
}
