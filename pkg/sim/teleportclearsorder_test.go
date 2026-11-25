package sim

import (
	"bytes"
	"testing"
)

// A first-game unit under a Move order that a spell-26 trap moves stays at the
// destination: the reset clears its order (MOVE-129, AI-535).
func TestFirstGameTeleportTrapClearsTheMoveOrder(t *testing.T) {
	hero := engFighterOne(1, SelfSlot, 8, 10)
	hero.Speed = 16
	w := firstTeleportWorld(t, hero)
	cmds := []Command{MoveTo(1, CellPoint{X: 12, Y: 10})}
	moved := 0
	for tick := 1; tick <= 400 && (moved == 0 || tick <= moved+64); tick++ {
		Step(w, cmds)
		cmds = nil
		if e, _ := w.Entity(1); moved == 0 && e.X == 30 && e.Y == 40 {
			moved = tick
		}
	}
	if moved == 0 {
		t.Fatal("the unit was never moved")
	}
	if e, _ := w.Entity(1); e.X != 30 || e.Y != 40 {
		t.Fatalf("64 ticks after the move the unit is at (%d,%d), want (30,40)", e.X, e.Y)
	}
	assertSaveRoundTrip(t, w)
}

// A unit moved by the Teleport action under a Move order stays at the target,
// and a member of its group that was not moved keeps its order.
func TestFirstGameTeleportActionClearsTheMoveOrder(t *testing.T) {
	s, err := NewExtendedScript(100, []ScriptCheck{
		{Op: ScriptCheckNearest, Register: 0, Player: SelfSlot, HasPlayer: true, Args: [scriptParams]int32{10, 10}},
		constCheck(1, 0),
	}, []ScriptInstant{
		{Op: ScriptInstantTeleport, Args: [scriptParams]int32{10, 10, 30, 40}},
	}, []ScriptTrigger{
		{Instants: acts(), Once: true, Latch: 0, Conditions: []ScriptPair{pair(0, 1, ScriptCmpEQ)}, Actions: []int32{0}},
	})
	if err != nil {
		t.Fatal(err)
	}
	u1 := engFighterOne(1, SelfSlot, 8, 10)
	u1.Speed = 16
	u2 := engFighterOne(2, SelfSlot, 6, 20)
	u2.Speed = 16
	w := teleportActionWorld(t, s, Terrain{}, u1, u2)
	cmds := []Command{MoveTo(1, CellPoint{X: 12, Y: 10}), MoveTo(2, CellPoint{X: 6, Y: 30})}
	moved := 0
	for tick := 1; tick <= 1000; tick++ {
		Step(w, cmds)
		cmds = nil
		e, _ := w.Entity(1)
		if moved == 0 && e.X == 30 && e.Y == 40 {
			moved = tick
		}
		if moved != 0 && tick == moved+64 && (e.X != 30 || e.Y != 40) {
			t.Fatalf("64 ticks after the move the unit is at (%d,%d), want (30,40)", e.X, e.Y)
		}
	}
	if moved == 0 {
		t.Fatal("the unit was never moved")
	}
	if e, _ := w.Entity(1); e.X != 30 || e.Y != 40 {
		t.Fatalf("the unit ended at (%d,%d), want (30,40)", e.X, e.Y)
	}
	if e, _ := w.Entity(2); e.X != 6 || e.Y != 30 {
		t.Fatalf("the other member is at (%d,%d), want its order's destination (6,30)", e.X, e.Y)
	}
	assertSaveRoundTrip(t, w)
}

// The second game's teleport keeps the order: the unit stands for the move
// body only where the profile says so, and a Move order walks it back.
func TestSecondGameTrapKeepsTheMoveOrder(t *testing.T) {
	hero := engFighterOne(1, SelfSlot, 8, 10)
	hero.Speed = 16
	w := trapWorld(t, hero)
	w.setCellTail(trapNode())
	cmds := []Command{MoveTo(1, CellPoint{X: 12, Y: 10})}
	moved := 0
	for tick := 1; tick <= 2000; tick++ {
		Step(w, cmds)
		cmds = nil
		if e, _ := w.Entity(1); moved == 0 && e.X == 30 && e.Y == 40 {
			moved = tick
		}
	}
	if moved == 0 {
		t.Fatal("the unit was never moved")
	}
	if e, _ := w.Entity(1); e.X != 12 || e.Y != 10 {
		t.Fatalf("the unit ended at (%d,%d), want its order's destination (12,10)", e.X, e.Y)
	}
}

func assertSaveRoundTrip(t *testing.T, w *World) {
	t.Helper()
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var loaded World
	if err := loaded.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	again, err := loaded.MarshalBinary()
	if err != nil || !bytes.Equal(raw, again) || loaded.Hash() != w.Hash() {
		t.Fatalf("world save round-trip differed: %v", err)
	}
}
