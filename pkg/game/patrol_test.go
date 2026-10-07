package game_test

// The mission's own patrol arm (0099 AC-2): mission 10's two Patrol nodes,
// run with no player commands at all, walk their two placed villagers off
// the cell the map put them on and back to it.
//
// It reads a LAWFUL INSTALL and is therefore skipped unless an asset root
// is configured, on cmd/missionrun's own TestTheTenthMissionRunsEndToEndOnALawfulInstall
// terms — the suite stays green with no game present, and nothing here is
// a fixture.

import (
	"os"
	"testing"

	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// patrolTrack is one patroller's whole story over a run: where it started,
// whether it has left that cell, whether it has come back, and the
// furthest cell it reached — kept for a readable failure message rather
// than for its own sake.
type patrolTrack struct {
	sx, sy         int32
	left, returned bool
	fx, fy         int32
}

// chebyshevFromStart is a cell's Chebyshev distance from the track's own
// start, which is what "furthest reached" is measured by — the same
// metric this package's own map arithmetic uses throughout.
func (p *patrolTrack) chebyshevFromStart(x, y int32) int32 {
	dx, dy := x-p.sx, y-p.sy
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

// TestPatrolWalksTheMissionTensPatrollers is AC-2. The two units are
// resolved through mapload.ScriptUnits by their MAP ids — 57 and 58 —
// rather than by a slice index: a slice index is a fact about placement
// order, and the mission's own script names these two by id. Their
// placement cells, and the cells they are commanded to, are read off the
// started world itself rather than hardcoded, so this test does not trust
// the numbers that motivated it — only that whatever the map placed them
// on, they leave it and come back.
func TestPatrolWalksTheMissionTensPatrollers(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the mission-10 patrol drive needs a lawful install")
	}
	archives, err := game.OpenArchives(root)
	if err != nil {
		t.Fatalf("OpenArchives: %v", err)
	}
	ms, err := game.StartMission(archives.Containers, 10, nil, mapload.DifficultyNormal, nil)
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}

	ids := mapload.ScriptUnits(ms.Map, ms.Party)
	patrollers := make([]sim.EntityID, 0, 2)
	for _, mapID := range []uint16{57, 58} {
		id, ok := ids[mapID]
		if !ok {
			t.Fatalf("map id %d resolves to no entity — this install's mission 10 does not carry the "+
				"fixture this test assumes", mapID)
		}
		patrollers = append(patrollers, id)
	}

	cellOf := func(id sim.EntityID) (int32, int32) {
		for _, e := range ms.World.Entities() {
			if e.ID == id {
				return e.X, e.Y
			}
		}
		t.Fatalf("entity %d is no longer in the world", id)
		return 0, 0
	}

	tracks := make(map[sim.EntityID]*patrolTrack, len(patrollers))
	for _, id := range patrollers {
		x, y := cellOf(id)
		tracks[id] = &patrolTrack{sx: x, sy: y, fx: x, fy: y}
	}

	// A build that never dispatches the patrol arm leaves both standing —
	// AC-2's own control — so the bound below only has to be generous
	// enough for a real ring walk, never tight enough to risk flaking on
	// one.
	const ticks = 20000
	for i := 0; i < ticks; i++ {
		sim.Step(ms.World, nil)
		for _, id := range patrollers {
			tr := tracks[id]
			x, y := cellOf(id)
			switch {
			case !tr.left:
				if x != tr.sx || y != tr.sy {
					tr.left = true
				}
			case !tr.returned:
				if x == tr.sx && y == tr.sy {
					tr.returned = true
				}
			}
			if tr.chebyshevFromStart(x, y) > tr.chebyshevFromStart(tr.fx, tr.fy) {
				tr.fx, tr.fy = x, y
			}
		}
	}

	for _, id := range patrollers {
		tr := tracks[id]
		if !tr.left {
			t.Errorf("entity %d never left its placement cell (%d,%d) in %d ticks",
				id, tr.sx, tr.sy, ticks)
			continue
		}
		if !tr.returned {
			t.Errorf("entity %d left its placement cell (%d,%d) but never came back to it in %d ticks; "+
				"furthest reached was (%d,%d)", id, tr.sx, tr.sy, ticks, tr.fx, tr.fy)
		}
	}
}
