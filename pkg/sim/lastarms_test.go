package sim

import (
	"bytes"
	"testing"
)

// The last eight decoded script operations (0166): instants 7, 20, 25, 30 and
// 34, and group sub-commands 10, 11 and 15.
//
// The fixtures are engage_test.go's own — engWorld, engFighter, engRel — with
// two additions this story needs and that file has no use for: a flier, which is
// the only candidate the preference matrix vetoes, and a member of a named group.

// laFighter is engFighter in a group, at a NONZERO SPEED.
//
// The speed is what makes the formation gate's rate term visible: the term is
// the group's slowest member's speed narrowed to a byte, so a group of members
// all at speed 0 carries a term of 0 whether or not it is in formation, and the
// gate's two arms would be indistinguishable through that field.
func laFighter(id EntityID, owner, group uint32, x, y int32) Entity {
	e := engFighter(id, owner, x, y)
	e.Group = group
	e.Speed = 30
	return e
}

// laFlier is laFighter in the AIR domain, which is the whole of what makes it
// different: `preference`'s only zero cells are in its column, so a ground
// member scored against one is the arm's veto and nothing else here can be.
func laFlier(id EntityID, owner, group uint32, x, y int32) Entity {
	e := laFighter(id, owner, group, x, y)
	e.Domain = DomainAir
	return e
}

// laNode is one instant, run against w with nothing else happening.
func laNode(w *World, in ScriptInstant) { w.runInstant(in) }

// laEnt is the entity id, by value, or a fatal.
func laEnt(t *testing.T, w *World, id EntityID) Entity {
	t.Helper()
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		t.Fatalf("the world holds no entity %d", id)
	}
	return w.entities[i]
}

// laForm is w's byte form, or a fatal.
func laForm(t *testing.T, w *World) []byte {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	return b
}

// ------------------------------------------------------- AC-1, AC-2, AC-3

// TestInstantSevenWritesTheNamedPlayersFormationMode is AC-1: the mode is
// written raw into the named slot and no other slot moves.
func TestInstantSevenWritesTheNamedPlayersFormationMode(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 2, 5, 5))
	for p := uint32(0); p < relationSlots; p++ {
		if got := w.FormationMode(p); got != formationDefault {
			t.Fatalf("player %d opens at mode %d, want the default %d", p, got, formationDefault)
		}
	}
	// The campaign's own node, on map 110: player 2, mode 0.
	laNode(w, ScriptInstant{Op: ScriptInstantFormation, Player: 2, HasPlayer: true})
	if got := w.FormationMode(2); got != 0 {
		t.Errorf("player 2 is at mode %d after the node, want 0", got)
	}
	for p := uint32(0); p < relationSlots; p++ {
		if p == 2 {
			continue
		}
		if got := w.FormationMode(p); got != formationDefault {
			t.Errorf("player %d moved to mode %d, want the default %d", p, got, formationDefault)
		}
	}
}

func TestInstantSevenRefusesWhatItCannotName(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		in   ScriptInstant
	}{
		{"no player", ScriptInstant{Op: ScriptInstantFormation,
			Args: [scriptParams]int32{1}}},
		{"a slot past the roster", ScriptInstant{Op: ScriptInstantFormation,
			Player: relationSlots, HasPlayer: true, Args: [scriptParams]int32{1}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := engWorld(t, engRel(t), engFighter(1, 2, 5, 5))
			before := laForm(t, w)
			laNode(w, tc.in)
			if after := laForm(t, w); !bytes.Equal(before, after) {
				t.Error("the node changed the world, want bit-for-bit unchanged")
			}
		})
	}
}

