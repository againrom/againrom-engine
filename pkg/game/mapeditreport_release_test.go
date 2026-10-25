package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func installedInspector(t *testing.T) *MapInspector {
	t.Helper()
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: needs lawful install")
	}
	x, err := NewMapInspector(root)
	if err != nil {
		t.Fatal(err)
	}
	return x
}

// scriptedUnit is the index of a unit a script names by its id, whose id no
// other unit shares, with that id.
func scriptedUnit(t *testing.T, m *alm.Map) (int, uint16) {
	t.Helper()
	sc, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	named := map[uint32]bool{}
	for _, nodes := range [][]alm.ScriptNode{sc.Actions, sc.Conditions} {
		for _, n := range nodes {
			for p, tag := range n.Type {
				if tag == 4 && n.Value[p] < 10001 {
					named[n.Value[p]] = true
				}
			}
		}
	}
	count := map[uint16]int{}
	for _, u := range m.Units {
		count[u.UnitID]++
	}
	for i, u := range m.Units {
		if named[uint32(u.UnitID)] && count[u.UnitID] == 1 && u.UnitID != 0 {
			return i, u.UnitID
		}
	}
	t.Fatal("no unit is named by a script")
	return 0, 0
}

func recordOf(d *ui.InspectionDocument, kind string, index int) int {
	for i, r := range d.Records {
		if r.Kind == kind && r.Index == index {
			return i
		}
	}
	return -1
}

