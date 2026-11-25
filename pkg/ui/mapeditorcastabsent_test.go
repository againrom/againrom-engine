package ui

import (
	"image"
	"strings"
	"testing"
	"time"
)

type absentPowerHost struct{ *placementLayoutHost }

func (h *absentPowerHost) NewRecordStep(r InspectionNewRecord, key string) InspectionNewStep {
	if key == "power" {
		for _, v := range r.Values {
			if v.Key == "spell" && v.Value == 2 {
				return InspectionNewStep{Absent: true}
			}
		}
	}
	return h.placementLayoutHost.NewRecordStep(r, key)
}

func castAbsentEditor(t *testing.T) (*MapEditor, *modesCalls) {
	e := newPlacementLayoutEditor(t, "en")
	h := &absentPowerHost{e.doc.Edits.(*placementLayoutHost)}
	h.self, e.doc.Edits = h, h
	return e, h.calls
}

func castAddLines(c *modesCalls) []string {
	var out []string
	for _, l := range c.lines {
		if strings.HasPrefix(l, "AddRecord "+string(KindEnchantment)) {
			out = append(out, l)
		}
	}
	return out
}

func TestPaletteSpellWithoutPowerIsPlacedByClickAndStroke(t *testing.T) {
	for _, tc := range []struct {
		name     string
		cell     int
		hasPower bool
	}{{"teleport", 1, false}, {"trap", 0, true}} {
		t.Run(tc.name+"/click", func(t *testing.T) {
			e, c := castAbsentEditor(t)
			_, l := paletteGrid(t, e, "cellspells")
			chromeClick(e, center(l.GridCells[tc.cell]))
			clickCell(e, image.Pt(12, 12))
			lines := castAddLines(c)
			if len(lines) != 1 || strings.Contains(lines[0], "{power ") != tc.hasPower {
				t.Fatalf("click wrote %q (message %q), want one record, power %v", lines, e.message, tc.hasPower)
			}
		})
		t.Run(tc.name+"/stroke", func(t *testing.T) {
			e, c := castAbsentEditor(t)
			_, l := paletteGrid(t, e, "cellspells")
			chromeClick(e, center(l.GridCells[tc.cell]))
			if err := e.Key("traparea"); err != nil {
				t.Fatal(err)
			}
			a, b := e.CellPoint(image.Pt(12, 12)), e.CellPoint(image.Pt(14, 14))
			for i, p := range []image.Point{a, b, b} {
				e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: i < 2}, time.Unix(1, 0))
			}
			lines := castAddLines(c)
			if len(lines) != 1 || strings.Contains(lines[0], "{power ") != tc.hasPower || strings.Contains(e.message, "no enchantment field") {
				t.Fatalf("stroke wrote %q (message %q), want one record, power %v", lines, e.message, tc.hasPower)
			}
		})
	}
}
