package ui

import "testing"

func TestSpaceReturnsAWaitingHeroBeforeTogglingPanels(t *testing.T) {
	a, v := panelCasterApp(t)
	waiting, asked := true, 0
	v.SetHeroReturnSink(func() bool { asked++; return waiting })
	pressPanelKey(t, a, "space")
	if pack, book := panelsOpen(v); asked != 1 || pack || book {
		t.Fatalf("Space with a hero waiting asked %d times and left pack=%v book=%v, want one ask and no panel", asked, pack, book)
	}
	waiting = false
	pressPanelKey(t, a, "space")
	if pack, book := panelsOpen(v); asked != 2 || !pack || !book {
		t.Fatalf("Space with no hero waiting asked %d times and left pack=%v book=%v, want both panels open", asked, pack, book)
	}
}
