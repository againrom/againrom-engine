package sim

import "testing"

// emptyWalkStaff is the weapon Spell of the staff fixture: a damaging unit row
// the strike releases for no mana.
func emptyWalkStaff() SpellRule {
	return SpellRule{ID: 26, ManaCost: 20, School: 1, MaxRange: 7,
		DamageMin: 2, DamageMax: 3, TargetsUnit: true, Damaging: true}
}

// emptyWalkMage is engageMageWorld's computer mage knowing Fire Arrow (cost 3)
// with three mana, beside its victim 2, in the second game or the first. A
// staff gives it the weapon Spell 26.
func emptyWalkMage(t *testing.T, second, staff bool) *World {
	t.Helper()
	w := engageMageWorld(t, 1<<1, 3, hlArrow(), emptyWalkStaff())
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	e := &w.entities[0]
	e.Reach, e.DamageBase, e.AlwaysHits = 1, 3, true
	if staff {
		e.WeaponSpell, e.WeaponSpellSource, e.WeaponSpellLevel = 26, WeaponSpellItem, 1
	}
	return w
}

// emptyWalkCast runs the mage's first engage, a Fire Arrow at victim 2 kept
// as its retained order, to the release; the mage is then out of mana.
func emptyWalkCast(t *testing.T, w *World) {
	t.Helper()
	if !w.orderAttack(0, 2) {
		t.Fatal("fixture: the first engage cast nothing")
	}
	if k, ok := w.bookCastIndex(1); !ok || w.bookCasts[k].Spell != 1 || !w.bookCasts[k].Retained {
		t.Fatalf("fixture: the first engage armed %+v, want a retained Fire Arrow", w.bookCasts)
	}
	if ev, ok := runCast(w, 60); !ok || ev.Target != 2 {
		t.Fatalf("fixture: the Fire Arrow never released: %+v", ev)
	}
	if m := w.entities[0].Mana; m >= hlArrow().ManaCost {
		t.Fatalf("fixture: the mage keeps %d mana after the cast", m)
	}
}

// emptyWalkEngage runs the engage the group decision issues every 16 ticks
// for up to n ticks and reports the tick a strike or a staff release starts.
func emptyWalkEngage(w *World, n int) (int, bool) {
	for k := range n {
		if k%scriptCycle == 0 {
			w.orderAttack(0, 2)
		}
		Step(w, nil)
		if e := w.entities[0]; e.HasAttackTarget && e.AttackTarget == 2 &&
			(e.AttackPhase == AttackCharging || e.AttackPhase == AttackCasting) {
			return k, true
		}
	}
	return 0, false
}

// A second-game computer mage whose book holds no spell it can pay for
// attacks: the book walk's miss lets the engage's attack order replace the
// retained cast order (R2-ENGINE-902, R2-ENGINE-351; owner observation of the
// necromancers). With a staff the strike releases the staff's Spell; without
// one it is a plain blow. Before, the retained Fire Arrow held the mage
// standing until its mana came back.
func TestASecondGameMageOutOfManaStrikesWithItsStaff(t *testing.T) {
	for _, staff := range []bool{true, false} {
		w := emptyWalkMage(t, true, staff)
		emptyWalkCast(t, w)
		k, ok := emptyWalkEngage(w, 4*scriptCycle)
		if !ok {
			t.Fatalf("staff %v: no strike started in %d ticks; casts %+v", staff, 4*scriptCycle, w.bookCasts)
		}
		if _, cast := w.bookCastIndex(1); cast {
			t.Fatalf("staff %v: the retained cast order outlived the miss: %+v", staff, w.bookCasts)
		}
		want := AttackCharging
		if staff {
			want = AttackCasting
		}
		if got := w.entities[0].AttackPhase; got != want {
			t.Fatalf("staff %v: tick %d phase %v, want %v", staff, k, got, want)
		}
	}
}

// With the mana for its book spell the second-game mage still casts it: the
// draw rewrites the retained order and the Fire Arrow fires again.
func TestASecondGameMageWithTheManaStillCastsFromItsBook(t *testing.T) {
	w := emptyWalkMage(t, true, true)
	emptyWalkCast(t, w)
	w.entities[0].Mana = hlArrow().ManaCost
	w.orderAttack(0, 2)
	if k, ok := w.bookCastIndex(1); !ok || w.bookCasts[k].Spell != 1 {
		t.Fatalf("the paid engage holds %+v, want the Fire Arrow", w.bookCasts)
	}
	if ev, ok := runCast(w, 60); !ok || ev.Spell != 1 || ev.Target != 2 {
		t.Fatalf("the second Fire Arrow never released: %+v", ev)
	}
	if e := w.entities[0]; e.AttackPhase == AttackCharging || e.AttackPhase == AttackCasting {
		t.Fatalf("the paying mage loaded a strike, phase %v", e.AttackPhase)
	}
}

// Control: a second-game caster with a staff and no book strikes at once, as
// before.
func TestASecondGameStaffCasterWithoutABookStrikesAtOnce(t *testing.T) {
	w := emptyWalkMage(t, true, true)
	w.entities[0].KnownSpells = 0
	if !w.orderAttack(0, 2) {
		t.Fatal("the engage refused")
	}
	if _, cast := w.bookCastIndex(1); cast {
		t.Fatalf("a caster without a book armed %+v", w.bookCasts)
	}
	if k, ok := emptyWalkEngage(w, scriptCycle); !ok || k > 1 {
		t.Fatalf("the staff release started at tick %d (%v), want at once", k, ok)
	}
}

// emptyWalkFirstHash is the first-game run's World hash at its last tick,
// taken before the second game's miss rule existed: the first game keeps its
// waiting order tick for tick.
const emptyWalkFirstHash = 0xc7b70e7d21721c20

// Control: a first-game mage in the same fight keeps today's order: out of
// mana its retained Fire Arrow stands and waits, no strike starts, and the
// World over the run hashes as before.
func TestAFirstGameMageOutOfManaKeepsItsWaitingOrder(t *testing.T) {
	w := emptyWalkMage(t, false, true)
	emptyWalkCast(t, w)
	if k, ok := emptyWalkEngage(w, 4*scriptCycle); ok {
		t.Fatalf("a first-game mage started a strike at tick %d", k)
	}
	if k, ok := w.bookCastIndex(1); !ok || w.bookCasts[k].Spell != 1 || !w.bookCasts[k].Retained {
		t.Fatalf("the first-game retained order is %+v, want the Fire Arrow waiting", w.bookCasts)
	}
	if got := w.Hash(); got != emptyWalkFirstHash {
		t.Fatalf("first-game hash %#x, want %#x", got, uint64(emptyWalkFirstHash))
	}
}
