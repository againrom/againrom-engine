package sim

import (
	"fmt"
	"testing"
)

func TestSavedHalfTurnAdvancesTheClientOncePerSubTick(t *testing.T) {
	for _, tc := range []struct {
		rate byte
		want []uint8
	}{
		{16, []uint8{16, 32, 48, 64, 80, 96, 112, 128}},
		{21, []uint8{16, 32, 48, 64, 80, 96, 128}},
	} {
		t.Run(fmt.Sprintf("rate%d", tc.rate), func(t *testing.T) {
			w := savedTurnWorldForTest(t, 0, 128, tc.rate, 1, 0)
			w.savedOrder(7).Raw[8] = 0xb
			for call, drawn := range tc.want {
				cold := worldRoundTripForTest(t, w)
				Step(w, nil)
				Step(cold, nil)
				e, m := w.entities[0], w.motionFor(7)
				current := uint8(min(128, (call+1)*int(tc.rate)))
				if e.Facing != current || m.Mover[0] != current || e.DrawnFacing() != drawn {
					t.Fatalf("call %d: server/raw/drawn facing %d/%d/%d, want %d/%d/%d", call+1, e.Facing, m.Mover[0], e.DrawnFacing(), current, current, drawn)
				}
				if e.TurnState.DrawRemaining != uint8(len(tc.want)-call-1) || e.TurnTotal != uint8(len(tc.want)) || e.TurnState.Counter != uint8(call+1) || m.Mover[0x9d] != uint8(call+1) {
					t.Fatalf("call %d: server/client counters %+v, raw counter %d", call+1, e.TurnState, m.Mover[0x9d])
				}
				if e.X != 15 || e.Y != 16 || e.Transit != 0 {
					t.Fatalf("call %d: saved turn moved the actor", call+1)
				}
				if w.Hash() != cold.Hash() || e.DrawnFacing() != cold.entities[0].DrawnFacing() {
					t.Fatalf("call %d: saved turn changed after native reload", call+1)
				}
			}
		})
	}
}
