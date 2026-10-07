package data

import "math"

// Character generation's own arithmetic: what a statistic costs, what the budget
// is, and which spreads the screen could have produced.
//
// IT IS THE RULES AND NOT THE SCREEN. Nothing here draws a row, spends a point
// at run time or holds a pool that anything mutates; a generation screen is a
// consumer of this file and does not exist yet. What the file is for is that a
// hero this tree hands the player can be CHECKED — a build nobody could have
// generated is a divergence whether or not anyone noticed, and the check is
// worth more than any particular build, because the build will change and the
// constraint will not.
//
// EVERYTHING HERE IS A FUNCTION OF ITS ARGUMENTS. No clock, no generator, no
// global, no file.

// The two bounds a click refuses past. They are the operands of the two
// comparisons the handlers make, and they bound the SCREEN — a `+` refuses at
// 45 and a `-` refuses at 15, so a generated statistic lies in [15, 45].
//
// StatCeiling IS NOT StatCap. This 45 is character generation's click bound; the
// 50 beside it in hero.go is the recompute's own clamp, applied afterwards to
// every hero forever and reachable past 50 only through an effect. They come
// from different routines and answer different questions, and under the budget
// below neither 45 nor 50 is reachable at generation at all — the most any
// statistic can be bought to is 43.
const (
	StatFloor   int32 = 15
	StatCeiling int32 = 45
)

// The budget and the pool, which are ONE NUMBER SEEN TWICE. The accepting side
// loads 140 and subtracts the cost of each of the four submitted statistics; the
// screen starts every statistic at 25 with 100 in hand, and `4*T(25) + 100` is
// that same 140. So a client that spends its pool and a server that tests the
// budget cannot disagree, and neither reads a class, a race or a registry key.
//
// ChargenPool is what the counter on the screen shows at the start, and the
// counter counts what REMAINS rather than what has been spent.
const (
	ChargenBudget int32 = 140
	ChargenPool   int32 = 100
)

// The three constants of the cost curve: the base of the exponential, the factor
// in front of it and the rounding term added before truncation. The base is an
// inline immediate and the other two are named memory operands; the truncation
// is the same one the derive uses.
const (
	costBase   = 1.15
	costFactor = 0.349
	costRound  = 0.5
)

// PointCost is what one statistic standing at n costs, IN TOTAL.
//
// It is cumulative and not a price per step, which is the whole of why the
// budget can be tested by four subtractions on the accepting side without any
// history of how the four were reached. A step's price is a first difference of
// it — see StepCost — and the curve escalates hard: a step in the low twenties
// costs one point and the step into 45 costs twenty-two.
//
// The `+ 0.5` before truncation is a round-half-up and is part of the function
// rather than a courtesy of ours: the image adds that constant and then calls
// the truncating conversion, so a consumer that rounded some other way would
// disagree with the accepting side about which spreads are legal.
func PointCost(n int32) int32 {
	return ftol(costFactor*math.Pow(costBase, float64(n-1)) + costRound)
}

// StepCost is what a `+` click charges to raise a statistic from v to v+1, and
// therefore also what a `-` click refunds coming back down from v+1.
//
// ONE FUNCTION SERVES BOTH DIRECTIONS, and that is not a simplification: the
// charge is `T(v+1) - T(v)` and the refund is `T(v) - T(v-1)`, so the refund
// taken at v is the charge that reached v, exactly, for every v. The symmetry is
// algebraic — a consequence of the cost being cumulative — rather than something
// the handlers were written to preserve, which is why one function can state it
// without either handler being modelled.
func StepCost(v int32) int32 { return PointCost(v+1) - PointCost(v) }

// Spread is an assignment of the four primary statistics: the INPUT character
// generation produces and the derive consumes.
//
// It is a value type of four builtins, so it is copied by assignment and
// compared with ==. It names its fields rather than ordering them, because the
// storage order the original writes them in crosses two of them and a positional
// four-tuple here would be a second place for that crossing to be got wrong.
type Spread struct {
	Body, Reaction, Mind, Spirit int32
}

