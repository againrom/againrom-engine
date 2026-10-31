package sim

import (
	"cmp"
	"slices"
)

// secondSpellArms maps a second-game spell id to the first-game arm of the
// same spell (R2-ENGINE-019 names the rows, R2-ENGINE-024 groups them by
// arm). Blizzard runs arm 21's sampled staged program: 32 pulses three area
// updates apart, each one cell drawn x then y as RNG(5)-2 and admitted only
// inside 8..dimension-9 (R2-ENGINE-391, R2-ENGINE-392). Ice Missile and
// Diamond Dust reach the blank-effect default as Fire Arrow does and run its
// damage block alone (R2-ENGINE-024). Summon runs armSummon.
var secondSpellArms = [...]uint16{
	1: 1, 2: 2, 3: 3, 4: 5, 5: ArmNone, 6: 8, 7: 21, 8: 10, 9: 9, 10: 13,
	11: 14, 12: 15, 13: 16, 14: 17, 15: 12, 16: ArmNone, 17: 19, 18: 20, 19: 22,
	20: 23, 21: 24, 22: 25, 23: 26, 24: 6, 25: armSummon, 26: 11, 27: 18, 28: 27,
	29: 28,
}

// armSummon is the second game's Summon arm. The first game has no such
// spell, so no first-game id names it.
const armSummon uint16 = 0xfffe

// SecondGameSpellArm is the arm a second-game row id runs; an id past the
// published table runs none.
func SecondGameSpellArm(id uint16) uint16 {
	if int(id) < len(secondSpellArms) && id != 0 {
		return secondSpellArms[id]
	}
	return ArmNone
}

// AssignSecondGameArms marks every row of a second-game table with its arm.
func AssignSecondGameArms(rules []SpellRule) {
	for i := range rules {
		rules[i].Arm, rules[i].Second = SecondGameSpellArm(rules[i].ID), true
	}
}

// secondDrawSpan is the span of the second game's 1..n draw: a spread of 0
// still draws 1 (R2-ENGINE-026).
func secondDrawSpan(spread int64) int32 {
	return int32(max(spread, 1) - 1)
}

// secondDrainAmount is the second game's Drain Life roll before the cap:
// (base + U[1, spread]) cut by the target's Astral resistance
// (R2-ENGINE-026). draw is the 0-based draw of secondDrawSpan.
func secondDrainAmount(base int64, astral, draw int32) int64 {
	p := int64(min(max(astral, 0), 100))
	return (base + 1 + int64(draw)) * (100 - p) / 100
}

// secondBlankCurse is the second game's Stone Curse whose Effects string
// parses to no kind. Its cast attaches a blank timed effect under the row's
// own id (R2-ENGINE-453): kind none, magnitude 0. The record alone holds its
// target, by the first game's Stone Curse hold (stoneCursed; R2-ENGINE-454).
func secondBlankCurse(rule SpellRule) bool {
	return rule.Second && rule.arm() == 20 && rule.EffectKind == EffectNone
}

// blankEffectRow reports whether the table row id attaches a blank effect.
func (w *World) blankEffectRow(id uint16) bool { return blankEffectRowOf(w.spells, id) }

// BlankEffectRow is blankEffectRow for the SAV importer.
func (w *World) BlankEffectRow(id uint16) bool { return w.blankEffectRow(id) }

func blankEffectRowOf(spells []SpellRule, id uint16) bool {
	for _, r := range spells {
		if r.ID == id {
			return secondBlankCurse(r)
		}
	}
	return false
}

// secondCurseCeiling is the top of the second game's Stone Curse duration
// draw: ((P*240*m)/100) * (100 - earth)/100, with m 2 for a spell cast and 1
// for an item cast and earth the target's Earth protection held to 0..100
// (R2-ENGINE-025). The cast draws 0..ceiling; a draw of 0 attaches nothing.
func secondCurseCeiling(power, earth int32, item bool) uint16 {
	m := int64(2)
	if item {
		m = 1
	}
	p := int64(min(max(earth, 0), 100))
	d := int64(max(power, 0)) * 240 * m / 100 * (100 - p) / 100
	return uint16(min(d, durationTickCeiling))
}

