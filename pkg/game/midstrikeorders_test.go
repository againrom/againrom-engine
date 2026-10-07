package game

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func midStrikeCastFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := castOrderFront(t)
	actors := f.live.world.Entities()
	actors[0].HP = 40
	actors[0].HealthRegenPeriod, actors[1].HealthRegenPeriod = 0, 0
	actors[0].AttackCharge, actors[0].AttackRelax, actors[0].Reach = 40, 30, 4
	actors[0].DamageBase, actors[0].AlwaysHits = 5, true
	actors = append(actors, sim.Entity{ID: 3, Owner: 2, TypeID: 1, X: 11, Y: 10, HP: 200, MaxHP: 200, Capacity: 300, TokenSize: 1, Absorption: 10})
	var rel sim.Relations
	rel.Set(1, 2, 1)
	rel.Set(2, 1, 2)
	w, err := sim.NewStockedSpelledWorld(17, f.live.world.Bounds(), sim.ModeCanonical, sim.Terrain{}, actors, nil, rel, nil, nil, mapload.SpellRules(f.Table))
	if err != nil {
		t.Fatal(err)
	}
	f.live.world, f.live.mission.state.World = w, w
	sim.Step(w, []sim.Command{sim.Attack(1, 3)})
	return f
}

func midStrikeSAVControl(t *testing.T, f *FrontEnd, doc sav.DocumentData, change func(*sim.ActorContinuation)) *FrontEnd {
	t.Helper()
	return midStrikeSAVActionsControl(t, f, doc, func(a *currentActionData) {
		for i := range a.Actions.Actors {
			if a.Actions.Actors[i].Entity == 1 {
				change(&a.Actions.Actors[i])
			}
		}
	})
}

func midStrikeSAVActionsControl(t *testing.T, f *FrontEnd, doc sav.DocumentData, change func(*currentActionData)) *FrontEnd {
	t.Helper()
	loss, err := sav.CloneDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&loss)
	if err != nil || a == nil {
		t.Fatal(err)
	}
	change(a)
	payload, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&loss.State, payload); err != nil {
		t.Fatal(err)
	}
	raw, err := sav.EncodeDocumentData(loss)
	if err != nil {
		t.Fatal(err)
	}
	return openCurrentEffectSave(t, f, raw)
}

func midStrikeScrollFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := midStrikeCastFront(t)
	actors := f.live.world.Entities()
	actors[0].WeaponSpell, actors[0].WeaponSpellLevel, actors[0].WeaponSpellSource = 9, 77, sim.WeaponSpellLegacy
	item := sim.PlainItem(0xe10)
	item.Kind, item.Effects = 4, []sim.ItemEffect{{Kind: 41, Operand: 6 | 10<<16}}
	w, err := sim.NewStockedSpelledWorld(17, f.live.world.Bounds(), sim.ModeCanonical, sim.Terrain{}, actors, nil, f.live.world.Relations(), nil, []sim.Stock{{ID: 1, ItemInstances: []sim.ItemInstance{item, item}}}, mapload.SpellRules(f.Table))
	if err != nil {
		t.Fatal(err)
	}
	actions := f.live.world.Actions()
	actions.Actors[0].Current.WeaponSpell = 9
	actions.Actors[0].Current.WeaponSpellLevel = 77
	actions.Actors[0].Current.WeaponSpellSource = sim.WeaponSpellLegacy
	if err := w.RestoreActions(actions, nil); err != nil {
		t.Fatal(err)
	}
	f.live.world, f.live.mission.state.World = w, w
	return f
}

