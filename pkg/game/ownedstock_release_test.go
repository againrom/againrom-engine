package game

import (
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestReleaseOwnedStockDispatchesOnBothShippedMaps(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	for _, sample := range []struct {
		mapKey string
		owner  uint16
		codes  []uint16
	}{
		{"scenario/110.alm", 46, []uint16{0x5648, 0x212f, 0x2832}},
		{"scenario/111.alm", 91, []uint16{0x7173, 0x0e08}},
	} {
		doc, err := x.Open(sample.mapKey)
		if err != nil {
			t.Fatal(err)
		}
		m, err := doc.Edits.(*mapInspectionEdits).model.Map()
		if err != nil {
			t.Fatal(err)
		}
		w, err := mapload.FromALMWith(m, x.table, mapload.DifficultyNormal)
		if err != nil {
			t.Fatal(err)
		}
		index := slices.IndexFunc(m.Units, func(u alm.Unit) bool { return u.UnitID == sample.owner })
		if index < 0 {
			t.Fatal("stock owner missing", sample)
		}
		pack, _ := w.Carried(sim.EntityID(index))
		worn, _ := w.Equipped(sim.EntityID(index))
		for _, code := range sample.codes {
			if slices.Contains(pack, code) {
				t.Fatalf("%s owner %d still carries %#04x", sample.mapKey, sample.owner, code)
			}
			if code != 0x0e08 && !slices.Contains(worn[:], code) {
				t.Fatalf("%s owner %d did not wear %#04x", sample.mapKey, sample.owner, code)
			}
		}
	}
}
