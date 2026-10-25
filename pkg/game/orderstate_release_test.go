package game

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/sim"
)

type orderTick struct {
	X, Y, Transit  int32
	Moving         bool
	CastWait       uint8
	Mana, TargetHP int32
	Book           bool
}

func orderTrace(t *testing.T, f *FrontEnd, mage, foe sim.EntityID, ticks int) []orderTick {
	t.Helper()
	var out []orderTick
	for range ticks {
		f.live.tick()
		e := releaseEntity(t, f.live, mage)
		_, book := runningBook(f, mage)
		out = append(out, orderTick{X: e.X, Y: e.Y, Transit: int32(e.Transit), Moving: e.HasTarget && !e.HasAttackTarget,
			CastWait: e.CastWait, Mana: e.Mana, TargetHP: releaseEntity(t, f.live, foe).HP, Book: book})
	}
	return out
}

// orderTraceFault reads the owner's rule off a trace: no cast record while a
// move order is in force or a step is under way, and the body stands from the
// wind-up to the end of the recovery. It also reports whether a cast ran.
func orderTraceFault(trace []orderTick) (string, bool) {
	cast := false
	held, at := false, [2]int32{}
	for i, s := range trace {
		if s.Book && (s.Moving || s.Transit != 0) {
			return "a cast runs on the move at tick " + strconv.Itoa(i), cast
		}
		busy := s.Book || s.CastWait != 0
		cast = cast || s.Book
		switch {
		case busy && !held:
			held, at = true, [2]int32{s.X, s.Y}
		case busy && (s.X != at[0] || s.Y != at[1] || s.Transit != 0):
			return "the caster moved during its cast at tick " + strconv.Itoa(i), cast
		case !busy:
			held = false
		}
	}
	return "", cast
}

// TestReleaseArmedMageCastsOnlyAfterItsWalk is the owner's case through the
// mission route: a party mage on automatic casting is ordered to walk with a
// hostile in range the whole way. It casts nothing until it stands on the
// ordered cell, then casts standing.
func TestReleaseArmedMageCastsOnlyAfterItsWalk(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f, mage, foe := spellClickFixture(t)
	f.live.setAutocast(uint32(mage.ID), 1)
	f.live.enqueue(uint32(mage.ID), 16, 40)
	trace := orderTrace(t, f, mage.ID, foe.ID, 160)
	fault, cast := orderTraceFault(trace)
	if fault != "" {
		t.Fatal(fault)
	}
	if !cast {
		t.Fatal("the armed mage never cast after its walk")
	}
	walked := false
	for _, s := range trace {
		walked = walked || s.Transit != 0
		if s.Book && !walked {
			t.Fatal("the mage cast before it walked")
		}
	}
	if !walked {
		t.Fatal("fixture: the mage never walked")
	}
}

type orderWitness struct {
	Mage, Foe uint32
	Trace     []orderTick
}

// TestReleaseOrderStateSurvivesSAVE saves twice: once in a cast's recovery
// with a move order waiting behind it, once mid-step with the automatic cast
// waiting for the walk to end. A LOAD in this process and in a fresh process
// continue as the uninterrupted game does.
func TestReleaseOrderStateSurvivesSAVE(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	for _, c := range []struct {
		name, witness string
		setup         func(t *testing.T, f *FrontEnd, mage, foe sim.Entity)
	}{
		{"recovery", "orderstaterecovery", func(t *testing.T, f *FrontEnd, mage, foe sim.Entity) {
			f.live.attackOrCast(uint32(mage.ID), uint32(foe.ID), 1, 0, 0, false)
			for k := 0; k < 80 && releaseEntity(t, f.live, mage.ID).CastWait == 0; k++ {
				f.live.tick()
			}
			if releaseEntity(t, f.live, mage.ID).CastWait < 3 {
				t.Fatal("fixture: the cast never reached its recovery")
			}
			f.live.enqueue(uint32(mage.ID), 16, 40)
			f.live.tick()
		}},
		{"automatic cast waits for the step", "orderstateautostep", func(t *testing.T, f *FrontEnd, mage, foe sim.Entity) {
			f.live.setAutocast(uint32(mage.ID), 1)
			f.live.enqueue(uint32(mage.ID), 16, 40)
			for k := 0; k < 40; k++ {
				f.live.tick()
				if e := releaseEntity(t, f.live, mage.ID); e.Transit > 1 {
					return
				}
			}
			t.Fatal("fixture: the mage never stepped")
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			const ticks = 120
			if path := os.Getenv("AGAINROM_ORDER_STATE_INPUT"); path != "" {
				var want orderWitness
				proof, err := os.ReadFile(path + ".json")
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(proof, &want); err != nil {
					t.Fatal(err)
				}
				orderContinuation(t, path, want)
				return
			}
			f, mage, foe := spellClickFixture(t)
			c.setup(t, f, mage, foe)
			path := saveCorpseMission(t, f, t.TempDir())
			want := orderWitness{Mage: mage.SourceBinding.RuntimeID, Foe: foe.SourceBinding.RuntimeID, Trace: orderTrace(t, f, mage.ID, foe.ID, ticks)}
			first, last := want.Trace[0], want.Trace[len(want.Trace)-1]
			fault, cast := orderTraceFault(want.Trace)
			if fault != "" || !cast && first.CastWait == 0 {
				t.Fatalf("the uninterrupted game: fault %q, cast %v, recovery %d", fault, cast, first.CastWait)
			}
			if first.X == last.X && first.Y == last.Y {
				t.Fatal("the uninterrupted game never walked")
			}
			proof, _ := json.MarshalIndent(want, "", "  ")
			if err := os.WriteFile(path+".json", proof, 0600); err != nil {
				t.Fatal(err)
			}
			emitSpellWitness(t, path, c.witness, proof)
			orderContinuation(t, path, want)
			runSpellWitnessChild(t, path, "AGAINROM_ORDER_STATE_INPUT")
		})
	}
}

func orderContinuation(t *testing.T, path string, want orderWitness) {
	t.Helper()
	f := loadAreaContinuation(t, path)
	mage := worldEntityByRuntimeID(t, f, want.Mage)
	foe := worldEntityByRuntimeID(t, f, want.Foe)
	got := orderTrace(t, f, mage.ID, foe.ID, len(want.Trace))
	for i := range got {
		if got[i] != want.Trace[i] {
			t.Fatalf("tick %d after LOAD: %+v, uninterrupted %+v", i+1, got[i], want.Trace[i])
		}
	}
}
