package mapload_test

import (
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

const slotWeaponHands = 0xe

func wearWeaponRow(kind, min, max, toHit, defence, charge, relax, hands int32) []int32 {
	p := weaponRow(kind, min, max, toHit, defence, charge, relax)
	for len(p) <= slotWeaponHands {
		p = append(p, -1)
	}
	p[slotWeaponHands] = hands
	return p
}

// wearWeapons is this file's weapon collection: "Sword", a shipped-shaped
// one-handed weapon with Hands 0, and "Pike", a two-handed one — the only
// difference AC-3's own pair needs.
func wearWeapons() defCollection {
	return defCollection{
		{},
		{name: "Sword", params: wearWeaponRow(data.SkillBlade, 5, 9, 7, 2, 15, 9, 0)},
		{name: "Pike", params: wearWeaponRow(data.SkillBlade, 5, 9, 7, 2, 15, 9, 2)},
		{name: "Bow", params: wearWeaponRow(11, 5, 9, 7, 2, 15, 9, 0)},
	}
}

func wearShields() defCollection {
	return defCollection{{}, {name: "Buckler"}}
}

// slotArmorSlot is the Armors row's own Slot column, transcribed here for
// the same reason slotWeaponHands is.
const slotArmorSlot = 4

// slotArmorAbsorption is the Armors row's own Absorption column (0136:
// data.ArmorFromCode, wear.go's own fillArmor), transcribed here for the
// same reason slotArmorSlot is: the widest column any resolver in
// pkg/data reads off an Armors row since 0136, and this suite's own rows
// build out to it so a piece this file resolves is never refused on
// account of width alone (plan's own risk, "a fixture widening reaches
// another package's test file").
const slotArmorAbsorption = 10

// armorRow is an Armors row wide enough to carry the Slot column and every
// column past it 0136 widened resolution to reach, stating Slot alone —
// every other cell, defence and absorption included, is the format's own
// empty one, because this suite asks only where a piece is worn and never
// what it is worth.
func armorRow(slot int32) []int32 {
	p := make([]int32, slotArmorAbsorption+1)
	for i := range p {
		p[i] = -1
	}
	p[slotArmorSlot] = slot
	return p
}

// wearArmors is this file's armour collection: one row per Slot value this
// suite exercises, named for the slot it states — including the two AC-4
// names to make carried rather than worn.
func wearArmors() defCollection {
	return defCollection{
		{},
		{name: "Arm3", params: armorRow(3)},
		{name: "Arm4", params: armorRow(4)},
		{name: "Arm5", params: armorRow(5)},
		{name: "Arm6", params: armorRow(6)},
		{name: "Arm7", params: armorRow(7)},
		{name: "Arm8", params: armorRow(8)},
		{name: "Arm9", params: armorRow(9)},
		{name: "Arm10", params: armorRow(10)},
		{name: "Arm0", params: armorRow(0)},
		{name: "Arm13", params: armorRow(13)},
	}
}

func wearTable(equipment []string, weapons, shields, armors data.Collection) *mapload.Table {
	return &mapload.Table{
		Units:     defCollection{{}},
		Humans:    defCollection{{}, {name: "k7", params: fullHumanRow(), strings: equipment}},
		Shapes:    identityScale(),
		Materials: identityScale(),
		Weapons:   weapons, Shields: shields, Armors: armors,
	}
}

// wearWorld builds the one-placement world humanMap() names, over tbl, and
// hands back the world itself rather than reducing it to one entity —
// human_test.go's own loadHuman does the latter, but what is under test
// here lives beside the entity, in the world's own equipment set and
// container.
func wearWorld(t *testing.T, tbl *mapload.Table) *sim.World {
	t.Helper()
	w, err := mapload.FromALMWith(humanMap(), tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	return w
}

func TestAuthoredSpeedAndSightEffectsReachThePlacedPersonAndTheStoredPiece(t *testing.T) {
	t.Parallel()

	plainCells := make([]string, 10)
	plainCells[2] = "Arm3"
	enchantedCells := append([]string(nil), plainCells...)
	enchantedCells[2] = "Arm3{speed=5,scanRange=7}"
	plain := wearWorld(t, wearTable(plainCells, wearWeapons(), wearShields(), wearArmors()))
	enchanted := wearWorld(t, wearTable(enchantedCells, wearWeapons(), wearShields(), wearArmors()))
	base, got := plain.Entities()[0], enchanted.Entities()[0]
	if got.Speed != base.Speed+5 || got.ScanRange != base.ScanRange+7 {
		t.Fatalf("enchanted placed person speed/sight = %d/%d, want base %d/%d plus 5/7",
			got.Speed, got.ScanRange, base.Speed, base.ScanRange)
	}
	worn, _ := enchanted.EquippedItems(0)
	wantEffects := []sim.ItemEffect{{Kind: 17, Operand: 5}, {Kind: 19, Operand: 7}}
	if !reflect.DeepEqual(worn[2].Effects, wantEffects) {
		t.Fatalf("stored armour effects = %+v, want %+v", worn[2].Effects, wantEffects)
	}
}

func TestARowWearsWeaponShieldAndEightArmours(t *testing.T) {
	t.Parallel()

	weapons, shields, armors := wearWeapons(), wearShields(), wearArmors()
	armourNames := []string{"Arm3", "Arm4", "Arm5", "Arm6", "Arm7", "Arm8", "Arm9", "Arm10"}
	equipment := append([]string{"Sword", "Buckler"}, armourNames...)
	tbl := wearTable(equipment, weapons, shields, armors)
	w := wearWorld(t, tbl)

	eq, ok := w.Equipped(0)
	if !ok {
		t.Fatal("Equipped(0) answered not-ok for the one entity this world holds")
	}

	wantWeapon, err := data.ResolveWeapon("Sword", tbl.Shapes, tbl.Materials, weapons)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	if eq[0] != uint16(wantWeapon.Code) {
		t.Errorf("slot 1 = 0x%04x, want 0x%04x", eq[0], uint16(wantWeapon.Code))
	}
	wantShield, err := data.ResolveShield("Buckler", tbl.Shapes, tbl.Materials, shields)
	if err != nil {
		t.Fatalf("ResolveShield: %v", err)
	}
	if eq[1] != uint16(wantShield.Code) {
		t.Errorf("slot 2 = 0x%04x, want 0x%04x", eq[1], uint16(wantShield.Code))
	}
	// Arm3..Arm10 state slots 3..10 by their own name.
	for i, name := range armourNames {
		wantArmour, err := data.ResolveArmor(name, tbl.Shapes, tbl.Materials, armors)
		if err != nil {
			t.Fatalf("ResolveArmor(%q): %v", name, err)
		}
		slot := i + 3
		if eq[slot-1] != uint16(wantArmour.Code) {
			t.Errorf("slot %d (%s) = 0x%04x, want 0x%04x", slot, name, eq[slot-1], uint16(wantArmour.Code))
		}
	}
	// Slots 11 and 12 name no cell of this ten-cell row and stay empty.
	if eq[10] != 0 || eq[11] != 0 {
		t.Errorf("slots 11/12 = 0x%04x/0x%04x, want both empty", eq[10], eq[11])
	}
	if carried, ok := w.Carried(0); !ok || len(carried) != 0 {
		t.Errorf("Carried(0) = %v, ok=%v, want an empty container", carried, ok)
	}
}

// TestAnArmoursSlotIsItsOwnRowsColumnNotItsCellIndex is AC-2: cell 2's own
// armour states Slot 7 and cell 8's states Slot 4, so the first wears in
// slot 7 and the second in slot 4 — neither in the slot its cell index
// would suggest (slot 3 and slot 9 respectively).
func TestAnArmoursSlotIsItsOwnRowsColumnNotItsCellIndex(t *testing.T) {
	t.Parallel()

	armors := wearArmors()
	equipment := []string{"", "", "Arm7", "", "", "", "", "", "Arm4", ""}
	tbl := wearTable(equipment, wearWeapons(), wearShields(), armors)
	w := wearWorld(t, tbl)

	eq, ok := w.Equipped(0)
	if !ok {
		t.Fatal("Equipped(0) answered not-ok")
	}
	want7, err := data.ResolveArmor("Arm7", tbl.Shapes, tbl.Materials, armors)
	if err != nil {
		t.Fatalf("ResolveArmor(%q): %v", "Arm7", err)
	}
	want4, err := data.ResolveArmor("Arm4", tbl.Shapes, tbl.Materials, armors)
	if err != nil {
		t.Fatalf("ResolveArmor(%q): %v", "Arm4", err)
	}
	if eq[6] != uint16(want7.Code) {
		t.Errorf("slot 7 = 0x%04x, want 0x%04x — cell 2's own Slot column", eq[6], uint16(want7.Code))
	}
	if eq[3] != uint16(want4.Code) {
		t.Errorf("slot 4 = 0x%04x, want 0x%04x — cell 8's own Slot column", eq[3], uint16(want4.Code))
	}
	// The slots the two cells' own INDICES would have named, had this build
	// used them, stay empty.
	if eq[2] != 0 {
		t.Errorf("slot 3 (cell 2's own index) = 0x%04x, want empty", eq[2])
	}
	if eq[8] != 0 {
		t.Errorf("slot 9 (cell 8's own index) = 0x%04x, want empty", eq[8])
	}
}

// TestATwoHandedWeaponKeepsTheShieldInThePack applies the held-slot rule at
// mission construction: a row whose weapon fills both hands keeps the weapon
// worn and the shield remains owned.
func TestATwoHandedWeaponKeepsTheShieldInThePack(t *testing.T) {
	t.Parallel()

	weapons, shields := wearWeapons(), wearShields()
	tbl := wearTable([]string{"Pike", "Buckler"}, weapons, shields, wearArmors())
	w := wearWorld(t, tbl)

	eq, ok := w.Equipped(0)
	if !ok {
		t.Fatal("Equipped(0) answered not-ok")
	}
	wantWeapon, err := data.ResolveWeapon("Pike", tbl.Shapes, tbl.Materials, weapons)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	if eq[0] != uint16(wantWeapon.Code) || eq[1] != 0 {
		t.Errorf("held slots = 0x%04x/0x%04x, want weapon 0x%04x and no shield", eq[0], eq[1], uint16(wantWeapon.Code))
	}
	wantShield, err := data.ResolveShield("Buckler", tbl.Shapes, tbl.Materials, shields)
	if err != nil {
		t.Fatalf("ResolveShield: %v", err)
	}
	carried, ok := w.Carried(0)
	if !ok || len(carried) != 1 || carried[0] != uint16(wantShield.Code) {
		t.Errorf("Carried(0) = %v, ok=%v, want shield [0x%04x]", carried, ok, uint16(wantShield.Code))
	}
}

// TestAOneHandedWeaponIsNotDisplacedByTheShield is AC-3's second half: the
// same row with the shipped one-handed Hands value 0 wears both.
func TestAOneHandedWeaponIsNotDisplacedByTheShield(t *testing.T) {
	t.Parallel()

	weapons, shields := wearWeapons(), wearShields()
	tbl := wearTable([]string{"Sword", "Buckler"}, weapons, shields, wearArmors())
	w := wearWorld(t, tbl)

	eq, ok := w.Equipped(0)
	if !ok {
		t.Fatal("Equipped(0) answered not-ok")
	}
	wantWeapon, err := data.ResolveWeapon("Sword", tbl.Shapes, tbl.Materials, weapons)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	if eq[0] != uint16(wantWeapon.Code) {
		t.Errorf("slot 1 = 0x%04x, want 0x%04x — a one-handed weapon is not displaced", eq[0], uint16(wantWeapon.Code))
	}
	wantShield, err := data.ResolveShield("Buckler", tbl.Shapes, tbl.Materials, shields)
	if err != nil {
		t.Fatalf("ResolveShield: %v", err)
	}
	if eq[1] != uint16(wantShield.Code) {
		t.Errorf("slot 2 = 0x%04x, want 0x%04x", eq[1], uint16(wantShield.Code))
	}
	if carried, ok := w.Carried(0); !ok || len(carried) != 0 {
		t.Errorf("Carried(0) = %v, ok=%v, want empty — nothing displaces a one-handed weapon", carried, ok)
	}
}

func TestARangedWeaponKeepsTheShieldInThePackEvenWhenHandsIsZero(t *testing.T) {
	weapons, shields := wearWeapons(), wearShields()
	tbl := wearTable([]string{"Bow", "Buckler"}, weapons, shields, wearArmors())
	w := wearWorld(t, tbl)
	eq, _ := w.Equipped(0)
	if eq[0] == 0 || eq[1] != 0 {
		t.Fatalf("equipment = %v, want ranged weapon worn and shield carried", eq)
	}
	bow, err := data.ResolveWeapon("Bow", tbl.Shapes, tbl.Materials, weapons)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	carried, _ := w.Carried(0)
	shield, err := data.ResolveShield("Buckler", tbl.Shapes, tbl.Materials, shields)
	if err != nil {
		t.Fatalf("ResolveShield: %v", err)
	}
	if eq[0] != uint16(bow.Code) || len(carried) != 1 || carried[0] != uint16(shield.Code) {
		t.Fatalf("held/carried = %#x/%v, want bow %#x and shield %#x", eq[0], carried, uint16(bow.Code), uint16(shield.Code))
	}
}

func TestShieldWithoutWeaponStartsWornWithCompleteInstance(t *testing.T) {
	weapons, shields := wearWeapons(), wearShields()
	tbl := wearTable([]string{"", "Buckler"}, weapons, shields, wearArmors())
	w := wearWorld(t, tbl)
	equipped, _ := w.EquippedItems(0)
	carried, _ := w.CarriedItems(0)
	buckler, err := data.ResolveShield("Buckler", tbl.Shapes, tbl.Materials, shields)
	if err != nil {
		t.Fatalf("ResolveShield: %v", err)
	}
	if !equipped[0].Empty() || equipped[1].Code != uint16(buckler.Code) || len(carried) != 0 {
		t.Fatalf("shield-only construction left equipped=%+v carried=%+v, want the shield worn alone", equipped, carried)
	}
}

func TestNormalizeShieldLoadoutMovesAShieldBesideATwoHandedWeaponAndKeepsMagicAndValue(t *testing.T) {
	weapons := wearWeapons()
	tbl := wearTable(nil, weapons, wearShields(), wearArmors())
	pike, err := data.ResolveWeapon("Pike", tbl.Shapes, tbl.Materials, weapons)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	shield := sim.ItemInstance{Code: 0x0201, Kind: 1, Price: 722,
		Effects: []sim.ItemEffect{{Kind: 15, Operand: 9}}}
	var worn [sim.EquipSlots]sim.ItemInstance
	worn[0] = sim.ItemInstance{Code: uint16(pike.Code), Kind: 2, Price: 300}
	worn[1] = shield.Clone()
	var carried []sim.ItemInstance
	if !mapload.NormalizeShieldLoadout(&worn, &carried, tbl) {
		t.Fatal("NormalizeShieldLoadout reported no change for a shield beside a two-handed weapon")
	}
	if worn[0].Code != uint16(pike.Code) || !worn[1].Empty() || len(carried) != 1 || carried[0].Code != shield.Code ||
		carried[0].Kind != shield.Kind || carried[0].Price != shield.Price ||
		len(carried[0].Effects) != 1 || carried[0].Effects[0] != shield.Effects[0] {
		t.Fatalf("normalized worn/carried = %+v/%+v, want the weapon worn and complete shield %+v in pack", worn, carried, shield)
	}
}

func TestNormalizeShieldLoadoutKeepsAShieldWornAlone(t *testing.T) {
	shield := sim.ItemInstance{Code: 0x0201, Kind: 1, Price: 722,
		Effects: []sim.ItemEffect{{Kind: 15, Operand: 9}}}
	var worn [sim.EquipSlots]sim.ItemInstance
	worn[1] = shield.Clone()
	var carried []sim.ItemInstance
	if mapload.NormalizeShieldLoadout(&worn, &carried, wearTable(nil, wearWeapons(), wearShields(), wearArmors())) {
		t.Fatal("NormalizeShieldLoadout changed a shield worn alone")
	}
	if worn[1].Code != shield.Code || worn[1].Price != shield.Price || len(worn[1].Effects) != 1 || len(carried) != 0 {
		t.Fatalf("worn/carried = %+v/%+v, want the complete shield still worn and nothing carried", worn, carried)
	}
}

func TestAnArmourWithNoValidSlotIsCarried(t *testing.T) {
	t.Parallel()

	armors := wearArmors()
	for _, tc := range []struct {
		name string
		cell string
	}{
		{"Slot column 0", "Arm0"},
		{"Slot column 13", "Arm13"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tbl := wearTable([]string{"", "", tc.cell}, wearWeapons(), wearShields(), armors)
			w := wearWorld(t, tbl)

			eq, ok := w.Equipped(0)
			if !ok {
				t.Fatal("Equipped(0) answered not-ok")
			}
			if eq != ([sim.EquipSlots]uint16{}) {
				t.Errorf("Equipped(0) = %v, want every slot empty — %s names no slot", eq, tc.cell)
			}
			want, err := data.ResolveArmor(tc.cell, tbl.Shapes, tbl.Materials, armors)
			if err != nil {
				t.Fatalf("ResolveArmor(%q): %v", tc.cell, err)
			}
			carried, ok := w.Carried(0)
			if !ok || len(carried) != 1 || carried[0] != uint16(want.Code) {
				t.Errorf("Carried(0) = %v, ok=%v, want [0x%04x]", carried, ok, uint16(want.Code))
			}
		})
	}
}

// TestAnUnresolvableArmourCellIsDroppedAndTheRestOfTheRowStands is AC-5: an
// armour cell naming no row at all is neither worn nor carried, and the
// rest of the row — the weapon, the shield and a later armour cell — is
// unaffected.
func TestAnUnresolvableArmourCellIsDroppedAndTheRestOfTheRowStands(t *testing.T) {
	t.Parallel()

	weapons, shields, armors := wearWeapons(), wearShields(), wearArmors()
	tbl := wearTable([]string{"Sword", "Buckler", "Nonexistent", "Arm5"}, weapons, shields, armors)
	w := wearWorld(t, tbl)

	eq, ok := w.Equipped(0)
	if !ok {
		t.Fatal("Equipped(0) answered not-ok")
	}
	wantWeapon, err := data.ResolveWeapon("Sword", tbl.Shapes, tbl.Materials, weapons)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	wantShield, err := data.ResolveShield("Buckler", tbl.Shapes, tbl.Materials, shields)
	if err != nil {
		t.Fatalf("ResolveShield: %v", err)
	}
	wantArm5, err := data.ResolveArmor("Arm5", tbl.Shapes, tbl.Materials, armors)
	if err != nil {
		t.Fatalf("ResolveArmor: %v", err)
	}
	if eq[0] != uint16(wantWeapon.Code) || eq[1] != uint16(wantShield.Code) || eq[4] != uint16(wantArm5.Code) {
		t.Errorf("eq = %v, want slot1=0x%04x slot2=0x%04x slot5=0x%04x",
			eq, uint16(wantWeapon.Code), uint16(wantShield.Code), uint16(wantArm5.Code))
	}
	// Cell 2 ("Nonexistent") is the row's third cell — an armour cell whose
	// class this table can model but whose name resolves to no row — and it
	// wrote nothing: not slot 3 (its own index, had it resolved) and not the
	// container.
	if eq[2] != 0 {
		t.Errorf("slot 3 = 0x%04x, want empty — the unresolvable cell wrote no slot", eq[2])
	}
	if carried, ok := w.Carried(0); !ok || len(carried) != 0 {
		t.Errorf("Carried(0) = %v, ok=%v, want empty — a name naming no row is dropped, not carried", carried, ok)
	}
}

// An NPC template's death-time suppression cannot alter the living loadout.
// Weapon, shield and armour all resolve through the row's ordinary cells.
func TestAnNPCTemplatedPersonKeepsEveryLivingSlot(t *testing.T) {
	t.Parallel()

	weapons, shields, armors := wearWeapons(), wearShields(), wearArmors()
	equipment := []string{"Sword", "Buckler", "Arm3", "Arm4"}
	tbl := &mapload.Table{
		Units:     defCollection{{}},
		Humans:    defCollection{{}, {name: "NPC14_1", params: fullHumanRow(), strings: equipment}},
		Shapes:    identityScale(),
		Materials: identityScale(),
		Weapons:   weapons, Shields: shields, Armors: armors,
	}
	w := wearWorld(t, tbl)

	eq, ok := w.Equipped(0)
	if !ok {
		t.Fatal("Equipped(0) answered not-ok")
	}
	wantWeapon, err := data.ResolveWeapon("Sword", tbl.Shapes, tbl.Materials, weapons)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	wantShield, err := data.ResolveShield("Buckler", tbl.Shapes, tbl.Materials, shields)
	if err != nil {
		t.Fatalf("ResolveShield: %v", err)
	}
	if eq[0] != uint16(wantWeapon.Code) || eq[1] != uint16(wantShield.Code) {
		t.Errorf("slots 1/2 = 0x%04x/0x%04x, want live weapon/shield 0x%04x/0x%04x",
			eq[0], eq[1], uint16(wantWeapon.Code), uint16(wantShield.Code))
	}
	want3, err := data.ResolveArmor("Arm3", tbl.Shapes, tbl.Materials, armors)
	if err != nil {
		t.Fatalf("ResolveArmor: %v", err)
	}
	want4, err := data.ResolveArmor("Arm4", tbl.Shapes, tbl.Materials, armors)
	if err != nil {
		t.Fatalf("ResolveArmor: %v", err)
	}
	if eq[2] != uint16(want3.Code) || eq[3] != uint16(want4.Code) {
		t.Errorf("armour slots 3/4 = 0x%04x/0x%04x, want 0x%04x/0x%04x — the gate does not reach them",
			eq[2], eq[3], uint16(want3.Code), uint16(want4.Code))
	}
	ents := w.Entities()
	if len(ents) != 1 || !ents[0].SuppressCorpseLoot {
		t.Fatalf("NPC template entities = %+v, want the canonical corpse-loot property", ents)
	}

	// The control: the template name alone cannot change any live slot.
	plainTbl := wearTable(equipment, weapons, shields, armors)
	plain := wearWorld(t, plainTbl)
	if peq, _ := plain.Equipped(0); peq != eq {
		t.Fatalf("NPC live loadout %v differs from non-NPC control %v", eq, peq)
	}
	if plain.Entities()[0].SuppressCorpseLoot {
		t.Fatal("non-NPC control carries the corpse-loot suppression property")
	}
	_, roster, err := mapload.FromALMRoster(humanMap(), tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMRoster: %v", err)
	}
	if member, ok := roster[0]; !ok || !member.SuppressCorpseLoot {
		t.Fatalf("NPC roster template = %+v, ok=%v; want the canonical corpse-loot property", member, ok)
	}
}

func TestATableMissingArmorsRefusesOnlyArmourCells(t *testing.T) {
	t.Parallel()

	weapons, shields := wearWeapons(), wearShields()
	tbl := wearTable([]string{"Sword", "Buckler", "Arm3"}, weapons, shields, nil)
	w := wearWorld(t, tbl)

	eq, ok := w.Equipped(0)
	if !ok {
		t.Fatal("Equipped(0) answered not-ok")
	}
	wantWeapon, err := data.ResolveWeapon("Sword", tbl.Shapes, tbl.Materials, weapons)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	if eq[0] != uint16(wantWeapon.Code) {
		t.Errorf("slot 1 = 0x%04x, want 0x%04x — a missing Armors collection must not reach the weapon cell",
			eq[0], uint16(wantWeapon.Code))
	}
	wantShield, err := data.ResolveShield("Buckler", tbl.Shapes, tbl.Materials, shields)
	if err != nil {
		t.Fatalf("ResolveShield: %v", err)
	}
	if eq[1] != uint16(wantShield.Code) {
		t.Errorf("slot 2 = 0x%04x, want 0x%04x — nor the shield cell", eq[1], uint16(wantShield.Code))
	}
	if eq[2] != 0 {
		t.Errorf("slot 3 = 0x%04x, want empty — no Armors collection means no armour cell resolves", eq[2])
	}
	if carried, ok := w.Carried(0); !ok || len(carried) != 0 {
		t.Errorf("Carried(0) = %v, ok=%v, want empty — a refused cell is dropped, not carried", carried, ok)
	}
}

func TestATableMissingShieldsRefusesOnlyTheShieldCell(t *testing.T) {
	t.Parallel()

	weapons, armors := wearWeapons(), wearArmors()
	tbl := wearTable([]string{"Sword", "Buckler", "Arm3"}, weapons, nil, armors)
	w := wearWorld(t, tbl)

	eq, ok := w.Equipped(0)
	if !ok {
		t.Fatal("Equipped(0) answered not-ok")
	}
	wantWeapon, err := data.ResolveWeapon("Sword", tbl.Shapes, tbl.Materials, weapons)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	if eq[0] != uint16(wantWeapon.Code) {
		t.Errorf("slot 1 = 0x%04x, want 0x%04x — a missing Shields collection must not reach the weapon cell",
			eq[0], uint16(wantWeapon.Code))
	}
	if eq[1] != 0 {
		t.Errorf("slot 2 = 0x%04x, want empty — no Shields collection means the shield cell refuses", eq[1])
	}
	wantArm3, err := data.ResolveArmor("Arm3", tbl.Shapes, tbl.Materials, armors)
	if err != nil {
		t.Fatalf("ResolveArmor: %v", err)
	}
	if eq[2] != uint16(wantArm3.Code) {
		t.Errorf("slot 3 = 0x%04x, want 0x%04x — nor the armour cell", eq[2], uint16(wantArm3.Code))
	}
	if carried, ok := w.Carried(0); !ok || len(carried) != 0 {
		t.Errorf("Carried(0) = %v, ok=%v, want empty", carried, ok)
	}
}
