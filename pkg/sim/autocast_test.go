package sim

import (
	"bytes"
	"testing"
)

func TestIdleAutocastDoesNotAllocateSight(t *testing.T) {
	caster := acCaster(1, 2, 2, 50, 1<<6, 6)
	caster.HP = caster.MaxHP
	w := hlWorld(t, 29, Relations{}, []SpellRule{hlHeal()}, caster, spEnt(2, 3, 3))
	before := hlBytes(t, w)
	allocs := testing.AllocsPerRun(20, func() {
		if id, ok := w.autoCastTarget(0, hlHeal()); ok {
			t.Fatalf("full-health party selected %d", id)
		}
	})
	if allocs != 0 || !bytes.Equal(before, hlBytes(t, w)) {
		t.Fatalf("idle selection allocates %g times or mutates the world", allocs)
	}
	w.entities[1].HP = 20
	if id, ok := w.autoCastTarget(0, hlHeal()); !ok || id != 2 {
		t.Fatalf("newly wounded target = %d/%v, want 2/true", id, ok)
	}
}

// The autocast: what an entity does unbidden, what it aims at, and what it
// does not do. The fixtures are heal_test.go's own.

// acEnemies is the relation both damaging-autocast fixtures use: owner 1 and
// owner 2 at war, both ways.
func acEnemies(t *testing.T) Relations {
	t.Helper()
	return engRel(t, [3]uint32{1, 2, relationHostile}, [3]uint32{2, 1, relationHostile})
}

// acCaster is a mage carrying an autocast spell and nothing else new: the four
// fields a cast's economy reads, plus the id it casts unbidden.
func acCaster(id EntityID, x, y int32, mana int32, known uint32, auto uint16) Entity {
	e := spMage(id, x, y, 30, 50, mana, known)
	e.AutoSpell = auto
	return e
}

// ---------------------------------------------------------------- AC-7

// TestADamagingAutocastFiresWithNoCommandAndThenWaits is AC-7's first half: no
// command is sent on any tick of this test, and the victim still loses health,
// carries the mark and is left alone until the wait has run out.
func TestADamagingAutocastFiresWithNoCommandAndThenWaits(t *testing.T) {
	t.Parallel()

	caster := acCaster(1, 2, 2, 50, 1<<1, 1)
	caster.AttackCharge, caster.AttackRelax = 12, 4
	caster.Owner = 1
	enemy := spEnt(2, 4, 4)
	enemy.Owner = 2
	w := hlWorld(t, 23, acEnemies(t), []SpellRule{hlArrow()}, caster, enemy)

	var releaseTicks []uint64
	for range 40 {
		if events := StepObserved(w, nil); len(events) > 0 {
			releaseTicks = append(releaseTicks, w.Tick())
		}
	}
	if len(releaseTicks) != 2 {
		t.Fatalf("forty held-autocast ticks released at %v, want two casts rather than a projectile each tick", releaseTicks)
	}
	if releaseTicks[1]-releaseTicks[0] < uint64(caster.AttackCharge+caster.AttackRelax) {
		t.Errorf("release spacing is %d ticks, below charge+recovery %d", releaseTicks[1]-releaseTicks[0], caster.AttackCharge+caster.AttackRelax)
	}
	if got := spAt(t, w, 1).Mana; got != 44 {
		t.Errorf("the caster holds %d mana, want two costs and no machine-gun extras", got)
	}
}

// TestADamagingAutocastWithItsEnemyOutOfRangeSpendsNothing is AC-7's second
// half, against a control world with no autocast at all: out of reach, the
// sweep must be indistinguishable from not running.
func TestADamagingAutocastWithItsEnemyOutOfRangeSpendsNothing(t *testing.T) {
	t.Parallel()

	build := func(auto uint16) *World {
		caster := acCaster(1, 1, 1, 50, 1<<1, auto)
		caster.Owner = 1
		// Chebyshev 12, well past the row's own reach of 7.
		enemy := spEnt(2, 13, 13)
		enemy.Owner = 2
		return hlWorld(t, 23, acEnemies(t), []SpellRule{hlArrow()}, caster, enemy)
	}

	armed, control := build(1), build(0)
	Step(armed, nil)
	Step(control, nil)

	if got := spAt(t, armed, 1).Mana; got != 50 {
		t.Errorf("the caster holds %d mana with nothing in range, want its own untouched 50", got)
	}
	if got := spAt(t, armed, 2).HP; got != 100 {
		t.Errorf("the enemy holds %d health with the caster out of reach, want 100", got)
	}
	// The two worlds differ in the autocast id alone, so stripping that one
	// field back off the armed world must reproduce the control exactly —
	// which is what says the refused attempt drew nothing and marked nothing.
	i := indexOfEntity(armed.entities, 1)
	armed.entities[i].AutoSpell = 0
	if !bytes.Equal(hlBytes(t, armed), hlBytes(t, control)) {
		t.Error("an autocast that found nothing in range left a world the control does not")
	}
}