func TestMidStrikeScrollCurrentSAVKeepsReservationBesidePhysicalBody(t *testing.T) {
	f := midStrikeScrollFront(t)
	w := f.live.world
	f.live.pending = []sim.Command{sim.UseScroll(1, 0, 2)}
	f.live.tick()
	if e := pendingVictimEntity(t, w, 1); e.PendingOrder.Kind != sim.PendingScroll || e.WeaponSpell != 9 || e.AttackTarget != 3 || len(w.ScrollCasts()) != 1 || w.ScrollCasts()[0].Started {
		t.Fatalf("pending scroll pointer replaced active victim/spell: actor=%+v scrolls=%+v", e, w.ScrollCasts())
	}
	raw, doc, a := saveCurrentEffect(t, f)
	if id, _, ok := sim.ScrollSpell(a.Actions.Scrolls[0].Item); !ok || id != 6 {
		t.Fatal("current pointer lost requested scroll spell")
	}
	r := generatedActorRecord(t, &doc, 1)
	if savedRecordValueForTest(t, *r, "U5C") != savedRecordValueForTest(t, *generatedActorRecord(t, &doc, 3), "Identity") || savedRecordValueForTest(t, *r, "U6C") != uint32(pendingVictimEntity(t, w, 1).AttackCountdown) || savedRecordRawForTest(t, *r, "U158")[9] != 1 {
		t.Fatal("scroll SAVE changed old physical operands")
	}
	cold := openCurrentEffectSave(t, f, raw)
	loss := midStrikeSAVActionsControl(t, f, doc, func(a *currentActionData) { a.Actions.Scrolls[0].Target = 1 })
	first := false
	for range 160 {
		sim.Step(w, nil)
		sim.Step(cold.live.world, nil)
		sim.Step(loss.live.world, nil)
		assertCurrentWorldEqual(t, w, cold.live.world, "pending scroll cold next action")
		if casts := w.ScrollCasts(); len(casts) != 0 && casts[0].Started && !first {
			first = true
			if pendingVictimEntity(t, w, 3).HP != 195 || pendingVictimEntity(t, w, 1).HasAttackTarget {
				t.Fatal("scroll dispatch preceded old strike/recovery")
			}
		}
	}
	if !first || pendingVictimEntity(t, w, 2).HP <= 40 || pendingVictimEntity(t, loss.live.world, 2).HP != 40 || pendingVictimEntity(t, loss.live.world, 1).HP <= pendingVictimEntity(t, w, 1).HP {
		t.Fatal("requested-only scroll endpoint control did not change the next application")
	}
}

