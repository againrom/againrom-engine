package mapload

import (
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

// Literals below are assembled in source storage widths, independently of
// HumanState.Derive and the SAV projection. No expected value calls either.
func sourceLiteral1110() sim.SourceActor {
	s := sim.SourceActor{Class: 2, Stats: [14]uint16{10, 15, 10, 10, 37, 0, 500, 201, 40, 90, 100, 0, 0, 50}, Fighter: true, TypeID: 33, HasOwner: true, ManaReservePercent: 95, ManaFloor: 77, Sight: 1700, MoverSpeed: 37}
	s.Attack = [24]byte{99, 0, 7, 0, 44, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 9, 1, 31, 32, 0, 0, 0, 0xab, 0xcd}
	s.Base = [24]byte{88, 0, 0xbc, 1, 9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 22, 23, 5, 24, 25, 0, 0, 0, 0x12, 0x34}
	s.Modifier[4], s.Modifier[6], s.Modifier[8] = 2, 200, 4
	s.Modifier[18], s.Modifier[22] = 5, 2
	s.Modifier[32], s.Modifier[33], s.Modifier[35], s.Modifier[36] = 2, 3, 6, 7
	s.Modifier[42] = 3
	return s
}

func sourceWorld1110(t *testing.T, s sim.SourceActor, items []sim.ItemInstance) *sim.World {
	t.Helper()
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, sim.Terrain{}, []sim.Entity{{ID: 1, X: 1, Y: 1, HP: 40, MaxHP: 90, Speed: 37, TypeID: 33, Owner: 1}, {ID: 2, X: 2, Y: 1, HP: 20, MaxHP: 20}}, nil, sim.Relations{}, nil, []sim.Stock{{ID: 1, ItemInstances: items}})
	if err != nil {
		t.Fatal(err)
	}
	BindSourceDerive(w)
	load := sim.ActorLoadSnapshot{Inventory: sim.ActorLoad{Present: true, ContainerPresent: true, Accumulator: 406, Source: s}, Load: int32(int16(s.Stats[6])), Capacity: int32(int16(s.Stats[7])), Speed: int32(int16(s.Stats[4]))}
	load.Movement = sim.HumanMovement{Present: true, RawSpeed: int16(s.Stats[4]), NativeSpeed: load.Speed, Load: load.Load, Capacity: load.Capacity}
	if err := w.RestoreActorLoad(1, load); err != nil {
		t.Fatal(err)
	}
	return w
}

func sourceNative1110(t *testing.T, w *sim.World) *sim.World {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var out sim.World
	BindSourceDerive(&out)
	if err := out.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	if out.Hash() != w.Hash() {
		t.Fatal("native load changed source basis")
	}
	return &out
}

func TestSourceActor1110TransferPublishesFullHumanBeforeReturn(t *testing.T) {
	s := sourceLiteral1110()
	w := sourceWorld1110(t, s, []sim.ItemInstance{{Code: 0xe01, WeightPresent: true, Weight: 2}})
	for _, current := range []*sim.World{w, sourceNative1110(t, w)} {
		if err := current.MoveCarried(1, 2, 0xe01, 1); err != nil {
			t.Fatal(err)
		}
		e := current.Entities()[0]
		if e.Load != 202 || e.Capacity != 301 || e.Speed != 15 || e.HP != 24 || e.MaxHP != 24 || e.Mana != 0 || e.ScanRange != 5 || e.ToHit != 39 || e.Defence != 8 || e.DamageBase != 4 || e.DamageSpread != 3 || e.SecondBase != 6 || e.SecondSpread != 7 || e.Skill[1] != 11 {
			t.Fatalf("whole literal sheet after source transfer: %+v", e)
		}
		if e.ActorLoad.Source.Base != s.Base || e.ActorLoad.Source.Modifier != s.Modifier || e.ActorLoad.Source.Attack[22] != 0xab || e.ActorLoad.Source.Experience != 0 || e.ActorLoad.Source.MoverSpeed != 15 {
			t.Fatal("retained base/modifier/tail or mover changed incorrectly")
		}
		if speed, ok := e.RetainedHumanSpeed(); !ok || speed != 15 {
			t.Fatal("derived speed substituted native fallback", speed, ok)
		}
		_ = sourceNative1110(t, current)
	}
}

func TestSourceActor1110UnitCallbackDoesNotDeriveHuman(t *testing.T) {
	s := sourceLiteral1110()
	s.Class = 1
	s.Stats[12] = 80
	w := sourceWorld1110(t, s, []sim.ItemInstance{{Code: 0xe01, WeightPresent: true, Weight: 2}})
	w.BindSourceDerive(nil)
	if err := w.MoveCarried(1, 2, 0xe01, 1); err != nil {
		t.Fatal(err)
	}
	e := w.Entities()[0]
	if e.Load != 202 || e.Capacity != 201 || e.Speed != 37 || e.MaxHP != 90 || e.ActorLoad.Source.ManaFloor != 76 {
		t.Fatal("Unit callback took Human derive", e.CurrentActorLoad())
	}
}

