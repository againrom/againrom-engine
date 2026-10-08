package mapload_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func initialModifierTable(attackType int32, humanItems []string) *mapload.Table {
	return &mapload.Table{
		Units: defCollection{{}, {
			name:    "history-unit",
			params:  defRow(map[int]int32{0: 37, 4: 99, 11: 9, 12: 21, 14: 17, 15: 4, 29: 200, 30: 1}),
			strings: []string{"HistoryWeapon"},
		}},
		Humans: defCollection{{}, {
			name: "history-human",
			params: defRow(map[int]int32{
				0: 43, 1: 31, 2: 29, 3: 19, 4: 100, 5: 0,
				10: 61, 11: 11, 12: 22, 13: 33, 14: 44, 15: 55, 16: 7, 17: 1,
			}),
			strings: humanItems,
		}},
		Shapes: identityScale(), Materials: identityScale(),
		Weapons: defCollection{{}, {
			name:   "HistoryWeapon",
			params: []int32{0, 0, 100, 2, 0, attackType, 5, 14, 11, 7, 0, 4, 8, 4, 1, 1, 0},
		}},
	}
}

func initialModifierMap(class int16) *alm.Map {
	return &alm.Map{Width: 40, Height: 40, Units: []alm.Unit{{X: 0x1480, Y: 0x1480, ClassID: class, ClassSubID: 1}}}
}

func assertInitialModifier(t *testing.T, e sim.Entity, want [64]byte) {
	t.Helper()
	b := e.NativeBasis
	if !b.ModifierPresent {
		t.Error("initial Modifier is absent")
	}
	if b.ModifierKnown != ^uint64(0) {
		t.Errorf("ModifierKnown = %016x, want ffffffffffffffff", b.ModifierKnown)
	}
	for i, value := range want {
		if !b.ModifierByteKnown(i) || b.Modifier[i] != value {
			t.Errorf("Modifier[%d] = %02x known:%v, want %02x known:true", i, b.Modifier[i], b.ModifierByteKnown(i), value)
		}
	}
}

func TestResolvedUnitInitialModifierKeepsIndependentRangedGeneral(t *testing.T) {
	tbl := initialModifierTable(11, nil)
	w, err := mapload.FromALMWith(initialModifierMap(200), tbl, mapload.DifficultyHard)
	if err != nil {
		t.Fatal(err)
	}
	e := w.Entities()[0]
	if e.Skill[0] != 17 || e.ToHit != 84 {
		t.Fatalf("fixture lost pre-placement General17 or Hard live ToHit84: %+v", e)
	}
	worn, ok := w.EquippedItems(e.ID)
	q := worn[0].SourceEquipment
	if !ok || q.Class != sim.SourceWeapon || !q.Definition.Present || q.Definition.AttackType != 11 || q.Attack[0] != 11 || q.Attack[14] != 5 || q.Attack[15] != 9 {
		t.Fatalf("fixture lacks independently supplied ranged operands: %+v", q)
	}
	// Initial ranged attachment assigns General17, not weapon ToHit11,
	// current live ToHit84, or an inverse of the effective combat block.
	assertInitialModifier(t, e, [64]byte{18: 17, 37: 5, 38: 9, 39: 1})
	sim.Step(w, []sim.Command{sim.Unequip(e.ID, 1)})
	if worn, _ := w.EquippedItems(e.ID); !worn[0].Empty() {
		t.Fatal("real Unequip did not remove the weapon")
	}
	e, _ = w.Entity(e.ID)
	// The later event subtracts typed Weapon ToHit11: 17 - 11 = 6.
	assertInitialModifier(t, e, [64]byte{18: 6})
}

