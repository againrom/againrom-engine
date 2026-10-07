package sim

import (
	"fmt"
	"testing"
)

// A release marker or a pickup completion waits behind the whole loaded cycle,
// the recovery and the two boundary turns included, because the order machine
// consumes nonzero progress before it reads a pending order. A manual cast or
// scroll is not held: it replaces the cycle, and a held marker with it.

var recoveryPhases = []AttackPhase{AttackRelaxing, AttackBoundaryOne}

// recoveryAdvance steps w until actor 0 stands in phase on its victim.
func recoveryAdvance(t *testing.T, w *World, phase AttackPhase) {
	t.Helper()
	for range 400 {
		if e := w.entities[0]; e.HasAttackTarget && e.AttackPhase == phase {
			return
		}
		Step(w, nil)
	}
	t.Fatalf("fixture: phase %d not reached", phase)
}

// recoveryForce puts actor 0 in a recovery phase without the blow that would
// lead to it, so a victim's health shows only what a later order does.
func recoveryForce(w *World, ph AttackPhase) {
	e := &w.entities[0]
	e.AttackPhase, e.AttackCountdown = ph, 0
	if ph == AttackRelaxing {
		e.AttackCountdown = 8
	}
}

func recoveryMage(t *testing.T, ph AttackPhase) *World {
	t.Helper()
	w := manualCastWorld(t)
	w.entities[0].AlwaysHits, w.entities[0].DamageBase, w.entities[0].Reach = true, 5, 1
	w.entities[1].HP, w.entities[1].MaxHP = 200, 200
	w.orderAttack(0, 2)
	recoveryForce(w, ph)
	return w
}

func recoveryScroll(t *testing.T, ph AttackPhase) *World {
	t.Helper()
	w := scrollFixtureWorld(t, 12, 2)
	e := &w.entities[0]
	e.AttackCharge, e.AttackRelax, e.Reach = 20, 8, 1
	w.entities[1].X, w.entities[1].Y = e.X+1, e.Y
	w.orderAttack(0, 2)
	recoveryForce(w, ph)
	return w
}

// The standing no-pick and the group release leave a release marker beside the
// recovering cycle. The marker ends the victim when the cycle returns to ready,
// so no second blow loads. Control: with the marker removed the cycle runs on.
func TestRecoveryReleaseKeepsBodyAndEndsBeforeAnotherCycle(t *testing.T) {
	for _, ph := range recoveryPhases {
		for _, standing := range []bool{false, true} {
			w := lcPlayerWorld(t)
			w.relations.Set(SelfSlot, 2, 2)
			recoveryAdvance(t, w, ph)
			old := w.entities[0]
			if standing {
				w.entities[0].ScanRange = 0
				w.acquireStanding(0)
			} else {
				w.releaseAttack(0)
			}
			e := w.entities[0]
			if e.PendingOrder.Kind != PendingRelease || !e.HasAttackTarget || e.AttackPhase != old.AttackPhase || e.AttackCountdown != old.AttackCountdown {
				t.Fatalf("phase %d standing=%t: release dropped the recovering cycle: %+v", ph, standing, e)
			}
			back := worldRoundTripForTest(t, w)
			lost := worldRoundTripForTest(t, w)
			lost.entities[0].PendingOrder = PendingOrder{}
			if lost.Hash() == w.Hash() {
				t.Fatal("release marker does not discriminate the world hash")
			}
			ended := false
			for range 100 {
				Step(w, nil)
				Step(back, nil)
				Step(lost, nil)
				if w.Hash() != back.Hash() {
					t.Fatal("cold release changed continuation")
				}
				if w.entities[0].PendingOrder.Kind == PendingNone {
					ended = true
					break
				}
			}
			if !ended || w.entities[0].HasAttackTarget || w.entities[1].HP != 195 || !lost.entities[0].HasAttackTarget {
				t.Fatalf("phase %d standing=%t: release left a victim or lost the blow: ended=%t hp=%d", ph, standing, ended, w.entities[1].HP)
			}
		}
	}
}

