package mapload

import (
	"encoding/binary"
	"fmt"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// ConstructActorBasis captures the actual fresh spawn, with the exact table
// selected by startMission. Current combat values, maintained base skills and
// numeric modifiers have separate owners. This runs once before gameplay;
// SAVE never calls a definition lookup or reconstructs a basis from an ALM.
// humanRow, when positive, is the caller's own resolved Humans row for a
// party Human with no placement; the class search below cannot tell a
// companion's PC_ row from the first creature row sharing its TypeID.
func ConstructActorBasis(e sim.Entity, member PartyMember, placement *alm.Unit, t *Table, key, runtime uint32, human *data.HumanState, humanRow int) (sim.Entity, string, error) {
	if t == nil {
		return e, "", fmt.Errorf("actor constructor lacks definitions")
	}
	row, body, name := 0, member.Hero.Body, member.Name
	face := uint8(member.FigureFace)
	if placement != nil {
		r := Resolve(*placement, t)
		if r.Index < 0 || r.Index > 255 {
			return e, "", fmt.Errorf("actor %d definition is unavailable", e.ID)
		}
		row = r.Index
		if !e.Humanoid {
			face = uint8(placement.ClassSubID)
		}
		if !e.Humanoid {
			d, err := data.NewUnitDef(t.Units.EntryName(row), t.Units.EntryParams(row))
			if err != nil {
				return e, "", err
			}
			body, name = d.Body, t.Units.EntryName(row)
		} else if name == "" {
			name = t.Humans.EntryName(row)
		}
	} else if e.Humanoid {
		// TypeID is many-to-one over Human rows. The starting hero keeps his
		// own chargen row and face, independently of the drawable class.
		if member.StartingHero {
			fig := data.FigureDir(member.FigureDir)
			if _, n, ok := data.ChargenBase(t.Humans, fig.Mage(), fig.Female()); ok && n > 0 && n <= 255 {
				row = n
			}
		}
		if row == 0 && humanRow > 0 {
			row = humanRow
		}
		if row == 0 {
			row = data.FindHumanByType(t.Humans, e.TypeID)
			if member.Class > 0 {
				if n := data.FindHumanByType(t.Humans, member.Class); n >= 0 {
					row = n
				}
			}
		}
		if row < 0 || row > 255 {
			return e, "", fmt.Errorf("party actor %d definition is unavailable", e.ID)
		}
	} else if member.ID == "" {
		// A creature neither placed nor in the party was raised by Control
		// Spirit, which constructs it from the Units row named Ghost
		// (MAGIC-SING-019). SAV names that row, its face and its name.
		if n, d, ok := ghostRow(t.units()); ok && n <= 255 && d.TypeID == e.TypeID {
			row, face, body, name = n, uint8(d.Face), d.Body, t.units().EntryName(n)
		}
	}
	class, basis := sim.GeneratedUnitBinding, uint8(1)
	if e.Humanoid {
		class, basis = sim.GeneratedHumanBinding, 2
	}
	// The initial Document supplies the zero deadline. The common clock
	// importer binds it once, atomically with motion, before the first tick.
	e.ActionClock = sim.ActionClock{}
	h := data.HumanState{Body: uint16(body), Reaction: uint16(e.Reaction), Mind: uint16(e.Mind), Spirit: uint16(e.Spirit),
		Speed: uint16(e.Speed), Capacity: uint16(e.Capacity), Health: uint16(e.HP), HealthMax: uint16(e.MaxHP), Mana: uint16(e.Mana), ManaMax: uint16(e.MaxMana),
		HealthPeriod: uint16(e.HealthRegenPeriod), ManaPeriod: uint16(e.ManaRegenPeriod), Sight: uint16(e.ScanRange) << 8,
		Fighter: !e.Humanoid || member.Profile.Fighter, HasSpellbook: e.Book.WirePresent(e.KnownSpells), HasOwner: e.Owner != 0, ManaReservePercent: 95, TypeID: uint16(e.TypeID), MoverSpeed: uint8(e.RotationSpeed)}
	for i, v := range e.Skill {
		h.Attack.Skill[i] = uint16(v)
		if e.Humanoid {
			h.Base.Skill[i] = uint16(v)
		}
		h.SkillXP[i] = uint32(e.SkillXP[i])
		h.Experience += uint32(e.SkillXP[i])
	}
	h.Attack.ToHit, h.Attack.DamageBase, h.Attack.DamageSpread, h.Attack.Active = uint16(e.ToHit), uint8(e.DamageBase), uint8(e.DamageSpread), e.XPSlot
	h.Attack.SecondBase, h.Attack.SecondSpread = e.SecondBase, e.SecondSpread
	h.Attack.ElementalBase, h.Attack.ElementalSpread = e.SecondaryDamage.Base, e.SecondaryDamage.Spread
	if h.Attack.ElementalBase != 0 || h.Attack.ElementalSpread != 0 {
		h.Attack.ElementalKind = e.SecondaryDamage.Selector + 1
	}
	h.Defence.Defence, h.Defence.Absorption = uint16(e.Defence), uint16(e.Absorption)
	for i, v := range e.Protection {
		h.Defence.Protection[i+1] = uint16(v)
		h.Defence.Resistance[i+1] = e.Resistance[i]
	}
	if e.Humanoid {
		if human == nil {
			return e, "", fmt.Errorf("actor %d lacks its maintained Human constructor basis", e.ID)
		}
		h = *human
		e.Speed, e.Capacity, e.ScanRange = int32(int16(h.Speed)), int32(int16(h.Capacity)), uint8(h.Sight>>8)
		// The legacy potion-headroom cache has no consumer after source arithmetic
		// is installed; permanent caps live in Human.Modifier.StatCap instead.
		e.PotionHeadroom = [4]int32{}
	} else {
		h.Modifier.HealthRegeneration, h.Modifier.ManaRegeneration = uint16(e.HealthRegeneration), uint16(e.ManaRegeneration)
	}
	s := HumanSourceActor(h, basis)
	s.Reach, s.AttackCharge, s.AttackRelax, s.EquipmentRuntimePresent = e.Reach, uint8(e.AttackCharge), uint8(e.AttackRelax), true
	e.SourceBinding = sim.SourceBinding{Class: class, Identity: key, RuntimeID: runtime, TokenRow: uint8(row), TypeID: uint16(e.TypeID), Face: face}
	if e.Humanoid {
		e.SourceBinding.DisplayBacking = uint32(member.MercenaryType)
		if face == 0 || member.FigureFace > 127 {
			return e, "", fmt.Errorf("actor %d lacks valid Human portrait", e.ID)
		}
		if e.TypeID < 0x1a && data.FigureDir(member.FigureDir).Female() {
			e.SourceBinding.Face |= 0x80
		}
		seed, err := SourceActorSeed(e.SourceBinding, t)
		if err != nil {
			return e, "", err
		}
		e.DyingTime = seed.DyingTime
	}
	if !h.Fighter {
		e.SourceBinding.ClassFlags |= 6 // HERO-HP-071: mage and allocated book.
	}
	if h.HasSpellbook {
		e.SourceBinding.ClassFlags |= 2
	}
	if !e.Humanoid && (e.TypeID == 0x47 || e.TypeID == 0x48) {
		e.SourceBinding.ClassFlags |= 6
	}
	if e.AlwaysHits {
		e.SourceBinding.ClassFlags |= 0x10
	}
	e.ActorLoad = sim.ActorLoad{Present: true, ContainerPresent: true, Source: s}
	e.NativeBasis = sim.NativeActorBasis{}
	e.HumanMovement = sim.HumanMovement{Present: true, RawSpeed: int16(e.Speed), NativeSpeed: e.Speed, Load: e.Load, Capacity: e.Capacity}
	if err := e.SourceBinding.Validate(e); err != nil {
		return e, "", err
	}
	return e, name, nil
}

// ConstructActorLoad completes the same basis after the actual native stock
// has been enriched. Own weight and container load are independent accumulators.
func ConstructActorLoad(e sim.Entity, worn [sim.EquipSlots]sim.ItemInstance, pack []sim.ItemStack) sim.Entity {
	var own, carried int32
	for _, item := range worn {
		if item.Code != 0 {
			own += int32(item.Weight)
		}
	}
	for _, item := range pack {
		carried += int32(item.Weight) * int32(item.Count)
	}
	e.ActorLoad.OwnWeight, e.ActorLoad.Accumulator = int16(own), carried
	e.Load = e.ActorLoad.CurrentLoad()
	e.HumanMovement.Load = e.Load
	e.ActorLoad.Source.Stats[5], e.ActorLoad.Source.Stats[6] = uint16(e.ActorLoad.OwnWeight), uint16(e.Load)
	// Preserve the already-computed current mover rate independently of +8c.
	binary.LittleEndian.PutUint16(e.ActorLoad.Source.Modifier[10:], uint16(e.HealthRegeneration))
	return e
}
