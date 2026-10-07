package sim

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// The registry is independently authored input to the SAV-mode dispatcher.
// Its retained membership order deliberately differs from entity order.
func savedTacticalRegistry(t *testing.T, w *World, withOrders bool) {
	t.Helper()
	g := SavedGroup{ID: 41, Selector: 27, Words: []uint16{71, 73}, Path: []uint16{0x1414}}
	g.AI[0x45] = 1
	var orders []SavedActorOrder
	for i := len(w.entities) - 1; i >= 0; i-- {
		e := w.entities[i]
		g.Members = append(g.Members, SavedGroupMember{Entity: e.ID, Bound: true})
		if withOrders {
			o := SavedActorOrder{Entity: e.ID, RepairStage: 7}
			o.Raw[0x10], o.Raw[0x11], o.Raw[0x12], o.Raw[0x13] = 0xaa, 0xbb, 0xcc, 0xdd
			o.Raw[0x68] = 93
			orders = append(orders, o)
		}
	}
	if err := w.ImportSavedGroups([]SavedGroup{g}, orders); err != nil {
		t.Fatal(err)
	}
}

func savedTacticalContinue1115(t *testing.T, w *World, ticks int) *World {
	t.Helper()
	form := mustMarshal(t, w)
	var fresh World
	if err := fresh.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(form, mustMarshal(t, &fresh)) {
		t.Fatal("native SAVE changed the tactical checkpoint")
	}
	for n := range ticks {
		Step(w, nil)
		Step(&fresh, nil)
		if w.Hash() != fresh.Hash() {
			t.Fatalf("native tactical continuation diverged at step %d", n)
		}
	}
	return &fresh
}

func TestSavedTactical1115DefendSelectedAcquireCloseAndCover(t *testing.T) {
	for _, withOrders := range []bool{false, true} {
		name := "missing order payload"
		if withOrders {
			name = "retained order payload"
		}
		t.Run(name, func(t *testing.T) {
			w := engWorld(t, engRel(t, [3]uint32{1, 3, 1}),
				laFighter(1, 1, 7, 20, 20), laFighter(2, 2, 8, 10, 20), laFighter(3, 3, 9, 21, 20))
			w.entities[2].HP, w.entities[2].MaxHP = 10000, 10000
			savedTacticalRegistry(t, w, withOrders)
			w.tick = scriptPassPhase
			Step(w, []Command{
				{Kind: KindGroupDefend, Entity: 2, X: 1, Group: 9},
				{Kind: KindGroupDefend, Entity: 1, X: 1, Group: 9},
			})
			for id, state := range map[EntityID]uint32{1: 0xc, 2: 8} {
				o := w.savedOrder(id)
				if o == nil || !o.Authored || o.State != state || o.RepairStage != 0 || o.EscortBound {
					t.Fatalf("actor %d did not replace its saved order: %+v", id, o)
				}
				if withOrders && (o.Raw[0x10] != 0xaa || o.Raw[0x13] != 0xdd || o.Raw[0x68] != 93) {
					t.Fatal("native command rewrote uninterpreted source operands")
				}
			}
			if e := w.entities[0]; !e.HasAttackTarget || e.AttackTarget != 3 || e.HasTarget || e.X != 20 {
				t.Fatalf("selected subject did not acquire in place: %+v", e)
			}
			if e := w.entities[1]; !e.HasTarget || e.TargetX != 20 || e.X <= 10 || e.HasAttackTarget || w.savedOrder(2).Raw[0x70] != 3 {
				t.Fatalf("far defender did not close at its native range: %+v", e)
			}
			g := w.savedGroupFor(2)
			if g == nil || !g.Authored || g != w.savedGroupFor(1) || g.AI[0x20] != 0 ||
				len(g.Members) != 2 || g.Members[0].Entity != 2 || g.Members[1].Entity != 1 {
				t.Fatalf("selected actors did not share one current command Group: %+v", g)
			}
			savedTacticalContinue1115(t, w, 65)
			// Only the subject owns the hostile relation. Cover must use that
			// relation after a fresh LOAD, not the defender's peaceful row.
			w.entities[1].X, w.entities[1].Y, w.entities[1].Transit = 18, 20, 0
			w.entities[1].TransitTotal = 0
			w.entities[2].X, w.entities[2].Y = 23, 20
			w.clearOrder(1)
			w.entities[1].clearAttack()
			w.tick = scriptPassPhase
			Step(w, nil)
			if e := w.entities[1]; !e.HasAttackTarget || e.AttackTarget != 3 {
				t.Fatalf("near defender did not cover the subject: %+v", e)
			}
			savedTacticalContinue1115(t, w, 33)
		})
	}
}

