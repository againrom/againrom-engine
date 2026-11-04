package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// The names page is one "<unit id>=<name>" line per named unit: the reader
// skips a line without an id from 1 to 65535 or without a name and keeps an
// id's first line; the writer orders the lines by id and ends each with CRLF.
func TestUnitNamesPageReadsAndWritesItsLines(t *testing.T) {
	got := unitNames(" 12 = Bjorn \r\n0=Zero\r\nx=Bad\r\n12=Second\r\n7=\r\n65536=Big\r\nno line\r\n3=Ann")
	if want := map[uint16]string{12: "Bjorn", 3: "Ann"}; !maps.Equal(got, want) {
		t.Fatalf("read %v, want %v", got, want)
	}
	if b := unitNamesText(got); string(b) != "3=Ann\r\n12=Bjorn\r\n" {
		t.Fatalf("wrote %q", b)
	}
	if unitNamesText(nil) != nil && len(unitNamesText(nil)) != 0 {
		t.Fatal("an empty page has lines")
	}
}

// A placed unit's name, in both views of the unit window, floating and in
// the right panel, in English and Russian: the empty field shows the kind's
// name; a name set writes the unit's line in the map's names page and not a
// byte of the map, and the card shows it; a name over 32 characters is
// refused with the limit; an emptied field removes the line and the empty
// page; Undo restores each step. The basic view stands the name in its
// first block.
func TestEditorUnitNameInBothViews(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		for _, docked := range []bool{false, true} {
			for _, plain := range []bool{true, false} {
				t.Run(fmt.Sprintf("%s/docked=%v/plain=%v", lang, docked, plain), func(t *testing.T) {
					s, raw := viewsMap(t)
					s.inspector.archives.Base.Profile.Language = lang
					s.inspector.recordOptions = nil
					words := ui.AuthoredWords()
					for i := range words.UnitNames {
						words.UnitNames[i] = "Kind"
					}
					s.inspector.panelWords, s.unitViews = &words, nil
					id, err := unitID(s.model, 0)
					if err != nil || id == 0 {
						t.Fatalf("the fixture unit has id %d: %v", id, err)
					}
					name := map[string]string{"": "Bjorn the Red", "russian": "Бьорн Рыжий"}[lang]
					e, c := motionWindow(t, s, lang, plain, docked)
					fields := e.ObjectFields()
					f := fields[fieldIndex(t, fields, unitNameField)]
					if !f.Text || !f.Inherit || f.TextValue != "" || f.Inherited != "Kind" {
						t.Fatalf("empty name: text %v inherit %v value %q shows %q", f.Text, f.Inherit, f.TextValue, f.Inherited)
					}
					if plain {
						if block := e.PlainFieldBlocks()[unitNameField]; block != "main" {
							t.Fatalf("the basic view stands the name in %q", block)
						}
					}
					typeText(t, e, c, unitNameField, name)
					line := fmt.Sprintf("%d=%s\r\n", id, name)
					if got := string(s.workspace.Pages[unitNamesPage]); got != line || !bytes.Equal(s.model.Bytes(), raw) {
						t.Fatalf("page %q (want %q), map changed %v", got, line, !bytes.Equal(s.model.Bytes(), raw))
					}
					if f := e.ObjectFields()[fieldIndex(t, e.ObjectFields(), unitNameField)]; f.TextValue != name {
						t.Fatalf("the field shows %q after the edit", f.TextValue)
					}
					if subject, ok := e.UnitCharacterSubject(); ok {
						if shown, _ := ui.PanelSubjectName(subject); shown != s.inspector.inspectionUTF8(name) {
							t.Fatalf("the card shows %q", shown)
						}
					} else if plain {
						t.Fatal("the basic view shows no card")
					}
					refused, rc := motionWindow(t, s, lang, plain, docked)
					typeText(t, refused, rc, unitNameField, strings.Repeat("x", maxUnitNameRunes+1))
					if string(s.workspace.Pages[unitNamesPage]) != line {
						t.Fatal("a name over the limit was written")
					}
					if !strings.Contains(strings.Join(refused.PopupLines(), "\n"), fmt.Sprint(maxUnitNameRunes)) {
						t.Fatalf("a long name refused without its limit: %q", refused.PopupLines())
					}
					typeText(t, e, c, unitNameField, "")
					if _, ok := s.workspace.Pages[unitNamesPage]; ok {
						t.Fatal("an emptied name left the page")
					}
					if _, err := s.Undo(); err != nil || string(s.workspace.Pages[unitNamesPage]) != line {
						t.Fatalf("Undo of the clear: page %q, %v", s.workspace.Pages[unitNamesPage], err)
					}
					if _, err := s.Undo(); err != nil {
						t.Fatal(err)
					}
					if _, ok := s.workspace.Pages[unitNamesPage]; ok || !bytes.Equal(s.model.Bytes(), raw) {
						t.Fatal("Undo of the name left the page or moved the map")
					}
				})
			}
		}
	}
}

