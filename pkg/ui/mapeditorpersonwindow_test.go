package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/internal/witness"
)

type personPortraitProbe struct{ *personSettingsProbe }

func (p *personPortraitProbe) UnitSettings(index int, values []InspectionFieldValue) (MapEntity, []InspectionField, *InspectionUnitPreview, error) {
	entity, fields, _, err := p.personSettingsProbe.UnitSettings(index, values)
	pic := image.NewRGBA(image.Rect(0, 0, 160, 240))
	draw.Draw(pic, image.Rect(36, 8, 108, 100), image.NewUniform(color.RGBA{R: byte(entity.HP), G: 80, A: 255}), image.Point{}, draw.Src)
	draw.Draw(pic, image.Rect(50, 104, 100, 230), image.NewUniform(color.RGBA{B: 180, A: 255}), image.Point{}, draw.Src)
	return entity, fields, &InspectionUnitPreview{Doll: pic}, err
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
				witness.PNG(t, filepath.Join(dir, fmt.Sprintf("person-%dx%d.png", size.X, size.Y)), pic)
			}
			if !l.Rect.In(e.popupBounds()) {
				t.Error("person window exceeds its bounds", l.Rect)
			}
		})
	}
}

// The detailed view has no tabs: every section's fields stand at once, each
// section under its heading, in as many columns as the window's width holds.
func TestEditorPersonDetailedViewShowsEverySectionAtOnce(t *testing.T) {
	for _, size := range []image.Point{{640, 480}, {800, 600}, {1280, 1024}, {1920, 1080}} {
		t.Run(size.String(), func(t *testing.T) {
			e, host := popupEditor(t)
			_ = e.Open("popups")
			e.doc.Edits = &personSettingsProbe{&settingsPreviewProbe{popupHost: host}}
			e.Layout(size.X, size.Y)
			e.Select(0, false)
			s, _, _ := e.objectSpec()
			if len(s.Tabs) != 0 {
				t.Fatal("the detailed view still has tabs", s.Tabs)
			}
			want := min(4, max(1, (e.unitWindowWidth()-32-e.hostedAside(s)-popupPad)/personColumnWidth))
			if cardHeld(s) {
				// The card's block holds the stats section in the identity column.
				want = min(3, max(1, (e.unitWindowWidth()-32)/personColumnWidth))
			}
			if s.Columns != want {
				t.Errorf("columns %d, want %d", s.Columns, want)
			}
			shown := map[string]bool{}
			for _, i := range e.obj.rowField {
				shown[e.obj.fields[i].Section] = true
			}
			for _, f := range e.obj.fields {
				if f.Section != "" && !f.ReadOnly && f.Parent == "" && !hiddenField(e.doc.Records[e.selected], f) && !shown[f.Section] {
					t.Errorf("section %s is not shown", f.Section)
				}
			}
			for section := range shown {
				if !slices.ContainsFunc(s.Rows, func(r popupRow) bool { return r.Kind == popupHeading && r.Label == e.word("editor.person."+section) }) {
					t.Errorf("section %s has no heading", section)
				}
			}
			updateEditorKeys(t, e)
			updateEditorKeys(t, e, ebiten.KeyTab)
			if e.obj.focus < 0 {
				t.Error("Tab did not focus a field")
			} else if _, ok := e.FindControl(ControlQuery{Role: ControlField, N: e.obj.focus}); !ok {
				t.Error("Tab focused a hidden field")
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
				} else if _, ok := e.FindControl(ControlQuery{Role: ControlField, N: e.obj.focus}); !ok {
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
				t.Error("native Tab cannot visit every section with a field", seen)
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
					// The window shows no facing, speaker or map keys.
					if field.Section == name && !seen[i] && !hiddenField(e.doc.Records[e.selected], field) {
						t.Errorf("scrolling cannot reach %s", field.Key)
					}
				}
			}
		})
	}
}
