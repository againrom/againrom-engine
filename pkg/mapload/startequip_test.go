package mapload_test

// The starting-equipment hotfix, the loader's half: a class row's weapon
// reaches the SIMULATION as an item code in equipment slot 1, not only as
// numbers folded into a combat block — so a body has something to leave and
// the first equip of a mission has something to displace.
//
// It reuses unitequip_test.go's row and weapon fixtures and sightrange_test.go's
// one-placement load, on that file's own reason: the resolution under test is
// the same one those files already exercise, asked a different question.

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// seKey is this file's own class key, distinct from every other file's.
const seKey = 0x52

// seWorld is srcLoad's own load, kept whole rather than reduced to its one
// entity: what is under test lives beside the entity, in the world's
// equipment set.
func seWorld(t *testing.T, m *alm.Map, tbl *mapload.Table) *sim.World {
	t.Helper()
	w, err := mapload.FromALMWith(m, tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	return w
}

// seExpected is the code the resolver composes for the fixture's weapon —
// asked of data rather than written out, because the claim under test is
// that the item a unit WEARS is the very weapon his numbers were folded
// from, and a literal here would assert a number instead of that identity.
func seExpected(t *testing.T, name string, tbl *mapload.Table) uint16 {
	t.Helper()
	w, err := data.ResolveWeapon(name, tbl.Shapes, tbl.Materials, tbl.Weapons)
	if err != nil {
		t.Fatalf("ResolveWeapon(%q): %v", name, err)
	}
	return uint16(w.Code)
}

// TestAPlacedCreatureWearsTheWeaponItsNumbersWereFoldedFrom is the first
// half of the owner's first defect: before this hotfix a class row's weapon
// reached the definition's combat numbers and stopped there, so every unit
// on every map opened the mission with twelve empty slots and an empty
// container, and killing one left nothing behind.
func TestAPlacedCreatureWearsTheWeaponItsNumbersWereFoldedFrom(t *testing.T) {
	t.Parallel()

	row := equipUnitRow(seKey, 1, 3, 9, 10, 4, 99, 77, []string{"Melee"})
	tbl := equipTable(row, equipWeapons(), identityScale(), identityScale())
	w := seWorld(t, srcMap(seKey), tbl)

	eq, ok := w.Equipped(0)
	if !ok {
		t.Fatal("Equipped(0) answered not-ok for the one entity this world holds")
	}
	want := seExpected(t, "Melee", tbl)
	if eq[0] != want {
		t.Errorf("slot 1 = 0x%04x, want 0x%04x — the resolver's own code for the row's weapon",
			eq[0], want)
	}
	for i := 1; i < sim.EquipSlots; i++ {
		if eq[i] != 0 {
			t.Errorf("slot %d = 0x%04x, want empty — only the weapon is modelled", i+1, eq[i])
		}
	}
}

// TestAPlacedCreatureThatNamesNoWeaponWearsNothing is the other arm, and it
// is what keeps the case above from passing for a build that arms everything:
// a row whose equipment cell names nothing resolvable leaves all twelve slots
// empty, and the world holds no stock entry for it at all.
func TestAPlacedCreatureThatNamesNoWeaponWearsNothing(t *testing.T) {
	t.Parallel()

	row := equipUnitRow(seKey, 1, 3, 9, 10, 4, 99, 77, []string{"NoSuchThing"})
	tbl := equipTable(row, equipWeapons(), identityScale(), identityScale())
	w := seWorld(t, srcMap(seKey), tbl)

	if eq, _ := w.Equipped(0); eq != ([sim.EquipSlots]uint16{}) {
		t.Errorf("Equipped(0) = %v, want every slot empty", eq)
	}
	if got := w.Stock(); len(got) != 0 {
		t.Errorf("Stock() = %+v, want none — an actor holding nothing is not a holding", got)
	}
}

// A Units-row name containing NPC is not a person template. It keeps ordinary
// creature loadout and drop policy; the canonical predicate applies only after
// a placement resolves through Humans.
func TestAUnitsRowContainingNPCDoesNotAcquirePersonTemplatePolicy(t *testing.T) {
	t.Parallel()

	plain := equipUnitRow(seKey, 1, 3, 9, 10, 4, 99, 77, []string{"Melee"})
	merc := plain
	merc.name = "NPC14_1"

	items := equipWeapons()
	base := seWorld(t, srcMap(seKey), equipTable(plain, items, identityScale(), identityScale()))
	npcTbl := equipTable(merc, items, identityScale(), identityScale())
	npc := seWorld(t, srcMap(seKey), npcTbl)

	b, n := base.Entities()[0], npc.Entities()[0]
	if b != n {
		t.Errorf("a Units-row NPC name builds entity %+v, want ordinary control %+v", n, b)
	}
	baseEq, _ := base.Equipped(0)
	if baseEq[0] == 0 {
		t.Fatal("the non-NPC control wears nothing, so this case proves nothing")
	}
	if eq, _ := npc.Equipped(0); eq != baseEq {
		t.Errorf("an NPC-templated placement wears %v, want the authored live loadout %v", eq, baseEq)
	}
}

// TestAStartedPartyMemberWearsHisOwnWeapon is the owner's SECOND defect at
// its source: a member's weapon was a loader value that reached his combat
// numbers and nothing else, so the first weapon he equipped had an empty
// slot 1 to move into and superseded what he was holding.
//
// WIDENED BY 0134 (plan D-5): the mint no longer derives an equip array from
// Weapon by itself — a member's Stock entry is built from his own Worn array
// and Carried slice (this file's own GeneratedWornSet tests, below, exercise
// the resolution that fills them; this test's concern is that whatever fills
// Worn survives the mint). So the armed member here states Worn directly, at
// the same slot 1 the pre-0134 mint derived automatically, and the claim
// under test does not move: a member's starting equipment has to survive
// BOTH mission rebuilds, or the first equip he applies still has an empty
// slot to displace into.
//
// StartMissionScripted is asserted beside StartMission deliberately: it
// REBUILDS the world, and a rebuild carries only what it names. The sack list
// and the containers were each dropped by these two rebuilds for stories at a
// time, so a loadout that survives one and not the other is the exact shape
// of the defect this assertion exists to catch.
func TestAStartedPartyMemberWearsHisOwnWeapon(t *testing.T) {
	t.Parallel()

	tbl := equipTable(equipUnitRow(seKey, 1, 3, 9, 10, 4, 99, 77, nil),
		equipWeapons(), identityScale(), identityScale())
	weapon, err := data.ResolveWeapon("Melee", tbl.Shapes, tbl.Materials, tbl.Weapons)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	var armedWorn [sim.EquipSlots]uint16
	armedWorn[0] = uint16(weapon.Code)
	p := []mapload.PartyMember{{Class: 100}, {Class: 101, Weapon: &weapon, Worn: armedWorn}}

	for _, c := range []struct {
		name  string
		build func(*alm.Map) (*sim.World, mapload.Start, error)
	}{
		{"StartMission", func(m *alm.Map) (*sim.World, mapload.Start, error) {
			return mapload.StartMission(m, nil, mapload.DifficultyNormal, p)
		}},
		{"StartMissionScripted", func(m *alm.Map) (*sim.World, mapload.Start, error) {
			return mapload.StartMissionScripted(m, nil, mapload.DifficultyNormal, p, nil)
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			w, st, err := c.build(startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20}))
			if err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
			if eq, _ := w.Equipped(st.IDs[0]); eq != ([sim.EquipSlots]uint16{}) {
				t.Errorf("the bare member wears %v, want every slot empty", eq)
			}
			eq, ok := w.Equipped(st.IDs[1])
			if !ok {
				t.Fatalf("Equipped(%d) answered not-ok for a member this world holds", st.IDs[1])
			}
			if eq[0] != uint16(weapon.Code) {
				t.Errorf("the armed member's slot 1 = 0x%04x, want 0x%04x",
					eq[0], uint16(weapon.Code))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GeneratedWornSet (AC-1, AC-2, AC-15, plan D-4) — a generated character's
// own worn-set resolution, over the base row's ten equipment cells and the
// weapon character generation handed him. It reuses wear_test.go's
// wearWeapons, wearShields and wearArmors: the resolution under test is
// wearRow's and startingLoadout's, called from a second site, so the fixture
// shape is that file's own rather than a new one.
// ---------------------------------------------------------------------------

// gwsTable is a table carrying the three item collections GeneratedWornSet's
// tests resolve against, and no Units or Humans collection at all — a
// generated character's own resolution never searches either.
func gwsTable() *mapload.Table {
	return &mapload.Table{
		Shapes: identityScale(), Materials: identityScale(),
		Weapons: wearWeapons(), Shields: wearShields(), Armors: wearArmors(),
	}
}

// TestGeneratedWornSetWearsAnArmourInItsOwnRowsSlot is AC-1: an archetype's
// base row naming an armour past the weapon cell lands in the slot that
// piece's OWN row names — GeneratedWornSet's own claim over wearRow's,
// exercised from a generated character's row rather than a placed person's.
func TestGeneratedWornSetWearsAnArmourInItsOwnRowsSlot(t *testing.T) {
	t.Parallel()

	tbl := gwsTable()
	cells := []string{"", "", "Arm7", "", "", "", "", "", "", ""}
	worn, carried := mapload.GeneratedWornSet(cells, "", tbl)

	want, err := data.ResolveArmor("Arm7", tbl.Shapes, tbl.Materials, tbl.Armors)
	if err != nil {
		t.Fatalf("ResolveArmor(%q): %v", "Arm7", err)
	}
	if worn[6] != uint16(want.Code) {
		t.Errorf("slot 7 = 0x%04x, want 0x%04x — Arm7's own Slot column", worn[6], uint16(want.Code))
	}
	for i, v := range worn {
		if i != 6 && v != 0 {
			t.Errorf("slot %d = 0x%04x, want empty — only Arm7's own slot is written", i+1, v)
		}
	}
	if len(carried) != 0 {
		t.Errorf("carried = %v, want none", carried)
	}
}

// TestGeneratedWornSetCarriesAPieceNamingNoSlot is AC-2: a piece whose row
// names no slot among the twelve is carried, and no slot is overwritten by
// it.
func TestGeneratedWornSetCarriesAPieceNamingNoSlot(t *testing.T) {
	t.Parallel()

	tbl := gwsTable()
	cells := []string{"", "", "Arm0", "", "", "", "", "", "", ""}
	worn, carried := mapload.GeneratedWornSet(cells, "", tbl)

	if worn != ([sim.EquipSlots]uint16{}) {
		t.Errorf("worn = %v, want every slot empty — Arm0 names no slot among the twelve", worn)
	}
	want, err := data.ResolveArmor("Arm0", tbl.Shapes, tbl.Materials, tbl.Armors)
	if err != nil {
		t.Fatalf("ResolveArmor(%q): %v", "Arm0", err)
	}
	if len(carried) != 1 || carried[0] != uint16(want.Code) {
		t.Errorf("carried = %v, want [0x%04x]", carried, uint16(want.Code))
	}
}

// TestGeneratedWornSetCostsNothingForAnEmptyOrUnresolvableCell is AC-15: an
// empty cell and a cell naming no row this build can resolve both cost
// nothing and do not fail the resolution — asserted beside a resolvable
// cell so the two nothings are not mistaken for "nothing resolved at all".
func TestGeneratedWornSetCostsNothingForAnEmptyOrUnresolvableCell(t *testing.T) {
	t.Parallel()

	tbl := gwsTable()
	cells := []string{"", "", "Nonexistent", "Arm5", "", "", "", "", "", ""}
	worn, carried := mapload.GeneratedWornSet(cells, "", tbl)

	want, err := data.ResolveArmor("Arm5", tbl.Shapes, tbl.Materials, tbl.Armors)
	if err != nil {
		t.Fatalf("ResolveArmor(%q): %v", "Arm5", err)
	}
	if worn[4] != uint16(want.Code) {
		t.Errorf("slot 5 = 0x%04x, want 0x%04x — Arm5 still resolves beside the empty "+
			"and unresolvable cells", worn[4], uint16(want.Code))
	}
	for i, v := range worn {
		if i != 4 && v != 0 {
			t.Errorf("slot %d = 0x%04x, want empty", i+1, v)
		}
	}
	if len(carried) != 0 {
		t.Errorf("carried = %v, want none — an empty cell and an unresolvable one cost nothing", carried)
	}
}

// TestGeneratedWornSetSubstitutesTheHandedWeaponForTheRowsOwn is plan D-4's
// own claim: the weapon literal character generation handed a member takes
// the row's own weapon cell. The row's own weapon cell names "Pike" — a
// two-handed weapon this table's own wearWeapons states as such — so a
// build that resolved the row's own cell instead of the handed literal
// would show a displaced weapon in the container, and this fixture carries
// no shield to displace it with either way: this test is about WHICH name
// reaches slot 1, not about the displacement.
func TestGeneratedWornSetSubstitutesTheHandedWeaponForTheRowsOwn(t *testing.T) {
	t.Parallel()

	tbl := gwsTable()
	cells := []string{"Pike", "", "", "", "", "", "", "", "", ""}
	worn, carried := mapload.GeneratedWornSet(cells, "Sword", tbl)

	want, err := data.ResolveWeapon("Sword", tbl.Shapes, tbl.Materials, tbl.Weapons)
	if err != nil {
		t.Fatalf("ResolveWeapon(%q): %v", "Sword", err)
	}
	if worn[0] != uint16(want.Code) {
		t.Errorf("slot 1 = 0x%04x, want 0x%04x — the handed weapon, not the row's own Pike",
			worn[0], uint16(want.Code))
	}
	if len(carried) != 0 {
		t.Errorf("carried = %v, want none — nothing displaces a one-handed Sword", carried)
	}
}

// TestGeneratedWornSetLeavesTheWeaponSlotEmptyForABareCharacter is plan
// D-4's own bare case: the empty string in the weapon cell resolves through
// wearRow's own empty-cell skip exactly as any other row naming nothing
// there does, whatever the row's own weapon cell states.
func TestGeneratedWornSetLeavesTheWeaponSlotEmptyForABareCharacter(t *testing.T) {
	t.Parallel()

	tbl := gwsTable()
	cells := []string{"Pike", "", "", "", "", "", "", "", "", ""}
	worn, carried := mapload.GeneratedWornSet(cells, "", tbl)

	if worn[0] != 0 {
		t.Errorf("slot 1 = 0x%04x, want empty — a bare character's weapon cell is the empty string", worn[0])
	}
	if len(carried) != 0 {
		t.Errorf("carried = %v, want none", carried)
	}
}
