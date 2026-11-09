package widgetkit

import "testing"

func init() { UseMemoryTextClipboard() }

func TestClipboardDefaultsToMemoryDuringTests(t *testing.T) {
	c, ok := TextClipboard.(*MemoryClipboard)
	if !ok {
		t.Fatalf("test clipboard is %T", TextClipboard)
	}
	before := c.Value
	t.Cleanup(func() { c.Value = before })
	if err := c.Write("Привет"); err != nil {
		t.Fatal(err)
	}
	if got, err := c.Read(); err != nil || got != "Привет" {
		t.Fatal(got, err)
	}
}
