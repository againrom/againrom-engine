package game

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func releaseWholeModDamage(t *testing.T, power int32) {
	t.Helper()
	columns := func(name func(int) string) string {
		return fmt.Sprintf("[[spell]]\ntarget = %q\npower = [%d]\ndamage = { min = 300, max = 600 }\n", name(fireArrowID), power)
	}
	f := targetsModFront(t, columns)
	casterID, allyID := targetsMission(t, f)
	mw := f.live
	base, top := int64(300), int64(600)
	if power == 100 {
		base, top = 1300, 2600
	}
	checkBook := func(m *mapWorld) {
		t.Helper()
		line := m.view.Words().Hover[spellLabelDamage] + ": " + spellPair(base, top, "%d", "-")
		if book := popupOf(m, casterID); !slices.Contains(book[fireArrowID], line) {
			t.Fatalf("whole column popup lacks %q: %v", line, book[fireArrowID])
		}
	}
	checkBook(mw)
	caster, _ := mw.entity(casterID)
	enemy := hostileNear(t, mw, []sim.EntityID{casterID, allyID}, caster.X+5, caster.Y)
	mw.castAt(uint32(casterID), uint32(enemy), fireArrowID)
	for tick := 0; mw.world.PendingSpellDeliveries() == 0; tick++ {
		if tick >= 300 {
			t.Fatal("whole mod cast never prepared a queued payload")
		}
		mw.tick()
	}
	hash := mw.world.Hash()
	raw := exportSave(t, f)
	if mw.world.Hash() != hash {
		t.Fatal("SAVE changed the live whole-damage World")
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, o := range doc.Objects {
		if o.Class == "Effect_DirectDamage" {
			for _, field := range o.Raw {
				p := field.Bytes
				if field.Name == "EDD48" && len(p) == 24 && p[19] == 255 && p[20] == 255 {
					found++
				}
			}
		}
	}
	if found != 1 {
		t.Fatalf("SAV has %d saturated 255/255 payloads, want one", found)
	}
	g := targetsModFront(t, columns)
	open, town, err := g.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("cold LOAD: town %v, %v", town, err)
	}
	if err := g.App("whole damage cold load").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	checkBook(g.live)
	rows := g.live.world.NativeSpellDeliverySaveStates()
	if len(rows) != 1 || rows[0].Area.Payload.DirectDamage[19] != 255 || rows[0].Area.Payload.DirectDamage[20] != 255 {
		t.Fatalf("cold queued payload did not retain saturated bytes: %+v", rows)
	}
	t.Logf("power %d: popup %d..%d, SAV and cold queued payload 255/255, live World unchanged by SAVE", power, base, top)
}
