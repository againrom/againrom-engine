package mapload_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestCurrentHumanSpellDisplayReadsCurrentEquippedCast(t *testing.T) {
	table := &mapload.Table{Spells: defCollection{{}, {name: "Fire Arrow", params: spellRow(3, 1, 1, 7, 4, 8, 0)}}}
	p := mapload.PartyMember{ID: "hero", Carry: &mapload.Carry{LiveLoad: &sim.ActorLoadSnapshot{}}, Weapon: &data.Weapon{SpellName: "stale", SpellPower: 99}}
	p.Carry.LiveLoad.Inventory.Source.Class = 2
	p.Carry.EquippedItems[0] = sim.PlainItem(1)
	p.Carry.EquippedItems[0].Effects = []sim.ItemEffect{{Kind: 41, Operand: 1 | 23<<16}}
	d, hp, mana := mapload.PartyDisplayWithTable(p, table)
	if d.Combat.SpellName != "Fire_Arrow" || d.Combat.SpellPower != 23 {
		t.Fatal("source-backed display ignored current item", d.Combat)
	}
	p.Carry.EquippedItems[0].Effects = nil
	empty, eh, em := mapload.PartyDisplayWithTable(p, table)
	if empty.Combat.SpellName != "" || empty.Combat.SpellPower != 0 || hp != eh || mana != em || empty.HealthMax != d.HealthMax || empty.Combat.DamageBase != d.Combat.DamageBase {
		t.Fatal("removed enchantment or unrelated current values were replaced", empty)
	}
}

func TestTrainedHumanMissionUsesOriginalFoldOrderAndCombat(t *testing.T) {
	for _, tc := range []struct {
		name     string
		weight   int32
		modifier uint16
		raw      int16
		rate     int32
	}{
		{"capacity order", 900, 2, 20, 20},
		{"floor before modifier", 30000, 2, 8, 8},
		{"negative signed final", 900, 65506, -12, 1},
		{"word wrap before sign", 900, 32760, -32758, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := data.HumanState{Body: 41, Reaction: 35, Mind: 20, Spirit: 15, Weight: 40, Health: 110, HealthMax: 130, HealthPeriod: 100, ManaPeriod: 50,
				Fighter: true, TypeID: 33, InventoryWeight: tc.weight, Experience: 1400, SkillXP: [6]uint32{0, 1400},
				Attack: data.HumanAttack{Active: 1}, Base: data.HumanAttack{Skill: [6]uint16{0, 9}},
				Modifier: data.HumanModifier{Speed: tc.modifier, Capacity: 1000, Attack: data.HumanAttack{DamageBase: 40}}}
			h, err := h.Train(1)
			if err != nil {
				t.Fatal(err)
			}
			if int16(h.Speed) != tc.raw || h.Capacity != 1411 {
				t.Fatalf("original fold speed=%d capacity=%d", int16(h.Speed), h.Capacity)
			}
			p := mapload.PartyMember{ID: "hero", Class: 1, Profile: data.Profile{Fighter: true, HealthColumn: true}, Carry: &mapload.Carry{}, Saved: &mapload.Saved{HealthRegenPeriod: 100, ManaRegenPeriod: 50}}
			mapload.ApplyOriginalHuman(&p, h)
			w, st, err := mapload.StartMissionScripted(startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20}), nil, mapload.DifficultyNormal, []mapload.PartyMember{p}, nil)
			if err != nil {
				t.Fatal(err)
			}
			e := w.Entities()[0]
			if e.RotationSpeed != int32(h.MoverSpeed) {
				t.Errorf("retained Human mover byte=%d, spawned turn rate=%d", h.MoverSpeed, e.RotationSpeed)
			}
			if e.ID != st.IDs[0] || e.HP != 110 || e.MaxHP != 130 || e.ToHit != 45 || e.DamageBase != 44 || e.DamageSpread != 2 || e.Load != int32(h.Load) {
				t.Fatalf("wrong source projection %+v", e)
			}
			rate, _, _, ok := w.StepRate(e.ID, e.X+1, e.Y)
			if !ok || rate != tc.rate {
				t.Fatalf("rate=%d ok=%t want%d", rate, ok, tc.rate)
			}
			// The stored damage reaches an actual hit. A native item-only rederive
			// would discard the retained +40 modifier and could never deal 44..46.
			e.X, e.Y = 1, 1
			v := sim.Entity{ID: e.ID + 1, X: 2, Y: 1, HP: 1000, MaxHP: 1000, Owner: 2}
			combat, err := sim.NewWorld(7, sim.Bounds{Width: 4, Height: 4}, sim.ModeCanonical, nil, []sim.Entity{e, v})
			if err != nil {
				t.Fatal(err)
			}
			sim.Step(combat, []sim.Command{{Kind: sim.KindAttack, Entity: e.ID, X: int32(v.ID)}})
			hit := false
			for tick := 0; tick < 200; tick++ {
				sim.Step(combat, nil)
				hp := combat.Entities()[1].HP
				if hp < 1000 {
					if loss := 1000 - hp; loss < 44 || loss > 46 {
						t.Fatalf("first hit damage=%d", loss)
					}
					hit = true
					break
				}
			}
			if !hit {
				t.Fatal("no hit")
			}
		})
	}
}
