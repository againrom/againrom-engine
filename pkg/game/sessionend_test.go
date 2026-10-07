package game

// The owner's own reproduction: leave a running game for the main menu (the
// brooch), pick any mission from there, die, and the town that loads is the
// PREVIOUS game's — its chapter, its gold, its finished missions. This
// file witnesses the fix through the exact production path a player takes,
// not through resetSessionForNewGame's own unit test alone
// (frontend_session_test.go, which proves the reset method's field coverage
// but never proves the method is actually WIRED into starting a new game).

import (
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestLeavingToTheMenuAndPickingAMissionDoesNotResumeThePreviousGamesTown
// drives the owner's own repro at the front-end level: a first game reaches
// the town (winning mission 20 opens it, on continuityFront's own fixture
// campaign — the same one TestAWinAtTheTownsBoundaryOpensTheTown uses), then
// the player is taken to leave for the main menu and choose ANY mission
// again. That choice reaches newGameChargen exactly as pkg/ui's picker
// reaches it (flow.go's choose(), through the ChargenGate) and its Begin
// closure is called exactly as armChargen's confirmation calls it — this
// test calls the same two methods, not a stand-in.
//
// THE OBSERVABLE CHECK IS A LOST MISSION'S OWN DESTINATION, not a field read
// off Town directly, because that is what a player actually sees: mission 10
// has no town of its own in the fixture campign, and TestALostMissionStillReachesTheMenuBeforeTheTownIsOpen
// already establishes that a lost mission goes to ui.NoticeToMenu whenever
// the town has not been reached, and to ui.NoticeToTown whenever it has been
// — which is exactly the fork the stale Town.open latch corrupted: reported,
// the second game's first loss returned to the FIRST game's own town,
// because the town's open latch and its chapter/gold/finished state all
// carried over uncleared.
func TestLeavingToTheMenuAndPickingAMissionDoesNotResumeThePreviousGamesTown(t *testing.T) {
	f := continuityFront(t)
	// The reset now commits only after the returned opener succeeds. Supply a
	// synthetic mission archive so this test crosses that production boundary.
	mapFront := missionFrontEnd(t)
	f.Archives, f.Tiles = mapFront.Archives, mapFront.Tiles

	// The first game: win mission 20, which opens the town at the fixture
	// campaign's own boundary (TestAWinAtTheTownsBoundaryOpensTheTown).
	f.continuity(20, continuityMission(t, 20, sim.ScriptInstantWin), listAdvance)()
	if !f.Town.Open() {
		t.Fatal("setup: the first game's town did not open")
	}
	if len(f.Carried) == 0 {
		t.Fatal("setup: the first game's win did not carry a party")
	}
	if f.Offered == 0 {
		t.Fatal("setup: the first game's win did not offer a next mission")
	}

	// The player leaves for the main menu here (pkg/ui's own escape()/
	// chooseGameMenu paths, which this package cannot reach directly — see
	// this file's own header). What the front end can and must witness is
	// what happens when NEW GAME is picked again afterwards: this is that
	// choice, over row 0, which the fixture campaign's own mission 10 owns.
	f.Maps = []MapEntry{{Mission: 10}}
	entry := f.newGameChargen(0)
	if entry == nil {
		t.Fatal("setup: row 0 did not open generation")
	}
	open, err := entry.Begin(ui.ChargenResult{})
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
		t.Fatal(err)
	}

	// The second game's FIRST mission is lost, exactly as the owner's own
	// report describes ("умереть"). Before the fix this returned
	// ui.NoticeToTown, holding the FIRST game's own Town — reachable because
	// f.Town.open was never cleared. After the fix, mission 10 has no town of
	// its own reached yet in the SECOND game, and this must read exactly as
	// TestALostMissionStillReachesTheMenuBeforeTheTownIsOpen's own case does.
	dest, _, _ := f.continuity(10, continuityMission(t, 10, sim.ScriptInstantLose), menuAdvance)()
	if dest != ui.NoticeToMenu {
		t.Errorf("destination %v after the second game's first loss, want ui.NoticeToMenu — "+
			"got ui.NoticeToTown, which means the front end is still showing the FIRST game's own "+
			"town instead of a fresh one", dest)
	}

	if f.Town.Open() {
		t.Error("Town.Open() is true after starting a new game — the previous game's town survived the reset")
	}
	if len(f.Carried) != 0 {
		t.Errorf("Carried = %+v after starting a new game, want none — the previous game's party survived the reset", f.Carried)
	}
	if f.Offered != 0 {
		t.Errorf("Offered = %d after starting a new game, want 0 — the previous game's map-list bookkeeping survived the reset", f.Offered)
	}
}
