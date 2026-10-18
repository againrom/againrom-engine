package sim

import "testing"

// approachWorld holds the player's mage at (1,8) and a hostile fighter at
// (12,8), eleven cells off: beyond the arrow's and the ball's reach and
// inside the mage's sight.
func approachWorld(t *testing.T, more ...Entity) *World {
	t.Helper()
	return approachWorldSeeing(t, sightRings, more...)
}

func approachWorldSeeing(t *testing.T, scan uint8, more ...Entity) *World {
	t.Helper()
	caster := spMage(1, 1, 8, 30, 100, 100, 1<<1|1<<2|1<<26)
	caster.Owner, caster.Speed, caster.AttackCharge, caster.AttackRelax = SelfSlot, 30, 8, 4
	caster.ScanRange = scan
	enemy := spEnt(2, 12, 8)
	enemy.Owner, enemy.HP, enemy.MaxHP = 2, 500, 500
	ball := SpellRule{ID: 2, ManaCost: 5, School: 1, MaxRange: 5, Area: true, Distribution: 2, Radius: 1,
		Damaging: true, DamageMin: 10, DamageMax: 10}
	port := SpellRule{ID: 26, ManaCost: 7, School: 5, MaxRange: 5, Defensive: true}
	return hlWorld(t, 7, acEnemies(t), []SpellRule{hlArrow(), ball, port}, append([]Entity{caster, enemy}, more...)...)
}

// armApproach issues cmd and requires an armed approach that has paid nothing.
func armApproach(t *testing.T, w *World, cmd Command) {
	t.Helper()
	events := StepObserved(w, []Command{cmd})
	k, ok := w.bookCastIndex(1)
	if len(events) != 0 || !ok || w.bookCasts[k].Phase != bookApproach || spAt(t, w, 1).Mana != 100 {
		t.Fatalf("the cast out of reach armed no approach: events %v, casts %+v", events, w.bookCasts)
	}
	if !spAt(t, w, 1).HasTarget {
		t.Fatal("the armed caster holds no walk")
	}
}

// runCast steps w until caster 1 releases a cast, at most n ticks.
func runCast(w *World, n int) (CastEvent, bool) {
	for range n {
		for _, ev := range StepObserved(w, nil) {
			if ev.Caster == 1 {
				return ev, true
			}
		}
	}
	return CastEvent{}, false
}

func TestAPlayerCastAtAUnitOutOfReachWalksAndCasts(t *testing.T) {
	w := approachWorld(t)
	if r := w.BookSpellRefusal(1, 2, 1); r != refusalTargetOutOfRange {
		t.Fatalf("fixture: refusal %q, want %q", r, refusalTargetOutOfRange)
	}
	armApproach(t, w, Cast(1, 2, 1))
	ev, ok := runCast(w, 300)
	c := spAt(t, w, 1)
	if !ok || ev.Target != 2 || ev.AtCell {
		t.Fatalf("the approached cast never landed on its unit: %+v", ev)
	}
	if c.X <= 1 || c.Mana != 97 || spAt(t, w, 2).HP >= 500 {
		t.Fatalf("caster at %d mana %d, target health %d: no walk, no payment or no hit", c.X, c.Mana, spAt(t, w, 2).HP)
	}
	if c.HasTarget || c.HasAttackTarget {
		t.Fatal("the caster did not stand after the cast")
	}
}

func TestAPlayerCastAtACellOutOfReachWalksAndCasts(t *testing.T) {
	w := approachWorld(t)
	at := CellPoint{X: 12, Y: 8}
	if r := w.BookSpellCellRefusal(1, at.X, at.Y, 2); r != refusalCellOutOfRange {
		t.Fatalf("fixture: refusal %q, want %q", r, refusalCellOutOfRange)
	}
	armApproach(t, w, CastAt(1, 2, at))
	ev, ok := runCast(w, 300)
	if !ok || !ev.AtCell || ev.ToX != at.X || ev.ToY != at.Y {
		t.Fatalf("the approached cast never landed on its cell: %+v", ev)
	}
	if c := spAt(t, w, 1); c.X <= 1 || c.Mana != 95 {
		t.Fatalf("caster at %d mana %d: no walk or no payment", c.X, c.Mana)
	}
}

