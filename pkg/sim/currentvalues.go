package sim

import (
	"fmt"
	"reflect"
	"slices"
)

type ActorNumericResidue struct {
	Field uint8
	Wire  uint32
	Lift  int64
}

type ActorMovementFallback struct {
	WireSpeed   int16
	NativeSpeed int32
}

// ActorSpeedSplit keeps a native Human's Speed and SpeedModifier where the
// speed word and modifier word a SAVE writes do not return them. Each pair
// applies only while its wire still matches the document.
type ActorSpeedSplit struct {
	SpeedWire, ModifierWire uint16
	Speed, Modifier         int32
}

// ActorRuntimeType retains a distinct gameplay type while its ordinary
// arithmetic type remains unchanged.
type ActorRuntimeType struct {
	Wire  uint16
	Value int32
}

// ActorRuntimeCoordinate keeps a native coordinate when ordinary runtime IDs
// need a unique, narrower namespace. An edited ordinary coordinate wins.
type ActorRuntimeCoordinate struct {
	Wire, Value uint32
}

// ActorDeadSourceHealth retains a virtual body's imported HP while its
// ordinary Health field writes the current HP. An edited ordinary HP wins.
type ActorDeadSourceHealth struct {
	Wire, Value int16
}

type ActorManaReserve struct {
	Wire, Value, OwnerKey uint32
}

type ActorClassAbsence struct {
	Wire     uint8
	Humanoid bool
}

// ActorValues contains policy and operands absent from ordinary SAV fields.
// Width lifts apply only while their wire anchor still matches the document.
type ActorValues struct {
	SourceBound, LoadPresent, MovementPresent                bool
	SourceClass                                              uint8
	EquipmentRuntimePresent                                  bool
	NativeClassPresent                                       bool                    `json:",omitempty"`
	MovementFallback                                         *ActorMovementFallback  `json:",omitempty"`
	SpeedSplit                                               *ActorSpeedSplit        `json:",omitempty"`
	RuntimeType                                              *ActorRuntimeType       `json:",omitempty"`
	RuntimeID                                                *ActorRuntimeCoordinate `json:",omitempty"`
	DeadSourceHealth                                         *ActorDeadSourceHealth  `json:",omitempty"`
	ClassAbsent                                              *ActorClassAbsence      `json:",omitempty"`
	RetainedDead                                             bool                    `json:",omitempty"`
	RetainedOwnerAbsent                                      *uint32                 `json:",omitempty"`
	RetainedCreditAbsent                                     *uint32                 `json:",omitempty"`
	CurrentTerminal                                          *CurrentTerminalActor   `json:",omitempty"`
	ManaReserve                                              *ActorManaReserve       `json:",omitempty"`
	PotionStats, PotionHeadroom                              [4]int32
	NativeGroup                                              *uint32 `json:",omitempty"`
	SuppressCorpseLoot                                       bool
	SeeInvisible                                             uint8
	AlwaysHits                                               bool
	DyingTime, XPValue, GoldChance, TreasureMin, TreasureMax int32
	GainsXP                                                  bool
	LegacyBook                                               bool
	LegacyBookWire                                           *[32]byte `json:",omitempty"`
	ExtraSpells                                              uint32
	UnitXP                                                   *[skillSlots]int32 `json:",omitempty"`
	AttackNoticeCell                                         uint16
	AttackNoticeScans                                        uint8
	Widths                                                   []ActorNumericResidue  `json:",omitempty"`
	EffectWidths                                             []EffectNumericResidue `json:"-"`
	Decay                                                    DecayStage             `json:"-"`
	Dwell                                                    uint16                 `json:"-"`
}

type actorNumericField struct {
	value  *int32
	width  uint8
	signed bool
}

