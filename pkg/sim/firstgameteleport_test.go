package sim

import "testing"

// secondCellEvent is the second game's cell move event (R2-ENGINE-1778).
const secondCellEvent = 23

// firstCellMove is the first game's relocating cell spell (TRIG-CELLTAIL-035).
const firstCellMove = 26

// firstTeleport is a first-game cell record of spell 26 on (10,10) whose
// tail bytes 4 and 5 name (30,40), as a type-9 record writes it
// (TRIG-CELLTAIL-035, TERR-229).
var firstTeleport = CellTail{X: 10, Y: 10, Bytes: [cellTailLen]byte{firstCellMove, 7, 10, 10, 30, 40}}

func firstTeleportWorld(t *testing.T, ents ...Entity) *World {
	t.Helper()
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{}, ents, mustScript(t, nil, nil, nil), engRel(t))
	if err != nil {
		t.Fatal(err)
	}
	if w.SpellGame() != newGameProfile(ScriptROM1).spellGame {
		t.Fatal("the fixture is not a first-game world")
	}
	wakeGroups(w)
	if err := w.DeclareCellTails([]CellTail{firstTeleport}); err != nil {
		t.Fatal(err)
	}
	return w
}

// A first-game unit walking across a spell-26 cell is moved to tail bytes 4
// and 5 once its step ends centred on the cell, and then stands there with
// state 0xc, no walk and its post at the new cell (MOVE-129, AI-535). The
// record stays.
func TestFirstGameTeleportTrapMovesTheUnitWhoseStepEndsOnIt(t *testing.T) {
	hero := engFighterOne(1, SelfSlot, 8, 10)
	hero.Speed = 16
	w := firstTeleportWorld(t, hero)
	cmds := []Command{MoveTo(1, CellPoint{X: 12, Y: 10})}
	for tick := 1; tick <= 200; tick++ {
		Step(w, cmds)
		cmds = nil
		e, _ := w.Entity(1)
		if e.X == 30 && e.Y == 40 {
			if e.HasTarget || e.ActorState != actorStateAcquire || e.PostX != 30 || e.PostY != 40 {
				t.Fatalf("after the move: target %t state %#x post (%d,%d)", e.HasTarget, e.ActorState, e.PostX, e.PostY)
			}
			if got := w.CellTails(); len(got) != 1 || got[0] != firstTeleport {
				t.Fatalf("tails after the move %v, want the record kept", got)
			}
			return
		}
		if e.X > 10 {
			t.Fatalf("the unit walked on to (%d,%d) on tick %d", e.X, e.Y, tick)
		}
	}
	t.Fatal("the unit never reached the destination")
}

// A destination the footprint refuses leaves the unit on the trap cell
// (MOVE-129: both footprint checks must pass).
func TestFirstGameTeleportTrapRefusedDestinationKeepsTheUnit(t *testing.T) {
	w := firstTeleportWorld(t, Entity{ID: 1, HP: 50, MaxHP: 50, X: 10, Y: 10, Domain: DomainGround},
		Entity{ID: 2, HP: 50, MaxHP: 50, X: 30, Y: 40, Domain: DomainGround})
	w.stepCentred(0)
	if casts := w.ScriptCasts(); len(casts) != 1 {
		t.Fatalf("the centred step queued %v, want the move", casts)
	}
	w.stepScriptCasts(nil)
	if e, _ := w.Entity(1); e.X != 10 || e.Y != 10 {
		t.Fatalf("unit at (%d,%d), want it kept on a refused destination", e.X, e.Y)
	}
}

// A trap whose target is (0,0) queues nothing at the step's centre, so no
// Teleport spell is cast at the unit in its place.
func TestFirstGameTeleportTrapToCellZeroQueuesNothing(t *testing.T) {
	w := firstTeleportWorld(t, Entity{ID: 1, HP: 50, MaxHP: 50, X: 10, Y: 10, Domain: DomainGround})
	w.writeCellTail(tailKey(10, 10), [cellTailLen]byte{firstCellMove, 7, 10, 10, 0, 0})
	w.stepCentred(0)
	if casts := w.ScriptCasts(); len(casts) != 0 {
		t.Fatalf("a trap to (0,0) queued %v", casts)
	}
}

// An air unit is not moved: the arrival arm, like the entry arm, takes
// ground and ghost units only (UNIT-M10ENTRY-055).
func TestFirstGameTeleportTrapLeavesAnAirUnit(t *testing.T) {
	w := firstTeleportWorld(t, Entity{ID: 1, HP: 50, MaxHP: 50, X: 10, Y: 10, Domain: DomainAir})
	w.requestCellCast(0, 10, 10)
	w.stepCentred(0)
	if casts := w.ScriptCasts(); len(casts) != 0 {
		t.Fatalf("an air unit queued %v", casts)
	}
	w.stepScriptCasts(nil)
	if e, _ := w.Entity(1); e.X != 10 || e.Y != 10 {
		t.Fatalf("an air unit moved to (%d,%d)", e.X, e.Y)
	}
}

// A tail of another spell keeps the entry cast and queues no move at the
// step's centre (TRIG-CELLTAIL-035, UNIT-M10ENTRY-055).
func TestFirstGameOtherCellSpellStillCasts(t *testing.T) {
	w, err := NewSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, nil,
		[]Entity{{ID: 7, X: 2, Y: 3, HP: 1000, MaxHP: 1000, Domain: DomainGround}}, nil,
		[]SpellRule{{ID: 13, School: 3, DamageMin: 20, DamageMax: 20, Damaging: true, TargetsUnit: true}})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.DeclareCellTails([]CellTail{{X: 3, Y: 3, Bytes: [cellTailLen]byte{13, 1, 9, 10, 11, 12}}}); err != nil {
		t.Fatal(err)
	}
	Step(w, []Command{MoveTo(7, CellPoint{X: 3, Y: 3})})
	casts := w.ScriptCasts()
	if len(casts) != 1 || casts[0] != (ScriptCast{FromX: 9, FromY: 10, Spell: 13, Power: 1, Target: 7, AtUnit: true}) {
		t.Fatalf("entry casts %+v, want the spell-13 cast alone", casts)
	}
	if e, _ := w.Entity(7); e.X != 3 || e.Y != 3 {
		t.Fatalf("unit at (%d,%d)", e.X, e.Y)
	}
}

// Instant 25 of spell 26 keeps the original's six bytes: x is not stored,
// so the destination is (0, y) (TRIG-CELLTAIL-035), and a LOAD's repair
// leaves it.
func TestFirstGameInstantTwentyFiveTeleportKeepsItsBytes(t *testing.T) {
	node := ScriptInstant{Op: ScriptInstantCellTail, Args: [scriptParams]int32{firstCellMove, 7, 10, 12}}
	w := scriptWorld(t, mustScript(t, nil, []ScriptInstant{node}, nil), []Entity{{ID: 1, HP: 50, MaxHP: 50, X: 5, Y: 5}})
	w.setCellTail(node)
	w.RepairCellEvents()
	want := [cellTailLen]byte{firstCellMove, 7, 0, 12, 0, 12}
	if got := w.CellTails(); len(got) != 1 || got[0].X != 10 || got[0].Y != 12 || got[0].Bytes != want {
		t.Fatalf("tails %v, want one at (10,12) with %v", got, want)
	}
}
