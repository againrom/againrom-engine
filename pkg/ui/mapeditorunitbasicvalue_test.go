package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"
)

// derivedHost is the card fixture whose card states a body the game derives
// from the stored one (shift more than typed) and a defence it applies
// whatever is typed (an NPC record's), with the host's meaning line on it.
type derivedHost struct {
	*cardHost
	shift int
	npc   bool
}

func (h *derivedHost) UnitSettings(index int, values []InspectionFieldValue) (MapEntity, []InspectionField, *InspectionUnitPreview, error) {
	entity, fields, preview, err := h.cardHost.UnitSettings(index, values)
	entity.Char.Body += h.shift
	if h.npc {
		entity.Combat.Defence = 99
		fields = append([]InspectionField(nil), fields...)
		for i := range fields {
			if fields[i].Key == "defence" {
				fields[i].Meaning = "Ignored by this unit; in game: 99"
			}
		}
	}
	return entity, fields, preview, err
}

func derivedEditor(t *testing.T, lang string, panel bool, shift int, npc bool) *MapEditor {
	t.Helper()
	e, h := cardEditor(t, lang, 300, panel)
	d := &derivedHost{cardHost: h, shift: shift, npc: npc}
	e.open = func(string) (*InspectionDocument, error) {
		doc := wideDocument(h.wideHost)
		doc.Edits = d
		return doc, nil
	}
	if err := e.Open("views"); err != nil {
		t.Fatal(err)
	}
	e.Layout(1920, 1080)
	e.Select(viewsUnit, false)
	return e
}

// shownWords are the lines of text the basic view's rows show, not the
// hover lines.
func shownWords(e *MapEditor) string {
	s, _, _ := e.objectSpec()
	var out []string
	for _, row := range s.Rows {
		if row.Kind == popupText && !row.placed() {
			out = append(out, row.Label)
		}
	}
	return strings.Join(strings.Fields(strings.Join(out, " ")), " ")
}

// An input on the card replaces the card's value: the box holds the input's
// own rendering and no pixel of the card's value, whatever the card states.
func TestMapEditorUnitBasicViewInputCoversTheCardValue(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, panel := range []bool{false, true} {
			interior := func(shift int) *image.RGBA {
				e := derivedEditor(t, lang, panel, shift, false)
				typeCardField(t, e, "body", "15")
				r, ok := e.FieldRect(cardFieldIndex(t, e, "body"))
				if !ok {
					t.Fatalf("%s: no body input", lang)
				}
				pic := e.ChromeImage()
				box := r.Inset(2)
				out := image.NewRGBA(image.Rect(0, 0, box.Dx(), box.Dy()))
				for y := range box.Dy() {
					for x := range box.Dx() {
						out.Set(x, y, pic.At(box.Min.X+x, box.Min.Y+y))
					}
				}
				return out
			}
			a, b := interior(0), interior(20)
			if string(a.Pix) != string(b.Pix) {
				t.Fatalf("%s panel=%v: the card's value shows inside the body input", lang, panel)
			}
		}
	}
}

// Where the card's value is the game's and differs from the stored one, the
// rows say so in words, naming the game's value; equal values add none.
func TestMapEditorUnitBasicViewStatesTheGamesValue(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, panel := range []bool{false, true} {
			e := derivedEditor(t, lang, panel, 20, false)
			derived := e.word("editor.plain.derived")
			ingame := strings.Fields(strings.TrimSpace(derived[:strings.Index(derived, "%s")]))
			typeCardField(t, e, "body", "15")
			if got := shownWords(e); !strings.Contains(got, strings.Join(strings.Fields(fmt.Sprintf(derived, "35")), " ")) {
				t.Fatalf("%s panel=%v: derived body words %q", lang, panel, got)
			}
			e = derivedEditor(t, lang, panel, 0, false)
			typeCardField(t, e, "body", "15")
			if got := shownWords(e); strings.Contains(got, strings.Join(ingame, " ")) {
				t.Fatalf("%s panel=%v: equal values add words %q", lang, panel, got)
			}
			e = derivedEditor(t, lang, panel, 0, true)
			typeCardField(t, e, "defence", "55")
			if got := shownWords(e); !strings.Contains(got, "Ignored by this unit; in game: 99") {
				t.Fatalf("%s panel=%v: NPC arm words %q", lang, panel, got)
			}
		}
	}
}