func actorNumericFields(e *Entity) []actorNumericField {
	r := []actorNumericField{{&e.HP, 16, true}, {&e.MaxHP, 16, false}, {&e.Mana, 16, false}, {&e.MaxMana, 16, false},
		{&e.Speed, 16, true}, {&e.Reaction, 16, true}, {&e.Mind, 16, true}, {&e.Spirit, 16, true},
		{&e.ToHit, 16, true}, {&e.Defence, 16, true}, {&e.Absorption, 16, true}, {&e.DamageBase, 8, false}, {&e.DamageSpread, 8, false},
		{&e.HealthRegenPeriod, 16, true}, {&e.ManaRegenPeriod, 16, true}, {&e.HealthRegeneration, 16, true}, {&e.ManaRegeneration, 16, true},
		{&e.AttackCharge, 8, false}, {&e.AttackRelax, 8, false}, {&e.Load, 16, true}, {&e.Capacity, 16, true}, {&e.TypeID, 16, false}}
	for i := range e.Skill {
		r = append(r, actorNumericField{&e.Skill[i], 16, true})
	}
	for i := range e.Protection {
		r = append(r, actorNumericField{&e.Protection[i], 16, true})
	}
	return r
}

func (e Entity) Values() ActorValues {
	v := ActorValues{SourceBound: e.SourceBinding.Class != 0, LoadPresent: e.ActorLoad.Present, SourceClass: e.ActorLoad.Source.Class,
		NativeClassPresent:      e.NativeClass.Present,
		EquipmentRuntimePresent: e.ActorLoad.Source.EquipmentRuntimePresent, MovementPresent: e.HumanMovement.Present,
		PotionStats: e.PotionStats, PotionHeadroom: e.PotionHeadroom, SuppressCorpseLoot: e.SuppressCorpseLoot, SeeInvisible: e.SeeInvisible,
		AlwaysHits: e.AlwaysHits, DyingTime: e.DyingTime, XPValue: e.XPValue, GoldChance: e.GoldChance, TreasureMin: e.TreasureMin, TreasureMax: e.TreasureMax,
		GainsXP: e.GainsXP, LegacyBook: e.Book.State == BookLegacy, ExtraSpells: e.KnownSpells & ^uint32(0x1ffffffe),
		AttackNoticeCell: e.attackNotice.Cell, AttackNoticeScans: e.attackNotice.Scans, Decay: e.Decay, Dwell: e.Dwell}
	if !e.Humanoid && e.SkillXP != ([skillSlots]int32{}) {
		xp := e.SkillXP
		v.UnitXP = &xp
	}
	if s := e.ActorLoad.Source; s.Class != 0 && int32(s.TypeID) != e.TypeID {
		v.RuntimeType = &ActorRuntimeType{Wire: s.TypeID, Value: e.TypeID}
	}
	if e.HumanMovement.Present && e.Speed != int32(e.HumanMovement.RawSpeed) {
		v.MovementFallback = &ActorMovementFallback{e.HumanMovement.RawSpeed, e.Speed}
	}
	if e.Humanoid && e.ActorLoad.Source.Class == 0 {
		word, modifier := uint16(e.SpeedWord()), e.SpeedModifierWord()
		if word != uint16(e.Speed) || int32(int16(modifier)) != e.SpeedModifier {
			v.SpeedSplit = &ActorSpeedSplit{word, modifier, e.Speed, e.SpeedModifier}
		}
	}
	for i, field := range actorNumericFields(&e) {
		if field.value == &e.TypeID && v.RuntimeType != nil {
			continue
		}
		mask := uint32(1)<<field.width - 1
		wire := uint32(*field.value) & mask
		decoded := int64(wire)
		if field.signed && wire&(uint32(1)<<(field.width-1)) != 0 {
			decoded -= int64(1) << field.width
		}
		if lift := int64(*field.value) - decoded; lift != 0 {
			v.Widths = append(v.Widths, ActorNumericResidue{uint8(i), wire, lift})
		}
	}
	return v
}

// CurrentActorHealthFromWords resolves the two current pool fields used by
// provisional actor admission, where a width operand can turn a word that
// reads as dead into a living full value. Callers must have an exact current
// binding; original files without current metadata keep their ordinary
// admission. It owns no state and decodes exactly as restoreValues does.
func CurrentActorHealthFromWords(hp, maximum uint16, widths []ActorNumericResidue) (int32, int32, error) {
	values := [2]int32{int32(int16(hp)), int32(maximum)}
	signed := [2]bool{true, false}
	seen := [2]bool{}
	for _, x := range widths {
		if x.Field > 1 {
			continue
		}
		if seen[x.Field] {
			return 0, 0, fmt.Errorf("sim: invalid current numeric residue")
		}
		seen[x.Field] = true
		next, err := restoreNumericResidue(values[x.Field], 16, signed[x.Field], x)
		if err != nil {
			return 0, 0, err
		}
		values[x.Field] = next
	}
	return values[0], values[1], nil
}

