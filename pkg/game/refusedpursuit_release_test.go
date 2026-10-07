package game

import (
	"testing"

	"againrom/pkg/sim"
)

func TestReleaseMission121BridgeTrollKeepsItsVictimWhenItsRouteIsRefused(t *testing.T) {
	const escortA, escortB sim.EntityID = 14, 15
	run := func(t *testing.T, pair bool) {
		f := releaseFront(t)
		p, _, err := StartScenarioMission(f.Archives, HeadlessScenario{Version: 6, Stage: StageMission, Mission: 121})
		if err != nil {
			t.Fatal(err)
		}
		troll, err := p.Resolve("u63")
		if err != nil {
			t.Fatal(err)
		}
		victim, err := p.Resolve("p0")
		if err != nil {
			t.Fatal(err)
		}
		if e, _ := p.entity(troll); e.TokenSize != 2 {
			t.Fatalf("script unit 63 holds a footprint of %d cells, want the 2x2 bridge troll", e.TokenSize)
		}
		home := map[sim.EntityID][2]int32{}
		for _, id := range []sim.EntityID{escortA, escortB} {
			e, _ := p.entity(id)
			home[id] = [2]int32{e.X, e.Y}
		}
		cells := map[sim.EntityID][2]int32{victim: {36, 15}}
		if pair {
			cells[escortA], cells[escortB] = [2]int32{35, 15}, [2]int32{35, 16}
		}
		for id, c := range cells {
			if err := p.World.HeadlessPlace(id, c[0], c[1]); err != nil {
				t.Fatal(err)
			}
		}
		pin := func() []sim.Command {
			if !pair {
				return nil
			}
			return []sim.Command{sim.MoveTo(escortA, sim.CellPoint{X: 35, Y: 15}), sim.MoveTo(escortB, sim.CellPoint{X: 35, Y: 16})}
		}
		const pinned, freed = 300, 300
		stood, still, idleTicks, firstIdle := 0, 0, 0, -1
		lastX, lastY := int32(-1), int32(-1)
		for k := 0; k < pinned; k++ {
			p.step(pin())
			e, _ := p.entity(troll)
			if !pair {
				continue
			}
			if e.X > 33 {
				t.Fatalf("tick %d: the troll's footprint reached column %d, past the pinned pair", k, e.X)
			}
			if !e.PursuitIdle {
				if e.X == lastX && e.Y == lastY && e.Transit == 0 {
					still++
				} else {
					still = 0
				}
				lastX, lastY = e.X, e.Y
				continue
			}
			if !e.HasAttackTarget || e.AttackTarget != victim || e.AcquirePursuit || e.HasTarget || e.Stall != 0 {
				t.Fatalf("tick %d: idle troll holds victim %v/%d, acquisition %v, walk %v, stall %d; want victim %d kept and no walk",
					k, e.HasAttackTarget, e.AttackTarget, e.AcquirePursuit, e.HasTarget, e.Stall, victim)
			}
			if firstIdle < 0 {
				firstIdle, stood = k, still
			}
			idleTicks++
		}
		if pair {
			t.Logf("first refused at tick %d after standing %d tick(s); idle on %d of the next %d ticks", firstIdle, stood, idleTicks, pinned-firstIdle)
			if firstIdle < 0 {
				t.Fatal("the troll's route was never refused")
			}
			if stood >= 8 {
				t.Fatalf("the troll stood %d ticks before it was refused, want the refusal at the search that came back empty", stood)
			}
			if idleTicks < (pinned-firstIdle)*9/10 {
				t.Fatalf("the order read idle on %d of %d ticks, want it idle throughout", idleTicks, pinned-firstIdle)
			}
			for id, c := range home {
				if err := p.World.HeadlessPlace(id, c[0], c[1]); err != nil {
					t.Fatal(err)
				}
			}
		}
		struck := false
		for k := 0; k < freed && !struck; k++ {
			p.step(nil)
			if v, _ := p.entity(victim); v.HP < 145 {
				struck = true
			}
		}
		if !struck {
			e, _ := p.entity(troll)
			t.Fatalf("the troll never struck its victim after the pair left: at %d,%d idle %v", e.X, e.Y, e.PursuitIdle)
		}
	}
	t.Run("pair between", func(t *testing.T) { run(t, true) })
	t.Run("control: no pair", func(t *testing.T) { run(t, false) })
}
