package sim

import (
	"fmt"
	"testing"
)

// osCastWatch follows one caster tick by tick: no cast record charges while a
// move order is in force or a step is under way, the body stays on one cell
// from the wind-up to the end of the recovery, and every release faces its
// target.
type osCastWatch struct {
	id       EntityID
	held     bool
	heldAt   [2]int32
	charged  int
	released int
}

func (c *osCastWatch) observe(t *testing.T, w *World, tick int, events []CastEvent) {
	t.Helper()
	i := indexOfEntity(w.entities, c.id)
	e := w.entities[i]
	s := w.orderStateAt(i)
	if s.charging() {
		c.charged++
		if e.HasTarget && !e.HasAttackTarget {
			t.Fatalf("tick %d: a cast charges while a move order is in force (target %d,%d)", tick, e.TargetX, e.TargetY)
		}
		if e.Transit != 0 {
			t.Fatalf("tick %d: a cast charges in a step (transit %d)", tick, e.Transit)
		}
	}
	for _, ev := range events {
		if ev.Caster != c.id {
			continue
		}
		c.released++
		if ev.AtCell || ev.Target == c.id {
			continue
		}
		ti := indexOfEntity(w.entities, ev.Target)
		if ti < 0 {
			continue
		}
		if want := w.headingBetween(e, w.entities[ti]); e.Facing != want || e.Turning() {
			t.Fatalf("tick %d: the release faces %d, its target lies at %d (turning %v)", tick, e.Facing, want, e.Turning())
		}
	}
	busy := s.charging() || s.wait || s.book == castRecovering
	switch {
	case busy && !c.held:
		c.held, c.heldAt = true, [2]int32{e.X, e.Y}
	case busy && (e.X != c.heldAt[0] || e.Y != c.heldAt[1]):
		t.Fatalf("tick %d: the caster moved from %v to %d,%d between its wind-up and the end of its recovery", tick, c.heldAt, e.X, e.Y)
	case !busy:
		c.held = false
	}
}

// A mage with an armed automatic attack spell, ordered to walk past enemies in
// range, casts nothing on the way. After it arrives it casts standing.
func TestAnArmedMageCastsOnlyAfterItsWalkEnds(t *testing.T) {
	w := osWorld(t, [2]int32{10, 6}, [2]int32{16, 6})
	w.entities[0].AutoSpell = 1
	to := CellPoint{X: 20, Y: 4}
	watch := &osCastWatch{id: osMage}
	cmds := []Command{MoveTo(osMage, to)}
	arrivedAt := -1
	for tick := range 500 {
		events := StepObserved(w, cmds)
		cmds = nil
		watch.observe(t, w, tick, events)
		e := w.entities[0]
		if arrivedAt < 0 && e.X == to.X && e.Y == to.Y && e.Transit == 0 {
			arrivedAt = tick
			if watch.released != 0 {
				t.Fatalf("the mage released %d casts on its walk", watch.released)
			}
		}
	}
	if arrivedAt < 0 {
		e := w.entities[0]
		t.Fatalf("the mage never reached %v; it stands at %d,%d", to, e.X, e.Y)
	}
	if watch.released == 0 {
		t.Fatal("the mage never cast after it arrived")
	}
}

// The automatic cast issues the cast a player's order issues at that moment:
// a standing mage casting automatically and the same mage given the cast by
// hand turn, wind up, release and recover alike, tick for tick.
func TestTheAutomaticCastIsThePlayersCastAtTheSameMoment(t *testing.T) {
	auto, hand := osWorld(t, [2]int32{8, 6}), osWorld(t, [2]int32{8, 6})
	auto.entities[0].AutoSpell = 1
	type frame struct {
		casts             []bookCast
		x, y              int32
		facing            uint8
		mana, enemyHP     int32
		castWait          uint8
		released, turning bool
	}
	at := func(w *World, events []CastEvent) frame {
		e := w.entities[0]
		return frame{casts: append([]bookCast(nil), w.bookCasts...), x: e.X, y: e.Y, facing: e.Facing,
			mana: e.Mana, enemyHP: w.entities[1].HP, castWait: e.CastWait, released: len(events) != 0, turning: e.Turning()}
	}
	cmds := []Command{Cast(osMage, osEnemy, 1)}
	recovered := false
	for tick := 0; tick < 80 && !recovered; tick++ {
		a := at(auto, StepObserved(auto, nil))
		h := at(hand, StepObserved(hand, cmds))
		cmds = nil
		if fmt.Sprint(a) != fmt.Sprint(h) {
			t.Fatalf("tick %d: automatic %+v, by hand %+v", tick, a, h)
		}
		recovered = tick > 0 && a.castWait == 0 && len(a.casts) == 0 && a.mana < 5000
	}
	if !recovered {
		t.Fatal("fixture: the cast never ran to the end of its recovery")
	}
}
