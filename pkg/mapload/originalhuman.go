package mapload

import (
	"reflect"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// OriginalHuman retains a source-backed current-state basis across city/native
// saves. The ownership guard prevents a subsequent item, potion or unrelated
// character edit from silently reusing modifiers belonging to another state.
type OriginalHuman struct {
	Version   uint32
	Retired   bool
	PartyID   string
	State     data.HumanState
	Equipment [sim.EquipSlots]sim.ItemInstance
	Inventory []sim.ItemInstance
	Weapon    *data.Weapon
}

func BindOriginalHuman(p PartyMember, state data.HumanState) *OriginalHuman {
	out := &OriginalHuman{Version: 1, PartyID: p.ID, State: state,
		Equipment: MemberItemEquipment(p, nil), Inventory: cloneItemInstances(MemberCarriedItems(p, nil))}
	if p.Weapon != nil {
		w := *p.Weapon
		out.Weapon = &w
	}
	return out
}

func (p PartyMember) OriginalHumanState() (data.HumanState, bool) {
	h := p.OriginalHuman
	if h == nil || h.Retired || h.Version != 1 || h.PartyID != p.ID || p.Mage == h.State.Fighter || p.Hired() || p.PotionEffect != nil ||
		h.State.Hero() != p.Hero || p.Carry == nil || p.Saved == nil ||
		!reflect.DeepEqual(h.Weapon, p.Weapon) ||
		!reflect.DeepEqual(h.Equipment, MemberItemEquipment(p, nil)) ||
		!reflect.DeepEqual(h.Inventory, cloneItemInstances(MemberCarriedItems(p, nil))) {
		return data.HumanState{}, false
	}
	for i, xp := range p.Carry.SkillXP {
		if uint32(xp) != h.State.SkillXP[i] {
			return data.HumanState{}, false
		}
	}
	s, n := p.Saved, h.State
	if s.HP != int32(int16(n.Health)) || s.MaxHP != int32(int16(n.HealthMax)) ||
		s.Mana != int32(int16(n.Mana)) || s.MaxMana != int32(int16(n.ManaMax)) ||
		s.HealthRegenPeriod != int32(n.HealthPeriod) || s.ManaRegenPeriod != int32(n.ManaPeriod) {
		return data.HumanState{}, false
	}
	return n, true
}

// RetireOriginalHuman stops using source-derived arithmetic after an ordinary
// native mutation. Keep the immutable basis for validation and explicit export
// refusal; dropping it would let a malformed retained state escape validation.
func (p *PartyMember) RetireOriginalHuman() {
	if p.OriginalHuman != nil && !p.OriginalHuman.Retired {
		h := *p.OriginalHuman
		h.Retired = true
		p.OriginalHuman = &h
	}
}

// ApplyOriginalHuman changes only the coupled fields a retained Human derive
// owns. Equipment, identity, effects and every unrelated party field remain.
func ApplyOriginalHuman(p *PartyMember, state data.HumanState) {
	p.Hero = state.Hero()
	for i, xp := range state.SkillXP {
		p.Carry.SkillXP[i] = int32(xp)
	}
	p.Saved.HP, p.Saved.MaxHP = int32(int16(state.Health)), int32(int16(state.HealthMax))
	p.Saved.Mana, p.Saved.MaxMana = int32(int16(state.Mana)), int32(int16(state.ManaMax))
	p.OriginalHuman = BindOriginalHuman(*p, state)
	applyHumanLoad(p, state)
}

func originalHumanSpawn(p PartyMember, t *Table) (data.Derived, int32, int32, bool) {
	if p.Carry != nil && p.Carry.LiveLoad != nil && p.Carry.LiveLoad.Inventory.Source.Class == 2 {
		s := p.Carry.LiveLoad
		h := SourceHumanState(s.Inventory.Source, s.Inventory.Accumulator)
		h.Weight, h.Load, h.Capacity, h.Speed = uint16(s.Inventory.OwnWeight), uint16(s.Load), uint16(s.Capacity), uint16(s.Movement.RawSpeed)
		d := h.Derived(MemberWeapon(p, t), int32(h.MoverSpeed))
		currentHumanWeaponSpell(&d, p, t)
		if v := s.Inventory.Source; v.EquipmentRuntimePresent {
			d.Combat.Reach, d.Combat.AttackChargeTime, d.Combat.AttackRelaxTime = int32(v.Reach), int32(v.AttackCharge), int32(v.AttackRelax)
		}
		d.SpeedModifier = 0
		return d, int32(int16(h.Health)), int32(int16(h.Mana)), true
	}
	h, ok := p.OriginalHumanState()
	if !ok {
		return data.Derived{}, 0, 0, false
	}
	d := h.Derived(MemberWeapon(p, t), int32(h.MoverSpeed))
	currentHumanWeaponSpell(&d, p, t)
	// A source-backed Human's modifier lives in its source record, not in a
	// native SpeedModifier.
	d.SpeedModifier = 0
	return d, int32(int16(h.Health)), int32(int16(h.Mana)), true
}

func currentHumanWeaponSpell(d *data.Derived, p PartyMember, t *Table) {
	d.Combat.SpellName, d.Combat.SpellPower = currentItemWeaponSpell(MemberItemEquipment(p, t)[0], p.Weapon, t)
}

func initializeOriginalHumanMovement(w *sim.World, party []PartyMember, st Start) error {
	BindSourceDerive(w)
	for i, p := range party {
		if i >= len(st.IDs) {
			break
		}
		if h, ok := p.OriginalHumanState(); ok {
			w.SetHumanMovement(st.IDs[i], int16(h.Speed), int32(int16(h.Load)))
		}
		if p.Carry != nil && p.Carry.LiveLoad != nil {
			if err := w.RestoreActorLoad(st.IDs[i], *p.Carry.LiveLoad); err != nil {
				return err
			}
		}
	}
	return nil
}
