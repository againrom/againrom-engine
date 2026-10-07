package sim

import (
	"encoding/binary"
	"fmt"

	"againrom/pkg/rules"
)

// SourceActor is the source-derived actor's integer runtime basis. Stats and
// the four blocks keep their serialized widths, including unmaintained words.
// Class 1 is Unit; 2 is Human or Humanoid (the same vt+50). Class zero is the
// old-native compatibility arm, never a test of whether a saved word is zero.
type SourceActor struct {
	Class                            uint8
	Stats                            [14]uint16
	Attack, Base                     [24]byte
	Defence                          [22]byte
	Modifier                         [64]byte
	SkillXP                          [6]uint32
	Experience                       uint32
	ManaFloor, Sight, TypeID         uint16
	MoverSpeed                       uint8
	Fighter, HasSpellbook, HasOwner  bool
	ManaReservePercent               uint32
	Reach, AttackCharge, AttackRelax uint8
	EquipmentRuntimePresent          bool
}

// SourceDerive is pure arithmetic over this value and the signed container
// accumulator. The loader supplies the versioned rule, not a closure over game,
// UI, table, time, or another world. It is rebound before a native world can
// execute source producers. It is never run just to read or load an actor.
type SourceDerive func(SourceActor, int32, Rules) (SourceActor, error)

func (w *World) BindSourceDerive(rule SourceDerive) { w.sourceDerive = rule }

func (s SourceActor) Validate() error {
	if s.Class > 2 || s.Class == 0 && s != (SourceActor{}) {
		return fmt.Errorf("sim: invalid source actor class/residue")
	}
	if s.Class != 0 && s.Attack[16] > 5 {
		return fmt.Errorf("sim: invalid source actor active skill")
	}
	if !s.EquipmentRuntimePresent && (s.Reach != 0 || s.AttackCharge != 0 || s.AttackRelax != 0) {
		return fmt.Errorf("sim: source equipment runtime has no presence")
	}
	return nil
}

// sourceNow overlays pools and progression that the combat core owns. No
// inventory sum, current load or capacity is reconstructed here.
func (e Entity) SourceNow() SourceActor {
	s := e.ActorLoad.Source
	if s.Class == 0 {
		return s
	}
	s.Stats[4], s.Stats[5], s.Stats[6], s.Stats[7] = uint16(e.HumanMovement.RawSpeed), uint16(e.ActorLoad.OwnWeight), uint16(e.Load), uint16(e.Capacity)
	if !e.HumanMovement.Present {
		s.Stats[4] = uint16(e.Speed)
	}
	s.Stats[8], s.Stats[9], s.Stats[11], s.Stats[12] = uint16(e.HP), uint16(e.MaxHP), uint16(e.Mana), uint16(e.MaxMana)
	// Death/revival owns a direct live defence shift, independently of the
	// modifier block. A later equipment virtual must see that current word.
	binary.LittleEndian.PutUint16(s.Defence[:], uint16(e.Defence))
	binary.LittleEndian.PutUint16(s.Defence[2:], uint16(e.Absorption))
	if s.EquipmentRuntimePresent {
		s.Reach, s.AttackCharge, s.AttackRelax = e.Reach, uint8(e.AttackCharge), uint8(e.AttackRelax)
	}
	for i := range s.SkillXP {
		s.SkillXP[i] = uint32(e.SkillXP[i])
	}
	return s
}

func (e *Entity) setCurrentHealth(value int32) {
	e.HP = value
	if e.ActorLoad.Source.Class != 0 {
		e.ActorLoad.Source.Stats[8] = uint16(value)
	}
}

func (e *Entity) setCurrentMana(value int32) {
	e.Mana = value
	if e.ActorLoad.Source.Class != 0 {
		e.ActorLoad.Source.Stats[11] = uint16(value)
	}
}

func (e *Entity) setCurrentDefence(value int32) {
	e.Defence = value
	if e.ActorLoad.Source.Class != 0 {
		binary.LittleEndian.PutUint16(e.ActorLoad.Source.Defence[:], uint16(value))
	}
}

