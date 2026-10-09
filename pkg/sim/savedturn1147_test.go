package sim

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func savedTurnWorldForTest(t *testing.T, current, desired, rate byte, active uint32, counter byte) *World {
	t.Helper()
	w, m, cells, blocks := motionFixture1115(t, 128, 128, 0, 0, 0, 0, 0)
	m.StaticRoute, m.DynamicRoute = nil, nil
	m.Mover[0], m.Mover[1], m.Mover[10], m.Mover[0x9d] = current, desired, rate, counter
	m.Mover[0xa4] = 99 // A stale estimate must not become a countdown.
	binary.LittleEndian.PutUint32(m.Mover[0xa0:], active)
	w.savedOrder(7).Raw[8], w.savedOrder(7).Raw[9] = 1, 0
	importMotion1115(t, w, m, cells, blocks)
	return w
}

func TestSavedTurn1147LocalBranches(t *testing.T) {
	for _, tc := range []struct {
		name                                 string
		current, desired, rate               byte
		active                               uint32
		counter, next, estimate, nextCounter byte
		nextActive                           uint32
	}{
		{"fresh short", 0, 32, 16, 0, 7, 32, 1, 1, 0},
		{"active short", 0, 32, 16, 1, 7, 16, 2, 8, 1},
		{"full dword", 0, 32, 16, 256, 7, 16, 2, 8, 1},
		{"half circle", 0, 128, 16, 1, 7, 16, 8, 8, 1},
		{"wrap add", 250, 32, 16, 1, 7, 10, 3, 8, 1},
		{"wrap subtract", 16, 240, 16, 1, 7, 0, 2, 8, 1},
		{"terminal clamp and counter wrap", 0, 8, 16, 1, 255, 8, 1, 0, 0},
		{"equal facings active high byte", 64, 64, 16, 65536, 7, 64, 0, 8, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := savedTurnWorldForTest(t, tc.current, tc.desired, tc.rate, tc.active, tc.counter)
			before := *w.motionFor(7)
			if before.Issue != "" || !w.ActorMotionActive(7) {
				t.Fatalf("turn not admitted: %+v", before)
			}
			want := before.Mover
			want[0], want[0xa4], want[0x9d] = tc.next, tc.estimate, tc.nextCounter
			binary.LittleEndian.PutUint32(want[0xa0:], tc.nextActive)
			cold := retreatRoundTrip1089(t, w)
			Step(w, nil)
			Step(cold, nil)
			got := w.motionFor(7)
			if got.Mover != want || got.Position != before.Position || w.entities[0].Facing != tc.next {
				t.Fatalf("wrong stores: facing=%d a4=%d counter=%d active=%d", got.Mover[0], got.Mover[0xa4], got.Mover[0x9d], binary.LittleEndian.Uint32(got.Mover[0xa0:]))
			}
			if w.Hash() != cold.Hash() {
				t.Fatal("cold turn continuation differs")
			}
			_ = mustMarshal(t, w)
		})
	}
}

func TestSavedTurn1147CompletesBeforeFollowingRoute(t *testing.T) {
	w := savedTurnWorldForTest(t, 0, 96, 16, 1, 0)
	m := w.motionFor(7)
	m.DynamicRoute, m.StaticRoute = []uint16{0x1110}, []uint16{0x1010}
	w.savedOrder(7).Raw[10], w.savedOrder(7).Raw[11] = 16, 17
	wantDynamic, wantStatic := append([]uint16{}, m.DynamicRoute...), append([]uint16{}, m.StaticRoute...)
	for n, want := range []byte{16, 32, 48, 64, 80, 96} {
		cold := retreatRoundTrip1089(t, w)
		Step(w, nil)
		Step(cold, nil)
		m = w.motionFor(7)
		if m.Mover[0] != want || m.Mover[0xa4] != byte(6-n) || w.entities[0].X != 15 || w.entities[0].Y != 16 {
			t.Fatalf("turn step %d: facing=%d estimate=%d cell=%d,%d", n, m.Mover[0], m.Mover[0xa4], w.entities[0].X, w.entities[0].Y)
		}
		if !reflect.DeepEqual(m.DynamicRoute, wantDynamic) || !reflect.DeepEqual(m.StaticRoute, wantStatic) {
			t.Fatal("turn changed a route")
		}
		if w.Hash() != cold.Hash() {
			t.Fatal("mid-turn native reload changed continuation")
		}
	}
	if w.ActorMotionActive(7) || !w.entities[0].HasTarget || !reflect.DeepEqual(w.routes[0], []cell{{16, 17}}) {
		t.Fatal("completed move turn did not hand off its dynamic route")
	}
	if w.savedMotion.Blocks[0].Cell != 0x100f || w.savedMotion.Blocks[0].Dyn != 3 {
		t.Fatal("turn handoff left an imported occupancy bit behind")
	}
	Step(w, nil)
	if w.entities[0].X != 16 || w.entities[0].Y != 17 {
		t.Fatalf("following body update did not move: %+v", w.entities[0])
	}
}

