package sim

import "testing"

// A bound equipment class rule runs after each Equip, Unequip and DropWorn
// command, reads the slots that command left, and its answer is the class the
// actor holds from there on in the step. A fixed class and a World with no
// rule bound keep their class.
func TestAnEquipmentCommandStoresTheClassTheBoundRuleAnswers(t *testing.T) {
	const sword = 0x101
	rule := func(w *World, id EntityID) (int32, bool) {
		slots, ok := w.Equipped(id)
		if !ok {
			return 0, false
		}
		if slots[0] == sword {
			return 3, true
		}
		return 1, true
	}
	commands := []Command{Equip(1, 0, 1), Unequip(1, 1), Equip(1, 0, 1), DropWorn(1, 1, CellPoint{X: 5, Y: 5})}
	for _, tc := range []struct {
		name         string
		fixed, bound bool
		want         []int32
	}{
		{"bound", false, true, []int32{3, 1, 3, 1}},
		{"fixed class", true, true, []int32{9, 9, 9, 9}},
		{"unbound", false, false, []int32{9, 9, 9, 9}},
	} {
		w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5, HP: 10, MaxHP: 10, Class: 9, ClientClassFixed: tc.fixed}},
			[]Stock{{ID: 1, Items: []uint16{sword}}})
		if tc.bound {
			w.SetEquipmentClassRule(rule)
		}
		for k, c := range commands {
			Step(w, []Command{c})
			e, _ := w.Entity(1)
			slots, _ := w.Equipped(1)
			if e.Class != tc.want[k] {
				t.Errorf("%s: after command %d (kind %d, slot 1 %#x) class %d, want %d", tc.name, k, c.Kind, slots[0], e.Class, tc.want[k])
			}
		}
	}
}