// TestTheFormationModeGatesTheGroupMoveDistribution is AC-2 and AC-3: the
// mode's three behaviours, over one geometry the spread test REFUSES.
//
// The geometry is what makes the three arms discriminable. Two members standing
// more than the spread apart fail the spread test, so mode 2 gives no
// distribution and no rate term; mode 0 gives none either but without running
// the test; and any other nonzero mode gives both, over the very geometry the
// test refuses. So the mode-1 arm and the mode-2 arm differ HERE and nowhere
// else, and a build that read the mode and then ran the spread test anyway would
// fail the mode-1 case alone.
func TestTheFormationModeGatesTheGroupMoveDistribution(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		mode     uint8
		spread   bool
		wantTerm bool
	}{
		{"mode 0 never", 0, false, false},
		{"mode 2 tests the spread", formationDefault, false, false},
		{"mode 1 always", 1, true, true},
		{"mode 255 always", 255, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Two members far enough apart that inFormation refuses them.
			w := engWorld(t, engRel(t),
				laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 30, 30))
			w.formations[2] = tc.mode
			if inFormation(w.entities, []int{0, 1}, 17, 17) {
				t.Fatal("this fixture's geometry passes the spread test — it cannot tell the arms apart")
			}
			w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
				Args: [scriptParams]int32{int32(orderMove), 20, 20}})

			e := laEnt(t, w, 1)
			if !e.HasTarget {
				t.Fatalf("member 1 took no destination at all: %+v", e)
			}
			// UNDER THE FORMATION ARM the member's destination is the ordered
			// cell offset by its own displacement from the centroid; under
			// either other arm it is the ordered cell itself.
			atOrdered := e.TargetX == 20 && e.TargetY == 20
			if tc.spread == atOrdered {
				t.Errorf("member 1 walks to (%d,%d) under mode %d; in formation was %v",
					e.TargetX, e.TargetY, tc.mode, tc.spread)
			}
			if got := e.GroupSpeed != 0; got != tc.wantTerm {
				t.Errorf("member 1 carries a rate term %v under mode %d, want %v",
					got, tc.mode, tc.wantTerm)
			}
		})
	}
}

// TestInstantTwentyDropsTheWholeContainerAtTheUnitsFeet is AC-4: the container
// empties onto a sack at the unit's own cell, in the unit's own order.
func TestInstantTwentyDropsTheWholeContainerAtTheUnitsFeet(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 2, 5, 5))
	w.carried[0] = []ItemStack{{Code: 0x111, Count: 2}, {Code: 0x222, Count: 1}}
	laNode(w, ScriptInstant{Op: ScriptInstantDropAll, Unit: 1, HasUnit: true})

	if got, _ := w.CarriedStacks(1); len(got) != 0 {
		t.Errorf("the unit still holds %+v, want an empty container", got)
	}
	sacks := w.Sacks()
	if len(sacks) != 1 {
		t.Fatalf("the world holds %d sack(s), want one", len(sacks))
	}
	if sacks[0].X != 5 || sacks[0].Y != 5 {
		t.Errorf("the sack stands at (%d,%d), want the unit's own (5,5)", sacks[0].X, sacks[0].Y)
	}
	if want := []uint16{0x111, 0x111, 0x222}; !equalCodes(sacks[0].Items, want) {
		t.Errorf("the sack holds %v, want %v in the unit's own order", sacks[0].Items, want)
	}
	if sacks[0].Gold != 0 {
		t.Errorf("the sack holds %d gold, want 0 — this arm forwards a literal zero", sacks[0].Gold)
	}
}

// TestInstantTwentyMergesIntoASackAlreadyThere is AC-5: one sack, holding what
// it held followed by what the unit held.
func TestInstantTwentyMergesIntoASackAlreadyThere(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 2, 5, 5))
	w.pourSack(5, 5, 9, plainItems([]uint16{0x333}))
	w.carried[0] = []ItemStack{{Code: 0x111, Count: 1}}
	laNode(w, ScriptInstant{Op: ScriptInstantDropAll, Unit: 1, HasUnit: true})

	sacks := w.Sacks()
	if len(sacks) != 1 {
		t.Fatalf("the world holds %d sack(s), want one merged", len(sacks))
	}
	if want := []uint16{0x333, 0x111}; !equalCodes(sacks[0].Items, want) {
		t.Errorf("the merged sack holds %v, want %v", sacks[0].Items, want)
	}
	if sacks[0].Gold != 9 {
		t.Errorf("the merged sack holds %d gold, want the 9 it already had", sacks[0].Gold)
	}
}

