package game

// THE PLACEHOLDER WALKABOUT, kept as TEST SUPPORT and removed from the game.
//
// It lived in pkg/mapload as an exported Schedule and was wired into BOTH
// front-end paths — the picker's openMapWorld and the campaign's
// openMission — so opening any map made every mob on it walk a closed
// square, six laps, and then stand still forever. It was authored by 0020 to
// give the tick loop something to advance before there was any AI, and
// nothing retired it when there was.
//
// What it is still good for is the one thing 0020 built it for: a COMMAND
// STREAM that is a pure function of a map, so a driven run and a headless
// run can be advanced on identical input and their digests compared. So the
// generator moves here, unexported, and the production paths pass no
// schedule at all.
//
// It is a straight copy of the deleted mapload.Schedule, including legOffset and
// the cell conversion it called, so the tests that depended on it are testing
// the same stream they always were.

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The schedule's three numbers.
//
// legCells is a leg's length in cells, and — a step moving one cell per tick —
// also the ticks a full-length leg takes. legs is how many legs each entity is
// given: 24, six laps of the four-legged square. stagger is how much later each
// further entity's first order comes, and it is COPRIME with legCells so that no
// two entities within a run of 24 ids ever turn on the same tick.
const (
	schedLegCells = 24
	schedLegs     = 24
	schedStagger  = 7
)

// placeholderSchedule returns the order script for a decoded map: entry t holds
// the commands to apply at tick t of a world FromALM built from the same map.
//
// Every entity walks a closed square of four legs from its own start cell, lap
// after lap; the j-th target is that cell offset by (dx,0), (dx,dy), (0,dy),
// (0,0) in turn, so every fourth leg returns it to where it began. Entity i is
// given schedLegs targets, the j-th at tick i*schedStagger + j*schedLegCells.
//
// It reads m.Units, m.Width and m.Height and nothing else — no clock, no
// generator, no environment and no file — which is the property the digest
// comparison rests on: one map is one schedule, in every process and on every
// machine.
func placeholderSchedule(m *alm.Map) [][]sim.Command {
	if len(m.Units) == 0 {
		return nil
	}
	w, h := int32(m.Width), int32(m.Height)

	// One past the last order: the highest-numbered entity's last leg.
	sched := make([][]sim.Command, (len(m.Units)-1)*schedStagger+(schedLegs-1)*schedLegCells+1)

	for i, u := range m.Units {
		x, y := int32(u.X>>8), int32(u.Y>>8)
		dx, dy := schedLegOffset(x, w), schedLegOffset(y, h)
		square := [4][2]int32{{dx, 0}, {dx, dy}, {0, dy}, {0, 0}}

		for j := 0; j < schedLegs; j++ {
			off := square[j%4]
			t := i*schedStagger + j*schedLegCells
			sched[t] = append(sched[t], sim.Command{
				Entity: sim.EntityID(i),
				X:      x + off[0],
				Y:      y + off[1],
			})
		}
	}
	return sched
}

// schedLegOffset is one axis' leg: how far from pos a leg on an axis of extent
// cells reaches, signed toward the side with more room, and capped at the room
// actually there so the target stays on the map. A tie goes to the low side.
func schedLegOffset(pos, extent int32) int32 {
	if high := extent - 1 - pos; high > pos {
		return min(schedLegCells, high)
	}
	return -min(schedLegCells, pos)
}

func TestNeitherFrontEndPathCarriesAWalkabout(t *testing.T) {
	m := worldFixtureMap()

	t.Run("the campaign path", func(t *testing.T) {
		v := worldFixtureViewer(t, m)
		mw := openMission(&Mission{Number: 7, Map: m, World: mapload.FromALM(m)}, nil, nil, v, nil, nil, nil)
		if got := len(mw.sched); got != 0 {
			t.Errorf("openMission carries a %d-entry schedule; the placeholder walkabout is out of the game", got)
		}
	})

	t.Run("the picker path", func(t *testing.T) {
		v := worldFixtureViewer(t, m)
		mw, err := openMapWorld(m, nil, nil, v)
		if err != nil {
			t.Fatalf("openMapWorld: %v", err)
		}
		if got := len(mw.sched); got != 0 {
			t.Errorf("openMapWorld carries a %d-entry schedule; the placeholder walkabout is out of the game", got)
		}
	})

	// And the generator itself still works, or the test above is vacuous for a
	// reason that has nothing to do with the front end.
	if got := len(placeholderSchedule(m)); got == 0 {
		t.Fatal("placeholderSchedule built nothing over the fixture map: the two assertions above prove nothing")
	}
}
