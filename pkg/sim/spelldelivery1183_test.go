package sim

import (
	"bytes"
	"testing"
)

func deliveryTestWorld(t *testing.T, id uint16) *World {
	t.Helper()
	caster := effectMage(1, 1, 1, uint32(1)<<id)
	caster.Mana, caster.MaxMana, caster.AttackCharge = 100, 100, 8
	caster.Mind = 0
	target := spEnt(2, 5, 1)
	target.Owner, target.HP, target.MaxHP = 2, 1000, 1000
	rule := SpellRule{ID: id, ManaCost: 10, School: 1, TargetsUnit: true, Damaging: true, MaxRange: 15, DamageMin: 5, DamageMax: 5, Delivery: 2, EffectSpeed: 128}
	w := spWorld(t, 1183, []SpellRule{rule}, caster, target)
	w.relations.turnHostile(SelfSlot, 2)
	return w
}

func deliveryRoundTrip1183(t *testing.T, w *World) *World {
	t.Helper()
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	reencoded, err := back.MarshalBinary()
	if err != nil || !bytes.Equal(raw, reencoded) {
		t.Fatal("noncanonical delivery round trip", err)
	}
	return &back
}

func TestBookDeliveryPaysOnAdmissionAndSurvivesChargeAndFlight(t *testing.T) {
	for _, id := range []uint16{1, 13, 14} {
		t.Run(string(rune('A'+id)), func(t *testing.T) {
			w := deliveryTestWorld(t, id)
			events := StepObserved(w, []Command{spCast(1, 2, uint32(id))})
			if w.entities[0].Mana != 90 || w.entities[1].HP != 1000 {
				t.Fatal("admission applied damage or failed to pay", w.entities[0].Mana, w.entities[1].HP)
			}
			if id == 14 && (len(events) != 1 || len(events[0].Victims) != 1 || len(w.deliveries) != 1) {
				t.Fatal("admission fan absent", events, w.deliveries)
			}
			back := deliveryRoundTrip1183(t, w)
			for tick := 0; len(w.deliveries) == 0 && tick < 64; tick++ {
				Step(w, nil)
				StepObserved(back, nil)
				if w.Hash() != back.Hash() {
					t.Fatal("charge LOAD changed state", tick)
				}
			}
			if len(w.deliveries) != 1 || w.entities[1].HP != 1000 {
				t.Fatal("payload was not delayed", len(w.deliveries), w.entities[1].HP)
			}
			want := uint16(8)
			if id == 13 || id == 14 {
				want = 10
			}
			if w.deliveries[0].Remaining != want {
				t.Fatal("counter", w.deliveries[0].Remaining, want)
			}
			back = deliveryRoundTrip1183(t, w)
			for tick := uint16(1); tick <= want+1; tick++ {
				Step(w, nil)
				StepObserved(back, nil)
				if w.Hash() != back.Hash() {
					t.Fatal("flight LOAD changed state", tick)
				}
				if tick <= want && w.entities[1].HP != 1000 {
					t.Fatal("early impact", tick)
				}
			}
			if w.entities[1].HP >= 1000 || w.PendingSpellDeliveries() != 0 {
				t.Fatal("missing impact", w.entities[1].HP, w.deliveries)
			}
			hp := w.entities[1].HP
			for tick := 0; tick < 64; tick++ {
				Step(w, nil)
			}
			if w.entities[1].HP != hp || w.entities[0].Mana != 90 {
				t.Fatal("duplicate damage/payment", w.entities[1].HP, w.entities[0].Mana)
			}
		})
	}
}

