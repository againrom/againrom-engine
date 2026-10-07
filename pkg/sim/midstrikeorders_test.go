package sim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestMidStrikeReleaseKeepsBodyAndEndsBeforeAnotherCycle(t *testing.T) {
	for _, standing := range []bool{false, true} {
		w := lcPlayerWorld(t)
		w.relations.Set(SelfSlot, 2, 2)
		old := w.entities[0]
		if standing {
			w.entities[0].ScanRange = 0
			w.acquireStanding(0)
		} else {
			w.releaseAttack(0)
		}
		if e := w.entities[0]; e.AttackTarget != old.AttackTarget || e.AttackPhase != old.AttackPhase || e.AttackCountdown != old.AttackCountdown || !e.HasAttackTarget {
			t.Fatalf("release discarded running body: %+v", e)
		}
		back := worldRoundTripForTest(t, w)
		lost := worldRoundTripForTest(t, w)
		lost.entities[0].PendingOrder = PendingOrder{}
		if lost.Hash() == w.Hash() {
			t.Fatal("release marker does not discriminate world hash")
		}
		boundary := false
		for range 190 {
			Step(w, nil)
			Step(back, nil)
			Step(lost, nil)
			if w.Hash() != back.Hash() {
				t.Fatal("cold release changed continuation")
			}
			if w.entities[0].PendingOrder.Kind == PendingNone {
				boundary = true
				break
			}
		}
		if e := w.entities[0]; !boundary || e.HasAttackTarget || w.entities[1].HP != 195 || !lost.entities[0].HasAttackTarget || lost.entities[0].AttackPhase == AttackReady {
			t.Fatalf("release lost old blow or started extra cycle: actor=%+v hp=%d lossHP=%d", e, w.entities[1].HP, lost.entities[1].HP)
		}
	}
}

func TestMidStrikeManualCastKeepsBodyAndDispatchesAfterProgress(t *testing.T) {
	for _, cellCast := range []bool{false, true} {
		w := manualCastWorld(t)
		w.entities[0].AlwaysHits, w.entities[0].DamageBase = true, 5
		w.entities[0].Reach = 1
		w.entities[1].HP, w.entities[1].MaxHP = 200, 200
		w.orderAttack(0, 2)
		w.entities[0].AttackPhase, w.entities[0].AttackCountdown = AttackCharging, 3
		old := w.entities[0]
		cmd := Cast(1, 2, 1)
		if cellCast {
			cmd = CastAt(1, 26, CellPoint{X: 6, Y: 3})
		}
		Step(w, []Command{cmd})
		if e := w.entities[0]; !e.HasAttackTarget || e.AttackTarget != old.AttackTarget || e.AttackPhase != AttackCharging || e.AttackCountdown != 2 || len(w.bookCasts) != 0 || e.PendingOrder.Kind < PendingActorCast || e.Mana != old.Mana {
			t.Fatalf("manual setter replaced active body or admitted cast early: %+v casts=%+v", e, w.bookCasts)
		}
		back := worldRoundTripForTest(t, w)
		firstCast := false
		for range 100 {
			Step(w, nil)
			Step(back, nil)
			if w.Hash() != back.Hash() {
				t.Fatal("cold manual cast continuation differs")
			}
			if len(w.bookCasts) != 0 && w.bookCasts[0].Phase == bookCharging && !firstCast {
				firstCast = true
				if w.entities[0].HasAttackTarget || w.entities[1].HP != 195 {
					t.Fatal("manual dispatch preceded old blow/recovery or repeated attack")
				}
			}
		}
		if !firstCast || cellCast && (w.entities[0].X != 6 || w.entities[0].Y != 3) || !cellCast && w.entities[1].HP >= 195 {
			t.Fatalf("manual order missing: firstCast=%v cell=%v actor=%+v hp=%d casts=%+v", firstCast, cellCast, w.entities[0], w.entities[1].HP, w.bookCasts)
		}
	}
}