// ---------------------------------------------------------------- AC-8

// TestARestorativeAutocastUsesStableRangeOrdering is the owner's in-tier
// distance then deficit and id order.
func TestARestorativeAutocastUsesStableRangeOrdering(t *testing.T) {
	t.Parallel()

	t.Run("the nearest eligible ally", func(t *testing.T) {
		caster := acCaster(1, 2, 2, 50, 1<<6, 6)
		lightly := spEnt(2, 3, 3)
		lightly.HP = 90
		badly := spEnt(3, 4, 4)
		badly.HP = 20
		w := hlWorld(t, 29, Relations{}, []SpellRule{hlHeal()}, caster, lightly, badly)

		spRunUnbidden(w)

		if got := spAt(t, w, 2).HP; got <= 90 {
			t.Errorf("the nearer ally holds %d health, want a heal to have landed", got)
		}
		if got := spAt(t, w, 3).HP; got != 20 {
			t.Errorf("the farther ally holds %d health, want its own untouched 20", got)
		}
	})

	t.Run("the lower id when two are equally hurt", func(t *testing.T) {
		caster := acCaster(1, 2, 2, 50, 1<<6, 6)
		first := spEnt(2, 3, 3)
		first.HP = 50
		second := spEnt(3, 4, 4)
		second.HP = 50
		w := hlWorld(t, 29, Relations{}, []SpellRule{hlHeal()}, caster, first, second)

		spRunUnbidden(w)

		if got := spAt(t, w, 2).HP; got <= 50 {
			t.Errorf("entity 2 holds %d health, want the tie broken toward the lower id", got)
		}
		if got := spAt(t, w, 3).HP; got != 50 {
			t.Errorf("entity 3 holds %d health, want its own untouched 50", got)
		}
	})

	t.Run("nothing at all when every ally is at full health", func(t *testing.T) {
		caster := acCaster(1, 2, 2, 50, 1<<6, 6)
		w := hlWorld(t, 29, Relations{}, []SpellRule{hlHeal()}, caster, spEnt(2, 3, 3))

		Step(w, nil)

		if got := spAt(t, w, 1).Mana; got != 50 {
			t.Errorf("the caster holds %d mana with nobody to heal, want 50", got)
		}
		if got := spAt(t, w, 1).CastWait; got != 0 {
			t.Errorf("the caster's wait is %d after an attempt that found nobody, want 0", got)
		}
	})

	t.Run("the installed defensive bit does not turn Heal into a self-only buff", func(t *testing.T) {
		caster := acCaster(1, 2, 2, 50, 1<<6, 6)
		hurt := spEnt(2, 3, 2)
		hurt.HP = 40
		heal := hlHeal()
		heal.Defensive = true
		w := hlWorld(t, 31, Relations{}, []SpellRule{heal}, caster, hurt)
		spRunUnbidden(w)
		if got := spAt(t, w, 2).HP; got <= 40 {
			t.Errorf("wounded patient holds %d, want installed Defensive Heal to enumerate it", got)
		}
	})
}

