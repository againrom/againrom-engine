package sim

// AutoCastSetting is one Player's automatic-casting preference byte: a level
// mask and three target bits (R2-ENGINE-345, R2-ENGINE-360, R2-ENGINE-361).
// Both games read this one setting (DIV-3006).
type AutoCastSetting uint8

const (
	AutoCastOwn     AutoCastSetting = 1
	AutoCastAllies  AutoCastSetting = 2
	AutoCastNeutral AutoCastSetting = 4
	autoCastTargets                 = AutoCastOwn | AutoCastAllies | AutoCastNeutral

	AutoCastMinimum AutoCastSetting = 0x08
	AutoCastAverage AutoCastSetting = 0x18
	AutoCastMaximum AutoCastSetting = 0x38
	autoCastLevels                  = AutoCastMaximum

	// DefaultAutoCast is the preference constructor's value: Average, Own
	// (R2-ENGINE-345). A second-game Player without a stored setting reads it
	// (DIV-3007).
	DefaultAutoCast = AutoCastAverage | AutoCastOwn
)

// PlayerParameterAutoCast carries a whole setting byte in the command value.
const PlayerParameterAutoCast PlayerParameter = 5

// autoCastLevelList lists the three levels in order; AutoCastLevel numbers them
// 1 to 3 after Off, 0.
var autoCastLevelList = [...]AutoCastSetting{AutoCastMinimum, AutoCastAverage, AutoCastMaximum}

// Level returns the level mask, or zero when the byte names none of the three.
func (s AutoCastSetting) Level() AutoCastSetting {
	switch l := s & autoCastLevels; l {
	case AutoCastMinimum, AutoCastAverage, AutoCastMaximum:
		return l
	}
	return 0
}

// Targets returns the three target bits.
func (s AutoCastSetting) Targets() AutoCastSetting { return s & autoCastTargets }

// LevelNumber is 0 for Off and 1, 2 or 3 for Minimum, Average or Maximum.
func (s AutoCastSetting) LevelNumber() int {
	for i, l := range autoCastLevelList {
		if s.Level() == l {
			return i + 1
		}
	}
	return 0
}

// AutoCastLevel is the level LevelNumber numbers; 0 and any other number is
// Off.
func AutoCastLevel(number int) AutoCastSetting {
	if number >= 1 && number <= len(autoCastLevelList) {
		return autoCastLevelList[number-1]
	}
	return 0
}

// WithLevel keeps the target bits and replaces the level; a zero level is
// Off.
func (s AutoCastSetting) WithLevel(level AutoCastSetting) AutoCastSetting {
	return s.Targets() | level.Level()
}

// AutoCast returns the Player's setting. A Player without a stored setting
// reads Off with the Own target bit in the first game (DIV-3006), so a level
// chosen later keeps the constructor's target, and the constructor's
// DefaultAutoCast in the second (DIV-3007). Off casts nothing by level.
func (w *World) AutoCast(player uint32) AutoCastSetting {
	if player < relationSlots {
		if s := w.autoCastSetting[player]; s.Level() != 0 {
			return s
		}
	}
	return autoCastDefaultBodies[w.game.autoCastDefault]()
}

// autoCastDefaultRule selects the setting a Player without a stored one reads.
type autoCastDefaultRule uint8

const (
	autoCastDefaultTargets autoCastDefaultRule = iota
	autoCastDefaultFull
)

// autoCastDefaultBodies holds the default each autoCastDefaultRule reads.
var autoCastDefaultBodies = [...]func() AutoCastSetting{
	autoCastDefaultTargets: func() AutoCastSetting { return DefaultAutoCast.Targets() },
	autoCastDefaultFull:    func() AutoCastSetting { return DefaultAutoCast },
}

// AutoCastStored reports whether the Player carries a stored setting.
func (w *World) AutoCastStored(player uint32) bool {
	return player < relationSlots && w.autoCastSetting[player].Level() != 0
}

// SetAutoCast stores a setting. A byte without a level clears the stored
// setting, so the Player reads its game's default again: Off in the first
// game. A byte carrying bits outside the level and target masks, or a level
// mask that is none of the three, is refused.
func (w *World) SetAutoCast(player uint32, s AutoCastSetting) bool {
	if player >= relationSlots || s&^(autoCastLevels|autoCastTargets) != 0 || s&autoCastLevels != 0 && s.Level() == 0 {
		return false
	}
	if s.Level() == 0 {
		s = 0
	}
	w.autoCastSetting[player] = s
	return true
}

