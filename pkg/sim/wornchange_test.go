package sim

import "testing"

// recordingHolder is a WornHolder that counts the writes the transaction asks
// of it.
type recordingHolder struct {
	keeps         bool
	drops, builds int
	refuse        bool
	builtSpell    SourceItemSpell
	lastDropSlot  int
	lastBuildSlot int
}

func (h *recordingHolder) KeepsWeaponSpell() bool { return h.keeps }

func (h *recordingHolder) DropWeaponSpell(slot int, item *ItemInstance) bool {
	h.drops++
	h.lastDropSlot = slot
	item.SourceEquipment.Spell = SourceItemSpell{}
	return !h.refuse
}

func (h *recordingHolder) BuildWeaponSpell(slot int, item *ItemInstance) bool {
	h.builds++
	h.lastBuildSlot = slot
	if h.refuse {
		return false
	}
	item.SourceEquipment.Spell = h.builtSpell
	return true
}

// TestWornChangeAsksTheHolderOnlyUnderTheRule: the take-off asks its holder to
// drop a Spell only for a classed Weapon that holds one, outside the second
// game's row, and the wear asks for a build only for an item that holds none.
// The controls are the cases that do ask: a transaction that never asked, or
// always asked, fails one of them.
func TestWornChangeAsksTheHolderOnlyUnderTheRule(t *testing.T) {
	spell := SourceItemSpell{Present: true, ID: 14, Range: 7, ManaCost: 80}
	held, gap := spelllessStaff(true), spelllessStaff(false)
	plain := ItemInstance{Code: 0x0201, Kind: 3}
	plain.SourceEquipment.Spell = spell
	for _, tc := range []struct {
		name  string
		keeps bool
		slot  int
		item  ItemInstance
		ok    bool
		drops int
	}{
		{"a held Spell", false, 1, held, true, 1},
		{"a Weapon in another slot", false, 2, held, true, 1},
		{"no Spell", false, 1, gap, true, 0},
		{"no Weapon", false, 3, plain, true, 0},
		{"the second game", true, 1, held, true, 0},
		{"slot 0", false, 0, held, false, 0},
		{"slot 13", false, EquipSlots + 1, held, false, 0},
	} {
		h := &recordingHolder{keeps: tc.keeps}
		got, ok := TakeOffWorn(h, tc.slot, tc.item.Clone())
		if ok != tc.ok || h.drops != tc.drops || h.builds != 0 {
			t.Fatalf("take-off, %s: ok %v, %d drops, %d builds; want ok %v, %d drops", tc.name, ok, h.drops, h.builds, tc.ok, tc.drops)
		}
		if tc.drops != 0 && (got.SourceEquipment.Spell.Present || h.lastDropSlot != tc.slot) {
			t.Fatalf("take-off, %s: the item keeps %+v, slot %d", tc.name, got.SourceEquipment.Spell, h.lastDropSlot)
		}
		if tc.drops == 0 && !ItemEqual(got, tc.item) {
			t.Fatalf("take-off, %s: the item changed to %+v", tc.name, got)
		}
	}
	if _, ok := TakeOffWorn(&recordingHolder{refuse: true}, 1, held.Clone()); ok {
		t.Fatal("take-off: a holder's refusal was not returned")
	}
	for _, tc := range []struct {
		name   string
		keeps  bool
		slot   int
		item   ItemInstance
		builds int
	}{
		{"no Spell", false, 1, gap, 1},
		{"a held Spell", false, 1, held, 0},
		{"an empty slot", false, 1, ItemInstance{}, 0},
		{"the second game", true, 1, gap, 0},
		{"slot 0", false, 0, gap, 0},
	} {
		h := &recordingHolder{keeps: tc.keeps, builtSpell: spell}
		got, built := WearWorn(h, tc.slot, tc.item.Clone())
		if built != (tc.builds != 0) || h.builds != tc.builds || h.drops != 0 {
			t.Fatalf("wear, %s: built %v, %d builds, %d drops; want %d builds", tc.name, built, h.builds, h.drops, tc.builds)
		}
		if tc.builds != 0 && (got.SourceEquipment.Spell != spell || h.lastBuildSlot != tc.slot) {
			t.Fatalf("wear, %s: the item holds %+v", tc.name, got.SourceEquipment.Spell)
		}
	}
	if got, built := WearWorn(&recordingHolder{refuse: true}, 1, gap.Clone()); built || !ItemEqual(got, gap) {
		t.Fatal("wear: a refused build changed the item")
	}
}

// TestWornItemEqualLeavesOutOnlyTheSpell: a Weapon with and without its Spell
// is one held item; a different Weapon, the control, is not.
func TestWornItemEqualLeavesOutOnlyTheSpell(t *testing.T) {
	if !WornItemEqual(spelllessStaff(true), spelllessStaff(false)) {
		t.Fatal("the staff taken off is not the staff worn")
	}
	other := spelllessStaff(true)
	other.Effects = staffEffects(6, 20)
	if WornItemEqual(spelllessStaff(true), other) {
		t.Fatal("control: a staff with another castSpell Effect is the same item")
	}
}
