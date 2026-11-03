package ui

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/words"
)

func TestMapEditorStructureGridRuinedSwitchOnlyChangesCopies(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e, _ := newModesEditor(t, "full")
		e.SetWords(words.For(lang))
		e.Layout(640, 480)
		chromeClick(e, barPoint(t, e, "structures"))
		button, ok := e.ListButtonKeyRect("ruined")
		if !ok {
			t.Fatalf("%s: Structures has no ruined switch beside Add", lang)
		}
		add, _ := e.ListButtonKeyRect("add")
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