// autoCastTiers lists, per level bit, the first-game arms the level adds:
// Heal and Shield (R2-ENGINE-346); Bless and Haste; the four protections and
// Invisibility (R2-ENGINE-347). Second-game rows reach them through their arm.
var autoCastTiers = [...]struct {
	bit  AutoCastSetting
	mana int32
	arms []uint16
}{
	{0x08, 5, autoCastArmsAt(1)},
	{0x10, 10, autoCastArmsAt(2)},
	{0x20, 40, autoCastArmsAt(3)},
}

// AutoCastLevelOfArm is the level whose list holds arm, or zero.
func AutoCastLevelOfArm(arm uint16) AutoCastSetting {
	if l := spellArmOf(arm).autoCast.level; l > 0 {
		return autoCastLevelList[l-1]
	}
	return 0
}

// AutoCastLevelOfSpell is AutoCastLevelOfArm for a table row id.
func (w *World) AutoCastLevelOfSpell(id uint16) AutoCastSetting {
	return AutoCastLevelOfArm(w.spellArm(id))
}

type supportCastPolicy struct {
	minimumMana  int32
	requiresHeal bool
	idleOnly     bool
	// pick: candidates are Heal's area list, and support past Heal goes to one
	// drawn candidate (AI-785, AI-786, R2-ENGINE-1244).
	pick supportPickRule
	// hold: the idle-only policy's hold (supportHolds).
	hold supportHoldRule
	// yieldsToWithdrawal: a unit whose withdrawal step walks or casts it now
	// prepares no heal, since the step overwrites it on the pass (R2-ENGINE-1442).
	yieldsToWithdrawal bool
	groundSlotOnly     bool // AI-866, R2-ENGINE-2003
	shield             supportShield
}

// supportHeld is the idle-only policy's hold, by the rule the policy holds.
func (w *World) supportHeld(ci int, policy supportCastPolicy) bool {
	return supportHolds[policy.hold](w, ci)
}

// supportHoldRule is what keeps the idle-only support policy from running.
type supportHoldRule uint8

const (
	supportHeldByOwnOrders supportHoldRule = iota
	supportHeldInCombat
)

// supportHolds: the caster's own cast and attack target (R2-ENGINE-1420
// tests only the member's own target), or the inCombat predicate, where a
// living hostile holding the caster as its unit victim holds it too.
var supportHolds = [...]func(w *World, ci int) bool{
	supportHeldByOwnOrders: func(w *World, ci int) bool {
		return w.castHolds(ci, waitIdleHeal) || w.entities[ci].HasAttackTarget
	},
	supportHeldInCombat: (*World).inCombat,
}

// supportShield is where the sequence tries Shield at the caster.
type supportShield uint8

const (
	shieldBeforeCandidates supportShield = iota
	shieldAfterCandidateHeal
	shieldNone
)

// computerSupportSetting admits every non-hostile owner; DIV-4756 narrows it.
const computerSupportSetting = AutoCastMinimum | AutoCastOwn | AutoCastAllies | AutoCastNeutral

// computerSupport is the profile's computer policy for a computer mage.
func (w *World) computerSupport(e Entity) (supportCastPolicy, bool) {
	p := w.game.computerSupport
	return p, p.minimumMana != 0 && w.computerControlledUnit(e)
}

// supportCast shares the self-support prefix and book admission. Computer
// policy is profile data (R2-ENGINE-1244, AI-785).
func (w *World) supportCast(ci int) {
	caster := w.entities[ci]
	s := w.AutoCast(caster.Owner)
	policy := supportCastPolicy{minimumMana: autoCastTiers[0].mana}
	if computer, ok := w.computerSupport(caster); ok {
		policy, s = computer, computerSupportSetting
	} else if caster.Owner != SelfSlot {
		return
	}
	if caster.Mana < policy.minimumMana || s&autoCastTiers[0].bit == 0 || policy.idleOnly && w.supportHeld(ci, policy) {
		return
	}
	if policy.yieldsToWithdrawal && w.withdrawalHolds(ci) {
		return
	}
	heal := autoCastRoleArm(autoCastHealer)
	var healRule SpellRule
	if policy.requiresHeal {
		id := w.spellIDOfArm(heal)
		rule, ok := w.bookSpell(caster, uint32(id))
		if !ok || !knowsSpell(caster, uint32(id)) {
			return
		}
		healRule = rule
	}
	if caster.HP != caster.MaxHP && w.autoCastAt(ci, ci, heal, false) {
		return
	}
	if policy.shield == shieldBeforeCandidates && w.autoCastAt(ci, ci, autoCastRoleArm(autoCastShielder), true) {
		return
	}
	pick := supportPicks[policy.pick]
	candidates := pick.candidates(w, ci, s, healRule, policy)
	for _, ti := range candidates {
		if t := w.entities[ti]; t.MaxHP > 0 && t.HP/t.MaxHP < 1 {
			if w.autoCastAt(ci, ti, heal, false) {
				return
			}
			break
		}
	}
	pick.finish(w, ci, s, candidates, policy)
}

