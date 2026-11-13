package ui

import (
	"image"
	"testing"
)

// textsInsideEdits is a host with the save window's switch: on only for an
// extended map, which ExtendScript makes.
type textsInsideEdits struct {
	*editorTextEdits
	extended, inside bool
	extends          int
}

func (p *textsInsideEdits) TextsInside() bool { return p.inside && p.extended }
func (p *textsInsideEdits) SetTextsInside(on bool) error {
	if on && !p.extended {
		return &ScriptFormNeeded{Why: "only an extended map holds its texts"}
	}
	p.inside = on
	return nil
}
func (p *textsInsideEdits) ExtendScript() (*InspectionDocument, error) {
	p.extended = true
	p.extends++
	return nil, nil
}
func (p *textsInsideEdits) FoldScript() (*InspectionDocument, error) {
	p.extended = false
	return nil, nil
}

// textsInsideSwitch is the save window's switch row and the middle of its
// button; false when the window shows none.
func textsInsideSwitch(e *MapEditor) (popupRow, image.Point, bool) {
	s, ok := e.promptSpec()
	if !ok {
		return popupRow{}, image.Point{}, false
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	for _, r := range l.Rows {
		if s.Rows[r.Index].Kind == popupSwitch {
			return s.Rows[r.Index], center(r.Control), true
		}
	}
	return popupRow{}, image.Point{}, false
}

// The save window shows the switch off by default. On an original map a
// press asks the script form question; Yes extends the map once, turns the
// switch on and shows the window again with the typed .alm name made .almx.
// A second press turns it off and extends nothing. A host without the
// switch shows no row.
func TestSaveWindowSwitchPutsTheTextsInsideTheMap(t *testing.T) {
	e, edits := selectedTextEditor(t, panelFont(), image.Pt(1280, 800))
	host := &textsInsideEdits{editorTextEdits: edits}
	e.doc.Edits = host
	if err := e.Key("saveas"); err != nil || e.prompt.kind != promptSave {
		t.Fatalf("the save window: %v kind %v", err, e.prompt.kind)
	}
	e.TextInput("bridge.alm")
	row, at, ok := textsInsideSwitch(e)
	if !ok || row.Value != e.word("editor.save.textsinside.off") {
		t.Fatalf("the switch row %+v shown %v", row, ok)
	}
	modesClick(e, at)
	if e.prompt.kind != promptScriptForm || host.extends != 0 || host.inside {
		t.Fatalf("an original map: kind %v extends %d", e.prompt.kind, host.extends)
	}
	e.scriptFormGo()
	row, at, ok = textsInsideSwitch(e)
	if host.extends != 1 || !host.TextsInside() || e.prompt.kind != promptSave || e.prompt.path != "bridge.almx" ||
		!ok || row.Value != e.word("editor.save.textsinside.on") {
		t.Fatalf("after Yes: extends %d inside %v kind %v path %q row %+v", host.extends, host.TextsInside(), e.prompt.kind, e.prompt.path, row)
	}
	modesClick(e, at)
	if row, _, _ := textsInsideSwitch(e); host.TextsInside() || host.extends != 1 || e.prompt.kind != promptSave || row.Value != e.word("editor.save.textsinside.off") {
		t.Fatalf("the second press: inside %v extends %d row %+v", host.TextsInside(), host.extends, row)
	}
	e.doc.Edits = edits
	if _, _, ok := textsInsideSwitch(e); ok {
		t.Fatal("a host without the switch shows the row")
	}
}
