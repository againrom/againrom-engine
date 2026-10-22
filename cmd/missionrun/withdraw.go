package main

import (
	"fmt"
	"io"

	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// withdrawalWitness is one production mission tick whose final AI tail
// replaced the selected actor's preceding order with a retreat destination.
// The instrument keeps only the facts it prints; none reaches the world.
type withdrawalWitness struct {
	mode, class       string
	entity            sim.EntityID
	tick              uint64
	threshold, radius int32
	beforeX, beforeY  int32
	beforeHasTarget   bool
	beforeTargetX     int32
	beforeTargetY     int32
	beforeHasAttack   bool
	beforeAttack      sim.EntityID
	afterX, afterY    int32
	afterTargetX      int32
	afterTargetY      int32
	hostiles          int

	// 1047: the production canonical turn state at the same two ticks the
	// position and target fields above already capture. This instrument
	// does not step past tick 7 to observe a return turn after
	// reacquisition (contract.md's "Witnesses" section names that a
	// separate, still-open requirement) — it reports only what the existing
	// two-tick capture already has in hand.
	rotationSpeed               int32
	beforeFacing, beforeDesired uint8
	beforeRemaining             uint8
	afterFacing, afterDesired   uint8
	afterRemaining              uint8
}

// driveWithdrawal starts from the mission's real production world, wounds one
// eligible authored actor to its threshold by the script's health write, then
// reaches phase 6 by the same sim.Step calls the game uses. When no hostile is
// inside the actor's search square yet, the ordinary dispatcher runs that tail
// and the instrument waits for the next phase 6, up to withdrawalCycles of
// them. Each candidate starts from a binary clone of the untouched world, so a
// failed attempt cannot perturb the next.
func driveWithdrawal(ms *game.Mission, table *mapload.Table, out io.Writer) error {
	if ms == nil || ms.World == nil {
		return fmt.Errorf("withdrawal: mission has no world")
	}
	form, err := ms.World.MarshalBinary()
	if err != nil {
		return err
	}
	// Prefer the literal-radius Withdraw arm. Wimpy remains a complete
	// fallback and has its own release/unit witnesses, but the story's visible
	// result should point at the ranged-monster threshold where one is present.
	for _, wantedMode := range []string{"withdraw", "wimpy"} {
		for _, candidate := range ms.World.Entities() {
			threshold, hostiles := withdrawalCandidate(ms.World, candidate, wantedMode)
			if threshold <= 0 || len(hostiles) == 0 {
				continue
			}
			var world sim.World
			if err := world.UnmarshalBinary(form); err != nil {
				return err
			}
			// A Group the party's coverage does not reach skips its whole
			// dispatch, the withdrawal tail included (AI-ACTIVITY-324). The
			// witness runs in safe mode, which dispatches every Group, so a
			// candidate far from the party still reaches the tail.
			world.SetSafeMode(true)
			before, ok := withdrawalEntity(world.Entities(), candidate.ID)
			if !ok || before.HP <= 0 || before.MaxHP <= 0 {
				continue
			}
			if damage := before.HP - threshold; damage > 0 {
				if err := world.HeadlessDamage(before.ID, damage); err != nil {
					continue
				}
			}
			sim.Step(&world, nil)
			// The wound lands before tick zero. The fifth quiet step leaves the
			// world at tick 6; the sixth executes the dispatcher and its tail.
			// A phase 6 that finds no hostile in the square runs as an ordinary
			// step, and the next phase 6 is 16 ticks later; the actor is wounded
			// back to its threshold just before it.
			var pre sim.Entity
			var mode string
			var triggeredAt, radius int32
			var liveHostiles []sim.Entity
			for cycle := 0; cycle < withdrawalCycles; cycle++ {
				quiet := 5
				if cycle > 0 {
					sim.Step(&world, nil)
					quiet = 15
				}
				for i := 0; i < quiet; i++ {
					sim.Step(&world, nil)
				}
				pre, ok = withdrawalEntity(world.Entities(), before.ID)
				if !ok || pre.HP <= 0 {
					break
				}
				// Regeneration may lift the actor past its threshold between
				// passes; the script's health write wounds it back.
				if damage := pre.HP - threshold; cycle > 0 && damage > 0 {
					if world.HeadlessDamage(pre.ID, damage) != nil {
						break
					}
					pre, ok = withdrawalEntity(world.Entities(), before.ID)
				}
				mode, triggeredAt, radius, liveHostiles = withdrawalTrigger(&world, pre)
				if mode != "" {
					break
				}
			}
			if !ok || pre.HP <= 0 || mode != wantedMode || len(liveHostiles) == 0 || pre.HP > triggeredAt {
				continue
			}
			tailTick := world.Tick()
			decisions := sim.StepWithdrawalTraced(&world, nil)
			after, afterOK := withdrawalEntity(world.Entities(), before.ID)
			decision, decisionOK := withdrawalDecision(decisions, before.ID)
			if !afterOK || !decisionOK || !after.HasTarget || !decision.After.HasTarget {
				continue
			}
			// A retreat points to the actor's side of the hostile mean. Reject an
			// ordinary attack approach that merely set a target on the same phase.
			meanX, meanY := withdrawalMean(liveHostiles)
			awayX, awayY := pre.X-meanX, pre.Y-meanY
			toTargetX, toTargetY := decision.After.TargetX-pre.X, decision.After.TargetY-pre.Y
			if awayX*toTargetX+awayY*toTargetY <= 0 {
				continue
			}
			ordinary := decision.Before
			witness := withdrawalWitness{
				mode: mode, class: withdrawalClassName(ms, table, before.ID, before.Class),
				entity: before.ID, threshold: triggeredAt, radius: radius, tick: tailTick,
				beforeX: ordinary.X, beforeY: ordinary.Y, beforeHasTarget: ordinary.HasTarget,
				beforeTargetX: ordinary.TargetX, beforeTargetY: ordinary.TargetY,
				beforeHasAttack: ordinary.HasAttackTarget, beforeAttack: ordinary.AttackTarget,
				afterX: after.X, afterY: after.Y,
				afterTargetX: decision.After.TargetX, afterTargetY: decision.After.TargetY,
				hostiles: len(liveHostiles),

				rotationSpeed: ordinary.RotationSpeed,
				beforeFacing:  ordinary.Facing, beforeDesired: ordinary.DesiredFacing, beforeRemaining: ordinary.TurnRemaining,
				afterFacing: after.Facing, afterDesired: after.DesiredFacing, afterRemaining: after.TurnRemaining,
			}
			printWithdrawalWitness(out, ms, witness)
			return nil
		}
	}
	return fmt.Errorf("withdrawal: mission %d has no eligible authored actor beside a hostile", ms.Number)
}

// withdrawalCycles bounds the phase-6 passes the instrument waits for an
// actor that sees a hostile to come within its search square.
const withdrawalCycles = 32

func withdrawalDecision(decisions []sim.WithdrawalDecision, id sim.EntityID) (sim.WithdrawalDecision, bool) {
	for _, decision := range decisions {
		if decision.Before.ID == id {
			return decision, true
		}
	}
	return sim.WithdrawalDecision{}, false
}

func withdrawalTrigger(w *sim.World, actor sim.Entity) (string, int32, int32, []sim.Entity) {
	if actor.HP <= 0 || actor.Owner == 0 {
		return "", 0, 0, nil
	}
	if actor.Wimpy > 0 && actor.HP <= actor.Wimpy {
		radius := wimpyRadius(w, actor)
		h := withdrawalHostiles(w, actor, radius)
		if len(h) > 0 {
			return "wimpy", actor.Wimpy, radius, h
		}
	}
	if actor.Withdraw > 0 && actor.HP <= actor.Withdraw {
		h := withdrawalHostiles(w, actor, 2)
		if len(h) > 0 {
			return "withdraw", actor.Withdraw, 2, h
		}
	}
	return "", 0, 0, nil
}

func withdrawalCandidate(w *sim.World, actor sim.Entity, mode string) (int32, []sim.Entity) {
	if actor.HP <= 0 || actor.Owner == 0 {
		return 0, nil
	}
	switch mode {
	case "withdraw":
		if actor.Withdraw > 0 {
			return actor.Withdraw, withdrawalHostiles(w, actor, max(2, int32(actor.ScanRange)))
		}
	case "wimpy":
		// The story's pointable witness is a ranged-monster actor. A Wimpy-
		// only animal exercises the same tail but is not evidence for that
		// population; requiring Withdraw here selects the shipped ranged set
		// while the production rule itself remains general.
		if actor.Wimpy > 0 && actor.Withdraw > 0 {
			return actor.Wimpy, withdrawalHostiles(w, actor, max(wimpyRadius(w, actor), int32(actor.ScanRange)))
		}
	}
	return 0, nil
}

// withdrawalHostiles mirrors the first-game withdrawal square for radius,
// centred on the actor's anchor, its top-left footprint cell (AI-696): a living
// ground hostile is listed once per footprint cell inside it, a flyer never
// (AI-697, AI-698). The entries are what the flee point averages.
func withdrawalHostiles(w *sim.World, actor sim.Entity, radius int32) []sim.Entity {
	rel := w.Relations()
	var out []sim.Entity
	for _, other := range w.Entities() {
		if other.ID == actor.ID || other.HP <= 0 || other.Domain == sim.DomainAir || !rel.Hostile(actor.Owner, other.Owner) {
			continue
		}
		side := int32(max(other.TokenSize, 1))
		for dy := int32(0); dy < side; dy++ {
			for dx := int32(0); dx < side; dx++ {
				if withdrawalAbs(other.X+dx-actor.X) <= radius && withdrawalAbs(other.Y+dy-actor.Y) <= radius {
					out = append(out, other)
				}
			}
		}
	}
	return out
}

// wimpyRadius is the Wimpy search radius, mover byte +0x08: the constructor's
// 5 for every actor of a mission start (AI-690, AI-692).
func wimpyRadius(*sim.World, sim.Entity) int32 {
	return int32(sim.ConstructorSearchRadius)
}

func withdrawalMean(ents []sim.Entity) (int32, int32) {
	var x, y int64
	for _, e := range ents {
		x += int64(e.X)
		y += int64(e.Y)
	}
	return int32(x / int64(len(ents))), int32(y / int64(len(ents)))
}

func withdrawalEntity(ents []sim.Entity, id sim.EntityID) (sim.Entity, bool) {
	for _, e := range ents {
		if e.ID == id {
			return e, true
		}
	}
	return sim.Entity{}, false
}

func withdrawalClassName(ms *game.Mission, table *mapload.Table, id sim.EntityID, class int32) string {
	if ms != nil && ms.Map != nil && int(id) >= 0 && int(id) < len(ms.Map.Units) && table != nil {
		r := mapload.Resolve(ms.Map.Units[int(id)], table)
		if r.Arm == mapload.ArmUnits && r.Found() && table.Units != nil {
			return table.Units.EntryName(r.Index)
		}
	}
	if table == nil || table.Units == nil || class < 0 || int(class) >= table.Units.Len() {
		return fmt.Sprintf("entry-%d", class)
	}
	return table.Units.EntryName(int(class))
}

func printWithdrawalWitness(out io.Writer, ms *game.Mission, w withdrawalWitness) {
	beforeTarget := "none"
	if w.beforeHasTarget {
		beforeTarget = fmt.Sprintf("(%d,%d)", w.beforeTargetX, w.beforeTargetY)
	}
	beforeAttack := "none"
	if w.beforeHasAttack {
		beforeAttack = fmt.Sprintf("%d", w.beforeAttack)
	}
	fmt.Fprintf(out, "withdrawal mission=%d map=%q entity=%d class=%q mode=%s threshold=%d radius=%d hostiles=%d rotationspeed=%d\n",
		ms.Number, ms.Address, w.entity, w.class, w.mode, w.threshold, w.radius, w.hostiles, w.rotationSpeed)
	fmt.Fprintf(out, "  before-tail tick=%d cell=(%d,%d) target=%s attack=%s facing=%d desired=%d remaining=%d\n",
		w.tick, w.beforeX, w.beforeY, beforeTarget, beforeAttack, w.beforeFacing, w.beforeDesired, w.beforeRemaining)
	fmt.Fprintf(out, "  after  tick=%d cell=(%d,%d) target=(%d,%d) facing=%d desired=%d remaining=%d\n",
		w.tick+1, w.afterX, w.afterY, w.afterTargetX, w.afterTargetY, w.afterFacing, w.afterDesired, w.afterRemaining)
}

func withdrawalAbs(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