// supportPickRule selects how support past Heal chooses its target: every
// candidate in turn, or one drawn candidate.
type supportPickRule uint8

const (
	supportPicksAll supportPickRule = iota
	supportPicksDrawn
)

// supportPick holds a supportPickRule's candidate list and its cast past Heal.
type supportPick struct {
	candidates func(w *World, ci int, s AutoCastSetting, heal SpellRule, policy supportCastPolicy) []int
	finish     func(w *World, ci int, s AutoCastSetting, candidates []int, policy supportCastPolicy)
}

// supportPicks holds the body each supportPickRule runs.
var supportPicks = [...]supportPick{
	supportPicksAll: {
		candidates: func(w *World, ci int, s AutoCastSetting, _ SpellRule, _ supportCastPolicy) []int {
			return w.autoCastCandidates(ci, s, false)
		},
		finish: func(w *World, ci int, s AutoCastSetting, candidates []int, _ supportCastPolicy) {
			for _, l := range autoCastTiers[1:] {
				if s&l.bit == 0 || w.entities[ci].Mana < l.mana {
					return
				}
				for _, ti := range candidates {
					for _, arm := range l.arms {
						if spellArmOf(arm).autoCast.role == autoCastHider && (ti == ci || w.autoCastNeutral(ci, ti)) {
							continue
						}
						if w.autoCastAt(ci, ti, arm, true) {
							return
						}
					}
				}
			}
			w.autoCastAt(ci, ci, autoCastRoleArm(autoCastHider), true)
		},
	},
	supportPicksDrawn: {
		candidates: func(w *World, ci int, s AutoCastSetting, heal SpellRule, policy supportCastPolicy) []int {
			return w.areaListOrder(ci, w.autoCastCandidates(ci, s, heal.HealHostile), int64(heal.MaxRange), policy.groundSlotOnly)
		},
		finish: func(w *World, ci int, _ AutoCastSetting, candidates []int, policy supportCastPolicy) {
			if policy.shield == shieldAfterCandidateHeal && w.autoCastAt(ci, ci, autoCastRoleArm(autoCastShielder), true) {
				return
			}
			w.supportDraw(ci, candidates, policy)
		},
	},
}

// autoCastAt attempts arm on the actor at ti. A buff whose effect stands on
// the target is not attempted.
func (w *World) autoCastAt(ci, ti int, arm uint16, buff bool) bool {
	rule, ok := w.autoCastAdmits(ci, ti, arm)
	if !ok {
		return false
	}
	id := rule.ID
	if _, standing := effectIndex(w.attached, w.entities[ti].ID, id); buff && standing {
		return false
	}
	if w.unsafeUnorderedArea(ci, ti, rule) {
		return false
	}
	return w.castOrder(ci, Cast(w.entities[ci].ID, w.entities[ti].ID, SpellID(id)))
}

// autoCastAdmits is autoCastAt's book, cost and target admission of arm.
func (w *World) autoCastAdmits(ci, ti int, arm uint16) (SpellRule, bool) {
	id := w.spellIDOfArm(arm)
	rule, ok := w.bookSpell(w.entities[ci], uint32(id))
	if !ok || !rule.runsArm(arm) || !knowsSpell(w.entities[ci], uint32(id)) || w.entities[ci].Mana < int32(rule.ManaCost) {
		return SpellRule{}, false
	}
	return rule, w.autoCastArmAdmits(arm, w.entities[ti])
}

