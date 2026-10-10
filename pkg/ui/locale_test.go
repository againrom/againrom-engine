package ui

import (
	"testing"

	"againrom/pkg/locale"
)

// The accelerator fold lowers CP866 uppercase only under a selector whose
// locale writes code page 866.
func TestMenuFoldFollowsTheLocaleCodePage(t *testing.T) {
	for _, l := range locale.All() {
		folded := gameMenuLower(0x80, l.Selector) != 0x80
		if folded != (l.CodePage == 866) {
			t.Errorf("locale %q: fold=%t, code page %d", l.Code, folded, l.CodePage)
		}
		if gameMenuLower('A', l.Selector) != 'a' {
			t.Errorf("locale %q: ASCII not lowered", l.Code)
		}
	}
}
