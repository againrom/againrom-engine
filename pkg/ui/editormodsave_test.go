package ui

import (
	"errors"
	"testing"
)

// modSaveEdits is a host that saves into a mod named target.
type modSaveEdits struct {
	editorTextEdits
	target string
	saves  int
	err    error
}

func (p *modSaveEdits) ModTarget() string { return p.target }
func (p *modSaveEdits) SaveIntoMod() (string, error) {
	p.saves++
	if p.err != nil {
		return "", p.err
	}
	return "mods/x/mission/en/map.alm", nil
}

func modSaveEditor(t *testing.T, edits *modSaveEdits) *MapEditor {
	t.Helper()
	doc := &InspectionDocument{Viewer: litViewer(t), Source: "scenario/20.alm", Edits: edits}
	e := NewMapEditor(nil, nil, func(string) (*InspectionDocument, error) { return doc, nil })
	if err := e.Open("scenario/20.alm"); err != nil {
		t.Fatal(err)
	}
	return e
}

func TestSaveGoesIntoTheModTheHostNames(t *testing.T) {
	edits := &modSaveEdits{target: "x"}
	e := modSaveEditor(t, edits)
	if got := e.saveHelp(); got != "Ctrl+Z / Ctrl+Y / Ctrl+S: save into mod x" {
		t.Fatalf("help %q", got)
	}
	if err := e.Key("saveas"); err != nil {
		t.Fatal(err)
	}
	if edits.saves != 1 || edits.writes != 0 || e.pathMode || e.doc.Source != "mods/x/mission/en/map.alm" || e.message != "" {
		t.Fatalf("saves %d writes %d path %v source %q message %q", edits.saves, edits.writes, e.pathMode, e.doc.Source, e.message)
	}
	edits.err = errors.New(`no mission of mod "x" names this map`)
	if err := e.Key("saveas"); err == nil || e.message != edits.err.Error() || e.doc.Source != "mods/x/mission/en/map.alm" {
		t.Fatalf("refusal: %v, message %q", err, e.message)
	}
}

func TestSaveAsAsksForAPathWithoutAMod(t *testing.T) {
	edits := &modSaveEdits{}
	e := modSaveEditor(t, edits)
	if got := e.saveHelp(); got != "Ctrl+Z / Ctrl+Y / Ctrl+S" {
		t.Fatalf("help %q", got)
	}
	if err := e.Key("saveas"); err != nil {
		t.Fatal(err)
	}
	if edits.saves != 0 || !e.pathMode || !e.pathSave {
		t.Fatalf("saves %d path %v save %v", edits.saves, e.pathMode, e.pathSave)
	}
}
