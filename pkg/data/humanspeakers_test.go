package data

import "testing"

// A Humans row is a speaker record at its server id, dressed by the outfit
// reader; a server-built id above 10000 gives none; key me is the player's
// own character.
func TestHumanSpeakerFaces(t *testing.T) {
	c := testCollection{
		{},
		{name: "Mage", params: row(map[int]int32{16: 24, 17: 4, 18: 1, 24: 517})},
		{name: "Peasant", params: row(map[int]int32{16: 1, 17: 46, 18: 0, 24: 2108})},
		{name: "Start", params: row(map[int]int32{16: 3, 17: 32, 18: 0, 24: 10210})},
		{name: "Horse", params: row(map[int]int32{16: 21, 17: 1, 18: 0, 24: 21})},
	}
	var robe Equipment
	robe.SetCode(5, 0x0301)
	faces := HumanSpeakerFaces(c, 21, func(r int) Equipment {
		if r == 1 {
			return robe
		}
		return Equipment{}
	})
	mage := faces[517]
	if mage.Kind != NPCFigure || mage.Dir != FigureDirFor(true, true) || mage.Face != 4 || mage.Worn != robe {
		t.Fatalf("517 = %+v", mage)
	}
	if !mage.Tokens.Has(NPCTokenHuman) || !mage.Tokens.Has(NPCTokenMage) || !mage.Tokens.Has(NPCTokenFemale) || !mage.Tokens.Has(NPCTokenFace) || mage.Tokens.Has(NPCTokenNotMage) {
		t.Fatalf("517 tokens = %b", mage.Tokens)
	}
	peasant := faces[2108]
	if peasant.Dir != FigureDirFor(false, false) || peasant.Face != 46 || !peasant.Tokens.Has(NPCTokenNotFemale) || peasant.Worn != (Equipment{}) {
		t.Fatalf("2108 = %+v", peasant)
	}
	if _, ok := faces[10210]; ok {
		t.Fatal("a server-built id gave a record")
	}
	me := faces[21]
	if me.Kind != NPCNoPicture || !me.Tokens.Has(NPCTokenMe) || !me.Tokens.Has(NPCTokenHero) || me.Start || me.Archetype {
		t.Fatalf("21 = %+v", me)
	}
	if len(faces) != 3 {
		t.Fatalf("records = %d, want 3", len(faces))
	}
	if _, ok := HumanSpeakerFaces(c, 0, nil)[21]; !ok {
		t.Fatal("without a player key the row at 21 is its own record")
	}
}

// A speaker no live unit carries is the second game's fallback actor: a bare
// female mage doll with face byte key-80, for keys whose byte is 1..255
// (R2-ENGINE-583).
func TestFallbackSpeakerFace(t *testing.T) {
	rec, ok := FallbackSpeakerFace(97)
	if !ok || rec.Kind != NPCFigure || rec.Dir != FigureDirWomanMage || rec.Face != 17 || rec.Worn != (Equipment{}) {
		t.Fatalf("speaker 97: %+v %v", rec, ok)
	}
	if !rec.Tokens.Has(NPCTokenMage) || !rec.Tokens.Has(NPCTokenFemale) || rec.Tokens.Has(NPCTokenHero) {
		t.Errorf("speaker 97 tokens %v", rec.Tokens)
	}
	for _, k := range []int32{FallbackSpeakerFirst - 1, FallbackSpeakerLast + 1} {
		if _, ok := FallbackSpeakerFace(k); ok {
			t.Errorf("key %d has a fallback record", k)
		}
	}
	if FallbackSpeakerFirst != 81 || FallbackSpeakerLast != 335 {
		t.Errorf("keys %d..%d", FallbackSpeakerFirst, FallbackSpeakerLast)
	}
}