func TestMidStrikeKnownLogicalScrollRowsCurrentSAVKeepPointerAndActiveSpell(t *testing.T) {
	for _, ready := range []bool{false, true} {
		f := midStrikeScrollFront(t)
		w := f.live.world
		actions := w.Actions()
		actions.Actors[0].ActorState = 0x16
		actions.Actors[0].Retreat = &sim.RetreatContinuation{Known: true}
		if ready {
			actions.Actors[0].AttackPhase, actions.Actors[0].AttackCountdown = sim.AttackReady, 0
			actions.Actors[0].Retreat.Progress, actions.Actors[0].Retreat.Counter, actions.Actors[0].Retreat.Complete = 1, 1, true
		}
		if err := w.RestoreActions(actions, nil); err != nil {
			t.Fatal(err)
		}
		before := pendingVictimEntity(t, w, 1)
		f.live.pending = []sim.Command{sim.UseScroll(1, 0, 2)}
		f.live.tick()
		h := pendingVictimEntity(t, w, 1)
		if h.PendingOrder.Kind != sim.PendingScroll || h.PendingOrder.RowAdmitted == ready || h.AttackPhase != before.AttackPhase || h.AttackCountdown != before.AttackCountdown || h.WeaponSpell != 9 || len(w.ScrollCasts()) != 1 || w.ScrollCasts()[0].Started {
			t.Fatal("scroll pointer setter conflated logical progress, active Spell and physical carrier")
		}
		raw, doc, supplement := saveCurrentEffect(t, f)
		cold := openCurrentEffectSave(t, f, raw)
		assertCurrentWorldEqual(t, w, cold.live.world, "logical scroll request SAVE cut")
		r := generatedActorRecord(t, &doc, 1)
		pointer, _ := savedObjectRefs(r, "U68")
		reserved := supplement.Actions.Scrolls
		if len(reserved) != 1 || len(pointer) != 1 || pointer[0] != 0 {
			t.Fatal("current scroll pointer was not stored before logical row admission")
		}
		if id, _, ok := sim.ScrollSpell(reserved[0].Item); !ok || id != 6 || reserved[0].Started {
			t.Fatal("current scroll pointer was not stored before logical row admission")
		}
		if ready {
			order := savedRecordRawForTest(t, *r, "U158")
			if order[8] != 0 || order[9] != 1 || savedRecordValueForTest(t, *r, "U5C") != savedRecordValueForTest(t, *generatedActorRecord(t, &doc, 3), "Identity") {
				t.Fatal("physical Ready bypassed logical scroll progress")
			}
			incomplete := midStrikeSAVControl(t, f, doc, func(row *sim.ActorContinuation) { row.Retreat.Complete = false })
			for range 2 {
				sim.Step(w, nil)
				sim.Step(cold.live.world, nil)
				sim.Step(incomplete.live.world, nil)
				assertCurrentWorldEqual(t, w, cold.live.world, "logical completion to scroll row")
			}
			if !pendingVictimEntity(t, w, 1).PendingOrder.RowAdmitted || pendingVictimEntity(t, incomplete.live.world, 1).PendingOrder.RowAdmitted || incomplete.live.world.ScrollCasts()[0].Started {
				t.Fatal("completion-only loss did not govern first scroll row")
			}
			raw, doc, _ = saveCurrentEffect(t, f)
			cold = openCurrentEffectSave(t, f, raw)
			r = generatedActorRecord(t, &doc, 1)
		}
		order := savedRecordRawForTest(t, *r, "U158")
		if order[8] != 8 || order[9] != 2 || binary.LittleEndian.Uint32(savedRecordRawForTest(t, *r, "U54")) != 0xd || savedRecordValueForTest(t, *r, "U5C") != savedRecordValueForTest(t, *generatedActorRecord(t, &doc, 2), "Identity") || savedRecordValueForTest(t, *r, "U64") != 0 || pendingVictimEntity(t, w, 1).WeaponSpell != 9 {
			t.Fatal("admitted scroll row substituted the active Spell or lost requested row stores")
		}
		assertCurrentWorldEqual(t, w, cold.live.world, "logical scroll admitted SAVE cut")
		requested := midStrikeSAVActionsControl(t, f, doc, func(a *currentActionData) { a.Actions.Scrolls[0].Target = 1 })
		for _, next := range []*sim.World{w, cold.live.world, requested.live.world} {
			sim.Step(next, nil)
			if len(next.ScrollCasts()) != 1 || !next.ScrollCasts()[0].Started || pendingVictimEntity(t, next, 3).HP != 200 {
				t.Fatal("first scroll execution reapplied logically completed physical carrier")
			}
		}
		assertCurrentWorldEqual(t, w, cold.live.world, "logical scroll first execution")
		applied := false
		for range 100 {
			sim.Step(w, nil)
			sim.Step(cold.live.world, nil)
			sim.Step(requested.live.world, nil)
			assertCurrentWorldEqual(t, w, cold.live.world, "logical scroll first application")
			if pendingVictimEntity(t, w, 2).HP > 40 {
				applied = true
				break
			}
		}
		if !applied || pendingVictimEntity(t, requested.live.world, 2).HP != 40 || pendingVictimEntity(t, requested.live.world, 1).HP <= 40 || pendingVictimEntity(t, w, 3).HP != 200 {
			t.Fatal("requested scroll operand did not determine first application")
		}
	}
}