func TestATeleportOutOfReachStillWalksAndTeleports(t *testing.T) {
	w := approachWorld(t)
	at := CellPoint{X: 13, Y: 3}
	armApproach(t, w, CastAt(1, 26, at))
	if _, ok := runCast(w, 300); !ok {
		t.Fatal("the approached Teleport never cast")
	}
	for range 40 {
		Step(w, nil)
	}
	if c := spAt(t, w, 1); c.X != at.X || c.Y != at.Y {
		t.Fatalf("the caster stands at (%d,%d), want the Teleport cell", c.X, c.Y)
	}
}

func TestAnotherOrderReplacesTheCastApproach(t *testing.T) {
	for _, tc := range []struct {
		name  string
		order Command
	}{
		{"move", MoveTo(1, CellPoint{X: 1, Y: 14})},
		{"stand", GroupStance(1, int32(OrderStandGround), 0)},
		{"attack the same unit", Attack(1, 2)},
		{"another cast in reach", Cast(1, 1, 26)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := approachWorld(t)
			armApproach(t, w, Cast(1, 2, 1))
			Step(w, nil)
			Step(w, []Command{tc.order})
			if k, ok := w.bookCastIndex(1); ok && w.bookCasts[k].Spell == 1 {
				t.Fatalf("the cast approach survived the %s order: %+v", tc.name, w.bookCasts[k])
			}
			for range 300 {
				for _, ev := range StepObserved(w, nil) {
					if ev.Caster == 1 && ev.Spell == 1 {
						t.Fatalf("the replaced cast landed after the %s order", tc.name)
					}
				}
			}
		})
	}
}

func TestATargetThatDiesEndsTheApproachAndStandsTheCaster(t *testing.T) {
	w := approachWorld(t)
	armApproach(t, w, Cast(1, 2, 1))
	Step(w, nil)
	Step(w, []Command{TerminalKill(2)})
	for range 40 {
		Step(w, nil)
	}
	if _, ok := w.bookCastIndex(1); ok {
		t.Fatal("the cast outlived its dead target")
	}
	if c := spAt(t, w, 1); c.HasTarget || c.Mana != 100 {
		t.Fatalf("the caster walks on or paid after its target died: %+v", c.Mana)
	}
}

func TestACastInReachStartsOnTheClickTick(t *testing.T) {
	w := approachWorld(t)
	w.entities[indexOfEntity(w.entities, 2)].X = 4
	Step(w, []Command{Cast(1, 2, 1)})
	k, ok := w.bookCastIndex(1)
	if !ok || w.bookCasts[k].Phase != bookCharging || spAt(t, w, 1).X != 1 || spAt(t, w, 1).HasTarget {
		t.Fatalf("a cast in reach did not charge in place on its click: %+v", w.bookCasts)
	}
}

func TestACastApproachSavesAndLoads(t *testing.T) {
	w := approachWorld(t)
	armApproach(t, w, Cast(1, 2, 1))
	for range 3 {
		Step(w, nil)
	}
	back := worldRoundTripForTest(t, w)
	if back.Hash() != w.Hash() {
		t.Fatal("a reload changed the digest of a cast approach")
	}
	landed := false
	for tick := range 300 {
		a, b := StepObserved(w, nil), StepObserved(back, nil)
		if w.Hash() != back.Hash() || len(a) != len(b) {
			t.Fatalf("the reloaded approach diverged at tick %d", tick)
		}
		for _, ev := range b {
			landed = landed || ev.Caster == 1
		}
	}
	if !landed {
		t.Fatal("the reloaded approach never cast")
	}
}

// engageMageWorld holds a computer mage at (2,8) with a hostile fighter next
// to it and a second one far off.
func engageMageWorld(t *testing.T, known uint32, mana int32, spells ...SpellRule) *World {
	t.Helper()
	mage := spMage(1, 2, 8, 30, 100, mana, known)
	mage.Owner, mage.Speed, mage.AttackCharge, mage.AttackRelax = 2, 30, 8, 4
	near := spEnt(2, 3, 8)
	near.Owner = 1
	far := spEnt(3, 13, 8)
	far.Owner = 1
	return hlWorld(t, 7, acEnemies(t), spells, mage, near, far)
}