// restoreNumericResidue applies one width operand while the value's low bits
// still equal the operand's wire anchor. The result is the canonical decode of
// those low bits plus the lift, so it does not depend on the high bits already
// held and applying the operand to an already lifted value changes nothing.
func restoreNumericResidue(value int32, width uint8, signed bool, x ActorNumericResidue) (int32, error) {
	mask := uint32(1)<<width - 1
	if x.Wire > mask || x.Lift == 0 || x.Lift%(int64(1)<<width) != 0 {
		return 0, fmt.Errorf("sim: invalid current numeric lift")
	}
	if uint32(value)&mask != x.Wire {
		return value, nil
	}
	decoded := int64(x.Wire)
	if signed && x.Wire&(uint32(1)<<(width-1)) != 0 {
		decoded -= int64(1) << width
	}
	next := decoded + x.Lift
	if next < -1<<31 || next > 1<<31-1 {
		return 0, fmt.Errorf("sim: current numeric residue overflows")
	}
	return int32(next), nil
}

func (e *Entity) restoreValues(v ActorValues) error {
	if v.ClassAbsent != nil {
		p := v.ClassAbsent
		if !v.RetainedDead || v.SourceBound || v.SourceClass != 0 || p.Wire < 1 || p.Wire > 3 || p.Humanoid == (p.Wire != 1) {
			return fmt.Errorf("sim: invalid absent current actor class")
		}
		if e.SourceBinding.Class == p.Wire {
			e.Humanoid = p.Humanoid
		}
	}
	if v.RuntimeID != nil {
		if !v.SourceBound || e.SourceBinding.Class == 0 || v.RuntimeID.Wire > 65535 || v.RuntimeID.Wire == v.RuntimeID.Value {
			return fmt.Errorf("sim: invalid current runtime coordinate")
		}
		if e.SourceBinding.RuntimeID == v.RuntimeID.Wire {
			e.SourceBinding.RuntimeID = v.RuntimeID.Value
		}
	}
	fields := actorNumericFields(e)
	seen := map[uint8]bool{}
	for _, x := range v.Widths {
		if int(x.Field) >= len(fields) || seen[x.Field] {
			return fmt.Errorf("sim: invalid current numeric residue")
		}
		seen[x.Field] = true
		field := fields[x.Field]
		if v.RuntimeType != nil && field.value == &e.TypeID {
			return fmt.Errorf("sim: runtime type has conflicting operands")
		}
		next, err := restoreNumericResidue(*field.value, field.width, field.signed, x)
		if err != nil {
			return err
		}
		*field.value = next
	}
	if v.RuntimeType != nil {
		if v.SourceClass == 0 || v.RuntimeType.Value == int32(v.RuntimeType.Wire) {
			return fmt.Errorf("sim: invalid distinct runtime type operand")
		}
		if e.ActorLoad.Source.TypeID == v.RuntimeType.Wire {
			e.TypeID = v.RuntimeType.Value
		}
	}
	if v.NativeGroup != nil {
		e.Group = *v.NativeGroup
	} else {
		e.Group = e.SourceBinding.GroupSelector
	}
	class := NativeClass{}
	if v.NativeClassPresent {
		if v.SourceClass != 0 {
			return fmt.Errorf("sim: native class lacks an ordinary Human")
		}
		switch {
		case e.ActorLoad.Source.Class == 2:
			class = NativeClass{Present: true, Fighter: e.ActorLoad.Source.Fighter}
		case e.ActorLoad.Source.Class == 0 && e.NativeClass.Present:
			class = e.NativeClass
		case v.ClassAbsent != nil && e.SourceBinding.Class != 0:
			class = NativeClass{Present: true, Fighter: e.SourceBinding.ClassFlags&4 == 0}
		default:
			return fmt.Errorf("sim: native class lacks an ordinary Human")
		}
	}
	e.NativeClass = class
	if !v.SourceBound {
		e.SourceBinding = SourceBinding{}
	}
	if err := e.restoreSpeedSplit(v); err != nil {
		return err
	}
	if !v.LoadPresent {
		e.ActorLoad = ActorLoad{}
	} else {
		if v.SourceClass == 0 {
			e.ActorLoad.Source = SourceActor{}
		} else {
			if e.ActorLoad.Source.Class != v.SourceClass {
				return fmt.Errorf("sim: current actor arithmetic class differs")
			}
			e.ActorLoad.Source.EquipmentRuntimePresent = v.EquipmentRuntimePresent
		}
	}
	if v.ManaReserve != nil {
		if v.ManaReserve.Wire == v.ManaReserve.Value || v.ManaReserve.OwnerKey == 0 || v.SourceClass == 0 {
			return fmt.Errorf("sim: invalid distinct current mana reserve")
		}
		if e.ActorLoad.Source.HasOwner && e.ActorLoad.Source.ManaReservePercent == v.ManaReserve.Wire {
			e.ActorLoad.Source.ManaReservePercent = v.ManaReserve.Value
		}
	}
	if !v.MovementPresent {
		e.HumanMovement = HumanMovement{}
	} else {
		if v.MovementFallback != nil && v.MovementFallback.WireSpeed == e.HumanMovement.RawSpeed {
			e.Speed = v.MovementFallback.NativeSpeed
		}
		e.HumanMovement.NativeSpeed, e.HumanMovement.Load, e.HumanMovement.Capacity = e.Speed, e.Load, e.Capacity
	}
	e.PotionStats, e.PotionHeadroom = v.PotionStats, v.PotionHeadroom
	e.SuppressCorpseLoot, e.SeeInvisible, e.AlwaysHits, e.DyingTime = v.SuppressCorpseLoot, v.SeeInvisible, v.AlwaysHits, v.DyingTime
	e.XPValue, e.GoldChance, e.TreasureMin, e.TreasureMax, e.GainsXP = v.XPValue, v.GoldChance, v.TreasureMin, v.TreasureMax, v.GainsXP
	mode := BookPresent
	if v.LegacyBook {
		mode = BookLegacy
	}
	book, known, err := RestoreBookMode(e.Book, e.KnownSpells, mode, v.LegacyBookWire, v.ExtraSpells)
	if err != nil {
		return err
	}
	e.Book, e.KnownSpells = book, known
	if v.UnitXP != nil {
		if e.Humanoid {
			return fmt.Errorf("sim: Human XP duplicates ordinary fields")
		}
		e.SkillXP = *v.UnitXP
	}
	e.Decay, e.Dwell = v.Decay, v.Dwell
	e.attackNotice = attackNotice{Cell: v.AttackNoticeCell, Scans: v.AttackNoticeScans}
	return nil
}