func (w *World) sourceMutationReady(i int) bool {
	e := w.entities[i]
	if !e.ActorLoad.Present {
		return true
	}
	if e.Capacity == 0 {
		return false
	}
	return w.sourceDeriveReady(i)
}

func (w *World) sourceDeriveReady(i int) bool {
	e := w.entities[i]
	if e.ActorLoad.Source.Class != 2 {
		return true
	}
	if w.sourceDerive == nil {
		return false
	}
	// Refuse an unsupported numeric domain before any item, sack, spell or
	// purse writer runs. Valid source attributes only decrease at this pass.
	_, err := w.sourceDerive(e.SourceNow(), e.ActorLoad.Accumulator, w.rules)
	return err == nil
}

// sourcePotion takes the live attribute and current-pool arms followed by the
// generic effect tail's vt+50. Permanent attributes do not increment the cap
// modifier bytes; the derive's own cap can consume a potion without a gain.
func (w *World) sourcePotion(i int, kind uint8, amount int32) bool {
	e := &w.entities[i]
	s := e.SourceNow()
	if s.Class != 2 || w.sourceDerive == nil {
		return false
	}
	if int16(binary.LittleEndian.Uint16(s.Modifier[4:])) > 24 {
		binary.LittleEndian.PutUint16(s.Modifier[4:], 0)
	}
	switch kind {
	case 2, 3, 4, 5:
		slot := [...]int{0, 0, 0, 2, 1, 3}[kind]
		s.Stats[slot] = uint16(min(100, int32(int16(s.Stats[slot]))+amount))
	case 6:
		s.Stats[8] = uint16(potionPool(e.HP, e.MaxHP, amount))
	case 9:
		if !s.Fighter {
			s.Stats[11] = uint16(potionPool(e.Mana, e.MaxMana, amount))
		}
	default:
		return false
	}
	n, err := w.sourceDerive(s, e.ActorLoad.Accumulator, w.rules)
	if err != nil {
		return false
	}
	w.finishSourceDerive(i, n)
	return true
}

func (w *World) sourceEffect(i int, kind EffectKind, amount int32) (int32, bool) {
	var key uint8
	switch kind {
	case EffectHealth:
		key = 6
	case EffectHealthRegeneration:
		key = 8
	case EffectManaRegeneration:
		key = 11
	case EffectAbsorption:
		key = 16
	case EffectSpeed:
		key = 17
	case EffectScanRange:
		key = 19
	case EffectProtectionFire, EffectProtectionWater, EffectProtectionAir, EffectProtectionEarth:
		key = 21 + uint8(kind-EffectProtectionFire)
	case EffectInvisible:
		key = 38
	case EffectBless:
		key = 39
	case EffectCurse:
		key = 40
	default:
		return 0, false
	}
	// Keep the existing non-resurrection rule for spell effects. Equipment
	// virtuals use their direct source dispatcher instead of this admission.
	if kind == EffectHealth && amount > 0 && !w.entities[i].Alive() {
		return 0, true
	}
	n := w.sourceMutationCopy(i)
	if !n.sourceItemEffect(i, ItemEffect{Kind: key, Operand: uint32(amount)}, 1) {
		return 0, false
	}
	*w = n
	return amount, true
}

