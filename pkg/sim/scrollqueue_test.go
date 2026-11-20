package sim

import (
	"reflect"
	"testing"
)

// scrollQueueWorld is the manual caster carrying a stack of teleport scrolls
// (index 0) and one scroll of the damaging spell (index 1), in the profile of
// the named game.
func scrollQueueWorld(t *testing.T, second bool, teleports uint32) *World {
	t.Helper()
	w := manualCastWorld(t)
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	w.carried[0] = []ItemStack{
		StackItem(ItemInstance{Code: 0xe10, Kind: 4, Price: 50, Effects: []ItemEffect{{Kind: 41, Operand: 26 | 60<<16}}}, teleports),
		StackItem(ItemInstance{Code: 0xe10, Kind: 4, Price: 51, Effects: []ItemEffect{{Kind: 41, Operand: 1 | 60<<16}}}, 1),
	}
	return w
}

func scrollQueueGames(t *testing.T, run func(t *testing.T, second bool)) {
	for _, second := range []bool{false, true} {
		name := "first"
		if second {
			name = "second"
		}
		t.Run(name, func(t *testing.T) { run(t, second) })
	}
}

func scrollQueuedCount(w *World) int {
	n := 0
	for _, c := range w.scrollCasts {
		if c.Queued {
			n++
		}
	}
	return n
}

func teleportStack(w *World) uint32 {
	if len(w.carried[0]) == 0 || w.carried[0][0].Price != 50 {
		return 0
	}
	return w.carried[0][0].Count
}

// Two teleport scrolls used in a row both cast, the second after the first.
func TestScrollQueueTwoTeleportsBothCast(t *testing.T) {
	scrollQueueGames(t, func(t *testing.T, second bool) {
		w := scrollQueueWorld(t, second, 2)
		first, next := CellPoint{X: 6, Y: 5}, CellPoint{X: 8, Y: 7}
		Step(w, []Command{UseScrollAt(1, 0, first)})
		Step(w, []Command{UseScrollAt(1, 0, next)})
		if scrollQueuedCount(w) != 1 || len(w.scrollCasts) != 2 || w.scrollCasts[0].Queued {
			t.Fatalf("second scroll not queued behind the first: %+v", w.scrollCasts)
		}
		var seen []CellPoint
		for n := 0; n < 400 && (len(w.scrollCasts) != 0 || len(seen) < 2); n++ {
			Step(w, nil)
			at := CellPoint{X: w.entities[0].X, Y: w.entities[0].Y}
			if (at == first || at == next) && (len(seen) == 0 || seen[len(seen)-1] != at) {
				seen = append(seen, at)
			}
		}
		if !reflect.DeepEqual(seen, []CellPoint{first, next}) || len(w.carried[0]) != 1 || len(w.scrollCasts) != 0 {
			t.Fatalf("relocations %v, pack %+v, casts %+v", seen, w.carried[0], w.scrollCasts)
		}
	})
}

// A scroll used during a book wind-up leaves the wind-up running and casts
// after the release.
func TestScrollQueueWaitsBehindBookWindUp(t *testing.T) {
	scrollQueueGames(t, func(t *testing.T, second bool) {
		w := scrollQueueWorld(t, second, 1)
		target := CellPoint{X: 6, Y: 5}
		Step(w, []Command{Cast(1, 2, 1)})
		if !w.orderStateAt(0).charging() {
			t.Fatal("no wind-up to wait behind")
		}
		Step(w, []Command{UseScrollAt(1, 0, target)})
		if scrollQueuedCount(w) != 1 || !w.orderStateAt(0).charging() || teleportStack(w) != 0 {
			t.Fatalf("scroll did not wait: casts %+v book %+v", w.scrollCasts, w.bookCasts)
		}
		hurt, moved := -1, -1
		for n := 0; n < 400 && moved < 0; n++ {
			Step(w, nil)
			if hurt < 0 && w.entities[1].HP < 100 {
				hurt = n
			}
			if w.entities[0].X == target.X && w.entities[0].Y == target.Y {
				moved = n
			}
		}
		if hurt < 0 || moved < 0 || moved <= hurt || len(w.scrollCasts) != 0 {
			t.Fatalf("book hit at %d, teleport at %d, casts %+v", hurt, moved, w.scrollCasts)
		}
	})
}