// TestIdleHealUsesOwnerAllianceNeutralPriority states the owner-authored tier
// order independently from distance: own team first, then a locked ally, then
// a neutral. Each control makes the lower tier closer than the winner.
func TestIdleHealUsesOwnerAllianceNeutralPriority(t *testing.T) {
	build := func(rel Relations, targets ...Entity) *World {
		caster := acCaster(1, 2, 2, 50, 1<<6, 0)
		caster.Owner = 1
		return hlWorld(t, 61, rel, []SpellRule{hlHeal()}, append([]Entity{caster}, targets...)...)
	}
	wounded := func(id EntityID, owner uint32, x int32) Entity {
		e := spEnt(id, x, 2)
		e.Owner, e.HP = owner, 40
		return e
	}

	t.Run("own team outranks closer ally and neutral", func(t *testing.T) {
		// A diagonal hostile bit can be produced by a same-slot blow. It does
		// not turn one's own team into an ineligible Heal target.
		rel := engRel(t, [3]uint32{1, 1, relationHostile}, [3]uint32{1, 2, relationLocked})
		own := wounded(2, 1, 5)
		ally := wounded(3, 2, 3)
		neutral := wounded(4, 3, 3)
		w := build(rel, own, ally, neutral)
		spRunUnbidden(w)
		if spAt(t, w, 2).HP <= 40 || spAt(t, w, 3).HP != 40 || spAt(t, w, 4).HP != 40 {
			t.Fatalf("health own/ally/neutral = %d/%d/%d, want only own healed",
				spAt(t, w, 2).HP, spAt(t, w, 3).HP, spAt(t, w, 4).HP)
		}
	})

	t.Run("ally outranks closer neutral", func(t *testing.T) {
		rel := engRel(t, [3]uint32{1, 2, relationLocked})
		ally := wounded(2, 2, 5)
		neutral := wounded(3, 3, 3)
		w := build(rel, ally, neutral)
		spRunUnbidden(w)
		if spAt(t, w, 2).HP <= 40 || spAt(t, w, 3).HP != 40 {
			t.Fatalf("health ally/neutral = %d/%d, want only ally healed", spAt(t, w, 2).HP, spAt(t, w, 3).HP)
		}
	})
}

// TestIdleHealLeavesAnActiveMoveOrderAttached prevents the out-of-battle scan
// from detaching a mage from a current script/player move order.
func TestIdleHealLeavesAnActiveMoveOrderAttached(t *testing.T) {
	caster := acCaster(1, 2, 2, 50, 1<<6, 0)
	caster.Owner = 1
	hurt := spEnt(2, 3, 2)
	hurt.Owner, hurt.HP = 1, 40
	w := hlWorld(t, 67, Relations{}, []SpellRule{hlHeal()}, caster, hurt)
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 15, Y: 2}})
	ordered := spAt(t, w, 1)
	if !underCommand(ordered) {
		t.Fatal("fixture did not attach the move order")
	}
	for range 8 {
		Step(w, nil)
	}
	if got := spAt(t, w, 2).HP; got != 40 {
		t.Errorf("ordered mage healed target to %d, want untouched 40", got)
	}
	if got := spAt(t, w, 1); !underCommand(got) {
		t.Error("idle-heal scan detached the mage's current move order")
	}
}

// TestNearbyHostilityAloneDoesNotMakeAnIdleMageParticipateInBattle prevents a
// proximity scan from suppressing the owner's idle heal. The hostile is close,
// but neither side holds an attack target; the wounded teammate is still healed.
func TestNearbyHostilityAloneDoesNotMakeAnIdleMageParticipateInBattle(t *testing.T) {
	caster := acCaster(1, 2, 2, 50, 1<<6, 0)
	caster.Owner = 1
	hurt := spEnt(2, 3, 2)
	hurt.Owner, hurt.HP = 1, 40
	hostile := spEnt(3, 2, 3)
	hostile.Owner = 2
	w := hlWorld(t, 71, acEnemies(t), []SpellRule{hlHeal()}, caster, hurt, hostile)
	spRunUnbidden(w)
	if got := spAt(t, w, 2).HP; got <= 40 {
		t.Errorf("nearby but unengaged mage left ally at %d, want an idle heal", got)
	}
}

// ---------------------------------------------------------------- AC-9

func TestARefusedAutocastDoesNotResetTheWait(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what   string
		caster Entity
	}{
		{"one mana short of the cost", acCaster(1, 2, 2, 2, 1<<1, 1)},
		{"a caster that does not know the spell", acCaster(1, 2, 2, 50, 0, 1)},
		{"a caster that is not a mage", func() Entity {
			e := acCaster(1, 2, 2, 0, 1<<1, 1)
			e.MaxMana, e.Mana = 0, 0
			return e
		}()},
	} {
		t.Run(tc.what, func(t *testing.T) {
			caster := tc.caster
			caster.Owner = 1
			enemy := spEnt(2, 4, 4)
			enemy.Owner = 2
			w := hlWorld(t, 31, acEnemies(t), []SpellRule{hlArrow()}, caster, enemy)

			Step(w, nil)

			if got := spAt(t, w, 2).HP; got != 100 {
				t.Errorf("the enemy holds %d health, want its own untouched 100", got)
			}
			if got := spAt(t, w, 1).CastWait; got != 0 {
				t.Errorf("the wait is %d after a refused attempt, want 0 so the next tick tries again", got)
			}
		})
	}
}