// keepsLargerOnRepeat is a row whose repeat cast keeps the effect already on
// its target and takes the larger magnitude and the larger duration
// (R2-ENGINE-454). Every other row replaces the record.
func keepsLargerOnRepeat(rule SpellRule) bool {
	return rule.Second && rule.arm() == 20
}

// SpellCreature is one actor a spell creates, built by the loader from the
// Units row Name names: the actor, its pack and what it wears.
type SpellCreature struct {
	Name  string
	Actor Entity
	Pack  []ItemStack
	Worn  [EquipSlots]ItemInstance
}

// DeclareSpellCreatures installs the actors a Summon cast can create. A
// World with none refuses the cast before it is paid.
func (w *World) DeclareSpellCreatures(creatures []SpellCreature) {
	w.spellCreatures = nil
	for _, c := range creatures {
		c.Pack = cloneStacks(c.Pack)
		w.spellCreatures = append(w.spellCreatures, c)
	}
}

// SpellCreatures is the declared creature list, for a rebuild to carry.
func (w *World) SpellCreatures() []SpellCreature {
	out := make([]SpellCreature, len(w.spellCreatures))
	for i, c := range w.spellCreatures {
		c.Pack = cloneStacks(c.Pack)
		out[i] = c
	}
	return out
}

func (w *World) spellCreature(name string) (SpellCreature, bool) {
	for _, c := range w.spellCreatures {
		if c.Name == name {
			return c, true
		}
	}
	return SpellCreature{}, false
}

// SummonCreatureNames are the Units row names a second-game Summon builds
// from: kind roll 1, 2 and 3, each bare at level 1 and with `.` and the level
// digit above it (R2-ENGINE-455, R2-ASSET-107).
var SummonCreatureNames = [...]string{"Squirrel", "Turtle", "Foot_Animated"}

// summonSkill is the Astral skill slot the Summon level reads.
const summonSkill = 5

// summonRadius is the radius of the placement search around the caster.
const summonRadius = 6

// SummonCreatureName is the Units row a Summon by a caster of the given Astral
// skill builds for kind roll k in 1..3: level clamp(astral/25 + 1, 1, 4)
// (R2-ENGINE-026), the level appended above 1, so `.5` is never built.
func SummonCreatureName(astral, k int32) string {
	level := min(max(astral/25+1, 1), 4)
	name := SummonCreatureNames[k-1]
	if level > 1 {
		name += "." + string(rune('0'+level))
	}
	return name
}

// castSummon is the second game's Summon arm (R2-ENGINE-455): it rolls the
// kind 1..3, builds the creature its level names for the caster's Player and
// places it alone in a new group by the positioning search around the
// caster's cell. A failed placement aborts the cast; the cost is already paid
// at the cast's entry (R2-ENGINE-544). The creature carries the caster's key,
// so the world pass caps it (summonCapPass); it is never a reused dead unit
// and carries no summoned mark (DIV-4152).
func (w *World) castSummon(ci int) bool {
	caster := w.entities[ci]
	k := 1 + w.rng.uniform(secondDrawSpan(int64(len(SummonCreatureNames))))
	c, ok := w.spellCreature(SummonCreatureName(caster.Skill[summonSkill], k))
	if !ok {
		return false
	}
	actor := c.Actor
	actor.Owner, actor.summonKey = caster.Owner, summonKeyOf(caster.ID)
	id, err := w.createActor(actor, c.Pack, c.Worn, caster.X, caster.Y, secondPlacement(caster.X, caster.Y, summonRadius), true)
	if err != nil {
		return false
	}
	w.castObservation.recordCreatedActor(id, c.Name)
	return true
}

// summonKeyOf is the key a caster gives its creatures: the id plus one, so a
// real caster never yields 0, which means no key (R2-ENGINE-455 stores the
// caster's own nonzero word, and the original reads 0 as no key). The map is
// increasing, so the pass's ascending key order is ascending caster id.
func summonKeyOf(caster EntityID) EntityID { return caster + 1 }

// summonCapSpan is the span of the world pass's draw per key: 1..100.
const summonCapSpan = 100