func TestAComputerMageCastsAtItsEngageVictim(t *testing.T) {
	for _, victim := range []EntityID{2, 3} {
		w := engageMageWorld(t, 1<<1, 100, hlArrow())
		if !w.orderAttack(0, victim) {
			t.Fatalf("victim %d: the engage refused", victim)
		}
		k, ok := w.bookCastIndex(1)
		if !ok || w.bookCasts[k].Target != victim || w.bookCasts[k].Spell != 1 {
			t.Fatalf("victim %d: the mage's cast is %+v, want spell 1 at the victim", victim, w.bookCasts)
		}
		want := bookCharging
		if victim == 3 {
			want = bookApproach
		}
		if w.bookCasts[k].Phase != want {
			t.Fatalf("victim %d: cast phase %d, want %d", victim, w.bookCasts[k].Phase, want)
		}
	}
}

func TestAComputerMageWithAnEmptyListCastsIdTwentyEight(t *testing.T) {
	slow := hlArrow()
	slow.ID, slow.ManaCost = 28, 40
	for _, holds := range []bool{true, false} {
		known := uint32(1 << 1)
		if holds {
			known |= 1 << 28
		}
		w := engageMageWorld(t, known, 0, hlArrow(), slow)
		w.orderAttack(0, 2)
		k, ok := w.bookCastIndex(1)
		switch {
		case holds && (!ok || w.bookCasts[k].Spell != 28 || w.bookCasts[k].Target != 2 || w.bookCasts[k].Phase != bookPending):
			t.Fatalf("a book holding id 28 with nothing affordable: casts %+v, want id 28 waiting for mana", w.bookCasts)
		case !holds && (ok || !w.entities[0].HasAttackTarget || w.entities[0].AttackTarget != 2):
			t.Fatalf("a book without id 28 and nothing affordable: casts %+v, want the ordinary engage", w.bookCasts)
		}
	}
}

func TestAHighMindMageHandsThreeInTenEngagesBack(t *testing.T) {
	handed, cast := 0, 0
	for seed := uint64(1); seed <= 200; seed++ {
		w := engageMageWorld(t, 1<<1, 100, hlArrow())
		w.entities[0].Mind = 60
		w.rng = rng{state: seed}
		w.orderAttack(0, 2)
		if _, ok := w.bookCastIndex(1); ok {
			cast++
		} else if w.entities[0].HasAttackTarget {
			handed++
		}
	}
	if handed == 0 || cast == 0 || handed+cast != 200 || handed > 90 {
		t.Fatalf("over 200 seeds %d engages were handed back and %d cast", handed, cast)
	}
}

func TestACastAtAUnitOnlyAnotherMemberSeesWalksUntilTheCasterSees(t *testing.T) {
	for _, scout := range []bool{true, false} {
		var more []Entity
		if scout {
			ally := spEnt(3, 10, 8)
			ally.Owner, ally.ScanRange = SelfSlot, sightRings
			more = append(more, ally)
		}
		w := approachWorldSeeing(t, 3, more...)
		if r := w.BookSpellRefusal(1, 2, 1); r != refusalTargetNotVisible {
			t.Fatalf("fixture: refusal %q, want %q", r, refusalTargetNotVisible)
		}
		events := StepObserved(w, []Command{Cast(1, 2, 1)})
		_, armed := w.bookCastIndex(1)
		if len(events) != 0 || armed != scout || spAt(t, w, 1).HasTarget != scout {
			t.Fatalf("scout %v: armed %v walking %v", scout, armed, spAt(t, w, 1).HasTarget)
		}
		if !scout {
			continue
		}
		ev, ok := runCast(w, 300)
		if !ok || ev.Target != 2 || spAt(t, w, 2).HP >= 500 {
			t.Fatalf("the cast at a unit only the scout saw never landed: %+v", ev)
		}
	}
}