func TestMidStrikeReleaseSameEndpointRewriteAndExplicitReplacement(t *testing.T) {
	w := lcPlayerWorld(t, lcVictim(3, 2, 23, 20))
	w.releaseAttack(0)
	old := w.entities[0]
	if w.attachAttack(0, 2, AttackTargetUnit, false) || w.entities[0].PendingOrder.Kind != PendingRelease || w.entities[0].AttackCountdown != old.AttackCountdown {
		t.Fatal("same-endpoint AI rewrite erased pending release")
	}
	Step(w, []Command{Attack(1, 3)})
	if e := w.entities[0]; e.PendingOrder.Kind != PendingNone || !e.HasPendingAttackTarget || e.PendingAttackTarget != 3 || e.AttackTarget != 2 || e.AttackPhase != old.AttackPhase || e.AttackCountdown != old.AttackCountdown-1 {
		t.Fatal("admitted explicit replacement was blocked by release marker")
	}
	w = lcPlayerWorld(t, lcVictim(3, 2, 23, 20))
	w.releaseAttack(0)
	if !w.attachAttack(0, 3, AttackTargetUnit, false) || w.entities[0].AttackTarget != 3 || w.entities[0].PendingOrder.Kind != PendingNone {
		t.Fatal("distinct active endpoint writer was blanket-blocked")
	}
}

func TestMidStrikeOptionalTailKeepsTacticalBlockingWeaponEffects(t *testing.T) {
	w := scrollFixtureWorld(t, 12, 2)
	w.entities[0].WeaponSpell, w.entities[0].WeaponSpellLevel, w.entities[0].WeaponSpellSource = 1, 77, WeaponSpellLegacy
	w.entities[1].X, w.entities[1].Y = 2, 1
	w.orderAttack(0, 2)
	w.entities[0].AttackPhase, w.entities[0].AttackCountdown = AttackCasting, 3
	w.releaseAttack(0)
	w.DeclareStructures([]Structure{{ID: 1, Col: 6, Row: 6, Width: 1, Height: 1, Attach: 1, Blocking: 2}})
	if err := w.RestoreActorTraversal([]EntityID{2, 1}); err != nil {
		t.Fatal(err)
	}
	rule := SpellRule{ID: 7, Area: true, Distribution: distributionDiamond, AreaDuration: 5, DamageMin: 5, DamageMax: 5, Damaging: true, TargetsUnit: true}
	w.spells = append(w.spells, rule)
	if !w.landArea(rule, 0, 1, true, 1, 1, 5, 5, nil) {
		t.Fatal("area fixture refused")
	}
	full := mustMarshal(t, w)
	if full[0] != 102 || !HasStructureBlockingForm(full) {
		t.Fatal("nested optional form lost blocking")
	}
	base := bytes.Clone(full[:len(full)-33])
	base[0] = full[len(full)-5]
	if base[0] != 101 || string(base[len(base)-4:]) != "TAC1" {
		t.Fatal("ORD1 did not wrap TAC1")
	}
	without := *w
	without.entities = append([]Entity(nil), w.entities...)
	without.entities[0].PendingOrder = PendingOrder{}
	if !bytes.Equal(base, mustMarshal(t, &without)) {
		t.Fatal("independent ORD1 peel changed predecessor bytes")
	}
	back := worldRoundTripForTest(t, w)
	if back.entities[0].WeaponSpell != 1 || back.entities[0].WeaponSpellLevel != 77 || len(back.CellEffects()) == 0 || back.structures[0].Blocking != 2 || back.ActorTraversal()[0] != 2 {
		t.Fatal("stacked form lost active weapon, effect, blocking or traversal")
	}
	Step(w, nil)
	Step(back, nil)
	if w.Hash() != back.Hash() {
		t.Fatal("stacked form changed next active action")
	}
}

