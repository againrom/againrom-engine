package sim

import "fmt"

// CurrentProfileBasis records authority, not an original process pointer.
// Retirement is persistent: a later native load never reinstates the old
// source sheet from a construction-time roster cache (DIV-738).
type CurrentProfileBasis uint8

const (
	ProfileNative CurrentProfileBasis = iota
	ProfileOriginalCurrent
	ProfileNativeRetired
)

// OriginalActorProfile contains only consumed current fields. Periods and
// percentages are signed words; fractional state is the entire saved byte.
// Pools have their own final import; equipment, books, skill progress and
// incoming orders are not part of this mutation.
type OriginalActorProfile struct {
	ID                                   EntityID
	ProfileBasis                         *CurrentProfileBasis
	Reaction, Mind, Spirit               int16
	ToHit, Defence, Absorption           int16
	DamageBase, DamageSpread, XPSlot     uint8
	SecondBase, SecondSpread             uint8
	SecondaryDamage                      SecondaryDamage
	Protection                           [5]int16
	Resistance                           [5]uint8
	HealthPeriod, ManaPeriod             int16
	HealthRegeneration, ManaRegeneration int16
	HealthHundredths, ManaHundredths     uint8
}

// ImportOriginalActorProfiles validates the complete batch before its first
// write. Assigning source current values must not replay effects, refresh a
// spellbook, recompute a template, consume RNG, or advance any action clock.
func (w *World) ImportOriginalActorProfiles(batch []OriginalActorProfile) error {
	if w == nil {
		return fmt.Errorf("original current profile: no world")
	}
	seen := make(map[EntityID]bool, len(batch))
	for _, p := range batch {
		i := indexOfEntity(w.entities, p.ID)
		if i < 0 || seen[p.ID] {
			return fmt.Errorf("original current profile: absent or duplicate entity %d", p.ID)
		}
		e := w.entities[i]
		if !e.Alive() && !originalDyingEntity(e) {
			return fmt.Errorf("original current profile: entity %d is not living or dying", p.ID)
		}
		if p.ProfileBasis != nil && *p.ProfileBasis > ProfileNativeRetired {
			return fmt.Errorf("original current profile: entity %d invalid arithmetic policy %d", p.ID, *p.ProfileBasis)
		}
		if p.XPSlot >= skillSlots {
			return fmt.Errorf("original current profile: entity %d unsupported active skill %d", p.ID, p.XPSlot)
		}
		if err := secondaryDamageFault(p.SecondaryDamage); err != nil {
			return fmt.Errorf("original current profile: entity %d: %w", p.ID, err)
		}
		seen[p.ID] = true
	}
	for _, p := range batch {
		e := &w.entities[indexOfEntity(w.entities, p.ID)]
		e.CurrentProfileBasis = ProfileOriginalCurrent
		if p.ProfileBasis != nil {
			e.CurrentProfileBasis = *p.ProfileBasis
		}
		e.Reaction, e.Mind, e.Spirit = int32(p.Reaction), int32(p.Mind), int32(p.Spirit)
		e.ToHit, e.Defence, e.Absorption = int32(p.ToHit), int32(p.Defence), int32(p.Absorption)
		e.DamageBase, e.DamageSpread, e.XPSlot = int32(p.DamageBase), int32(p.DamageSpread), p.XPSlot
		e.SecondBase, e.SecondSpread, e.SecondaryDamage = p.SecondBase, p.SecondSpread, p.SecondaryDamage
		for i := range p.Protection {
			e.Protection[i] = int32(p.Protection[i])
		}
		e.Resistance = p.Resistance
		e.HealthRegenPeriod, e.ManaRegenPeriod = int32(p.HealthPeriod), int32(p.ManaPeriod)
		e.HealthRegeneration, e.ManaRegeneration = int32(p.HealthRegeneration), int32(p.ManaRegeneration)
		e.HealthHundredths, e.ManaHundredths = p.HealthHundredths, p.ManaHundredths
		e.liftActiveSkillTerms()
	}
	return nil
}

func (e *Entity) retireCurrentProfile() {
	if e.CurrentProfileBasis == ProfileOriginalCurrent {
		e.CurrentProfileBasis = ProfileNativeRetired
	}
}