// A pickup completion keeps its record beside the recovering cycle and
// completes once the cycle has ended. Control: a copy with the completion record
// removed does not complete, and its cycle runs on.
func TestRecoveryPickupCompletionWaitsForTheCycle(t *testing.T) {
	for _, ph := range recoveryPhases {
		w := lcPlayerWorld(t)
		w.sacks = []Sack{{X: 20, Y: 20, Gold: 500}}
		recoveryAdvance(t, w, ph)
		Step(w, []Command{PickUp(1, CellPoint{X: 20, Y: 20})})
		if e := w.entities[0]; e.PendingOrder.Kind != PendingPickup || !e.HasAttackTarget {
			t.Fatalf("phase %d: pickup request lost the cycle", ph)
		}
		back := worldRoundTripForTest(t, w)
		if !w.CompleteSackPickup(1) || !back.CompleteSackPickup(1) {
			t.Fatal("pickup completion refused")
		}
		if e := w.entities[0]; e.PendingOrder.Kind != PendingPickupComplete || !e.HasAttackTarget || e.AttackPhase == AttackReady {
			t.Fatalf("phase %d: pickup completion dropped the recovering cycle: %+v", ph, e)
		}
		lost := worldRoundTripForTest(t, w)
		lost.entities[0].PendingOrder = PendingOrder{}
		if lost.Hash() == w.Hash() {
			t.Fatal("completion record does not discriminate the world hash")
		}
		done := false
		for range 100 {
			Step(w, nil)
			Step(back, nil)
			Step(lost, nil)
			if w.Hash() != back.Hash() {
				t.Fatal("pickup completion cold continuation differs")
			}
			if e := w.entities[0]; e.PendingOrder.Kind == PendingNone && e.ActorState != actorStatePickupComplete {
				done = true
				break
			}
		}
		if e := w.entities[0]; !done || e.HasAttackTarget || w.entities[1].HP != 195 {
			t.Fatalf("phase %d: completion lost or a second blow loaded: done=%t hp=%d", ph, done, w.entities[1].HP)
		}
		if lost.entities[0].PendingOrder.Kind != PendingNone || !lost.entities[0].HasAttackTarget && lost.entities[1].HP == 195 {
			t.Fatalf("phase %d: control did not run on without the completion record", ph)
		}
	}
}

// A manual cast, a scroll and a Teleport approach given while a release marker
// or a pickup completion is held beside the recovering cycle replace the cycle
// and the marker at once, as they do with no marker. Control: the marker is
// present before the order, and a pending cast row is never written.
func TestRecoveryCastScrollAndApproachReplaceAHeldMarker(t *testing.T) {
	markers := []struct {
		name string
		set  func(t *testing.T, w *World)
		kind uint8
	}{
		{"release", func(t *testing.T, w *World) { w.releaseAttack(0) }, PendingRelease},
		{"completion", func(t *testing.T, w *World) {
			if !w.CompleteSackPickup(1) {
				t.Fatal("pickup completion refused")
			}
		}, PendingPickupComplete},
	}
	orders := []string{"unit cast", "cell cast", "teleport approach", "scroll"}
	for _, m := range markers {
		for _, ph := range recoveryPhases {
			for _, order := range orders {
				t.Run(fmt.Sprintf("%s/%d/%s", m.name, ph, order), func(t *testing.T) {
					var w *World
					if order == "scroll" {
						w = recoveryScroll(t, ph)
					} else {
						w = recoveryMage(t, ph)
						w.entities[0].Owner = SelfSlot
						for k := range w.spells {
							if w.spells[k].ID == teleportSpellID {
								w.spells[k].MaxRange = 1
							}
						}
					}
					m.set(t, w)
					if e := w.entities[0]; e.PendingOrder.Kind != m.kind || !e.HasAttackTarget || e.AttackPhase != ph {
						t.Fatalf("%s/%d/%s: marker not held beside the cycle: %+v", m.name, ph, order, e.PendingOrder)
					}
					var cmd Command
					switch order {
					case "unit cast":
						cmd = Cast(1, 2, 1)
					case "cell cast":
						cmd = CastAt(1, 26, CellPoint{X: 6, Y: 3})
					case "teleport approach":
						cmd = CastAt(1, 26, CellPoint{X: 15, Y: 15})
					default:
						cmd = UseScroll(1, 0, 2)
					}
					Step(w, []Command{cmd})
					e := w.entities[0]
					if e.PendingOrder.Kind != PendingNone {
						t.Fatalf("%s/%d/%s: the order queued behind the marker: %+v", m.name, ph, order, e.PendingOrder)
					}
					switch order {
					case "scroll":
						if len(w.scrollCasts) != 1 || e.HasAttackTarget {
							t.Fatalf("%s/%d: the scroll did not replace the cycle: casts=%d %+v", m.name, ph, len(w.scrollCasts), e)
						}
					case "teleport approach":
						if len(w.bookCasts) != 1 || w.bookCasts[0].Phase != bookApproach {
							t.Fatalf("%s/%d: the Teleport approach was refused: %+v", m.name, ph, w.bookCasts)
						}
					default:
						if len(w.bookCasts) != 1 {
							t.Fatalf("%s/%d/%s: the cast did not start at once: %+v", m.name, ph, order, w.bookCasts)
						}
					}
				})
			}
		}
	}
}

