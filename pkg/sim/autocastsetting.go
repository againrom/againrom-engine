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
	if w.SecondGame() {
		return DefaultAutoCast
	}
	return DefaultAutoCast.Targets()
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

// autoCastLevelArms lists, per level bit, the first-game arms the level adds:
// Heal and Shield (R2-ENGINE-346); Bless and Haste; the four protections and
// Invisibility (R2-ENGINE-347). Second-game rows reach them through their arm.
var autoCastLevelArms = [...]struct {
	bit  AutoCastSetting
	mana int32
	arms []uint16
}{
	{0x08, 5, []uint16{6, 18}},
	{0x10, 10, []uint16{23, 24}},
	{0x20, 40, []uint16{16, 22, 5, 10, 15}},
}

// AutoCastLevelOfArm is the lowest level whose list holds arm, or zero.
func AutoCastLevelOfArm(arm uint16) AutoCastSetting {
	for i, l := range autoCastLevelArms {
		for _, a := range l.arms {
			if a == arm {
				return autoCastLevelList[i]
			}
		}
	}
	return 0
}

// AutoCastLevelOfSpell is AutoCastLevelOfArm for a table row id.
func (w *World) AutoCastLevelOfSpell(id uint16) AutoCastSetting {
	return AutoCastLevelOfArm(w.spellArm(id))
}

// AutoCastCasts reports whether the setting's level lists the row id.
func (w *World) AutoCastCasts(s AutoCastSetting, id uint16) bool {
	l := w.AutoCastLevelOfSpell(id)
	return l != 0 && s.Level()&l == l
}

const (
	autoCastHeal         = 6
	autoCastShield       = 18
	autoCastInvisibility = 15
)

// supportCast is the level-gated sequence for the human player's mage at
// slot ci (R2-ENGINE-346, R2-ENGINE-347, R2-ENGINE-362). Each attempt goes
// through beginBookSpellOnce, the routine a player's own cast reaches, so
// book, cost and range admission are the cast's own. The first attempt that
// begins ends the sequence.
func (w *World) supportCast(ci int) {
	caster := w.entities[ci]
	s := w.AutoCast(caster.Owner)
	if caster.Mana < autoCastLevelArms[0].mana || s&autoCastLevelArms[0].bit == 0 {
		return
	}
	if caster.HP != caster.MaxHP && w.autoCastAt(ci, ci, autoCastHeal, false) {
		return
	}
	if w.autoCastAt(ci, ci, autoCastShield, true) {
		return
	}
	candidates := w.autoCastCandidates(ci, s)
	for _, ti := range candidates {
		if t := w.entities[ti]; t.MaxHP > 0 && t.HP/t.MaxHP < 1 {
			if w.autoCastAt(ci, ti, autoCastHeal, false) {
				return
			}
			break
		}
	}
	for _, l := range autoCastLevelArms[1:] {
		if s&l.bit == 0 || w.entities[ci].Mana < l.mana {
			return
		}
		for _, ti := range candidates {
			for _, arm := range l.arms {
				if arm == autoCastInvisibility && (ti == ci || w.autoCastNeutral(ci, ti)) {
					continue
				}
				if w.autoCastAt(ci, ti, arm, true) {
					return
				}
			}
		}
	}
	w.autoCastAt(ci, ci, autoCastInvisibility, true)
}

// autoCastAt attempts arm on the actor at ti. A buff whose effect stands on
// the target is not attempted.
func (w *World) autoCastAt(ci, ti int, arm uint16, buff bool) bool {
	id := w.spellIDOfArm(arm)
	rule, ok := w.bookSpell(w.entities[ci], uint32(id))
	if !ok || rule.arm() != arm || !knowsSpell(w.entities[ci], uint32(id)) || w.entities[ci].Mana < int32(rule.ManaCost) {
		return false
	}
	if _, standing := effectIndex(w.attached, w.entities[ti].ID, id); buff && standing {
		return false
	}
	if w.unsafeUnorderedArea(ci, ti, rule) {
		return false
	}
	return w.castOrder(ci, Cast(w.entities[ci].ID, w.entities[ti].ID, SpellID(id)))
}

func (w *World) autoCastNeutral(ci, ti int) bool {
	c, t := w.entities[ci], w.entities[ti]
	return c.Owner != t.Owner && !w.relations.Locked(c.Owner, t.Owner) && !w.hostileTo(&c, &t)
}

// autoCastCandidates lists living or raisable, non-hostile actors the caster sees and the
// target bits admit, in entity order. The caster is always admitted
// (R2-ENGINE-360). Range is the cast's own admission.
func (w *World) autoCastCandidates(ci int, s AutoCastSetting) []int {
	caster := w.entities[ci]
	var visible []byte
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
				continue
			case t.Owner == caster.Owner:
				if s&AutoCastOwn == 0 {
					continue
				}
			case w.relations.Locked(caster.Owner, t.Owner):
				if s&AutoCastAllies == 0 {
					continue
				}
			case s&AutoCastNeutral == 0:
				continue
			}
			if visible == nil {
				visible = w.actorSight(ci)
			}
			if !w.sightShows(visible, cell{x: t.X, y: t.Y}) {
				continue
			}
		}
		out = append(out, i)
	}
	return out
}
