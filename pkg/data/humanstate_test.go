package data

import "testing"

func trainingHuman() HumanState {
	h := HumanState{Body: 41, Reaction: 35, Mind: 20, Spirit: 15,
		Weight: 40, InventoryWeight: 900, Health: 190, HealthMax: 200, HealthPeriod: 100,
		Mana: 20, ManaPeriod: 50, Fighter: true, TypeID: 33,
		Experience: 2300, SkillXP: [6]uint32{1000, 1000, 100, 100, 100, 0},
		Attack: HumanAttack{Skill: [6]uint16{7, 99}, Active: 1, Tail: [2]byte{0xab, 0xcd}},
		Base:   HumanAttack{ToHit: 321, Skill: [6]uint16{444, 9}, Active: 5, Tail: [2]byte{0x12, 0x34}},
		Modifier: HumanModifier{Speed: 2, Capacity: 3, HealthMax: 4, Sight: 5,
			Attack: HumanAttack{ToHit: 65520, Skill: [6]uint16{65000, 2}, DamageBase: 250, DamageSpread: 255,
				Active: 5, SecondBase: 6, SecondSpread: 7, ElementalBase: 8, ElementalSpread: 9, ElementalKind: 3, Tail: [2]byte{0xde, 0xad}},
			Defence: HumanDefence{Defence: 65530, Absorption: 65535, Protection: [6]uint16{23, 100, 65526, 2, 3, 4}, Resistance: [6]uint8{9, 1, 2, 3, 4, 5}}}}
	return h
}

func TestHumanTrainingWholeCoupledState(t *testing.T) {
	h := trainingHuman()
	n, err := h.Train(1)
	if err != nil {
		t.Fatal(err)
	}
	want := h
	want.Base.Skill[1], want.Attack.Skill[1] = 10, 12
	want.SkillXP[1], want.Experience = 1594, 2894
	want.Speed, want.Load, want.Capacity = 20, 490, 414
	want.Health, want.HealthMax, want.Mana = 140, 140, 0
	want.Sight, want.MoverSpeed = 1592, 20
	want.Attack.ToHit, want.Attack.DamageBase, want.Attack.DamageSpread = 35, 254, 1
	want.Attack.SecondBase, want.Attack.SecondSpread = 6, 7
	want.Attack.ElementalBase, want.Attack.ElementalSpread, want.Attack.ElementalKind = 8, 9, 3
	want.Defence = HumanDefence{Defence: 5, Absorption: 0, Protection: [6]uint16{23, 77, 0, 9, 10, 11}, Resistance: [6]uint8{9, 1, 2, 3, 4, 5}}
	if n != want {
		t.Fatalf("coupled training\ngot  %+v\nwant %+v", n, want)
	}
	if h != trainingHuman() {
		t.Fatal("training mutated its source")
	}
	price, err := h.TrainingPrice(1)
	if err != nil || price != 471 {
		t.Fatalf("base9 price %d %v", price, err)
	}
	price, err = n.TrainingPrice(1)
	if err != nil || price != 518 {
		t.Fatalf("base10/live12 price %d %v", price, err)
	}
	second, err := n.Train(1)
	if err != nil || second.Base.Skill[1] != 11 || second.Attack.Skill[1] != 13 || second.SkillXP[1] != 1854 {
		t.Fatalf("repeat %+v %v", second, err)
	}
}

func TestHumanSecondPhysicalProjectsCurrentThenTrainsModifier(t *testing.T) {
	h := trainingHuman()
	h.Attack.SecondBase, h.Attack.SecondSpread = 231, 249
	d := h.Derived(nil, 0)
	if d.Combat.SecondBase != 231 || d.Combat.SecondSpread != 249 || h.ProjectionError() != nil {
		t.Fatal("LOAD did not project current bytes")
	}
	n, err := h.Train(1)
	if err != nil {
		t.Fatal(err)
	}
	d = n.Derived(nil, 0)
	if d.Combat.SecondBase != 6 || d.Combat.SecondSpread != 7 {
		t.Fatal("Train did not clear then fold modifier")
	}
	if h.FighterProjectionError() == nil {
		t.Fatal("native fallback accepted unproducible pair")
	}
}

func TestHumanTrainingPreservesShadowsAndWrapsBeforeClamps(t *testing.T) {
	h := trainingHuman()
	h.Body, h.Modifier.StatCap[0] = 80, -20
	h.Modifier.Speed = 65506 // -30; clears modifier, not wrapped live speed
	h.Base.Skill[1], h.Modifier.Attack.Skill[1] = 99, 32767
	n, err := h.Train(1)
	if err != nil {
		t.Fatal(err)
	}
	if n.Body != 30 || n.Base.Skill[1] != 100 || n.Attack.Skill[1] != 0 || n.Modifier.Speed != 0 || int16(n.Speed) != -12 || n.MoverSpeed != 244 {
		t.Fatalf("signed/width stages: %+v", n)
	}
	if n.Base.Skill[0] != 444 || n.Attack.Skill[0] != 7 || n.Base.ToHit != h.Base.ToHit || n.Base.Tail != h.Base.Tail || n.Attack.Tail != h.Attack.Tail || n.Modifier.Attack != h.Modifier.Attack {
		t.Fatal("unmaintained shadow, active mirror or opaque tail changed")
	}
}

func TestHumanTrainingRefusesUnsupportedInputsTransactionally(t *testing.T) {
	for _, mutate := range []func(*HumanState){
		func(h *HumanState) { h.Attack.Active = 6 },
		func(h *HumanState) { h.Base.Skill[1] = 1000 },
		func(h *HumanState) { h.Experience = 0x80000000 },
	} {
		h := trainingHuman()
		mutate(&h)
		got, err := h.Train(1)
		if err == nil || got != h {
			t.Fatalf("unsupported mutation committed: %+v %v", got, err)
		}
	}
	for _, slot := range []int{-1, 0, 6} {
		h := trainingHuman()
		got, err := h.Train(slot)
		if err == nil || got != h {
			t.Fatal("unsupported school slot admitted")
		}
	}
}
