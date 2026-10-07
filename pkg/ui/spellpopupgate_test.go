package ui

// The spellbook popup's availability gate (TEXT-HOVERTEXT-052's required
// availability bit): a cell the selected units cannot cast states nothing,
// on both call sites that show this popup — the mission book
// (tooltip_runtime.go's tooltipTarget) and the shop's Book toggle
// (shopscreen.go's ShopHoverLines, DIV-118) — while the cell stays hoverable
// and clickable in both, since spellbookEntryAt and shopSpellEntryAt are also
// how a player selects a cell at all.

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func TestMissionSpellPopupGatesOnAvailability(t *testing.T) {
	a := spellbookTestApp(t)
	v := a.flow.viewer
	v.SetFont(messageFont())
	v.SetEntities(sbEntities())
	v.sel = selection{sbUnitID}
	book := []SpellEntry{
		{ID: 1, Name: "Fire Arrow", Info: []string{"Fire Arrow", "Mana cost: 3"}},
		{ID: 6, Name: "Heal", Unavailable: true, Info: []string{"Heal", "Mana cost: 10"}},
	}
	v.SetSpellbook(sbUnitID, book)

	ax, ay := sbEntryPoint(t, v, book, 0)
	v.cursorX, v.cursorY, v.hasCursor = ax, ay, true
	if target := v.tooltipTarget(); target.kind != tooltipSpell || len(target.lines) == 0 {
		t.Fatalf("available cell answered %+v, want the spell's own popup", target)
	}
	if idx, ok := v.spellbookEntryAt(ax, ay); !ok || idx != 0 {
		t.Fatalf("available cell hit-test = %d,%v, want 0,true", idx, ok)
	}

	ux, uy := sbEntryPoint(t, v, book, 1)
	v.cursorX, v.cursorY, v.hasCursor = ux, uy, true
	if target := v.tooltipTarget(); target.kind == tooltipSpell {
		t.Fatalf("unavailable cell answered a popup: %+v", target)
	}
	if idx, ok := v.spellbookEntryAt(ux, uy); !ok || idx != 1 {
		t.Fatalf("unavailable cell hit-test = %d,%v, want 1,true — the cell must stay reachable so a player can still see which catalog entries exist", idx, ok)
	}
}

func TestShopBookPopupGatesOnAvailability(t *testing.T) {
	icon := image.NewRGBA(image.Rect(0, 0, 36, 36))
	draw.Draw(icon, icon.Bounds(), &image.Uniform{C: color.RGBA{R: 0xa4, B: 0xd8, A: 0xff}}, image.Point{}, draw.Src)
	view := ShopScreenView{Book: true, Spells: []SpellEntry{
		{ID: 7, Name: "Fire Arrow", Icon: icon, Info: []string{"Fire Arrow", "Mana cost: 4"}},
		{ID: 9, Name: "Heal", Icon: icon, Unavailable: true, Info: []string{"Heal", "Mana cost: 10"}},
	}}
	cells := bookCellRects(shopTableRegion, shopSpellColumns())
	available := cells[0].Min.Add(image.Pt(4, 4))
	unavailable := cells[1].Min.Add(image.Pt(4, 4))

	if lines, ok := ShopHoverLines(view, available); !ok || len(lines) == 0 {
		t.Fatalf("available spell hover = %v,%v, want the entry's own popup", lines, ok)
	}
	if c, ok := shopScreenControlAt(view, available); !ok || c.Kind != ShopControlSpell || c.Index != 0 {
		t.Fatalf("available spell cell = %#v,%v, want inspection-only spell 0", c, ok)
	}

	if lines, ok := ShopHoverLines(view, unavailable); ok {
		t.Fatalf("unavailable spell hover = %v,%v, want none", lines, ok)
	}
	if c, ok := shopScreenControlAt(view, unavailable); !ok || c.Kind != ShopControlSpell || c.Index != 1 {
		t.Fatalf("unavailable spell cell = %#v,%v, want inspection-only spell 1 — the cell must stay reachable", c, ok)
	}
}
