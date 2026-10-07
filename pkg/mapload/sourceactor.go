package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/sim"
	"encoding/binary"
)

// BindSourceDerive installs a stateless arithmetic rule. It captures no table,
// actor, session or presentation object. Rebinding a native world is therefore
// independent of the App instance that originally imported its SAV.
func BindSourceDerive(w *sim.World) { w.BindSourceDerive(deriveSourceActor) }

// BindSourceItemDefinition resolves the original class-selected saved row.
// Native persistence retains these reached values, so a native producer is
// self-contained and never silently reads an appearance-code row instead.
func BindSourceItemDefinition(item sim.ItemInstance, t *Table) sim.ItemInstance {
	s := &item.SourceEquipment
	if s.Class != sim.SourceWeapon || t == nil || t.Weapons == nil || int(s.DefinitionRow) >= t.Weapons.Len() {
		return item
	}
	p := t.Weapons.EntryParams(int(s.DefinitionRow))
	if len(p) > 15 {
		s.Definition = sim.SourceWeaponDefinition{Present: true, AttackType: p[5], Hands: p[14], Charge: p[12], Relax: p[13], Suitable: p[15]}
	}
	return item
}

func deriveSourceActor(s sim.SourceActor, accumulator int32, r sim.Rules) (sim.SourceActor, error) {
	h := SourceHumanState(s, accumulator)
	h = h.WithSkillCap(r.SkillCap())
	n, err := h.Derive()
	if err != nil {
		return s, err
	}
	return humanSourceWithRuntime(n, s), nil
}

func humanSourceWithRuntime(h data.HumanState, prior sim.SourceActor) sim.SourceActor {
	out := HumanSourceActor(h, prior.Class)
	out.Reach, out.AttackCharge, out.AttackRelax, out.EquipmentRuntimePresent = prior.Reach, prior.AttackCharge, prior.AttackRelax, prior.EquipmentRuntimePresent
	return out
}

func sourceAttack(b [24]byte) data.HumanAttack {
	var a data.HumanAttack
	a.ToHit = binary.LittleEndian.Uint16(b[:])
	for i := range a.Skill {
		a.Skill[i] = binary.LittleEndian.Uint16(b[2+2*i:])
	}
	a.DamageBase, a.DamageSpread, a.Active = b[14], b[15], b[16]
	a.SecondBase, a.SecondSpread = b[17], b[18]
	a.ElementalBase, a.ElementalSpread, a.ElementalKind = b[19], b[20], b[21]
	a.Tail = [2]byte{b[22], b[23]}
	return a
}

func attackSource(a data.HumanAttack) (b [24]byte) {
	binary.LittleEndian.PutUint16(b[:], a.ToHit)
	for i, v := range a.Skill {
		binary.LittleEndian.PutUint16(b[2+2*i:], v)
	}
	copy(b[14:], []byte{a.DamageBase, a.DamageSpread, a.Active, a.SecondBase, a.SecondSpread, a.ElementalBase, a.ElementalSpread, a.ElementalKind, a.Tail[0], a.Tail[1]})
	return
}

func sourceDefence(b [22]byte) data.HumanDefence {
	d := data.HumanDefence{Defence: binary.LittleEndian.Uint16(b[:]), Absorption: binary.LittleEndian.Uint16(b[2:])}
	for i := range d.Protection {
		d.Protection[i] = binary.LittleEndian.Uint16(b[4+2*i:])
		d.Resistance[i] = b[16+i]
	}
	return d
}

func defenceSource(d data.HumanDefence) (b [22]byte) {
	binary.LittleEndian.PutUint16(b[:], d.Defence)
	binary.LittleEndian.PutUint16(b[2:], d.Absorption)
	for i, v := range d.Protection {
		binary.LittleEndian.PutUint16(b[4+2*i:], v)
		b[16+i] = d.Resistance[i]
	}
	return
}

func SourceHumanState(s sim.SourceActor, accumulator int32) data.HumanState {
	v := s.Stats
	h := data.HumanState{Body: v[0], Reaction: v[1], Mind: v[2], Spirit: v[3], Speed: v[4], Weight: v[5], Load: v[6], Capacity: v[7], Health: v[8], HealthMax: v[9], HealthPeriod: v[10], Mana: v[11], ManaMax: v[12], ManaPeriod: v[13],
		Attack: sourceAttack(s.Attack), Base: sourceAttack(s.Base), Defence: sourceDefence(s.Defence), SkillXP: s.SkillXP, Experience: s.Experience, ManaFloor: s.ManaFloor, Sight: s.Sight, TypeID: s.TypeID, MoverSpeed: s.MoverSpeed, Fighter: s.Fighter, HasSpellbook: s.HasSpellbook, HasOwner: s.HasOwner, ManaReservePercent: s.ManaReservePercent, InventoryWeight: accumulator}
	b := s.Modifier
	for i := range h.Modifier.StatCap {
		h.Modifier.StatCap[i] = int8(b[i])
	}
	m := &h.Modifier
	m.Speed, m.Capacity, m.HealthMax, m.HealthRegeneration = binary.LittleEndian.Uint16(b[4:]), binary.LittleEndian.Uint16(b[6:]), binary.LittleEndian.Uint16(b[8:]), binary.LittleEndian.Uint16(b[10:])
	m.ManaMax, m.ManaRegeneration, m.Sight = binary.LittleEndian.Uint16(b[12:]), binary.LittleEndian.Uint16(b[14:]), binary.LittleEndian.Uint16(b[16:])
	m.Attack = sourceAttack([24]byte(b[18:42]))
	m.Defence = sourceDefence([22]byte(b[42:64]))
	return h
}

func HumanSourceActor(h data.HumanState, class uint8) sim.SourceActor {
	s := sim.SourceActor{Class: class, Stats: [14]uint16{h.Body, h.Reaction, h.Mind, h.Spirit, h.Speed, h.Weight, h.Load, h.Capacity, h.Health, h.HealthMax, h.HealthPeriod, h.Mana, h.ManaMax, h.ManaPeriod},
		Attack: attackSource(h.Attack), Base: attackSource(h.Base), Defence: defenceSource(h.Defence), SkillXP: h.SkillXP, Experience: h.Experience, ManaFloor: h.ManaFloor, Sight: h.Sight, TypeID: h.TypeID, MoverSpeed: h.MoverSpeed, Fighter: h.Fighter, HasSpellbook: h.HasSpellbook, HasOwner: h.HasOwner, ManaReservePercent: h.ManaReservePercent}
	b := &s.Modifier
	m := h.Modifier
	for i, v := range m.StatCap {
		b[i] = byte(v)
	}
	for i, v := range []uint16{m.Speed, m.Capacity, m.HealthMax, m.HealthRegeneration, m.ManaMax, m.ManaRegeneration, m.Sight} {
		binary.LittleEndian.PutUint16(b[4+2*i:], v)
	}
	a, d := attackSource(m.Attack), defenceSource(m.Defence)
	copy(b[18:42], a[:])
	copy(b[42:64], d[:])
	return s
}
