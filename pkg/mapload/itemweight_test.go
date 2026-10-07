package mapload_test

// The four doors a code reaches a world through, and the weight table each
// of them has to fill.
//
// declareItemWeights' own doc block enumerates the population: equipment slots,
// containers, ground sacks and the compiled script's item literals. This file
// witnesses one door per test, so a door that stops being walked fails on its
// own name rather than inside a total.

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// cwWeightColumn is runtime column 3 of an Armors, Shields or Weapons row,
// transcribed here rather than imported for this suite's standing reason: a
// test asserting that production reads column 3 must not read the constant
// that says 3.
const cwWeightColumn = 3

// cwWeaponRow is human_test.go's weaponRow with the weight column filled. The
// base row leaves every unread cell at -1, so a build reading a neighbouring
// column answers -1 rather than this number.
func cwWeaponRow(weight int32) []int32 {
	p := weaponRow(data.SkillBlade, 5, 9, 7, 2, 15, 9)
	p[cwWeightColumn] = weight
	return p
}

// cwArmorAbsorptionColumn is the last Armors column ArmorFromCode reads.
const cwArmorAbsorptionColumn = 10

// cwArmorRow is an Armors row carrying its own slot cell and the weight column,
// wide enough for the code path to read it.
func cwArmorRow(slot, weight int32) []int32 {
	p := make([]int32, cwArmorAbsorptionColumn+1)
	for i := range p {
		p[i] = -1
	}
	p[slotArmorSlot] = slot
	p[cwWeightColumn] = weight
	return p
}

// cwTable resolves type id 7 to a person wearing the named equipment, over a
// Weapons collection holding one 30-unit sword and an Armors collection holding
// one 12-unit helmet in slot 1.
//
// The scale tables are identityScale, whose every factor is 1, so a row's own
// column is its weight and this file states one number per item rather than a
// product. The product itself is pkg/data's to witness and is pinned there.
func cwTable(equipment []string) *mapload.Table {
	return &mapload.Table{
		Units:  defCollection{{}},
		Humans: defCollection{{}, {name: "k7", params: fullHumanRow(), strings: equipment}},
		Shapes: identityScale(), Materials: identityScale(),
		Weapons: defCollection{{}, {name: "Sword", params: cwWeaponRow(30)}},
		Armors:  defCollection{{}, {name: "Helm", params: cwArmorRow(4, 12)}},
	}
}

// cwWeightOf reads one code's declared weight off the world's own table, and
// reports whether the table names it at all.
func cwWeightOf(w *sim.World, code uint16) (int32, bool) {
	for _, e := range w.ItemWeights() {
		if e.Code == code {
			return e.Weight, true
		}
	}
	return 0, false
}

