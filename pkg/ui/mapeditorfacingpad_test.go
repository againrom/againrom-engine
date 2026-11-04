package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/words"
)

type facingPadHost struct {
	*modesFull
	model   *mapedit.Editor
	choices []InspectionChoice
	pad     []image.Point
}

func (h *facingPadHost) Fields(kind RecordKind, index int) []InspectionField {
	if kind != KindUnit {
		return h.modesFull.Fields(kind, index)
	}
	v, _ := h.model.FieldValue(mapedit.KindUnit, 0, "facing")
	return []InspectionField{{Key: "facing", Label: "Facing", Value: v, Choices: h.choices, ChoicePad: h.pad}, {Key: "speed", Label: "Speed", Value: 0}}
}

func (h *facingPadHost) SetFields(kind RecordKind, index int, values []InspectionFieldValue) (*InspectionDocument, error) {
	sets := []mapedit.FieldSet{}
	for _, v := range values {
		sets = append(sets, mapedit.FieldSet{Key: v.Key, Value: v.Value})
	}
	if err := h.model.SetFields(mapedit.KindUnit, 0, sets...); err != nil {
		return nil, err
	}
	return h.doc(h), nil
}

func (h *facingPadHost) SetField(kind RecordKind, index int, key string, value int64) (*InspectionDocument, error) {
	return h.SetFields(kind, index, []InspectionFieldValue{{Key: key, Value: value}})
}

func (h *facingPadHost) NewRecordStep(r InspectionNewRecord, key string) InspectionNewStep {
	if key == "facing" {
		return InspectionNewStep{Choices: h.choices, ChoicePad: h.pad}
	}
	return h.modesFull.NewRecordStep(r, key)
}

func facingPadEditor(t *testing.T, lang string) (*MapEditor, *facingPadHost) {
	t.Helper()
	e, _ := newModesEditor(t, "full")
	m, err := mapedit.New(synth.ALM(synth.ALMOptions{Width: 50, Height: 50, Units: []synth.ALMUnit{{X: 10 * 256, Y: 10 * 256, ClassID: 40}}}))
	if err != nil {
		t.Fatal(err)
	}
	h := &facingPadHost{modesFull: e.doc.Edits.(*modesFull), model: m, pad: []image.Point{{0, 0}, {0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}}}
	w := words.For(words.Language(lang))
	for n := range 9 {
		key := fmt.Sprintf("editor.unit.facing.%d", n)
		if n == 0 {
			key = "editor.unit.facing.random"
		}
		h.choices = append(h.choices, InspectionChoice{Value: int64(n), Label: w.Text(key)})
	}
	h.self, e.doc.Edits = h, h
	e.SetWords(w)
	e.Select(0, false)
	return e, h
}

func facingPadRow(t *testing.T, e *MapEditor, add bool) (popupSpec, popupRowRect) {
	t.Helper()
	var s popupSpec
	row := -1
	if add {
		var acts []addRowAct
		s, acts = e.addSpec()
		for n, a := range acts {
			if a.asked {
				row = n
			}
		}
		s = e.sizedPrompt(s)
	} else {
		s, _, _ = e.objectSpec()
		row = e.fieldRow(0)
	}
	for _, rr := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if rr.Index == row {
			if len(rr.Buttons) != 9 {
				t.Fatalf("facing has %d buttons, want a nine-button pad (kind %d)", len(rr.Buttons), s.Rows[row].Kind)
			}
			return s, rr
		}
	}
	t.Fatal("facing row is not shown")
	return s, popupRowRect{}
}

