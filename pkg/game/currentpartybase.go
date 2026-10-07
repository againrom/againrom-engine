package game

import (
	"encoding/binary"
	"fmt"
	"math"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
)

type currentPartyBase struct {
	Native         bool
	LegacyTraining bool                   `json:",omitempty"`
	LegacyClass    bool                   `json:",omitempty"`
	Lifts          []currentPartyBaseLift `json:",omitempty"`
}

type currentPartyBaseLift struct {
	Field    uint8
	Wire     uint16
	Modifier int8
	Potion   int32
	Lift     int64
}

func partyHeroFields(h *data.Hero) []*int32 {
	out := []*int32{&h.Body, &h.Reaction, &h.Mind, &h.Spirit}
	for i := range h.Skill {
		out = append(out, &h.Skill[i])
	}
	return out
}

// Native base skills have ordinary Base words. Attribute modifiers and earned
// potion gains must be removed once; a capped or wide base needs an anchored lift.
func partyBaseFields(c sav.Character, native bool, potion [4]int32) ([]currentPartyBaseLift, error) {
	if c.Basis == nil {
		return nil, fmt.Errorf("current party has no ordinary actor basis")
	}
	fields := make([]currentPartyBaseLift, 4+data.SkillSlots)
	for i := range fields {
		v := &fields[i]
		v.Field = uint8(i)
		if i < 4 {
			v.Wire = c.Stats[i]
			if native {
				v.Modifier, v.Potion = int8(c.Basis.Human.Fields.Modifier[i]), potion[i]
			}
		} else {
			slot := i - 4
			v.Wire = c.SkillLevels[slot]
			if native && slot > 0 {
				v.Wire = binary.LittleEndian.Uint16(c.Basis.Human.Fields.Base[2+2*slot:])
			}
		}
	}
	return fields, nil
}

func (v currentPartyBaseLift) base() int64 {
	return int64(int16(v.Wire)) - int64(v.Modifier) - int64(v.Potion)
}

func capturePartyBase(h data.Hero, c sav.Character, native bool, potion [4]int32) (*currentPartyBase, error) {
	out := &currentPartyBase{Native: native}
	if !native {
		return out, nil
	}
	fields, err := partyBaseFields(c, native, potion)
	if err != nil {
		return nil, err
	}
	for i, value := range partyHeroFields(&h) {
		v := fields[i]
		v.Lift = int64(*value) - v.base()
		if v.Lift != 0 {
			out.Lifts = append(out.Lifts, v)
		}
	}
	return out, nil
}

func (p currentPartyBase) restore(c sav.Character, potion [4]int32) (data.Hero, error) {
	if !p.Native && (p.LegacyTraining || p.LegacyClass) {
		return data.Hero{}, fmt.Errorf("current source party declares native state absence")
	}
	var out data.Hero
	fields, err := partyBaseFields(c, p.Native, potion)
	if err != nil {
		return out, err
	}
	if !p.Native && len(p.Lifts) != 0 {
		return out, fmt.Errorf("current source party duplicates ordinary base values")
	}
	seen := map[uint8]bool{}
	for _, lift := range p.Lifts {
		if int(lift.Field) >= len(fields) || seen[lift.Field] || lift.Lift == 0 {
			return data.Hero{}, fmt.Errorf("invalid current party base lift")
		}
		if lift.Field >= 4 && (lift.Modifier != 0 || lift.Potion != 0) ||
			lift.Lift < math.MinInt32-lift.base() || lift.Lift > math.MaxInt32-lift.base() {
			return data.Hero{}, fmt.Errorf("invalid current party base operand")
		}
		seen[lift.Field] = true
		v := &fields[lift.Field]
		if lift.Wire == v.Wire && lift.Modifier == v.Modifier && lift.Potion == v.Potion {
			v.Lift = lift.Lift
		}
	}
	for i, value := range partyHeroFields(&out) {
		v := fields[i]
		base := v.base()
		if v.Lift < math.MinInt32-base || v.Lift > math.MaxInt32-base {
			return data.Hero{}, fmt.Errorf("current party base exceeds native width")
		}
		*value = int32(base + v.Lift)
	}
	return out, nil
}
