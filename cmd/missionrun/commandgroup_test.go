package main

import (
	"os"
	"testing"

	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestATenthMissionHostileNeverHoldsItsOwnPostAfterBeingCommanded is 0117's
// AC-1: on the tenth mission, against a lawful install, a hostile commanded
// away from its placement never afterwards holds its own post as a
// destination. Before 0117 the same drive gives it that destination one tick
// after it arrives — this is that observation made direct and assertable,
// on script unit 21, the same unit TestTheTenthMissionRunsEndToEndOnALawfulInstall drives.
//
// It reads a LAWFUL INSTALL and is therefore skipped without one, on that
// test's own precedent: the suite stays green with no game present, and this
// file cannot be executed in an environment with no asset root configured.
func TestATenthMissionHostileNeverHoldsItsOwnPostAfterBeingCommanded(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("no AGAINROM_ASSETS: the campaign drive needs a lawful install")
	}

	root := game.ResolveAssetRoot("", os.Getenv("AGAINROM_ASSETS"))
	archives, err := game.OpenArchives(root)
	if err != nil {
		t.Fatalf("OpenArchives: %v", err)
	}
	// LoadDefinitions, not LoadTable — the same walk main.go's own run() takes,
	// so this fixture starts the identical party the game and the milestone
	// test both would.
	defs, err := game.LoadDefinitions(archives.Containers)
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}
	ms, err := game.StartMission(archives.Containers, 10, defs.Table,
		mapload.DifficultyNormal, game.MissionParty(defs.StartWeapon, defs.Bodies, defs.Table))
	if err != nil {
		t.Fatalf("StartMission(10): %v", err)
	}
	w := ms.World
	script := mapload.ScriptUnits(ms.Map, ms.Party)
	id, ok := script[21]
	if !ok {
		t.Fatal("mission 10's script names no unit 21 — this test's own fixture assumption is wrong")
	}
	if _, held := hpOf(w, id); !held {
		t.Fatal("mission 10's world holds no entity for script unit 21 at start")
	}

	postX, postY := at(w, id)

	// Aim at an open cell a few cells off the placement, reusing the tool's
	// own aim/blocked machinery rather than a bare offset that might land in
	// a wall: four candidate directions, so a placement hard against the
	// map's edge on one axis still finds an order that sticks.
	block := mapload.PassabilityWith(ms.Map, defs.Table)
	width, height := int32(ms.Map.Width), int32(ms.Map.Height)
	blocked := func(x, y int32) bool {
		if x < 0 || y < 0 || x >= width || y >= height {
			return true
		}
		return block[int(y)*int(width)+int(x)]&0x01 != 0
	}
	var destX, destY int32
	found := false
	for _, off := range [4][2]int32{{5, 0}, {-5, 0}, {0, 5}, {0, -5}} {
		p := waypoint{x: postX + off[0], y: postY + off[1], radius: 3}
		if x, y, ok := aim(postX, postY, p, blocked, width, height); ok {
			destX, destY, found = x, y, true
			break
		}
	}
	if !found {
		t.Fatal("no open cell within 3 of five cells off unit 21's placement in any of four " +
			"directions — this test's own fixture assumption is wrong")
	}

	left := false
	leftAt := -1
	for i := 0; i < 20000 && w.Outcome() == sim.OutcomeUndecided; i++ {
		var cmds []sim.Command
		if i == 0 {
			cmds = []sim.Command{{Kind: sim.KindMoveTo, Entity: id, X: destX, Y: destY}}
		}
		sim.Step(w, cmds)

		e := entityAt(w, id)
		if !left && (e.X != postX || e.Y != postY) {
			left, leftAt = true, i
		}
		if left && e.HasTarget && e.TargetX == postX && e.TargetY == postY {
			t.Fatalf("tick %d: script unit 21 holds its own placement (%d,%d) as a destination after "+
				"being commanded away to (%d,%d) — the walk home this story removes for a commanded actor",
				w.Tick(), postX, postY, destX, destY)
		}
		// Once it has demonstrably left, the spec's own account of the defect
		// (arrival, then the walk home one tick later) fires within single
		// digits of ticks; a few hundred past leaving is ample without
		// running the whole tick ceiling for a fixture that has already
		// shown what it needs to.
		if left && i-leftAt > 500 {
			break
		}
	}
	if !left {
		t.Fatal("script unit 21 never left its placement cell — this test's own order did not stick")
	}
}
