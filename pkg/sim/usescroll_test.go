package sim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestScrollBlockedStaleAndGroupInterruptReturnReservedUnit(t *testing.T) {
	for _, kind := range []string{"blocked", "stale", "stand", "patrol", "swarm"} {
		t.Run(kind, func(t *testing.T) {
			w := scrollWorld1090(t, 12, 65535)
			if kind == "blocked" {
				for i := range w.grid {
					w.grid[i] = 1
				}
				w.grid[17] = 0
				w.grid[28] = 0
			}
			Step(w, []Command{{Kind: KindUseScroll, Entity: 1, X: 2}})
			switch kind {
			case "stale":
				Step(w, []Command{{Kind: KindDamage, Entity: 2, X: 200}})
			case "stand":
				Step(w, []Command{{Kind: KindGroupStance, Entity: 1, X: int32(OrderStandGround)}})
			case "patrol":
				Step(w, []Command{{Kind: KindGroupPatrolTo, Entity: 1, X: 2, Y: 3}})
			case "swarm":
				Step(w, []Command{{Kind: KindGroupSwarmTo, Entity: 1, X: 2, Y: 3}})
			}
			for n := 0; n < 100 && len(w.scrollCasts) > 0; n++ {
				Step(w, nil)
			}
			stock, _ := w.CarriedStacks(1)
			if len(w.scrollCasts) != 0 || len(stock) != 1 || stock[0].Count != 65535 {
				t.Fatalf("refund=%+v cast=%+v", stock, w.scrollCasts)
			}
		})
	}
}

func TestScrollPointTeleportAndIneffectiveCompletedUseStillConsume(t *testing.T) {
	for _, point := range []bool{false, true} {
		w := scrollWorld1090(t, 2, 1)
		id := uint16(13)
		w.spells = []SpellRule{{ID: id, Restorative: true, TargetsUnit: true, DamageMin: 10, DamageMax: 10, MaxRange: 8}}
		if point {
			id = 26
			w.spells = []SpellRule{{ID: id, MaxRange: 8}}
		}
		w.carried[0][0].Effects[0].Operand = uint32(id) | 60<<16
		kind := KindUseScroll
		if point {
			kind = KindUseScrollAt
		}
		Step(w, []Command{{Kind: kind, Entity: 1, X: 2, Y: 3}})
		for n := 0; n < 50 && len(w.scrollCasts) > 0; n++ {
			Step(w, nil)
		}
		if len(w.carried[0]) != 0 || len(w.scrollCasts) != 0 {
			t.Fatal("completed release did not consume exactly one")
		}
		if point && (w.entities[0].X != 2 || w.entities[0].Y != 3) {
			t.Fatal("point scroll did not teleport")
		}
		if !point && w.entities[1].HP != 100 {
			t.Fatal("full-health target changed")
		}
	}
}

func TestScrollNativeRejectsBoundedMalformedReservation(t *testing.T) {
	w := scrollWorld1090(t, 12, 1)
	Step(w, []Command{{Kind: KindUseScroll, Entity: 1, X: 2}})
	form, _ := w.MarshalBinary()
	at := len(form) - entityIDFloorLen - spellDeliverySpanLen - 65 - relationLen - w.originalDeadSectionLen() - w.instanceWeightSectionLen() - w.actorLoadSectionLen() - w.scriptSectionLen() - w.scrollSectionLen()
	for _, kind := range []string{"count", "flags", "caster", "coordinate", "phase", "item"} {
		bad := bytes.Clone(form)
		switch kind {
		case "count":
			binary.LittleEndian.PutUint32(bad[at:], 0xffffffff)
		case "flags":
			bad[at+4+18] = 255
		case "caster":
			binary.LittleEndian.PutUint32(bad[at+4:], 99)
		case "coordinate":
			binary.LittleEndian.PutUint32(bad[at+4+8:], 0x7fffffff)
		case "phase":
			bad[at+4+19] = 2
		case "item":
			bad[at+4+20+2] = 3
		}
		before := w.Hash()
		if err := w.UnmarshalBinary(bad); err == nil {
			t.Fatalf("accepted %s", kind)
		}
		if w.Hash() != before {
			t.Fatalf("%s changed receiver", kind)
		}
	}
}