func (w *World) sourceSkillAward(i int, slot int32, gain int32) bool {
	e := &w.entities[i]
	s := e.SourceNow()
	base := int32(int16(binary.LittleEndian.Uint16(s.Base[2+2*slot:])))
	s.SkillXP[slot] += uint32(gain)
	s.Experience += uint32(gain)
	raised := int32(s.SkillXP[slot]) > w.rules.SkillXP(base)
	if raised {
		binary.LittleEndian.PutUint16(s.Base[2+2*slot:], uint16(base+1))
		n, err := w.sourceDerive(s, e.ActorLoad.Accumulator, w.rules)
		if err != nil {
			return false
		}
		w.finishSourceDerive(i, n)
		return true
	}
	// The no-raise tail restores only class skills from base plus modifier.
	for j := 1; j <= 5; j++ {
		level := binary.LittleEndian.Uint16(s.Base[2+2*j:]) + binary.LittleEndian.Uint16(s.Modifier[20+2*j:])
		level = uint16(max(0, min(w.rules.SkillCap(), int32(int16(level)))))
		binary.LittleEndian.PutUint16(s.Attack[2+2*j:], level)
		e.Skill[j] = int32(level)
	}
	e.ActorLoad.Source = s
	// The lift adds to the source terms, so start from them as publishSource
	// does; an award that raises nothing then leaves the combat block fixed.
	e.ToHit = int32(int16(binary.LittleEndian.Uint16(s.Attack[:])))
	e.DamageBase = int32(s.Attack[14])
	e.liftEffectiveSkills(w.rules)
	e.SkillXP[slot] = int32(s.SkillXP[slot])
	return false
}

func (w *World) deriveSource(i int) bool {
	e := &w.entities[i]
	s := e.SourceNow()
	if s.Class == 0 {
		return false
	}
	if s.Class == 1 {
		s.ManaFloor = s.Stats[12]
		if s.HasOwner {
			s.ManaFloor = uint16(int32(int16(s.Stats[12])) * int32(s.ManaReservePercent) / 100)
		}
		e.ActorLoad.Source = s
		return true
	}
	if w.sourceDerive == nil {
		return false
	}
	n, err := w.sourceDerive(s, e.ActorLoad.Accumulator, w.rules)
	if err != nil {
		return false
	}
	w.finishSourceDerive(i, n)
	return true
}

// Active-skill terms of the attack block: three times the level joins the
// to-hit and a fifth of it the damage base (pkg/data restores the same pair).
const (
	skillToHitPerLevel = 3
	skillDamageDivisor = 5
)

// liftEffectiveSkills raises the live skill levels of a source-backed Human
// from the original domain to base plus bonus. The source block holds the
// level the original writes: base plus bonus clamped to the training cap. A
// slot whose stored level sits at that clamp and whose base plus bonus is
// higher takes the higher level, and the active slot's to-hit and damage base
// take the extra terms that level adds. The source block itself is not
// touched, so a SAVE writes the original's bytes and the next LOAD lifts again.
func (e *Entity) liftEffectiveSkills(r Rules) {
	s := &e.ActorLoad.Source
	if s.Class != 2 {
		return
	}
	for j := 1; j <= 5; j++ {
		stored := int32(int16(binary.LittleEndian.Uint16(s.Attack[2+2*j:])))
		if stored != rules.ROM1SkillCap && stored != r.SkillCap() {
			continue
		}
		base := int32(int16(binary.LittleEndian.Uint16(s.Base[2+2*j:])))
		bonus := int32(int16(binary.LittleEndian.Uint16(s.Modifier[20+2*j:])))
		effective := r.EffectiveSkill(base, bonus)
		if effective <= stored {
			continue
		}
		e.Skill[j] = effective
		if int(s.Attack[16]) == j {
			e.ToHit += skillToHitPerLevel * (effective - stored)
			e.DamageBase += effective/skillDamageDivisor - stored/skillDamageDivisor
		}
	}
}

// liftActiveSkillTerms adds to the to-hit and damage base of a source-backed
// Human the terms its lifted active skill carries above the source level. The
// saved profile holds those two numbers in the original domain, so a profile
// import that follows the skill lift restores them here.
func (e *Entity) liftActiveSkillTerms() {
	s := &e.ActorLoad.Source
	j := int(e.XPSlot)
	if s.Class != 2 || j < 1 || j > 5 {
		return
	}
	stored := int32(int16(binary.LittleEndian.Uint16(s.Attack[2+2*j:])))
	if e.Skill[j] <= stored {
		return
	}
	e.ToHit += skillToHitPerLevel * (e.Skill[j] - stored)
	e.DamageBase += e.Skill[j]/skillDamageDivisor - stored/skillDamageDivisor
}

