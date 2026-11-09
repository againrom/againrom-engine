package ui

import (
	"image"
	"testing"
)

// A scroll armed from the pack releases on a click at the caster's own figure
// for a fighter (no mana, every book cell unavailable) as for a mage, with the
// book bar shut or open.
func TestAFighterAimsAnArmedScrollAtHimselfByClick(t *testing.T) {
	for _, tc := range []struct {
		name string
		ent  MapEntity
	}{
		{"mage", MapEntity{ID: panelCasterID, Owner: 1, Cell: image.Pt(3, 3), Life: LifeAlive, CastCapable: true, MaxMana: 50, SpellStateKnown: true, KnownSpells: 1 << 3}},
		{"fighter", MapEntity{ID: panelCasterID, Owner: 1, Cell: image.Pt(3, 3), Life: LifeAlive, SpellStateKnown: true}},
	} {
		for _, open := range []bool{false, true} {
			t.Run(tc.name, func(t *testing.T) {
				_, v := panelCasterApp(t)
				v.SetEntities([]MapEntity{tc.ent, {ID: panelVictimID, Owner: 2, Cell: image.Pt(7, 3), Life: LifeAlive}})
				var got []uint32
				v.SetItemCastSink(func(owner uint32, index int, key string, victim uint32, x, y int, cell bool) {
					got = append(got, victim)
				})
				if open && !v.hudShown(hudPanelBook) {
					v.toggleHudPanel(hudPanelBook)
				}
				entries := make([]SpellEntry, 24)
				for i := range entries {
					entries[i] = SpellEntry{ID: uint32(i + 1), Unavailable: tc.name == "fighter"}
				}
				v.SetSpellbookCatalog(panelCasterID, entries)
				if !v.ArmItemCast(panelCasterID, 0, 3, "k") {
					t.Fatal("not armed")
				}
				x, y := cellPoint(v, 3, 3)
				ords, ok := tapAt(v, x, y)
				t.Logf("book open %v: orders %+v ok %v sink %v", open, ords, ok, got)
				if len(got) != 1 || got[0] != panelCasterID {
					t.Fatalf("self aim: sink %v", got)
				}
			})
		}
	}
}
