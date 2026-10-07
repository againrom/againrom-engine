package sim

type SpellDeliverySaveState struct {
	Remaining                      uint16
	Released                       bool
	Attribution                    bool
	Target                         EntityID
	HasTarget                      bool
	Caster                         EntityID
	HasCaster                      bool
	FromX, FromY                   int32
	Area                           AreaSaveState
	AtCell                         bool
	Policy                         CurrentDeliveryPolicy `json:"-"`
	TransportStopped, ChildStopped bool                  `json:"-"`
}

func (w *World) preparedSpellPayload(rule SpellRule, power int32, target int) SavedEffect {
	p := SavedEffect{Class: "Effect", E0C: uint8(rule.ID)}
	if rule.Damaging {
		base, spread := spellDamageUnder(w.rules, rule, power)
		p.Class = "Effect_DirectDamage"
		p.DirectDamage[19], p.DirectDamage[20], p.DirectDamage[21] = clampByte(base), clampByte(spread), rule.School
	} else {
		kind, magnitude, duration, mode := w.pointEffect(target, rule, power)
		p.E3C, p.E3D = OriginalEffectKind(kind), uint8(mode)
		p.E40 = uint32(magnitude)
		if mode != 0 {
			p.E40 = uint32(uint16(magnitude)) | uint32(duration)<<16
		}
	}
	return p
}

func (w *World) NativeSpellDeliverySaveStates() []SpellDeliverySaveState {
	out := make([]SpellDeliverySaveState, 0, len(w.deliveries))
	for _, d := range w.deliveries {
		a := AreaSaveState{Key: cellKey(d.X, d.Y), Spell: d.Rule.ID, Radius: d.Rule.Radius, Mode: areaModeFor(d.Rule), Direction: areaDirection(d.X-d.FromX, d.Y-d.FromY)}
		if d.FixedFacing {
			a.Direction = uint8(FacingDir(d.Facing))
		}
		if a.Mode == areaModeCloud {
			a.Remaining = areaLife(d.Rule, uint16(d.Power)) - 1
		}
		a.Payload = w.preparedSpellPayload(d.Rule, d.Power, indexOfEntity(w.entities, d.Target))
		if d.Payload != nil {
			a.Payload = *d.Payload
		}
		if d.Current != nil {
			a = *d.Current
		} else if d.AtCell && d.Rule.ID == 4 {
			// Sacrifice constructs its payload from impact-time pools. The
			// queued table payload is not yet the effect that will be applied.
			a.Payload = SavedEffect{Class: "Effect", E0C: uint8(d.Rule.ID)}
		}
		p := CurrentDeliveryPolicy{BirthTick: d.BirthTick, FanHead: d.FanHead, Caster: d.Caster, HasCaster: d.HasCaster, Power: d.Power,
			DerivedArea: d.Current == nil, Restorative: d.Rule.Restorative, Distribution: d.Rule.Distribution}
		if a.Payload.Class != "Effect_DirectDamage" {
			school := d.Rule.School
			p.School = &school
		}
		if d.Released {
			p.Origin = &CellPoint{X: d.FromX, Y: d.FromY}
			timer := d.Remaining
			p.ReleasedTimer = &timer
		}
		if !SavedAreaPayloadSupported(&a.Payload) {
			p.Special = &CurrentDeliveryOperands{Damaging: d.Rule.Damaging, DamageMin: d.Rule.DamageMin, DamageMax: d.Rule.DamageMax, Kind: d.Rule.EffectKind,
				Mode: d.Rule.EffectMode, Magnitude: d.Rule.EffectMagnitude, Duration: d.Rule.EffectDuration, SpellDuration: d.Rule.SpellDuration}
		}
		out = append(out, SpellDeliverySaveState{Remaining: d.Remaining, Released: d.Released, Attribution: !d.Rule.Defensive, Target: d.Target, HasTarget: !d.AtCell, Caster: d.Caster, HasCaster: d.HasCaster, FromX: d.FromX, FromY: d.FromY, Area: a, AtCell: d.AtCell, Policy: p, TransportStopped: d.TransportStopped, ChildStopped: d.ChildStopped})
	}
	return out
}
