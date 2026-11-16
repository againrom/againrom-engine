package ui

import (
	"image"
	"slices"
	"testing"

	"againrom/pkg/words"
)

type charactersHost struct {
	*modesWorkspace
	full bool
	lang string
}

func (h charactersHost) WorkspaceFields(scope string) []InspectionField {
	if scope == ModCharactersScope {
		book := words.For(h.lang)
		if h.full {
			return []InspectionField{{Key: "name", Label: book.Text("editor.characters.name"), Meaning: book.Text("editor.characters.name.hint"), Text: true, TextValue: "Visitor"},
				{Key: "face", Label: book.Text("editor.characters.face"), Meaning: book.Text("editor.characters.face.hint"), Choices: []InspectionChoice{{Value: 0, Label: book.Text("editor.characters.inherit")}}},
				{Key: "remove", Label: book.Text("editor.characters.remove"), Meaning: book.Text("editor.characters.remove.hint"), Action: true, Column: 1}}
		}
		return []InspectionField{{Key: "new", Label: book.Text("editor.characters.add"), Meaning: book.Text("editor.characters.add.hint"), Choices: []InspectionChoice{{Value: 1, Label: "Guard"}}}}
	}
	return h.modesWorkspace.WorkspaceFields(scope)
}

func TestModCharactersPageTitleAndRowsHaveHelp(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e, host := modMapActionsEditor(t, lang, image.Pt(1280, 800))
		for _, full := range []bool{false, true} {
			e.doc.Edits = charactersHost{modesWorkspace: host, full: full, lang: lang}
			e.showModPage(modPageCharacters)
			s, _, ok := e.modSpec()
			want := e.editorHostText(e.plainWord("editor.characters.help") + "\n" + e.plainWord("editor.mod.keys"))
			if !ok || s.TitleHelp != want || layoutPopup(s, e.font, e.bodyFont()).TitleHelp.Empty() {
				t.Fatalf("%s full %v: Characters title help %q, want %q", lang, full, s.TitleHelp, want)
			}
			if len(e.obj.rowField) != len(e.obj.fields) || len(e.obj.fields) == 0 {
				t.Fatalf("%s full %v: fields did not reach the popup rows", lang, full)
			}
			for row, field := range e.obj.rowField {
				f := e.obj.fields[field]
				if f.Meaning == "" || s.Rows[row].Help != e.editorHostText(f.Meaning) {
					t.Errorf("%s full %v: field %s has popup help %q", lang, full, f.Key, s.Rows[row].Help)
				}
			}
		}
	}
}

func TestModWindowShowsCharactersThroughTheScopeTable(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	host := e.doc.Edits.(*modesWorkspace)
	e.doc.Edits = charactersHost{modesWorkspace: host}
	_ = e.Key("package")
	tabs, _ := e.modTabs()
	if len(tabs) != 2 || tabs[1] != "Characters" {
		t.Fatalf("tabs %v", tabs)
	}
	e.turnModPage(1)
	if e.modPage != modPageCharacters || e.modFieldScope() != ModCharactersScope || len(e.obj.fields) != 1 {
		t.Fatalf("characters page %d scope %q fields %+v", e.modPage, e.modFieldScope(), e.obj.fields)
	}
	e.doc.Edits = charactersHost{modesWorkspace: host, full: true}
	e.showModPage(modPageCharacters)
	spec, _, ok := e.modSpec()
	if !ok || !slices.ContainsFunc(spec.Rows, func(row popupRow) bool { return slices.Contains(row.Buttons, "Delete character") }) {
		t.Fatalf("the popup row builder lost the delete button: %+v", spec.Rows)
	}
}
