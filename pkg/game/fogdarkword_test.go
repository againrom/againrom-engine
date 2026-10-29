package game

import (
	"testing"

	"againrom/pkg/sim"
)

// A loaded body whose saved sight word has a high byte of 0 and a nonzero low
// byte uncovers its block at the first stamp after a LOAD; a saved word of 0
// uncovers nothing (TERR-TILE-079, TERR-FOG-268).
func TestAStampAfterLoadOpensTheBlockOfAWordWithNoWholeCell(t *testing.T) {
	for _, tc := range []struct {
		low  uint32
		want int
	}{{0x80, 4}, {0, 0}} {
		e := sim.Entity{ID: 1, X: 22, Y: 22, HP: 100, MaxHP: 100, Owner: sim.SelfSlot, TokenSize: 1,
			Humanoid: true, Mind: 25, Reaction: 36}
		e.NativeBasis.ScalarsPresent = true
		e.NativeBasis.ScalarKnown = 1 << sim.ScalarUA4
		e.NativeBasis.Scalars[sim.ScalarUA4] = tc.low
		mw := fogMoverWorld(t, e)
		mw.fog = newFogPlane(int(fogTestBounds.Width), int(fogTestBounds.Height))
		mw.stampAfterLoad()
		if got := countState(mw.fog.visible, 1); got != tc.want {
			t.Errorf("saved low byte %#x: the stamp after LOAD lights %d visible cells, want %d", tc.low, got, tc.want)
		}
		if got := countState(mw.fog.explored, 1); got != tc.want {
			t.Errorf("saved low byte %#x: the stamp after LOAD explores %d cells, want %d", tc.low, got, tc.want)
		}
	}
}