// ChargenSpread is where every character starts before a point is spent: the
// initialiser's own immediate, four times over.
//
// It is a FUNCTION rather than a package variable for the reason the party's own
// seam is: a variable is writable from anywhere, and "generation always starts
// here" would then be true by mutation rather than by design.
func ChargenSpread() Spread {
	return Spread{Body: ChargenStat, Reaction: ChargenStat, Mind: ChargenStat, Spirit: ChargenStat}
}

// Cost is what this spread spends of the budget: the sum of the four totals, and
// not a sum of steps.
func (s Spread) Cost() int32 {
	return PointCost(s.Body) + PointCost(s.Reaction) + PointCost(s.Mind) + PointCost(s.Spirit)
}

// Remaining is what the screen's counter would show for this spread — the budget
// less what it costs. It goes NEGATIVE for a spread over the budget, which is
// the same sign test the accepting side makes, and it is not clamped: a clamp
// here would erase the only distinction between "spent it all" and "cannot be
// generated".
func (s Spread) Remaining() int32 { return ChargenBudget - s.Cost() }

// Legal reports whether character generation could have produced this spread.
//
// TWO BOUNDS, BOTH REQUIRED, AND NEITHER IMPLIES THE OTHER. Every statistic must
// lie in [15, 45], which is the pair of click refusals; and the four costs
// together must not exceed the budget, which is the accepting side's test. The
// range alone would admit 45/45/45/45, which the budget refuses many times over;
// the budget alone would admit a statistic at 0, which no click can reach.
//
// The four statistics are INDEPENDENT in it: no term reads another, exactly as
// neither handler nor the accepting side does. So this is a complete predicate
// over the whole space and not a sampler of the spreads anyone happened to try.
func (s Spread) Legal() bool {
	for _, v := range [...]int32{s.Body, s.Reaction, s.Mind, s.Spirit} {
		if v < StatFloor || v > StatCeiling {
			return false
		}
	}
	return s.Cost() <= ChargenBudget
}

// SkillName is what the shipped definition table calls the skill at a slot.
//
// The file gives each of the five shared slots a warrior name and a mage name in
// ONE string — `Skill.Blade (Fire)` — and this tree has no class axis, so the
// warrior half is what it can honestly print. A slot outside 0..5 names nothing.
// Slot 0 is `General`, and it is named here even though the character sheet does
// not show it: this function answers what the table calls a slot, not what a
// screen chooses to draw.
func SkillName(slot int32) string {
	switch slot {
	case SkillGeneral:
		return "General"
	case SkillBlade:
		return "Blade"
	case SkillAxe:
		return "Axe"
	case SkillBludgen:
		return "Bludgen"
	case SkillPike:
		return "Pike"
	case SkillShoot:
		return "Shooting"
	}
	return ""
}

// MageSkillName is the OTHER half of the same shipped string SkillName
// reads — the parenthesised name beside the warrior one, `(Fire)` beside
// `Blade` — for the five slots a class renames: Fire, Water, Air, Earth,
// Astral for slots 1 to 5. Slot 0 has no such second half — the character
// sheet does not show it for either class (hero.go's own doc on SkillSlots)
// — and a slot outside 1..5 names nothing, the same refusal SkillName gives
// outside its own wider 0..5.
func MageSkillName(slot int32) string {
	switch slot {
	case SkillBlade:
		return "Fire"
	case SkillAxe:
		return "Water"
	case SkillBludgen:
		return "Air"
	case SkillPike:
		return "Earth"
	case SkillShoot:
		return "Astral"
	}
	return ""
}

// SkillNames is the five shared slots' names, in slot order 1 to 5 — the
// warrior set (SkillName) when mage is false, the mage set (MageSkillName)
// when it is true. Slot 0 is not among them, for MageSkillName's own reason:
// it is the slot neither character sheet shows, so the screen's skill row
// has no zeroth option to offer either.
func SkillNames(mage bool) []string {
	names := make([]string, 0, SkillSlots-1)
	for slot := int32(SkillBlade); slot < SkillSlots; slot++ {
		if mage {
			names = append(names, MageSkillName(slot))
		} else {
			names = append(names, SkillName(slot))
		}
	}
	return names
}
