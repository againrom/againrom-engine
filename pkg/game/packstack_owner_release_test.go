package game

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// packStackPickUp orders hero to take the Sack lying on (x, y) and runs
// ordinary ticks until it is gone.
func packStackPickUp(t *testing.T, f *FrontEnd, hero sim.EntityID, x, y int32) {
	t.Helper()
	lying := func() bool { return groundAt(f.live.world.Sacks(), x, y) != nil }
	if !lying() {
		t.Fatalf("no Sack lies on (%d,%d)", x, y)
	}
	f.live.grab(uint32(hero), int(x), int(y), true)
	for tick := 0; tick < 20000 && lying(); tick++ {
		f.live.tick()
	}
	if lying() {
		t.Fatalf("actor %d did not pick up the Sack on (%d,%d)", hero, x, y)
	}
}

// packStackSavedItems lists "x<count> <price> <W52 bytes 22-23>" for each
// saved item record of code in the SAV at path, in file order. The bytes are
// "-" for a record that is not a Weapon.
func packStackSavedItems(t *testing.T, path string, code uint16) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if !savedItemClass(r.Class) {
			continue
		}
		if c, err := savedStructureValue(r, "F40"); err != nil || c != uint32(code) {
			continue
		}
		count, _ := savedStructureValue(r, "F42")
		price, _ := savedStructureValue(r, "T1C")
		tail := "-"
		if r.Class == "Weapon" {
			w52, err := savedObjectRaw(r, "W52", 24)
			if err != nil {
				t.Fatal(err)
			}
			tail = fmt.Sprintf("%x", w52[22:])
		}
		out = append(out, fmt.Sprintf("x%d %d %s", count, int32(price), tail))
	}
	return out
}

// TestReleaseSackBowsJoinTheRestoredCell: a LOAD of the owner's original
// mission SAV game0007.sav, pinned by its SHA-256, restores Danath in
// mission 20 with the Wood Short Bow 0x8114 x9 in one cell whose Weapon bytes
// 22 and 23 are 0f01, and two Sacks on the map that hold a 0x8114 each, with
// the bytes f700 at (41,75) and e700 at (49,83). Picking both Sacks up joins
// them into that cell: 0x8114 x11 keeping its bytes 0f01, the incoming
// Items' two bytes dropped, through a mission SAVE and cold LOAD. The Sack's
// other items join their own equal cells (ITEM-STACK-003, ITEM-MERGE-129,
// DIV-762: no claim names a reader of bytes 22 and 23).
func TestReleaseSackBowsJoinTheRestoredCell(t *testing.T) {
	const bow, arrow, bolt = 0x8114, 0xa70f, 0xac1c
	_, raw := groundCorpusFile(t, "2026-08-02/game0007.sav", "a7cb35ea5d87c9c7a9b8cfd70f8a1f61b6927ca089e468a9d6fe983da777156e")
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("LOAD of game0007.sav: town=%v err=%v", town, err)
	}
	if err := f.App("sack bows").OpenMission(open); err != nil {
		t.Fatalf("open mission 20: %v", err)
	}
	hero := equipmentReturnHero(t, f)
	stacks, _ := f.live.world.CarriedStacks(hero)
	i := slices.IndexFunc(stacks, func(st sim.ItemStack) bool { return st.Code == bow })
	if got := packStackCells(stacks, arrow, bow, bolt); i < 0 || got != "[0xa70f x10 0x8114 x9 0xac1c x12]" {
		t.Fatalf("restored pack %s; want 0xa70f x10, 0x8114 x9 and 0xac1c x12", got)
	}
	blocks := stacks[i].SourceEquipment
	if got := fmt.Sprintf("%x", blocks.Attack[22:]); got != "0f01" {
		t.Fatalf("restored 0x8114 Weapon bytes 22-23 %s, want 0f01", got)
	}
	for _, sack := range []struct {
		x, y int32
		tail string
	}{{41, 75, "f700"}, {49, 83, "e700"}} {
		at := groundAt(f.live.world.Sacks(), sack.x, sack.y)
		if at == nil || len(at.ItemInstances) == 0 || at.ItemInstances[0].Code != bow {
			t.Fatalf("Sack (%d,%d) %+v, want the 0x8114 first", sack.x, sack.y, at)
		}
		held := at.ItemInstances[0].SourceEquipment
		if got := fmt.Sprintf("%x", held.Attack[22:]); got != sack.tail {
			t.Fatalf("Sack (%d,%d) 0x8114 Weapon bytes 22-23 %s, want %s", sack.x, sack.y, got, sack.tail)
		}
		same, want := held, blocks
		same.Attack[22], same.Attack[23], want.Attack[22], want.Attack[23] = 0, 0, 0, 0
		if same != want {
			t.Fatalf("Sack (%d,%d) 0x8114 differs from the pack's in more than bytes 22-23", sack.x, sack.y)
		}
	}
	packStackPickUp(t, f, hero, 49, 83)
	packStackPickUp(t, f, hero, 41, 75)
	check := func(label string, f *FrontEnd) {
		t.Helper()
		stacks, _ := f.live.world.CarriedStacks(equipmentReturnHero(t, f))
		got := packStackCells(stacks, arrow, bow, bolt)
		j := slices.IndexFunc(stacks, func(st sim.ItemStack) bool { return st.Code == bow })
		if got != "[0xa70f x11 0x8114 x11 0xac1c x13]" || j != i || stacks[j].SourceEquipment != blocks {
			t.Fatalf("%s pack %s with the bow at cell %d and its Weapon bytes kept %v; want 0xa70f x11, 0x8114 x11 in the restored cell %d and 0xac1c x13",
				label, got, j, j == i && stacks[j].SourceEquipment == blocks, i)
		}
	}
	check("after the pickups", f)
	path := saveCorpseMission(t, f, t.TempDir())
	saved := packStackSavedItems(t, path, bow)
	if !slices.Contains(saved, "x11 133 0f01") || slices.ContainsFunc(saved, func(s string) bool { return strings.HasSuffix(s, " f700") || strings.HasSuffix(s, " e700") }) {
		t.Fatalf("mission SAVE writes 0x8114 records %v; want one x11 with bytes 0f01 and none with f700 or e700", saved)
	}
	check("cold LOAD", loadAreaContinuation(t, path))
}