func TestInstantTwentyRefusesWhatItCannotDrop(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		at   [2]int32
		hold bool
		in   ScriptInstant
	}{
		{"no reference", [2]int32{5, 5}, true, ScriptInstant{Op: ScriptInstantDropAll}},
		{"a reference this world does not hold", [2]int32{5, 5}, true,
			ScriptInstant{Op: ScriptInstantDropAll, Unit: 77, HasUnit: true}},
		{"a unit holding nothing", [2]int32{5, 5}, false,
			ScriptInstant{Op: ScriptInstantDropAll, Unit: 1, HasUnit: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := engWorld(t, engRel(t), engFighter(1, 2, tc.at[0], tc.at[1]))
			if tc.hold {
				w.carried[0] = []ItemStack{{Code: 0x111, Count: 1}}
			}
			before := laForm(t, w)
			laNode(w, tc.in)
			if after := laForm(t, w); !bytes.Equal(before, after) {
				t.Error("the node changed the world, want bit-for-bit unchanged")
			}
		})
	}
}

// ------------------------------------------------------------- AC-6, AC-7

// TestInstantTwentyFiveWritesTheCellTail is AC-6, over the campaign's own first
// shipped node: spell 3, power 20, at (59, 54).
func TestInstantTwentyFiveWritesTheCellTail(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 2, 5, 5))
	laNode(w, ScriptInstant{Op: ScriptInstantCellTail,
		Args: [scriptParams]int32{3, 20, 59, 54}})

	tails := w.CellTails()
	if len(tails) != 1 {
		t.Fatalf("the world holds %d cell tail(s), want one", len(tails))
	}
	if tails[0].X != 59 || tails[0].Y != 54 {
		t.Errorf("the tail stands at (%d,%d), want (59,54)", tails[0].X, tails[0].Y)
	}
	// THE AUTHORED X IS NOT IN THE TAIL: the arm packs it into the key and
	// passes y twice.
	if want := [cellTailLen]byte{3, 20, 0, 54, 0, 54}; tails[0].Bytes != want {
		t.Errorf("the tail holds % x, want % x", tails[0].Bytes, want)
	}
}

// TestTwoNodesOnOneCellLeaveOneTail is AC-7: the record is updated in place, so
// the second node's bytes are what stands there.
func TestTwoNodesOnOneCellLeaveOneTail(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 2, 5, 5))
	laNode(w, ScriptInstant{Op: ScriptInstantCellTail, Args: [scriptParams]int32{3, 20, 59, 54}})
	laNode(w, ScriptInstant{Op: ScriptInstantCellTail, Args: [scriptParams]int32{9, 60, 59, 54}})

	tails := w.CellTails()
	if len(tails) != 1 {
		t.Fatalf("the world holds %d cell tail(s), want one", len(tails))
	}
	if want := [cellTailLen]byte{9, 60, 0, 54, 0, 54}; tails[0].Bytes != want {
		t.Errorf("the tail holds % x, want the second node's % x", tails[0].Bytes, want)
	}
}

// TestTheTailKeyIsNotTheAreaEffectKey is D-6: instant 25 ORs byte truncations
// and instant 29 ADDs word truncations, so an x above 255 carries in one and is
// masked in the other. Written as a comparison of the two combiners rather than
// of two worlds, because no map this build can load authors such an x.
func TestTheTailKeyIsNotTheAreaEffectKey(t *testing.T) {
	t.Parallel()

	wide := int32(300)
	if got, want := tailKey(wide, 4), uint16(4)<<8|uint16(uint8(wide)); got != want {
		t.Errorf("tailKey(300,4) is %#04x, want %#04x — the x is masked to a byte", got, want)
	}
	if tailKey(wide, 4) == cellKey(wide, 4) {
		t.Error("the two cell keys agree at x=300, and the whole reason they are two functions " +
			"is that instant 29's ADD carries into the y byte where instant 25's OR cannot")
	}
}