func TestAnAutocastNamingNoRowIsANoOp(t *testing.T) {
	t.Parallel()

	caster := acCaster(1, 2, 2, 50, 0xffffffff, 27)
	caster.Owner = 1
	enemy := spEnt(2, 4, 4)
	enemy.Owner = 2
	w := hlWorld(t, 31, acEnemies(t), []SpellRule{hlArrow()}, caster, enemy)

	spRunUnbidden(w)

	if got := spAt(t, w, 1).Mana; got != 50 {
		t.Errorf("the caster holds %d mana, want its own untouched 50", got)
	}
	if got := spAt(t, w, 1).AutoSpell; got != 27 {
		t.Errorf("the autocast id is %d, want the 27 it was handed — an unresolvable id is not folded", got)
	}
}

func TestTheAutocastCommandStoresTheIDAndZeroClearsIt(t *testing.T) {
	t.Parallel()

	// Out of anyone's reach and with no enemy, so the sweep never fires and
	// the only change on these ticks is the command's own.
	caster := acCaster(1, 2, 2, 50, 1<<1, 0)
	w := hlWorld(t, 37, Relations{}, []SpellRule{hlArrow()}, caster)

	Step(w, []Command{{Kind: KindAutocast, Entity: 1, X: 1}})
	if got := spAt(t, w, 1).AutoSpell; got != 1 {
		t.Errorf("the autocast id is %d after the command, want 1", got)
	}
	if got := spAt(t, w, 1).CastWait; got != 0 {
		t.Errorf("the wait is %d after the toggle, want 0 — setting it starts no cooldown", got)
	}

	Step(w, []Command{{Kind: KindAutocast, Entity: 1, X: 0}})
	if got := spAt(t, w, 1).AutoSpell; got != 0 {
		t.Errorf("the autocast id is %d after being cleared, want 0", got)
	}

	Step(w, []Command{{Kind: KindAutocast, Entity: 1, X: 0x1ffff}})
	if got := spAt(t, w, 1).AutoSpell; got != 0 {
		t.Errorf("the autocast id is %d after an id wider than the field, want 0", got)
	}
}

func TestAnAutocastTouchesOnlyItsCasterItsTargetAndTheGenerator(t *testing.T) {
	t.Parallel()

	caster := acCaster(1, 2, 2, 50, 1<<1, 1)
	caster.Owner = 1
	enemy := spEnt(2, 4, 4)
	enemy.Owner = 2
	// Owner 1 as well, so it is not hostile to the caster and is never a
	// candidate — but it stands inside the row's own reach of both.
	bystander := spEnt(3, 3, 3)
	bystander.Owner, bystander.Group = 1, 7
	w := hlWorld(t, 41, acEnemies(t), []SpellRule{hlArrow()}, caster, enemy, bystander)

	was := spAt(t, w, 3)
	spRunUnbidden(w)
	if got := spAt(t, w, 3); got != was {
		t.Errorf("the bystander is\n %+v\nand was\n %+v", got, was)
	}
	if got := spAt(t, w, 2).HP; got == 100 {
		t.Fatal("nothing was cast at all — this test would pass for the wrong reason")
	}
}

func TestTwoWorldsAutocastingFromTheSameStateStayByteIdentical(t *testing.T) {
	t.Parallel()

	build := func() *World {
		caster := acCaster(1, 2, 2, 50, (1<<1)|(1<<6), 1)
		caster.Owner = 1
		healer := acCaster(4, 2, 3, 50, 1<<6, 6)
		healer.Owner = 1
		enemy := spEnt(2, 4, 4)
		enemy.Owner = 2
		hurt := spEnt(3, 3, 3)
		hurt.Owner, hurt.HP = 1, 30
		return hlWorld(t, 43, acEnemies(t), []SpellRule{hlArrow(), hlHeal()}, caster, enemy, hurt, healer)
	}

	a, b := build(), build()
	for i := 0; i < 60; i++ {
		Step(a, nil)
		Step(b, nil)
		if !bytes.Equal(hlBytes(t, a), hlBytes(t, b)) {
			t.Fatalf("two worlds stepped identically diverged on tick %d", i+1)
		}
	}
	if got := spAt(t, a, 2).HP; got == 100 {
		t.Error("no autocast landed in sixty ticks — this test would pass for the wrong reason")
	}
}
