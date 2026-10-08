package game

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// castOrderFront opens the fixture mission with a mage (1) who knows Heal (6)
// and an area row (3), and a wounded ally (2) two cells east.
func castOrderFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	if err := f.App("cast order flag").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	spells := make(dbCollection, 7)
	for i := 1; i < len(spells); i++ {
		spells[i] = dbEntry{name: "synthetic spell", params: make([]int32, 19)}
	}
	heal, area := spells[6].params, spells[3].params
	heal[1], heal[2], heal[4], heal[6], heal[8], heal[16], heal[17] = 5, 1, 1, 8, 1, 10, 20
	area[1], area[2], area[6], area[8], area[9], area[11], area[16], area[17] = 5, 1, 8, 4, 2, 15, 4, 4
	f.Table.Spells = spells
	rules := mapload.SpellRules(f.Table)
	actors := []sim.Entity{
		{ID: 1, X: 10, Y: 10, HP: 100, MaxHP: 100, Mind: 60, Mana: 900, MaxMana: 900,
			KnownSpells: 1<<6 | 1<<3, ScanRange: 19, AttackCharge: 4, AttackRelax: 2},
		{ID: 2, X: 12, Y: 10, HP: 40, MaxHP: 100},
	}
	for i := range actors {
		actors[i].Owner, actors[i].TokenSize, actors[i].TypeID, actors[i].Speed = 1, 1, 1, 10
		actors[i].HealthRegenPeriod, actors[i].ManaRegenPeriod, actors[i].Capacity = 1, 1, data.UnitCapacity()
	}
	w, err := sim.NewSpelledWorld(1, f.live.world.Bounds(), sim.ModeCanonical, nil, actors, nil, rules)
	if err != nil {
		t.Fatal(err)
	}
	f.live.world, f.live.mission.state.World = w, w
	return f
}

func castOrderHP(w *sim.World, id sim.EntityID) int32 {
	for _, e := range w.Entities() {
		if e.ID == id {
			return e.HP
		}
	}
	return 0
}

func castOrderRecord(t *testing.T, doc sav.DocumentData, a *currentActionData, id sim.EntityID) sav.DocumentRecordData {
	t.Helper()
	for _, b := range a.Bindings {
		if !b.Structure && !b.Missing && b.ID == id && b.Object != 0 && int(b.Object) <= len(doc.Objects) {
			return doc.Objects[b.Object-1]
		}
	}
	t.Fatalf("actor %d has no SAV object", id)
	return sav.DocumentRecordData{}
}

// A SAV written while a book cast charges restores that cast in the original:
// progress 2 selects action 0xd or 0xe from order byte +0x5c (AI-RETREAT-272),
// 1 at a unit and 0 at a cell (DIV-1491). Cold LOAD then lands it unchanged.
func TestCurrentCastOrderCarriesItsUnitCastFlag(t *testing.T) {
	for _, tc := range []struct {
		name          string
		cast          sim.Command
		action        uint32
		pending, flag byte
		landed        func(hpBefore int32, after *sim.World) bool
	}{
		// Heal's smallest roll is 10; regeneration never adds that in one tick.
		{"unit", sim.Cast(1, 2, 6), 0xd, 8, 1, func(hpBefore int32, after *sim.World) bool {
			return hpBefore+10 <= castOrderHP(after, 2)
		}},
		{"cell", sim.CastAt(1, 3, sim.CellPoint{X: 14, Y: 10}), 0xe, 9, 0, func(_ int32, after *sim.World) bool {
			return len(after.CellEffects()) != 0
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := castOrderFront(t)
			w := f.live.world
			sim.Step(w, []sim.Command{tc.cast})
			for tick := 0; tick < 16 && (len(w.Actions().Books) != 1 || w.Actions().Books[0].Phase != 1); tick++ {
				sim.Step(w, nil)
			}
			if books := w.Actions().Books; len(books) != 1 || books[0].Phase != 1 || books[0].Remaining == 0 {
				t.Fatalf("player cast did not reach its charge: %+v", books)
			}
			raw, doc, a := saveCurrentEffect(t, f)
			r := castOrderRecord(t, doc, a, 1)
			spells, _ := savedObjectRefs(&r, "Spells")
			spell := uint16(6)
			if tc.flag == 0 {
				spell = 3
			}
			if spell == 0 || int(spell) > len(spells) || spells[spell-1] == 0 {
				t.Fatal("current cast lacks exact sparse-book Spell node")
			}
			selected := savedRecordValueForTest(t, doc.Objects[spells[spell-1]-1], "This")
			if got := savedRecordValueForTest(t, r, "U44"); got == 0 || got != selected {
				t.Fatalf("admitted current book selection U44=%#x, current Spell%d key%#x", got, spell, selected)
			}
			order := savedRecordRawForTest(t, r, "U158")
			target := uint32(0)
			if tc.flag == 1 {
				target = savedRecordValueForTest(t, castOrderRecord(t, doc, a, 2), "Identity")
			}
			if got := binary.LittleEndian.Uint32(savedRecordRawForTest(t, r, "U54")); got != tc.action || order[8] != tc.pending || order[9] != 2 ||
				binary.LittleEndian.Uint32(order[0x28:]) != target {
				t.Fatalf("cast wire action %#x order %#x/%d target %#x", got, order[8], order[9], binary.LittleEndian.Uint32(order[0x28:]))
			}
			if order[0x5c] != tc.flag {
				t.Fatalf("order +0x5c = %d, want %d: the original restores the wrong cast action", order[0x5c], tc.flag)
			}
			cold := openCurrentEffectSave(t, f, raw)
			assertCurrentWorldEqual(t, w, cold.live.world, "mid-cast cold LOAD")
			_, second, b := saveCurrentEffect(t, cold)
			if again := savedRecordRawForTest(t, castOrderRecord(t, second, b, 1), "U158"); !bytes.Equal(again, order) {
				t.Fatal("second SAVE changed the cast order block")
			}
			for tick := 0; ; tick++ {
				if tick == 64 {
					t.Fatal("cast did not land within 64 ticks")
				}
				before, coldBefore := castOrderHP(w, 2), castOrderHP(cold.live.world, 2)
				sim.Step(w, nil)
				sim.Step(cold.live.world, nil)
				if w.Hash() != cold.live.world.Hash() {
					t.Fatalf("cold continuation diverged at tick %d", tick)
				}
				if landed := tc.landed(before, w); landed || tc.landed(coldBefore, cold.live.world) {
					if !landed || !tc.landed(coldBefore, cold.live.world) {
						t.Fatalf("cast landed out of step at tick %d", tick)
					}
					break
				}
			}
		})
	}
}