// A scroll used during a cast recovery waits for the recovery to end.
func TestScrollQueueWaitsBehindRecovery(t *testing.T) {
	scrollQueueGames(t, func(t *testing.T, second bool) {
		w := scrollQueueWorld(t, second, 1)
		target := CellPoint{X: 6, Y: 5}
		w.entities[0].CastWait = 40
		Step(w, []Command{UseScrollAt(1, 0, target)})
		if scrollQueuedCount(w) != 1 || w.entities[0].CastWait == 0 {
			t.Fatalf("scroll did not wait: casts %+v wait %d", w.scrollCasts, w.entities[0].CastWait)
		}
		moved := -1
		for n := 1; n < 400 && moved < 0; n++ {
			Step(w, nil)
			if w.entities[0].X == target.X && w.entities[0].Y == target.Y {
				moved = n
			}
		}
		if moved < 30 || len(w.scrollCasts) != 0 || len(w.carried[0]) != 1 {
			t.Fatalf("teleport at %d, casts %+v, pack %+v", moved, w.scrollCasts, w.carried[0])
		}
	})
}

// A waiting scroll whose target is gone before it starts returns its item.
func TestScrollQueueRefundsWhenTargetDies(t *testing.T) {
	scrollQueueGames(t, func(t *testing.T, second bool) {
		w := scrollQueueWorld(t, second, 1)
		w.entities[0].CastWait = 200
		Step(w, []Command{UseScroll(1, 1, 2)})
		if scrollQueuedCount(w) != 1 || len(w.carried[0]) != 1 {
			t.Fatalf("scroll did not wait: casts %+v pack %+v", w.scrollCasts, w.carried[0])
		}
		Step(w, []Command{Damage(2, 500)})
		for n := 0; n < 20 && len(w.scrollCasts) != 0; n++ {
			Step(w, nil)
		}
		if len(w.scrollCasts) != 0 || len(w.carried[0]) != 2 || w.carried[0][1].Count != 1 {
			t.Fatalf("item not refunded: casts %+v pack %+v", w.scrollCasts, w.carried[0])
		}
		if w.entities[0].CastWait == 0 {
			t.Fatal("the refund ended the recovery in progress")
		}
	})
}

// A third scroll replaces the waiting one, returns its item and survives a
// native save and load.
func TestScrollQueueThirdReplacesWaitingAndSaves(t *testing.T) {
	scrollQueueGames(t, func(t *testing.T, second bool) {
		w := scrollQueueWorld(t, second, 3)
		a, b := CellPoint{X: 6, Y: 5}, CellPoint{X: 8, Y: 7}
		w.entities[0].CastWait = 200
		Step(w, []Command{UseScrollAt(1, 0, a)})
		Step(w, []Command{UseScrollAt(1, 0, b)})
		if scrollQueuedCount(w) != 1 || len(w.scrollCasts) != 1 || w.scrollCasts[0].X != b.X || w.scrollCasts[0].Y != b.Y || teleportStack(w) != 2 {
			t.Fatalf("waiting scroll not replaced: casts %+v pack %+v", w.scrollCasts, w.carried[0])
		}
		if second {
			// The synthetic second-game state has no native form to save.
			return
		}
		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var loaded World
		if err := loaded.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(loaded.ScrollCasts(), w.ScrollCasts()) || loaded.Hash() != w.Hash() {
			t.Fatalf("queued scroll changed through the save: %+v", loaded.ScrollCasts())
		}
	})
}
