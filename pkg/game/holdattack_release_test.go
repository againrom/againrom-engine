package game

import "testing"

func TestReleaseHoldPositionAttacksACreatureInReach(t *testing.T) {
	arena := openLoadedCycleArena(t, 1)
	hero := arena.heroes[0]
	arena.selectFirstHero(t)
	holdPress(t, arena, hero)
	held := arena.get(hero)
	full := arena.get(arena.east).HP
	lowest := full
	for range 600 {
		arena.live.tick()
		lowest = min(lowest, arena.get(arena.east).HP)
	}
	h := arena.get(hero)
	t.Logf("creature token size %d; its health fell from %d to %d", arena.get(arena.east).TokenSize, full, lowest)
	if h.X != held.X || h.Y != held.Y {
		t.Fatalf("the warrior left his cell (%d,%d) for (%d,%d)", held.X, held.Y, h.X, h.Y)
	}
	if lowest >= full {
		t.Fatalf("600 ticks on Hold Position beside a creature: its health never fell below %d", full)
	}
}