// TestReleaseEquallyEnchantedStavesShareOneCell: a LOAD of the owner's
// original mission SAV game9999.sav, pinned by its SHA-256, restores mission
// 141 with Danath holding one staff 0x916e whose effect is the spell link
// (41 0 1966094) at price 313071, and Naira holding three of those in one
// cell and two staves with another effect in a second. Danath drops his staff
// and walks off, Naira takes it: her cell of three becomes four, the other
// enchantment keeps its cell of two, and Danath's pack holds none, through a
// mission SAVE and cold LOAD. Equally enchanted equipment stacks (DIV-1473:
// ITEM-MERGE-129 gives an enchanted Item no cell to join).
func TestReleaseEquallyEnchantedStavesShareOneCell(t *testing.T) {
	const staff = 0x916e
	_, raw := groundCorpusFile(t, "2026-09-27/game9999.sav", "8da6bec860289ac804563d5a55df0e201d2403138ed6c55b3a32b43073f1cd09")
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("LOAD of game9999.sav: town=%v err=%v", town, err)
	}
	if err := f.App("enchanted staves").OpenMission(open); err != nil {
		t.Fatalf("open mission 141: %v", err)
	}
	member := func(f *FrontEnd, name string) sim.EntityID {
		t.Helper()
		for i, p := range f.live.mission.party {
			if p.Name == name {
				return f.live.mission.ids[i]
			}
		}
		t.Fatalf("mission 141 has no party member %s", name)
		return 0
	}
	cells := func(f *FrontEnd, name string) string {
		stacks, _ := f.live.world.CarriedStacks(member(f, name))
		var out []string
		for _, st := range stacks {
			if st.Code == staff {
				out = append(out, fmt.Sprintf("x%d@%d%v", st.Count, st.Price, st.Effects))
			}
		}
		return fmt.Sprint(out)
	}
	const (
		danathBefore = "[x1@313071[{41 0 1966094}]]"
		nairaBefore  = "[x2@205544[{41 0 3932173}] x3@313071[{41 0 1966094}]]"
		danathAfter  = "[]"
		nairaAfter   = "[x2@205544[{41 0 3932173}] x4@313071[{41 0 1966094}]]"
	)
	if d, n := cells(f, "Danath"), cells(f, "Naira"); d != danathBefore || n != nairaBefore {
		t.Fatalf("restored staves: Danath %s, Naira %s; want %s and %s", d, n, danathBefore, nairaBefore)
	}
	danath, naira := member(f, "Danath"), member(f, "Naira")
	stacks, _ := f.live.world.CarriedStacks(danath)
	slot := slices.IndexFunc(stacks, func(st sim.ItemStack) bool { return st.Code == staff })
	at, _ := f.live.world.Entity(danath)
	f.live.pending = append(f.live.pending, sim.DropCarried(danath, sim.ItemSlot(slot), sim.CellPoint{X: at.X, Y: at.Y}))
	f.live.tick()
	f.LiveOrder(uint32(danath), int(at.X)-3, int(at.Y))
	f.LiveAdvance(60)
	if now, _ := f.live.world.Entity(danath); now.X == at.X && now.Y == at.Y {
		t.Fatalf("Danath still stands on the Sack at (%d,%d)", at.X, at.Y)
	}
	packStackPickUp(t, f, naira, at.X, at.Y)
	check := func(label string, f *FrontEnd) {
		t.Helper()
		if d, n := cells(f, "Danath"), cells(f, "Naira"); d != danathAfter || n != nairaAfter {
			t.Fatalf("%s staves: Danath %s, Naira %s; want %s and %s", label, d, n, danathAfter, nairaAfter)
		}
	}
	check("after the hand-over", f)
	path := saveCorpseMission(t, f, t.TempDir())
	saved := packStackSavedItems(t, path, staff)
	slices.Sort(saved)
	if want := []string{"x1 313071 0000", "x2 205544 0000", "x4 313071 0000"}; !slices.Equal(saved, want) {
		t.Fatalf("mission SAVE writes 0x916e records %v; want %v (Fergard's worn staff, Naira's two cells)", saved, want)
	}
	check("cold LOAD", loadAreaContinuation(t, path))
}
