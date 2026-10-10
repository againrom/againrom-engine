package sim

import (
	"bytes"
	"testing"
)

func retreat1089(ids ...EntityID) []Command {
	var out []Command
	for _, id := range ids {
		out = append(out, Command{Kind: KindGroupRetreat, Entity: id, Player: 2, Group: 91})
	}
	return out
}

func retreatWorld1089(t *testing.T) *World {
	return engWorld(t, engRel(t, [3]uint32{2, 3, 1}),
		withdrawalFighter(1, 2, 20, 20, 100),
		withdrawalFighter(2, 3, 24, 20, 100),
		withdrawalFighter(3, 2, 20, 22, 100))
}

func retreatRoundTrip1089(t *testing.T, w *World) *World {
	return worldRoundTripForTest(t, w)
}

func worldRoundTripForTest(t *testing.T, w *World) *World {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	b2, err := back.MarshalBinary()
	if err != nil || !bytes.Equal(b, b2) {
		t.Fatal("Retreat form roundtrip", err)
	}
	return &back
}

func TestPlayerRetreat1089PersistsRecomputesAndLeavesThresholdsAlone(t *testing.T) {
	w := retreatWorld1089(t)
	w.tick = scriptPassPhase
	Step(w, retreat1089(1, 3, 1, 999))
	e, other := w.entities[0], w.entities[2]
	if e.ActorState != 0x16 || other.ActorState != 0x16 || e.CommandGroup == 0 || e.CommandGroup != other.CommandGroup {
		t.Fatalf("Retreat state/group = %d/%d %d/%d", e.ActorState, other.ActorState, e.CommandGroup, other.CommandGroup)
	}
	if e.X != 19 || e.TargetX != 17 || e.Withdraw != 0 || e.Wimpy != 0 {
		t.Fatalf("explicit retreat did not move healthy actor away: %+v", e)
	}
	for n := 0; n < 20; n++ {
		Step(w, nil)
	}
	if w.entities[0].ActorState != 0x16 {
		t.Fatal("arrival cleared persistent state")
	}
	// Relocate the hostile to the actor's left after the first retreat. The
	// next decision must choose a new rightward destination, not the old cell.
	x := w.entities[0].X
	w.entities[1].X, w.entities[1].Y = x-2, w.entities[0].Y
	w.armRetreat(0)
	if w.entities[0].TargetX != x+3 {
		t.Fatal("Retreat did not recompute", w.entities[0].TargetX)
	}
	back := retreatRoundTrip1089(t, w)
	for n := 0; n < 65; n++ {
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatalf("continuation at %d", n)
		}
	}
}

func TestPlayerRetreat1089FirstMemberScopeAndBoundedRefusals(t *testing.T) {
	for _, tc := range []struct {
		name     string
		commands []Command
		alter    func(*World)
	}{
		{"absent first", retreat1089(999, 1), nil},
		{"foreign first", retreat1089(2, 1), nil},
		{"dead first", retreat1089(1, 3), func(w *World) { w.entities[0].HP = -1; w.clearFelled(0) }},
		{"offmap first", retreat1089(1, 3), func(w *World) { w.entities[0].OffMap = true }},
		{"unknown player", []Command{{Kind: KindGroupRetreat, Entity: 1, Player: 99}}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := retreatWorld1089(t)
			if tc.alter != nil {
				tc.alter(w)
			}
			back := retreatRoundTrip1089(t, w)
			Step(w, tc.commands)
			Step(back, nil)
			if w.Hash() != back.Hash() {
				t.Fatal("refused command changed canonical state")
			}
		})
	}
	w := retreatWorld1089(t)
	Step(w, retreat1089(1, 999, 2, 3))
	if w.entities[0].ActorState != 0x16 || w.entities[2].ActorState != 0x16 || w.entities[1].ActorState == 0x16 {
		t.Fatal("later missing/foreign member mishandled")
	}
}

