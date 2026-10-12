package sim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func moverWorld(t *testing.T) *World {
	t.Helper()
	return mustWorld(t, 3, Bounds{Width: 24, Height: 24}, []Entity{
		witnessUnit(1, 3, 4, DomainGround, 2),
		witnessUnit(2, 8, 8, DomainGround, SelfSlot),
		witnessUnit(3, 12, 5, DomainAir, 2),
	})
}

// The builder holds the constructor's bytes, a placed actor's facing, rate,
// mask and occupy stores, and a carried member the constructor's alone
// (MOVE-119..MOVE-122).
func TestConstructedMoverHoldsTheConstructorAndTheOccupy(t *testing.T) {
	w := moverWorld(t)
	ground, _ := w.Entity(1)
	ground.Facing = 0x77
	m := w.constructedMover(ground, false)
	cost := int32(w.costAt(cell{3, 4}))
	if cost == 0 {
		t.Fatal("fixture cell has no cost byte")
	}
	want := uint16(int32(int16(ground.SpeedWord())) * 8 / cost)
	if m[0] != 0x77 || m[1] != 0x77 || m[5] != 0x41 || m[8] != 5 || m[9] != 0xff || m[10] != moverRate(ground) {
		t.Fatalf("placed constructor bytes % x", m[:11])
	}
	if got := binary.LittleEndian.Uint16(m[0x72:]); got != want || m[0x82] != 3 || m[0x83] != 4 || m[0x84] != 0x80 || m[0x85] != 0x80 {
		t.Fatalf("occupy stores %#x % x, want %#x", got, m[0x82:0x86], want)
	}
	air, _ := w.Entity(3)
	if m := w.constructedMover(air, false); m[5] != 0x82 || binary.LittleEndian.Uint16(m[0x72:]) != uint16(air.SpeedWord()) {
		t.Fatalf("air mover mask %#x cost %#x", m[5], binary.LittleEndian.Uint16(m[0x72:]))
	}
	joined := w.constructedMover(ground, true)
	var want0 [180]byte
	want0[5], want0[8], want0[9], want0[10] = 0x41, 5, 0xff, 0x10
	if joined != want0 {
		t.Fatal("a carried member holds more than the constructor's bytes")
	}
}

// A World whose records are as built writes no record form; a carried
// member's record is written and read back, and the World hash survives.
// Loss control: the form without its records restores a different World.
func TestMoverRecordsSurviveTheByteForm(t *testing.T) {
	w := moverWorld(t)
	form, err := w.MarshalBinary()
	if err != nil || form[0] == moverFormVersion {
		t.Fatal("an as-built World wrote a record form", err)
	}
	w.ConstructMissionMovers([]EntityID{2})
	if r := w.moverFor(2); r == nil || !r.Joined || r.Mask != 0x41 {
		t.Fatalf("carried member's record %+v", r)
	}
	if r := w.moverFor(1); r == nil || r.Joined || r.Mask != 0x41 {
		t.Fatalf("placed actor's record %+v", r)
	}
	form, err = w.MarshalBinary()
	if err != nil || form[0] != moverFormVersion {
		t.Fatal("a carried member's record wrote no form", err)
	}
	if err := CheckSaveForm(form); err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	again, _ := back.MarshalBinary()
	if !bytes.Equal(form, again) || back.Hash() != w.Hash() {
		t.Fatal("the record form does not round-trip")
	}
	var mover [180]byte
	if !back.ApplyMover(2, &mover) || binary.LittleEndian.Uint16(mover[0x72:]) != 0 {
		t.Fatal("the restored carried member lost the join walk's zero step cost")
	}
	span := int(binary.LittleEndian.Uint32(form[len(form)-9:]))
	stripped := bytes.Clone(form[:len(form)-9-span])
	stripped[0] = form[len(form)-5]
	var loss World
	if err := loss.UnmarshalBinary(stripped); err != nil {
		t.Fatal(err)
	}
	if loss.Hash() == w.Hash() {
		t.Fatal("loss control: the World without its records hashed equal")
	}
}

// The first crossing stores the step cost of the cell it enters (MOVE-128),
// so a carried member that has moved writes the cost of where it stands.
func TestACrossingEndsTheJoinWalksZeroCost(t *testing.T) {
	w := moverWorld(t)
	w.ConstructMissionMovers([]EntityID{2})
	Step(w, []Command{MoveTo(2, CellPoint{X: 12, Y: 8})})
	for i := 0; i < 400 && w.moverFor(2).Joined; i++ {
		Step(w, nil)
	}
	if w.moverFor(2).Joined {
		t.Fatal("the carried member never crossed")
	}
	e, _ := w.Entity(2)
	var mover [180]byte
	w.ApplyMover(2, &mover)
	if got, want := binary.LittleEndian.Uint16(mover[0x72:]), w.stepCostWord(e, moverCell(e)); got != want || want == 0 {
		t.Fatalf("step cost %#x, want %#x", got, want)
	}
}

// A removed actor's record goes with it.
func TestARemovedActorDropsItsMoverRecord(t *testing.T) {
	w := moverWorld(t)
	if !w.remove([]EntityID{3}) || w.moverFor(3) != nil || w.moverFor(1) == nil {
		t.Fatal("the record outlived its actor")
	}
}
