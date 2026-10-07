package sim

import "testing"

// creatureBookOf is a class book whose slot for id holds a spell of the given
// range, with the creature knowing exactly those ids.
func creatureBookOf(ranges map[uint32]uint8) (Spellbook, uint32) {
	book := Spellbook{State: BookPresent}
	var known uint32
	for id, r := range ranges {
		book.Slots[id-1] = BookSpell{Range: r, ManaCost: 3}
		known |= 1 << id
	}
	return book, known
}

func creatureCertainSlot(id uint32) [CreatureSpellSlots]CreatureSpell {
	return [CreatureSpellSlots]CreatureSpell{{ID: id, Threshold: creatureDrawMax + 1}}
}

func creatureAimRules() []SpellRule {
	return []SpellRule{
		hlArrow(),
		{ID: 2, ManaCost: 3, School: 1, MaxRange: 7, Area: true, Distribution: 3, Radius: 1},
		{ID: 9, ManaCost: 3, School: 1, MaxRange: 7, Area: true, Distribution: 4, Radius: 2, AreaDuration: 1},
		{ID: 15, ManaCost: 3, School: 1, MaxRange: 7, Defensive: true},
		{ID: 26, ManaCost: 3, School: 1, MaxRange: 7},
	}
}

// creatureFirstCast runs one decision of a creature that holds only the given
// slot and returns the cast it admitted.
func creatureFirstCast(t *testing.T, id uint32, victimX, victimY int32) (bookCast, *World) {
	t.Helper()
	book, known := creatureBookOf(map[uint32]uint8{id: 7})
	w := creatureWorld(t, creatureAimRules(), creatureCertainSlot(id), known, book)
	w.entities[indexOfEntity(w.entities, 2)].X, w.entities[indexOfEntity(w.entities, 2)].Y = victimX, victimY
	engRun(w, 1)
	k, ok := w.bookCastIndex(1)
	if !ok {
		t.Fatalf("spell %d: the creature began no cast", id)
	}
	return w.bookCasts[k], w
}

func TestACreatureAimsEachDrawnSpellAtItsArm(t *testing.T) {
	for _, tc := range []struct {
		name   string
		id     uint32
		target EntityID
		atCell bool
		x, y   int32
	}{
		{"victim arm", 1, 2, false, 0, 0},
		{"caster arm", 15, 1, false, 0, 0},
		{"victim cell arm", 2, 0, true, 9, 5},
		{"step cell arm of Acid Stream", 9, 0, true, 6, 5},
		{"step cell arm of Teleport", 26, 0, true, 6, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := creatureFirstCast(t, tc.id, 9, 5)
			if c.AtCell != tc.atCell || c.Target != tc.target || tc.atCell && (c.X != tc.x || c.Y != tc.y) {
				t.Errorf("spell %d cast %+v, want target %d at cell %v (%d,%d)", tc.id, c, tc.target, tc.atCell, tc.x, tc.y)
			}
			if !c.Retained {
				t.Errorf("spell %d cast is not a retained order", tc.id)
			}
		})
	}
}

func TestACreatureStepCellFollowsTheEightWayHeading(t *testing.T) {
	w := &World{bounds: Bounds{Width: 20, Height: 20}}
	caster := Entity{X: 10, Y: 10}
	for _, tc := range []struct {
		dx, dy int32
		x, y   int32
	}{
		{4, 0, 11, 10}, {4, 4, 11, 11}, {0, 4, 10, 11}, {-4, 4, 9, 11},
		{-4, 0, 9, 10}, {-4, -4, 9, 9}, {0, -4, 10, 9}, {4, -4, 11, 9},
		{4, 1, 11, 10}, {4, 3, 11, 11},
	} {
		x, y := w.creatureStepCell(caster, Entity{X: 10 + tc.dx, Y: 10 + tc.dy})
		if x != tc.x || y != tc.y {
			t.Errorf("victim at (%d,%d): step cell (%d,%d), want (%d,%d)", tc.dx, tc.dy, x, y, tc.x, tc.y)
		}
	}
	// The unclamped step leaves the map; the engine keeps it inside.
	edge := Entity{X: 0, Y: 0}
	if x, y := w.creatureStepCell(edge, edge); x != 0 || y != 0 {
		t.Errorf("a step from the corner left the map at (%d,%d)", x, y)
	}
}

func creatureShortBook() (Spellbook, uint32) { return creatureBookOf(map[uint32]uint8{1: 3}) }

func TestACreatureCastOutOfRangeWalksAndStaysArmedUntilInRange(t *testing.T) {
	short := hlArrow()
	short.MaxRange = 3
	book, known := creatureShortBook()
	w := creatureWorld(t, []SpellRule{short}, creatureCertain, known, book)
	engRun(w, 1)
	k, armed := w.bookCastIndex(1)
	if !armed || w.bookCasts[k].Phase != bookApproach || !w.bookCasts[k].Retained {
		t.Fatalf("the out-of-range cast is %+v, want an armed retained approach", w.bookCasts)
	}
	if victim, held := engVictim(w, 1); !held || victim != 2 {
		t.Fatal("the armed creature does not walk toward its victim")
	}
	const startX = 5
	cast := false
	for range 400 {
		Step(w, nil)
		if k, ok := w.bookCastIndex(1); ok && w.bookCasts[k].Phase == bookCharging {
			cast = true
			break
		}
	}
	if !cast {
		t.Fatal("the armed creature never cast")
	}
	if creatureEntity(w).X <= startX {
		t.Error("the creature cast without walking toward its victim")
	}
	if got := creatureEntity(w).chebyshevToEntity(w.entities[indexOfEntity(w.entities, 2)]); got > 3 {
		t.Errorf("the cast began %d cells from the victim, past the spell range 3", got)
	}
}

