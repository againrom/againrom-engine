package game

import (
	"image"
	"testing"

	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// TestEveryTipRowCloseTakesItsWholeButton: for every popup row of both
// games' tip tables (rooms, generator pages, mission), at the row's
// rectangle and at the smallest fitted height, with the row's own text
// inset, every point of the painted Close button is Close on press and on
// release.
func TestEveryTipRowCloseTakesItsWholeButton(t *testing.T) {
	for name, table := range tipTables {
		if table == nil || len(table.Rows) == 0 {
			t.Fatalf("tip table %s has no rows", name)
		}
		for i := range table.Rows {
			row := &table.Rows[i]
			rect := row.Rectangle()
			if rect.Empty() {
				continue
			}
			for _, r := range []image.Rectangle{rect, image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+104)} {
				v := ui.TipPanelView{Rect: r, Text: "tip", Art: &ui.TipPanelArt{}, Font: &text.Font{}, TextInset: table.Panel(row).Inset}
				cr := ui.TipPanelCloseRect(r)
				for y := cr.Min.Y; y < cr.Max.Y; y++ {
					for x := cr.Min.X; x < cr.Max.X; x++ {
						for _, down := range []bool{true, false} {
							if k, ok := ui.TipPanelEventAt(v, image.Pt(x, y), down); k != ui.TipControlClose || !ok {
								t.Fatalf("%s row %q rect %v point (%d,%d) down %v: %v,%v want Close", name, row.Name, r, x, y, down, k, ok)
							}
						}
					}
				}
			}
		}
	}
}
