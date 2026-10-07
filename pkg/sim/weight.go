package sim

import (
	"fmt"
	"sort"
)

// ItemWeight is the deterministic code-table fallback for native and legacy
// items without an explicit weight. Imported SAV instances own their signed
// Item+0x4a value (ITEM-STACK-003); that value takes precedence even when zero.
// Table resolution stays above pkg/sim's floating-point determinism boundary.
type ItemWeight struct {
	Code   uint16
	Weight int32
	// Constructor is resolved only for worlds containing retained actors.
	// It is a fresh definition-derived object, never a saved item's operand
	// override. The optional source-equipment section hashes it with this row.
	Constructor SourceEquipment
}

const (
	carrySaturation    int32 = 0xfa00
	carrySaturatedLoad int32 = 0x7d00
	carryHalving       int32 = 2
)

// The overload penalty's own two constants (HERO-SPEED-008, High).
// overloadFloor is the `max(speed, 6)` at L03930, and it
// sits INSIDE the penalty arm rather than over the whole speed law: it can
// only be reached by an actor whose load has already met his capacity.
//
// THE FLOOR CAN RAISE A SPEED, and that is the original's arithmetic rather
// than an oversight here. An actor whose unencumbered speed is below six and
// who is overloaded leaves this function at six, faster than he was. The claim
// gives the compare as unconditional inside the arm, so reproducing it means
// reproducing that; narrowing it to "never raise" would be this tree choosing a
// rule the original does not state.
const (
	overloadFloor    int32 = 6
	overloadDivisorZ int32 = 0 // capacity below which no penalty is evaluated
)

// normaliseItemWeights is the one legal representation of a weight table:
// sorted by code, at most one entry per code, and no entry for the zero code.
//
// IT REFUSES A DISAGREEMENT rather than picking a winner. Two entries naming
// one code with two weights is a caller that resolved the same item twice and
// got two answers, which cannot be true of a table whose weights are a function
// of the code — so it is a fault in what was handed over, not a shape to fold.
// Two entries naming one code with the SAME weight are folded silently, because
// a caller walking several containers reaches one code many times and that is
// not a disagreement about anything.
//
// THE ZERO CODE IS DROPPED, not refused, on carryFault's own split: zero is
// never an item and a zero travelling with a map's own null is not a mistake
// worth an error.
//
// The result is a fresh slice, so a caller may reuse the one it passed.
func normaliseItemWeights(in []ItemWeight) ([]ItemWeight, error) {
	out := make([]ItemWeight, 0, len(in))
	for _, w := range in {
		if w.Code == 0 {
			continue
		}
		if err := w.Constructor.Validate(); err != nil {
			return nil, err
		}
		if w.Constructor.Spell != (SourceItemSpell{}) || w.Constructor.EffectsUnsupported {
			return nil, fmt.Errorf("sim: item constructor carries instance-only state")
		}
		out = append(out, w)
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Code < out[b].Code })
	k := 0
	for i := range out {
		if i > 0 && out[i].Code == out[k-1].Code {
			if out[i].Weight != out[k-1].Weight {
				return nil, fmt.Errorf("sim: item code %d is declared at weight %d and at weight %d",
					out[i].Code, out[k-1].Weight, out[i].Weight)
			}
			if out[i].Constructor.Class != 0 {
				if out[k-1].Constructor.Class != 0 && out[k-1].Constructor != out[i].Constructor {
					return nil, fmt.Errorf("sim: item code %d has conflicting constructors", out[i].Code)
				}
				out[k-1].Constructor = out[i].Constructor
			}
			continue
		}
		out[k] = out[i]
		k++
	}
	return out[:k], nil
}

