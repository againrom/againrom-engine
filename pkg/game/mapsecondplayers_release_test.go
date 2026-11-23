//go:build release

package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

// rosterOf is the type-5 payload of a second-game map file.
func rosterOf(t *testing.T, raw []byte) []byte {
	t.Helper()
	doc, err := alm.OpenDocument(raw, alm.SecondFormat)
	if err != nil {
		t.Fatal(err)
	}
	for i, id := range doc.RecordTypeIDs() {
		if id == alm.TypeGroups {
			return doc.RecordPayload(i)
		}
	}
	t.Fatal("no type-5 record")
	return nil
}

func playerRecordsOf(d *ui.InspectionDocument) (n int) {
	for _, r := range d.Records {
		if r.Kind == ui.KindPlayer {
			n++
		}
	}
	return n
}

func fieldOf(fields []ui.InspectionField, key string) (ui.InspectionField, bool) {
	for _, f := range fields {
		if f.Key == key {
			return f, true
		}
	}
	return ui.InspectionField{}, false
}

// Every installed second-game map lists its roster, and a name, a colour and
// a relation edit saves, reloads and changes those words alone.
func TestReleaseSecondMapPlayerEditSavesAndReloads(t *testing.T) {
	x := installedInspector(t)
	sources, raws := installedSecondMaps(t, x)
	edited := 0
	for i, source := range sources {
		d, err := x.Open(source)
		if err != nil {
			t.Fatal(source, err)
		}
		was := rosterOf(t, raws[i])
		n := len(was) / alm.GroupSize
		if got := playerRecordsOf(d); got != n {
			t.Fatalf("%s: %d players listed, the map holds %d", source, got, n)
		}
		if n < 2 {
			continue
		}
		host := d.Edits.(*mapInspectionEdits)
		if _, ok := ui.Seam[ui.InspectionTextFields](host); !ok {
			t.Fatal("the second game's row offers no text fields")
		}
		fields := host.Fields(ui.KindPlayer, 1)
		for _, key := range []string{"name", "colour", "relation0", "flags", "unread", "note"} {
			if _, ok := fieldOf(fields, key); !ok {
				t.Fatalf("%s: player field %q is not listed", source, key)
			}
		}
		for _, key := range []string{"flags", "unread", "relation1"} {
			if f, _ := fieldOf(fields, key); !f.ReadOnly {
				t.Fatalf("%s: %q is not read-only", source, key)
			}
			value := int64(9)
			if key == "relation1" {
				value = 0
			}
			if _, err := host.SetField(ui.KindPlayer, 1, key, value); err == nil {
				t.Fatalf("%s: read-only %q took an edit", source, key)
			}
		}
		if _, err := host.SetField(ui.KindPlayer, 1, "relation0", 3); err == nil {
			t.Fatalf("%s: relation 3 was taken", source)
		}
		colour, relation := int64(7), int64(2)
		if cur, _ := fieldOf(fields, "colour"); cur.Value == colour {
			colour = 8
		}
		if cur, _ := fieldOf(fields, "relation0"); cur.Value == relation {
			relation = 0
		}
		if _, err := host.SetTextField(ui.KindPlayer, 1, "name", "Edited"); err != nil {
			t.Fatal(source, err)
		}
		if _, err := host.SetField(ui.KindPlayer, 1, "colour", colour); err != nil {
			t.Fatal(source, err)
		}
		if _, err := host.SetField(ui.KindPlayer, 1, "relation0", relation); err != nil {
			t.Fatal(source, err)
		}
		target, err := host.SaveAs(filepath.Join(t.TempDir(), "players.alm"))
		if err != nil {
			t.Fatal(source, err)
		}
		saved, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		reloaded, err := x.Open(target)
		if err != nil || !bytes.Equal(reloaded.SourceBytes, saved) {
			t.Fatalf("%s: reload: %v", source, err)
		}
		again := reloaded.Edits.(*mapInspectionEdits).Fields(ui.KindPlayer, 1)
		if f, _ := fieldOf(again, "name"); f.TextValue != "Edited" {
			t.Fatalf("%s: reloaded name %q", source, f.TextValue)
		}
		if f, _ := fieldOf(again, "colour"); f.Value != colour {
			t.Fatalf("%s: reloaded colour %d", source, f.Value)
		}
		if f, _ := fieldOf(again, "relation0"); f.Value != relation {
			t.Fatalf("%s: reloaded relation %d", source, f.Value)
		}
		now := rosterOf(t, saved)
		le := binary.LittleEndian
		rec, old := now[alm.GroupSize:2*alm.GroupSize], was[alm.GroupSize:2*alm.GroupSize]
		if le.Uint32(rec[alm.GroupColour:]) != uint32(colour) || le.Uint16(rec[alm.GroupRelations:]) != uint16(relation) {
			t.Fatalf("%s: record bytes %x", source, rec)
		}
		for k := range rec {
			inName := k >= alm.GroupName && k < alm.GroupName+alm.GroupNameLen
			inColour := k < alm.GroupColour+4
			inRelation := k == alm.GroupRelations || k == alm.GroupRelations+1
			if rec[k] != old[k] && !inName && !inColour && !inRelation {
				t.Fatalf("%s: byte %d of the player changed", source, k)
			}
		}
		if !bytes.Equal(now[:alm.GroupSize], was[:alm.GroupSize]) || !bytes.Equal(now[2*alm.GroupSize:], was[2*alm.GroupSize:]) {
			t.Fatalf("%s: another player changed", source)
		}
		edited++
		if edited == 3 {
			break
		}
	}
	if edited == 0 {
		t.Fatal("no map held two players")
	}
	t.Logf("PASS: %d player edits saved and reloaded (%s)", edited, installLanguage(x))
}

