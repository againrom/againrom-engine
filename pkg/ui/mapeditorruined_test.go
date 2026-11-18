package ui

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/words"
)

// Each button of the Structures list holds its caption, the ruined switch's
// mark and word included, in both languages.
func TestMapEditorStructureListButtonsHoldTheirCaptions(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e, _ := newModesEditor(t, "full")
		e.SetWords(words.For(lang))
		e.Layout(640, 480)
		chromeClick(e, barPoint(t, e, "structures"))
		font := e.bodyFont()
		if font == nil {
			t.Fatal("the editor has no body font to measure captions with")
		}
		s := mustListSpec(t, e)
		r, ok := e.FindControl(ControlQuery{Role: ControlListButtonKey, Key: "ruined"})
		if !ok {
			t.Fatalf("%s: no Ruined switch", lang)
		}
		if need := font.Advance(s.SwitchButton) + 16; r.Dx() < need {
			t.Errorf("%s: Ruined switch %v is narrower than its caption %q needs (%d)", lang, r, s.SwitchButton, need)
		}
	}
}

func TestMapEditorStructureGridRuinedSwitchOnlyChangesCopies(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e, _ := newModesEditor(t, "full")
		e.SetWords(words.For(lang))
		e.Layout(640, 480)
		chromeClick(e, barPoint(t, e, "structures"))
		button, ok := e.FindControl(ControlQuery{Role: ControlListButtonKey, Key: "ruined"})
		if !ok {
			t.Fatalf("%s: Structures has no ruined switch in its button row", lang)
		}
		s := mustListSpec(t, e)
		l := layoutPopup(s, e.font, e.bodyFont())
		if button.Min.X != l.Rect.Min.X+popupPad || button.Min.Y < l.Grid.Max.Y || len(l.Buttons) != 0 {
			t.Fatalf("%s: ruined %v does not stand alone at the footer's left edge", lang, button)
		}
		e.placeChoose(0)
		standing := e.addPlacement(image.Pt(25, 20))
		chromeClick(e, center(button))
		ruined := e.addPlacement(standing.Cell)
		want := standing
		want.Values = append(append([]InspectionFieldValue(nil), standing.Values...), InspectionFieldValue{Key: "placement", Value: 0})
		if !reflect.DeepEqual(ruined, want) {
			t.Fatalf("%s: switch changed %+v into %+v; want %+v", lang, standing, ruined, want)
		}
		chromeClick(e, center(button))
		if got := e.addPlacement(standing.Cell); !reflect.DeepEqual(got, standing) {
			t.Fatalf("%s: switch off did not restore standing copies: %+v", lang, got)
		}
		chromeClick(e, center(button))
		if r, ok := e.addRunning(); ok {
			r.brush = true
			if got := e.addPlacement(standing.Cell); !reflect.DeepEqual(got, standing) {
				t.Fatalf("%s: switch changed the Objects brush: %+v", lang, got)
			}
		}

	}
}
