package game

import (
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

const (
	releaseUnreachableMission              = 10
	releaseUnreachableFoe     sim.EntityID = 14
)

type unreachableArena struct {
	front  *FrontEnd
	live   *mapWorld
	hero   sim.EntityID
	beside sim.EntityID
}

func openUnreachableArena(t *testing.T, placeBeside bool, offset int32, heroAt ...int32) *unreachableArena {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
	app := f.App("unreachable creature")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(releaseUnreachableMission, party)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	for k := 0; k < 16 && live.mission.open; k++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	bindings, err := live.mission.state.Map.CellBindings()
	if err != nil {
		t.Fatal(err)
	}
	tails := make([]sim.CellTail, len(bindings))
	for i, binding := range bindings {
		tails[i] = sim.CellTail{X: int32(binding.X), Y: int32(binding.Y), Bytes: [6]byte{binding.Spell, binding.Power, binding.SourceX, binding.SourceY, binding.LastX, binding.LastY}}
	}
	if err := live.world.DeclareCellTails(tails); err != nil {
		t.Fatal(err)
	}
	heroID := live.mission.ids[0]
	if len(heroAt) == 2 {
		if err := live.world.HeadlessPlace(heroID, heroAt[0], heroAt[1]); err != nil {
			t.Fatal(err)
		}
	}
	hero, ok := live.entity(heroID)
	if !ok {
		t.Fatal("no party hero")
	}
	rel := live.world.Relations()
	victim, ok := live.entity(releaseUnreachableFoe)
	if !ok || !victim.Alive() || victim.OffMap || victim.TokenSize > 1 || !rel.Hostile(victim.Owner, hero.Owner) {
		t.Fatalf("mission %d creature %d is not a hostile one-cell creature", releaseUnreachableMission, releaseUnreachableFoe)
	}
	var beside sim.EntityID
	weakest := int32(-1)
	for _, e := range live.world.Entities() {
		if e.ID != releaseUnreachableFoe && e.Alive() && !e.OffMap && e.TokenSize == 1 && e.DamageBase > 0 &&
			rel.Hostile(e.Owner, hero.Owner) && (weakest < 0 || e.MaxHP < weakest) {
			beside, weakest = e.ID, e.MaxHP
		}
	}
	if weakest < 0 {
		t.Fatal("mission has no other fighting hostile")
	}
	if placeBeside {
		placed := false
		for k := 0; k < 400 && !placed; k++ {
			if err := live.world.HeadlessPlace(beside, hero.X+offset, hero.Y); err != nil {
				continue
			}
			e, _ := live.entity(beside)
			d := max(releaseAbs32(e.X-hero.X), releaseAbs32(e.Y-hero.Y))
			placed = (offset <= 1 && d == 1) || (offset > 1 && d >= 5)
		}
		if !placed {
			t.Fatalf("no cell about %d from the hero admits the hostile", offset)
		}
	}
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	live.push()
	inspectionCentre(live, int(hero.X), int(hero.Y))
	if err := app.HeadlessSelectEntity(uint32(heroID)); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("attack"); err != nil {
		t.Fatal(err)
	}
	inspectionCentre(live, int(victim.X), int(victim.Y))
	x, y, err := app.HeadlessEntityPoint(uint32(releaseUnreachableFoe))
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	if len(live.pending) != 1 || live.pending[0].Kind != sim.KindAttack || live.pending[0].Entity != heroID ||
		live.pending[0].X != int32(releaseUnreachableFoe) {
		t.Fatalf("the click queued %+v, want the hero's attack on creature %d", live.pending, releaseUnreachableFoe)
	}
	return &unreachableArena{front: f, live: live, hero: heroID, beside: beside}
}

func (a *unreachableArena) get(id sim.EntityID) sim.Entity {
	e, _ := a.live.entity(id)
	return e
}

func (a *unreachableArena) reloadCold(t *testing.T) *FrontEnd {
	t.Helper()
	snapshot, label, err := a.front.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	hash := a.live.world.Hash()
	raw, err := a.front.ExportCurrentSave(snapshot, label)
	if err != nil || a.live.world.Hash() != hash {
		t.Fatal("SAVE failed or changed the World", err)
	}
	back := releaseFront(t)
	back.Options = OptionsStore{}
	back.SetDeterministicFrames(true)
	opener, town, err := back.RestoreOriginal(raw)
	if err != nil || town || opener == nil {
		t.Fatal("LOAD", err, town)
	}
	app := back.App("unreachable creature LOAD")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	return back
}

func TestReleaseAnOrderOntoAnUnreachableCreatureTakesTheHostileBesideTheHero(t *testing.T) {
	t.Run("hostile beside the hero", func(t *testing.T) {
		// From the hero's mission start, 29 cells from the creature, the first
		// full search settles on a cell ten from it and the hero walks off
		// before refusing (MOVE-099). From 27 cells the first full search
		// finds no cell, so the refusal comes while the hostile is beside him.
		a := openUnreachableArena(t, true, 1, 19, 64)
		start := a.get(a.beside).HP
		var picked, struck int
		var cold *FrontEnd
		for n := 0; n < 240; n++ {
			a.live.tick()
			h := a.get(a.hero)
			justLoaded := false
			if picked == 0 && h.AcquirePursuit && h.HasAttackTarget && h.AttackTarget == a.beside && !h.PursuitIdle {
				picked = n + 1
				if h.AttackTarget == releaseUnreachableFoe {
					t.Fatal("the hero kept the unreachable creature")
				}
				cold = a.reloadCold(t)
				assertCurrentWorldEqual(t, a.live.world, cold.live.world, "LOAD of the acquired pick")
				justLoaded = true
			}
			if cold != nil && !justLoaded {
				cold.live.tick()
				assertEntitiesEqual(t, a.live.world, cold.live.world, []sim.EntityID{a.hero, releaseUnreachableFoe, a.beside}, "successor of the acquired pick")
			}
			if b := a.get(a.beside); b.HP < start && struck == 0 {
				struck = n + 1
			}
		}
		if picked == 0 {
			h := a.get(a.hero)
			t.Fatalf("the hero never took the hostile beside it: victim %v/%d acquisition %v idle %v", h.HasAttackTarget, h.AttackTarget, h.AcquirePursuit, h.PursuitIdle)
		}
		if struck == 0 {
			t.Fatalf("the hostile beside the hero kept %d health: the hero never struck its pick", start)
		}
		t.Logf("pick at tick %d, first blow on it at tick %d", picked, struck)
	})
	t.Run("no hostile beside the hero", func(t *testing.T) {
		{
			a := openUnreachableArena(t, false, 0)
			var cold *FrontEnd
			for n := 0; n < 120; n++ {
				a.live.tick()
				h := a.get(a.hero)
				if h.AcquirePursuit {
					t.Fatalf("tick %d: the hero took a victim with none in reach", n)
				}
				justLoaded := false
				if cold == nil && h.PursuitIdle && h.HasAttackTarget && h.AttackTarget == releaseUnreachableFoe {
					cold = a.reloadCold(t)
					assertCurrentWorldEqual(t, a.live.world, cold.live.world, "LOAD of the idle order")
					if back, ok := cold.live.entity(a.hero); !ok || !back.PursuitIdle || back.AttackTarget != releaseUnreachableFoe {
						t.Fatalf("the loaded hero lost the idle order on the unreachable creature: %+v", back)
					}
					justLoaded = true
				}
				if cold != nil && !justLoaded {
					cold.live.tick()
					assertEntitiesEqual(t, a.live.world, cold.live.world, []sim.EntityID{a.hero, releaseUnreachableFoe, a.beside}, "successor of the idle order")
				}
			}
			h := a.get(a.hero)
			if !h.PursuitIdle || !h.HasAttackTarget || h.AttackTarget != releaseUnreachableFoe || h.AcquirePursuit {
				t.Fatalf("hero: idle %v acquisition %v victim %v/%d; want the idle order on creature %d", h.PursuitIdle, h.AcquirePursuit, h.HasAttackTarget, h.AttackTarget, releaseUnreachableFoe)
			}
			if cold == nil {
				t.Fatal("the hero never idled, so no idle order was saved")
			}
		}
	})
}

func assertEntitiesEqual(t *testing.T, want, got *sim.World, ids []sim.EntityID, cut string) {
	t.Helper()
	find := func(w *sim.World, id sim.EntityID) (sim.Entity, bool) {
		for _, e := range w.Entities() {
			if e.ID == id {
				return e, true
			}
		}
		return sim.Entity{}, false
	}
	for _, id := range ids {
		a, aok := find(want, id)
		b, bok := find(got, id)
		if aok != bok {
			t.Fatalf("%s: actor %d presence %v / %v", cut, id, aok, bok)
		}
		x, y := reflect.ValueOf(a), reflect.ValueOf(b)
		for i := 0; i < x.NumField(); i++ {
			if x.Field(i).CanInterface() && !reflect.DeepEqual(x.Field(i).Interface(), y.Field(i).Interface()) {
				t.Fatalf("%s actor %d %s: %v / %v", cut, id, x.Type().Field(i).Name, x.Field(i).Interface(), y.Field(i).Interface())
			}
		}
	}
}
