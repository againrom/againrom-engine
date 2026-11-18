package ui

import (
	"image"
	"strings"
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
	if !ok || row.Value != e.word("editor.switch.off") {
		t.Fatalf("the switch row %+v shown %v", row, ok)
	}
	modesClick(e, at)
	if e.prompt.kind != promptScriptForm || host.extends != 0 || host.inside {
		t.Fatalf("an original map: kind %v extends %d", e.prompt.kind, host.extends)
	}
	e.scriptFormGo()
	row, at, ok = textsInsideSwitch(e)
	if host.extends != 1 || !host.TextsInside() || e.prompt.kind != promptSave || e.prompt.path != "bridge.almx" ||
		!ok || row.Value != e.word("editor.switch.on") {
		t.Fatalf("after Yes: extends %d inside %v kind %v path %q row %+v", host.extends, host.TextsInside(), e.prompt.kind, e.prompt.path, row)
	}
	modesClick(e, at)
	if row, _, _ := textsInsideSwitch(e); host.TextsInside() || host.extends != 1 || e.prompt.kind != promptSave || row.Value != e.word("editor.switch.off") {
		t.Fatalf("the second press: inside %v extends %d row %+v", host.TextsInside(), host.extends, row)
	}
	e.doc.Edits = edits
	if _, _, ok := textsInsideSwitch(e); ok {
		t.Fatal("a host without the switch shows the row")
	}
}

// The script form question shows the host's refusal whole as its note and
// its own text behind the title's (?), so the player reads what the
// extended map lifts.
func TestScriptFormQuestionShowsTheRefusalWhole(t *testing.T) {
	e, edits := selectedTextEditor(t, panelFont(), image.Pt(1280, 800))
	e.doc.Edits = &textsInsideEdits{editorTextEdits: edits}
	why := "No slot is free for a new condition: it would take the slot of variable 50"
	if !e.askScriptForm(&ScriptFormNeeded{Why: why}, func() {}) || e.prompt.kind != promptScriptForm {
		t.Fatalf("the refusal did not open the question: kind %v", e.prompt.kind)
	}
	notes := strings.Join(strings.Fields(strings.Join(e.PromptNotes(), " ")), " ")
	if s, _ := e.promptSpec(); notes != why || !strings.HasPrefix(s.TitleHelp, e.word("editor.scriptform.title.help")) {
		t.Fatalf("the question reads %q with help %q, want %q", notes, s.TitleHelp, why)
	}
}
