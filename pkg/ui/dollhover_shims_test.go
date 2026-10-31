package ui

import "image"

// The doll tests ask their screens in window pixels and answer in zero-based
// slots. These adapters put those questions to the description each screen
// builds, so the tests read the one hit function and no second path.

func (v *Viewer) inventoryWornSlotAt(x, y int) (int, bool) {
	return dollSlotAt(v.wornHover(), image.Pt(x, y))
}

func (v *Viewer) dollFigureSlotAt(x, y int) (int, bool) {
	return dollSlotAt(v.dollHover(), image.Pt(x, y))
}

func (v *Viewer) dollFigureSlotAtUnsuppressed(x, y int) (int, bool) {
	return dollSlotAt(v.dollHoverUnsuppressed(), image.Pt(x, y))
}

func shopDollSlotAt(v ShopScreenView, mask *SlotMask, p image.Point) (int, bool) {
	return dollSlotAt(shopDollHover(v, mask), p)
}

func shopDollAreaAt(pane TownCharacterView, p image.Point) bool {
	return characterPaneDollHover(pane, nil, [12][]string{}).Covers(p)
}

func chargenDollHoverLines(c *Chargen, p image.Point) ([]string, int, bool) {
	lines, slot, ok := c.dollHover().TipAt(p)
	if !ok {
		return nil, 0, false
	}
	return lines, slot + 1, true
}
