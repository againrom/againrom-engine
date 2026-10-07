package game

import (
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// Every installed ranged weapon row describes itself with a range line that
// carries the installed range caption and the row's own range column, as the
// last line. Every melee row states none. Rows that carry a weapon spell are
// described by their spell lines and are not part of the census.
func TestReleaseRangedWeaponInformationStatesItsRange(t *testing.T) {
	f := releaseFront(t)
	caption := f.Words.Hover[spellLabelRange]
	if caption == "" {
		t.Fatal("the install states no range caption")
	}
	weapons := f.Table.Weapons
	ranged, melee := 0, 0
	for row := 1; row < weapons.Len() && row < 32; row++ {
		code := data.ComposeItemCode(0, 1, 0, row) // class 1 is the weapon slot class
		w, err := data.WeaponFromCode(code, f.Table.Shapes, f.Table.Materials, weapons)
		if err != nil {
			continue
		}
		lines := itemInstanceInfoLines(sim.ItemInstance{Code: uint16(code)}, f.Table, f.Words)
		last := lines[len(lines)-1]
		has := strings.HasPrefix(last, caption+": ")
		if w.Ranged() || w.AttackType == data.SkillShoot {
			want := fmt.Sprintf("%s: %d", caption, w.Range)
			if last != want {
				t.Errorf("row %d %q (attack type %d): lines %q, want last %q", row, lines[0], w.AttackType, lines, want)
			}
			t.Logf("row %d attack type %d: %q", row, w.AttackType, lines)
			ranged++
			continue
		}
		if has {
			t.Errorf("melee row %d %q states a range line: %q", row, lines[0], lines)
		}
		if melee < 2 {
			t.Logf("melee control row %d attack type %d: %q", row, w.AttackType, lines)
		}
		melee++
	}
	if ranged == 0 || melee == 0 {
		t.Fatalf("census found %d ranged and %d melee rows", ranged, melee)
	}
}
