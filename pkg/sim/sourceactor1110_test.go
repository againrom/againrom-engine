package sim

import (
	"encoding/binary"
	"testing"
)

func TestSourceActor1110SkillSinkUsesBaseAndCommitsBeforeReturn(t *testing.T) {
	a, b := skAwarder(1, 1), skSource(2, 2)
	w := cbWorld(t, 1, a, b)
	s := SourceActor{Class: 2, Fighter: true, Stats: [14]uint16{10, 20, 60, 10, 18, 0, 0, 101, 100, 100, 100, 0, 0, 50}, Experience: 2300}
	s.Attack[16] = 3
	binary.LittleEndian.PutUint16(s.Attack[8:], 99) // live skill includes an independent retained modifier
	binary.LittleEndian.PutUint16(s.Base[8:], 9)
	binary.LittleEndian.PutUint16(s.Modifier[26:], 2)
	s.SkillXP[3] = 1357
	load := ActorLoadSnapshot{Inventory: ActorLoad{Present: true, ContainerPresent: true, Source: s}, Capacity: 101, Speed: 18, Movement: HumanMovement{Present: true, RawSpeed: 18, NativeSpeed: 18, Capacity: 101}}
	if err := w.RestoreActorLoad(1, load); err != nil {
		t.Fatal(err)
	}
	var committed SourceActor
	w.BindSourceDerive(func(n SourceActor, acc int32, _ Rules) (SourceActor, error) {
		if n.Experience != 2300 {
			committed = n
			n.Stats[4], n.Stats[7], n.Stats[9], n.MoverSpeed = 23, 111, 120, 23
			binary.LittleEndian.PutUint16(n.Attack[8:], 12)
		}
		return n, nil
	})
	// Mind60: 100*(4*60+30)/120=225; base9 has room236. The live99
	// is not the cap/threshold basis. Aggregate discrepancy is retained.
	if !w.awardSkill(0, 0, 100, 1) {
		t.Fatal("source raise refused")
	}
	e := w.entities[0]
	if committed.Experience != 2525 || committed.SkillXP[3] != 1582 || binary.LittleEndian.Uint16(committed.Base[8:]) != 10 || e.Skill[3] != 12 || e.SkillXP[3] != 1582 || e.MaxHP != 120 || e.Capacity != 111 || e.Speed != 23 {
		t.Fatal("award returned a partial source sheet", committed, e)
	}
}

func TestSourceActor1110MissingRuleFreezesTimerBeforeAnyMutation(t *testing.T) {
	w := cbWorld(t, 1, skAwarder(1, 1))
	w.entities[0].ActorLoad = ActorLoad{Present: true, ContainerPresent: true, Source: SourceActor{Class: 2}}
	w.attached = []attachedEffect{{Target: 1, Kind: EffectAbsorption, Mode: EffectDuration, Magnitude: 50, Remaining: 1}}
	before := w.Hash()
	Step(w, nil)
	if w.Hash() != before || w.Tick() != 0 || len(w.attached) != 1 {
		t.Fatal("unbound source timer partly expired")
	}
}

func TestSourceActor1110NativeRecordShapeAndCorruptFlags(t *testing.T) {
	if binary.Size(SourceActor{}) != 209 {
		t.Fatal("source record width drift")
	}
	w := cbWorld(t, 1, skAwarder(1, 1))
	w.entities[0].ActorLoad = ActorLoad{Present: true, ContainerPresent: true, Source: SourceActor{Class: 2}}
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	start := len(b) - entityIDFloorLen - spellDeliverySpanLen - 65 - 2500 - 4 - 224
	for _, offset := range []int{6, 15 + 198, 15 + 199, 15 + 200, 15 + 208} {
		bad := append([]byte(nil), b...)
		bad[start+offset] = 2
		var fresh World
		if err := fresh.UnmarshalBinary(bad); err == nil {
			t.Fatalf("noncanonical flag at%d admitted", offset)
		}
	}
	bad := append([]byte(nil), b...)
	bad[start+15] = 0
	bad[start+16] = 1
	var fresh World
	if err := fresh.UnmarshalBinary(bad); err == nil {
		t.Fatal("absent source residue admitted")
	}
}
