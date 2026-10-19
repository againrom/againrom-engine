package sim

import (
	"fmt"
	"testing"
)

func TestZeroMaximumStructureApproachAndBoundedUse(t *testing.T) {
	for _, kind := range []uint16{34, 35, 39, 40, 66} {
		t.Run(fmt.Sprint(kind), func(t *testing.T) {
			w := structureUseWorld(t, kind)
			w.structures[0].Col, w.structures[0].Row = 12, 12
			w.structures[0].Field42, w.structures[0].MaxHealth = 0, 0
			Step(w, []Command{UseStructure(0, 0)})
			if len(w.StructureUses()) != 1 {
				t.Fatal("use did not retain approach")
			}
			finishStructureUse(t, w)
			goal, radius := structureUseGoal(w.structures[0])
			if cellOf(&w.entities[0]).chebyshevTo(goal) > radius || w.structures[0].Field42 != 0 || w.entities[0].HP != 10 || w.entities[0].Mana != 20 {
				t.Fatal("use failed approach or invented a pool effect")
			}
			w.structures[0].Field42 = 1
			w.applyStructureUse(0, 0)
			if w.structures[0].Field42 != 1 || w.entities[0].HP != 10 {
				t.Fatal("non-fountain use applied a potion")
			}
		})
	}
}
