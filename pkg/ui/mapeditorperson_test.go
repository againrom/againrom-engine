package ui

import (
	"bytes"
	"fmt"
	"image"
	"strconv"
	"testing"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/words"
)

type personSettingsProbe struct{ *settingsPreviewProbe }

func (p *personSettingsProbe) Fields(kind string, index int) []InspectionField {
	fields := p.settingsPreviewProbe.Fields(kind, index)
	for i := range fields {
		fields[i].Section = "identity"
	}
	for slot := 1; slot <= 12; slot++ {
		fields = append(fields, InspectionField{Key: fmt.Sprintf("wear%d", slot), Label: fmt.Sprintf("Slot %d", slot), Section: "equipment", Value: -1, Choices: []InspectionChoice{{Value: -1, Label: "Inherit"}, {Value: 257, Label: "Long Sword"}}})
	}
	for i := 0; i < 24; i++ {
		fields = append(fields, InspectionField{Key: fmt.Sprintf("stat%d", i), Label: fmt.Sprintf("Stat %d", i), Section: "stats", Inherit: true})
	}
	return append(fields, InspectionField{Key: "carryadd", Label: "Add carried item", Section: "inventory", Choices: []InspectionChoice{{Value: 0, Label: "Choose"}, {Value: 257, Label: "Long Sword"}}}, InspectionField{Key: "body", Section: "stats", Label: "Body", Inherit: true})
}

func (p *personSettingsProbe) UnitSettings(index int, values []InspectionFieldValue) (MapEntity, []InspectionField, *InspectionUnitPreview, error) {
	entity, _, pic, err := p.settingsPreviewProbe.UnitSettings(index, values)
	fields := p.Fields("Unit", index)
	for _, v := range values {
		for i := range fields {
			if fields[i].Key == v.Key {
				fields[i].Value = v.Value
			}
		}
	}
	return entity, fields, pic, err
}

func TestEditorPersonWindowFitsAndPaintsBothLanguages(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, size := range []image.Point{{1280, 800}, {800, 600}, {640, 480}, {1024, 768}, {1280, 1024}, {1920, 1080}} {
			t.Run(lang+size.String(), func(t *testing.T) {
				e, host := popupEditor(t)
				_ = e.Open("popups")
				e.doc.Edits = &personSettingsProbe{&settingsPreviewProbe{popupHost: host}}
				e.SetWords(words.For(lang))
				e.SetHostTextEncoder(func(s string) string {
					encoded, _ := textinput.EncodeText(s, e.font.Selector, textinput.Substitute)
					return encoded
				})
				e.Layout(size.X, size.Y)
				e.Select(0, false)
				raw := bytes.Clone(e.doc.SourceBytes)
				for section := 0; section < 4; section++ {
					e.obj.section = section
					s, _, ok := e.objectSpec()
					l := layoutPopup(s, e.font, e.bodyFont())
					if !ok || !l.Rect.In(image.Rectangle{Max: size}) || len(l.Tabs) != 4 || len(l.Buttons) != 3 || s.Card == nil {
						t.Fatal("person layout", size, section, l.Rect)
					}
					for _, r := range append(l.Buttons, l.Tabs...) {
						if !r.In(l.Rect) {
							t.Fatal("pinned control outside popup", r)
						}
					}
					if size.X == 1280 && section == 0 {
						for slot := 1; slot <= 12; slot++ {
							key := fmt.Sprintf("wear%d", slot)
							at := -1
							for i, f := range e.obj.fields {
								if f.Key == key {
									at = i
								}
							}
							r, ok := e.FieldRect(at)
							if !ok || r.Empty() {
								t.Fatal("wide setup hides slot", slot)
							}
						}
					}
					pic := e.ChromeImage()
					if pic == nil || pic.Bounds().Size() != size {
						t.Fatal("person painter size")
					}
					doll := image.NewRGBA(pic.Bounds())
					e.drawObjectPreview(doll, s)
					painted := false
					for at := 0; at < len(doll.Pix); at += 4 {
						painted = painted || doll.Pix[at+3] != 0
					}
					if !painted {
						t.Fatal("pinned doll not painted", size, section)
					}
				}
				e.obj.section = 0
				e.objectSpec()
				e.obj.buf[0] = "42"
				e.objectSpec()
				if len(e.obj.pending) != 1 || e.obj.buf[0] != strconv.Itoa(42) || !bytes.Equal(raw, e.doc.SourceBytes) || len(host.calls) != 0 {
					t.Fatal("native draft preview wrote or lost its step")
				}
				e.obj.section = 3
				e.objectSpec()
				body := -1
				for i, f := range e.obj.fields {
					if f.Key == "body" {
						body = i
					}
				}
				e.obj.buf[body] = "12"
				e.objectSpec()
				e.objectButton(2)
				e.objectSpec()
				if e.obj.buf[body] != "" {
					t.Fatal("Reset overrides did not restore inheritance")
				}
				_ = e.Key("escape")
				if len(e.obj.pending) != 0 || !bytes.Equal(raw, e.doc.SourceBytes) || len(host.calls) != 0 {
					t.Fatal("native Cancel retained or wrote a draft")
				}
			})
		}
	}
}