func TestPlayerRetreat1089LoadedAttackFinishesOnceBeforeFleeing(t *testing.T) {
	w := retreatWorld1089(t)
	w.entities[2].Owner = 0 // Only the commanded actor may damage this victim.
	w.entities[1].X = 21
	w.entities[1].PostX = 21
	w.entities[0].AttackCharge = 4
	w.orderAttack(0, 2)
	Step(w, nil)
	if w.entities[0].AttackPhase != AttackCharging {
		t.Fatal("fixture did not start attack")
	}
	w.tick = scriptPassPhase
	Step(w, retreat1089(1))
	if e := w.entities[0]; e.ActorState != 0x16 || e.AttackPhase != AttackCharging || !e.HasAttackTarget || e.X != 20 {
		t.Fatal("Retreat cancelled loaded action")
	}
	back := retreatRoundTrip1089(t, w)
	for n := 0; n < 40; n++ {
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatal("attack continuation", n)
		}
	}
	if w.entities[1].HP != 99 || w.entities[0].X >= 20 || w.entities[0].ActorState != 0x16 {
		t.Fatalf("expected one completed blow then flee: hp=%d x=%d state=%d", w.entities[1].HP, w.entities[0].X, w.entities[0].ActorState)
	}
}

func TestPlayerRetreat1089BookWindupAndLaterCastReplacement(t *testing.T) {
	retreat := []Command{{Kind: KindGroupRetreat, Entity: 1, Player: SelfSlot, Group: 91}}
	newWorld := func() *World {
		caster := spMage(1, 20, 20, 60, 50, 20, 1<<1)
		caster.Owner, caster.AttackCharge = SelfSlot, 8
		victim := withdrawalFighter(2, 3, 24, 20, 100)
		w, err := NewSpelledWorld(42, engBounds, ModeCanonical, nil, []Entity{caster, victim}, nil,
			[]SpellRule{{ID: 1, ManaCost: 5, MaxRange: 5, DamageMin: 4, DamageMax: 4, TargetsUnit: true, Damaging: true}})
		if err != nil {
			t.Fatal(err)
		}
		w.relations = engRel(t, [3]uint32{SelfSlot, 3, 1})
		return w
	}
	w := newWorld()
	Step(w, []Command{spCast(1, 2, 1)})
	if len(w.bookCasts) != 1 {
		t.Fatal("cast not admitted")
	}
	w.tick = scriptPassPhase
	Step(w, retreat)
	if len(w.bookCasts) != 1 || w.entities[0].ActorState != 0x16 || w.entities[0].HasTarget {
		t.Fatal("Retreat cancelled or bypassed cast windup")
	}
	back := retreatRoundTrip1089(t, w)
	for n := 0; n < 100; n++ {
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatal("cast continuation", n)
		}
	}
	if w.entities[0].Mana != 15 || w.entities[1].HP >= 100 || w.entities[0].X >= 20 {
		t.Fatal("cast did not release once before Retreat", w.entities[0].Mana, w.entities[1].HP, w.entities[0].X)
	}
	w = newWorld()
	Step(w, retreat)
	Step(w, []Command{spCast(1, 2, 1)})
	if len(w.bookCasts) != 1 || w.entities[0].ActorState == 0x16 {
		t.Fatal("admitted later cast did not replace Retreat")
	}
	retreatRoundTrip1089(t, w)
}

func TestPlayerRetreat1089TransitAndStoneHoldSurviveAdmission(t *testing.T) {
	for _, stone := range []bool{false, true} {
		w := retreatWorld1089(t)
		e := &w.entities[0]
		e.Speed, e.Transit, e.TransitTotal = 10, 3, 4
		if stone {
			w.attached = []attachedEffect{{Target: 1, Spell: 20, Kind: EffectAbsorption, Mode: EffectDuration, Remaining: 20}}
		}
		w.tick = scriptPassPhase
		Step(w, retreat1089(1))
		if e.ActorState != 0x16 || e.X != 20 || e.Transit == 0 || e.HasTarget {
			t.Fatal("progress hold lost", stone, *e)
		}
		back := retreatRoundTrip1089(t, w)
		for n := 0; n < 49; n++ {
			Step(w, nil)
			Step(back, nil)
			if w.Hash() != back.Hash() {
				t.Fatal("hold continuation", stone, n)
			}
		}
		if e.X >= 20 {
			t.Fatal("hold never yielded to Retreat", stone)
		}
	}
}

func TestPlayerRetreat1089LaterMoveAttackAndStanceReplaceState(t *testing.T) {
	for _, c := range []Command{
		{Kind: KindMoveTo, Entity: 1, X: 30, Y: 20},
		{Kind: KindGroupMoveTo, Entity: 1, X: 30, Y: 20},
		{Kind: KindAttack, Entity: 1, X: 2},
		{Kind: KindGroupStance, Entity: 1, X: OrderGuard},
		{Kind: KindGroupDefend, Entity: 1, X: 3},
	} {
		w := retreatWorld1089(t)
		Step(w, retreat1089(1))
		Step(w, []Command{c})
		if w.entities[0].ActorState == 0x16 {
			t.Fatalf("command %d retained Retreat", c.Kind)
		}
		retreatRoundTrip1089(t, w)
	}
}

