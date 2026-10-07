package game

import (
	"encoding/binary"
	"math"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
)

func TestCurrentPartyBaseDoesNotApplyEquipmentOrPotionsTwice(t *testing.T) {
	c := sav.Character{Stats: [sav.UnitStatWords]uint16{45, 100, 0xfffd, 20}, SkillLevels: [6]uint16{7, 12}, Basis: &sav.ActorBasis{}}
	c.Basis.Human.Fields.Modifier[0], c.Basis.Human.Fields.Modifier[1] = 1, 2
	binary.LittleEndian.PutUint16(c.Basis.Human.Fields.Base[4:], 10)
	potion := [4]int32{2, 50}
	want := data.Hero{Body: 42, Reaction: 99, Mind: -3, Spirit: 1<<20 + 20, Skill: [6]int32{7, 10}}
	p, err := capturePartyBase(want, c, true, potion)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Lifts) != 2 || p.Lifts[0].Field != 1 || p.Lifts[1].Field != 3 {
		t.Fatal("uncapped ordinary operands were copied into continuation", p)
	}
	got, err := p.restore(c, potion)
	if err != nil || got != want {
		t.Fatal("base, capped stat, wide stat or ordinary base skill changed", got, err)
	}
	c.Stats[0], c.Stats[1], c.Stats[3] = 46, 90, 21
	binary.LittleEndian.PutUint16(c.Basis.Human.Fields.Base[4:], 11)
	got, err = p.restore(c, potion)
	if err != nil || got.Body != 43 || got.Reaction != 38 || got.Spirit != 21 || got.Skill[1] != 11 {
		t.Fatal("stale base lift overwrote edited ordinary fields", got, err)
	}
	for _, policy := range []currentPartyBase{
		{Native: false, Lifts: p.Lifts},
		{Native: true, Lifts: append(p.Lifts[:1:1], p.Lifts[0])},
		{Native: true, Lifts: []currentPartyBaseLift{{Field: 10, Lift: 1}}},
		{Native: true, Lifts: []currentPartyBaseLift{{Field: 3, Wire: 21, Lift: math.MaxInt64}}},
		{Native: true, Lifts: []currentPartyBaseLift{{Field: 3, Wire: 21, Lift: math.MinInt64}}},
		{Native: true, Lifts: []currentPartyBaseLift{{Field: 3, Wire: 999, Lift: math.MaxInt64}}},
		{Native: true, Lifts: []currentPartyBaseLift{{Field: 4, Wire: 7, Modifier: 1, Lift: 2}}},
	} {
		if got, err := policy.restore(c, potion); err == nil || got != (data.Hero{}) {
			t.Fatal("invalid base policy returned a partial hero", got, err)
		}
	}
}

func TestCurrentPartySourceBaseReadsOnlyOrdinaryFields(t *testing.T) {
	c := sav.Character{Stats: [sav.UnitStatWords]uint16{43, 27, 0xfffd, 31}, SkillLevels: [6]uint16{3, 12, 14}, Basis: &sav.ActorBasis{}}
	c.Basis.Human.Fields.Modifier[0] = 1
	potion := [4]int32{2, 9}
	p, err := capturePartyBase(data.Hero{Body: 999, Skill: [6]int32{99}}, c, false, potion)
	if err != nil || len(p.Lifts) != 0 {
		t.Fatal("source arithmetic retained a second Hero", p, err)
	}
	want := data.Hero{Body: 43, Reaction: 27, Mind: -3, Spirit: 31, Skill: [6]int32{3, 12, 14}}
	got, err := p.restore(c, potion)
	if err != nil || got != want {
		t.Fatal("current source fields were recomputed or replaced", got, err)
	}
}
