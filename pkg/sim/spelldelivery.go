package sim

// A prepared effect keeps its power, rule and destination while its own clock
// runs. It owns no renderer object (MAGIC-DELIVERY-170). Target/source lifetime
// and the cell-centre metric are the bounded native policy in DIV-1271.
type spellDelivery struct {
	ConstructSacrifice bool `json:",omitempty"`
	BirthTick          uint64
	FanHead            bool
	Caster, Target     EntityID
	HasCaster          bool
	FromX, FromY, X, Y int32
	Rule               SpellRule
	Power              int32
	Remaining          uint16
	AtCell             bool
	Facing             uint8
	FixedFacing        bool
	Payload            *SavedEffect   `json:",omitempty"`
	Released           bool           `json:",omitempty"`
	Current            *AreaSaveState `json:",omitempty"`
	TransportStopped   bool           `json:",omitempty"`
	ChildStopped       bool           `json:",omitempty"`
	// aim names the actor aimed at; it sets the countdown and is not kept.
	aim areaAim
}

// areaAim is the actor an area spell was aimed at; Has is false for a cast at
// a cell.
type areaAim struct {
	Target EntityID
	Has    bool
}

func spellDeliveryTicks(rule SpellRule, x, y, toX, toY int32) uint16 {
	if rule.arm() == 13 || rule.arm() == 14 {
		return 10
	}
	if rule.EffectSpeed <= 0 {
		return 0
	}
	// Native actors are cell-centred; the original Position unit is 1/256 cell.
	distance := (cell{x: x, y: y}).chebyshevTo(cell{x: toX, y: toY}) * 256
	return uint16(distance / int64(rule.EffectSpeed))
}

// spellDeliveryRemaining is the countdown a new delivery starts at. A Fire_Ball
// transport waits the truncated Euclidean distance in 1/256 cell units from the
// caster's current point to the aimed actor's cell and fine position over the
// row's speed (SAV-1155). An actor wider than one cell is aimed at its footprint
// centre less one sub-unit per axis, and a cast at a cell at the cell's centre
// (MAGIC-245). Other deliveries keep the Chebyshev distance (DIV-1271).
func (w *World) spellDeliveryRemaining(d spellDelivery) uint16 {
	if d.Rule.ID != fireBallSpell || d.Rule.EffectSpeed <= 0 {
		return spellDeliveryTicks(d.Rule, d.FromX, d.FromY, d.X, d.Y)
	}
	fx, fy := d.FromX*256+128, d.FromY*256+128
	if i := indexOfEntity(w.entities, d.Caster); d.HasCaster && i >= 0 {
		fx, fy = w.savedProjectileTargetPoint(w.entities[i])
	}
	tx, ty := d.X*256+128, d.Y*256+128
	if i := indexOfEntity(w.entities, d.aim.Target); d.aim.Has && i >= 0 {
		tx, ty = w.savedProjectileTargetPoint(w.entities[i])
		if side := footprintSide(w.entities[i].TokenSize); side > 1 {
			// A footprint wider than one cell aims at its centre less one
			// sub-unit per axis, in sixteen bits (MAGIC-245).
			half := (side-1)*128 - 1
			tx, ty = int32(uint16(tx+half)), int32(uint16(ty+half))
		}
	}
	dx, dy := int64(tx)-int64(fx), int64(ty)-int64(fy)
	return uint16(min(isqrt64(dx*dx+dy*dy)/int64(d.Rule.EffectSpeed), 0xffff))
}

func (w *World) queueSpellDelivery(d spellDelivery) {
	d.BirthTick = w.tick
	d.Remaining = w.spellDeliveryRemaining(d)
	d.aim = areaAim{}
	// Resolve the transient book flag into the immutable payload before SAVE.
	if d.Rule.bookInstance {
		d.Rule.Defensive = d.Rule.bookDefensive == 1
	}
	d.Rule.bookInstance, d.Rule.bookDefensive = false, 0
	d.Rule.Delivery = 1 // the child is already delivered when consumed
	payload := w.preparedSpellPayload(d.Rule, d.Power, indexOfEntity(w.entities, d.Target))
	d.Payload = &payload
	w.prepareEffectAppend()
	w.deliveries = append(w.deliveries, d)
	w.noteEffectAppend(WorldEffectRef{Kind: EffectNativeDelivery, Index: uint32(len(w.deliveries) - 1)})
}

func (w *World) queuePointDelivery(ci, ti int, rule SpellRule, power int32) bool {
	if ti < 0 || ti >= len(w.entities) || !spellTargetable(w.entities[ti], rule) {
		return false
	}
	target := w.entities[ti]
	d := spellDelivery{Target: target.ID, X: target.X, Y: target.Y, Rule: rule, Power: power}
	if ci >= 0 && ci < len(w.entities) {
		caster := w.entities[ci]
		d.Caster, d.HasCaster, d.FromX, d.FromY = caster.ID, true, caster.X, caster.Y
	}
	w.queueSpellDelivery(d)
	return true
}