func TestMidStrikePickupOrderCompletionAndReplacement(t *testing.T) {
	w := lcPlayerWorld(t)
	w.sacks = []Sack{{X: 20, Y: 20, Gold: 500}}
	Step(w, []Command{PickUp(1, CellPoint{X: 20, Y: 20})})
	if w.entities[0].PendingOrder.Kind != PendingPickup || !w.entities[0].HasAttackTarget {
		t.Fatal("pickup intent lost running strike")
	}
	back := worldRoundTripForTest(t, w)
	if !w.CompleteSackPickup(1) || !back.CompleteSackPickup(1) {
		t.Fatal("pickup completion refused")
	}
	if w.entities[0].PendingOrder.Kind != PendingPickupComplete || !w.entities[0].HasAttackTarget {
		t.Fatal("pickup completion discarded body")
	}
	completed := false
	for range 150 {
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatal("pickup completion cold continuation differs")
		}
		if w.entities[0].PendingOrder.Kind == PendingNone && w.entities[0].ActorState != actorStatePickupComplete {
			completed = true
			break
		}
	}
	if !completed || w.entities[1].HP != 195 || w.entities[0].HasAttackTarget || w.entities[0].ActorState == actorStatePickupComplete {
		t.Fatalf("pickup completion lost or repeated loaded strike at dispatch: completed=%v actor=%+v victimHP=%d", completed, w.entities[0], w.entities[1].HP)
	}
	w = lcPlayerWorld(t)
	w.sacks = []Sack{{X: 18, Y: 20}}
	Step(w, []Command{PickUp(1, CellPoint{X: 18, Y: 20}), MoveTo(1, CellPoint{X: 17, Y: 20})})
	if w.entities[0].PendingOrder.Kind != PendingNone {
		t.Fatal("later move retained pickup intent")
	}
}

func TestMidStrikeReadyCastAndPendingReplacementKeepAdmittedBounds(t *testing.T) {
	w := manualCastWorld(t)
	Step(w, []Command{Cast(1, 2, 1)})
	if len(w.bookCasts) != 1 || w.entities[0].PendingOrder.Kind != PendingNone {
		t.Fatal("ready cast failed immediate admission")
	}
	w = manualCastWorld(t)
	w.orderAttack(0, 2)
	w.entities[0].AttackPhase, w.entities[0].AttackCountdown = AttackCharging, 10
	Step(w, []Command{Cast(1, 2, 1), CastAt(1, 26, CellPoint{X: 6, Y: 3})})
	p := w.entities[0].PendingOrder
	if p.Kind != PendingCellCast || p.Spell != 26 || p.X != 6 || w.entities[0].AttackTarget != 2 || len(w.bookCasts) != 0 {
		t.Fatal("accepted pending replacement changed active body")
	}
	Step(w, []Command{CastAt(1, 26, CellPoint{X: -1, Y: 3})})
	if w.entities[0].PendingOrder != p {
		t.Fatal("refused cast overwrote accepted pending request")
	}
	Step(w, []Command{MoveTo(1, CellPoint{X: 8, Y: 3})})
	if w.entities[0].PendingOrder.Kind != PendingNone || w.entities[0].AttackTarget != 2 || !w.entities[0].HasTarget {
		t.Fatal("later accepted move failed to cancel pending cast while retaining body")
	}
}

func TestMidStrikeScrollPointerWaitsBesideActiveWeaponSpell(t *testing.T) {
	w := scrollFixtureWorld(t, 12, 2)
	e := &w.entities[0]
	e.AttackCharge, e.AttackRelax, e.Reach = 20, 2, 1
	w.entities[1].X, w.entities[1].Y = e.X+1, e.Y
	w.orderAttack(0, 2)
	e.AttackPhase, e.AttackCountdown = AttackCharging, 3
	e.WeaponSpell, e.WeaponSpellLevel, e.WeaponSpellSource = 9, 77, WeaponSpellLegacy
	old := *e
	Step(w, []Command{UseScroll(1, 0, 2)})
	if e = &w.entities[0]; !e.HasAttackTarget || e.AttackTarget != 2 || e.AttackPhase != AttackCharging || e.WeaponSpell != old.WeaponSpell || e.WeaponSpellLevel != 77 || len(w.scrollCasts) != 1 || w.scrollCasts[0].Started {
		t.Fatal("scroll pointer changed active body/spell or refused pending admission")
	}
	back := worldRoundTripForTest(t, w)
	for range 100 {
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatal("scroll continuation differs after cold decode")
		}
	}
	if len(w.scrollCasts) != 0 || len(w.carried[0]) != 1 || w.carried[0][0].Count != 1 {
		t.Fatal("ordinary physical body did not admit the queued scroll afterwards")
	}
}