func TestRetreat1089CorrectedWholeListCentroidAndDistinctDeadGates(t *testing.T) {
	w := retreatWorld1089(t)
	w.entities[1].X, w.entities[1].Y = 18, 18
	w.entities[2].Owner, w.entities[2].X, w.entities[2].Y, w.entities[2].HP = 3, 18, 24, -1
	w.withdrawFrom(0, []int{1, 2})
	if e := w.entities[0]; e.TargetX != 23 || e.TargetY != 19 {
		t.Fatal("corpse omitted from centroid", e.TargetX, e.TargetY)
	}
	w.clearOrder(0)
	w.entities[1].HP = -1
	w.withdrawFrom(0, []int{1, 2})
	if w.entities[0].HasTarget {
		t.Fatal("all-dead explicit list fled")
	}
	w.withdrawFromAny(0, []int{1, 2})
	if e := w.entities[0]; !e.HasTarget || e.TargetX != 23 || e.TargetY != 19 {
		t.Fatal("fixed-radius helper incorrectly used positive-HP gate")
	}
}

func TestPlayerRetreat1089SelectionCapAndDenseDivisorStayBounded(t *testing.T) {
	var ents []Entity
	var ids []EntityID
	for n := 0; n < 260; n++ {
		id := EntityID(n + 1)
		ents = append(ents, withdrawalFighter(id, 2, 20, 20, 100))
		ids = append(ids, id)
	}
	w := engWorld(t, engRel(t), ents...)
	Step(w, retreat1089(ids...))
	for i, e := range w.entities {
		if (e.ActorState == 0x16) != (i < 253) {
			t.Fatal("253 member cap", i, e.ActorState)
		}
	}
	retreatRoundTrip1089(t, w)
	var list []int
	for i := 1; i <= 256; i++ {
		w.entities[i].X = 22
		list = append(list, i)
	}
	w.withdrawFrom(0, list)
	if w.entities[0].TargetX != 17 {
		t.Fatal("dense custom list did not use bounded divisor")
	}
}

func TestPlayerRetreat1089ActorZeroAndDeathRemainCanonical(t *testing.T) {
	w := engWorld(t, engRel(t), withdrawalFighter(0, 2, 20, 20, 100))
	Step(w, retreat1089(0))
	if w.entities[0].ActorState != 0x16 {
		t.Fatal("actor zero was treated as absent")
	}
	Step(w, []Command{{Kind: KindKill, Entity: 0}})
	if w.entities[0].ActorState == 0x16 || w.entities[0].Alive() {
		t.Fatal("Retreat survived own death")
	}
	retreatRoundTrip1089(t, w)
}

func retreatScrollWorld1089(t *testing.T, targetX int32) *World {
	t.Helper()
	caster, victim := spEnt(1, 20, 20), spEnt(2, targetX, 20)
	caster.Owner, caster.ScanRange, caster.AttackCharge = 2, 12, 8
	victim.Owner = 3
	w, err := NewSpelledWorld(1089, engBounds, ModeCanonical, nil, []Entity{caster, victim}, nil,
		[]SpellRule{{ID: 1, MaxRange: 5, DamageMin: 10, DamageMax: 10, TargetsUnit: true, Damaging: true}})
	if err != nil {
		t.Fatal(err)
	}
	w.relations = engRel(t, [3]uint32{2, 3, 1})
	item := ItemInstance{Code: 0xe10, Kind: 4, Price: 73, Effects: []ItemEffect{{Kind: 41, Operand: 1 | 60<<16}}}
	w.carried[0] = []ItemStack{StackItem(item, 2)}
	w.recomputeLoad(0)
	return w
}

