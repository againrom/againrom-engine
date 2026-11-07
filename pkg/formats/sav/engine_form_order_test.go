package sav

import (
	"encoding/binary"
	"slices"
	"testing"
)

// attackDocument is an engine document in the first game's records with
// one Player and two Humans in its group, identities 0x20 and 0x30. The
// first Human is under engine order code, in state 3 and phase 5, with key
// at command +0x0c, prepared byte 5 and range 6.
func attackDocument(t *testing.T, code, key uint32) DocumentData {
	t.Helper()
	const owner = uint32(0x10)
	fixed := make([]byte, 51)
	binary.LittleEndian.PutUint16(fixed[0:2], 1)
	binary.LittleEndian.PutUint32(fixed[2:6], 1)
	binary.LittleEndian.PutUint32(fixed[43:47], 0x20)
	binary.LittleEndian.PutUint32(fixed[47:51], owner)
	human := func(identity uint32, name string) *CityUnitData {
		return &CityUnitData{
			Token: cityTestToken(identity, owner, 0), RawA6: make([]byte, 24), RawBE: make([]byte, 22),
			Raw114: make([]byte, 24), RawD4: make([]byte, 64), Raw154: make([]byte, 180), Raw158: make([]byte, 148),
			Scalar1: make([]byte, 19), Name: name, Scalar2: make([]byte, 55),
			ScalarTail: make([]byte, 17), XP: make([]byte, 24), Equipment: make([]uint16, 13),
		}
	}
	city, err := CityFromData(CityData{
		Version: CityDataVersion, FileVersion: MinVersion, Head: [13]uint32{12: 2},
		PlayerList: 1, Players: []uint16{1}, Marker: GeneratedCityMarker,
		TrailerState: make([]byte, generatedCityTrailerLen),
		Objects: []CityObjectData{
			{Class: "Player", Player: &CityPlayerData{
				Name: "Player", Fixed: fixed, Raw32: make([]byte, 32),
				Groups: []CityGroupData{{Raw80: make([]byte, 80), F44: owner, Actors: []uint16{2, 3}}},
			}},
			{Class: "Human", Unit: human(0x20, "Hero")},
			{Class: "Human", Unit: human(0x30, "Foe")},
		},
		State: NewCityStateData("Hero"),
	})
	if err != nil {
		t.Fatal(err)
	}
	doc, err := city.DocumentData()
	if err != nil {
		t.Fatal(err)
	}
	hero := doc.Objects[1]
	binary.LittleEndian.PutUint32(documentRaw(t, hero, "U50"), code)
	binary.LittleEndian.PutUint32(documentRaw(t, hero, "U54"), 3)
	binary.LittleEndian.PutUint32(documentRaw(t, hero, "U58"), 5)
	command := documentRaw(t, hero, "U158")
	binary.LittleEndian.PutUint32(command[0x0c:], key)
	command[8], command[0x14] = 5, 6
	return doc
}

// The second game's Unit record never holds order 3 with command +0x0c
// zero, which the original's attack planner reads unchecked
// (R2-ENGINE-936). An attack order whose target the document holds writes
// that target's key; one with no target, a target the document does not
// hold, or itself as target writes order 12, state 0 and phase 0, what the
// original holds for this hero with no order (R2-SESSION-356). Other
// orders keep today's record.
func TestTheSecondGameUnitRecordNeverHoldsAnAttackOrderWithoutItsTarget(t *testing.T) {
	cases := []struct {
		name                string
		code, key           uint32
		order, state, phase uint32
		target              uint32
	}{
		{"a held target", 3, 0x30, 3, 3, 5, 0x30},
		{"no target", 3, 0, 12, 0, 0, 0},
		{"a target the document does not hold", 3, 0x99, 12, 0, 0, 0},
		{"itself as target", 3, 0x20, 12, 0, 0, 0},
		{"a guard order", 11, 0x30, 11, 3, 5, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc := attackDocument(t, c.code, c.key)
			out, changed, err := DocumentInProfile(doc, SecondGame, ProfileKeeps{})
			if err != nil || !changed {
				t.Fatalf("second form: %v %v", changed, err)
			}
			hero := out.Objects[1]
			order := binary.LittleEndian.Uint32(documentRaw(t, hero, "U50"))
			state := binary.LittleEndian.Uint32(documentRaw(t, hero, "U54"))
			phase := binary.LittleEndian.Uint32(documentRaw(t, hero, "U58"))
			command := documentRaw(t, hero, "URaw184")
			if order != c.order || state != c.state || phase != c.phase || u32(command, 0x0c) != c.target {
				t.Fatalf("order %d state %d phase %d target %#x, want %d %d %d %#x",
					order, state, phase, u32(command, 0x0c), c.order, c.state, c.phase, c.target)
			}
			if command[8] != 0 || command[0x14] != 0 || u16(command, 0x16) != commandWord {
				t.Fatalf("command +8 %d +0x14 %d +0x16 %d", command[8], command[0x14], u16(command, 0x16))
			}
			if rest := slices.Concat(command[:0x0c], command[0x10:0x16], command[0x18:]); slices.ContainsFunc(rest, func(b byte) bool { return b != 0 }) {
				t.Fatal("the command block holds bytes beyond its target and word")
			}
			if engine := documentRaw(t, doc.Objects[1], "U50"); binary.LittleEndian.Uint32(engine) != c.code {
				t.Fatal("the producer changed the engine document")
			}
		})
	}
}
