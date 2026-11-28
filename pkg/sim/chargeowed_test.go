package sim

import (
	"fmt"
	"testing"
)

// A second-game world saved at any tick of an attack must load again, and the
// reloaded world must hold the digest of the continuous one.
func TestACombatSaveLoadsAtEveryTick(t *testing.T) {
	w := gwWorld(t, true, cell{6, 6}, false)
	for tick := range 400 {
		cmds := []Command(nil)
		if tick == 0 {
			cmds = []Command{Attack(1, 3), Attack(2, 4)}
		}
		Step(w, cmds)
		t.Run(fmt.Sprintf("tick%d", tick), func(t *testing.T) {
			back := worldRoundTripForTest(t, w)
			if back.Hash() != w.Hash() {
				t.Fatalf("the reloaded world differs from the continuous one at tick %d", tick)
			}
		})
	}
}
