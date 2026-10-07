package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// PotionHero folds only earned permanent gains, not a temporary modifier.
func PotionHero(h data.Hero, gains [4]int32) data.Hero {
	h.Body += gains[0]
	h.Reaction += gains[1]
	h.Mind += gains[2]
	h.Spirit += gains[3]
	return h
}

// PotionHeadroom is the remaining single-use gain under the ordinary attribute
// cap and the hard ceiling. Equipment bonuses remain modifiers, not earned stats.
func PotionHeadroom(h data.Hero, d data.Derived) (out [4]int32) {
	base := [4]int32{h.Body, h.Reaction, h.Mind, h.Spirit}
	live := [4]int32{d.Body, d.Reaction, d.Mind, d.Spirit}
	for i := range out {
		out[i] = min(50-base[i], 100-live[i])
		out[i] = max(0, min(100, out[i]))
	}
	return out
}

// PartyDisplayWithTable is the read-only town/preview projection. Spawn stays
// base-only because initializePotions attaches the carried timer to the new
// actor separately. Display folds it once without advancing or storing state.
func PartyDisplayWithTable(p PartyMember, t *Table) (d data.Derived, health, mana int32) {
	// The first two hire types use fixed creature templates, including in
	// the town card. StartMission resolves this same placement block.
	resolvedSiege := false
	if p.MercenaryType == 1 || p.MercenaryType == 2 {
		if b, err := blockFor(alm.Unit{ClassID: int16(p.Class), ClassSubID: uint16(p.FigureFace)}, t, DifficultyNormal); err == nil {
			d = data.Derived{Body: p.Hero.Body, Reaction: b.reaction, Mind: b.mind, Spirit: b.spirit,
				HealthMax: b.health, ManaMax: b.manaMax, Combat: b.combat,
				Protection: b.protection, Speed: b.speed, Sight: int32(b.sight),
				HealthRegeneration: b.healthRegeneration, ManaRegeneration: b.manaRegeneration,
				RotationSpeed: b.rotationSpeed, SecondaryDamage: b.secondaryDamage}
			for i, value := range b.resistance {
				d.Resistance[i] = int32(value)
			}
			health, mana = b.health, b.mana
			resolvedSiege = true
		}
	}
	if !resolvedSiege {
		d, health, mana = PartySpawnWithTable(p, t)
	}
	if p.PotionEffect == nil || p.Carry != nil && p.Carry.LiveLoad != nil && p.Carry.LiveLoad.Inventory.Source.Class == 2 {
		return
	}
	base := sim.Entity{HP: health, MaxHP: d.HealthMax, Mana: mana, MaxMana: d.ManaMax,
		Absorption: d.Combat.Absorption, HealthRegeneration: d.HealthRegeneration, ManaRegeneration: d.ManaRegeneration}
	if projected, ok := sim.ProjectPotionEffect(base, *p.PotionEffect); ok {
		d.Combat.Absorption = projected.Absorption
		d.HealthRegeneration = projected.HealthRegeneration
		d.ManaRegeneration = projected.ManaRegeneration
	}
	return
}

func initializePotions(w *sim.World, party []PartyMember, st Start, table *Table) {
	for i, p := range party {
		if i >= len(st.IDs) {
			break
		}
		d, _, _ := PartySpawnWithTable(p, table)
		w.SetPotionHeadroom(st.IDs[i], PotionHeadroom(p.Hero, d))
		if p.PotionEffect != nil {
			if p.Carry != nil && p.Carry.LiveLoad != nil && p.Carry.LiveLoad.Inventory.Source.Class == 2 {
				w.RestoreAppliedPotionEffect(st.IDs[i], *p.PotionEffect)
			} else {
				w.RestorePotionEffect(st.IDs[i], *p.PotionEffect)
			}
		}
	}
}

