package sim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func trainingWorld(t *testing.T, base, bonus int32) *World {
	t.Helper()
	a, s := skAwarder(1, 2), skSource(2, 3)
	a.Mind = 400
	a.Skill[3], a.SkillXP[3] = (Rules{}).EffectiveSkill(base, bonus), skillXPFor(base)
	a.NativeTraining = NativeTraining{Present: true, Levels: [skillSlots]int32{0, 0, 0, base}}
	w := cbWorld(t, 1, a, s)
	w.equipment[0][11] = ItemInstance{Code: 0xec7e, Effects: []ItemEffect{{Kind: 29, Operand: uint32(bonus)}}}
	return w
}

func TestNativeTrainingAwardsAtEffectiveBounds(t *testing.T) {
	for _, tc := range []struct {
		base, bonus int32
		imported    bool
	}{{100, 200, false}, {90, 200, false}, {40, -100, false}, {100, 200, true}, {90, 200, true}, {40, -100, true}} {
		w := trainingWorld(t, tc.base, tc.bonus)
		w.hasSessionClock = tc.imported
		if w.NativeTrainingNeedsProducer(1) {
			t.Fatal("known worn producer refused a reproducible sheet")
		}
		before := w.entities[0]
		raised := w.awardSkill(0, 0, 1_000_000, 1)
		if raised != (tc.base < 100) {
			t.Fatalf("base %d bonus %d: raised=%v", tc.base, tc.bonus, raised)
		}
		want := tc.base
		if raised {
			want++
		}
		e := w.entities[0]
		if e.NativeTraining.Levels[3] != want || e.Skill[3] != before.Skill[3] || !raised && e.SkillXP != before.SkillXP {
			t.Fatalf("base %d bonus %d: base %d skill %d XP %v", tc.base, tc.bonus, e.NativeTraining.Levels[3], e.Skill[3], e.SkillXP)
		}
		raw, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var cold World
		if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() {
			t.Fatal("trained base lost at binary/hash boundary", err)
		}
		w.awardSkill(0, 0, 1_000_000, 1)
		cold.awardSkill(0, 0, 1_000_000, 1)
		if cold.Hash() != w.Hash() {
			t.Fatal("the next award differs after cold decode")
		}
	}
}

func TestImportedNativeTrainingKeepsAnUnreproducedSheet(t *testing.T) {
	w := trainingWorld(t, 15, 5)
	w.hasSessionClock = true
	w.entities[0].Skill[3], w.entities[0].SkillXP[3] = 100, 12_527_588
	w.entities[0].MaxHP, w.entities[0].MaxMana = 110, 452
	w.equipment[0][11].Effects[0].Kind = 35
	if w.nativeSkillBonuses(0)[3] != 5 || w.trainedSkill(0, 3) != 15 {
		t.Fatal("independent base and known worn producer changed")
	}
	before := w.entities[0]
	hash := w.Hash()
	if w.awardSkill(0, 3, 1_000_000, 1) || w.Hash() != hash {
		a := w.entities[0]
		t.Fatalf("unreproduced imported award changed base %d->%d effective %d->%d XP %d->%d pools %d/%d->%d/%d", before.NativeTraining.Levels[3], a.NativeTraining.Levels[3], before.Skill[3], a.Skill[3], before.SkillXP[3], a.SkillXP[3], before.MaxHP, before.MaxMana, a.MaxHP, a.MaxMana)
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != hash || cold.awardSkill(0, 3, 1_000_000, 1) || cold.Hash() != hash {
		t.Fatal("unreproduced sheet changed at cold decode or next award", err)
	}
	levels := cold.entities[0].Skill
	levels[3] = 20
	if !cold.SetSkillLevels(1, levels) || cold.NativeTrainingNeedsProducer(1) || !cold.awardSkill(0, 3, 1_000_000, 1) || cold.entities[0].NativeTraining.Levels[3] != 16 || cold.entities[0].Skill[3] != 21 {
		t.Fatal("known producer update did not resume normal native awards")
	}
}

func TestNativeTrainingDistinguishesIdenticalEffectiveLevels(t *testing.T) {
	w := trainingWorld(t, 90, 200)
	before := w.Hash()
	base := w.entities[0].NativeTraining.Levels
	base[3] = 100
	w.SetNativeTraining(1, NativeTraining{Present: true, Levels: base})
	if w.Hash() == before || w.entities[0].Skill[3] != 255 {
		t.Fatal("training does not enter canonical identity independently")
	}
}

func TestNativeTrainingLegacyFallbackAndKnownBase(t *testing.T) {
	w := trainingWorld(t, 100, 200)
	w.entities[0].NativeTraining = NativeTraining{}
	raw, err := w.MarshalBinary()
	if err != nil || raw[0] == nativeTrainingFormVersion {
		t.Fatal("an absent base changed the historical wire", err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.trainedSkill(0, 3) != 55 {
		t.Fatal("historical fallback changed", err)
	}
	base := [skillSlots]int32{0, 0, 0, 100}
	if !cold.SetNativeTraining(1, NativeTraining{Present: true, Levels: base}) || cold.trainedSkill(0, 3) != 100 {
		t.Fatal("known ordinary or constructor base did not replace the inverse")
	}
	cold.entities[0].ActorLoad.Source.Class = 2
	if cold.SetNativeTraining(1, NativeTraining{Present: true, Levels: base}) {
		t.Fatal("source arithmetic accepted a native base")
	}
}

func TestNativeTrainingWireRejectsMalformedPopulation(t *testing.T) {
	w := trainingWorld(t, 40, -100)
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	start := len(raw) - 9 - int(binary.LittleEndian.Uint32(raw[len(raw)-9:]))
	for _, mutate := range []func([]byte){
		func(b []byte) { binary.LittleEndian.PutUint32(b[start:], ^uint32(0)) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+4:], 999) },
		func(b []byte) { b[len(b)-1] = 0 },
	} {
		bad := bytes.Clone(raw)
		mutate(bad)
		var cold World
		before := cold.Hash()
		if err := cold.UnmarshalBinary(bad); err == nil || cold.Hash() != before {
			t.Fatal("malformed training was accepted or changed the receiver", err)
		}
	}
	w.entities[0].NativeTraining.Present = false
	if _, err := w.MarshalBinary(); err == nil {
		t.Fatal("absent training with residual levels was accepted")
	}
}
