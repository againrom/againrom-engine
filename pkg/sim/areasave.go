package sim

import "fmt"

type AreaSaveState struct {
	Key, Spell, Remaining          uint16
	Mode, Radius, Direction, Stage uint8
	Cells                          []uint16
	Payload                        SavedEffect
	Stopped                        bool
	Policy                         CurrentAreaPolicy
}

// NativeAreaSaveStates projects existing effects without applying a cast or
// changing its clock. Caster pointers have no original serialized field.
func (w *World) NativeAreaSaveStates() ([]AreaSaveState, error) {
	out := make([]AreaSaveState, 0, len(w.effects))
	for _, e := range w.effects {
		rule, ok := w.findSpell(uint32(e.Spell))
		if !ok {
			return nil, fmt.Errorf("sim: current area %d has no spell rule", e.Spell)
		}
		row := AreaSaveState{Key: e.Key, Spell: e.Spell, Remaining: e.Remaining,
			Mode: e.Mode, Radius: rule.Radius, Direction: e.Direction,
			Cells: append([]uint16(nil), e.Cells...), Policy: CurrentAreaPolicy{Power: e.Power, Caster: e.Caster, HasCaster: e.HasCaster, DerivedPayload: e.Current == nil}}
		if e.Mode == areaModeCloud {
			row.Policy.CloudPhase = e.Phase
		}
		if e.Mode == areaModeRing {
			row.Policy.RingLife = e.Remaining
			row.Stage = e.Phase/3 + 1
			row.Remaining = 2 - uint16(e.Phase%3)
		}
		payload := areaSavePayload(w, e, rule)
		row.Payload = payload
		if e.Current != nil {
			row.Payload, row.Radius, row.Stopped = e.Current.Payload, e.Current.Radius, e.Current.Stopped
		}
		out = append(out, row)
	}
	return out, nil
}

func OriginalEffectKind(kind EffectKind) uint8 {
	switch kind {
	case EffectHealth:
		return 6
	case EffectHealthRegeneration:
		return 8
	case EffectManaRegeneration:
		return 11
	case EffectAbsorption:
		return 16
	case EffectSpeed:
		return 17
	case EffectScanRange:
		return 19
	case EffectProtectionFire:
		return 21
	case EffectProtectionWater:
		return 22
	case EffectProtectionAir:
		return 23
	case EffectProtectionEarth:
		return 24
	case EffectInvisible:
		return 38
	case EffectBless:
		return 39
	case EffectCurse:
		return 40
	}
	return 0
}

func areaSavePayload(w *World, e cellEffect, rule SpellRule) SavedEffect {
	payload := SavedEffect{Class: "Effect", E0C: uint8(e.Spell)}
	if rule.Damaging || e.DamageMin != 0 || e.DamageMax != 0 {
		base, spread := spellDamageUnder(w.rules, rule, int32(e.Power))
		if e.DamageMin != 0 || e.DamageMax != 0 {
			base, spread = int64(e.DamageMin), int64(e.DamageMax-e.DamageMin)
		}
		payload.Class = "Effect_DirectDamage"
		payload.DirectDamage[19], payload.DirectDamage[20], payload.DirectDamage[21] = clampByte(base), clampByte(spread), rule.School
	} else if w.spellArm(e.Spell) != 19 {
		kind, magnitude, duration, mode := w.pointEffect(-1, rule, int32(e.Power))
		payload.E3C, payload.E3D = OriginalEffectKind(kind), uint8(mode)
		payload.E40 = uint32(magnitude)
		if mode != 0 {
			payload.E40 = uint32(uint16(magnitude)) | uint32(duration)<<16
		}
	}
	return payload
}