func TestMidStrikeCurrentSAVKeepsRequestedCastActiveVictimAndClock(t *testing.T) {
	for _, cellCast := range []bool{false, true} {
		f := midStrikeCastFront(t)
		cmd := sim.Cast(1, 2, 6)
		if cellCast {
			cmd = sim.CastAt(1, 3, sim.CellPoint{X: 14, Y: 10})
		}
		f.live.pending = []sim.Command{cmd}
		f.live.tick()
		w := f.live.world
		h := pendingVictimEntity(t, w, 1)
		if h.AttackTarget != 3 || h.AttackPhase != sim.AttackCharging || h.AttackCountdown < 2 || h.PendingOrder.Kind == sim.PendingNone || len(w.Actions().Books) != 0 {
			t.Fatal("production cast route lost loaded physical fields or admitted early")
		}
		raw, doc, _ := saveCurrentEffect(t, f)
		r := generatedActorRecord(t, &doc, 1)
		order := savedRecordRawForTest(t, *r, "U158")
		victimKey := savedRecordValueForTest(t, *generatedActorRecord(t, &doc, 3), "Identity")
		if savedRecordValueForTest(t, *r, "U5C") != victimKey || savedRecordValueForTest(t, *r, "U6C") != uint32(h.AttackCountdown) || order[9] != 1 || order[8] != 0 || binary.LittleEndian.Uint32(order[0x30:]) == 0 {
			t.Fatal("SAV conflated pending cast with active physical body")
		}
		cold := openCurrentEffectSave(t, f, raw)
		assertCurrentWorldEqual(t, w, cold.live.world, "pending cast cold LOAD")
		requestedLoss := midStrikeSAVControl(t, f, doc, func(row *sim.ActorContinuation) { row.PendingOrder = sim.PendingOrder{} })
		activeLoss := midStrikeSAVControl(t, f, doc, func(row *sim.ActorContinuation) {
			row.HasAttackTarget = false
			row.AttackTarget = 0
			row.AttackPhase = sim.AttackReady
			row.AttackCountdown = 0
			row.AcquirePursuit = false
		})
		clockLoss := midStrikeSAVControl(t, f, doc, func(row *sim.ActorContinuation) { row.AttackCountdown = 1 })
		sim.Step(clockLoss.live.world, nil)
		if pendingVictimEntity(t, clockLoss.live.world, 3).HP != 195 || pendingVictimEntity(t, w, 3).HP != 200 {
			t.Fatal("clock-only control did not move first physical application")
		}
		landed, cellEffect, lostCellEffect, activeChecked := false, false, false, false
		for range 180 {
			sim.Step(w, nil)
			sim.Step(cold.live.world, nil)
			sim.Step(requestedLoss.live.world, nil)
			if !activeChecked {
				sim.Step(activeLoss.live.world, nil)
				if books := activeLoss.live.world.Actions().Books; len(books) != 0 && books[0].Phase == 1 {
					activeChecked = true
					if pendingVictimEntity(t, activeLoss.live.world, 3).HP != 200 {
						t.Fatal("active-only loss did not remove the first physical application")
					}
				}
			}
			cellEffect = cellEffect || len(w.CellEffects()) != 0
			lostCellEffect = lostCellEffect || len(requestedLoss.live.world.CellEffects()) != 0
			if w.Hash() != cold.live.world.Hash() {
				t.Fatal("cold continuation differs")
			}
			if len(w.Actions().Books) != 0 && w.Actions().Books[0].Phase == 1 {
				landed = true
				if pendingVictimEntity(t, w, 3).HP != 195 || pendingVictimEntity(t, w, 1).AttackPhase != sim.AttackReady {
					t.Fatal("cast dispatched before old application/recovery or old attack restarted")
				}
			}
		}
		if !landed || !activeChecked {
			t.Fatalf("active-only loss failed to remove old application: cell=%v sourceAdmitted=%v lossAdmitted=%v actor=%+v books=%+v loss=%+v", cellCast, landed, activeChecked, pendingVictimEntity(t, w, 1), w.Actions().Books, activeLoss.live.world.Actions().Books)
		}
		if cellCast {
			if !cellEffect || lostCellEffect {
				t.Fatal("requested-only loss failed to remove pending cell action")
			}
		} else if pendingVictimEntity(t, w, 2).HP <= 40 || pendingVictimEntity(t, requestedLoss.live.world, 2).HP != 40 {
			t.Fatal("requested-only loss failed to remove pending heal")
		}
	}
}