// TestInstantThirtyRetimesTheAttachedEffect is AC-8, over the campaign's own
// spell and duration. The canonical attached record moves; its presentation
// mark does not select or own the write.
func TestInstantThirtyRetimesTheAttachedEffect(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		carries uint16
		spell   int32
		want    uint16
	}{
		{"the ids match", 20, 20, 60000},
		{"the ids differ", 19, 20, spellFXLife},
		{"the node names zero and the unit carries another effect", 20, 0, spellFXLife},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := engWorld(t, engRel(t), engFighter(1, 2, 5, 5), engFighter(2, 2, 6, 5))
			w.attached = []attachedEffect{instant30Effect(1, tc.carries, spellFXLife)}
			w.markSpellEffect(0, tc.carries)
			laNode(w, ScriptInstant{Op: ScriptInstantUnitEffectAge, Unit: 1, HasUnit: true,
				Args: [scriptParams]int32{tc.spell, 60000}})
			if got := w.ActiveEffects()[0].Remaining; got != tc.want {
				t.Errorf("the canonical effect has %d tick(s) left, want %d", got, tc.want)
			}
			if got := laEnt(t, w, 1).SpellFX; got != spellFXLife {
				t.Errorf("the presentation mark moved to %d, want unchanged %d", got, spellFXLife)
			}
		})
	}
}

// TestInstantThirtyDoesNotMatchAnEntityCarryingNothing is D-8: the mark and its
// id fall to zero together, so a node authoring id 0 must not match an entity
// with no mark at all.
func TestInstantThirtyDoesNotMatchAnEntityCarryingNothing(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 2, 5, 5))
	before := laForm(t, w)
	laNode(w, ScriptInstant{Op: ScriptInstantUnitEffectAge, Unit: 1, HasUnit: true,
		Args: [scriptParams]int32{0, 60000}})
	if after := laForm(t, w); !bytes.Equal(before, after) {
		t.Error("a node naming effect 0 marked an entity carrying no effect at all")
	}
}

func TestInstantThirtyWritesAZeroDuration(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 2, 5, 5), engFighter(2, 2, 6, 5))
	w.attached = []attachedEffect{instant30Effect(1, 7, 90)}
	laNode(w, ScriptInstant{Op: ScriptInstantUnitEffectAge, Unit: 1, HasUnit: true,
		Args: [scriptParams]int32{7, 0}})
	if got := w.ActiveEffects()[0].Remaining; got != 0 {
		t.Fatalf("the canonical effect has %d tick(s) left, want the authored 0", got)
	}
	Step(w, nil)
	if got := w.ActiveEffects(); len(got) != 0 {
		t.Errorf("after a tick the canonical effect remains: %+v", got)
	}
	if got := laEnt(t, w, 1); got.SpellFX != 0 || got.SpellFXSpell != 0 {
		t.Errorf("the canonical write created a presentation mark %d/%d", got.SpellFX, got.SpellFXSpell)
	}
}

// ------------------------------------------------------------- AC-9, AC-10