func TestATargetTakenOffTheMapEndsTheApproachUnpaid(t *testing.T) {
	w := approachWorld(t)
	armApproach(t, w, Cast(1, 2, 1))
	Step(w, nil)
	if !w.takeOffMap(indexOfEntity(w.entities, 2)) {
		t.Fatal("fixture: the target did not leave the map")
	}
	if ev, ok := runCast(w, 300); ok {
		t.Fatalf("the approach cast at a target off the map: %+v", ev)
	}
	if _, ok := w.bookCastIndex(1); ok {
		t.Fatal("the approach outlived its target leaving the map")
	}
	if c := spAt(t, w, 1); c.HasTarget || c.Mana != 100 || spAt(t, w, 2).HP != 500 {
		t.Fatalf("walking %v, mana %d, target health %d: want standing, unpaid, unhurt", c.HasTarget, c.Mana, spAt(t, w, 2).HP)
	}
}

func TestADistantCastClickedDuringAStrikeIsKeptAndLands(t *testing.T) {
	for _, atCell := range []bool{false, true} {
		struck := spEnt(3, 2, 8)
		struck.Owner, struck.HP, struck.MaxHP = 2, 500, 500
		w := approachWorld(t, struck)
		if !w.attachAttack(0, 3, AttackTargetUnit, false) {
			t.Fatal("fixture: no strike")
		}
		w.entities[0].AttackPhase, w.entities[0].AttackCountdown = AttackCharging, 3
		cmd := Cast(1, 2, 1)
		if atCell {
			cmd = CastAt(1, 2, CellPoint{X: 12, Y: 8})
		}
		Step(w, []Command{cmd})
		if _, book := w.bookCastIndex(1); !book && !w.entities[0].PendingOrder.IsCastRow() {
			t.Fatalf("cell %v: the distant cast clicked during a strike was dropped", atCell)
		}
		ev, ok := runCast(w, 400)
		if !ok || ev.AtCell != atCell || !atCell && ev.Target != 2 || atCell && (ev.ToX != 12 || ev.ToY != 8) {
			t.Fatalf("cell %v: the kept cast never landed on its target: %+v %v", atCell, ev, ok)
		}
	}
}

func TestAPlayerCreatureWalksItsDistantBookCast(t *testing.T) {
	for _, cmd := range []Command{Cast(1, 2, 1), CastAt(1, 2, CellPoint{X: 12, Y: 8})} {
		w := approachWorld(t)
		w.entities[0].CreatureSpells = creatureCertainSlot(1)
		armApproach(t, w, cmd)
		ev, ok := runCast(w, 300)
		if !ok || ev.AtCell != (cmd.Kind == KindCastAt) || spAt(t, w, 1).X <= 1 {
			t.Fatalf("kind %v: the creature's distant book cast never walked and landed: %+v", cmd.Kind, ev)
		}
	}
}

func TestAFarMageWithIdTwentyEightWaitsForManaInReach(t *testing.T) {
	slow := hlArrow()
	slow.ID, slow.ManaCost = 28, 40
	w := engageMageWorld(t, 1<<28, 0, slow)
	w.entities[indexOfEntity(w.entities, 2)].Owner = 2
	w.orderAttack(0, 3)
	for tick := 1; tick <= 300; tick++ {
		w.entities[0].Mana = 0
		Step(w, nil)
		if _, ok := w.bookCastIndex(1); !ok {
			t.Fatalf("tick %d: the id-28 order was lost at (%d,%d)", tick, w.entities[0].X, w.entities[0].Y)
		}
	}
	k, _ := w.bookCastIndex(1)
	if w.bookCasts[k].Phase != bookPending || w.entities[0].X <= 2 {
		t.Fatalf("the mage at %d holds phase %d, want a walk and the order waiting for mana", w.entities[0].X, w.bookCasts[k].Phase)
	}
	w.entities[0].Mana = 100
	if ev, ok := runCast(w, 100); !ok || ev.Target != 3 || ev.Spell != 28 {
		t.Fatalf("the waiting id-28 order never cast once the mana came: %+v", ev)
	}
}
