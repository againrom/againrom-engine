package game

import (
	"testing"

	"againrom/pkg/sim"
)

// The far search of a unit no human participant owns spends max(5, D>>2) + D
// generations, D being the Chebyshev distance from its cell to the cell it was
// ordered to, and settles on a nearer cell when its wave has not labelled the
// goal by then; a one-cell unit of the party keeps the flat thousand
// (MOVE-TERM-003).
//
// A wave of B generations labels only cells within B steps. The route read back
// from its labels can be a few cells longer than B when a cheaper way round a
// costly cell takes more steps, so every bound below allows releaseFarSlack
// cells more. Over the commanded drives of every shipped campaign mission, with
// the hero sent onto each of its twelve nearest hostile creatures in turn, the
// excess of a chase route that ends on its victim was at most 2 cells, and the
// flat thousand exceeded B by up to 30.
const (
	releaseFarWalkMission              = 31
	releaseFarWalkFoe     sim.EntityID = 2
	releaseFarWalkHorizon              = 420
	releaseFarFleeMission              = 40
	releaseFarFleeFoe     sim.EntityID = 36
	releaseFarFleeHorizon              = 780
	releaseFarSlack                    = 4
	releaseFarFleeStep                 = 3
)

// releaseFarBudget is what a unit no participant owns may spend to reach a cell
// D cells away.
func releaseFarBudget(d int32) int32 { return max(5, d>>2) + d }

// releaseFarDistance is the Chebyshev distance between two cells.
func releaseFarDistance(a, b [2]int32) int32 {
	return max(releaseAbs32(a[0]-b[0]), releaseAbs32(a[1]-b[1]))
}

// releaseStoredRoute is a route an entity holds after a tick that it did not hold
// before: none then, a different last cell, or a longer one.
type releaseStoredRoute struct {
	entity sim.Entity
	from   [2]int32
	route  [][2]int32
}

type releaseRouteState struct {
	cell [2]int32
	n    int
	last [2]int32
}

type releaseRouteWatch struct {
	prev map[sim.EntityID]releaseRouteState
}

// newReleaseRouteWatch starts a watch that remembers where every entity of the
// mission stands and what route it holds now.
func newReleaseRouteWatch(live *mapWorld) *releaseRouteWatch {
	w := &releaseRouteWatch{prev: map[sim.EntityID]releaseRouteState{}}
	w.stored(live)
	return w
}

// stored reports, in entity order, every route that is new since the last call,
// with the cell its owner stood on before the tick that stored it.
func (w *releaseRouteWatch) stored(live *mapWorld) []releaseStoredRoute {
	var out []releaseStoredRoute
	for _, e := range live.world.Entities() {
		rt := live.world.Route(e.ID)
		p := w.prev[e.ID]
		cur := releaseRouteState{cell: [2]int32{e.X, e.Y}, n: len(rt)}
		if len(rt) > 0 {
			cur.last = rt[len(rt)-1]
		}
		w.prev[e.ID] = cur
		if len(rt) == 0 || !(p.n == 0 || cur.last != p.last || len(rt) > p.n) {
			continue
		}
		out = append(out, releaseStoredRoute{entity: e, from: p.cell, route: rt})
	}
	return out
}

// releaseFarOrder opens the mission with the party and orders its first hero onto
// creature foe through the App's own input: the hero is selected, the attack key
// pressed and the creature clicked.
func releaseFarOrder(t *testing.T, f *FrontEnd, mission int, foe sim.EntityID) (*mapWorld, sim.EntityID) {
	t.Helper()
	party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
	app := f.App("far search")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(mission, party)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	for k := 0; k < 16 && live.mission.open; k++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	heroID := live.mission.ids[0]
	hero, ok := live.entity(heroID)
	if !ok {
		t.Fatal("no party hero")
	}
	target, ok := live.entity(foe)
	if !ok || !target.Alive() || target.OffMap || target.TokenSize > 1 ||
		target.Owner == 0 || target.Owner == sim.SelfSlot || !live.world.Relations().Hostile(target.Owner, hero.Owner) {
		t.Fatalf("mission %d creature %d is not a hostile one-cell creature of another owner", mission, foe)
	}
	inspectionCentre(live, int(hero.X), int(hero.Y))
	if err := app.HeadlessSelectEntity(uint32(heroID)); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("attack"); err != nil {
		t.Fatal(err)
	}
	inspectionCentre(live, int(target.X), int(target.Y))
	x, y, err := app.HeadlessEntityPoint(uint32(foe))
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	if len(live.pending) != 1 || live.pending[0].Kind != sim.KindAttack || live.pending[0].Entity != heroID ||
		live.pending[0].X != int32(foe) {
		t.Fatalf("the click queued %+v, want the hero's attack on creature %d", live.pending, foe)
	}
	return live, heroID
}

