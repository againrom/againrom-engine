package game

import "testing"

// TestTownSeatClockCountsSixTicksPerFrame holds the first game's tavern seat
// clock to its shipped step: one frame per six client ticks, restarted by a
// press and stopped, back at zero, while no room surface is shown. The model
// is the tick count divided by six.
func TestTownSeatClockCountsSixTicksPerFrame(t *testing.T) {
	f := shellFrontEnd()
	s := f.townUI
	s.room = roomTavern
	tick := 0
	for step := 1; step <= 60; step++ {
		switch step {
		case 17:
			s.TownSeatRestart()
			tick = 0
		case 31:
			s.TownSeatTick(false)
			tick = 0
		default:
			s.TownSeatTick(true)
			tick++
		}
		if got, want := s.TownSurface().AnimationFrame, tick/6; got != want {
			t.Fatalf("step %d: seat frame %d, want %d", step, got, want)
		}
	}
}

// TestTownSeatClockLeavesTimedSeatsAlone keeps the clock of a room whose seats
// count milliseconds off the client tick.
func TestTownSeatClockLeavesTimedSeatsAlone(t *testing.T) {
	for _, name := range matrixDescriptions {
		s, d, _ := matrixScreen(t, name)
		if d.TickSeats() != nil {
			continue
		}
		for i := 0; i < 20; i++ {
			s.TownSeatTick(true)
		}
		if got := s.townSeatPhase(); got != 0 {
			t.Fatalf("%s: tick-driven seat frame %d, want 0", name, got)
		}
	}
}