// One Apply that sets a unit's name and its speed is one step: Undo takes
// back the names page and the record together.
func TestEditorUnitNameAndSpeedAreOneUndoStep(t *testing.T) {
	s, raw := viewsMap(t)
	if _, err := s.SetFields(ui.KindUnit, 0, []ui.InspectionFieldValue{{Key: unitNameField, Text: true, TextValue: "Bjorn"}, {Key: "speed", Value: 37}}); err != nil {
		t.Fatal(err)
	}
	rec, _ := s.model.UnitRecord(0)
	if rec[alm.UnitSpeed] != 37 || !strings.HasSuffix(string(s.workspace.Pages[unitNamesPage]), "=Bjorn\r\n") {
		t.Fatalf("speed %d page %q", rec[alm.UnitSpeed], s.workspace.Pages[unitNamesPage])
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.workspace.Pages[unitNamesPage]; ok || !bytes.Equal(s.model.Bytes(), raw) {
		t.Fatal("one Undo did not take back both")
	}
}

// unitSection is the payload of map b's type-6 section, the placed units'
// records, sharing b's bytes.
func unitSection(t *testing.T, b []byte) []byte {
	t.Helper()
	for off := 20; off+20 <= len(b); {
		size := int(binary.LittleEndian.Uint32(b[off+8:]))
		if binary.LittleEndian.Uint32(b[off+12:]) == 6 && off+20+size <= len(b) {
			return b[off+20 : off+20+size]
		}
		off += 20 + size
	}
	t.Fatal("the map has no type-6 section")
	return nil
}

// overrideUnitRecord is one placed unit with id, its kind and cell the
// authoring fixture's.
func overrideUnitRecord(t *testing.T, id uint16) []byte {
	t.Helper()
	b := synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Units: []synth.ALMUnit{{X: 9*256 + 128, Y: 10*256 + 128, ClassID: 40, ClassSubID: 2, UnitID: id}}})
	rec := bytes.Clone(unitSection(t, b)[:alm.UnitSize])
	rec[alm.UnitOwner] = 1
	return rec
}

// A map the editor opens and saves without an edit writes its bytes again
// and no texts folder.
func TestEditorUntouchedMapWritesByteEqual(t *testing.T) {
	dir := t.TempDir()
	b := synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Units: []synth.ALMUnit{{X: 3*256 + 128, Y: 4*256 + 77, ClassID: 40, ClassSubID: 2, UnitID: 4}}})
	unitSection(t, b)[alm.UnitOwner] = 1 // an owner in the map's roster
	source := filepath.Join(dir, "source.alm")
	if err := os.WriteFile(source, b, 0o600); err != nil {
		t.Fatal(err)
	}
	x := &MapInspector{tiles: &terrain.Tileset{}, table: authoringTable()}
	e := x.Editor()
	if err := e.Open(source); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	target := filepath.Join(dir, "copy.alm")
	s.AllowErrors() // the fixture's kinds are not an install's
	if _, err := s.SaveAs(target); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(target); err != nil || !bytes.Equal(got, b) {
		at := 0
		for at < min(len(got), len(b)) && got[at] == b[at] {
			at++
		}
		t.Fatalf("the untouched map wrote other bytes from %d (%d vs %d bytes): %v", at, len(got), len(b), err)
	}
	if _, err := os.Stat(mapTextFolder(target)); !os.IsNotExist(err) {
		t.Fatalf("the untouched map wrote a texts folder: %v", err)
	}
}