func TestEditorFacingPadUsesCompassButtons(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, add := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/add=%v", lang, add), func(t *testing.T) {
				e, h := facingPadEditor(t, lang)
				if add {
					e.openAdd(KindUnit)
					for i := 0; i < 8; i++ {
						key, _, ok := e.AddStep()
						if !ok {
							t.Fatal("Add closed")
						}
						if key == "facing" {
							break
						}
						addAdvance(t, e, 1)
					}
				}
				_, rr := facingPadRow(t, e, add)
				for n, pos := range h.pad {
					b := rr.Buttons[n]
					if b.Size() != rr.Buttons[0].Size() || b.Dx() != b.Dy() {
						t.Fatal("buttons are not equal squares")
					}
					if center(b).Sub(center(rr.Buttons[0])) != pos.Mul(b.Dx()+2) {
						t.Fatalf("value %d is at %v", n, b)
					}
				}
				for n := range 9 {
					_, rr = facingPadRow(t, e, add)
					modesClick(e, center(rr.Buttons[n]))
					value := e.prompt.path
					if !add {
						value = e.obj.buf[0]
					}
					if value != strconv.Itoa(n) {
						t.Fatalf("button %d chose %q", n, value)
					}
					s, selected := facingPadRow(t, e, add)
					l := layoutPopup(s, e.font, e.bodyFont())
					if s.Rows[selected.Index].Selected != n {
						t.Fatal("pressed choice did not follow the value")
					}
					var buttons []pushButton
					widgetRecord = func(c widgetCall) {
						if c.kind == widgetPushButton {
							for _, b := range selected.Buttons {
								if c.rect == b {
									buttons = append(buttons, c.state.(pushButton))
								}
							}
						}
					}
					drawPopup(image.NewRGBA(image.Rect(0, 0, e.window.X, e.window.Y)), e.font, e.bodyFont(), nil, s, l, editorPopupInk, image.Point{}, false)
					widgetRecord = nil
					if len(buttons) != 9 {
						t.Fatalf("the shared button painter drew %d pad buttons", len(buttons))
					}
					for k, b := range buttons {
						if b.sunk() != (k == n) {
							t.Fatalf("button %d pressed=%v after value %d", k, b.sunk(), n)
						}
					}
					tip, ok := popupTipAt(image.Rect(0, 0, e.window.X, e.window.Y), e.font, e.bodyFont(), s, l, center(selected.Buttons[n]), true)
					if !ok || strings.Join(tip.Lines, " ") != h.choices[n].Label {
						t.Fatalf("value %d tip %v, want %q", n, tip, h.choices[n].Label)
					}
				}
				_ = e.Key("up")
				if add && e.prompt.path != "7" || !add && e.obj.buf[0] != "7" {
					t.Fatal("Up did not keep the choice keyboard order")
				}
				_ = e.Key("down")
				if add && e.prompt.path != "8" || !add && e.obj.buf[0] != "8" {
					t.Fatal("Down did not keep the choice keyboard order")
				}
			})
		}
	}
}

func TestEditorFacingPadKeepsSavedBytesAndCommandChoices(t *testing.T) {
	for n := range 9 {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			e, h := facingPadEditor(t, "en")
			before := h.model.Bytes()
			_, rr := facingPadRow(t, e, false)
			modesClick(e, center(rr.Buttons[n]))
			e.applyObject()
			want, _ := mapedit.New(before)
			if err := want.SetField(mapedit.KindUnit, 0, "facing", int64(n)); err != nil {
				t.Fatal(err)
			}
			got, err := h.model.SaveBytes()
			if err != nil {
				t.Fatal(err)
			}
			old, err := want.SaveBytes()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, old) {
				t.Fatal("pad bytes differ from the drop-down setter")
			}
			rec, err := h.model.UnitRecord(0)
			if err != nil || rec[alm.UnitFacing] != byte(n) {
				t.Fatalf("saved facing: %x %v", rec, err)
			}
			if n != 0 && (!h.model.Undo() || !bytes.Equal(h.model.Bytes(), before) || h.model.Undo()) {
				t.Fatal("pad changed the undo boundary")
			}
			read, err := e.readRecord(json.RawMessage(`{"kind":"Unit","index":0,"verbose":true}`))
			if err != nil {
				t.Fatal(err)
			}
			fields := read.(map[string]any)["fields"].([]InspectionField)
			if len(fields[0].Choices) != 9 {
				t.Fatal("command lost choices")
			}
			for k, c := range fields[0].Choices {
				if c.Value != int64(k) {
					t.Fatal("command choice order moved")
				}
			}
			_ = e.Key("escape")
			for _, c := range editorCommandRows() {
				if c.name == "set_field" {
					_, err := c.mcpRun(e, json.RawMessage(fmt.Sprintf(`{"kind":"Unit","index":0,"key":"facing","value":%d}`, n)))
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			v, err := h.model.FieldValue(mapedit.KindUnit, 0, "facing")
			if err != nil || v != int64(n) {
				t.Fatal("command did not set the facing", v, err)
			}
		})
	}
}