func TestDeliveryCounterUsesSignedDecrementAndIgnoresPresentation(t *testing.T) {
	for _, counter := range []uint16{0, 1, 2, 4, 10, 65535} {
		w := deliveryTestWorld(t, 1)
		if !w.queuePointDelivery(0, 1, w.spells[0], 0) {
			t.Fatal("prepare")
		}
		w.deliveries[0].Remaining = counter
		back := deliveryRoundTrip1183(t, w)
		steps := int(counter)
		if steps == 0 || int16(counter) < 0 {
			steps = 1
		}
		for tick := 1; tick <= steps+1; tick++ {
			w.stepSpellDeliveries(nil)
			back.stepSpellDeliveries(&castObs{})
			if w.Hash() != back.Hash() {
				t.Fatal("observer changed delivery")
			}
			if tick <= steps && w.entities[1].HP != 1000 {
				t.Fatal("early delivery", counter, tick)
			}
		}
		if w.entities[1].HP != 995 || len(w.deliveries) != 0 {
			t.Fatal("wrong delivery", counter, w.entities[1].HP)
		}
	}
}

func TestDeliveryDecoderRefusesMalformedStateAtomically(t *testing.T) {
	w := deliveryTestWorld(t, 1)
	w.queuePointDelivery(0, 1, w.spells[0], 0)
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{raw[:len(raw)-1], append(append([]byte(nil), raw...), 0), bytes.Replace(raw, []byte(`"Delivery":1`), []byte(`"Delivery":9`), 1)} {
		back := deliveryRoundTrip1183(t, w)
		before := back.Hash()
		if err := back.UnmarshalBinary(bad); err == nil || back.Hash() != before {
			t.Fatal("bad delivery admitted or partially changed world", err)
		}
	}
}

func TestDeliveryAreaWaitsAndKeepsItsAimedCell(t *testing.T) {
	w := deliveryTestWorld(t, 2)
	r := w.spells[0]
	r.Area, r.TargetsUnit, r.Distribution, r.Radius = true, false, distributionDiamond, 1
	if !w.landArea(r, 0, 1, true, 1, 1, 5, 1, nil) || len(w.deliveries) != 1 {
		t.Fatal("area preparation")
	}
	w.entities[1].X = 10 // moving after admission must not move the blast
	back := deliveryRoundTrip1183(t, w)
	for i := 0; i < 9; i++ {
		w.stepSpellDeliveries(nil)
		back.stepSpellDeliveries(&castObs{})
		if w.Hash() != back.Hash() || w.entities[1].HP != 1000 {
			t.Fatal("area followed moving target or LOAD changed the countdown", i)
		}
	}
	if w.PendingSpellDeliveries() != 0 {
		t.Fatal("area did not arrive")
	}
	w.entities[1].X = 5
	w.landArea(r, 0, 1, true, 1, 1, 5, 1, nil)
	for i := 0; i < 8; i++ {
		w.stepSpellDeliveries(nil)
		if w.entities[1].HP != 1000 {
			t.Fatal("early blast")
		}
	}
	w.stepSpellDeliveries(nil)
	if w.entities[1].HP >= 1000 {
		t.Fatal("blast payload absent")
	}
}

func TestDeliveryPointKeepsTargetAndIgnoresLaterCasterLoss(t *testing.T) {
	w := deliveryTestWorld(t, 1)
	w.queuePointDelivery(0, 1, w.spells[0], 0)
	w.entities[0].HP = 0
	w.clearFelled(0)
	w.entities[1].X = 12
	back := deliveryRoundTrip1183(t, w)
	for i := 0; i < 9; i++ {
		w.stepSpellDeliveries(nil)
		back.stepSpellDeliveries(nil)
	}
	if w.Hash() != back.Hash() || w.entities[1].HP != 995 {
		t.Fatal("prepared point was retimed or cancelled")
	}
	w.queuePointDelivery(0, 1, w.spells[0], 0)
	w.entities[1].HP = -10
	for i := 0; i < 30; i++ {
		w.stepSpellDeliveries(nil)
	}
	if w.entities[1].HP != -10 || w.PendingSpellDeliveries() != 0 {
		t.Fatal("dead target received an effect")
	}
}