func TestPlayerRetreat1089ScrollProgressAndNativeNextAction(t *testing.T) {
	for _, started := range []bool{false, true} {
		name, targetX := "reserved approach", int32(32)
		if started {
			name, targetX = "started cast", 24
		}
		t.Run(name, func(t *testing.T) {
			w := retreatScrollWorld1089(t, targetX)
			item := w.carried[0][0].Instance()
			Step(w, []Command{{Kind: KindUseScroll, Entity: 1, X: 2}})
			Step(w, nil)
			if len(w.scrollCasts) != 1 || w.scrollCasts[0].Started != started || w.carried[0][0].Count != 1 {
				t.Fatalf("wrong scroll fixture: casts=%+v stock=%+v", w.scrollCasts, w.carried[0])
			}
			if !started && !w.entities[0].HasTarget {
				t.Fatal("unstarted scroll has no approach")
			}
			w.tick = scriptPassPhase
			beforeX := w.entities[0].X
			before := retreatRoundTrip1089(t, w)
			Step(w, retreat1089(1))
			Step(before, retreat1089(1))
			if w.Hash() != before.Hash() {
				t.Fatal("native next Retreat action differed")
			}
			if e := w.entities[0]; e.ActorState != actorStateRetreat || e.Withdraw != 0 || e.Wimpy != 0 {
				t.Fatal("Retreat state/thresholds", e)
			}
			if started {
				if len(w.scrollCasts) != 1 || !w.scrollCasts[0].Started || w.entities[0].X != beforeX {
					t.Fatal("Retreat interrupted or moved through a started scroll")
				}
			} else if len(w.scrollCasts) != 0 || w.carried[0][0].Count != 2 ||
				!ItemEqual(w.carried[0][0].Instance(), item) || w.carried[0][0].Price != item.Price {
				t.Fatalf("Retreat did not refund the unstarted scroll: casts=%+v stock=%+v", w.scrollCasts, w.carried[0])
			}
			back := retreatRoundTrip1089(t, w)
			releases := 0
			for n := 0; n < 100; n++ {
				releases += len(StepObserved(w, nil))
				Step(back, nil)
				if w.Hash() != back.Hash() {
					t.Fatal("scroll/Retreat continuation", n)
				}
			}
			wantCount, wantReleases := uint32(2), 0
			if started {
				wantCount, wantReleases = 1, 1
			}
			if len(w.scrollCasts) != 0 || releases != wantReleases || len(w.carried[0]) != 1 || w.carried[0][0].Count != wantCount {
				t.Fatalf("scroll economy/releases: casts=%+v stock=%+v releases=%d", w.scrollCasts, w.carried[0], releases)
			}
			if w.entities[0].X >= beforeX || w.entities[0].ActorState != actorStateRetreat {
				t.Fatal("cast/approach did not yield to Retreat", w.entities[0])
			}
			if !started && w.entities[1].HP != 100 || started && w.entities[1].HP >= 100 {
				t.Fatal("wrong scroll effect", w.entities[1].HP)
			}
			if w.entities[0].Mana != 0 || w.entities[0].KnownSpells != 0 {
				t.Fatal("scroll changed mana or learned spells")
			}
			next := []Command{{Kind: KindMoveTo, Entity: 1, X: 20, Y: 25}}
			Step(w, next)
			Step(back, next)
			if w.Hash() != back.Hash() || w.entities[0].ActorState == actorStateRetreat {
				t.Fatal("native next Move did not replace Retreat exactly")
			}
		})
	}
}

func TestPlayerRetreat1089RefusalKeepsScrollAndLaterScrollReplacesRetreat(t *testing.T) {
	w := retreatScrollWorld1089(t, 32)
	Step(w, []Command{{Kind: KindUseScroll, Entity: 1, X: 2}})
	back := retreatRoundTrip1089(t, w)
	Step(w, retreat1089(999, 1))
	Step(back, nil)
	if w.Hash() != back.Hash() || len(w.scrollCasts) != 1 {
		t.Fatal("invalid first member canceled the scroll")
	}
	w = retreatScrollWorld1089(t, 24)
	Step(w, retreat1089(1))
	back = retreatRoundTrip1089(t, w)
	next := []Command{{Kind: KindUseScroll, Entity: 1, X: 2}}
	Step(w, next)
	Step(back, next)
	if w.Hash() != back.Hash() || len(w.scrollCasts) != 1 || w.entities[0].ActorState == actorStateRetreat {
		t.Fatal("admitted scroll did not replace Retreat after native load")
	}
	retreatRoundTrip1089(t, w)
}