func TestReleaseMapEditorRefusesASaveThatLeavesAScriptDangling(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	e := x.Editor()
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	m, err := alm.Open(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	unit, id := scriptedUnit(t, m)
	installed := append([]byte(nil), e.Document().SourceBytes...)
	if errs, _ := ui.CountFindings(e.Document().Edits.(ui.InspectionChecker).Report()); errs != 0 {
		t.Fatalf("the installed map reports %d errors", errs)
	}
	loot, err := m.Loot()
	if err != nil {
		t.Fatal(err)
	}
	stock := -1
	for i, record := range loot.Records {
		if record.Owner == uint32(id) {
			if stock >= 0 {
				t.Fatalf("scripted unit %d has more than one owned stock", id)
			}
			stock = i
		}
	}
	if stock >= 0 {
		row := recordOf(e.Document(), "Stock", stock)
		if row < 0 {
			t.Fatalf("owned Stock #%d is missing from the editor", stock)
		}
		e.Select(row, false)
		if err := e.Key("delete"); err != nil || !e.Dirty() {
			t.Fatalf("delete owned stock: %v", err)
		}
		model := e.Document().Edits.(*mapInspectionEdits).model
		unitRecord, err := model.UnitRecord(unit)
		if err != nil || binary.LittleEndian.Uint32(unitRecord[0x18:]) != 0 {
			t.Fatalf("owned stock removal did not clear the unit link: %v", err)
		}
		after, err := model.Map()
		if err != nil || len(after.Units) != len(m.Units) || after.Units[unit].UnitID != id {
			t.Fatalf("stock removal changed the scripted unit: %v", err)
		}
		remaining, err := after.Loot()
		if err != nil || len(remaining.Records) != len(loot.Records)-1 {
			t.Fatalf("stock removal changed another loot record: %v", err)
		}
		for i, record := range remaining.Records {
			original := i
			if original >= stock {
				original++
			}
			if !reflect.DeepEqual(record, loot.Records[original]) {
				t.Fatalf("stock removal changed surviving loot record %d", original)
			}
		}
	}
	if errs, _ := ui.CountFindings(e.Document().Edits.(ui.InspectionChecker).Report()); errs != 0 {
		t.Fatalf("stock cleanup reports %d errors before deleting the scripted unit", errs)
	}
	postStock := bytes.Clone(e.Document().SourceBytes)
	e.Select(recordOf(e.Document(), "Unit", unit), false)
	if err := e.Key("delete"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(e.PromptNotes()) > 0 {
		r, ok := e.PromptButtonRect(0)
		(&editorClicker{t: t}).click(e, r, ok, "Empty it")
	}
	if !e.Dirty() {
		t.Fatal("delete changed nothing")
	}
	dir := filepath.Join(t.TempDir(), "mod")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "edited.alm")
	_ = e.Key("saveas")
	e.TextInput(target)
	err = e.Key("enter")
	refused, ok := err.(*ui.ReportRefusal)
	if !ok {
		t.Fatalf("Save As of the map with a dangling reference: %v", err)
	}
	found := false
	for _, f := range refused.Findings {
		found = found || f.Error && f.Rule == "script-reference" && strings.Contains(f.Text, "unit "+strconv.Itoa(int(id)))
	}
	if _, statErr := os.Stat(target); statErr == nil || !found {
		t.Fatalf("file written %v, finding for unit %d %v: %+v", statErr == nil, id, found, refused.Findings)
	}
	if rows, open := e.ReportOpen(); !open || len(rows) != len(refused.Findings) || !e.Dirty() {
		t.Fatal("the refusal did not keep the report on screen")
	}
	// Selecting the finding selects the script record that names the unit.
	for n, f := range refused.Findings {
		if f.Rule == "script-reference" {
			for e.ReportSelected() < n {
				_ = e.Key("down")
			}
			if sel := e.Document().Records[e.Selected()]; sel.Kind != f.Kind || sel.Index != f.Index {
				t.Fatalf("the finding names %s #%d, the window selected %s #%d", f.Kind, f.Index, sel.Kind, sel.Index)
			}
			break
		}
	}
	if err := e.Key("undo"); err != nil || e.Dirty() != (stock >= 0) || !bytes.Equal(e.Document().SourceBytes, postStock) {
		t.Fatalf("undo: %v", err)
	}
	if stock >= 0 {
		if err := e.Key("undo"); err != nil || e.Dirty() || !bytes.Equal(e.Document().SourceBytes, installed) {
			t.Fatalf("undo stock cleanup: %v", err)
		}
	}
	_ = e.Key("saveas")
	e.TextInput(target)
	if err := e.Key("enter"); err != nil {
		t.Fatalf("Save As after the undo: %v", err)
	}
	if got, err := os.ReadFile(target); err != nil || !bytes.Equal(got, installed) {
		t.Fatalf("saved map: %v", err)
	}
}

func TestReleaseMapEditorSavesAUnitOnGroundItCannotEnterAndListsTheWarning(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	e := x.Editor()
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	s := e.Document().Edits.(*mapInspectionEdits)
	m, err := s.model.Map()
	if err != nil {
		t.Fatal(err)
	}
	w, err := mapload.FromALMWith(m, x.table, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	plane := mapload.Passability(m)
	unit, blocked := -1, -1
	for i := range m.Units {
		if unit < 0 && w.Entities()[i].Domain == sim.DomainGround {
			unit = i
		}
	}
	for c, b := range plane {
		cx, cy := c%m.Width, c/m.Width
		if b&1 != 0 && cx > 8 && cy > 8 && cx < m.Width-8 && cy < m.Height-8 {
			blocked = c
			break
		}
	}
	if unit < 0 || blocked < 0 {
		t.Fatalf("unit %d blocked cell %d", unit, blocked)
	}
	cell := image.Pt(blocked%m.Width, blocked/m.Width)
	if _, err := s.Move("Unit", unit, cell); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "moved.alm")
	_ = e.Key("saveas")
	e.TextInput(target)
	if err := e.Key("enter"); err != nil {
		t.Fatalf("Save As with only a warning: %v", err)
	}
	if got, err := os.ReadFile(target); err != nil || !bytes.Equal(got, s.model.Bytes()) {
		t.Fatalf("saved map: %v", err)
	}
	rows, open := e.ReportOpen()
	listed := false
	for _, f := range rows {
		listed = listed || !f.Error && f.Rule == "unit-terrain" && f.Kind == "Unit" && f.Index == unit && f.Cell == cell
	}
	if !open || !listed {
		t.Fatalf("report open %v, unit-terrain finding for unit %d at %v listed %v", open, unit, cell, listed)
	}
}

func TestReleaseEveryInstalledMapReportsNoError(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	warnings := map[string]int{}
	for _, entry := range x.Maps {
		key := "loose/" + entry.Source
		if entry.FromArchive {
			key = "scenario/" + entry.Source
		}
		d, err := x.Open(key)
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		for _, f := range d.Edits.(ui.InspectionChecker).Report() {
			if f.Error {
				t.Errorf("%s: %s", key, f.Line())
			} else {
				warnings[f.Rule]++
			}
		}
	}
	var names []string
	for rule := range warnings {
		names = append(names, rule)
	}
	sort.Strings(names)
	for _, rule := range names {
		t.Logf("%d maps: warnings %s = %d", len(x.Maps), rule, warnings[rule])
	}
}

func TestReleaseMapEditorHelpLinesFitTheRail(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	for _, line := range ui.EditorHelpLinesIn(x.archives.Base.Profile.Language) {
		if width := x.font.Advance(x.inspectionUTF8(line)); width > 300 {
			t.Errorf("%q is %d wide; the rail holds 300", line, width)
		}
	}
}