// TestInstantThirtyFourWritesTheSelectedProperty is AC-9: three selectors store
// and a fourth value stores nothing at all.
func TestInstantThirtyFourWritesTheSelectedProperty(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		selector int32
		value    int32
		read     func(Entity) int32
	}{
		{"selector 6 is current health", propertyHealth, 7, func(e Entity) int32 { return e.HP }},
		{"selector 15 is defence", propertyDefence, 33, func(e Entity) int32 { return e.Defence }},
		{"selector 16 is absorption", propertyAbsorption, 200, func(e Entity) int32 { return e.Absorption }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := engWorld(t, engRel(t), engFighter(1, 2, 5, 5))
			laNode(w, ScriptInstant{Op: ScriptInstantProperty, Unit: 1, HasUnit: true,
				Args: [scriptParams]int32{tc.selector, tc.value}})
			if got := tc.read(laEnt(t, w, 1)); got != tc.value {
				t.Errorf("the property reads %d, want the authored %d", got, tc.value)
			}
		})
	}

	// AN UNSUPPORTED SELECTOR STORES NOTHING, over the whole byte form.
	w := engWorld(t, engRel(t), engFighter(1, 2, 5, 5))
	before := laForm(t, w)
	laNode(w, ScriptInstant{Op: ScriptInstantProperty, Unit: 1, HasUnit: true,
		Args: [scriptParams]int32{7, 99}})
	if after := laForm(t, w); !bytes.Equal(before, after) {
		t.Error("selector 7 changed the world, want bit-for-bit unchanged")
	}
}

func TestInstantThirtyFourWritesAHealthAboveTheMaximum(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 2, 5, 5))
	laNode(w, ScriptInstant{Op: ScriptInstantProperty, Unit: 1, HasUnit: true,
		Args: [scriptParams]int32{propertyHealth, 5000}})
	e := laEnt(t, w, 1)
	if e.HP != 5000 {
		t.Errorf("health is %d, want the authored 5000 unclamped", e.HP)
	}
	if e.MaxHP != 20 {
		t.Errorf("maximum health moved to %d, want the 20 it was — this arm updates no maximum", e.MaxHP)
	}
}

// TestInstantThirtyFourFellsAUnitItWritesToZero is AC-10 and D-9: the world
// stays one this package can marshal and read back.
func TestInstantThirtyFourFellsAUnitItWritesToZero(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 2, 5, 5))
	laNode(w, ScriptInstant{Op: ScriptInstantProperty, Unit: 1, HasUnit: true,
		Args: [scriptParams]int32{propertyHealth, 0}})
	e := laEnt(t, w, 1)
	if e.Alive() {
		t.Fatalf("the unit is still alive at %d/%d", e.HP, e.MaxHP)
	}
	if e.Decay == DecayNone {
		t.Error("a unit written to zero health carries no decay stage — the byte form refuses that pair")
	}
	form := laForm(t, w)
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary of a world this arm produced: %v", err)
	}
}

func TestInstantThirtyFourRevivesAnAuthoredBodyAndSignExtendsItsWord(t *testing.T) {
	body := engFighter(1, 2, 5, 5)
	body.HP, body.Defence = 0, 40
	PrepareAuthoredBody(&body)
	w := engWorld(t, engRel(t), body)
	if got := laEnt(t, w, 1); got.Decay != DecayFallen || got.Defence != 20 {
		t.Fatalf("setup body = decay %d defence %d", got.Decay, got.Defence)
	}

	laNode(w, ScriptInstant{Op: ScriptInstantProperty, Unit: 1, HasUnit: true,
		Args: [scriptParams]int32{propertyHealth, 1}})
	got := laEnt(t, w, 1)
	if got.HP != 1 || got.Decay != DecayNone || got.Dwell != 0 || got.Defence != 40 {
		t.Fatalf("script HP=1 produced HP %d decay %d dwell %d defence %d",
			got.HP, got.Decay, got.Dwell, got.Defence)
	}

	laNode(w, ScriptInstant{Op: ScriptInstantProperty, Unit: 1, HasUnit: true,
		Args: [scriptParams]int32{propertyHealth, 0xffff}})
	if got := laEnt(t, w, 1); got.HP != -1 || got.Decay != DecayFallen {
		t.Fatalf("script word 0xffff produced HP %d decay %d, want signed -1/fallen", got.HP, got.Decay)
	}
}

