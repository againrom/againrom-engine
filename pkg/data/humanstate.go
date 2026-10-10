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
	return humanFTOL(math.Pow(1.1, float64(humanSigned(h.Base.Skill[slot]))) * 200)
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
	xp, err := humanFTOL((math.Pow(1.1, float64(humanSigned(n.Attack.Skill[slot]))) - 1) * 1000)
	if err != nil {
		return h, err
	}
	nextXP := uint32(xp) + 1
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
	n.Load = n.Weight
	if n.InventoryWeight < 64000 {
		n.Load += uint16(n.InventoryWeight / 2)
	} else {
		n.Load = 32000
	}
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
	stats := []*uint16{&h.Body, &h.Reaction, &h.Mind, &h.Spirit}
	for i, p := range stats {
		*p = uint16(min(humanSigned(*p), 50+int32(h.Modifier.StatCap[i])))
	}
	body, reaction, mind, spirit := humanSigned(h.Body), humanSigned(h.Reaction), humanSigned(h.Mind), humanSigned(h.Spirit)
	pool := func(base int32, stat int32, multiplier float64) (uint16, error) {
		v, err := humanFTOL(float64(base) + math.Log(float64(int32(h.Experience))/5000+1)/math.Log(1.1)*multiplier)
		if err != nil {
			return 0, err
		}
		v, err = humanFTOL(float64(int16(v)) * (math.Pow(1.1, float64(stat))/100 + 1))
		return uint16(v), err
	}
	var err error
	healthMultiplier, manaMultiplier := float64(1), float64(2)
	if h.Fighter {
		healthMultiplier, manaMultiplier = 2, 1
	}
	h.HealthMax = uint16(body * int32(healthMultiplier))
	if h.HealthMax != 0 {
		h.HealthMax, err = pool(humanSigned(h.HealthMax), body, healthMultiplier)
		if err != nil {
			return err
		}
	}
	if h.ManaMax != 0 {
		h.ManaMax, err = pool(humanSigned(uint16(spirit*2)), spirit, manaMultiplier)
		if err != nil {
			return err
		}
	} else {
		h.Mana = 0
	}
	v, err := humanFTOL((float64(mind+reaction)/25 + 4) * 256)
	if err != nil {
		return err
	}
	h.Sight, h.Capacity = uint16(v), uint16(body*10+1)
	speed := reaction
	if reaction >= 12 {
		speed = reaction/5 + 12
	}
	if h.TypeID == 0x13 || h.TypeID == 0x15 {
		speed += 10
	}
	h.Speed, h.Load = uint16(speed), h.Weight
	if h.InventoryWeight < 64000 {
		h.Load += uint16(h.InventoryWeight / 2)
	} else {
		h.Load = 32000
	}
	// The modifier joins here rather than in foldModifier: nothing between
	// reads the speed, and the capacity compared is still the unmodified one.
	speed16, kept, ok := rules.HumanSpeed(int16(h.Speed), int16(h.Modifier.Speed), int16(h.Load), int16(h.Capacity))
	if !ok {
		return fmt.Errorf("Human derive would divide by zero capacity")
	}
	h.Speed, h.Modifier.Speed = uint16(speed16), uint16(kept)
	v, err = humanFTOL(math.Pow(1.1, float64(body)) / 20)
	if err != nil {
		return err
	}
	h.Attack.DamageBase, h.Attack.DamageSpread = uint8(v), uint8(v)
	v, err = humanFTOL((math.Pow(1.1, float64(body)) + math.Pow(1.1, float64(reaction))) / 5)
	if err != nil {
		return err
	}
	h.Attack.ToHit = uint16(v)
	for i := 1; i <= 5; i++ {
		// Word ADD precedes the signed clamps; General is in neither loop.
		level := h.Base.Skill[i] + h.Modifier.Attack.Skill[i]
		h.Attack.Skill[i] = uint16(max(0, min(h.skillLimit(), humanSigned(level))))
	}
	if h.Attack.Active != 0 {
		level := humanSigned(h.Attack.Skill[h.Attack.Active])
		h.Attack.ToHit += uint16(3 * level)
		h.Attack.DamageBase += uint8(level / 5)
	}
	h.Attack.SecondBase, h.Attack.SecondSpread = 0, 0
	h.Attack.ElementalBase, h.Attack.ElementalSpread = 0, 0
	h.Defence = HumanDefence{Defence: uint16(reaction / 3)}
	for i := 1; i <= 5; i++ {
		h.Defence.Protection[i] = uint16(spirit / 2)
	}
	h.foldModifier()
	h.Health = uint16(min(humanSigned(h.Health), humanSigned(h.HealthMax)))
	h.Mana = uint16(max(0, min(humanSigned(h.Mana), humanSigned(h.ManaMax))))
	h.ManaFloor = h.ManaMax
	if h.HasOwner {
		h.ManaFloor = uint16(int32(h.ManaReservePercent) * humanSigned(h.ManaMax) / 100)
	}
	h.MoverSpeed = uint8(h.Speed)
	h.Defence.Defence = uint16(max(0, humanSigned(h.Defence.Defence)))
	h.Defence.Absorption = uint16(max(0, humanSigned(h.Defence.Absorption)))
	h.Load = uint16(max(0, humanSigned(h.Load)))
	for i := 1; i <= 5; i++ {
		h.Defence.Protection[i] = uint16(max(0, min(100, spirit/2+70, humanSigned(h.Defence.Protection[i]))))
	}
	return nil
}

func (h *HumanState) foldModifier() {
	m := &h.Modifier
	h.Capacity += m.Capacity
	h.HealthMax += m.HealthMax
	h.ManaMax += m.ManaMax
	h.Sight += m.Sight
	h.Defence.Defence += m.Defence.Defence
	h.Defence.Absorption += m.Defence.Absorption
	for i := range h.Defence.Protection {
		h.Defence.Protection[i] += m.Defence.Protection[i]
		h.Defence.Resistance[i] += m.Defence.Resistance[i]
	}
	h.Attack.ToHit += m.Attack.ToHit
	h.Attack.DamageBase += m.Attack.DamageBase
	h.Attack.DamageSpread += m.Attack.DamageSpread
	h.Attack.SecondBase += m.Attack.SecondBase
	h.Attack.SecondSpread += m.Attack.SecondSpread
	h.Attack.ElementalBase += m.Attack.ElementalBase
	h.Attack.ElementalSpread += m.Attack.ElementalSpread
	h.Attack.ElementalKind = m.Attack.ElementalKind
}

func (h HumanState) Hero() Hero {
	out := Hero{Body: humanSigned(h.Body), Reaction: humanSigned(h.Reaction), Mind: humanSigned(h.Mind), Spirit: humanSigned(h.Spirit)}
	for i := range out.Skill {
		out.Skill[i] = humanSigned(h.Attack.Skill[i])
	}
	return out
}

func (h HumanState) NativeMovementBase() int32 {
	speed := humanSigned(h.Reaction)
	if speed >= 12 {
		speed = speed/5 + 12
	}
	if h.TypeID == 0x13 || h.TypeID == 0x15 {
		speed += 10
	}
	return humanSigned(uint16(speed) + h.Modifier.Speed)
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