func TestMidStrikePickupCurrentSAVColdLoadCompletesProductionTransfer(t *testing.T) {
	f := midStrikeCastFront(t)
	w := f.live.world
	if err := w.ReplaceGroundSacks([]sim.Sack{{X: 10, Y: 10, Gold: 500}}); err != nil {
		t.Fatal(err)
	}
	f.live.orderPickup(1, 10, 10)
	f.live.tick()
	if len(w.Sacks()) != 1 || pendingVictimEntity(t, w, 1).PendingOrder.Kind != sim.PendingPickup {
		t.Fatal("pickup transferred before progress-zero dispatch")
	}
	raw, doc, _ := saveCurrentEffect(t, f)
	cold := openCurrentEffectSave(t, f, raw)
	loss := midStrikeSAVControl(t, f, doc, func(row *sim.ActorContinuation) { row.PendingOrder = sim.PendingOrder{} })
	for range 150 {
		f.live.tick()
		cold.live.tick()
		loss.live.tick()
		if w.Hash() != cold.live.world.Hash() {
			t.Fatal("cold pickup approach/completion differs")
		}
		if len(w.Sacks()) == 0 && pendingVictimEntity(t, w, 1).ActorState != 2 {
			break
		}
	}
	if len(w.Sacks()) != 0 || w.Purse(1) != 500 || len(loss.live.world.Sacks()) != 1 || pendingVictimEntity(t, w, 3).HP != 195 {
		t.Fatalf("pickup cold LOAD lost completion or requested-only control still transferred: sacks=%d purse=%d lossSacks=%d victimHP=%d actor=%+v", len(w.Sacks()), w.Purse(1), len(loss.live.world.Sacks()), pendingVictimEntity(t, w, 3).HP, pendingVictimEntity(t, w, 1))
	}
}

func TestMidStrikeUnitCastBothSAVCutsKeepOrderedResumeEndpoint(t *testing.T) {
	for _, target := range []sim.EntityID{1, 2} {
		f := midStrikeCastFront(t)
		f.live.pending = []sim.Command{sim.Cast(1, target, 6)}
		f.live.tick()
		raw, doc, _ := saveCurrentEffect(t, f)
		cold := openCurrentEffectSave(t, f, raw)
		h := pendingVictimEntity(t, cold.live.world, 1)
		if !h.HasAttackTarget || h.AttackTarget != 3 || h.PendingOrder.Target != target {
			t.Fatal("pending SAVE cut replaced ordered resume endpoint with requested cast endpoint")
		}
		requested := binary.LittleEndian.Uint32(savedRecordRawForTest(t, *generatedActorRecord(t, &doc, 1), "U158")[0x28:])
		if requested != savedRecordValueForTest(t, *generatedActorRecord(t, &doc, target), "Identity") {
			t.Fatal("pending cast target is absent from ordinary SAV operands")
		}
		admitted := false
		for range 160 {
			sim.Step(f.live.world, nil)
			sim.Step(cold.live.world, nil)
			assertCurrentWorldEqual(t, f.live.world, cold.live.world, "pending cast next action")
			if b, ok := castOrderBook(f.live.world, 1); ok && b.Phase == 1 {
				admitted = true
				break
			}
		}
		if !admitted || pendingVictimEntity(t, f.live.world, 3).HP != 195 {
			t.Fatalf("unit cast did not admit exactly after old strike: target=%d admitted=%v victimHP=%d actor=%+v books=%+v", target, admitted, pendingVictimEntity(t, f.live.world, 3).HP, pendingVictimEntity(t, f.live.world, 1), f.live.world.Actions().Books)
		}
		raw, doc, a := saveCurrentEffect(t, f)
		if len(a.Actions.Books) != 1 || a.Actions.Books[0].Target != target {
			t.Fatal("admitted SAVE cut lacks requested book target")
		}
		for _, row := range a.Actions.Actors {
			if row.Entity == 1 && (!row.HasAttackTarget || row.AttackTarget != 3 || row.PendingOrder.Kind != sim.PendingNone) {
				t.Fatal("admitted SAVE cut lost distinct resume endpoint")
			}
		}
		if savedRecordValueForTest(t, *generatedActorRecord(t, &doc, 1), "U5C") != savedRecordValueForTest(t, *generatedActorRecord(t, &doc, target), "Identity") {
			t.Fatal("ordinary active cast target did not differ from supplemental resume endpoint")
		}
		cold = openCurrentEffectSave(t, f, raw)
		resume := false
		for range 160 {
			sim.Step(f.live.world, nil)
			sim.Step(cold.live.world, nil)
			assertCurrentWorldEqual(t, f.live.world, cold.live.world, "admitted cast next action")
			if _, casting := castOrderBook(f.live.world, 1); casting || pendingVictimEntity(t, f.live.world, 1).CastWait != 0 {
				if pendingVictimEntity(t, f.live.world, 3).HP != 195 {
					t.Fatal("another physical strike preceded cast completion")
				}
				continue
			}
			h := pendingVictimEntity(t, f.live.world, 1)
			if h.AttackPhase == sim.AttackCharging {
				resume = h.HasAttackTarget && h.AttackTarget == 3
				break
			}
		}
		if !resume {
			t.Fatal("unit cast resumed its cast target instead of ordered victim")
		}
	}
}