// TestSubCommandTenEngagesEveryOtherMember is AC-11: the named unit acquires in
// place, and the rest hold an attack order on it.
func TestSubCommandTenEngagesEveryOtherMember(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t),
		laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 6, 5), laFighter(3, 2, 7, 7, 5))
	// THE NAMED UNIT IS WALKING, and that is the one member whose stop is
	// witnessed here: an engaging member's destination is cleared by
	// orderAttack whether or not the stop ran, and the named unit takes no
	// attack order at all, so its cleared destination is the stop's own.
	w.entities[0].TargetX, w.entities[0].TargetY, w.entities[0].HasTarget = 40, 40, true
	w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
		Unit: 1, HasUnit: true, Args: [scriptParams]int32{subCommandAttack}})

	named := laEnt(t, w, 1)
	if named.ActorState != actorStateAcquire {
		t.Errorf("the named unit is in state %d, want acquire (%d)", named.ActorState, actorStateAcquire)
	}
	if named.HasAttackTarget {
		t.Errorf("the named unit holds an attack order on %d, want none", named.AttackTarget)
	}
	if named.PostX != 5 || named.PostY != 5 {
		t.Errorf("the named unit's post is (%d,%d), want its own cell (5,5)", named.PostX, named.PostY)
	}
	if named.HasTarget {
		t.Errorf("the named unit still walks to (%d,%d) — the stop did not run",
			named.TargetX, named.TargetY)
	}
	for _, id := range []EntityID{2, 3} {
		e := laEnt(t, w, id)
		if !e.HasAttackTarget || e.AttackTarget != 1 {
			t.Errorf("member %d holds victim %d/%v, want the named unit 1", id, e.AttackTarget, e.HasAttackTarget)
		}
		if e.HasTarget {
			t.Errorf("member %d still holds a destination (%d,%d)", id, e.TargetX, e.TargetY)
		}
	}
	if got, _, _ := w.groupState(2, 7); got != orderNone {
		t.Errorf("the group's stored order is %d, want none (%d)", got, orderNone)
	}
}

// TestSubCommandTenHoldsAVetoedMemberInPlace is AC-12.
//
// The veto is `AI-PREF-070`'s matrix cell 0, which needs a candidate of movement
// domain 3 — a flier. It is AUTHORED-UNREACHABLE in the shipped campaign,
// established positively: all five shipped references are Humans and therefore
// domain 1, and the matrix holds no zero in that column. This fixture reaches it
// on purpose, which is the whole reason the gate is built.
func TestSubCommandTenHoldsAVetoedMemberInPlace(t *testing.T) {
	t.Parallel()

	// The named unit is the flier and it is in the group, so the two members
	// scored against it are both ground and both vetoed.
	w := engWorld(t, engRel(t),
		laFlier(1, 2, 7, 5, 5), laFighter(2, 2, 7, 6, 5))
	if !w.targetVetoed(1, 0) {
		t.Fatal("this fixture's ground member is not vetoed against its flier — it cannot witness the gate")
	}
	w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
		Unit: 1, HasUnit: true, Args: [scriptParams]int32{subCommandAttack}})

	e := laEnt(t, w, 2)
	if e.HasAttackTarget {
		t.Errorf("the vetoed member holds an attack order on %d, want none at all", e.AttackTarget)
	}
	if e.ActorState != actorStateAcquire {
		t.Errorf("the vetoed member is in state %d, want acquire (%d)", e.ActorState, actorStateAcquire)
	}

	// AND THE CONTROL: the same fixture with a GROUND named unit engages.
	c := engWorld(t, engRel(t),
		laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 6, 5))
	c.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
		Unit: 1, HasUnit: true, Args: [scriptParams]int32{subCommandAttack}})
	if got := laEnt(t, c, 2); !got.HasAttackTarget || got.AttackTarget != 1 {
		t.Errorf("the control member holds victim %d/%v, want the named unit 1",
			got.AttackTarget, got.HasAttackTarget)
	}
}