// A unit with all four overrides keeps them through Save As of the map, which
// writes the names page into the map's texts folder, and through the save
// into a mod package, which carries the page among its mission texts; the
// reopened package shows the name.
func TestEditorUnitOverridesSurviveSaveAsAndAMod(t *testing.T) {
	f, e := workspaceFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	if _, err := s.model.Place(mapedit.KindUnit, overrideUnitRecord(t, 5)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetFields(ui.KindUnit, 0, []ui.InspectionFieldValue{{Key: unitNameField, Text: true, TextValue: "Bjorn"},
		{Key: "speed", Value: 37}, {Key: "sight", Value: 7}, {Key: "facing", Value: 3}}); err != nil {
		t.Fatal(err)
	}
	authored := func(b []byte) bool {
		rec := unitSection(t, b)
		return len(rec) >= alm.UnitSize && rec[alm.UnitSpeed] == 37 && rec[alm.UnitSight] == 7 && rec[alm.UnitFacing] == 3
	}
	target := filepath.Join(t.TempDir(), "named.alm")
	s.AllowErrors() // the fixture's kinds are not an install's
	if _, err := s.SaveAs(target); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(target); err != nil || !authored(b) {
		t.Fatalf("Save As lost a record override: %v", err)
	}
	pages, _ := filepath.Glob(filepath.Join(mapTextFolder(target), "*", unitNamesPage))
	if len(pages) != 1 {
		t.Fatalf("Save As wrote names pages %v", pages)
	}
	if b, _ := os.ReadFile(pages[0]); string(b) != "5=Bjorn\r\n" {
		t.Fatalf("Save As wrote %q", b)
	}
	if err := workspaceField(e, "package", 1, "named"); err != nil {
		t.Fatal(err)
	}
	if err := workspaceField(e, "package", 2, "Named unit"); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(t.TempDir(), "named")
	if err := workspaceField(e, "package", 0, folder); err != nil {
		t.Fatal(err, s.Report())
	}
	m := loadWorkspaceMod(t, f.root, folder)
	if string(m.TextFiles[unitNamesPage]) != "5=Bjorn\r\n" || !authored(m.MapBytes) {
		t.Fatalf("the mod carries page %q, overrides %v", m.TextFiles[unitNamesPage], authored(m.MapBytes))
	}
	x2 := *f.x
	x2.mod = nil
	fresh := x2.Editor()
	if err := fresh.Open("new/32x32"); err != nil {
		t.Fatal(err)
	}
	if err := workspaceField(fresh, "package", 0, folder); err != nil {
		t.Fatal(err)
	}
	reopened := fresh.Document().Edits.(*mapInspectionEdits)
	if got := reopened.authoredUnitNames()[5]; got != "Bjorn" {
		t.Fatalf("the reopened package names unit 5 %q", got)
	}
}

// A mod's names page is served beside its other mission texts under the
// mission number, and the game reads it from there.
func TestModServesTheUnitNamesPage(t *testing.T) {
	f := guestFixtureFront(t)
	decl := guestDecl()
	decl.TextFiles[unitNamesPage] = []byte("5=Bjorn\r\n")
	if _, err := f.SetModMissions(mod.MissionData{Missions: []mod.Mission{decl}}); err != nil {
		t.Fatal(err)
	}
	src := f.Archives.Containers
	if got := missionUnitNames(src, InstallTextCode(src, tableEdition(f.Table)), 25); got[5] != "Bjorn" || len(got) != 1 {
		t.Fatalf("the game reads %v", got)
	}
}

// namedOverrideFront is the pool fixture whose map gives unit 91 speed 37,
// sight 7 and facing 3 (east) and whose names page calls it Bjorn.
func namedOverrideFront(t *testing.T) *FrontEnd {
	t.Helper()
	b := poolFixtureMap(91, 92)
	rec := unitSection(t, b)
	rec[alm.UnitSpeed], rec[alm.UnitSight], rec[alm.UnitFacing] = 37, 7, 3
	f := poolFixtureFrontMap(t, b)
	g := currentPoolFixtureFront(t, 91, 92)
	f.Table, f.Humans, f.Campaign = g.Table, g.Humans, g.Campaign
	f.Archives.Containers.Overlay(unitNamesSource.at(10).path, func() ([]byte, error) { return []byte("91=Bjorn\r\n"), nil })
	return f
}

// The game shows the map's name for the placed unit and spawns it with the
// authored speed, sight and facing byte; a SAVE and LOAD keeps all four, the
// name derived again from the names page by the unit's map id.
func TestPlacedUnitOverridesReachTheGameAndSurviveSAV(t *testing.T) {
	f := namedOverrideFront(t)
	if err := f.App("named overrides").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	check := func(f *FrontEnd, when string) {
		t.Helper()
		e, other := poolEntity(t, f.live.world, 91), poolEntity(t, f.live.world, 92)
		if got := f.live.shownName(e.ID); got != "Bjorn" {
			t.Fatalf("%s: unit 91 shows %q", when, got)
		}
		if f.live.shownName(other.ID) == "Bjorn" {
			t.Fatalf("%s: unit 92 took the name", when)
		}
		if e.Speed != 37 || e.ScanRange != 7 || e.Facing != 0x40 {
			t.Fatalf("%s: speed %d sight %d facing %#x, want 37, 7 and 0x40", when, e.Speed, e.ScanRange, e.Facing)
		}
	}
	check(f, "spawn")
	raw := currentRuntimeSave(t, f)
	cold := namedOverrideFront(t)
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal(town, err)
	}
	if err := cold.App("restored named overrides").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	check(cold, "LOAD")
}