func TestAnApproachedCastRepeatsInPlaceAsARetainedOrder(t *testing.T) {
	short := hlArrow()
	short.MaxRange = 3
	book, known := creatureShortBook()
	w := creatureWorld(t, []SpellRule{short}, creatureCertain, known, book)
	engRun(w, 1)
	released := 0
	var at [2]int32
	for range 400 {
		Step(w, nil)
		k, ok := w.bookCastIndex(1)
		if ok && w.bookCasts[k].Phase == bookRelaxing {
			released++
			e := creatureEntity(w)
			at = [2]int32{e.X, e.Y}
		}
	}
	if released == 0 {
		t.Fatal("the cast never released into a retained recovery")
	}
	if e := creatureEntity(w); at != [2]int32{e.X, e.Y} {
		t.Errorf("the creature walked on after its cast: %v then (%d,%d)", at, e.X, e.Y)
	}
}

func (e Entity) chebyshevToEntity(o Entity) int64 {
	return (cell{x: e.X, y: e.Y}).chebyshevTo(cell{x: o.X, y: o.Y})
}

func TestAnArmedApproachSurvivesAReloadAndCastsLikeTheLiveWorld(t *testing.T) {
	short := hlArrow()
	short.MaxRange = 3
	book, known := creatureShortBook()
	w := creatureWorld(t, []SpellRule{short}, creatureCertain, known, book)
	engRun(w, 1)
	if k, ok := w.bookCastIndex(1); !ok || w.bookCasts[k].Phase != bookApproach {
		t.Fatal("no armed approach to reload")
	}
	back := worldRoundTripForTest(t, w)
	if back.Hash() != w.Hash() {
		t.Fatal("a reload changed the digest of an armed approach")
	}
	for tick := range 300 {
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatalf("the reloaded world diverged at tick %d", tick)
		}
	}
	if hp := back.entities[indexOfEntity(back.entities, 2)].HP; hp >= 5000 {
		t.Error("the armed approach never cast after the reload")
	}
}

func TestADrawRunsWhileTheCreatureCastIsPending(t *testing.T) {
	book, known := creatureBookOf(map[uint32]uint8{1: 7})
	w := creatureWorld(t, []SpellRule{hlArrow()}, creatureCertain, known, book)
	engRun(w, 1)
	k, ok := w.bookCastIndex(1)
	if !ok || w.bookCasts[k].Phase != bookCharging {
		t.Fatal("no cast winding up")
	}
	before := w.rng
	want := before
	want.uniform(creatureDrawMax)
	if w.orderAttack(w.indexOf(1), 2) {
		t.Error("an order was written over a winding cast")
	}
	if w.rng != want {
		t.Error("a creature with a pending cast did not draw exactly once for its one slot")
	}
	if k, ok := w.bookCastIndex(1); !ok || !w.bookCasts[k].Retained || w.bookCasts[k].Phase != bookCharging {
		t.Error("a hit during the wind-up changed the cast in flight")
	}
}

func (w *World) indexOf(id EntityID) int { return indexOfEntity(w.entities, id) }

func TestAMissWhilePendingEndsTheRetainedOrderAndKeepsTheRecovery(t *testing.T) {
	book, known := creatureBookOf(map[uint32]uint8{1: 7})
	w := creatureWorld(t, []SpellRule{hlArrow()}, creatureCertain, known, book)
	engRun(w, 1)
	var remaining uint8
	for range 400 {
		Step(w, nil)
		if k, ok := w.bookCastIndex(1); ok && w.bookCasts[k].Phase == bookRelaxing {
			remaining = w.bookCasts[k].Remaining
			break
		}
	}
	if remaining == 0 {
		t.Fatal("the cast never reached recovery")
	}
	w.entities[w.indexOf(1)].CreatureSpells = [CreatureSpellSlots]CreatureSpell{{ID: 1}}
	if w.orderAttack(w.indexOf(1), 2) {
		t.Error("an order was written during recovery")
	}
	if _, ok := w.bookCastIndex(1); ok {
		t.Error("a miss left the retained cast order armed")
	}
	if got := creatureEntity(w).CastWait; got != remaining {
		t.Errorf("the recovery left %d ticks, want %d", got, remaining)
	}
}