// TestAWornItemIsDeclaredAndCountsTowardTheLoad is the equipment-slot door,
// end to end through the map loader.
//
// The person's row leaves his container empty, so his load is the worn sum
// alone: 30 for the sword plus 12 for the helm, both at factor 1, is 42. His
// capacity is body x 10 + 1 over the fixture's own body of 40, which is 401.
func TestAWornItemIsDeclaredAndCountsTowardTheLoad(t *testing.T) {
	w, err := mapload.FromALMWith(humanMap(), cwTable([]string{"Sword", "", "Helm"}), mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	ents := w.Entities()
	if len(ents) != 1 {
		t.Fatalf("the fixture built %d entities, want 1", len(ents))
	}
	e := ents[0]

	worn, ok := w.Equipped(e.ID)
	if !ok {
		t.Fatal("Equipped: no entry for the one entity this fixture built")
	}
	var declared int
	for _, code := range worn {
		if code == 0 {
			continue
		}
		if _, named := cwWeightOf(w, code); !named {
			t.Errorf("worn code 0x%04x is not in the world's weight table", code)
			continue
		}
		declared++
	}
	if declared != 2 {
		t.Fatalf("the fixture armed %d slots, want 2", declared)
	}

	if e.Load != 42 {
		t.Errorf("the load is %d, want 42 — 30 for the sword and 12 for the helm", e.Load)
	}
	if e.Capacity != 401 {
		t.Errorf("the capacity is %d, want 401 — body 40 x 10 + 1", e.Capacity)
	}
}

// TestASackOnTheGroundIsDeclared is the sack door. A ground loot record's codes
// belong to nobody's load, so nothing about the entity moves — what has to move
// is the table, because the item can be picked up later and no producer runs
// again when it is.
//
// The record is assembled from the documented type-8 byte layout, the same
// grammar loot_test.go's own fixtures use.
func TestASackOnTheGroundIsDeclared(t *testing.T) {
	sword := data.ComposeItemCode(0, 1, 0, 1) // material 0, weapon class, shape 0, row 1
	body := cwLootRecord(1, 0, 3, 4, 0, uint32(sword))
	m := &alm.Map{Width: 40, Height: 40, FormatVersion: 990,
		Meta:        alm.Meta{Word2C: 1},
		LootSection: alm.LootSection{Body: body},
		Units:       []alm.Unit{{X: 0x0C80, Y: 0x0C80, ClassID: 7}}}

	w, err := mapload.FromALMWith(m, cwTable(nil), mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	if len(w.Sacks()) != 1 {
		t.Fatalf("the fixture placed %d sacks, want 1", len(w.Sacks()))
	}
	got, named := cwWeightOf(w, uint16(sword))
	if !named {
		t.Fatalf("the sack's code 0x%04x is not in the world's weight table", uint16(sword))
	}
	if got != 30 {
		t.Errorf("the sack's sword is declared at %d, want 30", got)
	}
}

// TestAScriptsOwnItemLiteralIsDeclared is the script door, and it is the one
// door whose codes need not appear anywhere in the world at construction: an
// add-item instant names a code as a literal and nothing holds it until the
// instant fires.
//
// It goes through StartMissionScripted because that is the entry point every
// mission comes through and the only one that carries a compiled script.
func TestAScriptsOwnItemLiteralIsDeclared(t *testing.T) {
	helm := data.ComposeItemCode(0, 1, 0, 1) // the Weapons row this table holds
	s, err := sim.NewScript(nil, []sim.ScriptInstant{{Item: uint16(helm), HasItem: true}}, nil)
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}

	w, _, err := mapload.StartMissionScripted(humanMap(), cwTable(nil), mapload.DifficultyNormal, nil, s)
	if err != nil {
		t.Fatalf("StartMissionScripted: %v", err)
	}
	// Nothing in the world holds the code: the one entity wears nothing and
	// carries nothing, and there are no sacks.
	for _, e := range w.Entities() {
		if held, ok := w.CarriedStacks(e.ID); ok && len(held) != 0 {
			t.Fatalf("entity %d carries %v; this fixture must reach the table through the script alone", e.ID, held)
		}
	}
	if got, named := cwWeightOf(w, uint16(helm)); !named || got != 30 {
		t.Errorf("the script's own item literal is declared as (%d, %v), want (30, true)", got, named)
	}
}

// TestATableMissingACollectionStillBuildsTheWorld is the partial-install arm at
// the loader. A table with no Armors must still load a map, still declare the
// classes it can read, and simply state no weight for the class it cannot.
func TestATableMissingACollectionStillBuildsTheWorld(t *testing.T) {
	tbl := cwTable([]string{"Sword", "", "Helm"})
	tbl.Armors = nil

	w, err := mapload.FromALMWith(humanMap(), tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	ents := w.Entities()
	if len(ents) != 1 {
		t.Fatalf("the fixture built %d entities, want 1", len(ents))
	}
	worn, ok := w.Equipped(ents[0].ID)
	if !ok {
		t.Fatal("Equipped: no entry for the one entity this fixture built")
	}
	var armed int
	for _, code := range worn {
		if code != 0 {
			armed++
		}
	}
	if armed != 1 {
		t.Fatalf("the fixture armed %d slots, want 1: the weapon alone", armed)
	}
	if ents[0].Load != 30 {
		t.Errorf("the load is %d, want 30 — the sword alone, the helm never resolving", ents[0].Load)
	}
}

// TestDeclareCodeWeightsIsTheDoorForAnOutsideCaller is the fifth producer, the
// one declareItemWeights cannot see: a caller replacing an actor's whole
// holdings from outside the mission hands its own codes over.
func TestDeclareCodeWeightsIsTheDoorForAnOutsideCaller(t *testing.T) {
	tbl := cwTable(nil)
	w, err := mapload.FromALMWith(humanMap(), tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	sword := uint16(data.ComposeItemCode(0, 1, 0, 1))
	if _, named := cwWeightOf(w, sword); named {
		t.Fatalf("code 0x%04x is declared before anything named it", sword)
	}

	mapload.DeclareCodeWeights(w, tbl, []uint16{sword, sword, 0})
	got, named := cwWeightOf(w, sword)
	if !named || got != 30 {
		t.Errorf("after the declaration the code reads (%d, %v), want (30, true)", got, named)
	}
	if len(w.ItemWeights()) != 1 {
		t.Errorf("the table holds %d entries, want 1: the duplicate and the zero are dropped",
			len(w.ItemWeights()))
	}

	// And the load follows, through the same replace door a campaign carry uses.
	ents := w.Entities()
	if !w.ReplaceStock(sim.Stock{ID: ents[0].ID, Items: []uint16{sword, sword}}) {
		t.Fatal("ReplaceStock refused the world's own entity")
	}
	for _, e := range w.Entities() {
		if e.ID == ents[0].ID && e.Load != 30 {
			t.Errorf("carrying two swords gives load %d, want 30 — 60 halved", e.Load)
		}
	}
}

// cwLootRecord builds one wide type-8 record: its element count, owner, cell
// pair and gold, followed by count ten-byte elements naming code.
func cwLootRecord(n, owner uint32, x, y int32, gold uint32, code uint32) []byte {
	le32 := func(v uint32) []byte {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, v)
		return b
	}
	out := append([]byte(nil), le32(n)...)
	out = append(out, le32(owner)...)
	out = append(out, le32(uint32(x)<<8)...)
	out = append(out, le32(uint32(y)<<8)...)
	out = append(out, le32(gold)...)
	for i := uint32(0); i < n; i++ {
		out = append(out, le32(code)...)
		out = append(out, 0, 0)
		out = append(out, le32(0)...)
	}
	return out
}

func TestPartyLoadIsTheSameLawOutsideAMission(t *testing.T) {
	tbl := cwTable(nil)
	sword := uint16(data.ComposeItemCode(0, 1, 0, 1))

	var p PartyMemberFixture
	p.Worn[0] = sword
	p.Carried = []uint16{sword, sword}
	if got := mapload.PartyLoad(p.member(), tbl); got != 60 {
		t.Errorf("the member's load is %d, want 60 — 30 worn in full and 60 carried halved", got)
	}

	// The saturation is the same one, reached through the same statement of
	// the law: a container at or past 64000 flattens the load to 32000 and
	// discards the worn term.
	heavy := p
	heavy.Carried = nil
	for i := 0; i < 64000/30+1; i++ {
		heavy.Carried = append(heavy.Carried, sword)
	}
	if got := mapload.PartyLoad(heavy.member(), tbl); got != 32000 {
		t.Errorf("a saturated member's load is %d, want 32000", got)
	}

	// A member whose table resolves nothing carries nothing, which is a real
	// load and not a refusal.
	if got := mapload.PartyLoad(p.member(), &mapload.Table{}); got != 0 {
		t.Errorf("over an empty table the load is %d, want 0", got)
	}
}

// PartyMemberFixture is the two arrays PartyLoad reads, so this file states
// them without restating every other field of a party member.
type PartyMemberFixture struct {
	Worn    [sim.EquipSlots]uint16
	Carried []uint16
}

func (f PartyMemberFixture) member() mapload.PartyMember {
	return mapload.PartyMember{Worn: f.Worn, Carried: f.Carried}
}