// ApplyTownPotion uses the simulation's exact Item application without a town
// clock. The returned member owns its copied permanent/timed state; callers
// commit it only after taking one source unit. Campaign pool minting is unchanged.
func ApplyTownPotion(p PartyMember, item sim.ItemInstance, table *Table) (PartyMember, bool) {
	if !sim.UsablePotion(item) {
		return p, false
	}
	if p.Carry != nil && p.Carry.LiveLoad != nil && p.Carry.LiveLoad.Inventory.Source.Class == 2 {
		return applySourceTownPotion(p, item)
	}
	d, hp, mana := PartySpawnWithTable(p, table)
	e := sim.Entity{ID: 1, HP: hp, MaxHP: d.HealthMax, Mana: mana, MaxMana: d.ManaMax, Absorption: d.Combat.Absorption,
		HealthRegeneration: d.HealthRegeneration, ManaRegeneration: d.ManaRegeneration, PotionHeadroom: PotionHeadroom(p.Hero, d)}
	w, err := sim.NewStockedWorld(0, sim.Bounds{Width: 1, Height: 1}, sim.ModeCanonical, sim.Terrain{}, []sim.Entity{e}, nil, sim.Relations{}, nil, []sim.Stock{{ID: 1, ItemInstances: []sim.ItemInstance{item}}})
	if err != nil {
		return p, false
	}
	if p.PotionEffect != nil && !w.RestorePotionEffect(1, *p.PotionEffect) {
		return p, false
	}
	if !w.UseCarriedPotion(1, 0) {
		return p, false
	}
	out := clonePartyMember(p)
	actor := w.Entities()[0]
	out.Hero = PotionHero(out.Hero, actor.PotionStats)
	out.PotionEffect = nil
	for _, effect := range w.ActiveEffects() {
		if effect.Spell == 0 {
			out.PotionEffect = &effect
			break
		}
	}
	if out.Saved != nil {
		out.Saved.HP, out.Saved.Mana = actor.HP, actor.Mana
	}
	out.RetireOriginalHuman()
	UpdatePartyLoad(p, &out, table, true, true)
	return out, true
}

func applySourceTownPotion(p PartyMember, item sim.ItemInstance) (PartyMember, bool) {
	// The city transaction owns the actual source item. This isolated use
	// applies its effects only; commit refreshes from the post-transaction
	// accumulator. Do not subtract a fictitious carried unit's weight twice.
	item = item.Clone()
	item.WeightPresent, item.Weight = true, 0
	e := sim.Entity{ID: 1, HP: 1, MaxHP: 1}
	w, err := sim.NewStockedWorld(0, sim.Bounds{Width: 1, Height: 1}, sim.ModeCanonical, sim.Terrain{}, []sim.Entity{e}, nil, sim.Relations{}, nil, []sim.Stock{{ID: 1, ItemInstances: []sim.ItemInstance{item}}})
	if err != nil {
		return p, false
	}
	BindSourceDerive(w)
	if p.PotionEffect != nil && !w.RestorePotionEffect(1, *p.PotionEffect) {
		return p, false
	}
	if w.RestoreActorLoad(1, *p.Carry.LiveLoad) != nil || !w.UseCarriedPotion(1, 0) {
		return p, false
	}
	if w.Entities()[0].Capacity == 0 {
		return p, false
	}
	out := clonePartyMember(p)
	out.Carry.LiveLoad = w.Entities()[0].CurrentActorLoad()
	out.Hero = SourceHumanState(out.Carry.LiveLoad.Inventory.Source, out.Carry.LiveLoad.Inventory.Accumulator).Hero()
	out.PotionEffect = nil
	for _, effect := range w.ActiveEffects() {
		if effect.Spell == 0 {
			out.PotionEffect = &effect
			break
		}
	}
	out.RetireOriginalHuman()
	return out, true
}

// CommitSourceTownPotion publishes a preflighted source effect after the city
// has removed its item. The result's retained modifiers remain authoritative;
// only the live container bookkeeping comes from the transaction.
func CommitSourceTownPotion(member *PartyMember, result PartyMember) bool {
	if member.Carry == nil || member.Carry.LiveLoad == nil || result.Carry == nil || result.Carry.LiveLoad == nil || result.Carry.LiveLoad.Inventory.Source.Class != 2 {
		return false
	}
	s := *member.Carry.LiveLoad
	h := SourceHumanState(result.Carry.LiveLoad.Inventory.Source, s.Inventory.Accumulator)
	h.Weight = uint16(s.Inventory.OwnWeight)
	n, _, err := h.RefreshInventoryLoad()
	if err != nil {
		return false
	}
	s.Inventory.Source = humanSourceWithRuntime(n, s.Inventory.Source)
	s.Load, s.Capacity, s.Speed = int32(int16(n.Load)), int32(int16(n.Capacity)), int32(int16(n.Speed))
	s.Movement = sim.HumanMovement{Present: true, RawSpeed: int16(n.Speed), NativeSpeed: s.Speed, Load: s.Load, Capacity: s.Capacity}
	member.Carry.LiveLoad = &s
	member.Hero, member.PotionEffect = n.Hero(), result.PotionEffect
	member.RetireOriginalHuman()
	if member.Saved != nil {
		member.Saved.HP, member.Saved.MaxHP, member.Saved.Mana, member.Saved.MaxMana = int32(int16(n.Health)), int32(int16(n.HealthMax)), int32(int16(n.Mana)), int32(int16(n.ManaMax))
	}
	return true
}