// preparePrismatic queues one delivery per selected victim, in output order.
func (w *World) preparePrismatic(ci, ti int, rule SpellRule, power int32, itemCast bool) []CellPoint {
	if ti < 0 || ti >= len(w.entities) || prismaticBodyPrimary(w.entities[ti], rule) {
		return nil
	}
	var victims []CellPoint
	for _, i := range w.prismaticVictims(ci, ti, rule, power) {
		if !w.queuePointDelivery(ci, i, rule, power) {
			continue
		}
		victims = append(victims, CellPoint{X: w.entities[i].X, Y: w.entities[i].Y})
		if ci >= 0 && !itemCast {
			w.awardSkill(ci, int32(rule.School), (int64(rule.ManaCost)+1)/2, -1)
		}
	}
	return victims
}

func (w *World) stepSpellDeliveries(obs *castObs) { w.stepWorldSpellEffects(obs) }

func (w *World) applySpellDelivery(d spellDelivery, obs *castObs) {
	if d.AtCell {
		current := d.Current
		if d.ConstructSacrifice && current != nil {
			v := *current
			v.Payload = w.preparedSpellPayload(d.Rule, d.Power, -1)
			if ci := indexOfEntity(w.entities, d.Caster); d.HasCaster && ci >= 0 {
				base, spread := sacrificeDamage(w.entities[ci], uint16(d.Power))
				if base != 0 || spread != 0 {
					v.Payload = SavedEffect{Class: "Effect_DirectDamage", E0C: uint8(d.Rule.ID)}
					v.Payload.DirectDamage[19], v.Payload.DirectDamage[20], v.Payload.DirectDamage[21] = uint8(base), uint8(spread), d.Rule.School
				}
			}
			current = &v
			// The ordinary graph carries only the unconstructed Effect/E0C
			// placeholder for a deferred sacrifice. Let the impact-time rule
			// build the area in that case; an edited direct-damage payload stays
			// explicit and is therefore retained through the landing.
			if d.Current.Payload.Class == "Effect" && d.Current.Payload.E0C == uint8(d.Rule.ID) {
				current = nil
			}
		}
		w.landAreaFacing(d.Rule, uint16(d.Power), d.Caster, d.HasCaster, d.FromX, d.FromY, d.X, d.Y, d.Facing, d.FixedFacing, obs, current)
		return
	}
	ti := indexOfEntity(w.entities, d.Target)
	if ti < 0 || !spellTargetable(w.entities[ti], d.Rule) {
		return
	}
	ci := -1
	if d.HasCaster {
		ci = indexOfEntity(w.entities, d.Caster)
	}
	if d.Payload != nil && SavedAreaPayloadSupported(d.Payload) {
		w.applyFrozenPoint(ci, ti, d.Rule, d.Payload)
		return
	}
	if w.ordinaryEffect(ci, ti, d.Rule, d.Power) {
		w.markSpellEffect(ti, d.Rule.ID)
	}
}

// PendingSpellDeliveries is gameplay state, including off-screen casts.
func (w *World) PendingSpellDeliveries() int { return len(w.deliveries) }

func (w *World) bookAlreadyPaid(caster EntityID, spell uint32) bool {
	i, ok := w.bookCastIndex(caster)
	return ok && w.bookCasts[i].Spell == uint16(spell) && w.bookCasts[i].Paid
}

func (w *World) admitBookPayment(ci int, rule SpellRule) {
	if rule.Delivery == 0 {
		return
	}
	i, ok := w.bookCastIndex(w.entities[ci].ID)
	if !ok || w.bookCasts[i].Paid {
		return
	}
	debitBook(&w.entities[ci], rule)
	w.bookCasts[i].Paid = true
	if w.bookCasts[i].AtCell || w.bookCasts[i].Target != w.entities[ci].ID {
		w.removeAttachedSpell(w.entities[ci].ID, w.armSpellID(15))
	}
	if rule.arm() == 14 && rule.Delivery == 2 && !w.bookCasts[i].AtCell {
		ti := indexOfEntity(w.entities, w.bookCasts[i].Target)
		if ti >= 0 {
			level := int32(0)
			if rule.School < skillSlots {
				level = w.entities[ci].Skill[rule.School]
			}
			first := len(w.deliveries)
			w.applyPrismatic(ci, ti, rule, spellPowerUnder(w.rules, rule, level, w.entities[ci].Mind))
			if len(w.deliveries) > first {
				w.deliveries[first].FanHead = true
			}
		}
	}
}

// Admission fan visuals are returned in the same Step that prepared them.
// BirthTick is also retained in a mid-flight save; LOAD does not replay it.
func (w *World) observeAdmissionFans(obs *castObs) {
	if obs == nil {
		return
	}
	for i, d := range w.deliveries {
		if !d.FanHead || d.BirthTick != w.tick {
			continue
		}
		ci, ti := indexOfEntity(w.entities, d.Caster), indexOfEntity(w.entities, d.Target)
		if ci < 0 || ti < 0 {
			continue
		}
		var victims []CellPoint
		for j := i; j < len(w.deliveries); j++ {
			p := w.deliveries[j]
			if j > i && p.FanHead || p.BirthTick != d.BirthTick || p.Caster != d.Caster || p.Rule.arm() != 14 {
				break
			}
			victims = append(victims, CellPoint{X: p.X, Y: p.Y})
		}
		obs.recordFrom(w, ci, ti, d.Rule, false, d.FromX, d.FromY)
		obs.recordVictims(victims)
	}
}
