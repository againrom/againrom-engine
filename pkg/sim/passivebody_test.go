package sim

import (
	"bytes"
	"hash/fnv"
	"reflect"
	"testing"
)

func passiveBodyWorld(t *testing.T, stocked bool) *World {
	t.Helper()
	unplaced, placed := spEnt(3, 4, 4), spEnt(4, 6, 6)
	placed.MapUnitID = 24
	for _, e := range []*Entity{&unplaced, &placed} {
		e.HP, e.Decay = decayGoneHP-1, decayLast
	}
	var stock []Stock
	if stocked {
		stock = []Stock{
			{ID: 3, Items: []uint16{0x1234, 0x1235}, Equipped: [EquipSlots]uint16{0x2345, 0, 0, 0x2346}},
			{ID: 4, Equipped: [EquipSlots]uint16{0x2347}},
		}
	}
	w, err := NewStockedWorld(9, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, []Entity{unplaced, placed, spEnt(8, 1, 1)}, nil, Relations{}, nil, stock)
	if err != nil {
		t.Fatal(err)
	}
	toLadderSubTick(w)
	return w
}

// A departed actor's worn and carried items survive its removal, a cold LOAD
// of the byte form and stay out of the digest of play.
func TestPassiveBodyKeepsDepartedContents(t *testing.T) {
	w := passiveBodyWorld(t, true)
	Step(w, nil)
	if _, held := w.Entity(3); held || len(w.CurrentTerminalActors()) != 2 {
		t.Fatal("fixture did not remove both bodies")
	}
	got := w.PassiveBodies()
	if len(got) != 2 || got[0].ID != 3 || got[1].ID != 4 {
		t.Fatalf("passive bodies %+v", got)
	}
	unplaced, placed := got[0], got[1]
	if unplaced.Worn[0].Code != 0x2345 || unplaced.Worn[3].Code != 0x2346 || len(unplaced.Carried) != 2 ||
		unplaced.Carried[0].Code != 0x1234 || unplaced.Carried[1].Code != 0x1235 {
		t.Fatalf("unplaced contents %+v", unplaced)
	}
	if unplaced.Body == nil || unplaced.Body.ID != 3 || unplaced.Body.MapUnitID != 0 {
		t.Fatal("unplaced departed actor lost its body")
	}
	if placed.Worn[0].Code != 0x2347 || placed.Body != nil || len(placed.Carried) != 0 {
		t.Fatalf("placed contents %+v", placed)
	}
	got[0].Carried[0].Code = 1
	if w.PassiveBodies()[0].Carried[0].Code != 0x1234 {
		t.Fatal("getter aliases retained contents")
	}
	bare := *w
	bare.passiveBodies = nil
	plain, err := bare.MarshalBinary()
	if err != nil || bare.Hash() != fnvOf(plain) {
		t.Fatalf("a World without passive contents changed its digest: %x", bare.Hash())
	}
	form, err := w.MarshalBinary()
	if err != nil || w.Hash() != fnvOf(form) || w.Hash() == bare.Hash() {
		t.Fatalf("passive contents left the digest: %x / %x", bare.Hash(), w.Hash())
	}
	if form[0] != passiveBodyFormVersion || !bytes.Equal(form[len(form)-4:], []byte("PBC1")) || !bytes.HasPrefix(form[1:], plain[1:]) {
		t.Fatal("passive contents are not the outermost form")
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil || cold.Hash() != w.Hash() || !reflect.DeepEqual(cold.PassiveBodies(), w.PassiveBodies()) {
		t.Fatal("cold LOAD lost passive contents", err)
	}
	again, err := cold.MarshalBinary()
	if err != nil || !bytes.Equal(again, form) {
		t.Fatal("cold form is not canonical", err)
	}
	Step(&cold, nil)
	Step(w, nil)
	if cold.Hash() != w.Hash() || len(cold.PassiveBodies()) != 2 {
		t.Fatal("passive contents changed play after LOAD")
	}
}

func TestPassiveBodyRestoreBoundaries(t *testing.T) {
	w := passiveBodyWorld(t, true)
	Step(w, nil)
	rows := w.PassiveBodies()
	before, _ := w.MarshalBinary()
	stray := rows[1].Clone()
	stray.ID = 8
	if err := w.RestorePassiveBodies([]PassiveBody{stray}); err == nil {
		t.Fatal("live entity admitted as a passive body")
	}
	stray.ID = 40
	if err := w.RestorePassiveBodies([]PassiveBody{stray}); err == nil {
		t.Fatal("passive body without a terminal row admitted")
	}
	placedBody := rows[1].Clone()
	body := *rows[0].Body
	body.ID = 4
	placedBody.Body = &body
	if err := w.RestorePassiveBodies([]PassiveBody{placedBody}); err == nil {
		t.Fatal("body admitted for a row the writer rebuilds from its placement")
	}
	if after, _ := w.MarshalBinary(); !bytes.Equal(before, after) {
		t.Fatal("refused restore changed the World")
	}
	replaced := rows[1].Clone()
	replaced.Worn[0].ObjectID = 77
	replaced.Worn[2] = ItemInstance{Code: 0x2348}
	if err := w.RestorePassiveBodies([]PassiveBody{replaced}); err != nil {
		t.Fatal(err)
	}
	got := w.PassiveBodies()
	if len(got) != 2 || got[1].Worn[0].ObjectID != 0 || got[1].Worn[2].Code != 0x2348 || got[0].Worn[0].Code != 0x2345 {
		t.Fatalf("restore did not replace exactly one row: %+v", got)
	}
	if err := w.RestorePassiveBodies([]PassiveBody{{ID: 4}}); err != nil || len(w.PassiveBodies()) != 1 || w.PassiveBodies()[0].ID != 3 {
		t.Fatal("a row without items did not remove exactly its own row", err)
	}
}

func TestPassiveBodyFollowsActorIdentity(t *testing.T) {
	w := passiveBodyWorld(t, true)
	Step(w, nil)
	ids := map[EntityID]EntityID{}
	for _, id := range w.ActorIdentityReferences() {
		ids[id] = id + 100
	}
	if err := w.RestoreActorIdentities(ids, nil); err != nil {
		t.Fatal(err)
	}
	got := w.PassiveBodies()
	if len(got) != 2 || got[0].ID != 103 || got[0].Body == nil || got[0].Body.ID != 103 || got[1].ID != 104 {
		t.Fatalf("passive bodies kept old identities: %+v", got)
	}
	if _, err := w.MarshalBinary(); err != nil {
		t.Fatal(err)
	}
}

// One passive body item moves the digest, beyond SAV bytes.
func TestPassiveBodyItemMovesHash(t *testing.T) {
	a, b := passiveBodyWorld(t, true), passiveBodyWorld(t, true)
	Step(a, nil)
	Step(b, nil)
	if a.Hash() != b.Hash() {
		t.Fatal("fixture Worlds differ before the item change")
	}
	rows := b.PassiveBodies()
	rows[1].Worn[0].Code = 0x2349
	if err := b.RestorePassiveBodies(rows[1:]); err != nil {
		t.Fatal(err)
	}
	pa, pb := *a, *b
	pa.passiveBodies, pb.passiveBodies = nil, nil
	if !bytes.Equal(mustMarshal(t, &pa), mustMarshal(t, &pb)) {
		t.Fatal("Worlds differ outside the passive contents")
	}
	if a.Hash() == b.Hash() {
		t.Fatalf("one passive item left the digest unchanged: %x", a.Hash())
	}
}

func fnvOf(b []byte) uint64 {
	h := fnv.New64a()
	h.Write(b)
	return h.Sum64()
}

// A kept item takes the form its SAV record gives back: a stated weight, no
// weapon definition, and a stack no wider than one record's count.
func TestPassiveBodyItemsTakeRecordForm(t *testing.T) {
	w := passiveBodyWorld(t, false)
	weapon := PlainItem(0x2345)
	weapon.SourceEquipment.Class = SourceWeapon
	weapon.SourceEquipment.Definition = SourceWeaponDefinition{Present: true, AttackType: 5, Hands: -1}
	weapon.SourceEquipment.EffectsUnsupported = true
	got := w.passiveItem(weapon)
	if !got.WeightPresent || got.SourceEquipment.Definition != (SourceWeaponDefinition{}) || got.SourceEquipment.EffectsUnsupported || !passiveItemForm(got) {
		t.Fatalf("passive weapon %+v", got)
	}
	if again := w.passiveItem(got); !reflect.DeepEqual(again, got) {
		t.Fatal("passive item form is not a fixed point")
	}
	stacks := w.passiveStacks([]ItemStack{PlainStack(0x1234, 70000), PlainStack(0x1235, 2)})
	if len(stacks) != 3 || stacks[0].Count != 65535 || stacks[1].Count != 4465 || stacks[2].Count != 2 {
		t.Fatalf("passive stacks %+v", stacks)
	}
}