func (w *World) restoreActorValues(values map[EntityID]ActorValues) error {
	if len(values) < len(w.entities) {
		return fmt.Errorf("sim: current value population differs: %d bindings for %d actors", len(values), len(w.entities))
	}
	next := slices.Clone(w.entities)
	dead := slices.Clone(w.originalDead)
	held := make(map[EntityID]int, len(dead))
	for i, d := range dead {
		held[d.ID] = i
	}
	live := make(map[EntityID]bool, len(next))
	for i := range next {
		v, ok := values[next[i].ID]
		if !ok {
			return fmt.Errorf("sim: current actor value binding is absent for entity %d", next[i].ID)
		}
		live[next[i].ID] = true
		if v.DeadSourceHealth != nil {
			return fmt.Errorf("sim: current dead source health belongs to a live actor")
		}
		if v.RetainedDead {
			if _, retained := held[next[i].ID]; !retained {
				return fmt.Errorf("sim: current dead manager is absent")
			}
		}
		if _, retained := held[next[i].ID]; retained && !v.SourceBound {
			// This coordinate belongs to the existing dead manager. The
			// Entity's absent source binding remains absent.
			v.RuntimeID = nil
		}
		if err := next[i].restoreValues(v); err != nil {
			return err
		}
	}
	for id, v := range values {
		index, retained := held[id]
		if v.CurrentTerminal != nil {
			terminal := *v.CurrentTerminal
			v.CurrentTerminal = nil
			if terminal.ID != id || live[id] || !w.hasCurrentTerminalActor(terminal) || !reflect.DeepEqual(v, ActorValues{}) {
				return fmt.Errorf("sim: current terminal value has no exact retired actor binding")
			}
			continue
		}
		if v.RetainedCreditAbsent != nil {
			if !retained || !live[id] || !v.RetainedDead || *v.RetainedCreditAbsent == 0 {
				return fmt.Errorf("sim: invalid retained credit absence")
			}
			if dead[index].Source.References[4] == *v.RetainedCreditAbsent {
				next[indexOfEntity(next, id)].clearKillCredit()
			}
		}
		if v.RetainedOwnerAbsent != nil {
			if !retained || !live[id] || !v.RetainedDead || *v.RetainedOwnerAbsent == 0 {
				return fmt.Errorf("sim: invalid retained owner absence")
			}
			if dead[index].Source.OwnerKey == *v.RetainedOwnerAbsent {
				dead[index].Source.OwnerKey = 0
			}
		}
		if !live[id] {
			coordinate := v.RuntimeID
			health := v.DeadSourceHealth
			v.RuntimeID = nil
			v.DeadSourceHealth = nil
			if !retained || (coordinate == nil && health == nil) || !reflect.DeepEqual(v, ActorValues{}) {
				return fmt.Errorf("sim: current value has no exact retained actor binding")
			}
			v.RuntimeID = coordinate
			v.DeadSourceHealth = health
		}
		if !retained {
			continue
		}
		r := &dead[index]
		if p := v.DeadSourceHealth; p != nil {
			if live[id] || r.Source.MapUnitID != 0 || r.Source.State.Stage != r.terminal.Stage ||
				r.terminal.Stage < 2 || r.terminal.Stage >= 5 || p.Wire >= 0 || p.Value >= 0 || p.Value <= p.Wire {
				return fmt.Errorf("sim: invalid virtual dead source health")
			}
			if r.Source.State.HP == p.Wire {
				r.Source.State.HP = p.Value
				if err := deadSourceFault(r.Source, w.bounds); err != nil {
					return err
				}
				if !validHeldDeadContinuation(r.Source.State, r.terminal) {
					return fmt.Errorf("sim: virtual dead source health disagrees with its current body")
				}
			}
		}
		if v.RuntimeID == nil {
			continue
		}
		p := v.RuntimeID
		if p.Wire == 0 || p.Wire > 65535 || p.Value == 0 || p.Wire == p.Value || r.Source.State.Stage == 5 || r.terminal.Stage == 5 {
			return fmt.Errorf("sim: invalid retained runtime coordinate")
		}
		if r.Source.State.RuntimeID == p.Wire {
			r.Source.State.RuntimeID = p.Value
		}
		if r.terminal.Stage != 0 && r.terminal.RuntimeID == p.Wire {
			r.terminal.RuntimeID = p.Value
		}
		if err := deadSourceFault(r.Source, w.bounds); err != nil {
			return err
		}
		if r.terminal.Stage != 0 && !validHeldDeadContinuation(r.Source.State, r.terminal) {
			return fmt.Errorf("sim: retained runtime coordinate disagrees with its current body")
		}
	}
	w.entities, w.originalDead = next, dead
	return nil
}

// restoreSpeedSplit reads a native Human's modifier from its ordinary record's
// modifier speed word, then applies the split operand while both wires match.
// A SAV written before the split carries no operand: its speed word is Speed
// and its modifier word SpeedModifier.
func (e *Entity) restoreSpeedSplit(v ActorValues) error {
	native := v.SourceClass == 0 && e.Humanoid && e.ActorLoad.Source.Class == 2
	if v.SpeedSplit != nil && !native {
		return fmt.Errorf("sim: speed split lacks a native Human")
	}
	if !native {
		return nil
	}
	wire := uint16(e.ActorLoad.Source.Modifier[4]) | uint16(e.ActorLoad.Source.Modifier[5])<<8
	e.SpeedModifier = int32(int16(wire))
	if s := v.SpeedSplit; s != nil {
		if uint16(e.Speed) == s.SpeedWire {
			e.Speed = s.Speed
		}
		if wire == s.ModifierWire {
			e.SpeedModifier = s.Modifier
		}
	}
	return nil
}
