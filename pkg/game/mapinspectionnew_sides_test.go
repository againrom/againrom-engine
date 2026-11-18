package game

import (
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/words"
)

// The new map size prompt's help states the constructor's side rule and
// default side in both languages: the editor's words cannot read alm.
func TestNewMapSizeHelpStatesTheSideRule(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		help := words.For(lang).Text("editor.prompt.newmap.help")
		for _, n := range []int{alm.NewMapMinSide, alm.NewMapMaxSide, alm.NewMapSideStep} {
			if !strings.Contains(help, fmt.Sprintf(" %d", n)) {
				t.Errorf("%q: the help %q does not state %d", lang, help, n)
			}
		}
		if side := fmt.Sprintf("%dx%d", alm.NewMapDefaultSide, alm.NewMapDefaultSide); !strings.Contains(help, side) {
			t.Errorf("%q: the help %q does not show %s", lang, help, side)
		}
	}
}

// Every side the window lists follows the constructor's one rule: from
// alm.NewMapMinSide to alm.NewMapMaxSide in steps of alm.NewMapSideStep, and
// the list is ascending, without a repeat.
func TestNewMapSidesFollowTheConstructorsRule(t *testing.T) {
	if len(newMapSides) == 0 {
		t.Fatal("the window lists no side")
	}
	prev := 0
	for _, side := range newMapSides {
		if side <= prev {
			t.Errorf("side %d does not ascend past %d", side, prev)
		}
		prev = side
		if side < alm.NewMapMinSide || side > alm.NewMapMaxSide || side%alm.NewMapSideStep != 0 {
			t.Errorf("side %d breaks the rule %d..%d step %d", side, alm.NewMapMinSide, alm.NewMapMaxSide, alm.NewMapSideStep)
		}
		spec := alm.DefaultNewMap()
		spec.Width, spec.Height = side, side
		if err := spec.Validate(); err != nil {
			t.Errorf("side %d: %v", side, err)
		}
	}
}