func TestCompletedBookAdmissionWritesCurrentSelectionOnFirstSave(t *testing.T) {
	for _, command := range []sim.Command{sim.Cast(1, 2, 6), sim.CastAt(1, 3, sim.CellPoint{X: 14, Y: 10})} {
		f := castOrderFront(t)
		initial, _, _ := saveCurrentEffect(t, f)
		f = openCurrentEffectSave(t, f, initial)
		w := f.live.world
		sim.Step(w, []sim.Command{command})
		if len(w.Actions().Books) != 1 {
			t.Fatal("cast was not admitted")
		}
		for tick := 0; len(w.Actions().Books) != 0 || w.Entities()[0].PendingOrder.Kind != sim.PendingNone; tick++ {
			if tick == 96 {
				t.Fatal("cast did not complete")
			}
			sim.Step(w, nil)
		}
		raw, doc, actions := saveCurrentEffect(t, f)
		r := castOrderRecord(t, doc, actions, 1)
		spells, _ := savedObjectRefs(&r, "Spells")
		slot := uint16(6)
		if command.Kind == sim.KindCastAt {
			slot = 3
		}
		want := savedRecordValueForTest(t, doc.Objects[spells[slot-1]-1], "This")
		if got := savedRecordValueForTest(t, r, "U44"); got == 0 || got != want {
			t.Fatalf("first SAVE after completed cast U44=%#x, admitted slot%d key%#x", got, slot, want)
		}
		if command.Kind == sim.KindCastAt {
			continue
		}
		cold := openCurrentEffectSave(t, f, raw)
		assertCurrentWorldEqual(t, w, cold.live.world, "completed book admission cold LOAD")
		loss, err := sav.CloneDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		actor := castOrderRecord(t, loss, actions, 1)
		for i := range loss.Objects {
			if loss.Objects[i].Class == actor.Class && savedRecordValueForTest(t, loss.Objects[i], "Identity") == savedRecordValueForTest(t, actor, "Identity") {
				savedObjectSetValue(&loss.Objects[i], "U44", 0)
			}
		}
		lostRaw, err := sav.EncodeDocumentData(loss)
		if err != nil {
			t.Fatal(err)
		}
		lost := openCurrentEffectSave(t, f, lostRaw)
		if lost.live.world.Entities()[0].AdmittedBookSpell != 0 || lost.live.world.Hash() == w.Hash() {
			t.Fatal("ordinary selection removal failed its cold loss control")
		}
		_, next, nextActions := saveCurrentEffect(t, cold)
		if got := savedRecordValueForTest(t, castOrderRecord(t, next, nextActions, 1), "U44"); got != want {
			t.Fatalf("cold SAVE lost completed book admission: %#x != %#x", got, want)
		}
		for tick := 0; tick < 3; tick++ {
			sim.Step(w, nil)
			sim.Step(cold.live.world, nil)
			if w.Hash() != cold.live.world.Hash() {
				t.Fatal("completed selection changed post-LOAD continuation")
			}
		}
	}
}