func TestMidStrikePendingOrderMalformedBytesAndHistoricalDefaults(t *testing.T) {
	w := lcPlayerWorld(t)
	w.releaseAttack(0)
	good := mustMarshal(t, w)
	if good[0] != pendingOrderFormVersion || string(good[len(good)-4:]) != "ORD1" {
		t.Fatal("pending order lacks its tagged byte form")
	}
	for _, kind := range []string{"count", "kind", "actor", "operands", "span", "version", "pad", "combination", "admitted"} {
		bad := bytes.Clone(good)
		start := len(bad) - 9 - int(binary.LittleEndian.Uint32(bad[len(bad)-9:]))
		switch kind {
		case "count":
			binary.LittleEndian.PutUint32(bad[start:], ^uint32(0))
		case "kind":
			bad[start+8] = 255
		case "actor":
			binary.LittleEndian.PutUint32(bad[start+4:], 99)
		case "operands":
			bad[start+12] = 1
		case "span":
			binary.LittleEndian.PutUint32(bad[len(bad)-9:], ^uint32(0))
		case "version":
			bad[len(bad)-5] = 102
		case "pad":
			bad[start+9] = 1
		case "combination":
			bad[start+8] = PendingScroll
		case "admitted":
			bad[start+9] |= 0x80
		}
		before := w.Hash()
		if err := w.UnmarshalBinary(bad); err == nil || w.Hash() != before {
			t.Fatalf("malformed %s admitted or mutated receiver", kind)
		}
	}
	if !bytes.Equal(mustMarshal(t, pinWorld(t)), pinBytes) {
		t.Fatal("historical bytes changed without pending state")
	}
	old := bytes.Clone(good[:len(good)-33])
	old[0] = good[len(good)-5]
	var historical World
	if err := historical.UnmarshalBinary(old); err != nil || historical.entities[0].PendingOrder != (PendingOrder{}) {
		t.Fatal("historical default is not deterministic absent pending order", err)
	}
}

func TestMidStrikeWeaponCleanupReplacesPendingScrollPointer(t *testing.T) {
	w := scrollFixtureWorld(t, 12, 2)
	e := &w.entities[0]
	e.AttackCharge, e.AttackRelax, e.Reach = 20, 2, 1
	w.entities[1].X, w.entities[1].Y = e.X+1, e.Y
	w.orderAttack(0, 2)
	e.AttackPhase, e.AttackCountdown = AttackCasting, 2
	e.WeaponSpell, e.WeaponSpellLevel, e.WeaponSpellSource = 1, 77, WeaponSpellLegacy
	Step(w, []Command{UseScroll(1, 0, 2)})
	if len(w.scrollCasts) != 1 || w.scrollCasts[0].Started || w.entities[0].WeaponSpell != 1 {
		t.Fatal("scroll pointer replaced active spell")
	}
	Step(w, nil)
	if len(w.scrollCasts) != 0 || w.entities[0].PendingOrder.Kind != PendingRelease || len(w.carried[0]) != 1 || w.carried[0][0].Count != 2 {
		t.Fatal("weapon cleanup retained the queued scroll or lost its authored refund")
	}
}