// supportDraw is the computer's support past Heal: one drawn candidate of the
// caster's owner gets Heal when hurt, and at full mana one drawn Defensive
// book row unless its effect stands (AI-786, R2-ENGINE-1244). Nothing kept
// casts nothing (DIV-4951).
func (w *World) supportDraw(ci int, candidates []int, policy supportCastPolicy) {
	if len(candidates) == 0 {
		return
	}
	ti := candidates[w.rng.aiRange(int32(len(candidates)))]
	caster, t := w.entities[ci], w.entities[ti]
	heal, shield := autoCastRoleArm(autoCastHealer), autoCastRoleArm(autoCastShielder)
	if t.Owner != caster.Owner || t.HP != t.MaxHP && w.autoCastAt(ci, ti, heal, false) || caster.Mana != caster.MaxMana {
		return
	}
	var kept []uint16
	for _, row := range w.spells {
		arm := row.arm()
		if arm == heal || arm == shield && policy.shield != shieldNone || !autoCastable(row) {
			continue
		}
		if rule, ok := w.autoCastAdmits(ci, ti, arm); ok && rule.ID == row.ID && rule.Defensive {
			kept = append(kept, arm)
		}
	}
	if len(kept) > 0 {
		w.autoCastAt(ci, ti, kept[w.rng.aiRange(int32(len(kept)))], true)
	}
}

// areaListOrder keeps the candidates within reach cells of the caster in the
// order Heal's area list visits them: the caster's cell, then each ring
// outward (AI-336, AI-866, R2-ENGINE-1245).
func (w *World) areaListOrder(ci int, candidates []int, reach int64, groundSlotOnly bool) []int {
	centre := cellOf(&w.entities[ci])
	at := map[cell][]int{}
	for _, ti := range candidates {
		if groundSlotOnly && lawDomain(w.entities[ti].Domain) == 3 {
			continue
		}
		if c := cellOf(&w.entities[ti]); centre.chebyshevTo(c) <= reach {
			at[c] = append(at[c], ti)
		}
	}
	out := at[centre]
	walkCellRings(centre, reach, cellRingWalk{}, func(x, y int64) bool {
		out = append(out, at[cell{x: int32(x), y: int32(y)}]...)
		return false
	})
	return out
}

func (w *World) autoCastNeutral(ci, ti int) bool {
	c, t := w.entities[ci], w.entities[ti]
	return c.Owner != t.Owner && !w.relations.Locked(c.Owner, t.Owner) && !w.hostileTo(&c, &t)
}

type supportAffinityRule uint8

const (
	supportAffinityOwner supportAffinityRule = iota
	supportAffinityRelation
)

// supportAffinities: the second game reads the relation byte, whose map
// diagonal 0x12 holds the ally bit (R2-ENGINE-1793).
var supportAffinities = [...]func(w *World, c, t *Entity, s AutoCastSetting) bool{
	supportAffinityOwner: func(w *World, c, t *Entity, s AutoCastSetting) bool {
		switch {
		case t.Owner == c.Owner:
			return s&AutoCastOwn != 0
		case w.relations.Locked(c.Owner, t.Owner):
			return s&AutoCastAllies != 0
		}
		return s&AutoCastNeutral != 0
	},
	supportAffinityRelation: func(w *World, c, t *Entity, s AutoCastSetting) bool {
		rel := w.relations.Byte(c.Owner, t.Owner)
		return rel&(relationHostile|relationLocked) == 0 && s&AutoCastNeutral != 0 ||
			rel&relationLocked != 0 && s&AutoCastAllies != 0 || t.Owner == c.Owner && s&AutoCastOwn != 0
	},
}

// autoCastCandidates lists living or raisable, non-hostile actors the
// automatic choice sees (autoChoiceSees) and the target bits admit, in entity order. The caster is always admitted
// (R2-ENGINE-360). Range is the cast's own admission. healsHostile keeps hostiles
// for a computer's Heal row that heals them, unless DIV-4756 refuses them.
func (w *World) autoCastCandidates(ci int, s AutoCastSetting, healsHostile bool) []int {
	caster := w.entities[ci]
	var sight autoChoiceSight
	var out []int
	for i := range w.entities {
		t := w.entities[i]
		if !autoCastTargetAdmitted(t) {
			continue
		}
		// Support policy keeps living or raisable actors with health.
		if !t.Alive() && !(t.OrdinaryTargetable() && t.restorativeTargetable()) || t.MaxHP <= 0 {
			continue
		}
		// Support policy exempts self from target bits, hostility and sight.
		if i != ci {
			switch {
			case w.hostileTo(&caster, &t):
				if !healsHostile {
					continue
				}
			case !supportAffinities[w.game.supportAffinity](w, &caster, &t, s):
				continue
			}
			if w.computerHealRefuses(caster, t) || !w.autoChoiceSees(ci, i, &sight) {
				continue
			}
		}
		out = append(out, i)
	}
	return out
}
