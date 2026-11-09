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
		for _, b := range e.listButtons() {
			r, ok := e.FindControl(ControlQuery{Role: ControlListButtonKey, Key: b.key})
			if !ok {
				t.Fatalf("%s: no %s button", lang, b.key)
			}
			if need := font.Advance(b.label) + 16; r.Dx() < need {
				t.Errorf("%s: %s button %v is narrower than its caption %q needs (%d)", lang, b.key, r, b.label, need)
			}
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
			t.Fatalf("%s: Structures has no ruined switch beside Add", lang)
		}
		add, _ := e.FindControl(ControlQuery{Role: ControlListButtonKey, Key: "add"})
		if button.Min.X < add.Max.X || button.Min.Y != add.Min.Y {
			t.Fatalf("%s: Add %v and ruined %v are not adjacent", lang, add, button)
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
		e.openAdd(KindStructure)
		if got := e.addPlacement(standing.Cell); !reflect.DeepEqual(got, standing) {
			t.Fatalf("%s: switch changed the separate Add window: %+v", lang, got)
		}
	}
}