func TestDeliveryPrismaticCapsTheFanAndKeepsPrimaryFirst(t *testing.T) {
	w := deliveryTestWorld(t, 14)
	for id := EntityID(3); id <= 10; id++ {
		e := spEnt(id, 5, 2)
		e.Owner = 2
		w.entities = append(w.entities, e)
	}
	for _, power := range []int32{0, 100} {
		w.deliveries = nil
		victims := w.preparePrismatic(0, 9, w.spells[0], power, false)
		if len(victims) != int(power/20+2) || w.deliveries[0].Target != 10 {
			t.Fatal("primary or total-victim cap", power, victims, w.deliveries)
		}
		for _, e := range w.entities {
			if e.SpellFX != 0 {
				t.Fatal("impact mark before delivery")
			}
		}
	}
}

func TestDeliveryScriptUsesAuthoredSourceForItsClock(t *testing.T) {
	w := deliveryTestWorld(t, 1)
	_, _, ok := w.landPointCast(scriptCast{AtUnit: true, Target: 2, FromX: 4, FromY: 1}, w.spells[0])
	if !ok || len(w.deliveries) != 1 || w.deliveries[0].Remaining != 2 || w.entities[1].SpellFX != 0 {
		t.Fatal("script source or early impact", w.deliveries)
	}
}

func TestDeliveryCancellationDoesNotRefundOrRepeatAdmission(t *testing.T) {
	w := deliveryTestWorld(t, 1)
	Step(w, []Command{spCast(1, 2, 1)})
	Step(w, []Command{{Kind: KindKill, Entity: 1}})
	back := deliveryRoundTrip1183(t, w)
	for i := 0; i < 64; i++ {
		Step(w, nil)
		Step(back, nil)
	}
	if w.Hash() != back.Hash() || w.entities[0].Mana != 90 || w.entities[1].HP != 1000 || w.PendingSpellDeliveries() != 0 {
		t.Fatal("cancelled admission was refunded, replayed or delivered")
	}
}

func TestDeliverySignedBookCostSurvivesChargeAndFlight(t *testing.T) {
	for _, id := range []uint16{1, 13, 14} {
		w := deliveryTestWorld(t, id)
		w.entities[0].Book.State = BookPresent
		w.entities[0].Book.Slots[id-1] = BookSpell{Range: 15, ManaCost: 65535}
		w.entities[0].Mana = 0
		deliveryRoundTrip1183(t, w)
		Step(w, []Command{spCast(1, 2, uint32(id))})
		if w.entities[0].Mana != 1 {
			t.Fatal("signed admission changed", id)
		}
		back := deliveryRoundTrip1183(t, w)
		for n := 0; len(w.deliveries) == 0 && n < 64; n++ {
			Step(w, nil)
			Step(back, nil)
		}
		if len(w.deliveries) != 1 || w.Hash() != back.Hash() || w.deliveries[0].Rule.ManaCost != -1 {
			t.Fatal("signed prepared rule or charge LOAD", id)
		}
		back = deliveryRoundTrip1183(t, w)
		for n := 0; n < 64; n++ {
			Step(w, nil)
			StepObserved(back, nil)
			if w.Hash() != back.Hash() {
				t.Fatal("signed flight LOAD", id, n)
			}
		}
		if w.entities[0].Mana != 1 || w.entities[1].HP >= 1000 || len(w.deliveries) != 0 {
			t.Fatal("signed payment or impact", id)
		}
	}
}

func TestDeliverySignedWeaponPowerSurvivesFlight(t *testing.T) {
	rule := wpnRule(6, 6, 5)
	rule.Delivery, rule.EffectSpeed = 2, 128
	w := spWorld(t, 1, []SpellRule{rule}, wpnCaster(1, 0, 0, 1, -1, 1, 0), spEnt(2, 1, 0))
	deliveryRoundTrip1183(t, w)
	Step(w, []Command{cbOrder(1, 2)})
	if len(w.deliveries) != 1 || w.deliveries[0].Power != -1 {
		t.Fatal("signed weapon preparation")
	}
	back := deliveryRoundTrip1183(t, w)
	for n := 0; n < 3; n++ {
		Step(w, nil)
		StepObserved(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatal("signed weapon LOAD", n)
		}
	}
	if w.entities[1].HP >= 100 {
		t.Fatal("signed weapon payload lost")
	}
}