func TestSavedTactical1115RetreatDispatchAndOriginalUnknown(t *testing.T) {
	for _, withOrders := range []bool{false, true} {
		w := retreatWorld1089(t)
		savedTacticalRegistry(t, w, withOrders)
		w.tick = scriptPassPhase
		Step(w, retreat1089(1, 3))
		for _, id := range []EntityID{1, 3} {
			e, o := laEnt(t, w, id), w.savedOrder(id)
			if o == nil || !o.Authored || o.State != 0x16 || e.ActorState != 0x16 || e.X >= 20 || !e.HasTarget || o.Raw[8] != 1 {
				t.Fatalf("healthy SAV actor did not retreat: entity=%+v order=%+v", e, o)
			}
			if e.Withdraw != 0 || e.Wimpy != 0 {
				t.Fatal("explicit Retreat changed withdrawal thresholds")
			}
		}
		savedTacticalContinue1115(t, w, 65)
	}

	// The matching raw source number is still an unsupported continuation.
	// A native setter, not LOAD, is what grants the new dispatch authority.
	w := retreatWorld1089(t)
	savedTacticalRegistry(t, w, true)
	w.savedOrder(1).State = 0x16
	before := *w.savedOrder(1)
	w.tick = scriptPassPhase
	Step(w, nil)
	if w.entities[0].HasTarget || w.entities[0].X != 20 || !reflect.DeepEqual(before, *w.savedOrder(1)) {
		t.Fatal("raw original state 0x16 acquired native Retreat authority")
	}
	if !strings.Contains(strings.Join(w.SavedGroupIssues(), "\n"), "state 22 continuation unsupported") {
		t.Fatal("raw original state 0x16 lost its explicit unsupported boundary")
	}
	savedTacticalContinue1115(t, w, 33)
}

func TestSavedTactical1115ReplacementAndFirstRefusal(t *testing.T) {
	for _, command := range []Command{
		{Kind: KindMoveTo, Entity: 1, X: 30, Y: 20},
		{Kind: KindGroupMoveTo, Entity: 1, X: 30, Y: 20},
		{Kind: KindAttack, Entity: 1, X: 2},
		{Kind: KindGroupStance, Entity: 1, X: OrderStandGround},
		{Kind: KindGroupDefend, Entity: 1, X: 3},
	} {
		for _, defend := range []bool{false, true} {
			w := retreatWorld1089(t)
			savedTacticalRegistry(t, w, true)
			initial := retreat1089(1)
			if defend {
				initial = []Command{{Kind: KindGroupDefend, Entity: 1, X: 3}}
			}
			Step(w, initial)
			oldGroup := w.savedGroupFor(1).ID
			back := savedTacticalContinue1115(t, w, 7)
			Step(w, []Command{command})
			Step(back, []Command{command})
			if w.Hash() != back.Hash() || w.entities[0].ActorState == 0x16 || w.savedOrder(1).State == 0x16 || w.savedGroupFor(1).ID == oldGroup {
				t.Fatalf("command %d did not replace both native tactical surfaces (defend=%t)", command.Kind, defend)
			}
			if command.Kind != KindGroupDefend && (w.savedOrder(1).State == 8 || w.entities[0].HasEscortTarget) {
				t.Fatalf("command %d retained the old escort", command.Kind)
			}
			savedTacticalContinue1115(t, w, 33)
		}
	}
	w := retreatWorld1089(t)
	savedTacticalRegistry(t, w, true)
	back := savedTacticalContinue1115(t, w, 0)
	Step(w, retreat1089(999, 1))
	Step(back, nil)
	if w.Hash() != back.Hash() {
		t.Fatal("missing first selection changed SAV membership or actor order")
	}
}

func TestSavedTactical1115RetreatHiddenMemberRetainsState(t *testing.T) {
	w := retreatWorld1089(t)
	savedTacticalRegistry(t, w, true)
	Step(w, retreat1089(1, 3))
	w.takeOffMap(0)
	w.tick = scriptPassPhase
	beforeEntity, beforeOrder := w.entities[0], *w.savedOrder(1)
	members := append([]SavedGroupMember(nil), w.savedGroupFor(1).Members...)
	savedTacticalContinue1115(t, w, 65)
	if w.entities[0] != beforeEntity || !reflect.DeepEqual(beforeOrder, *w.savedOrder(1)) || !reflect.DeepEqual(members, w.savedGroupFor(1).Members) {
		t.Fatal("hidden tactical member lost presence, retained order or membership")
	}
	if w.entities[2].X >= 20 {
		t.Fatal("hidden head prevented its visible successor from retreating")
	}
}

