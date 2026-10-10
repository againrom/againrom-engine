package data

import (
	"fmt"
	"math"

	"againrom/pkg/rules"
)

// HumanAttack retains the three independent live, maintained-base and modifier
// blocks. The active-index mirror and unnamed tail are not folded into live
// state (SAV-HUMFOLD-446, SAV-HUMGAPS-449).
type HumanAttack struct {
	ToHit                                         uint16
	Skill                                         [6]uint16
	DamageBase, DamageSpread, Active              uint8
	SecondBase, SecondSpread                      uint8
	ElementalBase, ElementalSpread, ElementalKind uint8
	Tail                                          [2]byte
}

type HumanDefence struct {
	Defence, Absorption uint16
	Protection          [6]uint16
	Resistance          [6]uint8
}

type HumanModifier struct {
	StatCap                                        [4]int8
	Speed, Capacity, HealthMax, HealthRegeneration uint16
	ManaMax, ManaRegeneration, Sight               uint16
	Attack                                         HumanAttack
	Defence                                        HumanDefence
}

// HumanState is source-backed runtime state, not a reconstructed item sum.
// All word/byte operations retain the original storage width. Unknown attack
// tails and unmaintained base fields survive every training operation.
type HumanState struct {
	Body, Reaction, Mind, Spirit                uint16
	Speed, Weight, Load, Capacity               uint16
	Health, HealthMax, HealthPeriod             uint16
	Mana, ManaMax, ManaPeriod, ManaFloor, Sight uint16
	Attack, Base                                HumanAttack
	Defence                                     HumanDefence
	Modifier                                    HumanModifier
	SkillXP                                     [6]uint32
	Experience                                  uint32
	MoverSpeed                                  uint8
	Fighter, HasSpellbook                       bool
	TypeID                                      uint16
	InventoryWeight                             int32
	HasOwner                                    bool
	ManaReservePercent                          uint32

	// skillCap is the highest level a derived skill may hold. Zero is the
	// original game's 100, so a state built without rules derives as before.
	// It is unexported so no serialized form of the state carries it.
	skillCap int32
}

// WithSkillCap returns h with the highest level a derived skill may hold.
func (h HumanState) WithSkillCap(n int32) HumanState {
	h.skillCap = n
	return h
}

func (h HumanState) skillLimit() int32 {
	if h.skillCap == 0 {
		return SkillCap
	}
	return h.skillCap
}

func humanSigned(v uint16) int32 { return int32(int16(v)) }

// humanFTOL models the low dword of the original truncating signed-qword
// conversion only inside its defined numeric range. Outside it the producer's
// result is Unknown, so the caller must refuse before changing the purse.
func humanFTOL(v float64) (int32, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < -0x1p63 || v >= 0x1p63 {
		return 0, fmt.Errorf("Human arithmetic exceeds the proven signed-qword domain")
	}
	return int32(int64(v)), nil
}

func (h HumanState) TrainingPrice(slot int) (int32, error) {
	if slot < 1 || slot > 5 {
		return 0, fmt.Errorf("school training requires slot 1 through 5")
	}
	return rules.SchoolPrice(humanSigned(h.Base.Skill[slot]))
}

// ProjectionError validates the source-backed current combat projection.
func (h HumanState) ProjectionError() error {
	if h.Attack.Active > 5 {
		return fmt.Errorf("unsupported Human active skill index")
	}
	if (h.Attack.ElementalBase != 0 || h.Attack.ElementalSpread != 0) && (h.Attack.ElementalKind < 1 || h.Attack.ElementalKind > 5) {
		return fmt.Errorf("unsupported Human elemental protection selector")
	}
	return nil
}

// FighterProjectionError guards the native fallback, whose item-derived
// arithmetic cannot reconstruct an observed current or modifier second pair.
// Source-backed projection uses ProjectionError instead (DIV-674, DIV-675).
func (h HumanState) FighterProjectionError() error {
	if err := h.ProjectionError(); err != nil {
		return err
	}
	if h.Attack.SecondBase != 0 || h.Attack.SecondSpread != 0 || h.Modifier.Attack.SecondBase != 0 || h.Modifier.Attack.SecondSpread != 0 {
		return fmt.Errorf("SAV Human native fallback cannot derive the second physical damage component")
	}
	return nil
}

// Train applies R0838 -> R0280 -> R0840 to a value copy. It is
// transactional: no receiver changes survive any unsupported arithmetic.
func (h HumanState) Train(slot int) (HumanState, error) {
	if slot < 1 || slot > 5 || h.Attack.Active > 5 {
		return h, fmt.Errorf("unsupported Human skill index")
	}
	n := h
	for i := 1; i <= 5; i++ {
		n.Attack.Skill[i] = n.Base.Skill[i]
	}
	n.Attack.Skill[slot]++
	xp, err := rules.SchoolTrainedXP(humanSigned(n.Attack.Skill[slot]))
	if err != nil {
		return h, err
	}
	nextXP := uint32(xp)
	n.Experience += nextXP - n.SkillXP[slot]
	n.SkillXP[slot] = nextXP
	for i := 1; i <= 5; i++ {
		n.Base.Skill[i] = n.Attack.Skill[i]
	}
	if err := n.derive(); err != nil {
		return h, err
	}
	return n, nil
}

