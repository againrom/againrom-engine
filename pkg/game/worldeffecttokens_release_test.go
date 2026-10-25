package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestReleaseCurrentWorldEffectTokenFieldsSaveColdLoad(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	_, raw := groundCorpusFile(t, "2026-08-15/game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b")
	for role := 0; role < 2; role++ {
		for field := 0; field < 20; field++ {
			if role == 0 && field == 1 {
				continue
			}
			t.Run(fmt.Sprintf("record%d/field%d", role, field), func(t *testing.T) {
				f := releaseFront(t)
				app, _, _ := openWorldEffectsTestSave(t, f, raw, "tokens.sav")
				state, w := f.live.mission.state.savedDocument, f.live.world
				d := w.SavedWorldEffectDrivers()
				at := -1
				for i, object := range state.WorldEffects.TokenObjects {
					class := state.Document.Objects[object-1].Class
					if role == 0 && class == "PointEffect" || role == 1 && (class == "Effect" || class == "Effect_DirectDamage") {
						at = i
						break
					}
				}
				if at < 0 {
					t.Fatal("input lacks the requested Token class")
				}
				d.Tokens[at] = changedWorldEffectToken(d.Tokens[at], field)
				if err := w.ImportOriginalWorldEffectDrivers(d); err != nil {
					t.Fatal(err)
				}
				store, name, written := menuSAVE(t, f, app, OriginalStore{})
				doc, err := sav.DecodeDocumentData(written)
				if err != nil {
					t.Fatal(err)
				}
				a, err := readCurrentActions(&doc)
				if err != nil || a == nil || at >= len(a.WorldEffectTokens) {
					t.Fatal("SAVE lost Token binding", err)
				}
				got, err := savedItemToken(&doc.Objects[a.WorldEffectTokens[at].Object-1])
				if err != nil || got != d.Tokens[at] {
					t.Fatal("SAVE did not follow current Token", got, d.Tokens[at], err)
				}
				cold, coldApp := loadSAVWindow(t, store, name)
				assertCurrentWorldEqual(t, w, cold.live.world, "changed effect Token")
				store2, name2, _ := menuSAVE(t, cold, coldApp, OriginalStore{})
				cold2, _ := loadSAVWindow(t, store2, name2)
				assertCurrentWorldEqual(t, w, cold2.live.world, "changed effect Token second SAVE")
				for range 3 {
					f.live.tick()
					cold.live.tick()
					assertCurrentWorldEqual(t, w, cold.live.world, "changed effect Token next tick")
				}
			})
		}
	}
}

func TestReleaseCurrentWorldEffectTokensFollowWorldSAVEAndColdLOAD(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	_, raw := groundCorpusFile(t, "2027-09-07/game0125.sav", "3a055c8dcef6f053721e1c1478c82552c199b92fb178e9f034fe7500077b4efd")
	f := releaseFront(t)
	app, _, _ := openWorldEffectsTestSave(t, f, raw, "tokens.sav")
	w := f.live.world
	drivers := w.SavedWorldEffectDrivers()
	if drivers == nil || len(drivers.Tokens) == 0 {
		t.Fatal("original LOAD did not fill Token carrier")
	}
	for i := range drivers.Tokens {
		drivers.Tokens[i].T18 ^= uint16(0x100 + i)
		drivers.Tokens[i].T1C ^= uint32(0x12340000 + i)
	}
	if err := w.ImportOriginalWorldEffectDrivers(drivers); err != nil {
		t.Fatal(err)
	}
	want := w.SavedWorldEffectDrivers().Tokens
	store, name, _ := menuSAVE(t, f, app, OriginalStore{})
	cold, coldApp := loadSAVWindow(t, store, name)
	if got := cold.live.world.SavedWorldEffectDrivers().Tokens; !reflect.DeepEqual(got, want) {
		t.Fatalf("cold LOAD Tokens = %+v, want %+v", got, want)
	}
	assertCurrentWorldEqual(t, w, cold.live.world, "current effect Tokens")
	_, _, second := menuSAVE(t, cold, coldApp, OriginalStore{})
	doc, err := sav.DecodeDocumentData(second)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range doc.Objects {
		if r.Class != "AreaEffect" && r.Class != "Effect" && r.Class != "Effect_DirectDamage" {
			continue
		}
		key, _ := savedStructureValue(&r, "Identity")
		for _, token := range want {
			if token.Identity == key {
				got, err := savedItemToken(&r)
				if err != nil || got.T18 != token.T18 || got.T1C != token.T1C {
					t.Fatal("second SAVE lost current Token", key, got, token, err)
				}
			}
		}
	}
	for range 3 {
		f.live.tick()
		cold.live.tick()
		assertCurrentWorldEqual(t, w, cold.live.world, "current effect Tokens next tick")
	}
}

