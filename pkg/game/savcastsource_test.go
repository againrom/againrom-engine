package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
)

func castSourceActor(state, phase uint32, progress byte, spell uint32, borrow uint16) sav.DocumentRecordData {
	order := make([]byte, 148)
	order[8], order[9] = 0xb, progress
	return sav.DocumentRecordData{
		Class:  "Human",
		Values: []sav.DocumentValueData{{Name: "U64", Value: spell}, {Name: "U6C", Value: 9}},
		Raw: []sav.DocumentRawData{
			{Name: "U54", Bytes: binary.LittleEndian.AppendUint32(nil, state)},
			{Name: "U58", Bytes: binary.LittleEndian.AppendUint32(nil, phase)},
			{Name: "U158", Bytes: order},
		},
		RefSlots: []sav.DocumentRefsData{{Name: "U68", Objects: []uint16{borrow}}},
	}
}

func castSourceDocument() sav.DocumentData {
	return sav.DocumentData{Objects: []sav.DocumentRecordData{
		castSourceActor(0, 0, 2, 0, 0),      // 1: stale progress 2
		castSourceActor(0xd, 5, 2, 0, 0),    // 2: book state, no Spell
		castSourceActor(0xd, 5, 1, 0, 7),    // 3: Weapon borrow, no U64
		castSourceActor(0xd, 7, 0, 0, 0),    // 4: weapon cast recovery
		castSourceActor(0xd, 5, 2, 0x55, 0), // 5: book cast with its Spell
		castSourceActor(0xd, 5, 2, 0, 9),    // 6: scroll reservation
		{Class: "Weapon", RefSlots: []sav.DocumentRefsData{{Name: "WeaponSpell", Objects: []uint16{8}}}},
		{Class: "Spell", Values: []sav.DocumentValueData{{Name: "This", Value: 0x77}}},
		{Class: "Scroll"},
		castSourceActor(3, 5, 1, 0, 11), // 10: Weapon without a WeaponSpell
		{Class: "Weapon"},
		castSourceActor(0xd, 0, 1, 0, 0), // 12: cast state at progress 1
	}}
}

// Every actor the pass leaves at progress 2 has its Spell or is idle; an
// actor with a source, in a weapon cast's recovery, or at another progress
// keeps its words.
func TestSealCastSourcesWritesNoCastWithoutItsSpell(t *testing.T) {
	if len(castsWithoutSource(castSourceDocument())) != 4 {
		t.Fatal("control: the unsealed document does not hold its four casts without a Spell")
	}
	doc := castSourceDocument()
	sealCastSources(&doc)
	if bad := castsWithoutSource(doc); len(bad) != 0 {
		t.Fatal(bad)
	}
	words := func(object int) (uint32, uint32, byte, uint32, uint32, uint16) {
		r := &doc.Objects[object-1]
		state, _ := savedMotionRaw(r, "U54", 4)
		phase, _ := savedMotionRaw(r, "U58", 4)
		order, _ := savedMotionRaw(r, "U158", 148)
		spell, _ := savedStructureValue(r, "U64")
		countdown, _ := savedStructureValue(r, "U6C")
		refs, _ := savedObjectRefs(r, "U68")
		return binary.LittleEndian.Uint32(state), binary.LittleEndian.Uint32(phase), order[9], spell, countdown, refs[0]
	}
	type w struct {
		state, phase uint32
		progress     byte
		spell, cd    uint32
		borrow       uint16
	}
	for object, want := range map[int]w{
		1:  {0, 0, 0, 0, 9, 0},
		2:  {0, 0, 0, 0, 0, 0},
		3:  {0xd, 5, 1, 0x77, 9, 7},
		4:  {0xd, 7, 0, 0, 9, 0},
		5:  {0xd, 5, 2, 0x55, 9, 0},
		6:  {0xd, 5, 2, 0, 9, 9},
		10: {3, 5, 1, 0, 9, 0},
		12: {0xd, 0, 1, 0, 9, 0},
	} {
		state, phase, progress, spell, cd, borrow := words(object)
		if got := (w{state, phase, progress, spell, cd, borrow}); got != want {
			t.Errorf("object %d: %+v, want %+v", object, got, want)
		}
	}
}

func TestSavedWeaponCastBorrowReadsAReleaseCharge(t *testing.T) {
	doc := sav.DocumentData{Objects: []sav.DocumentRecordData{
		castSourceActor(3, 5, 1, 0x77, 2),
		{Class: "Weapon", RefSlots: []sav.DocumentRefsData{{Name: "WeaponSpell", Objects: []uint16{3}}}},
		{Class: "Spell", Values: []sav.DocumentValueData{{Name: "This", Value: 0x77}}},
		castSourceActor(3, 5, 1, 0, 2),
		castSourceActor(3, 5, 1, 0x77, 6),
		{Class: "Weapon"},
		castSourceActor(3, 5, 1, 0x77, 0),
	}}
	for object, want := range map[int]bool{1: true, 4: false, 5: false, 7: false} {
		if got := savedWeaponCastBorrow(&doc, &doc.Objects[object-1]); got != want {
			t.Errorf("object %d: borrow %v, want %v", object, got, want)
		}
	}
}
