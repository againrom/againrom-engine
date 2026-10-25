package game

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// errorMap is a map whose report holds one error.
func errorEdits(t *testing.T, source string) *mapInspectionEdits {
	t.Helper()
	m := built(t, cleanOptions(), nil, func(b []byte, payload func(uint32) int) {
		binary.LittleEndian.PutUint32(b[payload(6)+0x14:], 5)
	})
	return &mapInspectionEdits{inspector: checkInspector(), model: m, source: source, saved: m.Bytes()}
}

func asksToSave(err error) bool {
	var r *ui.ReportRefusal
	return errors.As(err, &r) && r.Ask
}

func TestSaveAsOfAMapWithAnErrorAsksThenWrites(t *testing.T) {
	s := errorEdits(t, "scenario/20.alm")
	target := filepath.Join(t.TempDir(), "out.alm")
	if _, err := s.SaveAs(target); !asksToSave(err) {
		t.Fatalf("Save As: %v", err)
	}
	if _, err := os.Stat(target); err == nil {
		t.Fatal("the question wrote a file")
	}
	s.AllowErrors()
	got, err := s.SaveAs(target)
	if err != nil || got != target {
		t.Fatalf("confirmed Save As: %q %v", got, err)
	}
	if b, err := os.ReadFile(target); err != nil || string(b) != string(mustSaveBytes(t, s)) {
		t.Fatalf("written map: %v", err)
	}
	// The answer covers one save.
	if _, err := s.SaveAs(filepath.Join(t.TempDir(), "again.alm")); !asksToSave(err) {
		t.Fatalf("second Save As: %v", err)
	}
}

func TestSaveIntoModOfAMapWithAnErrorAsksThenWrites(t *testing.T) {
	dir := writeEditorMod(t, t.TempDir(), editorModMissions)
	s := errorEdits(t, "scenario/20.alm")
	if err := s.inspector.SetMod(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveIntoMod(); !asksToSave(err) {
		t.Fatalf("Save into the mod: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "mission")); err == nil {
		t.Fatal("the question wrote into the mod")
	}
	s.AllowErrors()
	got, err := s.SaveIntoMod()
	if err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(got); err != nil || string(b) != string(mustSaveBytes(t, s)) {
		t.Fatalf("written map: %v", err)
	}
}

func TestSaveAsOfAMapWithWarningsOnlyDoesNotAsk(t *testing.T) {
	s := errorEdits(t, "scenario/20.alm")
	if err := s.model.SetUnitField(0, "owner", 1); err != nil {
		t.Fatal(err)
	}
	if err := s.gate(true); err != nil {
		t.Fatalf("gate: %v", err)
	}
	if _, err := s.SaveAs(filepath.Join(t.TempDir(), "ok.alm")); err != nil {
		t.Fatal(err)
	}
}

func TestSaveAsWritesTheMapWhenTheTextsFolderCannotBeRead(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "map.alm")
	folder := filepath.Join(dir, "map.text")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "Event01.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := built(t, cleanOptions(), nil, nil)
	s := &mapInspectionEdits{inspector: checkInspector(), model: m, source: source, saved: m.Bytes()}
	target := filepath.Join(t.TempDir(), "out.alm")
	if _, err := s.SaveAs(target); !asksToSave(err) {
		t.Fatalf("the texts-folder error did not ask: %v", err)
	}
	s.AllowErrors()
	got, err := s.SaveAs(target)
	if err != nil || got != target {
		t.Fatalf("Save As: %q %v", got, err)
	}
	if b, err := os.ReadFile(target); err != nil || string(b) != string(mustSaveBytes(t, s)) {
		t.Fatalf("the map file was not written: %v", err)
	}
	if note := s.SaveNote(); !strings.Contains(note, "Texts not written") || !strings.Contains(note, "Event01.txt") {
		t.Fatalf("note %q", note)
	}
}

func TestPackageAndTestPlayRefuseAMapWithAnErrorAndSaySaveWorks(t *testing.T) {
	s := errorEdits(t, "scenario/20.alm")
	err := s.gate(false)
	var r *ui.ReportRefusal
	if !errors.As(err, &r) || r.Ask || !strings.Contains(err.Error(), "Save still works") {
		t.Fatalf("gate for the package: %v", err)
	}
	s.AllowErrors()
	if err := s.gate(false); err == nil {
		t.Fatal("the save answer lifted the package refusal")
	}
}

func TestSaveIntoAnOwnedPackageWithAnErrorAsksThenWritesTheMapFile(t *testing.T) {
	f, e := workspaceFixture(t)
	if err := workspaceField(e, "package", 1, "owned-save"); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "owned-save")
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	if !f.x.mod.owned {
		t.Fatal("the package is not owned")
	}
	f.x.rules = []mapRule{{ID: "always", Error: true, run: func(*mapCheck) []mapHit { return []mapHit{{Text: "an error"}} }}}
	s := e.Document().Edits.(*mapInspectionEdits)
	s.reported, s.reportedFor = nil, nil
	if _, err := s.SaveIntoMod(); !asksToSave(err) {
		t.Fatalf("Save into the owned package: %v", err)
	}
	s.AllowErrors()
	got, err := s.SaveIntoMod()
	if err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(got); err != nil || string(b) != string(mustSaveBytes(t, s)) {
		t.Fatalf("written map file: %v", err)
	}
	if !strings.Contains(s.SaveNote(), "map file") {
		t.Fatalf("note %q", s.SaveNote())
	}
}