// TestReleaseAnAICreaturesFarSearchSettlesInsteadOfWalkingTheDetour runs the two
// shapes a far search takes in ordinary play on the shipped maps.
//
// The walk is mission 31. The hero's own route to a creature across an obstacle is
// longer than the budget of another owner's unit, so the party keeps the flat
// thousand. The creatures that acquire the hero from the far side of that obstacle
// must store no chase route longer than their own budget, and the hero's target
// must store one that stops short of the hero's cell. Before, they spent the flat
// thousand and walked the detour onto the hero's cell.
//
// The flee is mission 40. Ranged creatures back away when the hero stands beside
// them, to a cell three cells off, and some of those cells lie behind an obstacle
// that takes dozens of steps to walk round. A creature that backs away must store
// no route longer than the budget of a cell three cells off. Before, one stored a
// route of 47 cells to such a cell and ran the long way round past the hero.
func TestReleaseAnAICreaturesFarSearchSettlesInsteadOfWalkingTheDetour(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)

	t.Run("walk", func(t *testing.T) {
		live, heroID := releaseFarOrder(t, f, releaseFarWalkMission, releaseFarWalkFoe)
		foe, _ := live.entity(releaseFarWalkFoe)
		watch := newReleaseRouteWatch(live)
		var control int
		var settled bool
		for n := 0; n < releaseFarWalkHorizon; n++ {
			before, _ := live.entity(heroID)
			live.tick()
			hero, _ := live.entity(heroID)
			// The hero may step in the tick that stores a route, so a route that
			// reaches the hero's cell ends on the cell it stood on before or after.
			goal, was := [2]int32{hero.X, hero.Y}, [2]int32{before.X, before.Y}
			for _, s := range watch.stored(live) {
				e := s.entity
				end := s.route[len(s.route)-1]
				switch {
				case e.ID == heroID && control == 0:
					control = len(s.route)
					d := releaseFarDistance(s.from, [2]int32{foe.X, foe.Y})
					t.Logf("hero %d: %d-cell route to creature %d, %d cells away, a budget of %d for another owner's unit",
						heroID, control, releaseFarWalkFoe, d, releaseFarBudget(d))
					if control <= int(releaseFarBudget(d))+releaseFarSlack {
						t.Fatalf("the hero's own route is %d cells, not longer than another owner's unit could store: the drive does not discriminate", control)
					}
				case e.Owner != 0 && e.Owner != sim.SelfSlot && e.TokenSize <= 1 && e.HasAttackTarget && e.AttackTarget == heroID:
					d := releaseFarDistance(s.from, goal)
					switch {
					case (end == goal || end == was) && len(s.route) > int(releaseFarBudget(d))+releaseFarSlack:
						t.Errorf("tick %d: creature %d (owner %d) stored a %d-cell route onto the hero's cell (%d,%d), %d cells away: its far search spent more than the %d generations of a unit no participant owns",
							live.world.Tick(), e.ID, e.Owner, len(s.route), goal[0], goal[1], d, releaseFarBudget(d))
					case e.ID == releaseFarWalkFoe && !settled && releaseFarDistance(end, goal) >= 2 && releaseFarDistance(end, was) >= 2:
						settled = true
						t.Logf("tick %d: creature %d at (%d,%d) stores a %d-cell route ending at (%d,%d), short of the hero at (%d,%d), %d cells away",
							live.world.Tick(), e.ID, s.from[0], s.from[1], len(s.route), end[0], end[1], goal[0], goal[1], d)
					}
				}
			}
		}
		if !settled {
			t.Fatalf("creature %d stored no route that stops short of the hero within %d ticks: the drive does not reach the state under test",
				releaseFarWalkFoe, releaseFarWalkHorizon)
		}
	})

	t.Run("flee", func(t *testing.T) {
		live, heroID := releaseFarOrder(t, f, releaseFarFleeMission, releaseFarFleeFoe)
		watch := newReleaseRouteWatch(live)
		limit := int(releaseFarBudget(releaseFarFleeStep)) + releaseFarSlack
		var fled int
		for n := 0; n < releaseFarFleeHorizon; n++ {
			live.tick()
			hero, ok := live.entity(heroID)
			if !ok || !hero.Alive() {
				break
			}
			for _, s := range watch.stored(live) {
				e := s.entity
				if e.Owner == 0 || e.Owner == sim.SelfSlot || e.TokenSize > 1 || e.Withdraw <= 0 || e.Reach <= 1 || e.HasAttackTarget ||
					releaseFarDistance([2]int32{e.X, e.Y}, [2]int32{hero.X, hero.Y}) > releaseFarFleeStep {
					continue
				}
				fled++
				end := s.route[len(s.route)-1]
				t.Logf("tick %d: creature %d at (%d,%d) beside the hero at (%d,%d) stores a %d-cell route ending at (%d,%d)",
					live.world.Tick(), e.ID, s.from[0], s.from[1], hero.X, hero.Y, len(s.route), end[0], end[1])
				if len(s.route) > limit {
					t.Errorf("creature %d stored a %d-cell route while backing away from the hero, to a cell three cells from it: its far search spent more than the %d generations of a unit no participant owns",
						e.ID, len(s.route), releaseFarBudget(releaseFarFleeStep))
				}
			}
		}
		if fled == 0 {
			t.Fatalf("no ranged creature beside the hero stored a route within %d ticks: the drive does not reach the state under test", releaseFarFleeHorizon)
		}
	})
}