// DeclareItemWeights adds ws to this world's weight table and recomputes
// every actor's load against the result.
//
// IT IS ADDITIVE AND IDEMPOTENT: declaring a code already declared at the same
// weight changes nothing, and declaring one at a different weight is refused
// with the table left exactly as it was. That is what lets the two producers a
// world can have — the map load that builds it, and a caller that later
// replaces an actor's whole holdings out of an original save (ReplaceStock) —
// each declare what they know without either having to hold the other's list.
//
// IT IS A CONSTRUCTION-TIME ACT, ReplaceStock's own standing: it is not a tick,
// nothing schedules it, and calling it while a mission runs would move every
// actor's load in one statement. Nothing in this package calls it.
//
// Without a table, only explicit instance weights contribute. Native and
// predecessor items without such a value keep the historical zero fallback.
func (w *World) DeclareItemWeights(ws []ItemWeight) error {
	merged, err := normaliseItemWeights(append(append([]ItemWeight(nil), w.itemWeights...), ws...))
	if err != nil {
		return err
	}
	w.itemWeights = merged
	w.recomputeLoads()
	return nil
}

// ItemWeights is this world's whole weight table, in code order — a copy, so a
// caller cannot write through into the world's own.
func (w *World) ItemWeights() []ItemWeight {
	return append([]ItemWeight(nil), w.itemWeights...)
}

// itemWeightOf is what one unit of code weighs, and ZERO FOR A CODE THE
// TABLE DOES NOT NAME. A code nothing declared is an item this world knows
// the identity of and not the weight of, and there is no third answer
// available to a package that cannot open a definition table: the choice is
// between weighing it as nothing and refusing to hold it at all, and
// refusing would make a script's own give-item instant able to fail a
// mission.
//
// DIVERGENCES.md DIV-223 carries the row: the original's weight travels on
// the item object, so it has no undeclared case, and the cost of resolving
// above the determinism wall is a producer population that has to be kept
// complete by hand.
func (w *World) itemWeightOf(code uint16) int32 {
	i := sort.Search(len(w.itemWeights), func(i int) bool { return w.itemWeights[i].Code >= code })
	if i < len(w.itemWeights) && w.itemWeights[i].Code == code {
		if w.itemWeights[i].Constructor.Class != 0 {
			return int32(int16(w.itemWeights[i].Weight))
		}
		return w.itemWeights[i].Weight
	}
	return 0
}

// containerWeight is the container's own running sum, `Σ (weight x count)`
// over its elements — the original's `[actor+0x7c]+0x20`, maintained there
// by four IMUL sites inside the container class and computed here by walking
// the elements instead (ITEM-STACK-003, High).
//
// This recomputation is not preservation of a saved accumulator or current
// actor load. Their distinct state and mutation timing remain DIV-755.
//
// THE ARITHMETIC IS 32-BIT AND WRAPS, which is the original's dword and not an
// accident here. Count is uint32 in this package and u16 in the original (0138
// D-1's own widening), so a container this package can build but the original
// cannot may reach a product no dword holds; wrapping is what the original's
// own IMUL does with the low half, and it is a defined operation in Go for a
// sized signed type.
func (w *World) containerWeight(i int) int32 {
	var sum int32
	for _, st := range w.carried[i] {
		weight := w.itemWeightOf(st.Code)
		if st.WeightPresent {
			weight = int32(st.Weight)
		}
		sum += weight * int32(st.Count)
	}
	return sum
}

// wornWeight is the actor's OWN weight, `actor+0x8e`: the total weight of
// what he is wearing and holding.
//
// IT IS NOT A BODY WEIGHT AND NOT A COLUMN. So the field is an accumulator
// over the worn set, and an actor wearing nothing has a zero in it. That is
// what makes a drawn WEIGHT row of `0.0` a character with an empty doll
// rather than a defect -- the owner's own FERGARD card, whose ATTACK row is
// likewise absent.
//
// A SLOT'S CODE COUNTS ONCE. Equipment slots hold single items, not stacks.
func (w *World) wornWeight(i int) int32 {
	var sum int32
	for _, item := range w.equipment[i] {
		if item.Code == 0 {
			continue
		}
		sum += w.instanceWeight(item)
	}
	return sum
}