func TestSourceActor1110MissingRuleAndZeroCapacityRefuseBothSides(t *testing.T) {
	for _, zero := range []bool{false, true} {
		s := sourceLiteral1110()
		if zero {
			s.Stats[7] = 0
		}
		w := sourceWorld1110(t, s, []sim.ItemInstance{{Code: 0xe01, WeightPresent: true, Weight: 2}})
		if !zero {
			w.BindSourceDerive(nil)
		}
		before := w.Hash()
		if err := w.MoveCarried(1, 2, 0xe01, 1); err == nil || w.Hash() != before {
			t.Fatal("refusal changed transfer state", err)
		}
	}
}

func TestSourceActor1110PermanentAndTimedPotionMatrix(t *testing.T) {
	for _, tc := range []struct {
		name                                        string
		kind, mode                                  uint8
		operand                                     uint32
		body, capacity, speed, maxHP, absorb, regen int32
	}{
		{"body", 2, 0, 1, 11, 311, 16, 26, 0, 0},
		{"health regen", 8, 1, 100 | 2<<16, 10, 301, 15, 24, 0, 100},
		{"absorption", 16, 1, 50 | 2<<16, 10, 301, 15, 24, 50, 0},
		{"fighter mana gate", 11, 1, 100 | 2<<16, 10, 301, 15, 24, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := sourceLiteral1110()
			item := sim.ItemInstance{Code: 0xe01, Kind: 3, WeightPresent: true, Weight: 2, Effects: []sim.ItemEffect{{Kind: tc.kind, Mode: tc.mode, Operand: tc.operand}}}
			w := sourceWorld1110(t, s, []sim.ItemInstance{item})
			if !w.UseCarriedPotion(1, 0) {
				t.Fatal("source potion refused")
			}
			e := w.Entities()[0]
			if int32(e.ActorLoad.Source.Stats[0]) != tc.body || e.Load != 202 || e.Capacity != tc.capacity || e.Speed != tc.speed || e.MaxHP != tc.maxHP || e.Absorption != tc.absorb || e.HealthRegeneration != tc.regen || e.PotionStats != ([4]int32{}) {
				t.Fatalf("literal potion sheet: %+v", e)
			}
			if tc.mode == 1 {
				fresh := sourceNative1110(t, w)
				for tick := 0; tick < 2; tick++ {
					sim.Step(w, nil)
					sim.Step(fresh, nil)
				}
				if w.Hash() != fresh.Hash() || len(w.ActiveEffects()) != 0 || w.Entities()[0].HealthRegeneration != 0 || w.Entities()[0].Absorption != 0 {
					t.Fatal("expiry/native continuation differs")
				}
			}
		})
	}
}

func TestSourceActor1110NegativeFoldKeepsSpeedClearsOnlyModifier(t *testing.T) {
	s := sourceLiteral1110()
	binary.LittleEndian.PutUint16(s.Modifier[4:], 65516)
	w := sourceWorld1110(t, s, []sim.ItemInstance{{Code: 0xe01, WeightPresent: true, Weight: 2}})
	if err := w.MoveCarried(1, 2, 0xe01, 1); err != nil {
		t.Fatal(err)
	}
	e := w.Entities()[0]
	want := s.Modifier
	want[4], want[5] = 0, 0
	if e.Speed != -7 || e.HumanMovement.RawSpeed != -7 || e.ActorLoad.Source.MoverSpeed != 249 || !reflect.DeepEqual(e.ActorLoad.Source.Modifier, want) {
		t.Fatal("negative speed was normalized or retained modifier discarded", e.CurrentActorLoad())
	}
}

func TestSourceActor1110TownPotionAndCarryKeepAppliedState(t *testing.T) {
	s := sourceLiteral1110()
	w := sourceWorld1110(t, s, nil)
	p := PartyMember{ID: "source", Carry: &Carry{LiveLoad: w.Entities()[0].CurrentActorLoad()}, Saved: &Saved{HP: 40, MaxHP: 90}}
	item := sim.ItemInstance{Code: 0xe01, Kind: 3, WeightPresent: true, Weight: 2, Effects: []sim.ItemEffect{{Kind: 8, Mode: 1, Operand: 100 | 3<<16}}}
	result, ok := ApplyTownPotion(p, item, nil)
	if !ok || !CommitSourceTownPotion(&p, result) {
		t.Fatal("source town potion refused")
	}
	d, hp, _ := PartyDisplayWithTable(p, nil)
	if d.HealthRegeneration != 100 || d.HealthMax != 24 || hp != 24 || p.Carry.LiveLoad.Inventory.Accumulator != 406 || p.Carry.LiveLoad.Load != 203 || p.PotionEffect == nil || p.PotionEffect.Remaining != 3 {
		t.Fatal("town applied twice or changed container", p.Carry.LiveLoad, d, hp)
	}
	// Mission carry keeps the complete basis, not the entry Hero or a
	// reconstructed equipment sum. The source contains already-applied pools.
	carried := CarryParty([]PartyMember{{ID: "source"}}, w, []sim.EntityID{1})
	if len(carried) != 1 || carried[0].Hero.Body != 10 || carried[0].Carry.LiveLoad.Inventory.Source.Base != s.Base || carried[0].Carry.LiveLoad.Load != 500 {
		t.Fatal("mission carry dropped source state", carried)
	}
}