// The far search of a creature larger than one cell that no participant owns
// spends StaticScanAhead + D generations, D being the Chebyshev distance from its
// cell to its goal (MOVE-TERM-003, MOVE-PARAM-006). The one-cell form spends
// max(5, D>>2) + D. The two agree while D is under releaseLargeParted and part
// from there, so only a chase over that distance tells the arms apart.
//
// A wave of B generations labels only cells within B steps, and the route read
// back from its labels can be a cell or two longer when a cheaper way round a
// costly cell takes more steps. The bound allows releaseLargeSlack cells over B.
const (
	releaseLargeMission              = 120
	releaseLargeFoe     sim.EntityID = 54
	releaseLargeHorizon              = 1440
	releaseLargeParted               = 24
	releaseLargeSlack                = 1
)

// releaseLargeBudget is what a creature larger than one cell that no participant
// owns may spend to reach a cell D cells away.
func releaseLargeBudget(d int32) int32 { return 5 + d }

// TestReleaseALargeAICreaturesFarSearchSpendsTheNxNArm drives mission 120. The
// hero is ordered onto a creature across the map with the attack key and a click,
// and creature 30 of owner 4, a size-two Ogre, chases him from 25 to 29 cells away
// over ground where the way round is a few cells longer than the distance. Every
// route a creature larger than one cell stores toward the hero from
// releaseLargeParted cells or more must be no longer than its budget plus
// releaseLargeSlack. A pursuit's full search settles on a free cell beside the
// victim (MOVE-099), so the drive's routes end beside the hero and the bound
// applies to them as well as to any route onto his cell.
func TestReleaseALargeAICreaturesFarSearchSpendsTheNxNArm(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	live, heroID := releaseFarOrder(t, f, releaseLargeMission, releaseLargeFoe)
	watch := newReleaseRouteWatch(live)
	var onHero, beside int
	for n := 0; n < releaseLargeHorizon; n++ {
		before, _ := live.entity(heroID)
		live.tick()
		hero, ok := live.entity(heroID)
		if !ok || !hero.Alive() {
			break
		}
		goal, was := [2]int32{hero.X, hero.Y}, [2]int32{before.X, before.Y}
		for _, s := range watch.stored(live) {
			e := s.entity
			if e.Owner == 0 || e.Owner == sim.SelfSlot || e.TokenSize <= 1 || !e.HasAttackTarget || e.AttackTarget != heroID {
				continue
			}
			d := releaseFarDistance(s.from, goal)
			if d < releaseLargeParted {
				continue
			}
			end := s.route[len(s.route)-1]
			t.Logf("tick %d: creature %d (size %d) at (%d,%d) stores a %d-cell route ending at (%d,%d), the hero at (%d,%d), %d cells away, a budget of %d",
				live.world.Tick(), e.ID, e.TokenSize, s.from[0], s.from[1], len(s.route), end[0], end[1], goal[0], goal[1], d, releaseLargeBudget(d))
			if end != goal && end != was {
				beside++
			} else {
				onHero++
			}
			if len(s.route) > int(releaseLargeBudget(d))+releaseLargeSlack {
				t.Errorf("tick %d: creature %d (owner %d, size %d) stored a %d-cell route toward the hero's cell (%d,%d), %d cells away: its far search spent more than the %d generations of a creature larger than one cell",
					live.world.Tick(), e.ID, e.Owner, e.TokenSize, len(s.route), goal[0], goal[1], d, releaseLargeBudget(d))
			}
		}
	}
	if onHero+beside == 0 {
		t.Fatalf("%d routes onto the hero's cell and %d that stop beside it from over %d cells within %d ticks: the drive does not reach the state under test",
			onHero, beside, releaseLargeParted, releaseLargeHorizon)
	}
}
