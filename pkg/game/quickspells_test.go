package game

import (
	"againrom/pkg/formats/sav"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// An independent envelope producer. Its schema deliberately does not use
// Snapshot, so old/malformed tests cannot pass by sharing the production writer.

func TestQuickSpellsNativeIndependentOldNewAndMalformed(t *testing.T) {
	want := [4]uint32{89, 17, 3, 65535}
	f, _ := city1095Front(t)
	f.quickSpells = want
	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	leaf, present, err := sav.NativeActions(doc.State)
	if err != nil || !present {
		t.Fatal("current quick-spell policy absent", err)
	}
	var independent struct {
		Session *struct {
			QuickSpells []struct {
				Slot uint8
				ID   uint32
			}
		}
	}
	if err := json.Unmarshal(leaf, &independent); err != nil || independent.Session == nil {
		t.Fatal("independent current field", err)
	}
	seen := map[uint8]uint32{}
	for _, row := range independent.Session.QuickSpells {
		if row.Slot >= 4 || row.ID != want[row.Slot] {
			t.Fatal("independent current slot", row)
		}
		seen[row.Slot] = row.ID
	}
	if seen[0] != 89 || seen[3] != 65535 {
		t.Fatal("custom slot payload lost", seen)
	}
	g, _ := city1095Front(t)
	if _, town, err := g.RestoreOriginal(raw); err != nil || !town || g.quickSpells != want {
		t.Fatalf("fresh current SAV slots=%v town%v err%v", g.quickSpells, town, err)
	}
	for _, bad := range [][4]uint32{{65536}, {17, 17}} {
		t.Run("malformed", func(t *testing.T) {
			before := *f
			invalid := snapshot
			invalid.QuickSpells = bad
			if _, err := f.ExportCurrentSave(invalid, label); err == nil {
				t.Fatal("writer accepted malformed slots")
			}
			changed, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			actions, err := readCurrentActions(&changed)
			if err != nil || actions == nil || actions.Session == nil {
				t.Fatal("current session absent", err)
			}
			actions.Session.QuickSpells = nil
			for slot, id := range bad {
				if id != 0 {
					actions.Session.QuickSpells = append(actions.Session.QuickSpells, currentQuickSpell{Slot: uint8(slot), ID: id})
				}
			}
			leaf, err := json.Marshal(actions)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&changed.State, leaf); err != nil {
				t.Fatal(err)
			}
			altered, err := sav.EncodeDocumentData(changed)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := f.RestoreOriginal(altered); err == nil {
				t.Fatal("current SAV accepted malformed slots")
			}
			if !reflect.DeepEqual(before, *f) {
				t.Fatal("failed restore changed active session")
			}
		})
	}
}

func TestQuickSpellsCandidateAndMissionLifetime(t *testing.T) {
	f := missionFrontEnd(t)
	f.Town = NewTown(Campaign{})
	want := [4]uint32{17, 3, 6, 89}
	f.quickSpells = want
	open := f.MissionOpener(10)
	_, _, _, _, _, _, _, _, _, _, err := open()
	if err != nil {
		t.Fatal(err)
	}
	if f.quickSpells != want {
		t.Fatal("mission entry lost bindings")
	}
	f.quickSpells[0] = 91
	f.arriveInTown()
	if f.quickSpells[0] != 91 {
		t.Fatal("town arrival lost bindings")
	}
	before := *f
	_, source := city1095Front(t)
	bad, err := sav.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	bad.Head.Mission = 20
	if _, _, err := f.RestoreOriginal(bad.Marshal()); err == nil {
		t.Fatal("missing world accepted")
	}
	if !reflect.DeepEqual(before, *f) {
		t.Fatal("failed mission restore changed bindings/session")
	}
	f.installCandidate(&restoreCandidate{townOnly: true, town: NewTown(Campaign{}), quickSpells: want})
	if f.quickSpells != want {
		t.Fatal("candidate did not own bindings")
	}
	f.resetSessionForNewGame()
	if f.quickSpells != [4]uint32{} {
		t.Fatal("new campaign did not reset bindings")
	}
}

func TestQuickSpellsNewGameCommitsResetOnlyAfterPreparation(t *testing.T) {
	f := missionFrontEnd(t)
	f.Table, f.Humans = &mapload.Table{}, fourBaseHumans()
	f.Bodies, f.Town = data.NewBodyList("unarmed", "mage"), NewTown(Campaign{})
	want := [4]uint32{16, 1, 6, 19}
	f.quickSpells = want
	res := ui.ChargenResult{Choices: []int{1, 1, 2}, Stats: []int{30, 20, 18, 16}}
	if _, err := f.prepareNewGame(20, res); err == nil {
		t.Fatal("missing map should fail preparation")
	}
	if f.quickSpells != want {
		t.Fatal("failed NEW GAME cleared the running campaign")
	}
	open, err := f.prepareNewGame(10, res)
	if err != nil {
		t.Fatal(err)
	}
	if f.quickSpells != want {
		t.Fatal("successful draft changed unadopted session")
	}
	v, _, _, _, _, _, _, _, _, _, err := open()
	if err != nil {
		t.Fatal(err)
	}
	if slots, current, armed := v.QuickSpellState(); f.quickSpells != [4]uint32{} || slots != [4]uint32{} || current != 0 || armed {
		t.Fatal("NEW GAME commit did not reset only the adopted session")
	}
}

func TestQuickSpellsCustomTownUsesCurrentSAV(t *testing.T) {
	f, document := originalCityRouteFixture(t)
	document.err = errors.New("retained writer must not select SAVE")
	f.quickSpells = [4]uint32{17, 89}
	want := f.quickSpells
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !IsOriginal(name) {
		t.Fatal("custom quick-spell SAVE", name, err)
	}
	rows := list()
	if len(rows) != 1 || rows[0].Name != localOriginalSaveToken(name) {
		t.Fatal("custom quick-spell list", rows)
	}
	f.quickSpells = [4]uint32{}
	if _, town, err := load(rows[0].Name); err != nil || !town || f.quickSpells != want {
		t.Fatal("custom quick-spell LOAD", f.quickSpells, town, err)
	}
	if len(document.updates) != 0 {
		t.Fatal("custom quick spells selected retained writer")
	}
}
