package sim

import "testing"

const (
	osMage   = EntityID(1)
	osEnemy  = EntityID(2)
	osEnemy2 = EntityID(3)
)

// osWorld is a mage on an open map beside hostile units placed at enemies.
// The mage knows Fire Arrow (1, range 7) and Shield (18); hostiles carry
// 5000 health so no cast ends the walk by a death.
func osWorld(t *testing.T, enemies ...[2]int32) *World {
	t.Helper()
	mage := effectMage(osMage, 4, 4, 1<<1|1<<18)
	mage.Speed, mage.Capacity = 21, 1411
	mage.Mind, mage.MaxMana, mage.Mana = 30, 5000, 5000
	mage.AttackCharge, mage.AttackRelax = 8, 12
	ents := []Entity{mage}
	for k, at := range enemies {
		e := spEnt(osEnemy+EntityID(k), at[0], at[1])
		e.Owner = 2
		e.HP, e.MaxHP = 5000, 5000
		ents = append(ents, e)
	}
	rel := engRel(t, [3]uint32{SelfSlot, 2, relationHostile}, [3]uint32{2, SelfSlot, 2})
	w, err := NewStructuredWorld(5, engBounds, ModeCanonical, Terrain{}, ents, nil, rel, nil, nil,
		[]SpellRule{hlArrow(), acbShield()}, GhostTemplate{}, nil)
	if err != nil {
		t.Fatalf("NewStructuredWorld: %v", err)
	}
	return w
}

// legacyPredicates restates the five in-flight predicates as they were
// written before the order state, reading the raw carriers.
type legacyPredicates struct {
	booked, inFlight, paid, scrollPending, scrollStarted, castBusy bool
}

func legacyOf(w *World, i int) legacyPredicates {
	e := w.entities[i]
	var l legacyPredicates
	for _, c := range w.bookCasts {
		if c.Caster != e.ID {
			continue
		}
		l.booked = true
		l.inFlight = c.Phase == bookCharging
		l.paid = c.Phase == bookCharging && c.Paid
		l.castBusy = c.Phase != bookApproach
	}
	for _, s := range w.scrollCasts {
		if s.Caster == e.ID {
			l.scrollPending, l.scrollStarted = true, s.Started
		}
	}
	queued := e.PendingOrder.Kind == PendingActorCast || e.PendingOrder.Kind == PendingCellCast
	l.castBusy = l.castBusy || queued || l.scrollPending || e.CastWait != 0
	return l
}

func osAgree(t *testing.T, w *World, tick int) {
	t.Helper()
	for i := range w.entities {
		l := legacyOf(w, i)
		s := w.orderStateAt(i)
		got := legacyPredicates{booked: s.booked(), inFlight: s.charging(), paid: s.paidWindUp(),
			scrollPending: s.scrollPending(), scrollStarted: s.scroll == scrollStarted, castBusy: s.castBusy()}
		if got != l {
			t.Fatalf("tick %d entity %d: order state %+v, former predicates %+v", tick, w.entities[i].ID, got, l)
		}
		p := s.phase()
		if (p == orderPhaseChargingPaid) != l.paid || (p == orderPhaseChargingPaid || p == orderPhaseChargingUnpaid) != l.inFlight {
			t.Fatalf("tick %d entity %d: phase %d disagrees with %+v", tick, w.entities[i].ID, p, l)
		}
	}
}

// The order state answers what every former predicate answered, on every
// tick of a manual cast, an automatic cast, a scroll cast and a strike.
func TestOrderStateAgreesWithEveryFormerPredicate(t *testing.T) {
	cases := []struct {
		name  string
		want  orderPhase
		setup func(w *World) []Command
	}{
		{"manual cast", orderPhaseChargingUnpaid, func(w *World) []Command { return []Command{Cast(osMage, osEnemy, 1)} }},
		{"paid manual cast", orderPhaseChargingPaid, func(w *World) []Command {
			for k := range w.spells {
				w.spells[k].Delivery = 1
			}
			return []Command{Cast(osMage, osEnemy, 1)}
		}},
		{"manual self cast", orderPhaseRecovering, func(w *World) []Command { return []Command{Cast(osMage, osMage, 18)} }},
		{"automatic cast", orderPhaseChargingUnpaid, func(w *World) []Command { w.entities[0].AutoSpell = 1; return nil }},
		{"scroll cast", orderPhaseScroll, func(w *World) []Command {
			item := ItemInstance{Code: 0xe10, Kind: 4, Price: 50, Effects: []ItemEffect{{Kind: 41, Operand: 1 | 60<<16}}}
			w.carried[0] = []ItemStack{StackItem(item, 1)}
			return []Command{UseScroll(osMage, 0, osEnemy)}
		}},
		{"strike", orderPhaseStriking, func(w *World) []Command {
			w.entities[0].DamageBase, w.entities[0].AlwaysHits, w.entities[0].Reach = 10, true, 1
			return []Command{Attack(osMage, osEnemy)}
		}},
		{"cast during a step", orderPhaseQueued, func(w *World) []Command {
			return []Command{MoveTo(osMage, CellPoint{X: 30, Y: 4})}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := osWorld(t, [2]int32{8, 4})
			cmds := c.setup(w)
			seen := map[orderPhase]bool{}
			for tick := range 120 {
				if c.name == "cast during a step" && tick == 3 {
					cmds = []Command{Cast(osMage, osEnemy, 1)}
				}
				Step(w, cmds)
				cmds = nil
				osAgree(t, w, tick)
				seen[w.orderStateAt(0).phase()] = true
			}
			if !seen[c.want] {
				t.Fatalf("the case never reached phase %d: %v", c.want, seen)
			}
		})
	}
}
