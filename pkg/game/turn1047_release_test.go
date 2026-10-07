package game_test

import (
	"os"
	"testing"

	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestReleaseAMissionTenPatrolReversalAdvancesTheDrawnOctant runs the same
// no-command mission-10 drive as TestPatrolWalksTheMissionTensPatrollers,
// watching for a tick at which either patroller holds an ACTIVE turn of more
// than one remaining tick (TurnRemaining > 1, i.e. an arc bigger than one
// octant-width — the shape a reversal produces and a single-direction snap
// never does; requestFacing's own one-direction arm always finishes in
// exactly one tick). Across such an interval, Facing itself must stay at its
// pre-turn value the whole time and sim.Entity.DrawnFacing must visit more
// than the two endpoint octants — a static drawn value for the whole
// interval is exactly round 2's own defect, now measured against real
// shipped RotationSpeed and arc combinations instead of only the synthetic
// ones turn1047_test.go's fixture chooses.
//
// THIS IS A SIM-LEVEL WITNESS, NOT A RENDER-LEVEL ONE: it calls
// sim.Entity.DrawnFacing() directly and never reaches pkg/game/world.go's
// own oct := sheetOctant(e.DrawnFacing()) call site, because package
// game_test cannot see that unexported line. The call site itself is
// witnessed by TestTheDrawnOctantAdvancesAcrossAMultiTickTurn
// (turn1047_test.go), a synthetic fixture that does reach it and does fail
// under that mutation. The two together are what "shipped content" and "the
// render wiring" both need: this test supplies the first on the real
// RotationSpeed/arc population a lawful install ships, that one supplies the
// second.
func TestReleaseAMissionTenPatrolReversalAdvancesTheDrawnOctant(t *testing.T) {
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

	entityOf := func(id sim.EntityID) sim.Entity {
		for _, e := range ms.World.Entities() {
			if e.ID == id {
				return e
			}
		}
		t.Fatalf("entity %d is no longer in the world", id)
		return sim.Entity{}
	}

	type interval struct {
		startTick        int
		startFacing      uint8
		distinctOctants  map[int]bool
		facingMoved      bool
		longestRemaining uint8
	}
	open := make(map[sim.EntityID]*interval, len(patrollers))
	var closedLong []interval // completed intervals that reached TurnRemaining > 1 at some point

	octantOf := func(e sim.Entity) int { return sheetOctantForTest(e.DrawnFacing()) }

	const ticks = 20000
	for tick := 0; tick < ticks; tick++ {
		sim.Step(ms.World, nil)
		for _, id := range patrollers {
			e := entityOf(id)
			iv := open[id]
			if e.Turning() {
				if iv == nil {
					iv = &interval{startTick: tick, startFacing: e.Facing, distinctOctants: map[int]bool{}}
					open[id] = iv
				}
				iv.distinctOctants[octantOf(e)] = true
				if e.Facing != iv.startFacing {
					iv.facingMoved = true
				}
				if e.TurnRemaining > iv.longestRemaining {
					iv.longestRemaining = e.TurnRemaining
				}
			} else if iv != nil {
				if iv.longestRemaining > 1 {
					closedLong = append(closedLong, *iv)
				}
				delete(open, id)
			}
		}
	}
	// An interval still active when the drive ends is not scored: its final
	// snap tick, the one thing that would prove Facing never drifted mid-turn,
	// has not happened yet.

	if len(closedLong) == 0 {
		t.Fatalf("mission 10's patrol drive completed %d ticks with no multi-tick turn "+
			"(TurnRemaining briefly above 1) on either patroller; this test cannot witness "+
			"a reversal that did not occur", ticks)
	}
	for _, iv := range closedLong {
		if iv.facingMoved {
			t.Errorf("interval starting tick %d: Facing changed mid-turn; it must hold its "+
				"pre-turn value for the whole interval (advanceTurns)", iv.startTick)
		}
		if len(iv.distinctOctants) < 3 {
			t.Errorf("interval starting tick %d (longest remaining %d): drawn octant visited "+
				"only %d distinct values, want more than the two endpoints; a static octant "+
				"for the whole interval is round 2's own defect", iv.startTick, iv.longestRemaining, len(iv.distinctOctants))
		}
	}
	t.Logf("%d multi-tick turn interval(s) witnessed across %d ticks; example longest remaining %d",
		len(closedLong), ticks, closedLong[0].longestRemaining)
}

// TestReleaseAMissionTenTurnHashMatchesAcrossBothLawfulRoots is remaining-
// surface item 3 (adversarial return, section 7): RotationSpeed is hashed
// simulation state, and every measurement in this story that could differ
// by root had been reported identical without ever comparing a World.Hash()
// directly. This drives the same no-command mission-10 patrol run this file
// already uses (real turns confirmed active by the reversal test above) on
// both lawful roots and compares the digest at a fixed tick.
func TestReleaseAMissionTenTurnHashMatchesAcrossBothLawfulRoots(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the mission-10 patrol drive needs a lawful install")
	}
	// A GRACEFUL DEGRADE, NOT A SKIP. installtext_test.go's own two AGAINROM_
	// ASSETS_RU consumers (TestOriginalUITextWordSetExactDifferencesOverBoth-
	// LawfulInstalls and its sibling) log and return rather than call t.Skip
	// when the second root is absent, so a single-root invocation of
	// check-release-tests.sh never counts them as "still skipped" against an
	// AGAINROM_ variable and never fails for want of a root this run was not
	// asked to supply. A t.Skip here broke exactly that: any single-root run
	// of the gate, EN or RU, always reported one remaining skip and exited 1,
	// because this was the first two-root test in the package to use t.Skip
	// instead of the established log-and-return shape. Matched to convention.
	ruRoot := os.Getenv("AGAINROM_ASSETS_RU")
	if ruRoot == "" {
		t.Log("no AGAINROM_ASSETS_RU: the cross-root hash comparison was not selected")
		return
	}

	const ticks = 500 // well past the reversal test's own first multi-tick interval
	hashAt := func(assetsRoot string) uint64 {
		archives, err := game.OpenArchives(assetsRoot)
		if err != nil {
			t.Fatalf("OpenArchives(%q): %v", assetsRoot, err)
		}
		ms, err := game.StartMission(archives.Containers, 10, nil, mapload.DifficultyNormal, nil)
		if err != nil {
			t.Fatalf("StartMission(%q): %v", assetsRoot, err)
		}
		for i := 0; i < ticks; i++ {
			sim.Step(ms.World, nil)
		}
		return ms.World.Hash()
	}

	enHash, ruHash := hashAt(root), hashAt(ruRoot)
	if enHash != ruHash {
		t.Fatalf("World.Hash() after %d ticks of mission 10's patrol drive: EN=%#016x RU=%#016x, want equal",
			ticks, enHash, ruHash)
	}
	t.Logf("World.Hash() after %d ticks matches on both roots: %#016x", ticks, enHash)
}

// sheetOctantForTest mirrors pkg/game's own unexported sheetOctant exactly
// (sim.FacingDir(f)+4)&7 — this file is package game_test and cannot reach
// the unexported production function, so this is a second, independent
// transcription of the same one-line translation swing_test.go's own
// TestASheetOctantIsTheSimulationsOwnDirection already checks against
// production. It exists only to group this test's octants; it asserts
// nothing about the translation itself.
func sheetOctantForTest(facing uint8) int { return (sim.FacingDir(facing) + 4) & 7 }