func TestSavedTactical1115RetreatScrollOwnership(t *testing.T) {
	for _, started := range []bool{false, true} {
		targetX := int32(32)
		if started {
			targetX = 24
		}
		w := retreatScrollWorld1089(t, targetX)
		savedTacticalRegistry(t, w, true)
		originalItem := w.carried[0][0].Instance()
		Step(w, []Command{{Kind: KindUseScroll, Entity: 1, X: 2}})
		Step(w, nil)
		if len(w.scrollCasts) != 1 || w.scrollCasts[0].Started != started {
			t.Fatal("fixture has the wrong scroll progress")
		}
		beforeX := w.entities[0].X
		w.tick = scriptPassPhase
		Step(w, retreat1089(1))
		if started && (len(w.scrollCasts) != 1 || w.entities[0].X != beforeX) {
			t.Fatal("Retreat cancelled or moved through a started scroll")
		}
		if !started && (len(w.scrollCasts) != 0 || w.carried[0][0].Count != 2 || !ItemEqual(w.carried[0][0].Instance(), originalItem)) {
			t.Fatal("Retreat did not refund the reserved approach")
		}
		back := savedTacticalContinue1115(t, w, 0)
		releases := 0
		for n := range 100 {
			releases += len(StepObserved(w, nil))
			Step(back, nil)
			if w.Hash() != back.Hash() {
				t.Fatalf("scroll/Retreat continuation changed at %d", n)
			}
		}
		wantReleases, wantStock := 0, uint32(2)
		if started {
			wantReleases, wantStock = 1, 1
		}
		if releases != wantReleases || len(w.scrollCasts) != 0 || w.carried[0][0].Count != wantStock || w.entities[0].X >= beforeX {
			t.Fatalf("scroll did not finish/refund before Retreat: releases=%d stock=%d x=%d", releases, w.carried[0][0].Count, w.entities[0].X)
		}
	}
}

func TestSavedTactical1115RetreatKeepsLoadedAttackAndWithdrawalOwnership(t *testing.T) {
	w := retreatWorld1089(t)
	savedTacticalRegistry(t, w, true)
	w.entities[2].Owner = 0
	w.entities[1].X, w.entities[1].PostX = 21, 21
	w.entities[0].AttackCharge, w.entities[0].Wimpy, w.entities[0].Withdraw = 4, 100, 100
	w.orderAttack(0, 2)
	Step(w, nil)
	if w.entities[0].AttackPhase != AttackCharging {
		t.Fatal("fixture did not start attack")
	}
	w.savedOrder(1).Raw[8] = 5
	w.tick = scriptPassPhase
	Step(w, retreat1089(1))
	if e := w.entities[0]; e.ActorState != 0x16 || e.AttackPhase != AttackCharging || !e.HasAttackTarget || e.X != 20 || e.HasTarget || w.savedOrder(1).Raw[8] != 5 {
		t.Fatalf("loaded attack lost ownership to tactical/withdrawal dispatch: %+v", e)
	}
	savedTacticalContinue1115(t, w, 65)
	if w.entities[1].HP != 99 || w.entities[0].X >= 20 || w.entities[0].ActorState != 0x16 {
		t.Fatalf("expected one completed blow then retreat: hp=%d x=%d state=%d", w.entities[1].HP, w.entities[0].X, w.entities[0].ActorState)
	}
}

func TestSavedTactical1115RetreatAutocastAndTransitContinuation(t *testing.T) {
	for _, phase := range []string{"idle", "windup", "recovery"} {
		t.Run(phase, func(t *testing.T) {
			w := retreatAutocastWorld1089(t)
			savedTacticalRegistry(t, w, true)
			if phase != "idle" {
				Step(w, nil)
				if len(w.bookCasts) != 1 {
					t.Fatal("fixture did not start autocast")
				}
			}
			if phase == "recovery" {
				for n := 0; n < 40 && w.entities[0].CastWait == 0; n++ {
					Step(w, nil)
				}
				if w.entities[0].CastWait == 0 {
					t.Fatal("fixture did not reach recovery")
				}
			}
			beforeMana := w.entities[0].Mana
			Step(w, []Command{{Kind: KindGroupRetreat, Entity: 1, Player: SelfSlot}})
			back := savedTacticalContinue1115(t, w, 0)
			releases := 0
			for n := range 100 {
				releases += len(StepObserved(w, nil))
				Step(back, nil)
				if w.Hash() != back.Hash() {
					t.Fatalf("autocast/Retreat next step differs at %d", n)
				}
			}
			wantReleases := 0
			if phase == "windup" {
				wantReleases = 1
			}
			e := w.entities[0]
			if e.X >= 20 || e.ActorState != 0x16 || e.AutoSpell != 1 || releases != wantReleases || e.Mana != beforeMana-int32(wantReleases) {
				t.Fatalf("saved Retreat starved or changed loaded casting: x=%d state=%d releases=%d mana=%d", e.X, e.ActorState, releases, e.Mana)
			}
		})
	}
	w := retreatWorld1089(t)
	savedTacticalRegistry(t, w, true)
	w.entities[0].Speed, w.entities[0].Transit, w.entities[0].TransitTotal = 10, 3, 4
	w.entities[0].Wimpy, w.entities[0].Withdraw = 100, 100
	w.tick = scriptPassPhase
	Step(w, retreat1089(1))
	if e := w.entities[0]; e.X != 20 || e.Transit == 0 || e.HasTarget {
		t.Fatalf("tactical/withdrawal order interrupted a crossing: %+v", e)
	}
	savedTacticalContinue1115(t, w, 65)
	if w.entities[0].X >= 20 {
		t.Fatal("completed crossing never yielded to Retreat")
	}
}
