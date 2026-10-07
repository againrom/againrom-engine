package ui

import "image"

// hudPanel names one switchable box.
//
// THE ORDER IS THE OWNER'S OWN LIST, left to right: inventory, spellbook,
// doll, equipment. hudPanelCount closes it, and hud.go sizes what it still
// needs sized from that constant rather than from a written-down four.
type hudPanel int

const (
	hudPanelPack hudPanel = iota // I — the carried pack's own bar
	hudPanelBook                 // S — the spellbook bar
	hudPanelDoll                 // D — the figure, in the right-hand column
	hudPanelWorn                 // E — the worn set, above the spellbook
	hudPanelCount
)

// hudToggleLabels is the letter each switch carries, in hudPanel order —
// each the KEY that flips the same switch (app.go's own bindings), kept here
// for whatever names a switch in a message or a test failure now that no
// button on screen carries it.
var hudToggleLabels = [hudPanelCount]string{"I", "S", "D", "E"}

// hudShown reports whether panel p's own box is switched on. It reads the
// stored flag inverted — see this file's header for why the flag is the
// negative one.
//
// IT ANSWERS THE SETTING AND NOTHING ELSE. Whether the box actually draws
// this frame is its own present function's question: the pack still needs a
// subject, the book still needs entries, and the doll and the worn set still
// need something selected. A caller wanting "is it on screen" must ask that
// function, never this one.
func (v *Viewer) hudShown(p hudPanel) bool { return !v.hudHidden[p] }

// toggleHudPanel flips one switch and refreshes the game-surface height that
// follows the two bottom switches (owner: "settings for what is shown").
//
// IT IS UNGATED IN BOTH DIRECTIONS, which is the difference from the window
// binding it replaced: ToggleInventory used to refuse to open over the wrong
// selection, so a press with nothing selected did nothing at all and the
// player could not tell a refused binding from an unbound key. A setting is
// set whatever is selected, and the box it enables appears the moment
// something worth drawing is.
//
// It writes no selection or order. Pack and Book can change camera ViewH; the
// clamp then moves X/Y only when a newly enlarged surface would cross the map
// edge. Away from an edge the world origin is untouched and closing a panel
// simply reveals more terrain below.
//
// IT IS UNEXPORTED, and so is every other name in this file that takes or
// returns a hudPanel. Nothing outside this package has ever named a box of
// this window — pkg/game pushes CONTENT through the setters and reads none
// of the drawing back — so an exported method taking an unexported argument
// type would state a seam that does not exist.
func (v *Viewer) toggleHudPanel(p hudPanel) {
	if p < 0 || p >= hudPanelCount {
		return
	}
	v.hudHidden[p] = !v.hudHidden[p]
	v.syncMapViewport()
	// The original plays ibook after each child-2/3 add or remove. Pack and
	// book are those two children; doll and worn are not (VIDEO-SFX-016).
	if p == hudPanelPack || p == hudPanelBook {
		v.PlayUISound(UISoundBookToggle)
	}
}

func (v *Viewer) hudToggleBar() (image.Rectangle, bool) {
	return hudToggleBarRect(image.Pt(v.frameW, v.frameH))
}
