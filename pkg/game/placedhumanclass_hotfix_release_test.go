package game

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func placedClassMission(t *testing.T, f *FrontEnd, mission int) *mapWorld {
	t.Helper()
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	a := f.App("placed human class")
	t.Cleanup(a.StopAudio)
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpener(mission)); err != nil {
		t.Fatal(err)
	}
	return f.live
}

func placedClassID(t *testing.T, mw *mapWorld, unit uint16) sim.EntityID {
	t.Helper()
	for _, e := range mw.world.Entities() {
		if e.MapUnitID == unit {
			return e.ID
		}
	}
	t.Fatalf("map unit %d is not in the mission", unit)
	return 0
}

func placedClassSwingSlots(t *testing.T, f *FrontEnd, mw *mapWorld, id sim.EntityID) []int {
	t.Helper()
	hv, _ := mw.entity(id)
	var victim sim.EntityID
	found := false
	for _, e := range mw.world.Entities() {
		if e.ID != id && e.Owner != hv.Owner && e.Alive() && !e.OffMap {
			victim, found = e.ID, true
			break
		}
	}
	if !found {
		t.Fatal("the mission holds no other owner's person to strike")
	}
	if err := mw.world.HeadlessPlace(victim, hv.X+1, hv.Y); err != nil {
		t.Fatal(err)
	}
	heard := observeReleaseAudio(t, f)
	mw.strike(uint32(id), uint32(victim))
	for tick := 0; tick < 24; tick++ {
		mw.tick()
	}
	var slots []int
	for i, request := range heard.requests {
		if strings.HasPrefix(request.Selector, "registry:") && releaseAudioFromActor(t, heard, i, id) {
			slot, err := strconv.Atoi(strings.TrimPrefix(request.Selector, "registry:"))
			if err != nil {
				t.Fatal(err)
			}
			slots = append(slots, slot)
		}
	}
	return slots
}

func placedClassWord(t *testing.T, raw []byte, actor sim.EntityID) uint32 {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatal("written SAV lacks readable current actions", err)
	}
	for _, b := range a.Bindings {
		if b.ID != actor || b.Structure || b.Missing || b.Object == 0 || int(b.Object) > len(doc.Objects) {
			continue
		}
		word, err := savedStructureValue(&doc.Objects[b.Object-1], "T0E")
		if err != nil {
			t.Fatal(err)
		}
		return word
	}
	t.Fatalf("written SAV binds no actor %d", actor)
	return 0
}

type placedClassCase struct {
	unit  uint16
	class int32
}

func requirePlacedClasses(t *testing.T, mw *mapWorld, cases []placedClassCase, when string) {
	t.Helper()
	for _, c := range cases {
		id := placedClassID(t, mw, c.unit)
		e, _ := mw.entity(id)
		if e.Class != c.class || e.TypeID != c.class {
			t.Errorf("%s: unit %d holds class %d (type %d), want its row type %d", when, c.unit, e.Class, e.TypeID, c.class)
		}
		if got := mw.spellClientClass(id, e.Class); got != c.class {
			t.Errorf("%s: unit %d swing class %d, want %d", when, c.unit, got, c.class)
		}
	}
}

func placedClassDropWeapon(t *testing.T, mw *mapWorld, id sim.EntityID) {
	t.Helper()
	slots, ok := mw.world.Equipped(id)
	if !ok || slots[0] == 0 {
		t.Fatalf("actor %d holds no weapon to drop", id)
	}
	sim.Run(mw.world, [][]sim.Command{{sim.DropWorn(id, 1, sim.CellPoint{X: 1, Y: 1})}}, 3)
	if after, _ := mw.world.Equipped(id); after[0] != 0 {
		t.Fatalf("actor %d still holds %#x after DropWorn", id, after[0])
	}
}

func TestReleaseMission141PlacedMageHoldsItsRowClass(t *testing.T) {
	f := releaseFront(t)
	mw := placedClassMission(t, f, 141)
	cases := []placedClassCase{{unit: 5, class: 24}}
	requirePlacedClasses(t, mw, cases, "tick zero")
	if f.SoundClasses[24].Slots[0] != 510 {
		t.Fatalf("class 24 Sound[0] %v, want 510", f.SoundClasses[24].Slots)
	}
	id := placedClassID(t, mw, 5)

	_, raw := writeOrdinarySAV(t, f, "placed-mage.sav")
	if word := placedClassWord(t, raw, id); word != 24 {
		t.Errorf("written type word %d, want 24", word)
	}

	if got := placedClassSwingSlots(t, f, mw, id); len(got) == 0 {
		t.Error("the armed mage's attack requested no sound")
	}

	placedClassDropWeapon(t, mw, id)
	requirePlacedClasses(t, mw, cases, "after the weapon is dropped")

	path, raw := writeOrdinarySAV(t, f, "placed-mage-unarmed.sav")
	if word := placedClassWord(t, raw, id); word != 24 {
		t.Errorf("written type word after the drop %d, want 24", word)
	}
	g, _ := lancerLoad(t, filepath.Dir(path), filepath.Base(path))
	requirePlacedClasses(t, g.live, cases, "cold LOAD")
}

func TestReleaseMission41PlacedHumansHoldTheirRowClass(t *testing.T) {
	f := releaseFront(t)
	m := releaseMissionMap(t, f, 41)
	var cases []placedClassCase
	differing := 0
	for _, u := range m.Units {
		if u.ClassID < 0 || u.ClassID >= 0x1a || u.Flags&1 != 0 || u.DefID == 0 || u.DefID == 0xcdcdcdcd {
			continue
		}
		row := data.FindHumanByServerID(f.Table.Humans, int32(u.DefID))
		if row < 0 {
			continue
		}
		def, err := data.NewHumanDef(f.Table.Humans.EntryName(row), f.Table.Humans.EntryParams(row))
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, placedClassCase{unit: u.UnitID, class: def.TypeID})
		if def.TypeID != int32(u.ClassID) {
			differing++
		}
	}
	t.Logf("mission 41: %d definition-id placements, %d with a record key that differs from the row type", len(cases), differing)
	if len(cases) < 10 || differing == 0 {
		t.Fatalf("mission 41 holds %d definition-id placements, %d of them discriminating", len(cases), differing)
	}
	mw := placedClassMission(t, f, 41)
	requirePlacedClasses(t, mw, cases, "tick zero")

	path, raw := writeOrdinarySAV(t, f, "placed-humans.sav")
	for _, c := range cases {
		if word := placedClassWord(t, raw, placedClassID(t, mw, c.unit)); word != uint32(c.class) {
			t.Errorf("unit %d written type word %d, want %d", c.unit, word, c.class)
		}
	}
	for _, c := range cases {
		id := placedClassID(t, mw, c.unit)
		if slots, _ := mw.world.Equipped(id); slots[0] != 0 {
			placedClassDropWeapon(t, mw, id)
		}
	}
	requirePlacedClasses(t, mw, cases, "after the weapons are dropped")

	g, _ := lancerLoad(t, filepath.Dir(path), filepath.Base(path))
	requirePlacedClasses(t, g.live, cases, "cold LOAD")
}
