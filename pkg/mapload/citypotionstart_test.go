package mapload

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

func TestCityPotionMissionStartRestoresAppliedTimerOnce(t *testing.T) {
	for _, source := range []bool{false, true} {
		for _, scripted := range []bool{false, true} {
			p := PartyMember{ID: "potion owner", Class: 33, Hero: data.Hero{Body: 10, Reaction: 10, Mind: 10, Spirit: 10}}
			if source {
				h := data.HumanState{Body: 10, Reaction: 10, Mind: 10, Spirit: 10, Health: 20, HealthMax: 20, Speed: 40, Capacity: 200, Fighter: true, TypeID: 33, HasOwner: true, ManaReservePercent: 95}
				p.Carry = &Carry{LiveLoad: &sim.ActorLoadSnapshot{Inventory: sim.ActorLoad{Present: true, ContainerPresent: true, Source: HumanSourceActor(h, 2)}, Capacity: 200, Speed: 40}}
				p.Saved = &Saved{HP: 20, MaxHP: 20}
			}
			before, _, _ := PartyDisplayWithTable(p, nil)
			potion := sim.ItemInstance{Code: 0xe08, Kind: 3, Effects: []sim.ItemEffect{{Kind: 8, Mode: 1, Operand: 100 | 3<<16}}}
			result, ok := ApplyTownPotion(p, potion, nil)
			if !ok {
				t.Fatal("city potion use refused", source, scripted)
			}
			p = result
			d, _, _ := PartyDisplayWithTable(p, nil)
			if d.HealthRegeneration != before.HealthRegeneration+100 || p.PotionEffect == nil || p.PotionEffect.Remaining != 3 {
				t.Fatal("city fixture did not apply one timed modifier", source, scripted)
			}
			m := &alm.Map{Width: 32, Height: 32, Tiles: make([]uint16, 1024), Overlay: make([]uint8, 1024)}
			var w *sim.World
			var start Start
			var err error
			if scripted {
				w, start, err = StartMissionScripted(m, nil, DifficultyNormal, []PartyMember{p}, &sim.Script{})
			} else {
				w, start, err = StartMission(m, nil, DifficultyNormal, []PartyMember{p})
			}
			if err != nil {
				t.Fatal(err)
			}
			for tick := 0; tick <= 4; tick++ {
				e, ok := w.Entity(start.IDs[0])
				want := before.HealthRegeneration
				if tick < 3 {
					want += 100
				}
				if !ok || e.HealthRegeneration != want {
					t.Fatalf("source=%v scripted=%v tick=%d regeneration=%d want=%d", source, scripted, tick, e.HealthRegeneration, want)
				}
				effects := w.ActiveEffects()
				if tick < 3 && (len(effects) != 1 || effects[0].Remaining != uint16(3-tick) || effects[0].Target != e.ID) || tick >= 3 && len(effects) != 0 {
					t.Fatalf("source=%v scripted=%v tick=%d effects=%+v", source, scripted, tick, effects)
				}
				sim.Step(w, nil)
			}
		}
	}
}