func TestMidStrikePickupCompletionCurrentSAVKeepsSeparateStores(t *testing.T) {
	f := midStrikeCastFront(t)
	w := f.live.world
	before := pendingVictimEntity(t, w, 1)
	if !w.CompleteSackPickup(1) {
		t.Fatal("completion setter refused")
	}
	raw, doc, _ := saveCurrentEffect(t, f)
	r := generatedActorRecord(t, &doc, 1)
	order := savedRecordRawForTest(t, *r, "U158")
	if binary.LittleEndian.Uint32(savedRecordRawForTest(t, *r, "U50")) != 0xc || order[8] != 0 || binary.LittleEndian.Uint32(order[0x50:]) != 1 || savedRecordValueForTest(t, *r, "U5C") != savedRecordValueForTest(t, *generatedActorRecord(t, &doc, 3), "Identity") || savedRecordValueForTest(t, *r, "U6C") != uint32(before.AttackCountdown) {
		t.Fatal("completion state/pending/completion stores destroyed active body operands")
	}
	cold := openCurrentEffectSave(t, f, raw)
	assertCurrentWorldEqual(t, w, cold.live.world, "pickup completion SAVE cut")
	completed := false
	for range 160 {
		sim.Step(w, nil)
		sim.Step(cold.live.world, nil)
		assertCurrentWorldEqual(t, w, cold.live.world, "pickup completion next action")
		if e := pendingVictimEntity(t, w, 1); e.PendingOrder.Kind == sim.PendingNone && e.ActorState != 2 {
			completed = true
			if e.HasAttackTarget || pendingVictimEntity(t, w, 3).HP != 195 {
				t.Fatal("completion discarded old strike or started another before dispatch")
			}
			break
		}
	}
	if !completed {
		t.Fatal("completion did not dispatch after physical progress")
	}
}