func retreatAutocastWorld1089(t *testing.T) *World {
	t.Helper()
	caster := spMage(1, 20, 20, 60, 300, 300, 1<<1)
	caster.Owner, caster.AutoSpell = SelfSlot, 1
	caster.AttackCharge, caster.AttackRelax, caster.ScanRange = 4, 4, 10
	victim := spEnt(2, 24, 20)
	victim.Owner, victim.HP, victim.MaxHP = 3, 1000, 1000
	w, err := NewSpelledWorld(891, engBounds, ModeCanonical, nil, []Entity{caster, victim}, nil,
		[]SpellRule{{ID: 1, ManaCost: 1, MaxRange: 10, DamageMin: 1, DamageMax: 1, TargetsUnit: true, Damaging: true}})
	if err != nil {
		t.Fatal(err)
	}
	w.relations.Set(SelfSlot, 3, 1)
	return w
}

func TestPlayerRetreat1089AutocastKeepsOnlyPreexistingProgress(t *testing.T) {
	for _, phase := range []string{"idle", "windup", "recovery"} {
		t.Run(phase, func(t *testing.T) {
			w := retreatAutocastWorld1089(t)
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
			Step(w, []Command{{Kind: KindGroupRetreat, Entity: 1, Player: SelfSlot, Group: 1}})
			if phase == "windup" && (len(w.bookCasts) != 1 || w.entities[0].X != 20) {
				t.Fatal("Retreat canceled a pre-existing autocast")
			}
			back := retreatRoundTrip1089(t, w)
			releases := 0
			for n := 0; n < 100; n++ {
				releases += len(StepObserved(w, nil))
				Step(back, nil)
				if w.Hash() != back.Hash() {
					t.Fatal("native autocast/Retreat continuation", n)
				}
			}
			wantReleases := 0
			if phase == "windup" {
				wantReleases = 1
			}
			e := w.entities[0]
			if e.X >= 20 || e.ActorState != actorStateRetreat || e.AutoSpell != 1 ||
				releases != wantReleases || e.Mana != beforeMana-int32(wantReleases) || w.actorCastBusy(0) {
				t.Fatalf("Retreat starved or changed prior work/setting: x=%d state=%d auto=%d releases=%d mana=%d", e.X, e.ActorState, e.AutoSpell, releases, e.Mana)
			}
		})
	}

	// Replacing Retreat re-enables the existing preference without another
	// toggle. This boundary is canonical state and must survive native load.
	w := retreatAutocastWorld1089(t)
	Step(w, []Command{{Kind: KindGroupRetreat, Entity: 1, Player: SelfSlot}})
	for n := 0; n < 6; n++ {
		Step(w, nil)
	}
	back := retreatRoundTrip1089(t, w)
	next := []Command{{Kind: KindGroupStance, Entity: 1, X: OrderStandGround}}
	Step(w, next)
	Step(back, next)
	if w.Hash() != back.Hash() || w.entities[0].AutoSpell != 1 || len(w.bookCasts) != 1 {
		t.Fatal("autocast setting did not resume after replacing Retreat")
	}
}

// A unit in explicit Retreat from two hostiles that stand off its row walks for
// many ticks with the flee cell recomputed from fine positions every full tick.
// The world is saved and decoded at every tenth tick and the decoded copy runs
// on beside the live one to the same hash (AI-RETREAT-273, AI-WITHDRAW-028).
func TestRetreatFromFinePositionsRunsManyTicksAndEveryTenthTickReloadsIdentically(t *testing.T) {
	self := withdrawalFighter(1, 2, 40, 40, 100)
	self.ScanRange = 40
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self,
		withdrawalFighter(2, 3, 43, 37, 100),
		withdrawalFighter(3, 3, 44, 41, 100))
	Step(w, []Command{GroupRetreat(1, 2, 91)})
	start := w.entities[0]
	for n := 0; n < 300; n++ {
		Step(w, nil)
		if n%10 == 9 {
			back := worldRoundTripForTest(t, w)
			if back.Hash() != w.Hash() {
				t.Fatalf("tick %d: the decoded world's hash differs from the live world's", w.tick)
			}
			w = back
		}
	}
	end := w.entities[0]
	if end.ActorState != actorStateRetreat {
		t.Fatalf("the unit left Retreat for state %d", end.ActorState)
	}
	if d := cellOf(&end).chebyshevTo(cellOf(&start)); d < 8 {
		t.Fatalf("the unit moved %d cells from (%d,%d) to (%d,%d) in 300 ticks, want a sustained retreat", d, start.X, start.Y, end.X, end.Y)
	}
}
