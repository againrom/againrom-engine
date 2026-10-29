package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type personPortraitProbe struct{ *personSettingsProbe }

func (p *personPortraitProbe) UnitSettings(index int, values []InspectionFieldValue) (MapEntity, []InspectionField, *InspectionUnitPreview, error) {
	entity, fields, _, err := p.personSettingsProbe.UnitSettings(index, values)
	pic := image.NewRGBA(image.Rect(0, 0, 160, 240))
	draw.Draw(pic, image.Rect(36, 8, 108, 100), image.NewUniform(color.RGBA{R: byte(entity.HP), G: 80, A: 255}), image.Point{}, draw.Src)
	draw.Draw(pic, image.Rect(50, 104, 100, 230), image.NewUniform(color.RGBA{B: 180, A: 255}), image.Point{}, draw.Src)
	return entity, fields, &InspectionUnitPreview{Doll: pic, Face: pic}, err
}

func TestEditorPersonDialoguePortraitPaintsBesideDoll(t *testing.T) {
	e, host := popupEditor(t)
	_ = e.Open("popups")
	e.doc.Edits = &personPortraitProbe{&personSettingsProbe{&settingsPreviewProbe{popupHost: host}}}
	e.Layout(800, 600)
	e.Select(0, false)
	for _, value := range []string{"10", "42"} {
		e.objectSpec()
		e.obj.buf[0] = value
		s, _, _ := e.objectSpec()
		l := layoutPopup(s, e.font, e.bodyFont())
		pic := image.NewRGBA(image.Rect(0, 0, 800, 600))
		e.drawObjectPreview(pic, s)
		at := image.Pt(l.Rect.Max.X-popupPad-244+12, l.Tabs[0].Max.Y+2*popupPad+compactPanelH+12)
		want := color.RGBA{R: byte(e.obj.unit.entity.HP), G: 80, A: 255}
		if got := pic.RGBAAt(at.X, at.Y); got != want {
			t.Errorf("draft %s dialogue portrait at %v = %v, want %v", value, at, got, want)
		}
	}
}

func TestEditorPersonWindowOwnerFrames(t *testing.T) {
	for _, size := range []image.Point{{640, 480}, {800, 600}, {1024, 768}, {1280, 1024}, {1920, 1080}} {
		t.Run(size.String(), func(t *testing.T) {
			e, host := popupEditor(t)
			_ = e.Open("popups")
			e.doc.Edits = &personPortraitProbe{&personSettingsProbe{&settingsPreviewProbe{popupHost: host}}}
			e.Layout(size.X, size.Y)
			e.Select(0, false)
			s, _, _ := e.objectSpec()
			l := layoutPopup(s, e.font, e.bodyFont())
			pic := e.ChromeImage()
			t.Logf("editor=%dx%d popup=%v coverage=%.2f%% pitch=%d other-popup-pitch=%d rows=%d/%d", size.X, size.Y, l.Rect, float64(l.Rect.Dx()*l.Rect.Dy())*100/float64(size.X*size.Y), l.Pitch, layoutPopup(popupFixture(), e.font, e.bodyFont()).Pitch, len(l.Rows), len(s.Rows))
			if dir := os.Getenv("AGAINROM_PERSON_FRAMES"); dir != "" && (size.X == 800 || size.X == 1280) {
				file, err := os.Create(filepath.Join(dir, fmt.Sprintf("person-%dx%d.png", size.X, size.Y)))
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(file, pic)
				closeErr := file.Close()
				if err != nil || closeErr != nil {
					t.Fatal(err, closeErr)
				}
			}
			if !l.Rect.In(e.popupBounds()) || l.Rect.Dx() > 1180 || l.Rect.Dy() > 620 {
				t.Error("person window exceeds its bounded natural size", l.Rect)
			}
		})
	}
}

