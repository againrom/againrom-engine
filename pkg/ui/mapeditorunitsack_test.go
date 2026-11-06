package ui

import (
	"fmt"
	"image"
	"maps"
	"strings"
	"testing"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/words"
)

// sackHost is the wide fixture whose unit carries its own sack: the stock
// record after the script records, with the ground sack's fields under its
// own values, and an undo history.
type sackHost struct {
	*wideHost
	past, future []map[string]int64
}

// sackStock is the stock's record index in the fixture's document.
const sackStock = 5

func (h *sackHost) Fields(kind RecordKind, index int) []InspectionField {
	switch kind {
	case KindStock:
		out := h.wideHost.Fields(KindSack, index)
		for i := range out {
			out[i].Value = h.value(KindStock, out[i].Key)
		}
		return out
	}
	return h.wideHost.Fields(kind, index)
}

func (h *sackHost) UnitSettings(index int, values []InspectionFieldValue) (MapEntity, []InspectionField, *InspectionUnitPreview, error) {
	entity, _, preview, err := h.wideHost.UnitSettings(index, values)
	fields := h.Fields(KindUnit, index)
	for _, v := range values {
		for i := range fields {
			if fields[i].Key == v.Key {
				fields[i].Value = v.Value
			}
		}
	}
	return entity, fields, preview, err
}

func (h *sackHost) SetField(kind RecordKind, index int, key string, value int64) (*InspectionDocument, error) {
	return h.SetFields(kind, index, []InspectionFieldValue{{Key: key, Value: value}})
}

func (h *sackHost) SetFields(kind RecordKind, index int, values []InspectionFieldValue) (*InspectionDocument, error) {
	h.past, h.future = append(h.past, maps.Clone(h.values)), nil
	if _, err := h.viewsHost.SetFields(kind, index, values); err != nil {
		return nil, err
	}
	return sackDocument(h), nil
}

func (h *sackHost) CanUndo() bool { return len(h.past) > 0 }
func (h *sackHost) CanRedo() bool { return len(h.future) > 0 }

func (h *sackHost) Undo() (*InspectionDocument, error) {
	h.future = append(h.future, h.values)
	h.values, h.past = h.past[len(h.past)-1], h.past[:len(h.past)-1]
	return sackDocument(h), nil
}

func (h *sackHost) Redo() (*InspectionDocument, error) {
	h.past = append(h.past, h.values)
	h.values, h.future = h.future[len(h.future)-1], h.future[:len(h.future)-1]
	return sackDocument(h), nil
}

func sackDocument(h *sackHost) *InspectionDocument {
	d := wideDocument(h.wideHost)
	d.Edits = h
	d.Records = append(d.Records, InspectionRecord{Kind: KindStock, Index: 0, Section: 8, Label: "owner 1", Size: image.Pt(1, 1)})
	return d
}

// sackEditor is the editor over a sackHost in lang at 1920x1080, the right
// panel restored at its default width, in the profile's view.
func sackEditor(t *testing.T, lang, view string) (*MapEditor, *sackHost) {
	t.Helper()
	inner, _ := viewsEditor(t, lang)
	h := &sackHost{wideHost: &wideHost{viewsHost: inner.doc.Edits.(*viewsHost)}}
	h.values["Stock/gold"], h.values["Stock/item1"], h.values["Stock/item2"] = 12, 0x101, 0x102
	e := NewMapEditor(wordsFont(lang), []EditorMap{{Key: "views", Label: "Views"}}, func(string) (*InspectionDocument, error) { return sackDocument(h), nil })
	e.SetWords(words.For(lang))
	e.SetHostTextEncoder(func(s string) string {
		encoded, _ := textinput.EncodeText(s, e.font.Selector, textinput.Substitute)
		return encoded
	})
	e.SetPanelProfile([]byte(fmt.Sprintf(`{"left":{"minimized":false,"width":260},"right":{"minimized":false,"width":520,"view":%q}}`, view)), nil, nil)
	e.Layout(1920, 1080)
	if err := e.Open("views"); err != nil {
		t.Fatal(err)
	}
	return e, h
}

// sackRow is the object window's row that opens the unit's sack, -1 for
// none.
func sackRow(e *MapEditor) int {
	s, links, ok := e.objectSpec()
	if !ok {
		return -1
	}
	for n, r := range s.Rows {
		if r.Kind == popupLink && n < len(links) && links[n] == -(sackStock+1) {
			return n
		}
	}
	return -1
}

// openUnitBag selects the unit and shows its bag: the Bag tab of the
// detailed view. The basic view shows no bag; from it the switch turns the
// window to the detailed view first.
func openUnitBag(t *testing.T, e *MapEditor, view string) {
	t.Helper()
	e.Select(viewsUnit, false)
	if view == "plain" {
		if sackRow(e) >= 0 || strings.Contains(strings.Join(e.PopupLines(), "\n"), "Bag: Mace") {
			t.Fatalf("the basic view shows the bag:\n%s", strings.Join(e.PopupLines(), "\n"))
		}
		clickSwitch(t, e)
	}
	{
		r, ok := e.ObjectSectionRect("inventory")
		if !ok {
			t.Fatal("the person window shows no Bag tab")
		}
		chromeClick(e, center(r))
		if e.obj.section != 2 {
			t.Fatalf("the Bag tab shows section %d", e.obj.section)
		}
	}
	if _, _, ok := e.objectSpec(); !ok {
		t.Fatal("the unit shows no window")
	}
}

// The unit's bag, in the detailed view, is the pick that adds an item to
// the unit's sack and the sack's items; no row of it opens the sack, and
// the basic view shows no bag. The unit's sack, selected, opens in the
// window a selected sack takes, in the right panel.
func TestMapEditorUnitBagIsThePickAndTheItems(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, view := range []string{"plain", "technical"} {
			t.Run(lang+"/"+view, func(t *testing.T) {
				e, _ := sackEditor(t, lang, view)
				openUnitBag(t, e, view)
				s, _, _ := e.objectSpec()
				keys := map[string]bool{}
				for n := range s.Rows {
					if i, ok := e.obj.rowField[n]; ok {
						keys[e.obj.fields[i].Key] = true
					}
				}
				if !keys["carryadd"] || !keys["carry1"] || !keys["carry2"] || sackRow(e) >= 0 {
					t.Fatalf("the bag edits %v, sack row %d:\n%s", keys, sackRow(e), strings.Join(e.PopupLines(), "\n"))
				}
				e.Select(sackStock, false)
				if _, _, ok := e.objectSpec(); !ok || e.selected != sackStock || !e.panels.hosted || !e.objectInPanel() {
					t.Fatal("the unit's sack, selected, stands in no sack window in the right panel")
				}
			})
		}
	}
}