// RefreshInventoryLoad applies the sale's R0451(0) after the container
// mutation (SAV-CITYSALE-513). It uses the old stored capacity for BOTH signed
// quotients. Equal quotients preserve every field except load; only a changed
// quotient reaches Human derive. A faulting source divisor is refused before
// any state is published, not replaced with a guessed capacity.
func (h HumanState) RefreshInventoryLoad() (HumanState, bool, error) {
	if h.Capacity == 0 {
		return h, false, fmt.Errorf("Human load refresh would divide by zero capacity")
	}
	n := h
	n.Load = humanLoad(n.Weight, n.InventoryWeight)
	derived := humanSigned(h.Load)/humanSigned(h.Capacity) != humanSigned(n.Load)/humanSigned(h.Capacity)
	if derived {
		if err := n.derive(); err != nil {
			return h, false, err
		}
	}
	return n, derived, nil
}

// Derive is the reached Human/Humanoid vt+50, on a value copy. LOAD does not
// call it. The simulation adapter invokes it synchronously at a producer.
func (h HumanState) Derive() (HumanState, error) {
	n := h
	if err := n.derive(); err != nil {
		return h, err
	}
	return n, nil
}

func (h *HumanState) derive() error {
	m := &h.Modifier
	in := HumanInput{Active: int32(h.Attack.Active), Experience: int32(h.Experience), Fighter: h.Fighter,
		ManaPool: h.ManaMax != 0, Rider: RiderTypeID(int32(h.TypeID)), TrainingCap: h.skillCap}
	for i, v := range []uint16{h.Body, h.Reaction, h.Mind, h.Spirit} {
		in.Stat[i], in.StatCap[i] = humanSigned(v), int32(m.StatCap[i])
	}
	in.Skill[SkillGeneral] = humanSigned(h.Attack.Skill[SkillGeneral])
	for i := range in.Skill {
		if i != int(SkillGeneral) {
			in.Skill[i] = humanSigned(h.Base.Skill[i])
		}
		in.Terms.SkillBonus[i] = humanSigned(m.Attack.Skill[i])
	}
	h.Load = humanLoad(h.Weight, h.InventoryWeight)
	in.Load = humanSigned(h.Load)
	in.Terms = HumanTerms{Speed: humanSigned(m.Speed), Capacity: humanSigned(m.Capacity), HealthMax: humanSigned(m.HealthMax),
		ManaMax: humanSigned(m.ManaMax), Sight: humanSigned(m.Sight), SkillBonus: in.Terms.SkillBonus,
		ToHit: humanSigned(m.Attack.ToHit), DamageBase: int32(m.Attack.DamageBase), DamageSpread: int32(m.Attack.DamageSpread),
		Defence: humanSigned(m.Defence.Defence), Absorption: humanSigned(m.Defence.Absorption)}
	for i := range in.Terms.Protection {
		in.Terms.Protection[i], in.Terms.Resistance[i] = humanSigned(m.Defence.Protection[i+1]), int32(m.Defence.Resistance[i+1])
	}
	out, err := DeriveHuman(in)
	if err != nil {
		return err
	}
	h.Body, h.Reaction, h.Mind, h.Spirit = uint16(out.Stat[0]), uint16(out.Stat[1]), uint16(out.Stat[2]), uint16(out.Stat[3])
	h.HealthMax, h.ManaMax, h.Sight, h.Capacity = uint16(out.HealthMax), uint16(out.ManaMax), uint16(out.Sight), uint16(out.Capacity)
	if !in.ManaPool {
		h.Mana = 0
	}
	h.Speed, m.Speed = uint16(out.Speed), uint16(out.SpeedModifier)
	for i := range h.Attack.Skill {
		h.Attack.Skill[i] = uint16(out.Skill[i])
	}
	h.Attack.ToHit, h.Attack.DamageBase, h.Attack.DamageSpread = uint16(out.ToHit), out.DamageBase, out.DamageSpread
	// The modifier's second pair and elemental triple are the derived
	// block's own: the derive clears both and the fold adds them.
	h.Attack.SecondBase, h.Attack.SecondSpread = m.Attack.SecondBase, m.Attack.SecondSpread
	h.Attack.ElementalBase, h.Attack.ElementalSpread, h.Attack.ElementalKind = m.Attack.ElementalBase, m.Attack.ElementalSpread, m.Attack.ElementalKind
	h.Defence = HumanDefence{Defence: uint16(out.Defence), Absorption: uint16(out.Absorption)}
	h.Defence.Protection[0], h.Defence.Resistance[0] = m.Defence.Protection[0], m.Defence.Resistance[0]
	for i := range out.Protection {
		h.Defence.Protection[i+1], h.Defence.Resistance[i+1] = uint16(out.Protection[i]), uint8(out.Resistance[i])
	}
	h.Health = uint16(min(humanSigned(h.Health), humanSigned(h.HealthMax)))
	h.Mana = uint16(max(0, min(humanSigned(h.Mana), humanSigned(h.ManaMax))))
	h.ManaFloor = h.ManaMax
	if h.HasOwner {
		h.ManaFloor = uint16(int32(h.ManaReservePercent) * humanSigned(h.ManaMax) / 100)
	}
	h.MoverSpeed = uint8(h.Speed)
	h.Load = uint16(max(0, humanSigned(h.Load)))
	return nil
}