func TestEditorPersonSectionButtonsSelectTheirRows(t *testing.T) {
	for _, size := range []image.Point{{640, 480}, {800, 600}, {1280, 1024}} {
		t.Run(size.String(), func(t *testing.T) {
			e, host := popupEditor(t)
			_ = e.Open("popups")
			e.doc.Edits = &personSettingsProbe{&settingsPreviewProbe{popupHost: host}}
			e.Layout(size.X, size.Y)
			e.Select(0, false)
			for _, section := range []int{1, 2, 3, 0} {
				s, _, _ := e.objectSpec()
				l := layoutPopup(s, e.font, e.bodyFont())
				chromeClick(e, center(l.Tabs[section]))
				s, _, _ = e.objectSpec()
				if s.ActiveTab != section {
					t.Errorf("section button %d did nothing: active=%d drag=%d", section, s.ActiveTab, e.drag)
				}
				if size.X < 1100 {
					for _, i := range e.obj.rowField {
						if e.obj.fields[i].Section != []string{"identity", "equipment", "inventory", "stats"}[section] {
							t.Error("section button kept another section's field", e.obj.fields[i].Key)
						}
					}
				}
				updateEditorKeys(t, e)
				updateEditorKeys(t, e, ebiten.KeyTab)
				if e.obj.focus < 0 {
					t.Errorf("Tab after section button %d did not focus a field", section)
				} else if _, ok := e.FieldRect(e.obj.focus); !ok {
					t.Errorf("Tab after section button %d hid its field", section)
				}
			}
		})
	}
}

func TestEditorPersonNativeTabFromEveryField(t *testing.T) {
	for _, size := range []image.Point{{640, 480}, {1280, 1024}} {
		t.Run(size.String(), func(t *testing.T) {
			e, host := popupEditor(t)
			_ = e.Open("popups")
			e.doc.Edits = &personSettingsProbe{&settingsPreviewProbe{popupHost: host}}
			e.Layout(size.X, size.Y)
			e.Select(0, false)
			e.objectSpec()
			updateEditorKeys(t, e, ebiten.KeyTab)
			if e.obj.focus < 0 {
				t.Error("native Tab without field focus did nothing")
			}
			for i, field := range e.obj.fields {
				e.obj.section = map[string]int{"identity": 0, "equipment": 1, "inventory": 2, "stats": 3}[field.Section]
				e.focusObjectField(i)
				updateEditorKeys(t, e)
				updateEditorKeys(t, e, ebiten.KeyTab)
				if e.obj.focus == i || e.obj.focus < 0 {
					t.Errorf("native Tab from %s did nothing", field.Key)
				} else if _, ok := e.FieldRect(e.obj.focus); !ok {
					t.Errorf("native Tab from %s hid its destination", field.Key)
				}
			}
			e.obj.section = 0
			e.focusObjectField(0)
			seen := map[string]bool{}
			for range len(e.obj.fields) * 2 {
				seen[e.obj.fields[e.obj.focus].Section] = true
				updateEditorKeys(t, e)
				updateEditorKeys(t, e, ebiten.KeyTab)
			}
			if len(seen) != 4 {
				t.Error("native Tab cannot visit every section", seen)
			}
		})
	}
}

func TestEditorPersonEveryFieldRemainsReachable(t *testing.T) {
	for _, size := range []image.Point{{640, 480}, {800, 600}, {1024, 768}, {1280, 1024}, {1920, 1080}} {
		t.Run(size.String(), func(t *testing.T) {
			e, host := popupEditor(t)
			_ = e.Open("popups")
			e.doc.Edits = &personSettingsProbe{&settingsPreviewProbe{popupHost: host}}
			e.SetBodyFont(solidFont15())
			e.Layout(size.X, size.Y)
			e.Select(0, false)
			e.objectSpec()
			for section, name := range []string{"identity", "equipment", "inventory", "stats"} {
				e.obj.section = section
				e.objectSpec()
				seen := map[int]bool{}
				for top := 0; top < 100; top++ {
					e.objectTop = top
					s, _, _ := e.objectSpec()
					l := layoutPopup(s, e.font, e.bodyFont())
					if !l.Rect.In(e.popupBounds()) {
						t.Fatal("popup exceeds editor with body font", l.Rect)
					}
					for _, row := range l.Rows {
						if i, ok := e.obj.rowField[row.Index]; ok && e.obj.fields[i].Section == name {
							seen[i] = true
							if !row.Control.In(l.Rect) || row.Control.Empty() {
								t.Fatal("field has no usable control", e.obj.fields[i].Key)
							}
						}
					}
				}
				for i, field := range e.obj.fields {
					if field.Section == name && !seen[i] {
						t.Errorf("scrolling cannot reach %s", field.Key)
					}
				}
			}
		})
	}
}