func TestMidStrikeSnapshotPreservesNestedBlockingMask(t *testing.T) {
	entities := []sim.Entity{{ID: 1, X: 3, Y: 3, HP: 100, MaxHP: 100, AttackCharge: 10, AttackTarget: 2, HasAttackTarget: true, AttackPhase: sim.AttackCharging, AttackCountdown: 5, PendingOrder: sim.PendingOrder{Kind: sim.PendingRelease}}, {ID: 2, X: 4, Y: 3, HP: 100, MaxHP: 100}}
	makeWorld := func(mask uint32) *sim.World {
		w, err := sim.NewStructuredWorld(7, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, sim.Terrain{}, entities, nil, sim.Relations{}, nil, nil, nil, sim.GhostTemplate{}, []sim.Structure{{ID: 3, Col: 6, Row: 6, Width: 1, Height: 1, Attach: 1, Blocking: mask}})
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	old := makeWorld(2)
	if err := old.RestoreActorTraversal([]sim.EntityID{2, 1}); err != nil {
		t.Fatal(err)
	}
	raw, err := old.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if raw[0] != 102 || !sim.HasStructureBlockingForm(raw) {
		t.Fatal("ORD1/TAC1 hid SBK1")
	}
	ms := &Mission{World: makeWorld(1)}
	if err := resumeWorld(ms, &Snapshot{World: raw}, nil); err != nil {
		t.Fatal(err)
	}
	if ms.World.Structures()[0].Blocking != 2 || ms.World.Entities()[0].PendingOrder.Kind != sim.PendingRelease {
		t.Fatal("Snapshot resume replaced current blocking or pending state with mission defaults")
	}
}

func TestMidStrikeKnownLogicalRowsCurrentSAVRetainSeparatePhysicalCarrier(t *testing.T) {
	for _, ready := range []bool{false, true} {
		f := midStrikeCastFront(t)
		w := f.live.world
		a := w.Actions()
		a.Actors[0].ActorState = 0x16
		a.Actors[0].Retreat = &sim.RetreatContinuation{Known: true}
		if ready {
			a.Actors[0].AttackPhase, a.Actors[0].AttackCountdown = sim.AttackReady, 0
			a.Actors[0].Retreat.Progress, a.Actors[0].Retreat.Counter, a.Actors[0].Retreat.Complete = 1, 1, true
		}
		if err := w.RestoreActions(a, nil); err != nil {
			t.Fatal(err)
		}
		before := pendingVictimEntity(t, w, 1)
		f.live.pending = []sim.Command{sim.Cast(1, 2, 6)}
		f.live.tick()
		h := pendingVictimEntity(t, w, 1)
		if h.AttackTarget != 3 || h.AttackPhase != before.AttackPhase || h.AttackCountdown != before.AttackCountdown || h.PendingOrder.RowAdmitted == ready || len(w.Actions().Books) != 0 || h.Retreat.Progress != before.Retreat.Progress || h.Retreat.Complete != before.Retreat.Complete {
			t.Fatalf("setter replaced independent logical/physical fields: ready=%v before=%+v after=%+v", ready, before, h)
		}
		raw, doc, _ := saveCurrentEffect(t, f)
		cold := openCurrentEffectSave(t, f, raw)
		assertCurrentWorldEqual(t, w, cold.live.world, "known logical request SAVE cut")
		if ready {
			actor := generatedActorRecord(t, &doc, 1)
			order := savedRecordRawForTest(t, *actor, "U158")
			if order[8] != 0 || order[9] != 1 || savedRecordValueForTest(t, *actor, "U5C") != savedRecordValueForTest(t, *generatedActorRecord(t, &doc, 3), "Identity") {
				t.Fatal("physical Ready bypassed known nonzero logical progress on SAVE")
			}
			incomplete := midStrikeSAVControl(t, f, doc, func(row *sim.ActorContinuation) { row.Retreat.Complete = false })
			for range 2 {
				sim.Step(w, nil)
				sim.Step(cold.live.world, nil)
				sim.Step(incomplete.live.world, nil)
				assertCurrentWorldEqual(t, w, cold.live.world, "known completion to row admission")
			}
			if !pendingVictimEntity(t, w, 1).PendingOrder.RowAdmitted || pendingVictimEntity(t, incomplete.live.world, 1).PendingOrder.RowAdmitted || pendingVictimEntity(t, incomplete.live.world, 1).Retreat.Progress != 1 || len(incomplete.live.world.Actions().Books) != 0 {
				t.Fatal("completion-only control did not govern first row admission")
			}
			raw, doc, _ = saveCurrentEffect(t, f)
			cold = openCurrentEffectSave(t, f, raw)
		}
		actor := generatedActorRecord(t, &doc, 1)
		order := savedRecordRawForTest(t, *actor, "U158")
		if order[8] != 8 || order[9] != 2 || binary.LittleEndian.Uint32(savedRecordRawForTest(t, *actor, "U54")) != 0xd || savedRecordValueForTest(t, *actor, "U5C") != savedRecordValueForTest(t, *generatedActorRecord(t, &doc, 2), "Identity") || savedRecordValueForTest(t, *actor, "U64") == 0 {
			t.Fatal("known logical zero did not install ordinary actor row stores")
		}
		h = pendingVictimEntity(t, w, 1)
		if h.AttackTarget != 3 || h.AttackPhase != before.AttackPhase || h.AttackCountdown != before.AttackCountdown || h.Retreat.Progress != 0 || !h.PendingOrder.RowAdmitted || len(w.Actions().Books) != 0 {
			t.Fatal("admitted row replaced retained physical carrier before execution")
		}
		assertCurrentWorldEqual(t, w, cold.live.world, "admitted row SAVE cut")
		requested := midStrikeSAVControl(t, f, doc, func(row *sim.ActorContinuation) { row.PendingOrder.Target = 1 })
		clock := midStrikeSAVControl(t, f, doc, func(row *sim.ActorContinuation) { row.AttackCountdown = 1 })
		landed := false
		for range 160 {
			sim.Step(w, nil)
			sim.Step(cold.live.world, nil)
			sim.Step(requested.live.world, nil)
			sim.Step(clock.live.world, nil)
			assertCurrentWorldEqual(t, w, cold.live.world, "admitted row next execution")
			if pendingVictimEntity(t, w, 2).HP > 40 {
				landed = true
				break
			}
		}
		if !landed || pendingVictimEntity(t, requested.live.world, 2).HP != 40 || pendingVictimEntity(t, requested.live.world, 1).HP <= 40 || pendingVictimEntity(t, w, 3).HP != 200 || pendingVictimEntity(t, clock.live.world, 3).HP != 200 {
			t.Fatalf("admitted row effect reapplied old physical carrier or lost requested-only control: ready=%v landed=%v actor=%+v requested=%+v", ready, landed, pendingVictimEntity(t, w, 1), pendingVictimEntity(t, requested.live.world, 1))
		}
	}
}

func TestMidStrikeKnownZeroPickupRowSAVTransferCompletionAndNextDecision(t *testing.T) {
	f := midStrikeCastFront(t)
	w := f.live.world
	f.live.tick()
	a := w.Actions()
	a.Actors[0].ActorState = 0x16
	a.Actors[0].Retreat = &sim.RetreatContinuation{Known: true}
	if err := w.RestoreActions(a, nil); err != nil {
		t.Fatal(err)
	}
	if err := w.ReplaceGroundSacks([]sim.Sack{{X: 10, Y: 10, Gold: 500}}); err != nil {
		t.Fatal(err)
	}
	before := pendingVictimEntity(t, w, 1)
	sim.Step(w, []sim.Command{sim.PickUp(1, sim.CellPoint{X: 10, Y: 10})})
	if h := pendingVictimEntity(t, w, 1); !h.PendingOrder.RowAdmitted || h.AttackPhase != before.AttackPhase || h.AttackCountdown != before.AttackCountdown || len(w.Sacks()) != 1 || w.Purse(1) != 0 {
		t.Fatal("pickup row admission applied or lost the separate completed carrier")
	}
	raw, doc, _ := saveCurrentEffect(t, f)
	r := generatedActorRecord(t, &doc, 1)
	order := savedRecordRawForTest(t, *r, "U158")
	if order[8] != 7 || order[9] != 0 || binary.LittleEndian.Uint32(savedRecordRawForTest(t, *r, "U54")) != 2 || savedRecordValueForTest(t, *r, "U6C") != uint32(before.AttackCountdown) {
		t.Fatal("ordinary SAV lacks pickup row stores beside retained physical carrier")
	}
	cold := openCurrentEffectSave(t, f, raw)
	assertCurrentWorldEqual(t, w, cold.live.world, "pickup row SAVE cut")
	loss := midStrikeSAVControl(t, f, doc, func(row *sim.ActorContinuation) { row.PendingOrder.X = 14 })
	f.live.tick()
	cold.live.tick()
	loss.live.tick()
	assertCurrentWorldEqual(t, w, cold.live.world, "pickup transfer and completion")
	h := pendingVictimEntity(t, w, 1)
	if len(w.Sacks()) != 0 || w.Purse(1) != 500 || len(loss.live.world.Sacks()) != 1 || h.ActorState != 2 || h.PendingOrder.Kind != sim.PendingPickupComplete || pendingVictimEntity(t, w, 3).HP != 200 {
		t.Fatal("pickup transfer/completion conflated the next decision or reapplied old carrier")
	}
	raw, doc, _ = saveCurrentEffect(t, f)
	r = generatedActorRecord(t, &doc, 1)
	order = savedRecordRawForTest(t, *r, "U158")
	if binary.LittleEndian.Uint32(savedRecordRawForTest(t, *r, "U50")) != 0xc || order[8] != 0 || binary.LittleEndian.Uint32(order[0x50:]) != 1 {
		t.Fatal("completion SAVE cut lost separate completion stores")
	}
	cold = openCurrentEffectSave(t, f, raw)
	f.live.tick()
	cold.live.tick()
	assertCurrentWorldEqual(t, w, cold.live.world, "completion first next decision")
	if h := pendingVictimEntity(t, w, 1); h.ActorState == 2 || h.PendingOrder.Kind != sim.PendingNone || h.HasAttackTarget || pendingVictimEntity(t, w, 3).HP != 200 {
		t.Fatal("completion first decision applied completed physical carrier")
	}
}
