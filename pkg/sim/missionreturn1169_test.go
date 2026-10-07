package sim

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func TestMissionReturn1169NormalizesThroughInverseAndKeepsCurrentResidues(t *testing.T) {
	w := sourceMutationWorld(t, PlainItem(0xe01))
	e := &w.entities[0]
	e.Owner, e.TypeID = SelfSlot, HumanTypeID
	e.ActorLoad.Source.TypeID = 33
	e.ActorLoad.Source.EquipmentRuntimePresent = true
	e.Reach, e.AttackCharge, e.AttackRelax = 9, 23, 31
	e.HealthHundredths, e.ManaHundredths = 57, 93
	e.Mana, e.MaxMana = 7, 25
	e.SkillXP[1] = 123456
	e.ActorLoad.Source.Experience = 0xfffffff1
	e.MapUnitID = 777
	if !w.attachEffect(e.ID, e.ID, SpellRule{ID: 18}, EffectAbsorption, 7, 20, EffectDuration) {
		t.Fatal("source effect setup")
	}
	if binary.LittleEndian.Uint16(w.entities[0].ActorLoad.Source.Modifier[44:]) != 7 {
		t.Fatal("modifier setup missing")
	}
	if err := w.NormalizeMissionSurvivors(SelfSlot); err != nil {
		t.Fatal(err)
	}
	e = &w.entities[0]
	if len(w.attached) != 0 || e.ActorLoad.Source.Modifier[44] != 0 || e.HP != 100 || e.Mana != 25 || e.MapUnitID != 0 {
		t.Fatal("normalization did not inverse and heal")
	}
	load := e.CurrentActorLoad()
	if e.SkillXP[1] != 123456 || load.Inventory.Source.Experience != 0xfffffff1 || load.HealthHundredths != 57 || load.ManaHundredths != 93 || load.Inventory.Source.Reach != 9 || load.Inventory.Source.AttackCharge != 23 || load.Inventory.Source.AttackRelax != 31 {
		t.Fatal("current values were reset with actions")
	}
	// The next World receives current residues through the same stock/restore
	// seam used by StartMission. Its old snapshot arm deterministically has0.
	e.HealthHundredths, e.ManaHundredths = 0, 0
	if err := w.RestoreActorLoad(e.ID, *load); err != nil {
		t.Fatal(err)
	}
	if e.HealthHundredths != 57 || e.ManaHundredths != 93 {
		t.Fatal("next actor lost residues")
	}
}

func TestMissionReturn1169FailedInverseIsAtomic(t *testing.T) {
	w := sourceMutationWorld(t, PlainItem(0xe01))
	w.entities[0].Owner, w.entities[0].TypeID = SelfSlot, HumanTypeID
	if !w.attachEffect(1, 1, SpellRule{ID: 18}, EffectAbsorption, 7, 20, EffectDuration) {
		t.Fatal("source effect setup")
	}
	w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) {
		return s, fmt.Errorf("unsupported inverse")
	})
	before := w.Hash()
	if err := w.NormalizeMissionSurvivors(SelfSlot); err == nil || w.Hash() != before {
		t.Fatal("failed inverse partially changed World", err)
	}
}

func TestMissionReturn1169FelledInverseDoesNotMutateSharedActions(t *testing.T) {
	w := mustWorld(t, 1, Bounds{Width: 32, Height: 32}, []Entity{
		{ID: 1, Owner: SelfSlot, TypeID: HumanTypeID, HP: 5, MaxHP: 100},
		{ID: 2, Owner: SelfSlot, TypeID: HumanTypeID, HP: 80, MaxHP: 100},
	})
	w.attached = []attachedEffect{{Target: 1, Spell: 1, Kind: EffectHealth, Mode: EffectDuration, Magnitude: 20, Remaining: 10}}
	w.bookCasts = []bookCast{{Caster: 1, Target: 2, Spell: 1}, {Caster: 2, Target: 1, Spell: 18}}
	w.casts = []scriptCast{{Target: 1, AtUnit: true, Spell: 18}}
	w.structureUses = []StructureUse{{Entity: 1, Structure: 1}, {Entity: 2, Structure: 1}}
	before := w.Hash()
	if err := w.NormalizeMissionSurvivors(SelfSlot); err == nil || w.Hash() != before {
		t.Fatal("felled inverse mutated shared action state", err)
	}
}