func TestReleaseCurrentWorldEffectTokenCorpusResave(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	_ = releaseFront(t)
	fields := []string{"RuntimeID", "T0C", "T0E", "T08", "T18", "T1C", "Identity", "Reference", "Block12"}
	matched, changed := make([]int, len(fields)), make([]int, len(fields))
	files, tokens := 0, 0
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, f *FrontEnd) {
		if !mf.present {
			return
		}
		source, err := sav.DecodeDocumentData(mf.raw)
		if err != nil || len(source.World.Effects) == 0 {
			return
		}
		open, town, err := f.RestoreOriginal(mf.raw)
		if err != nil || town {
			t.Fatalf("effect corpus LOAD %s: %v, town %v", mf.rel, err, town)
		}
		if err := f.App("effect Token census").OpenMission(open); err != nil {
			t.Fatal(err)
		}
		snapshot, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(snapshot, label)
		if err != nil {
			t.Fatal(err)
		}
		written, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		var oldWorld sim.World
		if err := oldWorld.UnmarshalBinary(snapshot.World); err != nil {
			t.Fatal(err)
		}
		drivers := oldWorld.SavedWorldEffectDrivers()
		drivers.Tokens = nil
		if err := oldWorld.ImportOriginalWorldEffectDrivers(drivers); err != nil {
			t.Fatal(err)
		}
		oldSnapshot := snapshot
		oldSnapshot.World, err = oldWorld.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		oldWire, err := f.ExportCurrentSave(oldSnapshot, label)
		if err != nil {
			t.Fatal(err)
		}
		withoutMetadata := written
		actions, err := readCurrentActions(&withoutMetadata)
		if err != nil || actions == nil {
			t.Fatal("missing comparison supplement", err)
		}
		actions.WorldEffectTokens = nil
		actionWire, err := json.Marshal(actions)
		if err != nil {
			t.Fatal(err)
		}
		if err := sav.SetNativeActions(&withoutMetadata.State, actionWire); err != nil {
			t.Fatal(err)
		}
		comparison, err := sav.EncodeDocumentData(withoutMetadata)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(comparison, oldWire) {
			t.Error("SAVE differs from retained-source control outside Token metadata")
		}
		family, err := worldEffectRecords(&source)
		if err != nil {
			t.Fatal(err)
		}
		next, err := worldEffectRecords(&written)
		if err != nil || len(family) != len(next) {
			t.Fatal("unticked effect topology changed", len(family), len(next), err)
		}
		files++
		for i, object := range family {
			a, b := &source.Objects[object-1], &written.Objects[next[i]-1]
			if a.Class != b.Class {
				t.Fatal("effect class changed", a.Class, b.Class)
			}
			tokens++
			for j, field := range fields {
				equal := false
				if field == "Block12" {
					x, err := savedItemToken(a)
					if err != nil {
						t.Fatal(err)
					}
					y, err := savedItemToken(b)
					if err != nil {
						t.Fatal(err)
					}
					equal = x.Position == y.Position
				} else {
					x, err := savedStructureValue(a, field)
					if err != nil {
						t.Fatal(err)
					}
					y, err := savedStructureValue(b, field)
					if err != nil {
						t.Fatal(err)
					}
					equal = x == y
				}
				if equal {
					matched[j]++
				} else {
					changed[j]++
					t.Logf("Token normalization %s %s %d.%s", mf.rel, a.Class, object, field)
				}
			}
		}
	})
	if files == 0 || tokens == 0 {
		t.Fatal("no original world-effect Token population")
	}
	for i, field := range fields {
		t.Logf("WORLD-EFFECT-TOKEN-CENSUS field=%s files=%d tokens=%d equal=%d normalized=%d", field, files, tokens, matched[i], changed[i])
	}
}
