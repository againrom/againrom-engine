package sim

import (
	"fmt"
	"testing"
)

func TestModDamageColumnsKeepWholeRecordAndLandedEndpoints(t *testing.T) {
	for _, tc := range []struct {
		power        int32
		base, spread int32
	}{{0, 300, 300}, {100, 1300, 1300}, {255, 2850, 2850}} {
		for _, raw := range []int32{0, 32767} {
			t.Run(fmt.Sprintf("power%d/raw%d", tc.power, raw), func(t *testing.T) {
				target := spEnt(2, 3, 0)
				target.HP, target.MaxHP = 10000, 10000
				w := castDamageWorld(t, spEnt(1, 0, 0), target)
				state := crtStateFor(t, 0, raw)
				originalStream(w, state)
				rule := SpellRule{ID: 1, Damaging: true, DamageMin: 300, DamageMax: 600}
				w.applySpellDamage(0, 1, rule, tc.power)
				want := tc.base
				if raw == 32767 {
					want += tc.spread
				}
				if got := 10000 - w.entities[1].HP; got != want {
					t.Fatalf("damage %d, want %d from whole pair %d/%d", got, want, tc.base, tc.spread)
				}
				if uint32(w.RandomState()) != crtAfter(state, 2) {
					t.Fatal("physical accuracy and elemental spread must take two draws")
				}
				b, s := SpellRecordDamage(300, 600, tc.power)
				old := int32(b)
				if raw == 32767 {
					old += int32(s)
				}
				if old == want {
					t.Fatal("low-byte loss control agrees with whole damage")
				}
				t.Logf("whole pair %d/%d lands %d; low-byte control lands %d", tc.base, tc.spread, want, old)
			})
		}
	}
}

func TestModDamageQueuedPointKeepsWholeLiveAndSaturatedAfterLoad(t *testing.T) {
	for _, power := range []int32{0, 100} {
		w := deliveryTestWorld(t, 1)
		w.spells[0].DamageMin, w.spells[0].DamageMax = 300, 600
		w.entities[1].HP, w.entities[1].MaxHP = 10000, 10000
		w.queuePointDelivery(0, 1, w.spells[0], power)
		if w.deliveries[0].Payload != nil {
			t.Fatal("whole live columns were frozen into byte operands")
		}
		before := w.Hash()
		rows := w.NativeSpellDeliverySaveStates()
		if w.Hash() != before || rows[0].Area.Payload.DirectDamage[19] != 255 || rows[0].Area.Payload.DirectDamage[20] != 255 {
			t.Fatal("SAVE must project both columns to 255 without mutating live state")
		}
		cold := transferCurrentDelivery(t, w, nil)
		for _, world := range []*World{w, cold} {
			originalStream(world, crtStateFor(t, 0, 32767))
			world.applySpellDelivery(world.deliveries[0], nil)
		}
		want := int32(600)
		if power == 100 {
			want = 2600
		}
		if live, loaded := 10000-w.entities[1].HP, 10000-cold.entities[1].HP; live != want || loaded != 510 {
			t.Fatalf("power %d: live damage %d, cold damage %d, want %d and saturated 510", power, live, loaded, want)
		}
	}
}

func TestModDamageAreasKeepWholeLiveAndSaturatedAfterLoad(t *testing.T) {
	rule := SpellRule{ID: 3, Area: true, Damaging: true, School: 1, DamageMin: 300, DamageMax: 600}
	target := spEnt(1, 10, 10)
	target.HP, target.MaxHP = 10000, 10000
	w := spWorld(t, 7, []SpellRule{rule}, target)
	w.effects = []cellEffect{{Key: 0x0a0a, Spell: 3, Mode: areaModeCloud, Remaining: 32, Cells: []uint16{0x0a0a}}}
	cold := currentAreaOrdinaryWorld(t, w, nil)
	if cold.effects[0].Current == nil {
		t.Fatal("cold LOAD replaced the saturated payload with whole live columns")
	}
	for _, world := range []*World{w, cold} {
		originalStream(world, crtStateFor(t, 0, 32767))
		world.applyAreaCells(world.effects[0], rule, []uint16{0x0a0a})
	}
	if live, loaded := 10000-w.entities[0].HP, 10000-cold.entities[0].HP; live != 600 || loaded != 510 {
		t.Fatalf("area damage live %d cold %d, want 600 and saturated 510", live, loaded)
	}
}

func TestModDamageQueuedAreaKeepsSaturatedPayloadAfterLoad(t *testing.T) {
	w := deliveryTestWorld(t, 2)
	w.spells[0].Area, w.spells[0].TargetsUnit = true, false
	w.spells[0].DamageMin, w.spells[0].DamageMax = 300, 600
	w.entities[1].HP, w.entities[1].MaxHP = 10000, 10000
	w.queueSpellDelivery(spellDelivery{Rule: w.spells[0], AtCell: true, X: 5, Y: 1})
	cold := transferCurrentDelivery(t, w, nil)
	if cold.deliveries[0].Current == nil {
		t.Fatal("cold queued area replaced saturated bytes with whole columns")
	}
	for _, world := range []*World{w, cold} {
		originalStream(world, crtStateFor(t, 0, 32767))
		world.applySpellDelivery(world.deliveries[0], nil)
	}
	if live, loaded := 10000-w.entities[1].HP, 10000-cold.entities[1].HP; live != 600 || loaded != 510 {
		t.Fatalf("queued area damage live %d cold %d, want 600 and saturated 510", live, loaded)
	}
}

func TestModDamageFireBallDividesWholeComponents(t *testing.T) {
	for _, side := range []int32{1, 2} {
		target := spEnt(1, 10, 10)
		target.HP, target.MaxHP, target.TokenSize = 10000, 10000, uint8(side)
		rule := SpellRule{ID: 2, Area: true, Damaging: true, DamageMin: 300, DamageMax: 600}
		w := spWorld(t, 7, []SpellRule{rule}, target)
		originalStream(w, crtStateFor(t, 0, 32767))
		w.applyAreaCells(cellEffect{Spell: 2, Mode: areaModeBlast}, rule, []uint16{0x0a0a})
		if got, want := 10000-w.entities[0].HP, 600/(side*side); got != want {
			t.Fatalf("footprint side %d: damage %d, want %d", side, got, want)
		}
	}
}

func TestModDamageBuildingKeepsWholeRollLessFive(t *testing.T) {
	w := structureAreaWorld(t, []Structure{{ID: 1, Field42: 10000, MaxHealth: 10000, Width: 1, Height: 1, Attach: 1}})
	originalStream(w, crtStateFor(t, 32767))
	w.applyStructureSpellAt(0, 0, SpellRule{ID: 1, Damaging: true, DamageMin: 300, DamageMax: 600}, 0)
	if got := 10000 - w.structures[0].Field42; got != 595 {
		t.Fatalf("building damage %d, want 600 minus five", got)
	}
}
