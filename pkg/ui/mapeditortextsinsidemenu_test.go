package ui

import "testing"

// textsInsideMenuEdits is any host with the switch added: on only for an
// extended map, which ExtendScript makes.
type textsInsideMenuEdits struct {
	InspectionEdits
	extended, inside bool
	extends          int
}

func (p *textsInsideMenuEdits) TextsInside() bool { return p.inside && p.extended }
func (p *textsInsideMenuEdits) SetTextsInside(on bool) error {
	if on && !p.extended {
		return &ScriptFormNeeded{Why: "only an extended map holds its texts"}
	}
	p.inside = on
	return nil
}
func (p *textsInsideMenuEdits) ExtendScript() (*InspectionDocument, error) {
	p.extended = true
	p.extends++
	return nil, nil
}
func (p *textsInsideMenuEdits) FoldScript() (*InspectionDocument, error) {
	p.extended = false
	return nil, nil
}

// The File menu holds the switch as a row of the command table, after Save
// as, in EN and RU: a switch row with its mark, no key. On an original map
// the row asks the script form question and Yes extends the map once and
// turns the switch on, with no save window. The save window's row shows the
// same state and changes it, and the menu row follows. A host without the
// switch shows the row disabled.
func TestFileMenuRowSwitchesTheTextsInsideTheMap(t *testing.T) {
	names := map[string]string{}
	for _, lang := range []string{"", "russian"} {
		e, _, _ := menuEditor(t, lang)
		plain := e.doc.Edits
		host := &textsInsideMenuEdits{InspectionEdits: plain}
		e.doc.Edits = host
		name := e.word("editor.menu.textsinside")
		names[lang] = name
		row := func() menuItem {
			t.Helper()
			items := e.menuItems("file")
			n := menuKeyRow(t, e, "file", textsInsideKey)
			if items[n-1].key != "saveto" {
				t.Fatalf("%q: the row stands after %q", lang, items[n-1].key)
			}
			return items[n]
		}
		press := func() {
			t.Helper()
			e.toggleMenu("file")
			chromeClick(e, menuPoint(t, e, menuKeyRow(t, e, "file", textsInsideKey)))
		}
		if it := row(); it.off || it.label != markWord(false, name) || name == "" {
			t.Fatalf("%q: the row reads %q, disabled %v", lang, it.label, it.off)
		}
		press()
		if e.prompt.kind != promptScriptForm || host.extends != 0 || host.TextsInside() {
			t.Fatalf("%q: an original map: kind %v extends %d", lang, e.prompt.kind, host.extends)
		}
		e.scriptFormGo()
		if it := row(); host.extends != 1 || !host.TextsInside() || e.prompt.open() || it.label != markWord(true, name) {
			t.Fatalf("%q: after Yes: extends %d inside %v kind %v row %q", lang, host.extends, host.TextsInside(), e.prompt.kind, it.label)
		}

		e.openPrompt(editorPrompt{kind: promptSave, path: "bridge.almx"})
		window, at, ok := textsInsideSwitch(e)
		if !ok || window.Value != e.word("editor.save.textsinside.on") {
			t.Fatalf("%q: the save window's row %+v shown %v", lang, window, ok)
		}
		modesClick(e, at)
		if window, _, _ := textsInsideSwitch(e); host.TextsInside() || e.prompt.kind != promptSave || window.Value != e.word("editor.save.textsinside.off") {
			t.Fatalf("%q: the window's press: inside %v row %+v", lang, host.TextsInside(), window)
		}
		e.leave(leavePrompt)
		if it := row(); it.label != markWord(false, name) {
			t.Fatalf("%q: the menu row after the window's press reads %q", lang, it.label)
		}
		press()
		if it := row(); !host.TextsInside() || host.extends != 1 || e.prompt.open() || it.label != markWord(true, name) {
			t.Fatalf("%q: an extended map: inside %v extends %d kind %v", lang, host.TextsInside(), host.extends, e.prompt.kind)
		}
		press()
		if host.TextsInside() || host.extends != 1 {
			t.Fatalf("%q: the second press: inside %v extends %d", lang, host.TextsInside(), host.extends)
		}

		e.doc.Edits = plain
		if it := row(); !it.off {
			t.Fatalf("%q: a host without the switch shows the row choosable", lang)
		}
		for _, r := range editorKeyRows {
			if r.key == textsInsideKey {
				t.Fatalf("%q: the row has a key row", lang)
			}
		}
	}
	if names[""] == names["russian"] {
		t.Fatalf("the row reads %q in EN and RU", names[""])
	}
}