// humanLoad is the carried load: own weight plus half the container's
// running weight, or 32000 from a running weight of 64000 (HERO-SIGHT-007).
func humanLoad(weight uint16, inventory int32) uint16 {
	if inventory >= 64000 {
		return 32000
	}
	return weight + uint16(inventory/2)
}

func (h HumanState) Hero() Hero {
	out := Hero{Body: humanSigned(h.Body), Reaction: humanSigned(h.Reaction), Mind: humanSigned(h.Mind), Spirit: humanSigned(h.Spirit)}
	for i := range out.Skill {
		out.Skill[i] = humanSigned(h.Attack.Skill[i])
	}
	return out
}

func (h HumanState) NativeMovementBase() int32 {
	return humanSigned(uint16(HumanBaseSpeed(humanSigned(h.Reaction), RiderTypeID(int32(h.TypeID)))) + h.Modifier.Speed)
}

// Derived projects stored current values. It does not run derive on LOAD.
// Cadence/reach still come from the unchanged weapon definition; its additive
// arithmetic is replaced by the retained live attack and defence blocks.
func (h HumanState) Derived(weapon *Weapon, rotation int32) Derived {
	c := FoldWeapon(Combat{AttackChargeTime: BareChargeTime, AttackRelaxTime: BareRelaxTime, Reach: 1}, weapon, humanSigned(h.Attack.Skill[0]))
	c.ToHit, c.DamageBase, c.DamageSpread = humanSigned(h.Attack.ToHit), int32(h.Attack.DamageBase), int32(h.Attack.DamageSpread)
	c.SecondBase, c.SecondSpread = h.Attack.SecondBase, h.Attack.SecondSpread
	c.Defence, c.Absorption, c.SkillSlot = humanSigned(h.Defence.Defence), humanSigned(h.Defence.Absorption), int32(h.Attack.Active)
	secondary := SecondaryDamage{Base: h.Attack.ElementalBase, Spread: h.Attack.ElementalSpread, Selector: h.Attack.ElementalKind}
	if secondary.Base != 0 || secondary.Spread != 0 {
		// HERO-DMG2-029: raw selectors 1..5 name a permutation of the
		// canonical Protection index (ElementalSelectorOrder), not raw-1.
		// ProjectionError is this state's own gate for a raw kind outside
		// 1..5 whenever base/spread is nonzero; an already-invalid pair that
		// reaches here regardless keeps its prior plain decrement rather than
		// gain a new out-of-range table read.
		if secondary.Selector >= 1 && secondary.Selector <= 5 {
			secondary.Selector = ElementalSelectorOrder[secondary.Selector-1]
		} else {
			secondary.Selector--
		}
	} else {
		secondary.Selector = 0
	}
	c.SecondaryDamage = secondary
	hero := h.Hero()
	d := Derived{Body: hero.Body, Reaction: hero.Reaction, Mind: hero.Mind, Spirit: hero.Spirit, Skill: hero.Skill,
		Experience: int32(h.Experience), HealthMax: humanSigned(h.HealthMax), ManaMax: humanSigned(h.ManaMax),
		Combat: c, Speed: h.NativeMovementBase(), SpeedModifier: humanSigned(h.Modifier.Speed), Sight: int32(h.Sight >> 8), Capacity: humanSigned(h.Capacity),
		HealthRegeneration: humanSigned(h.Modifier.HealthRegeneration), ManaRegeneration: humanSigned(h.Modifier.ManaRegeneration),
		RotationSpeed: rotation, SecondaryDamage: secondary}
	for i := range d.SkillXP {
		d.SkillXP[i] = int32(h.SkillXP[i])
	}
	for i := range d.Protection {
		d.Protection[i], d.Resistance[i] = humanSigned(h.Defence.Protection[i+1]), int32(h.Defence.Resistance[i+1])
	}
	return d
}
