package ui

import (
	"image"
	"testing"
)

func TestMissionMenuPanelGrowsWithItsRowsAndTownStaysDecoded(t *testing.T) {
	first, _ := onMapWithKeys(t, "")
	first.flow.menuFont = gameMenuTestFont()
	first.flow.openGameMenu(ScreenMap)
	for name, a := range map[string]*App{"first": first, "native": nativeMenuApp(t, image.Pt(1280, 800))} {
		rows := a.flow.menuRows()
		if !gameMenuHasRow(rows, gameMenuKeys) {
			t.Fatalf("%s: the mission menu has no Keys row", name)
		}
		for n := len(rows); n >= 5; n-- {
			p := gameMenuPanelRectFor(gameMenuMission, rows[:n], a.flow.menuGeometryProfile())
			top := gameMenuRowRect(p, 0).Min.Y - p.Min.Y
			bottom := p.Max.Y - gameMenuRowRect(p, n-1).Max.Y
			if top != bottom {
				t.Fatalf("%s, %d rows: margin %d over the first row, %d under the last", name, n, top, bottom)
			}
		}
		pl := a.flow.menuGeometryProfile().placement()
		town := make([]gameMenuRow, 6)
		if got, want := gameMenuPanelRectFor(gameMenuTown, town, a.flow.menuGeometryProfile()), pl.panel(gameMenuTown); got != want {
			t.Fatalf("%s: the town panel %v is not the decoded %v", name, got, want)
		}
	}
}