func TestEditorFacingPadMovesOnlyFollowingRows(t *testing.T) {
	e, h := facingPadEditor(t, "en")
	for _, columns := range []int{1, 3} {
		old := popupSpec{Width: 1200, Columns: columns, At: image.Pt(10, 10), Rows: []popupRow{
			{Kind: popupEdit, Label: "Before"},
			{Kind: popupChoice, Label: "Direction"},
			{Kind: popupEdit, Label: "After"},
			{Kind: popupEdit, Label: "Other column", Column: columns - 1},
		}}
		oldLayout := layoutPopup(old, e.font, e.bodyFont())
		pad := old
		pad.Rows = append([]popupRow(nil), old.Rows...)
		pad.Rows[1] = pad.Rows[1].withChoices(h.choices, h.pad, "0")
		got := layoutPopup(pad, e.font, e.bodyFont())
		for n, r := range got.Rows {
			before := oldLayout.Rows[n]
			switch {
			case n == 1:
				if r.Rect.Min != before.Rect.Min || r.Label != before.Label || r.Rect.Dy() != before.Rect.Dy()+2*got.Pitch {
					t.Fatal("pad changed its anchor or label", columns, r, before)
				}
			case n == 0 || columns > 1 && n == 3:
				if r.Rect != before.Rect || r.Control != before.Control {
					t.Fatal("a row before the pad or in another column moved", columns, n)
				}
			default:
				if r.Rect != before.Rect.Add(image.Pt(0, 2*got.Pitch)) || r.Control != before.Control.Add(image.Pt(0, 2*got.Pitch)) {
					t.Fatal("a following row moved by more than the pad height", columns, n)
				}
			}
		}
	}
	plain := popupRow{Kind: popupChoice}.withChoices(h.choices, nil, "0")
	if plain.Kind != popupChoice {
		t.Fatal("choices without pad data changed their control")
	}
}

func TestEditorFacingPadFitsSizedRows(t *testing.T) {
	e, h := facingPadEditor(t, "en")
	height := fontHeight(e.font, 15) + popupRowExtra + 4*(fontHeight(e.bodyFont(), 15)+popupRowExtra) + 4 + 3*popupPad
	for _, columns := range []int{1, 3} {
		for top := 0; top <= 2; top++ {
			s := popupSpec{Width: 900, Size: image.Pt(900, height), Bounds: image.Rect(0, 0, 900, height), Columns: columns, MaxRows: 5, Top: top,
				Rows: []popupRow{{Kind: popupEdit, Label: "Before"}, {Kind: popupEdit, Label: "Before"},
					(popupRow{Label: "Facing"}).withChoices(h.choices, h.pad, "0"), {Kind: popupEdit, Label: "After"}, {Kind: popupEdit, Label: "Last"}},
				Buttons: []string{"Apply", "Cancel"}}
			l := layoutPopup(s, e.font, e.bodyFont())
			if !l.Rect.In(s.Bounds) {
				t.Fatalf("columns %d top %d: pad pushed the popup outside its bounds: %v", columns, top, l.Rect)
			}
			for _, r := range l.Buttons {
				if !r.In(s.Bounds) {
					t.Fatal("Apply or Cancel left the window", columns, top, r)
				}
			}
			shown := false
			for _, r := range l.Rows {
				if r.Index == 2 {
					shown = true
					for _, b := range r.Buttons {
						if !b.In(l.Rect) {
							t.Fatal("pad button cut", columns, top, b)
						}
					}
				}
			}
			if top == 2 && !shown {
				t.Fatal("the pad cannot be reached as the first shown row", columns)
			}
		}
	}
}

