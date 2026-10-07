package sim

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCurrentActorOrdinaryValuesRemainAuthoritative(t *testing.T) {
	e := Entity{HP: 145, MaxHP: 145, Mana: -1, MaxMana: 10, Reaction: 9, SeeInvisible: 3, PotionStats: [4]int32{2, 0, 0, 0}}
	p := e.Values()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"HP":`, `"MaxHP":`, `"Reaction":`, `"ActorLoad":`, `"Book":`} {
		if strings.Contains(string(b), field) {
			t.Fatalf("duplicated wire authority %s", field)
		}
	}
	decoded := Entity{HP: 144, MaxHP: 143, Mana: 65535, MaxMana: 10, Reaction: 8}
	if err := decoded.restoreValues(p); err != nil {
		t.Fatal(err)
	}
	if decoded.HP != 144 || decoded.MaxHP != 143 || decoded.Reaction != 8 || decoded.Mana != -1 || decoded.SeeInvisible != 3 || decoded.PotionStats[0] != 2 {
		t.Fatalf("wrong authority after LOAD: %+v", decoded)
	}
	decoded.Mana = 65534
	if err := decoded.restoreValues(p); err != nil {
		t.Fatal(err)
	}
	if decoded.Mana != 65534 {
		t.Fatal("changed ordinary word retained a stale extension")
	}
}

func TestCurrentClockOrdinaryCounterRemainsAuthoritative(t *testing.T) {
	w := scrollFixtureWorld(t, 2, 1)
	p := w.CurrentPolicy()
	p.TickHigh = 7
	w.tick = 19
	if err := w.restoreCurrentPolicy(p); err != nil {
		t.Fatal(err)
	}
	if w.tick != uint64(7)<<32|19 {
		t.Fatal("ordinary low counter was replaced")
	}
	w.tick, w.fullTick = 23, 91
	p.TickHigh, p.ClockKnown = 0, true
	if err := w.restoreCurrentPolicy(p); err != nil {
		t.Fatal(err)
	}
	if w.tick != 23 || w.fullTick != 91 || !w.hasSessionClock {
		t.Fatal("source clock was replaced")
	}
}

func TestCurrentRuntimeTypeHasOneOrdinaryAnchor(t *testing.T) {
	for _, runtimeType := range []int32{33, 65569, -3} {
		e := Entity{TypeID: runtimeType, ActorLoad: ActorLoad{Present: true, Source: SourceActor{Class: 2, TypeID: 34}}}
		policy := e.Values()
		if policy.RuntimeType == nil || policy.RuntimeType.Wire != 34 || policy.RuntimeType.Value != runtimeType {
			t.Fatal("distinct current runtime type was not captured")
		}
		for _, sourceType := range []uint16{34, 35} {
			decoded := Entity{TypeID: int32(sourceType), ActorLoad: ActorLoad{Present: true, Source: SourceActor{Class: 2, TypeID: sourceType}}}
			if err := decoded.restoreValues(policy); err != nil {
				t.Fatal(err)
			}
			want := int32(sourceType)
			if sourceType == 34 {
				want = runtimeType
			}
			if decoded.TypeID != want || decoded.ActorLoad.Source.TypeID != sourceType {
				t.Fatal("runtime operand replaced ordinary authority", decoded.TypeID, decoded.ActorLoad.Source.TypeID, want)
			}
		}
	}
	e := Entity{TypeID: 34, ActorLoad: ActorLoad{Present: true, Source: SourceActor{Class: 2, TypeID: 34}}}
	if e.Values().RuntimeType != nil {
		t.Fatal("matching type duplicated ordinary value")
	}
	p := e.Values()
	p.RuntimeType = &ActorRuntimeType{Wire: 34, Value: 34}
	if err := e.restoreValues(p); err == nil {
		t.Fatal("redundant runtime operand accepted")
	}
	p.RuntimeType.Value = 33
	p.Widths = []ActorNumericResidue{{Field: 21, Wire: 34, Lift: 65536}}
	if err := e.restoreValues(p); err == nil {
		t.Fatal("conflicting type lift accepted")
	}
}