func TestSavedTurn1147LegacyRefusalsAndUnrelatedDebt(t *testing.T) {
	for _, issue := range []string{"centered original turn continuation is not executed", "original pending turn after crossing is not executed", "original boundary speed callback is not executed"} {
		w := savedTurnWorldForTest(t, 0, 64, 16, 1, 0)
		w.motionFor(7).Issue = issue
		cold := retreatRoundTrip1089(t, w)
		if cold.motionFor(7).Issue != issue {
			t.Fatal("LOAD rewrote the historical issue")
		}
		Step(cold, nil)
		m := cold.motionFor(7)
		if issue == "original boundary speed callback is not executed" {
			if m.Mover[0] != 0 || m.Issue != issue {
				t.Fatal("unrelated callback debt was silently bypassed")
			}
		} else if m.Mover[0] != 16 || m.Issue != "" {
			t.Fatalf("old turn refusal was not re-admitted: %+v", m)
		}
	}
	w := savedTurnWorldForTest(t, 0, 64, 0, 1, 0)
	if w.motionFor(7).Issue == "" || w.ActorMotionActive(7) {
		t.Fatal("zero rotation rate admitted")
	}
	Step(w, nil)
	_ = mustMarshal(t, w)
}

func TestSavedTurn1147StandingTurnDoesNotWalkAStaleRoute(t *testing.T) {
	w := savedTurnWorldForTest(t, 0, 32, 16, 0, 0)
	w.savedOrder(7).Raw[8] = 0xb
	w.motionFor(7).StaticRoute = []uint16{0x1010}
	Step(w, nil)
	if w.entities[0].HasTarget || !w.motionFor(7).Current || len(w.routes[0]) != 0 {
		t.Fatal("standing facing turned into a movement order")
	}
}

func TestManualTeleportInterruptsSavedTurn1147(t *testing.T) {
	w := importedRelocationWorld1115(t)
	m := w.motionFor(1)
	m.Position.FineX, m.Position.FineY, m.Active, m.Issue = 128, 128, false, ""
	clear(m.Mover[0xaa:0xae])
	m.Mover[0], m.Mover[1], m.Mover[10], m.Mover[0xa0] = 0, 128, 16, 1
	w.entities[0].Facing = 0
	w.entities[0].clearTurn()
	Step(w, []Command{{Kind: KindCastAt, Entity: 1, Spell: 26, X: 7, Y: 5}})
	if w.ActorMotionActive(1) || w.motionFor(1).Current || len(w.bookCasts) != 1 {
		t.Fatal("saved turn prevented manual Teleport")
	}
	cold := retreatRoundTrip1089(t, w)
	for range 64 {
		Step(w, nil)
		Step(cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatal("manual takeover differs after reload")
		}
	}
	if w.entities[0].X != 7 || w.entities[0].Y != 5 {
		t.Fatal("manual Teleport did not finish")
	}
}

func TestSavedTurn1147WallDuringTurnKeepsTheRouteAndSaveReloadable(t *testing.T) {
	w := savedTurnWorldForTest(t, 0, 96, 16, 1, 0)
	w.motionFor(7).DynamicRoute = []uint16{0x1110}
	w.savedOrder(7).Raw[10], w.savedOrder(7).Raw[11] = 16, 17
	w.spells = []SpellRule{{ID: 19, Area: true, Distribution: 4, Radius: 2, AreaDuration: 15}}
	Step(w, nil)
	if !w.landArea(w.spells[0], 0, 0, false, 12, 17, 16, 17, nil) {
		t.Fatal("wall did not land")
	}
	for range 5 {
		Step(w, nil)
	}
	if w.entities[0].Facing != 96 || w.motionFor(7).Current || !w.entities[0].HasTarget || len(w.routes[0]) == 0 {
		t.Fatal("completed turn dropped the route across the closed cell or lost its goal", w.routes[0])
	}
	cold := retreatRoundTrip1089(t, w)
	for range 24 {
		Step(w, nil)
		Step(cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatal("wall continuation differs after reload")
		}
		_ = retreatRoundTrip1089(t, w)
	}
}
