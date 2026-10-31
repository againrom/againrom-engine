package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/words"
)

func TestPlayersWindowAndWizardNameARelationByTheSameWord(t *testing.T) {
	for _, lang := range []string{"", "ru"} {
		t.Run("lang="+lang, func(t *testing.T) {
			book := words.For(lang)
			players, _ := newModesEditor(t, "full")
			players.SetWords(book)
			_ = players.Key("players")
			// The chosen player's window floats clear of the matrix.
			players.objectAt = image.Pt(0, 0)
			wizard, _ := wizardChoiceEditor(t)
			wizard.SetWords(book)
			choices := wizard.wizardTargets(wpRelation)
			if len(choices) != 3 {
				t.Fatalf("the wizard lists %d relation choices, want 3: %+v", len(choices), choices)
			}
			cell, ok := players.PlaceGridCellRect(1)
			if !ok {
				t.Fatal("matrix cell 0,1 is not shown")
			}
			prefix := fmt.Sprintf(players.word("editor.relation.cell"), "Self", "Enemy", "")
			for _, v := range []int64{1, 2, 0} {
				s, ok := players.listSpec()
				if !ok || s.Grid == nil || s.Grid.Cells[1].Icon != relationIcon(v) {
					t.Fatalf("language %q: cell 0,1 does not show %d", lang, v)
				}
				tip := s.Grid.Cells[1].Tip
				if !strings.HasPrefix(tip, prefix) {
					t.Fatalf("language %q: hover line %q does not start with %q", lang, tip, prefix)
				}
				name := strings.TrimPrefix(tip, prefix)
				k := slices.IndexFunc(choices, func(c InspectionChoice) bool { return c.Value == v })
				if k < 0 || choices[k].Label != name {
					t.Errorf("language %q, value %d: the players window says %q, the wizard %+v", lang, v, name, choices)
				}
				chromeClick(players, center(cell))
			}
		})
	}
}

func TestWizardChangeRelationWritesCommandTenForEachValue(t *testing.T) {
	e, _ := wizardChoiceEditor(t)
	item := slices.IndexFunc(wizItems(false), func(it wizItem) bool { return it.opcode == 10 })
	if item < 0 {
		t.Fatal("the wizard has no action item for command 10")
	}
	for _, lang := range []string{"", "ru"} {
		e.SetWords(words.For(lang))
		got, _ := e.actionWord(10)
		if want := e.word(wizItems(false)[item].labelWord); got != want {
			t.Fatalf("language %q: the trigger list names command 10 %q, the wizard %q", lang, got, want)
		}
	}
	e.SetWords(words.For(""))
	wizardChoose(t, e, "a1.item", int64(item))
	for _, v := range []int64{0, 1, 2} {
		wizardChoose(t, e, "a1.p2", v)
		scriptView, _ := e.scriptPopup()
		plan, err := e.wizardPlan(scriptView)
		if err != nil {
			t.Fatalf("relation %d plans no trigger: %v", v, err)
		}
		n := plan.Actions[1]
		want := WizardNode{ID: 12, Opcode: 10, Types: [10]uint32{1, 1, 1}, Values: [10]uint32{1, 2, uint32(v)}}
		if n != want {
			t.Errorf("relation %d writes %+v; want %+v", v, n, want)
		}
	}
}