// loadOf is the carried load, `actor+0x90`, exactly as ITEM-LOAD-005 gives
// the derive at `L03920`..`L03921`:
//
//	load = ownWeight
//	if container != nil:
//	        if container.sum >= 0xfa00 : load  = 0x7d00
//	        else                       : load += container.sum / 2
//
// WHAT YOU WEAR COUNTS IN FULL AND WHAT YOU CARRY COUNTS HALF. That is the
// whole of the rule and it is the reason the two terms are computed separately.
//
// THE `container != nil` ARM IS ALWAYS TAKEN HERE. Every entity in this package
// has a container — a slice, possibly empty — where the original allocates one
// and may hold a null pointer. An empty container sums to zero and adds zero,
// so the two builds agree on the value for an actor with no container: the arm
// is unreachable rather than dropped.
//
// THE SATURATION ASSIGNS AND DOES NOT ADD. A container at or past 64 000 puts
// the load at a flat 32 000 whatever the actor is wearing. That is the
// instruction the claim names, and it is why this returns rather than falling
// through.
func (w *World) loadOf(i int) int32 {
	return CarriedLoad(w.wornWeight(i), w.containerWeight(i))
}

func (w *World) instanceWeight(item ItemInstance) int32 {
	if item.WeightPresent {
		return int32(item.Weight)
	}
	return w.itemWeightOf(item.Code)
}

// CarriedLoad is the load law itself, over the two sums an actor's own state
// gives it, and it is the ONE statement of that law in this tree.
//
// IT IS EXPORTED FOR A CALLER WITH NO WORLD. A party member between missions
// has a worn set and a carried list and no entity anywhere, and his sheet has
// to state the same load his sheet inside a mission states. The alternative was
// a second copy of the arithmetic in the tier that draws him, which is how the
// two would come to disagree about a saturated container.
//
// It takes the two sums rather than an actor because computing them is the
// caller's own job: inside a world they are the container and the equipment
// arrays, and outside one they are whatever the caller holds.
func CarriedLoad(worn, container int32) int32 {
	if container >= carrySaturation {
		return carrySaturatedLoad
	}
	return worn + container/carryHalving
}

// recomputeLoad writes entity i's load. It is the ONE writer of Entity.Load,
// so no producer can move a container and leave the field naming the contents
// it had before.
func (w *World) recomputeLoad(i int) {
	if i < 0 || i >= len(w.entities) {
		return
	}
	// Declarations and reconstruction do not enact an item operation on an
	// imported actor. Actual mutations use finishLoadMutation.
	if w.entities[i].ActorLoad.Present {
		return
	}
	w.entities[i].Load = w.loadOf(i)
	w.entities[i].HumanMovement = HumanMovement{}
}

// recomputeLoads writes every entity's load. It is what a whole-world act calls
// — construction, a weight-table declaration — where a per-entity producer calls
// recomputeLoad instead.
func (w *World) recomputeLoads() {
	for i := range w.entities {
		w.recomputeLoad(i)
	}
}

// overloadedSpeed is HERO-SPEED-008's own penalty and nothing else:
//
//	if load >= capacity : speed = max(speed - load/capacity, 6)
//
// IT IS THE ONLY CONSUMER OF THE LOAD. ITEM-LOAD-005 states that in as many
// words, and nothing in this build refuses a pick-up, a purchase, an equip or a
// sack transfer on weight.
//
// BELOW CAPACITY IT DOES NOTHING AT ALL — not a small penalty, not a scaled
// one. The gate is `load >= capacity` and the arm is skipped entirely below it.
//
// TWO GUARDS ARE THIS PACKAGE'S AND NOT THE CLAIM'S, and neither changes a
// value the claim covers:
//
//   - A base at or below zero is returned untouched. Speed is this package's
//     own "has a rate at all" predicate (rated, world.go), and an entity that
//     moves at no rate must not be turned into one that moves at six by the
//     floor. No actor the claim describes has a speed of zero.
//   - A capacity at or below zero is returned untouched. Capacity is `Body x 10
//   - 1` and is never zero for a character any recompute ran for, so a zero
//     here is an entity no spawn path stated a capacity for, a prop —
//     and it also removes the division.
func overloadedSpeed(base, load, capacity int32) int32 {
	if base <= 0 || capacity <= overloadDivisorZ || load < capacity {
		return base
	}
	if v := base - load/capacity; v > overloadFloor {
		return v
	}
	return overloadFloor
}