func TestAnAllZeroSlotBlockDrawsNothingAndTheDefaultEngageRuns(t *testing.T) {
	for name, slots := range map[string][CreatureSpellSlots]CreatureSpell{
		"all zero":             {},
		"threshold without id": {{Threshold: creatureDrawMax + 1}},
	} {
		t.Run(name, func(t *testing.T) {
			book, known := creatureBookOf(map[uint32]uint8{1: 7})
			w := creatureWorld(t, []SpellRule{hlArrow()}, slots, known, book)
			before := w.rng
			if !w.orderAttack(w.indexOf(1), 2) {
				t.Fatal("the default engage was not written")
			}
			if w.rng != before {
				t.Error("a block with no slot id drew a number")
			}
			if _, held := engVictim(w, 1); !held {
				t.Error("the default engage order was not written")
			}
		})
	}
}

func TestAZeroThresholdDrawsOnceAndNeverMatches(t *testing.T) {
	book, known := creatureBookOf(map[uint32]uint8{1: 7})
	w := creatureWorld(t, []SpellRule{hlArrow()}, [CreatureSpellSlots]CreatureSpell{{ID: 1}}, known, book)
	before := w.rng
	want := before
	want.uniform(creatureDrawMax)
	w.orderAttack(w.indexOf(1), 2)
	if w.rng != want {
		t.Error("a nonzero id with threshold zero did not draw exactly once")
	}
	if _, pending := w.bookCastIndex(1); pending {
		t.Error("a zero threshold matched a draw")
	}
}

func TestAMissDuringTheWindUpLetsTheCastCompleteOnceAndEndsTheRetainedOrder(t *testing.T) {
	book, known := creatureBookOf(map[uint32]uint8{1: 7})
	w := creatureWorld(t, []SpellRule{hlArrow()}, creatureCertain, known, book)
	engRun(w, 1)
	if k, ok := w.bookCastIndex(1); !ok || w.bookCasts[k].Phase != bookCharging || !w.bookCasts[k].Retained {
		t.Fatal("no retained cast winding up")
	}
	w.entities[w.indexOf(1)].CreatureSpells = [CreatureSpellSlots]CreatureSpell{{ID: 1}}
	if w.orderAttack(w.indexOf(1), 2) {
		t.Error("an order was written over a winding cast")
	}
	if k, ok := w.bookCastIndex(1); !ok || w.bookCasts[k].Retained || w.bookCasts[k].Phase != bookCharging {
		t.Fatalf("a miss during the wind-up left the cast %+v, want it winding up and not retained", w.bookCasts)
	}
	casts := 0
	for range 400 {
		for _, ev := range StepObserved(w, nil) {
			if ev.Caster == 1 {
				casts++
			}
		}
	}
	if casts != 1 {
		t.Errorf("the creature cast %d times after a miss during its wind-up, want 1", casts)
	}
}

// engageDraws counts the selector draws the creature made in the last decision
// pass.
func engageDraws(w *World, id EntityID) int {
	n := 0
	for _, d := range w.engageDrew {
		if d == id {
			n++
		}
	}
	return n
}

func TestACastingCreatureDrawsOncePerDecisionPassInAGroupFight(t *testing.T) {
	book, known := creatureBookOf(map[uint32]uint8{1: 7})
	w := creatureWorld(t, []SpellRule{hlArrow()}, creatureCertain, known, book)
	engRun(w, 1)
	if got := engageDraws(w, 1); got != 1 {
		t.Fatalf("the first pass drew %d times, want 1", got)
	}
	if k, ok := w.bookCastIndex(1); !ok || w.bookCasts[k].Phase != bookCharging {
		t.Fatal("no cast pending at the second pass")
	}
	w.commandGroup([]int{w.indexOf(1)}, orderGuard, cell{})
	w.engageCreatureFromNextToIt()
	for range scriptCycle {
		Step(w, nil)
	}
	if got := engageDraws(w, 1); got != 1 {
		t.Errorf("a creature with a pending cast drew %d times in one decision pass, want 1", got)
	}
}

// A member with a pending cast that the decision pass did not reach is taken
// by the join pass, which draws for it; one the decision pass already drew for
// is not asked again.
func TestAJoinPassReachesACastingCreatureTheDecisionPassDidNotDrawFor(t *testing.T) {
	book, known := creatureBookOf(map[uint32]uint8{1: 7})
	w := creatureWorld(t, []SpellRule{hlArrow()}, creatureCertain, known, book)
	engRun(w, 1)
	i := w.indexOf(1)
	if !w.actorCastBusy(i) {
		t.Fatal("no cast pending")
	}
	w.engageDrew = w.engageDrew[:0]
	if !w.joinsFight(i) {
		t.Error("a member with a pending cast that has not drawn this pass is not offered the join pass")
	}
	w.engageDrew = append(w.engageDrew, 1)
	if w.joinsFight(i) {
		t.Error("a member that already drew this pass is offered the join pass again")
	}
}

// engageCreatureFromNextToIt makes the creature's group an engaged one: its
// foe stands beside it holding it as its victim.
func (w *World) engageCreatureFromNextToIt() {
	vi := w.indexOf(2)
	w.entities[vi].X, w.entities[vi].Y = creatureEntity(w).X+1, creatureEntity(w).Y
	w.attachAttack(vi, 1, AttackTargetUnit, false)
}
