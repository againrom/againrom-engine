package text

import (
	"testing"

	"againrom/pkg/locale"
)

// SelectorConverting is the selector of the one locale whose font remaps a
// byte, and Convert moves bytes exactly under the selectors the locale table
// marks.
func TestSelectorConvertingIsTheRemappingLocale(t *testing.T) {
	remapping := 0
	for _, l := range locale.All() {
		if l.FontRemap {
			remapping++
			if l.Selector != SelectorConverting {
				t.Fatalf("locale %q remaps under selector %d, SelectorConverting is %d", l.Code, l.Selector, SelectorConverting)
			}
		}
		if moved := Convert(0x80, l.Selector) != 0x80; moved != l.FontRemap {
			t.Fatalf("locale %q: Convert moved=%t, record remap=%t", l.Code, moved, l.FontRemap)
		}
	}
	if remapping != 1 {
		t.Fatalf("%d locales remap", remapping)
	}
}