func (w *World) finishSourceDerive(i int, s SourceActor) {
	w.publishSourceDerived(i, s)
	RefreshBook(w.rules, &w.entities[i], w.spells)
	w.refreshSavedBookRoots(i)
}

func (w *World) publishSourceDerived(i int, s SourceActor) {
	// Retire the construction-time current-profile cache, not this full
	// source basis. The legacy tag name predates source-backed producers.
	w.entities[i].retireCurrentProfile()
	w.publishSource(i, s)
}

func (w *World) publishSource(i int, s SourceActor) {
	e := &w.entities[i]
	e.NativeTraining = NativeTraining{}
	e.NativeClass = NativeClass{}
	e.ActorLoad.Source = s
	e.RotationSpeed = int32(s.MoverSpeed)
	// Keep the turn rate current in a frozen imported mover.
	if m := w.motionFor(e.ID); m != nil {
		m.Mover[10] = s.MoverSpeed
	}
	// Settle an active turn when a derive removes its rate (MOVE-TURN-044).
	// ROM1's zero-rate lifecycle remains Unknown.
	if e.RotationSpeed <= 0 && e.Turning() {
		e.Facing = e.DesiredFacing
		e.clearTurn()
	}
	if s.EquipmentRuntimePresent {
		e.Reach, e.AttackCharge, e.AttackRelax = s.Reach, int32(s.AttackCharge), int32(s.AttackRelax)
	}
	e.Load, e.Capacity, e.Speed = int32(int16(s.Stats[6])), int32(int16(s.Stats[7])), int32(int16(s.Stats[4]))
	e.HumanMovement = HumanMovement{Present: true, RawSpeed: int16(s.Stats[4]), NativeSpeed: e.Speed, Load: e.Load, Capacity: e.Capacity}
	e.HP, e.MaxHP, e.Mana, e.MaxMana = int32(int16(s.Stats[8])), int32(int16(s.Stats[9])), int32(int16(s.Stats[11])), int32(int16(s.Stats[12]))
	e.HealthRegenPeriod, e.ManaRegenPeriod = int32(int16(s.Stats[10])), int32(int16(s.Stats[13]))
	e.Reaction, e.Mind, e.Spirit = int32(int16(s.Stats[1])), int32(int16(s.Stats[2])), int32(int16(s.Stats[3]))
	e.ScanRange = uint8(min(s.Sight>>8, 255))
	e.HealthRegeneration, e.ManaRegeneration = int32(int16(binary.LittleEndian.Uint16(s.Modifier[10:]))), int32(int16(binary.LittleEndian.Uint16(s.Modifier[14:])))
	e.ToHit = int32(int16(binary.LittleEndian.Uint16(s.Attack[:])))
	e.DamageBase, e.DamageSpread, e.SecondBase, e.SecondSpread = int32(s.Attack[14]), int32(s.Attack[15]), s.Attack[17], s.Attack[18]
	e.Defence, e.Absorption = int32(int16(binary.LittleEndian.Uint16(s.Defence[:]))), int32(int16(binary.LittleEndian.Uint16(s.Defence[2:])))
	e.XPSlot = s.Attack[16]
	for j := range e.Skill {
		e.Skill[j] = int32(int16(binary.LittleEndian.Uint16(s.Attack[2+2*j:])))
		e.SkillXP[j] = int32(s.SkillXP[j])
	}
	for j := range e.Protection {
		e.Protection[j] = int32(int16(binary.LittleEndian.Uint16(s.Defence[6+2*j:])))
		e.Resistance[j] = s.Defence[17+j]
	}
	e.liftEffectiveSkills(w.rules)
	if s.Attack[19] == 0 && s.Attack[20] == 0 {
		e.SecondaryDamage = SecondaryDamage{}
	} else {
		// Current source selectors are one-based indexes into the retained
		// protection block, as on the original current-profile import door.
		if s.Attack[21] >= 1 && s.Attack[21] <= 5 {
			e.SecondaryDamage = SecondaryDamage{s.Attack[19], s.Attack[20], s.Attack[21] - 1}
		}
	}
}