// An added player saves and the map reloads with it; the other records of
// the roster keep their bytes.
func TestReleaseSecondMapAddsPlayer(t *testing.T) {
	x := installedInspector(t)
	sources, raws := installedSecondMaps(t, x)
	added := 0
	for i, source := range sources {
		was := rosterOf(t, raws[i])
		n := len(was) / alm.GroupSize
		if n == 0 || n >= 16 {
			continue
		}
		d, err := x.Open(source)
		if err != nil {
			t.Fatal(source, err)
		}
		host := d.Edits.(*mapInspectionEdits)
		if !slices.Contains(host.AddKinds(), ui.KindPlayer) {
			t.Fatal("the second game's palettes offer no player")
		}
		doc, err := host.AddPlayer()
		if err != nil || playerRecordsOf(doc) != n+1 {
			t.Fatalf("%s: add: %v", source, err)
		}
		target, err := host.SaveAs(filepath.Join(t.TempDir(), "added.alm"))
		if err != nil {
			t.Fatal(source, err)
		}
		saved, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		reloaded, err := x.Open(target)
		if err != nil || playerRecordsOf(reloaded) != n+1 || !bytes.Equal(reloaded.SourceBytes, saved) {
			t.Fatalf("%s: reload: %v", source, err)
		}
		now := rosterOf(t, saved)
		if len(now) != len(was)+alm.GroupSize || !bytes.Equal(now[:len(was)], was) {
			t.Fatalf("%s: the roster's old records changed", source)
		}
		m, err := alm.OpenROM2(saved)
		if err != nil || len(m.Groups) != n+1 {
			t.Fatalf("%s: loader: %v", source, err)
		}
		if g := m.Groups[n]; g.Name != "New Player "+string(rune('1'+n)) && n < 9 {
			t.Fatalf("%s: new player %+v", source, g)
		}
		if _, err := host.Delete(ui.KindPlayer, n); err == nil {
			t.Fatalf("%s: a player was deleted", source)
		}
		added++
		if added == 3 {
			break
		}
	}
	if added == 0 {
		t.Fatal("no map took a player")
	}
	t.Logf("PASS: %d added players saved and reloaded (%s)", added, installLanguage(x))
}
