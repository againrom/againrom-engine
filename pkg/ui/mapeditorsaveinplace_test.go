package ui

import (
	"testing"

	"againrom/pkg/pathdialog"
)

// ownFileEdits is a host whose map may have a file of its own.
type ownFileEdits struct {
	reportEdits
	own     string
	inPlace []string
}

func (p *ownFileEdits) OwnFile() (string, bool) { return p.own, p.own != "" }
func (p *ownFileEdits) SaveInPlace() (string, error) {
	p.inPlace = append(p.inPlace, p.own)
	p.dirty = false
	return p.own, nil
}
func (p *ownFileEdits) SaveAs(path string) (string, error) {
	p.own = path
	return p.reportEdits.SaveAs(path)
}

func TestSaveWritesTheOwnFileWithoutAPromptAndSaveAsAsks(t *testing.T) {
	edits := &ownFileEdits{reportEdits: reportEdits{dirty: true}}
	e := reportEditor(t, edits)
	asked := 0
	bindEditorDialog(e, func(pathdialog.Request) (pathdialog.Result, error) {
		asked++
		return pathdialog.Result{Path: "chosen.alm"}, nil
	}, t.TempDir())
	// No file of its own: Save asks, then the chosen file is the own file.
	_ = e.Key("saveas")
	finishEditorDialog(t, e)
	if asked != 1 || edits.saves != 1 || e.doc.Source != "chosen.alm" {
		t.Fatalf("first Save: asked %d saves %d source %q", asked, edits.saves, e.doc.Source)
	}
	edits.dirty = true
	_ = e.Key("saveas")
	finishEditorDialog(t, e)
	if asked != 1 || len(edits.inPlace) != 1 || edits.saves != 1 || e.Dirty() {
		t.Fatalf("second Save: asked %d in place %v saves %d dirty %v", asked, edits.inPlace, edits.saves, e.Dirty())
	}
	// Save as keeps asking.
	e.menuAct("saveto")
	finishEditorDialog(t, e)
	if asked != 2 || edits.saves != 2 {
		t.Fatalf("Save as: asked %d saves %d", asked, edits.saves)
	}
}