func TestEditorFacingPadRowsAndTipStayInViewport(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, surface := range []string{"plain", "detailed", "add"} {
			t.Run(lang+"/"+surface, func(t *testing.T) {
				e, h := facingPadEditor(t, lang)
				e.font = wordsFont(lang)
				pad := (popupRow{Label: "Facing", Value: h.choices[3].Label}).withChoices(h.choices, h.pad, "3")
				s := popupSpec{Title: "Unit", At: image.Pt(10, 10), Width: 900, Top: 1, MaxRows: 1,
					Rows:  []popupRow{{Kind: popupText, Label: "Before"}, pad, {Kind: popupEdit, Label: "Following row", Value: "123"}},
					Notes: []string{"Notes stay below the rows"}, Buttons: []string{"Apply", "Cancel"}}
				if surface == "plain" {
					s.Card = &PanelSubject{Name: "Card title"}
					s.CardFont = cardFont()
					s.CardBelow = true
				}
				if surface == "detailed" {
					s.Columns = 3
				}
				if surface == "add" {
					e.openAdd(KindUnit)
					for n := 0; n < 8; n++ {
						key, _, ok := e.AddStep()
						if !ok {
							t.Fatal("Add closed")
						}
						if key == "facing" {
							break
						}
						addAdvance(t, e, 1)
					}
					var acts []addRowAct
					s, acts = e.addSpec()
					for n, a := range acts {
						if a.asked {
							s.Top = n
						}
					}
					s.MaxRows = 1
				}
				l := layoutPopup(s, e.font, e.bodyFont())
				viewport := image.Rect(l.Rect.Min.X+popupPad, l.Title.Max.Y+popupPad, l.Rect.Max.X-popupPad, l.Title.Max.Y+popupPad+3*l.Pitch)
				var button image.Rectangle
				for _, r := range l.Rows {
					if s.Rows[r.Index].placed() {
						continue
					}
					boxes := append([]image.Rectangle{r.Rect, r.Label, r.Control, r.Button, r.Grip}, r.Buttons...)
					for _, box := range boxes {
						if !box.Empty() && !box.In(viewport) {
							t.Errorf("row %d hit/paint rectangle %v outside %v", r.Index, box, viewport)
						}
					}
					if s.Rows[r.Index].Kind == popupDirection {
						button = r.Buttons[3]
					}
				}
				if button.Empty() {
					t.Fatal("pad is not scrolled into view")
				}
				ground := l
				ground.Rows = nil
				bounds := l.Rect.Inset(-2)
				for _, pointer := range []image.Point{bounds.Min, center(button)} {
					got, want := image.NewRGBA(bounds), image.NewRGBA(bounds)
					ink := editorPopupInk
					ink.Label = color.RGBA{255, 0, 255, 255}
					drawPopup(want, e.font, e.bodyFont(), nil, s, ground, ink, pointer, true)
					drawPopup(got, e.font, e.bodyFont(), nil, s, l, ink, pointer, true)
					for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
						for x := bounds.Min.X; x < bounds.Max.X; x++ {
							p := image.Pt(x, y)
							if p.In(viewport) {
								continue
							}
							part, _ := l.hit(p)
							if part == popupRowLine || part == popupRowControl || part == popupRowButton || part == popupRowGrip {
								t.Fatalf("row hit outside viewport at %v", p)
							}
							if got.RGBAAt(x, y) != want.RGBAAt(x, y) {
								t.Fatalf("row or hover tip painted outside viewport at %v (pointer %v)", p, pointer)
							}
						}
					}
				}
			})
		}
	}
}