func TestEditorPersonKeyboardChoosesBeforeApplying(t *testing.T) {
	e, host := popupEditor(t)
	_ = e.Open("popups")
	e.doc.Edits = &personSettingsProbe{&settingsPreviewProbe{popupHost: host}}
	e.Select(0, false)
	_ = e.Key("field")
	if e.prompt.open() || !e.objectTyping() {
		t.Fatal("E did not open the person window")
	}
	e.obj.section = 1
	e.objectSpec()
	for i, f := range e.obj.fields {
		if f.Key == "wear1" {
			e.obj.focus = i
		}
	}
	_ = e.Key("enter")
	if !e.dropdownOpen() {
		t.Fatal("Enter did not open the item dropdown")
	}
	e.TextInput("sWoRd")
	_ = e.Key("enter")
	if e.dropdownOpen() || len(host.calls) != 0 {
		t.Fatal("choosing applied the whole object")
	}
	e.objectSpec()
	if len(e.obj.pending) != 1 || e.obj.pending[0].Value != 257 {
		t.Fatal("keyboard did not stage item")
	}
	_ = e.Key("applyobject")
	if len(host.calls) != 1 {
		t.Fatal("Ctrl+Enter did not apply the draft once", host.calls)
	}
}

func TestEditorPersonTabKeepsFocusVisibleAcrossSections(t *testing.T) {
	e, host := popupEditor(t)
	_ = e.Open("popups")
	e.doc.Edits = &personSettingsProbe{&settingsPreviewProbe{popupHost: host}}
	e.Layout(640, 480)
	e.Select(0, false)
	_ = e.Key("field")
	seen := map[string]bool{}
	for range 60 {
		_ = e.Key("tab")
		e.objectSpec()
		if e.obj.focus < 0 {
			t.Fatal("Tab lost focus")
		}
		f := e.obj.fields[e.obj.focus]
		seen[f.Section] = true
		if _, ok := e.FieldRect(e.obj.focus); !ok {
			t.Fatal("Tab focused a scrolled-out field", f.Key)
		}
	}
	if len(seen) != 4 {
		t.Fatal("keyboard cannot reach every section", seen)
	}
}

func TestEditorPersonNumericDraftKeepsTypingUndo(t *testing.T) {
	e, host := popupEditor(t)
	_ = e.Open("popups")
	e.doc.Edits = &personSettingsProbe{&settingsPreviewProbe{popupHost: host}}
	e.Select(0, false)
	_ = e.Key("field")
	original := e.obj.buf[e.obj.focus]
	_ = e.Key("selectall")
	e.TextInput("42")
	e.objectSpec()
	_ = e.Key("textundo")
	e.objectSpec()
	if e.obj.buf[e.obj.focus] != original || len(host.calls) != 0 {
		t.Fatal("draft rebuild lost local typing Undo")
	}
	e.SetHostTextEncoder(func(s string) string { return s }, func(s string) string {
		if s == "encoded" {
			return "Броня"
		}
		return s
	})
	p := NewPicker([]PickerRow{{Text: "encoded", Choosable: true}})
	if !p.Find("бРоНя", e.editorSearchText) {
		t.Fatal("picker bypassed the host font decoder")
	}
}
