package game

import (
	"fmt"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentWorldEffectTokensIgnoreRetainedDocument(t *testing.T) {
	f := worldSpellFixtureFront(t)
	raw := stagedWorldEffectSource(t, 4, 2, 0)
	openWorldEffectsTestSave(t, f, raw, "tokens.sav")
	s := f.live.mission.state.savedDocument
	index := s.Document.World.Effects[0]
	want, err := savedStructureValue(&s.Document.Objects[index-1], "T18")
	if err != nil {
		t.Fatal(err)
	}
	savedObjectSetValue(&s.Document.Objects[index-1], "T18", want^0x102)
	b := &generatedDocumentBuilder{doc: sav.DocumentData{World: &sav.DocumentWorldData{}}, state: &SnapshotSAVDocument{}}
	if err := b.currentWorldEffects(s, f.live.world); err != nil {
		t.Fatal(err)
	}
	doc := b.doc
	got, err := savedStructureValue(&doc.Objects[doc.World.Effects[0]-1], "T18")
	if err != nil || got != want {
		t.Fatalf("World Token mask = %#x, want %#x; retained source was changed: %v", got, want, err)
	}
}

func TestCurrentWorldEffectTokenWordsFollowWorld(t *testing.T) {
	fields := []string{"RuntimeID", "T0C", "T0E", "T08", "T18", "T1C", "Identity", "Reference"}
	for role := 0; role < 2; role++ {
		for field := 0; field < len(fields)+12; field++ {
			if role == 0 && (field == 1 || field == 6 || field == 8 || field == 9) {
				continue
			}
			t.Run(fmt.Sprintf("record%d/field%d", role, field), func(t *testing.T) {
				f := worldSpellFixtureFront(t)
				openWorldEffectsTestSave(t, f, stagedWorldEffectSource(t, 4, 2, 0), "tokens.sav")
				state := f.live.mission.state.savedDocument
				drivers := f.live.world.SavedWorldEffectDrivers()
				if drivers == nil || len(drivers.Tokens) != 2 {
					t.Fatal("LOAD did not fill both Tokens")
				}
				token := &drivers.Tokens[role]
				*token = changedWorldEffectToken(*token, field)
				if err := f.live.world.ImportOriginalWorldEffectDrivers(drivers); err != nil {
					t.Fatal(err)
				}
				b := &generatedDocumentBuilder{doc: sav.DocumentData{World: &sav.DocumentWorldData{}}, state: &SnapshotSAVDocument{}}
				if err := b.currentWorldEffects(state, f.live.world); err != nil {
					t.Fatal(err)
				}
				got, err := savedItemToken(&b.doc.Objects[role])
				if err != nil || got != *token {
					t.Fatalf("Token %d/%d = %+v, want %+v: %v", role, field, got, *token, err)
				}
				loaded, err := savedItemToken(&state.Document.Objects[state.WorldEffects.TokenObjects[role]-1])
				if err != nil || loaded == got {
					t.Fatal("sensitivity control did not differ from loaded source", err)
				}
			})
		}
	}
}

func changedWorldEffectToken(token sim.SavedObjectToken, field int) sim.SavedObjectToken {
	switch field {
	case 0:
		token.RuntimeID = 700
	case 1:
		token.T0C ^= 1
	case 2:
		token.T0E ^= 0x102
	case 3:
		token.T08 ^= 0x10200
	case 4:
		token.T18 ^= 0x104
	case 5:
		token.T1C ^= 0x12345678
	case 6:
		token.Identity ^= 0x1000
	case 7:
		token.Reference ^= 0x10203040
	default:
		token.Position[field-8] ^= 0x42
	}
	return token
}

func TestWorldEffectTokensSurviveBinaryAndDetachedCopies(t *testing.T) {
	f := worldSpellFixtureFront(t)
	openWorldEffectsTestSave(t, f, stagedWorldEffectSource(t, 4, 2, 0), "tokens.sav")
	w := f.live.world
	want := w.SavedWorldEffectDrivers().Tokens
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold sim.World
	if err := cold.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if cold.Hash() != w.Hash() || len(cold.SavedWorldEffectDrivers().Tokens) != len(want) {
		t.Fatal("Token carrier changed through the byte form")
	}
	copy := cold.SavedWorldEffectDrivers()
	copy.Tokens[0].T18 ^= 1
	if cold.SavedWorldEffectDrivers().Tokens[0] != want[0] {
		t.Fatal("returned Tokens alias the World")
	}
}
