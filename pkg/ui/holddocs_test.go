package ui

import "testing"

// TestDocumentsPanelHoldsTheMapLikeTheInGameMenu drives the same haltRun span
// under the in-game menu and under the documents panel and requires the two
// to leave the same trace on the seam: zero world ticks and zero animation
// ticks during the hold, a declared stop, and the same bounded catch-up (not
// the whole span) on the first frame back.
func TestDocumentsPanelHoldsTheMapLikeTheInGameMenu(t *testing.T) {
	cases := []struct {
		name  string
		open  func(t *testing.T, h *haltFix)
		close func(f *flow)
	}{
		{
			name: "in-game menu",
			open: func(t *testing.T, h *haltFix) {
				if h.a.flow.escape() {
					t.Fatal("Escape on the map screen exited the program")
				}
			},
			close: func(f *flow) { f.closeGameMenu() },
		},
		{
			name: "documents panel",
			open: func(t *testing.T, h *haltFix) {
				h.a.flow.docSrc = &fakeDocumentSource{pages: []DocumentPage{{Text: "one"}}}
				h.a.flow.docArt = documentsTestArt()
				if !h.a.flow.openDocuments(ScreenMap) {
					t.Fatal("openDocuments refused a collection of one")
				}
			},
			close: func(f *flow) { f.closeDocuments() },
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newHaltFix(t, haltOpts{})
			c.open(t, h)
			if h.a.Screen() == ScreenMap {
				t.Fatalf("setup: still on ScreenMap after opening %s", c.name)
			}
			if !h.v.popupOpen() {
				t.Fatalf("%s: viewer.popupOpen() = false while it stands over the map", c.name)
			}
			animBefore := h.v.AnimationCounter()

			// THE HELD SPAN. haltRun frames, exactly cadence_test.go's own
			// schedule (3.1s, 50 world ticks' worth at the map's own period)
			// — nothing of it may reach the world or the ambient clock.
			if got := h.run(haltRun, haltNeutral()); got != 0 {
				t.Errorf("%s: %d world ticks ran during the %d-frame held span, want 0",
					c.name, got, haltRun)
			}
			if got := h.v.AnimationCounter() - animBefore; got != 0 {
				t.Errorf("%s: the animation counter rose %d during the held span, want 0", c.name, got)
			}
			if cc, ok := h.w.lastCadence(); !ok || !cc.stopped {
				t.Errorf("%s: last cadence told = %+v ok=%v, want a call with stopped=true",
					c.name, cc, ok)
			}

			// DISMISSAL. One frame back pays at most the pacer's own bounded
			// catch-up, never the whole 3.1s span the hold covered — this
			// fixture's clock is unbounded, so what is asserted is that the
			// two screens pay the SAME amount, not a specific figure.
			worldBefore := h.w.world
			c.close(h.a.flow)
			h.frame(haltNeutral())
			backWorld := h.w.world - worldBefore
			if backWorld == 0 || backWorld > haltFirstResumed {
				t.Errorf("%s: the first frame back ran %d world ticks, want the bounded resume "+
					"figure %d — not zero and not the whole held span", c.name, backWorld, haltFirstResumed)
			}
			if got := int(h.v.AnimationCounter() - animBefore); got != backWorld {
				// The fixture's own clock and the world clock share one period
				// and one schedule here, so the two are expected to agree
				// exactly on this driven span; a divergence means the two
				// hold statements are no longer paired the way holdMapUnder's
				// own three statements pair them.
				t.Errorf("%s: the animation counter rose %d on the first frame back, want %d "+
					"to match the world's own resume", c.name, got, backWorld)
			}
		})
	}
}
