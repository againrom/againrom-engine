package sim

import "testing"

type teleportStart struct {
	picture, x, y int32
	casterX       int32
	found         bool
}

func teleportStartOf(t *testing.T, warrior, secondGame bool) teleportStart {
	t.Helper()
	caster := spEnt(1, 1, 1)
	caster.AttackCharge, caster.AttackRelax = 4, 2
	if !warrior {
		caster = spMage(1, 1, 1, 30, 100, 100, 1<<26)
		caster.AttackCharge, caster.AttackRelax = 4, 2
	}
	w := spWorld(t, 1090, []SpellRule{{ID: 26, MaxRange: 8}}, caster)
	in := producerInputs(2)
	in.Profile.StartFromMovedCaster = secondGame
	in.Pictures[60] = FlightPicture{Present: true, Phases: 4, SecondRecord: true, LifeTicks: 60}
	w.SetFlightInputs(in)
	cmd := CastAt(1, 26, CellPoint{X: 3, Y: 3})
	if warrior {
		item := ItemInstance{Code: 0xe10, Kind: 4, Price: 50, Effects: []ItemEffect{{Kind: 41, Operand: 26 | 60<<16}}}
		w.carried[0] = []ItemStack{StackItem(item, 1)}
		cmd = UseScrollAt(1, 0, CellPoint{X: 3, Y: 3})
	}
	var got teleportStart
	for tick := 0; tick < 60 && !got.found; tick++ {
		var cmds []Command
		if tick == 0 {
			cmds = []Command{cmd}
		}
		if len(StepReported(w, cmds).Casts) == 0 {
			continue
		}
		e, _ := w.Entity(1)
		got.casterX, got.found = e.X, true
		for _, p := range w.SavedProjectiles().Items {
			got.picture, got.x, got.y = p.Picture, p.X, p.Y
			break
		}
	}
	return got
}

func TestAWarriorsTeleportStartsWhereAMagesDoes(t *testing.T) {
	for _, secondGame := range []bool{false, true} {
		mage := teleportStartOf(t, false, secondGame)
		warrior := teleportStartOf(t, true, secondGame)
		if !mage.found || mage.picture != 60 || mage.casterX != 1 {
			t.Fatalf("second game %v: mage start %+v", secondGame, mage)
		}
		if warrior != mage {
			t.Errorf("second game %v: warrior start %+v, mage start %+v", secondGame, warrior, mage)
		}
	}
}