func scrollWorld1090(t *testing.T, x int32, count uint32) *World {
	return scrollFixtureWorld(t, x, count)
}

func scrollFixtureWorld(t *testing.T, x int32, count uint32) *World {
	t.Helper()
	rule := SpellRule{ID: 1, ManaCost: 100, School: 1, MaxRange: 3, DamageMin: 10, DamageMax: 10, TargetsUnit: true, Damaging: true}
	e := spEnt(1, 1, 1)
	e.AttackCharge, e.AttackRelax = 4, 2
	w := spWorld(t, 1090, []SpellRule{rule}, e, spEnt(2, x, 1))
	item := ItemInstance{Code: 0xe10, Kind: 4, Price: 50, Effects: []ItemEffect{{Kind: 41, Operand: 1 | 60<<16}}}
	w.carried[0] = []ItemStack{StackItem(item, count)}
	return w
}

func TestScrollReservationApproachReleaseAndNativeContinuation(t *testing.T) {
	for _, count := range []uint32{1, 3} {
		w := scrollWorld1090(t, 12, count)
		Step(w, []Command{{Kind: KindUseScroll, Entity: 1, X: 2}})
		if len(w.ScrollCasts()) != 1 {
			t.Fatal("not reserved")
		}
		stock, _ := w.CarriedStacks(1)
		if count == 1 && len(stock) != 0 || count > 1 && (len(stock) != 1 || stock[0].Count != count-1) {
			t.Fatalf("stock=%+v", stock)
		}
		beforeHP := w.Entities()[1].HP
		for tick := 0; tick < 60 && len(w.ScrollCasts()) != 0; tick++ {
			form, _ := w.MarshalBinary()
			var loaded World
			if err := loaded.UnmarshalBinary(form); err != nil {
				t.Fatal(err)
			}
			Step(w, nil)
			Step(&loaded, nil)
			if w.Hash() != loaded.Hash() {
				t.Fatalf("continuation differs tick%d", tick)
			}
		}
		if len(w.ScrollCasts()) != 0 || w.Entities()[1].HP >= beforeHP {
			t.Fatalf("no release: actor=%+v target=%+v cast=%+v", w.Entities()[0], w.Entities()[1], w.ScrollCasts())
		}
		if w.Entities()[0].KnownSpells != 0 || w.Entities()[0].Mana != 0 {
			t.Fatal("scroll altered book/mana")
		}
	}
}

func TestScrollCancelAndRefusalRetainExactItem(t *testing.T) {
	for _, started := range []bool{false, true} {
		w := scrollWorld1090(t, 2, 2)
		before, _ := w.CarriedItems(1)
		Step(w, []Command{{Kind: KindUseScroll, Entity: 1, X: 2}})
		if started {
			Step(w, nil)
		}
		Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 1, Y: 3}})
		after, _ := w.CarriedItems(1)
		if len(w.ScrollCasts()) != 0 || len(after) != len(before) || !ItemEqual(after[0], before[0]) || after[0].Price != before[0].Price {
			t.Fatal("cancel lost item")
		}
		for n := 0; n < 10; n++ {
			Step(w, nil)
		}
		if w.Entities()[1].HP != 100 {
			t.Fatal("cancel applied damage")
		}
	}
	w := scrollWorld1090(t, 2, 1)
	for _, bad := range []struct {
		index  int
		target EntityID
		cell   bool
	}{{-1, 2, false}, {1, 2, false}, {0, 99, false}, {0, 0, true}} {
		before, _ := w.MarshalBinary()
		if w.beginScroll(0, bad.index, bad.target, 3, 3, bad.cell) {
			t.Fatal("invalid admission")
		}
		after, _ := w.MarshalBinary()
		if !bytes.Equal(before, after) {
			t.Fatal("refusal mutated state")
		}
	}
}
