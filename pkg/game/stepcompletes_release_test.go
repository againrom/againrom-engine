package game

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/sim"
)

// stepFigure is where the figure is drawn, in cells: the cell it entered, less
// the part of the step still owed.
func stepFigure(e sim.Entity) (float64, float64) {
	x, y := float64(e.X), float64(e.Y)
	if e.Transit > 0 && e.Stride.Present && e.TransitTotal > 0 {
		f := float64(e.Transit) / float64(e.TransitTotal)
		x -= float64(e.X-e.Stride.FromX) * f
		y -= float64(e.Y-e.Stride.FromY) * f
	}
	return x, y
}

// stepWalking walks the fixture's mage toward the foe until a step with
// more than half its ticks still owed, and returns the mage.
func stepWalking(t *testing.T, f *FrontEnd, mage sim.Entity) sim.Entity {
	t.Helper()
	f.live.enqueue(uint32(mage.ID), 16, 40)
	for range 80 {
		f.live.tick()
		e := releaseEntity(t, f.live, mage.ID)
		if e.Transit > 0 && e.Transit*2 > e.TransitTotal && e.Transit < e.TransitTotal-1 {
			return e
		}
	}
	t.Fatal("the mage never reached the middle of a step")
	return sim.Entity{}
}

type stepTick struct {
	X, Y, Transit, Total int32
	Mana, TargetHP       int32
	Pending              uint8
	Book                 bool
}

func stepTrace(t *testing.T, f *FrontEnd, mage, foe sim.EntityID, ticks int) []stepTick {
	t.Helper()
	var out []stepTick
	for range ticks {
		f.live.tick()
		e := releaseEntity(t, f.live, mage)
		_, book := runningBook(f, mage)
		out = append(out, stepTick{X: e.X, Y: e.Y, Transit: int32(e.Transit), Total: int32(e.TransitTotal),
			Mana: e.Mana, TargetHP: releaseEntity(t, f.live, foe).HP, Pending: e.PendingOrder.Kind, Book: book})
	}
	return out
}

type stepWitness struct {
	Mage, Foe uint32
	Trace     []stepTick
}

// TestReleaseSpellClickMidStepCompletesTheStep walks the mage and clicks a
// spell onto the hostile unit in the middle of a step. The figure is drawn
// one tick of ordinary motion further each tick, the click waits as the
// mage's order, and the cast starts at the step's end.
func TestReleaseSpellClickMidStepCompletesTheStep(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f, mage, foe := spellClickFixture(t)
	e := stepWalking(t, f, mage)
	cell := [2]int32{e.X, e.Y}
	px, py := stepFigure(e)
	mana := e.Mana
	f.live.attackOrCast(uint32(mage.ID), uint32(foe.ID), 1, 0, 0, false)
	started := false
	for range 60 {
		f.live.tick()
		e = releaseEntity(t, f.live, mage.ID)
		x, y := stepFigure(e)
		if d := math.Hypot(x-px, y-py); d > 0.12 {
			t.Fatalf("the figure jumped %.3f of a cell: now (%d,%d) %d/%d", d, e.X, e.Y, e.Transit, e.TransitTotal)
		}
		px, py = x, y
		if _, book := runningBook(f, mage.ID); book {
			started = true
			if e.Transit != 0 || e.X != cell[0] || e.Y != cell[1] {
				t.Fatalf("the cast started at (%d,%d) transit %d, want the cell the step entered %v", e.X, e.Y, e.Transit, cell)
			}
			break
		}
		if e.Transit != 0 && e.Mana < mana {
			t.Fatal("mana spent before the step ended")
		}
	}
	if !started {
		t.Fatal("the waiting cast never started")
	}
}

// TestReleaseSpellClickMidStepSurvivesSAVE saves while the click waits for the
// step's end. A LOAD in this process and a LOAD in a fresh process draw the
// figure and run the cast as the uninterrupted game does.
func TestReleaseSpellClickMidStepSurvivesSAVE(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	const ticks = 90
	if path := os.Getenv("AGAINROM_STEP_ORDER_INPUT"); path != "" {
		var want stepWitness
		proof, err := os.ReadFile(path + ".json")
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(proof, &want); err != nil {
			t.Fatal(err)
		}
		stepContinuation(t, path, want)
		return
	}
	f, mage, foe := spellClickFixture(t)
	stepWalking(t, f, mage)
	f.live.attackOrCast(uint32(mage.ID), uint32(foe.ID), 1, 0, 0, false)
	f.live.tick()
	if e := releaseEntity(t, f.live, mage.ID); e.PendingOrder.Kind != sim.PendingActorCast || e.Transit == 0 {
		t.Fatalf("the click is not a waiting order: pending %+v transit %d", e.PendingOrder, e.Transit)
	}
	path := saveCorpseMission(t, f, t.TempDir())
	want := stepWitness{Mage: mage.SourceBinding.RuntimeID, Foe: foe.SourceBinding.RuntimeID, Trace: stepTrace(t, f, mage.ID, foe.ID, ticks)}
	cast := false
	for _, s := range want.Trace {
		cast = cast || s.Book
	}
	if !cast {
		t.Fatal("the uninterrupted game never cast")
	}
	proof, _ := json.MarshalIndent(want, "", "  ")
	if err := os.WriteFile(path+".json", proof, 0600); err != nil {
		t.Fatal(err)
	}
	emitSpellWitness(t, path, "stepordermidstep", proof)
	stepContinuation(t, path, want)
	runSpellWitnessChild(t, path, "AGAINROM_STEP_ORDER_INPUT")
}

func stepContinuation(t *testing.T, path string, want stepWitness) {
	t.Helper()
	f := loadAreaContinuation(t, path)
	mage := worldEntityByRuntimeID(t, f, want.Mage)
	foe := worldEntityByRuntimeID(t, f, want.Foe)
	if mage.PendingOrder.Kind != sim.PendingActorCast || mage.Transit == 0 {
		t.Fatalf("LOAD lost the waiting order: pending %+v transit %d", mage.PendingOrder, mage.Transit)
	}
	got := stepTrace(t, f, mage.ID, foe.ID, len(want.Trace))
	for i := range got {
		if got[i] != want.Trace[i] {
			t.Fatalf("tick %d after LOAD: %+v, uninterrupted %+v", i+1, got[i], want.Trace[i])
		}
	}
}