func TestResolvedHumanInitialModifierFollowsOrderedTypedEvents(t *testing.T) {
	tbl := initialModifierTable(1, []string{
		"HistoryWeapon{body=5,fighterskill0=100,mageskill0=200,skillblade=7,damagefire=7-9,damagewater=3-8}",
	})
	w, err := mapload.FromALMWith(initialModifierMap(7), tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	e := w.Entities()[0]
	if !e.Humanoid || !e.NativeClass.Present || !e.NativeClass.Fighter {
		t.Fatalf("fixture lacks the explicit fresh Human class gate: %+v", e)
	}
	// Zero64 precedes the direct melee +11/+5/+9/+7 stores. Fighter
	// General +100 enters raw word20; the mage +200 arm is not selected.
	// The final ordered Effect45 assigns3/5/2 over the weapon and Effect44.
	assertInitialModifier(t, e, [64]byte{0: 5, 18: 11, 20: 100, 22: 7, 32: 5, 33: 9, 37: 3, 38: 5, 39: 2, 42: 7})
	if e.Skill[0] != 161 || e.Skill[1] != 18 {
		t.Fatalf("fixture did not apply the independently authored initial effects: %+v", e)
	}
}

func TestStartedPartyInitialModifierAndRemovalKeepEventOrder(t *testing.T) {
	tbl := initialModifierTable(11, nil)
	weapon, err := data.ResolveWeapon("HistoryWeapon", tbl.Shapes, tbl.Materials, tbl.Weapons)
	if err != nil {
		t.Fatal(err)
	}
	p := mapload.PartyMember{
		Class:   3,
		Hero:    data.Hero{Body: 39, Reaction: 31, Mind: 29, Spirit: 19, Skill: [6]int32{79, 13, 23, 31, 41, 53}},
		Profile: data.Profile{Fighter: true, HealthColumn: true}, Weapon: &weapon,
	}
	p.WornItems[0] = mapload.SourceConstructedItem(sim.PlainItem(0x0101), tbl)
	p.WornItems[0].Effects = []sim.ItemEffect{{Kind: 12, Operand: 3}, {Kind: 44, Operand: 7 | 2<<8}, {Kind: 45, Operand: 3 | 5<<8}}
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	w, st, err := mapload.StartMissionScripted(m, tbl, mapload.DifficultyNormal, []mapload.PartyMember{p}, nil)
	if err != nil {
		t.Fatal(err)
	}
	e, ok := w.Entity(st.IDs[0])
	if !ok || e.Skill[0] != 79 {
		t.Fatalf("party constructor lost the authored General79: %+v", e)
	}
	assertInitialModifier(t, e, [64]byte{18: 82, 37: 3, 38: 5, 39: 2})
	sim.Step(w, []sim.Command{sim.Unequip(e.ID, 1)})
	if worn, _ := w.EquippedItems(e.ID); !worn[0].Empty() {
		t.Fatal("real Unequip did not remove the weapon")
	}
	e, _ = w.Entity(e.ID)
	// Removal Effects leave18=79 and assign3/5/2. The following direct
	// ranged removal subtracts11/5/9 and clears39, leaving68/254/252/0.
	assertInitialModifier(t, e, [64]byte{18: 68, 37: 254, 38: 252})
}

func TestStartedCodeOnlyCarryKeepsCurrentKindDuringNativeEnrichment(t *testing.T) {
	tbl := initialModifierTable(1, nil)
	p := mapload.PartyMember{
		Class:   3,
		Hero:    data.Hero{Body: 39, Reaction: 31, Mind: 29, Spirit: 19, Skill: [6]int32{79, 13, 23, 31, 41, 53}},
		Profile: data.Profile{Fighter: true, HealthColumn: true},
		Carry:   &mapload.Carry{Equipped: [sim.EquipSlots]uint16{0x0101}},
	}
	w, st, err := mapload.StartMissionScripted(startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20}), tbl, mapload.DifficultyNormal, []mapload.PartyMember{p}, nil)
	if err != nil {
		t.Fatal(err)
	}
	worn, ok := w.EquippedItems(st.IDs[0])
	if !ok || worn[0].Kind != 0 || !worn[0].WeightPresent || worn[0].SourceEquipment.Class != sim.SourceWeapon || worn[0].SourceEquipment.DefinitionRow != 1 {
		t.Fatal("native constructor changed current Kind or lost class/weight enrichment", worn[0])
	}
	if !p.Carry.EquippedItems[0].Empty() {
		t.Fatal("mission construction changed caller-owned legacy Carry")
	}
	e, _ := w.Entity(st.IDs[0])
	assertInitialModifier(t, e, [64]byte{18: 11, 32: 5, 33: 9, 42: 7})
	raw, err := w.MarshalBinary()
	var cold sim.World
	if err != nil || cold.UnmarshalBinary(raw) != nil || cold.Hash() != w.Hash() {
		t.Fatal("enriched current Kind did not survive cold native LOAD", err)
	}
}

func TestInitialModifierGenericRebuildKeepsIndependentHistory(t *testing.T) {
	w, err := mapload.FromALMWith(initialModifierMap(200), initialModifierTable(11, nil), mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	// Known word18=0x1234 and unknown retained byte40=0xa5 are current
	// admitted history. A generic rebuild must neither zero nor promote it.
	b := sim.NativeActorBasis{ModifierPresent: true, ModifierKnown: ^uint64(0) &^ (uint64(1) << 40)}
	b.Modifier[18], b.Modifier[19], b.Modifier[40] = 0x34, 0x12, 0xa5
	if err := w.RestoreNativeActorBases([]sim.NativeActorBasisRecord{{ID: 0, Basis: b}}); err != nil {
		t.Fatal(err)
	}
	rebuilt, err := sim.NewWorld(mapload.Seed, w.Bounds(), sim.ModeCanonical, nil, w.Entities())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := rebuilt.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var restored sim.World
	if err := restored.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if got := restored.Entities()[0].NativeBasis; got != b {
		t.Fatalf("generic rebuild/codec changed admitted Modifier history: %+v, want %+v", got, b)
	}
}

func TestInitialModifierMissingTypedWeaponKeepsDependentBytesUnavailable(t *testing.T) {
	p := mapload.PartyMember{Hero: data.Hero{Skill: [6]int32{79}}, Profile: data.Profile{Fighter: true}}
	p.WornItems[0] = sim.PlainItem(0x0101)
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	w, st, err := mapload.StartMission(m, nil, mapload.DifficultyNormal, []mapload.PartyMember{p})
	if err != nil {
		t.Fatal(err)
	}
	e, ok := w.Entity(st.IDs[0])
	if !ok || !e.NativeBasis.ModifierPresent {
		t.Fatal("constructor Modifier absent", e.NativeBasis)
	}
	want := ^uint64(0)
	for _, i := range []int{18, 19, 32, 33, 37, 38, 39, 42, 43} {
		want &^= uint64(1) << i
	}
	if e.NativeBasis.ModifierKnown != want || e.NativeBasis.Modifier != ([64]byte{}) {
		t.Fatal("appearance-only weapon invented raw operands or erased unrelated zero history", e.NativeBasis)
	}
}
