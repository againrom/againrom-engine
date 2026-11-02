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
// own values, the bag's sack row linking it, and an undo history.
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
	case KindUnit:
		var out []InspectionField
		for _, f := range h.wideHost.Fields(KindUnit, index) {
			if f.Key == "carry1" {
				out = append(out, InspectionField{Key: "sack", ReadOnly: true, Section: "inventory", Label: "Open the unit's sack (2 items)", Link: RecordRef{KindStock, 0}})
			}
			out = append(out, f)
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

// pressSackRow scrolls the sack row into the window and presses it.
func pressSackRow(t *testing.T, e *MapEditor) {
	t.Helper()
	n := sackRow(e)
	if n < 0 {
		t.Fatalf("the unit window shows no sack row:\n%s", strings.Join(e.PopupLines(), "\n"))
	}
	r, ok := e.PopupRowRect(n)
	for step := 0; !ok && step < 80; step++ {
		e.ScrollDetails(1)
		r, ok = e.PopupRowRect(n)
	}
	if !ok {
		t.Fatal("the sack row does not come into the window")
	}
	chromeClick(e, center(r))
}

// A unit's sack opens from the unit's bag in the window a selected sack
// takes: in the right panel, in the profile's view, the sack's plain table
// or its technical rows; the bag edits none of the sack's fields; Back
// returns to the unit, on the tab it left.
func TestMapEditorUnitSackOpensInTheSackView(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, view := range []string{"plain", "technical"} {
			t.Run(lang+"/"+view, func(t *testing.T) {
				e, _ := sackEditor(t, lang, view)
				openUnitBag(t, e, view)
				if !e.panels.hosted || !e.objectInPanel() {
					t.Fatal("the unit's bag stands in a second floating window")
				}
				lines := strings.Join(e.PopupLines(), "\n")
				if strings.Contains(lines, "Carried item") || strings.Contains(lines, "Add carried item") || !strings.Contains(lines, "Bag: Mace") {
					t.Fatalf("the bag shows\n%s", lines)
				}
				pressSackRow(t, e)
				if e.selected != sackStock {
					t.Fatalf("the sack row selects %d", e.selected)
				}
				if view == "plain" {
					clickSwitch(t, e) // the sack's basic view
				}
				// The same window as the stock selected on its own, but for
				// Back; and the same view as the ground sack.
				twin, _ := sackEditor(t, lang, view)
				twin.Select(sackStock, false)
				a, _, okA := e.objectSpec()
				b, _, okB := twin.objectSpec()
				if !okA || !okB || !e.panels.hosted || rowsText(a) != rowsText(b) {
					t.Fatalf("the sack from its unit\n%s\nthe sack selected\n%s", rowsText(a), rowsText(b))
				}
				back := e.word("editor.trigger.back")
				if a.Buttons[len(a.Buttons)-1] != back || fmt.Sprint(b.Buttons) != fmt.Sprint(a.Buttons[:len(a.Buttons)-1]) {
					t.Fatalf("buttons from the unit %q, selected %q", a.Buttons, b.Buttons)
				}
				if e.PlainView() != (view == "plain") {
					t.Fatalf("the sack shows plain %v in the %s profile", e.PlainView(), view)
				}
				blocks := e.PlainFieldBlocks()
				twin.Select(viewsSack, false)
				if fmt.Sprint(blocks) != fmt.Sprint(twin.PlainFieldBlocks()) {
					t.Fatalf("the unit's sack's blocks %v, the ground sack's %v", blocks, twin.PlainFieldBlocks())
				}
				r, ok := e.PopupButtonRect(len(a.Buttons) - 1)
				if !ok {
					t.Fatal("no Back button")
				}
				chromeClick(e, center(r))
				// Back shows the unit in the view the sack showed: the bag's
				// tab in the detailed view, no bag in the basic view.
				if e.selected != viewsUnit || view == "technical" && (e.obj.section != 2 || sackRow(e) < 0) || view == "plain" && sackRow(e) >= 0 {
					t.Fatalf("Back shows record %d, section %d", e.selected, e.obj.section)
				}
			})
		}
	}
}

// An edit made in the unit's sack's window undoes and redoes as the same
// edit made in the stock's window selected on its own: the same values and
// the same window after each step, Back kept.
func TestMapEditorUnitSackEditUndoes(t *testing.T) {
	for _, view := range []string{"plain", "technical"} {
		t.Run(view, func(t *testing.T) {
			e, h := sackEditor(t, "en", view)
			openUnitBag(t, e, view)
			pressSackRow(t, e)
			if view == "plain" {
				clickSwitch(t, e) // the sack's basic view
			}
			twin, th := sackEditor(t, "en", view)
			twin.Select(sackStock, false)
			for _, m := range []*MapEditor{e, twin} {
				n := -1
				for i, f := range m.ObjectFields() {
					if f.Key == "gold" {
						n = i
					}
				}
				r, ok := m.FieldRect(n)
				if !ok {
					t.Fatalf("the gold field does not show:\n%s", strings.Join(m.PopupLines(), "\n"))
				}
				chromeClick(m, center(r))
				for range 4 {
					_ = m.Key("backspace")
				}
				m.TextInput("77")
				r, _ = m.PopupButtonRect(0)
				chromeClick(m, center(r))
			}
			step := func(what string, gold int64) {
				t.Helper()
				if h.values["Stock/gold"] != gold || th.values["Stock/gold"] != gold {
					t.Fatalf("%s: gold %d, the selected stock's %d, want %d", what, h.values["Stock/gold"], th.values["Stock/gold"], gold)
				}
				a, _, _ := e.objectSpec()
				b, _, _ := twin.objectSpec()
				if e.selected != sackStock || twin.selected != sackStock || rowsText(a) != rowsText(b) || len(e.objectBack) != 1 {
					t.Fatalf("%s: the windows differ, back %d\n%s\n%s", what, len(e.objectBack), rowsText(a), rowsText(b))
				}
			}
			step("Apply", 77)
			_ = e.Key("undo")
			_ = twin.Key("undo")
			step("undo", 12)
			_ = e.Key("redo")
			_ = twin.Key("redo")
			step("redo", 77)
		})
	}
}