func TestMidStrikeKnownProgressRowsKeepPhysicalCarrier(t *testing.T) {
	for _, ready := range []bool{false, true} {
		w := manualCastWorld(t)
		w.entities[0].AlwaysHits, w.entities[0].DamageBase = true, 5
		w.entities[1].HP, w.entities[1].MaxHP = 200, 200
		w.orderAttack(0, 2)
		e := &w.entities[0]
		e.ActorState = actorStateRetreat
		e.Retreat = RetreatContinuation{Known: true}
		if ready {
			e.Retreat.Progress, e.Retreat.Counter, e.Retreat.Complete = 1, 1, true
		} else {
			e.AttackPhase, e.AttackCountdown = AttackCharging, 3
		}
		before := *e
		if !w.beginManualCast(0, CastAt(1, 26, CellPoint{X: 6, Y: 3})) {
			t.Fatal("manual setter refused")
		}
		if e.AttackTarget != before.AttackTarget || e.AttackPhase != before.AttackPhase || e.AttackCountdown != before.AttackCountdown || e.Retreat != before.Retreat || e.PendingOrder.RowAdmitted == ready || len(w.bookCasts) != 0 {
			t.Fatal("setter conflated row admission, retained progress and physical carrier", *e)
		}
		back := worldRoundTripForTest(t, w)
		if ready {
			incomplete := worldRoundTripForTest(t, w)
			incomplete.entities[0].Retreat.Complete = false
			for range 3 {
				Step(w, nil)
				Step(back, nil)
				Step(incomplete, nil)
				if w.Hash() != back.Hash() {
					t.Fatal("logical completion cold continuation differs")
				}
			}
			if !w.entities[0].PendingOrder.RowAdmitted || w.entities[0].Retreat.Progress != 0 || incomplete.entities[0].PendingOrder.RowAdmitted || incomplete.entities[0].Retreat.Progress != 1 || len(incomplete.bookCasts) != 0 || w.entities[1].HP != 200 {
				t.Fatal("completion-only control did not govern the first row admission")
			}
			back = worldRoundTripForTest(t, w)
		} else {
			if !w.beginManualCast(0, CastAt(1, 26, CellPoint{X: 8, Y: 3})) || w.beginManualCast(0, CastAt(1, 26, CellPoint{X: -1, Y: 3})) || !w.entities[0].PendingOrder.RowAdmitted || w.entities[0].PendingOrder.X != 8 {
				t.Fatal("admitted row replacement/refusal lost its requested operands")
			}
			back = worldRoundTripForTest(t, w)
		}
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() || len(w.bookCasts) != 1 || w.bookCasts[0].Phase != bookCharging || w.entities[1].HP != 200 {
			t.Fatal("row execution reapplied a logically completed physical carrier")
		}
	}
}

func TestMidStrikeKnownProgressScrollPointerAndRowAdmission(t *testing.T) {
	for _, ready := range []bool{false, true} {
		w := scrollFixtureWorld(t, 2, 2)
		w.entities[0].AlwaysHits, w.entities[0].DamageBase = true, 5
		w.entities[1].HP, w.entities[1].MaxHP = 200, 200
		w.orderAttack(0, 2)
		e := &w.entities[0]
		e.ActorState = actorStateRetreat
		e.Retreat = RetreatContinuation{Known: true}
		e.WeaponSpell, e.WeaponSpellLevel, e.WeaponSpellSource = 9, 77, WeaponSpellLegacy
		if ready {
			e.Retreat.Progress, e.Retreat.Counter, e.Retreat.Complete = 1, 1, true
		} else {
			e.AttackPhase, e.AttackCountdown = AttackCharging, 3
		}
		before := *e
		if !w.beginScroll(0, 0, 2, 0, 0, false) {
			t.Fatal("scroll setter refused")
		}
		if e.PendingOrder.Kind != PendingScroll || e.PendingOrder.RowAdmitted == ready || e.AttackPhase != before.AttackPhase || e.AttackCountdown != before.AttackCountdown || e.Retreat != before.Retreat || e.WeaponSpell != 9 || len(w.scrollCasts) != 1 || w.scrollCasts[0].Started || w.carried[0][0].Count != 1 {
			t.Fatal("scroll pointer store conflated active Spell, progress and row admission")
		}
		back := worldRoundTripForTest(t, w)
		if ready {
			incomplete := worldRoundTripForTest(t, w)
			incomplete.entities[0].Retreat.Complete = false
			for range 3 {
				Step(w, nil)
				Step(back, nil)
				Step(incomplete, nil)
				if w.Hash() != back.Hash() {
					t.Fatal("logical scroll completion cold continuation differs")
				}
			}
			if !w.entities[0].PendingOrder.RowAdmitted || w.scrollCasts[0].Started || incomplete.entities[0].PendingOrder.RowAdmitted || incomplete.entities[0].Retreat.Progress != 1 || incomplete.scrollCasts[0].Started || w.entities[1].HP != 200 {
				t.Fatal("completion-only control did not govern first scroll row admission")
			}
			back = worldRoundTripForTest(t, w)
		}
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() || len(w.scrollCasts) != 1 || !w.scrollCasts[0].Started || w.entities[0].PendingOrder.Kind != PendingNone || w.entities[1].HP != 200 || w.entities[0].WeaponSpell != 9 {
			t.Fatal("scroll execution reapplied completed physical carrier or replaced active Spell")
		}
	}
}

