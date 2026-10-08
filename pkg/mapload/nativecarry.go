package mapload

import (
	"encoding/binary"
	"reflect"

	"againrom/pkg/sim"
)

// NativeCarryHistory distinguishes an observed absence from an unobserved
// constructor input. Its bytes and availability belong to the last live actor.
type NativeCarryHistory struct {
	Basis sim.NativeActorBasis
	Class sim.NativeClass
}

func CaptureNativeCarryHistory(e sim.Entity) *NativeCarryHistory {
	if e.ActorLoad.Source.Class != 0 {
		return nil
	}
	return &NativeCarryHistory{Basis: e.NativeBasis, Class: e.NativeClass}
}

func carriedNativeHistory(p PartyMember, e *sim.Entity) {
	if p.Carry != nil && p.Carry.NativeHistory != nil {
		e.NativeBasis = p.Carry.NativeHistory.Basis
		e.NativeClass = p.Carry.NativeHistory.Class
	}
}

func updateNativePartyHistory(before PartyMember, p *PartyMember, t *Table, derive bool) {
	if before.Carry == nil || before.Carry.NativeHistory == nil || p.Carry == nil {
		return
	}
	history := *before.Carry.NativeHistory
	if p.Carry.NativeHistory != nil {
		history = *p.Carry.NativeHistory
	}
	old, next := MemberItemEquipment(before, t), MemberItemEquipment(*p, t)
	d, _, _ := PartyDisplayWithTable(before, t)
	// A replacement lands before the item it displaces from the opposite hand.
	for _, removals := range []bool{false, true} {
		for slot := range old {
			if reflect.DeepEqual(old[slot], next[slot]) || next[slot].Empty() != removals {
				continue
			}
			history.Basis = sim.UpdateNativeEquipmentChangeBasis(history.Basis, SourceConstructedItem(old[slot], t), SourceConstructedItem(next[slot], t), d.Skill[0], true, history.Class)
		}
	}
	if derive {
		d, _, _ := PartyDisplayWithTable(*p, t)
		if p.Hero.Body != before.Hero.Body {
			history.Basis = history.Basis.WithBody(uint16(d.Body))
		}
		if history.Basis.BasePresent {
			for slot := 1; slot < len(p.Hero.Skill); slot++ {
				at := 2 + 2*slot
				binary.LittleEndian.PutUint16(history.Basis.Base[at:], uint16(p.Hero.Skill[slot]))
				history.Basis.BaseKnown |= uint32(3) << at
			}
		}
		if history.Basis.ModifierPresent {
			binary.LittleEndian.PutUint16(history.Basis.Modifier[10:], uint16(d.HealthRegeneration))
			binary.LittleEndian.PutUint16(history.Basis.Modifier[14:], uint16(d.ManaRegeneration))
			history.Basis.ModifierKnown |= uint64(3)<<10 | uint64(3)<<14
		}
	}
	p.Carry.NativeHistory = &history
}