// TestSubCommandsElevenAndFifteenAreOneShapeAndTwoStates is AC-13 and AC-14.
func TestSubCommandsElevenAndFifteenAreOneShapeAndTwoStates(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		sub   int32
		state uint8
	}{
		{"defend", subCommandDefend, actorStateDefend},
		{"follow", subCommandFollow, actorStateFollow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := engWorld(t, engRel(t),
				laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 6, 5), laFighter(3, 2, 7, 7, 5))
			w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
				Unit: 1, HasUnit: true, Args: [scriptParams]int32{tc.sub, 4}})

			if got := laEnt(t, w, 1); got.ActorState != actorStateAcquire || got.HasEscortTarget {
				t.Errorf("the named unit is in state %d escorting %v, want acquire (%d) and nothing",
					got.ActorState, got.HasEscortTarget, actorStateAcquire)
			}
			for _, id := range []EntityID{2, 3} {
				e := laEnt(t, w, id)
				if e.ActorState != tc.state {
					t.Errorf("member %d is in state %d, want %d", id, e.ActorState, tc.state)
				}
				if !e.HasEscortTarget || e.EscortTarget != 1 {
					t.Errorf("member %d escorts %d/%v, want the named unit 1",
						id, e.EscortTarget, e.HasEscortTarget)
				}
				if e.EscortRange != 4 {
					t.Errorf("member %d escorts at range %d, want the authored 4", id, e.EscortRange)
				}
			}
		})
	}
}

func TestAnEscortRangeOfZeroBecomesThree(t *testing.T) {
	t.Parallel()

	for _, authored := range []int32{0, 256} {
		w := engWorld(t, engRel(t), laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 6, 5))
		w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
			Unit: 1, HasUnit: true, Args: [scriptParams]int32{subCommandDefend, authored}})
		if got := laEnt(t, w, 2).EscortRange; got != escortRangeDefault {
			t.Errorf("an authored range of %d became %d, want %d", authored, got, escortRangeDefault)
		}
	}
}

func TestTheLastThreeSubCommandsRefuseWhatTheyCannotName(t *testing.T) {
	t.Parallel()

	for _, sub := range []int32{subCommandAttack, subCommandDefend, subCommandFollow} {
		for _, tc := range []struct {
			name string
			in   ScriptInstant
		}{
			{"no unit", ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
				Args: [scriptParams]int32{sub, 3}}},
			{"a unit this world does not hold", ScriptInstant{Op: ScriptInstantGroupOrder,
				Group: 7, HasGroup: true, Unit: 77, HasUnit: true,
				Args: [scriptParams]int32{sub, 3}}},
			{"no group", ScriptInstant{Op: ScriptInstantGroupOrder, Unit: 1, HasUnit: true,
				Args: [scriptParams]int32{sub, 3}}},
		} {
			t.Run("", func(t *testing.T) {
				t.Parallel()
				w := engWorld(t, engRel(t),
					laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 6, 5))
				w.entities[1].TargetX, w.entities[1].TargetY, w.entities[1].HasTarget = 40, 40, true
				before := laForm(t, w)
				w.runInstant(tc.in)
				if after := laForm(t, w); !bytes.Equal(before, after) {
					t.Errorf("sub-command %d with %s changed the world, want bit-for-bit unchanged",
						sub, tc.name)
				}
			})
		}
	}
}

// ---------------------------------------------------------------- AC-16