func TestMidStrikeRetreatReplacesQueuedCast(t *testing.T) {
	for _, cellCast := range []bool{false, true} {
		w := manualCastWorld(t)
		w.entities[0].Reach, w.entities[0].Speed, w.entities[0].ScanRange = 1, 10, 19
		w.entities[1].Owner = 1
		w.relations.Set(2, 1, 1)
		w.relations.Set(1, 2, 1)
		w.orderAttack(0, 2)
		w.entities[0].AttackPhase, w.entities[0].AttackCountdown = AttackCharging, 6
		cmd := Cast(1, 2, 1)
		if cellCast {
			cmd = CastAt(1, 26, CellPoint{X: 6, Y: 3})
		}
		Step(w, []Command{cmd})
		if w.entities[0].PendingOrder.Kind < PendingActorCast {
			t.Fatal("cast was not queued behind the loaded strike")
		}
		Step(w, []Command{GroupRetreat(1, 2, 7)})
		if e := w.entities[0]; e.PendingOrder.Kind != PendingNone || e.ActorState != actorStateRetreat || !e.Retreat.Known {
			t.Fatalf("retreat did not replace the queued cast: %+v", e)
		}
		back := worldRoundTripForTest(t, w)
		for range 120 {
			Step(w, nil)
			Step(back, nil)
			if w.Hash() != back.Hash() {
				t.Fatal("cold retreat continuation differs")
			}
			if len(w.bookCasts) != 0 {
				t.Fatal("queued cast started after retreat")
			}
		}
		if e := w.entities[0]; e.ActorState != actorStateRetreat || e.Mana != 1000 || e.X == 3 && e.Y == 3 {
			t.Fatalf("retreat did not persist or move: st=%d mana=%d pos=%d,%d ret=%+v tgt=%v/%d ph=%d", e.ActorState, e.Mana, e.X, e.Y, e.Retreat, e.HasTarget, e.AttackTarget, e.AttackPhase)
		}
	}
}

func TestMidStrikeRetreatReplacesQueuedPickup(t *testing.T) {
	w := lcPlayerWorld(t)
	w.sacks = []Sack{{X: 20, Y: 20, Gold: 500}}
	Step(w, []Command{PickUp(1, CellPoint{X: 20, Y: 20})})
	if w.entities[0].PendingOrder.Kind != PendingPickup {
		t.Fatal("pickup was not queued")
	}
	Step(w, []Command{GroupRetreat(1, SelfSlot, 7)})
	if e := w.entities[0]; e.PendingOrder.Kind == PendingPickup || e.ActorState != actorStateRetreat {
		t.Fatalf("retreat did not replace the queued pickup: %+v", e)
	}
	back := worldRoundTripForTest(t, w)
	for range 120 {
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatal("cold retreat continuation differs")
		}
		if e := w.entities[0]; e.PendingOrder.Kind == PendingPickup || e.PendingOrder.Kind == PendingPickupComplete {
			t.Fatalf("pickup returned after retreat: %+v", e)
		}
	}
}

func TestMidStrikeImportedProgressOnIdleReadyBodyDoesNotBlockPickup(t *testing.T) {
	w := lcPlayerWorld(t)
	w.entities[0].clearAttack()
	w.sacks = []Sack{{X: 20, Y: 20, Gold: 500}}
	w.savedGroups = &savedGroupState{Orders: []SavedActorOrder{{Entity: 1}}}
	w.savedGroups.Orders[0].Raw[9] = 1
	if w.ActorOrderProgress(1) != 0 {
		t.Fatal("idle ready body reports imported progress")
	}
	Step(w, []Command{PickUp(1, CellPoint{X: 20, Y: 20})})
	if e := w.entities[0]; e.PendingOrder.Kind != PendingPickup || !e.PendingOrder.RowAdmitted {
		t.Fatalf("pickup row not admitted: %+v", e.PendingOrder)
	}
	if w.savedGroups.Orders[0].Raw[9] != 0 {
		t.Fatal("stale imported progress survives a step")
	}
	back := worldRoundTripForTest(t, w)
	if w.Hash() != back.Hash() {
		t.Fatal("round trip changed hash")
	}
}