// summonedDeathHP is the health a death stores on a keyed actor
// (R2-ENGINE-776); it lies past the corpse ladder's last rung, so the body
// leaves the world when its dwell ends.
const summonedDeathHP = -10000

// summonCapPass is the second game's world pass cap on summoned actors
// (R2-ENGINE-776): count the keyed actors per key, draw r in 1..100 once per
// key in ascending key order and mark the key when r < n, then give the first
// keyed actor of each marked key, in ascending id, the death. So at most one
// actor per key leaves per pass, and a key with one actor never loses it.
func (w *World) summonCapPass() {
	type keyCount struct {
		key    EntityID
		n      int
		marked bool
	}
	var counts []keyCount
	find := func(key EntityID) (int, bool) {
		return slices.BinarySearchFunc(counts, key, func(c keyCount, k EntityID) int { return cmp.Compare(c.key, k) })
	}
	for i := range w.entities {
		if key := w.entities[i].summonKey; key != 0 {
			j, found := find(key)
			if !found {
				counts = slices.Insert(counts, j, keyCount{key: key})
			}
			counts[j].n++
		}
	}
	for j := range counts {
		counts[j].marked = int(1+w.rng.uniform(secondDrawSpan(summonCapSpan))) < counts[j].n
	}
	for i := range w.entities {
		if key := w.entities[i].summonKey; key != 0 {
			if j, _ := find(key); counts[j].marked {
				counts[j].marked = false
				w.entities[i].setCurrentHealth(summonedDeathHP)
				w.clearFelled(i)
			}
		}
	}
}

// secondPlacement is the second game's positioning search around (x, y)
// (R2-ENGINE-531): with half = r/2, r*r/2 + 2 tries each draw y then x as
// the centre less half plus a draw of 0..r; then, for r above 0, a scan of x
// from x-half to x+half, y inner. Every candidate is held to 8..dimension-9
// and passes when the actor's footprint is free there. The original also
// skips a candidate whose byte value is below 12 while a session flag is set,
// neither read; this search never skips (DIV-2618).
func secondPlacement(x, y, r int32) actorSearch {
	return func(n *World, i int) (int32, int32, bool) {
		half := r / 2
		held := func(cx, cy int32) (int32, int32) {
			return min(max(cx, 8), n.bounds.Width-9), min(max(cy, 8), n.bounds.Height-9)
		}
		for k := int32(0); k <= r*r/2+1; k++ {
			cy := y - half + n.rng.uniform(r)
			cx := x - half + n.rng.uniform(r)
			if px, py := held(cx, cy); n.placeFree(i, px, py) {
				return px, py, true
			}
		}
		if r == 0 {
			return 0, 0, false
		}
		for cx := x - half; cx <= x+half; cx++ {
			for cy := y - half; cy <= y+half; cy++ {
				if px, py := held(cx, cy); n.placeFree(i, px, py) {
					return px, py, true
				}
			}
		}
		return 0, 0, false
	}
}

// SpellRuleLands reports a row this build can land: an applicable row that is
// not a staged area with no stage program. The census reads it; casting
// reaches the same refusal.
func SpellRuleLands(rule SpellRule) bool {
	if !spellApplicable(rule) {
		return false
	}
	return !rule.Area || areaModeFor(rule) != areaModeRing || ringStageCount(rule.arm()) != 0
}

// armSpellID is the row id an attached effect of arm carries. A first-game
// table returns arm itself; a second-game table returns the row running arm,
// or ArmNone, which no effect carries, when no row runs it.
func (w *World) armSpellID(arm uint16) uint16 {
	if len(w.spells) == 0 || !w.spells[0].Second {
		return arm
	}
	for _, r := range w.spells {
		if r.arm() == arm {
			return r.ID
		}
	}
	return ArmNone
}

// HasEffectArm reports whether target carries the lasting effect of the row
// running arm: presentation asks by arm, so a second-game row draws as its
// own effect and never as the first-game spell of the same id.
func (w *World) HasEffectArm(target EntityID, arm uint16) bool {
	return w.hasAttachedSpell(target, w.armSpellID(arm))
}

// SpellArm is the arm the table row id runs; an id no row carries is its own
// arm.
func (w *World) SpellArm(id uint16) uint16 { return w.spellArm(id) }