// TestEveryKindOfStateThisStoryAddsIsCanonical is AC-16: it is in the form, it
// is in the digest, and it comes back whole.
func TestEveryKindOfStateThisStoryAddsIsCanonical(t *testing.T) {
	t.Parallel()

	build := func(t *testing.T, mode uint8, tailSpell int32, mark uint16, span uint8) *World {
		t.Helper()
		w := engWorld(t, engRel(t), laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 6, 5))
		w.setFormationMode(2, int32(mode))
		w.setCellTail(tailSpell, 20, 59, 54)
		w.markSpellEffect(0, 20)
		w.entities[0].SpellFX = mark
		w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
			Unit: 1, HasUnit: true, Args: [scriptParams]int32{subCommandFollow, int32(span)}})
		return w
	}

	w := build(t, 1, 3, 60000, 5)
	form := laForm(t, w)
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got, err := back.MarshalBinary(); err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	} else if !bytes.Equal(got, form) {
		t.Error("a world carrying this story's state does not marshal back to the bytes it came from")
	}
	if back.Hash() != w.Hash() {
		t.Errorf("the decoded world hashes %#016x, the original %#016x", back.Hash(), w.Hash())
	}
	if got := back.FormationMode(2); got != 1 {
		t.Errorf("the decoded formation mode is %d, want 1", got)
	}
	if got := back.CellTails(); len(got) != 1 || got[0].Bytes[0] != 3 {
		t.Errorf("the decoded cell tails are %+v, want one carrying spell 3", got)
	}
	if got := laEnt(t, &back, 1).SpellFX; got != 60000 {
		t.Errorf("the decoded mark has %d tick(s) left, want 60000", got)
	}
	if got := laEnt(t, &back, 2); got.ActorState != actorStateFollow || got.EscortRange != 5 {
		t.Errorf("the decoded escort is state %d range %d, want %d and 5",
			got.ActorState, got.EscortRange, actorStateFollow)
	}

	// AND EACH VALUE MOVES THE DIGEST, one at a time.
	for _, tc := range []struct {
		name string
		w    *World
	}{
		{"a different formation mode", build(t, 2, 3, 60000, 5)},
		{"a different cell tail", build(t, 1, 9, 60000, 5)},
		{"a different mark", build(t, 1, 3, 59999, 5)},
		{"a different escort range", build(t, 1, 3, 60000, 6)},
	} {
		if tc.w.Hash() == w.Hash() {
			t.Errorf("%s hashes the same as the fixture", tc.name)
		}
	}
}

// ---------------------------------------------------------------- AC-17

// TestTheLastEightOperationsAreNoLongerUnsupported is AC-17's first half: the
// compile-time report names none of them.
func TestTheLastEightOperationsAreNoLongerUnsupported(t *testing.T) {
	t.Parallel()

	instants := []ScriptInstant{
		{Op: ScriptInstantFormation},
		{Op: ScriptInstantDropAll},
		{Op: ScriptInstantCellTail},
		{Op: ScriptInstantUnitEffectAge},
		{Op: ScriptInstantProperty},
		{Op: ScriptInstantGroupOrder, Args: [scriptParams]int32{subCommandAttack}},
		{Op: ScriptInstantGroupOrder, Args: [scriptParams]int32{subCommandDefend}},
		{Op: ScriptInstantGroupOrder, Args: [scriptParams]int32{subCommandFollow}},
	}
	s := mustScript(t, []ScriptCheck{constCheck(0, 1)}, instants, nil)
	if gaps := s.Unsupported(); len(gaps) != 0 {
		t.Errorf("the report names %+v, want none of the eight", gaps)
	}
	// AND THE TWO TABLES AGREE WITH THE DISPATCH: every one of the eight is
	// reported supported, and an arm this build does not run is still reported.
	// That control was instant 2 until 0169 implemented it; it is opcode 11,
	// the item transfer, now — an arm no shipped map authors.
	for _, in := range instants {
		if !scriptInstantSupported(in) {
			t.Errorf("op %d sub %d is dispatched but reported unsupported", in.Op, in.Args[0])
		}
	}
	if scriptInstantSupported(ScriptInstant{Op: 11}) {
		t.Error("instant 11 is reported supported, and no arm of this build runs it")
	}
}