// Hold stores pending order 0 and keeps the body: each queued row is replaced,
// the cycle lands on its victim and no second one loads. Control: the same
// world without Hold runs the row.
func TestRecoveryHoldReplacesQueuedRowsAndKeepsBody(t *testing.T) {
	type fix struct {
		name  string
		build func(ph AttackPhase) *World
		queue func(w *World)
		ran   func(w *World) bool
	}
	fixes := []fix{
		{"release", func(ph AttackPhase) *World {
			w := lcPlayerWorld(t)
			recoveryAdvance(t, w, ph)
			return w
		}, func(w *World) { w.releaseAttack(0) }, func(w *World) bool { return !w.entities[0].HasAttackTarget }},
		{"pickup", func(ph AttackPhase) *World {
			w := lcPlayerWorld(t)
			w.sacks = []Sack{{X: 20, Y: 20, Gold: 500}}
			recoveryAdvance(t, w, ph)
			return w
		}, func(w *World) { Step(w, []Command{PickUp(1, CellPoint{X: 20, Y: 20})}) }, func(w *World) bool {
			return w.entities[0].PendingOrder.Kind == PendingPickup && w.entities[0].PendingOrder.RowAdmitted
		}},
	}
	for _, f := range fixes {
		for _, ph := range recoveryPhases {
			w := f.build(ph)
			f.queue(w)
			if w.entities[0].PendingOrder.Kind == PendingNone {
				t.Fatalf("%s phase %d: fixture queued nothing", f.name, ph)
			}
			control := worldRoundTripForTest(t, w)
			if control.entities[0].PendingOrder.Kind != w.entities[0].PendingOrder.Kind {
				t.Fatalf("%s phase %d: control lost the queued row", f.name, ph)
			}
			Step(w, []Command{GroupStance(1, OrderStandGround, SelfSlot)})
			e := w.entities[0]
			if e.PendingOrder.Kind != PendingNone || !e.HasAttackTarget && (ph == AttackRelaxing || f.name == "release") {
				t.Fatalf("%s phase %d: Hold left the row or dropped the body: pending=%+v attack=%t phase=%d", f.name, ph, e.PendingOrder, e.HasAttackTarget, e.AttackPhase)
			}
			back := worldRoundTripForTest(t, w)
			held, free, markerRan := false, false, false
			for range 150 {
				Step(w, nil)
				Step(back, nil)
				Step(control, nil)
				held, free = held || f.ran(w), free || f.ran(control)
				markerRan = markerRan || control.entities[0].PendingOrder.Kind == PendingNone
				if w.Hash() != back.Hash() {
					t.Fatalf("%s phase %d: cold Hold continuation differs", f.name, ph)
				}
			}
			if f.name == "release" {
				if !markerRan {
					t.Fatalf("release phase %d: control never ran its marker", ph)
				}
				continue
			}
			if held {
				t.Fatalf("%s phase %d: the row ran after Hold", f.name, ph)
			}
			if !free {
				t.Fatalf("%s phase %d: control never ran its row", f.name, ph)
			}
		}
	}
}
