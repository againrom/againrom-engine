package sim

import (
	"fmt"
	"slices"
	"testing"
)

func TestTurnCallsStepTheCurrentByteAtTheActorsRate(t *testing.T) {
	for _, rate := range []int32{16, 21} {
		for _, arc := range []uint8{16, 32, 64, 128} {
			t.Run(fmt.Sprintf("rate%d_arc%d", rate, arc), func(t *testing.T) {
				e := turnActor(1, 3, 3, 0, rate)
				wantCalls := 1
				if arc > 32 {
					wantCalls = (int(arc) + int(rate) - 1) / int(rate)
				}
				w := &World{entities: []Entity{e}}
				w.entities[0].requestFacing(arc)
				for call := 1; call <= wantCalls; call++ {
					want := min(int(arc), call*int(rate))
					if arc <= 32 {
						want = int(arc)
					}
					if got := w.entities[0].Facing; int(got) != want {
						t.Fatalf("call %d: facing %d, want %d", call, got, want)
					}
					if call < wantCalls {
						w.advanceTurns()
					}
				}
			})
		}
	}
}

func TestTurnStandsThroughTheCallThatReachesTheHeading(t *testing.T) {
	b := Bounds{Width: 7, Height: 7}
	w := mustWorldGrid(t, 1, b, ModeCanonical, openGrid(b), []Entity{turnActor(1, 3, 3, 0, 16)})
	for call := 1; call <= 8; call++ {
		var commands []Command
		if call == 1 {
			commands = []Command{MoveTo(1, CellPoint{X: 3, Y: 5})}
		}
		Step(w, commands)
		e := w.entities[0]
		if e.X != 3 || e.Y != 3 || e.Transit != 0 || int(e.Facing) != call*16 {
			t.Fatalf("call %d: position %d,%d transit %d facing %d", call, e.X, e.Y, e.Transit, e.Facing)
		}
	}
	Step(w, nil)
	if e := w.entities[0]; e.X != 3 || e.Y != 4 || e.Turning() {
		t.Fatalf("next call did not step south: %+v", e)
	}
}

func TestTurnDrawnFacingUsesTheClientAccumulator(t *testing.T) {
	for _, tc := range []struct {
		rate int32
		want []uint8
	}{
		{16, []uint8{16, 32, 48, 64, 80, 96, 112, 128}},
		{21, []uint8{16, 32, 48, 64, 80, 96, 128}},
	} {
		t.Run(fmt.Sprintf("rate%d", tc.rate), func(t *testing.T) {
			w := &World{entities: []Entity{turnActor(1, 3, 3, 0, tc.rate)}}
			w.entities[0].requestFacing(128)
			var got []uint8
			for call := range tc.want {
				if call > 0 {
					w.advanceTurns()
				}
				got = append(got, w.entities[0].DrawnFacing())
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("drawn facings %v, want %v", got, tc.want)
			}
		})
	}
}
