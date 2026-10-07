package game

import (
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The Defend, Patrol and structure use orders a player gives, played through the
// App against the arena of the loaded attack cycle witness. Each of them stores
// a state, a destination or a building and no attack progress, so a warrior whose
// blow is already loaded resolves it before the order takes hold
// (AI-ORDER-039, HERO-CADENCE-112; DIV-1577). A different victim still gives the
// blow up (DIV-1564).

// landing requires the warrior to be loaded on the creature when the order the
// last click queued lands on the next tick, and returns that tick with the ticks
// of the charge the order finds left.
func (arena *loadedCycleArena) landing(t *testing.T, hero, victim sim.EntityID) (applied int64, left int32) {
	t.Helper()
	arena.loadedOnLanding(t, hero, victim)
	return arena.now() + 1, arena.get(hero).AttackCountdown
}

// requireRetained lands the queued order and requires the warrior to still hold
// the charge he had loaded on the creature.
func (arena *loadedCycleArena) requireRetained(t *testing.T, hero, victim sim.EntityID) sim.Entity {
	t.Helper()
	arena.live.tick()
	h := arena.get(hero)
	t.Logf("after the order: victim %v/%d phase %d countdown %d state %d destination %v (%d,%d) at (%d,%d)",
		h.HasAttackTarget, h.AttackTarget, h.AttackPhase, h.AttackCountdown, h.ActorState, h.HasTarget, h.TargetX, h.TargetY, h.X, h.Y)
	if !h.HasAttackTarget || h.AttackTarget != victim || h.AttackPhase != sim.AttackCharging {
		t.Fatalf("the order dropped the loaded cycle: victim %v/%d phase %d", h.HasAttackTarget, h.AttackTarget, h.AttackPhase)
	}
	return h
}

// requireLoadedBlow requires the blow the warrior had loaded when the order
// landed to have resolved on the tick that charge ended, sooner than any fresh
// charge could have, with the warrior still on his cell and the creature the
// poorer for it.
func requireLoadedBlow(t *testing.T, out blowOutcome, applied int64, left, charge int32) {
	t.Helper()
	t.Logf("the order landed at tick %d with %d ticks of the charge left: the blow resolved at tick %d and took %d health; the warrior left his cell at tick %d",
		applied, left, out.strikeTick, out.lost, out.leftTick)
	if out.strikeTick < 0 || out.lost <= 0 {
		t.Fatalf("the loaded blow never landed: it resolved at tick %d and took %d health", out.strikeTick, out.lost)
	}
	if since := out.strikeTick - applied; since > int64(left)+1 || since >= int64(charge) {
		t.Fatalf("the blow resolved %d ticks after the order, want the %d the loaded charge had left and fewer than a fresh charge's %d",
			since, left, charge)
	}
	if out.leftTick >= 0 && out.leftTick <= out.strikeTick {
		t.Fatalf("the warrior left his cell at tick %d, before his blow resolved at tick %d", out.leftTick, out.strikeTick)
	}
}

// orderDefend arms Defend with its key and clicks the unit with the current
// selection standing, and requires the one command the click queued. It costs
// three frames.
func (arena *loadedCycleArena) orderDefend(t *testing.T, subject sim.EntityID) {
	t.Helper()
	if err := arena.app.HeadlessKey("defend"); err != nil {
		t.Fatal(err)
	}
	x, y, err := arena.app.HeadlessEntityPoint(uint32(subject))
	if err != nil {
		t.Fatal(err)
	}
	arena.tap(t, x, y)
	if got := arena.live.pending; len(got) != 1 || got[0].Kind != sim.KindGroupDefend || got[0].Entity != arena.heroes[0] ||
		got[0].X != int32(subject) {
		t.Fatalf("the click queued %+v, want the warrior's Defend of unit %d", got, subject)
	}
}

// orderPatrol arms Patrol with its key and clicks the ground cell with the
// current selection standing, and requires the one command the click queued. It
// costs three frames.
func (arena *loadedCycleArena) orderPatrol(t *testing.T, col, row int32) {
	t.Helper()
	if err := arena.app.HeadlessKey("patrol"); err != nil {
		t.Fatal(err)
	}
	got := arena.clickGround(t, col, row)
	if len(got) != 1 || got[0].Kind != sim.KindGroupPatrolTo || got[0].Entity != arena.heroes[0] || got[0].X != col || got[0].Y != row {
		t.Fatalf("the click queued %+v, want the warrior's Patrol to (%d,%d)", got, col, row)
	}
}

// orderUse centres the view on the structure and hovers and presses on it with
// the current selection standing, and requires the one command the click queued.
// It costs three frames.
func (arena *loadedCycleArena) orderUse(t *testing.T, s sim.Structure) {
	t.Helper()
	inspectionCentre(arena.live, int(s.Col), int(s.Row))
	arena.live.push()
	x, y, err := arena.live.view.InspectionPoint(ui.InspectionSubject{Kind: ui.InspectionStructure, ID: uint32(s.ID)})
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"hover", "press", "release"} {
		if err := arena.app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	if got := arena.live.pending; len(got) != 1 || got[0].Kind != sim.KindUseStructure || got[0].Entity != arena.heroes[0] ||
		got[0].X != int32(s.ID) {
		t.Fatalf("the click queued %+v, want the warrior's use of structure %d", got, s.ID)
	}
}

// A warrior told to Defend a unit while his blow was on the way gave the blow up
// and went about the order at once. The original finishes the blow it began and
// then carries the order out (AI-CMD-054, AI-FOLLOWSET-116, AI-CMD-033,
// AI-ORDER-039, HERO-CADENCE-112; DIV-1577).
//
// The warrior is selected, ordered onto a creature with the attack key and a
// click, and ordered again in the middle of his charge with the Defend key and a
// click on a unit, all through App input. Every App frame advances the world by
// one tick, so the order lands with at least one tick of the charge left. The
// creature never strikes back and its health is what the warrior's blow takes.
func TestReleaseALoadedAttackCycleFinishesBeforeADefendOrder(t *testing.T) {
	t.Run("acquire in place: the blow lands", func(t *testing.T) {
		arena := openLoadedCycleArena(t, 1)
		hero, east := arena.heroes[0], arena.east
		arena.selectFirstHero(t)
		arena.orderAttack(t, east)
		arena.untilLoaded(t, hero, east)
		charge := arena.get(hero).AttackCharge
		arena.orderDefend(t, hero)
		applied, left := arena.landing(t, hero, east)
		if h := arena.requireRetained(t, hero, east); h.ActorState != 0xc {
			t.Fatalf("the actor state is %d, want acquire 0xc written by the order", h.ActorState)
		}
		requireLoadedBlow(t, arena.watch(hero, east, 60), applied, left, charge)
	})

	t.Run("escort a companion: the blow lands and then the warrior walks to him", func(t *testing.T) {
		arena := openLoadedCycleArena(t, 2)
		hero, companion, east := arena.heroes[0], arena.heroes[1], arena.east
		arena.selectFirstHero(t)
		arena.orderAttack(t, east)
		arena.untilLoaded(t, hero, east)
		charge, start, mate := arena.get(hero).AttackCharge, arena.get(hero), arena.get(companion)
		arena.orderDefend(t, companion)
		applied, left := arena.landing(t, hero, east)
		if h := arena.requireRetained(t, hero, east); h.ActorState != 8 || !h.HasEscortTarget || h.EscortTarget != companion {
			t.Fatalf("the actor state is %d escorting %v/%d, want defend 8 of unit %d written by the order",
				h.ActorState, h.HasEscortTarget, h.EscortTarget, companion)
		}
		requireLoadedBlow(t, arena.watch(hero, east, 400), applied, left, charge)
		for range 400 {
			arena.live.tick()
			if cellDistance(arena.get(hero), mate.X, mate.Y) <= 3 {
				break
			}
		}
		end := arena.get(hero)
		if cellDistance(end, mate.X, mate.Y) > 3 {
			t.Fatalf("the warrior stands at (%d,%d), want him within three cells of the unit he defends at (%d,%d)", end.X, end.Y, mate.X, mate.Y)
		}
		t.Logf("the warrior walked from (%d,%d) to (%d,%d) beside the unit at (%d,%d) at tick %d", start.X, start.Y, end.X, end.Y, mate.X, mate.Y, arena.now())
	})
}

// A warrior told to Patrol while his blow was on the way gave the blow up. The
// original finishes it (AI-PATROL-018, AI-PATROL-017, AI-CMD-033, AI-ORDER-039,
// HERO-CADENCE-112; DIV-1577). The order is the Patrol key and a click on the
// ground, given in the middle of the charge.
func TestReleaseALoadedAttackCycleFinishesBeforeAPatrolOrder(t *testing.T) {
	arena := openLoadedCycleArena(t, 1)
	hero, east := arena.heroes[0], arena.east
	arena.selectFirstHero(t)
	arena.orderAttack(t, east)
	arena.untilLoaded(t, hero, east)
	charge := arena.get(hero).AttackCharge
	arena.orderPatrol(t, arena.cx-4, arena.cy-3)
	applied, left := arena.landing(t, hero, east)
	if h := arena.requireRetained(t, hero, east); h.ActorState != 0xa {
		t.Fatalf("the actor state is %d, want patrol 0xa written by the order", h.ActorState)
	}
	requireLoadedBlow(t, arena.watch(hero, east, 60), applied, left, charge)
}

// A warrior sent to use a structure while his blow was on the way gave the blow
// up and walked at once. The original finishes the blow and then walks
// (AI-STRUCTUSE-306, AI-ORDER-039, HERO-CADENCE-112; DIV-1577). The arena stands
// on the open ground of mission 20 nearest its fountain, so the warrior has a
// walk to the fountain; the order is a click on the fountain with the warrior
// selected.
func TestReleaseALoadedAttackCycleFinishesBeforeAStructureUse(t *testing.T) {
	const fountain = 29
	ready := func(t *testing.T) (*loadedCycleArena, sim.Structure) {
		t.Helper()
		arena := openLoadedCycleArenaBeside(t, 1, fountain)
		var s sim.Structure
		for _, c := range arena.live.world.Structures() {
			if int(c.ID) == fountain {
				s = c
			}
		}
		if !s.Usable() || int16(s.Field42) <= 0 {
			t.Fatalf("the fountain is not usable in the arena: %+v", s)
		}
		hero, east := arena.heroes[0], arena.east
		arena.selectFirstHero(t)
		arena.orderAttack(t, east)
		arena.untilLoaded(t, hero, east)
		return arena, s
	}
	charges := func(arena *loadedCycleArena) uint16 {
		for _, c := range arena.live.world.Structures() {
			if int(c.ID) == fountain {
				return c.Field42
			}
		}
		return 0
	}

	t.Run("use: the blow lands, the warrior walks to the fountain and uses it", func(t *testing.T) {
		arena, s := ready(t)
		hero, east := arena.heroes[0], arena.east
		charge, start, before := arena.get(hero).AttackCharge, arena.get(hero), charges(arena)
		arena.orderUse(t, s)
		applied, left := arena.landing(t, hero, east)
		arena.requireRetained(t, hero, east)
		if uses := arena.live.world.StructureUses(); len(uses) != 1 || uses[0].Entity != hero || uses[0].Structure != s.ID {
			t.Fatalf("the structure uses are %v, want the warrior's use of the fountain", uses)
		}
		requireLoadedBlow(t, arena.watch(hero, east, 400), applied, left, charge)
		for range 1500 {
			if len(arena.live.world.StructureUses()) == 0 {
				break
			}
			arena.live.tick()
		}
		end := arena.get(hero)
		if len(arena.live.world.StructureUses()) != 0 || charges(arena) != before-1 || cellDistance(end, s.Col, s.Row) > 1 {
			t.Fatalf("the fountain was not used: uses %v, charges %d to %d, the warrior at (%d,%d) beside (%d,%d)",
				arena.live.world.StructureUses(), before, charges(arena), end.X, end.Y, s.Col, s.Row)
		}
		t.Logf("the warrior walked from (%d,%d) to (%d,%d) and used the fountain at tick %d: charges %d to %d",
			start.X, start.Y, end.X, end.Y, arena.now(), before, charges(arena))
	})

	t.Run("save and load while the use waits behind the blow", func(t *testing.T) {
		arena, s := ready(t)
		hero, east := arena.heroes[0], arena.east
		before := charges(arena)
		arena.orderUse(t, s)
		arena.landing(t, hero, east)
		held := arena.requireRetained(t, hero, east)
		restored, control := arena.reload(t)
		back, _ := restored.entity(hero)
		t.Logf("after load: victim %v/%d phase %d countdown %d, %d structure uses", back.HasAttackTarget, back.AttackTarget,
			back.AttackPhase, back.AttackCountdown, len(restored.world.StructureUses()))
		if !back.HasAttackTarget || back.AttackTarget != east || back.AttackPhase != sim.AttackCharging ||
			back.AttackCountdown != held.AttackCountdown || len(restored.world.StructureUses()) != 1 {
			t.Fatalf("the loaded game lost the waiting use: victim %v/%d phase %d countdown %d, saved %d, uses %v",
				back.HasAttackTarget, back.AttackTarget, back.AttackPhase, back.AttackCountdown, held.AttackCountdown,
				restored.world.StructureUses())
		}
		for n := 0; n < 1500 && len(restored.world.StructureUses()) > 0; n++ {
			sim.Step(control, nil)
			sim.Step(restored.world, nil)
			if control.Hash() != restored.world.Hash() {
				t.Fatalf("the loaded game differs from the saved world bytes %d ticks after the load", n+1)
			}
		}
		used := uint16(0)
		for _, c := range restored.world.Structures() {
			if int(c.ID) == fountain {
				used = c.Field42
			}
		}
		if len(restored.world.StructureUses()) != 0 || used != before-1 {
			t.Fatalf("the loaded warrior never used the fountain: uses %v, charges %d to %d",
				restored.world.StructureUses(), before, used)
		}
		t.Logf("the loaded warrior used the fountain at tick %d with the world hash equal to the control's throughout", restored.world.Tick())
	})
}

// untilCycleEnding advances until the hero's cycle is in its recovery with at
// most left ticks of it remaining, and requires that to happen in the cycle
// loaded on victim.
func (arena *loadedCycleArena) untilCycleEnding(t *testing.T, hero, victim sim.EntityID, left int32) {
	t.Helper()
	for range 200 {
		arena.live.tick()
		if h := arena.get(hero); h.HasAttackTarget && h.AttackTarget == victim && h.AttackPhase == sim.AttackRelaxing && h.AttackCountdown <= left {
			return
		}
	}
	t.Fatalf("the warrior's cycle on creature %d never reached its last %d ticks of recovery", victim, left)
}

// cyclesLoaded advances n ticks and counts the cycles the hero begins.
func (arena *loadedCycleArena) cyclesLoaded(hero sim.EntityID, n int) int {
	loads, prev := 0, arena.get(hero).AttackPhase
	for range n {
		arena.live.tick()
		now := arena.get(hero).AttackPhase
		if prev == sim.AttackReady && now != sim.AttackReady {
			loads++
		}
		prev = now
	}
	return loads
}

// A warrior told to Defend a companion as his cycle was ending loaded one more
// cycle on the creature before the order took hold. The setter stores a state
// and pending order 0 and no progress, so once the cycle has ended nothing
// loads another (AI-CMD-054, AI-FOLLOWSET-116, AI-ORDER-039, HERO-CADENCE-112;
// DIV-1577). The order is the Defend key and a click on the companion given in
// the last ticks of the recovery.
func TestReleaseADefendOrderGivenAsACycleEndsLoadsNoSecondCycle(t *testing.T) {
	ending := 0
	for left := int32(1); left <= 12; left++ {
		arena := openLoadedCycleArena(t, 2)
		hero, companion, east := arena.heroes[0], arena.heroes[1], arena.east
		arena.selectFirstHero(t)
		arena.orderAttack(t, east)
		arena.untilLoaded(t, hero, east)
		arena.untilCycleEnding(t, hero, east, left)
		arena.orderDefend(t, companion)
		h := arena.get(hero)
		if h.AttackPhase == sim.AttackReady {
			continue
		}
		ending++
		loads := arena.cyclesLoaded(hero, 200)
		t.Logf("recovery ticks left at the click %d: the order landed in phase %d with %d ticks left; %d more cycles loaded",
			left, h.AttackPhase, h.AttackCountdown, loads)
		if loads != 0 {
			t.Fatalf("the warrior loaded %d more cycles after the Defend order given with %d recovery ticks left", loads, left)
		}
		if got := arena.get(hero); got.ActorState != 8 {
			t.Fatalf("the actor state is %d, want defend 8 written by the order", got.ActorState)
		}
	}
	if ending == 0 {
		t.Fatal("no order landed while a cycle was ending")
	}
}

// A saved game written while a Defend order waits behind the blow loads with the
// blow still loaded and carries on identically to the world bytes it was saved
// from: the blow lands, no second cycle loads, and the warrior walks to the
// companion (DIV-1577).
func TestReleaseADefendOrderWaitingBehindABlowSurvivesSaveAndLoad(t *testing.T) {
	arena := openLoadedCycleArena(t, 2)
	hero, companion, east := arena.heroes[0], arena.heroes[1], arena.east
	arena.selectFirstHero(t)
	arena.orderAttack(t, east)
	arena.untilLoaded(t, hero, east)
	arena.orderDefend(t, companion)
	arena.landing(t, hero, east)
	held := arena.requireRetained(t, hero, east)
	restored, control := arena.reload(t)
	back, _ := restored.entity(hero)
	if !back.HasAttackTarget || back.AttackTarget != east || back.AttackPhase != sim.AttackCharging ||
		back.AttackCountdown != held.AttackCountdown || back.ActorState != 8 {
		t.Fatalf("the loaded game lost the waiting order: victim %v/%d phase %d countdown %d (saved %d) state %d",
			back.HasAttackTarget, back.AttackTarget, back.AttackPhase, back.AttackCountdown, held.AttackCountdown, back.ActorState)
	}
	hp := func(w *sim.World) int32 {
		for _, e := range w.Entities() {
			if e.ID == east {
				return e.HP
			}
		}
		return 0
	}
	start := hp(restored.world)
	low := start
	for n := range 400 {
		sim.Step(control, nil)
		sim.Step(restored.world, nil)
		low = min(low, hp(restored.world))
		if control.Hash() != restored.world.Hash() {
			t.Fatalf("the loaded game differs from the saved world bytes %d ticks after the load", n+1)
		}
	}
	end, _ := restored.entity(hero)
	if lost := start - low; lost <= 0 || end.X == held.X && end.Y == held.Y {
		t.Fatalf("after the load the creature lost %d health and the warrior stands at (%d,%d), want the loaded blow and the walk to the companion",
			lost, end.X, end.Y)
	}
}
