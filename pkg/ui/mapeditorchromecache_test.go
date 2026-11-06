package ui

import (
	"image"
	"testing"
	"time"
)

// A frame whose chrome is the picture and text the last frame settled is not
// settled again: the second call answers the first call's upload.
func TestMapEditorChromeSettledOnceWhileUnchanged(t *testing.T) {
	e, _ := selectedTextEditor(t, panelFont(), image.Pt(1280, 800))
	_, first, calls, _ := e.chromePicture()
	if len(calls) == 0 {
		t.Fatal("fixture chrome captured no text")
	}
	for range 3 {
		_, again, _, _ := e.chromePicture()
		if again != first {
			t.Fatal("an unchanged chrome was settled again")
		}
	}
}

// Every kind of change the chrome shows settles it again, and the settled
// frame is the exact frame a fresh composition gives.
func TestMapEditorChromeSettledAgainOnEachChange(t *testing.T) {
	e, _ := selectedTextEditor(t, panelFont(), image.Pt(1280, 800))
	size := e.window
	changes := []struct {
		name string
		do   func()
	}{
		{"a panel opens", func() {
			if err := e.Key("reportpanel"); err != nil {
				t.Fatal(err)
			}
		}},
		{"a prompt opens", func() {
			if err := e.Key("saveas"); err != nil {
				t.Fatal(err)
			}
		}},
		{"text is typed", func() { e.TextInput("ab") }},
		{"the caret moves", func() {
			if err := e.Key("left"); err != nil {
				t.Fatal(err)
			}
		}},
		{"the pointer hovers a bar button", func() {
			r := e.barEntries()[0].rect
			e.Pointer(Input{CursorX: r.Min.X + 2, CursorY: r.Min.Y + 2}, time.Now())
		}},
		{"the window resizes", func() { e.Layout(1024, 768); size = e.window }},
		{"the prompt closes", func() {
			if err := e.Key("escape"); err != nil {
				t.Fatal(err)
			}
		}},
	}
	_, last, _, _ := e.chromePicture()
	for _, c := range changes {
		c.do()
		pic, blit, _, _ := e.chromePicture()
		if blit == last {
			t.Fatalf("%s: the chrome was not settled again", c.name)
		}
		fresh := e.ChromeImage()
		if string(pic.Pix) != string(fresh.Pix) || pic.Bounds().Size() != size {
			t.Fatalf("%s: the chrome is not what a fresh composition draws", c.name)
		}
		assertEditorPanelUpload(t, e, size.Y)
		_, again, _, _ := e.chromePicture()
		if again != blit {
			t.Fatalf("%s: the changed chrome was settled twice", c.name)
		}
		last = blit
	}
}

type countedUnitHost struct {
	*personSettingsProbe
	builds int
}

func (h *countedUnitHost) UnitSettings(index int, values []InspectionFieldValue) (MapEntity, []InspectionField, *InspectionUnitPreview, error) {
	h.builds++
	entity, fields, pic, err := h.personSettingsProbe.UnitSettings(index, values)
	for _, v := range values {
		for i := range fields {
			if v.Text && fields[i].Key == v.Key {
				fields[i].TextValue = v.TextValue
			}
		}
	}
	return entity, fields, pic, err
}

// Typing three characters into the unit window's name builds the preview once,
// after the typing pauses; the typed text is in the field at once.
func TestEditorUnitNameTypingBuildsPreviewAfterPause(t *testing.T) {
	e, host := popupEditor(t)
	_ = e.Open("popups")
	counted := &countedUnitHost{personSettingsProbe: &personSettingsProbe{&settingsPreviewProbe{popupHost: host}}}
	e.doc.Edits = counted
	e.Select(0, false)
	_ = e.Key("field")
	e.objectSpec()
	name := -1
	for i, f := range e.obj.fields {
		if f.Key == "name" {
			name = i
		}
	}
	if name < 0 || !e.obj.fields[name].Text {
		t.Fatal("the unit window has no name field")
	}
	e.obj.focus = name
	base := counted.builds
	typed := e.obj.buf[name]
	for _, s := range []string{"x", "y", "z"} {
		e.TextInput(s)
		typed += s
		e.objectSpec()
		if e.obj.buf[name] != typed {
			t.Fatalf("typed text %q is not in the field: %q", typed, e.obj.buf[name])
		}
	}
	if counted.builds != base {
		t.Fatalf("the preview was built %d times while typing", counted.builds-base)
	}
	time.Sleep(450 * time.Millisecond)
	e.objectSpec()
	e.objectSpec()
	if counted.builds != base+1 {
		t.Fatalf("the preview was built %d times after the pause, want 1", counted.builds-base)
	}
}
