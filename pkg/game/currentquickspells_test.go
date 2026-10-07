package game

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestCurrentCustomQuickSpellsKeepTwoCyclesAndOrdinaryEdits(t *testing.T) {
	f, _ := originalCityRouteFixture(t)
	f.quickSpells = [4]uint32{17, 89, 23, 0}
	for cycle := 0; cycle < 2; cycle++ {
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
		ordinary, err := originalQuickSpellsMustRead(raw)
		if err != nil || ordinary != ([4]uint32{0, 0, 23, 0}) {
			t.Fatal("ordinary custom shortcut projection", ordinary, err)
		}
		cold := &FrontEnd{InstallResources: f.InstallResources}
		if _, town, err := cold.RestoreOriginal(raw); err != nil || !town || cold.quickSpells != f.quickSpells {
			t.Fatal("custom shortcut cold LOAD", cycle, town, err)
		}
		for i := range doc.State.ValueRecords {
			r := &doc.State.ValueRecords[i]
			if r.Path == "/SpellBook/Shortcuts" {
				binary.LittleEndian.PutUint32(r.Value.Bytes, 0)
			}
		}
		edited, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		changed := &FrontEnd{InstallResources: f.InstallResources}
		if _, town, err := changed.RestoreOriginal(edited); err != nil || !town || changed.quickSpells != ([4]uint32{1, 89, 23, 0}) {
			t.Fatal("ordinary shortcut edit lost", changed.quickSpells, town, err)
		}
		f = cold
	}
}

func originalQuickSpellsMustRead(raw []byte) ([4]uint32, error) {
	file, err := sav.Open(raw)
	if err != nil {
		return [4]uint32{}, err
	}
	return originalQuickSpells(file)
}

func TestCurrentCustomShortcutMalformedPolicyIsAtomic(t *testing.T) {
	f, _ := originalCityRouteFixture(t)
	f.quickSpells = [4]uint32{89}
	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"slot", "known ID", "duplicate", "width"} {
		t.Run(mode, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "slot":
				a.Session.QuickSpells[0].Slot = 4
			case "known ID":
				a.Session.QuickSpells[0].ID = 1
			case "duplicate":
				a.Session.QuickSpells = append(a.Session.QuickSpells, currentQuickSpell{1, 89})
			case "width":
				a.Session.QuickSpells[0].ID = 65536
			}
			leaf, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
				t.Fatal(err)
			}
			altered, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			town, before := f.Town, f.quickSpells
			if _, _, err := f.RestoreOriginal(altered); err == nil || f.Town != town || f.quickSpells != before {
				t.Fatal("malformed custom shortcut accepted or changed session", err)
			}
		})
	}
}

func TestCurrentCustomPressedSpellFollowsOrdinaryCell(t *testing.T) {
	f := currentArchiveFixture(t)
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ApplicationState == nil {
		t.Fatal("fixture has no current view")
	}
	snapshot.ApplicationState.View.PressedSpell = 89
	raw, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	cold := openCurrentArchive(t, raw)
	if cold.live.view.SaveApplication().PressedSpell != 89 {
		t.Fatal("current pressed custom spell lost")
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	for i := range doc.State.ValueRecords {
		r := &doc.State.ValueRecords[i]
		if r.Path == "/SpellBook/Pressed" {
			r.Value.Int32 = 0
		}
	}
	changed, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	ordinary := openCurrentArchive(t, changed)
	if ordinary.live.view.SaveApplication().PressedSpell != 1 {
		t.Fatal("ordinary pressed cell was overwritten")
	}
}
