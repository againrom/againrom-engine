package sim

import "testing"

func TestComputerMageCastsWithGroupSightOnlyInsideSpellRange(t *testing.T) {
	for _, tc := range []struct {
		name                               string
		human, second, unseen, otherGroup  bool
		otherOwner, deadScout, offMapScout bool
		x                                  int32
		want, approach                     bool
	}{
		{name: "computer", x: 9, want: true},
		{name: "player", human: true, x: 9},
		{name: "second game", second: true, x: 9},
		{name: "outside spell range", x: 13, approach: true},
		{name: "outside spell range unseen by group", unseen: true, x: 13},
		{name: "unseen by group", unseen: true, x: 9},
		{name: "other group", otherGroup: true, x: 9},
		{name: "other owner", otherOwner: true, x: 9},
		{name: "dead scout", deadScout: true, x: 9},
		{name: "off map scout", offMapScout: true, x: 9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mage := spMage(1, 2, 8, 30, 100, 100, 1<<1)
			mage.Owner, mage.Group, mage.ScanRange = 2, 16, 2
			mage.AttackCharge, mage.AttackRelax = 8, 4
			if tc.human {
				mage.Owner = SelfSlot
			}
			target := spEnt(2, tc.x, 8)
			target.Owner = 1
			scout := spEnt(3, tc.x, 7)
			scout.Owner, scout.Group, scout.ScanRange = mage.Owner, mage.Group, 4
			if tc.unseen {
				scout.X = 1
			}
			if tc.otherGroup {
				scout.Group++
			}
			if tc.otherOwner {
				scout.Owner = 3
			}
			if tc.deadScout {
				scout.HP = -10
			}
			scout.OffMap = tc.offMapScout
			w := hlWorld(t, 7, acEnemies(t), []SpellRule{hlArrow()}, mage, target, scout)
			if tc.second {
				w.rom2 = &rom2ScriptState{}
			}
			if w.actorSeesEntity(0, 1) {
				t.Fatal("fixture target is personally visible")
			}
			if reason := w.BookSpellRefusal(1, 2, 1); (reason == "") != tc.want {
				t.Fatalf("admission = %q, want admitted %v", reason, tc.want)
			}
			if !w.orderAttack(0, 2) {
				t.Fatal("attack order refused")
			}
			k, casting := w.bookCastIndex(1)
			if casting != (tc.want || tc.approach) {
				t.Fatalf("casts %+v, want cast %v", w.bookCasts, tc.want || tc.approach)
			}
			if tc.approach {
				if w.bookCasts[k].Phase != bookApproach || !w.entities[0].HasAttackTarget || w.entities[0].AttackTarget != 2 {
					t.Fatalf("a target the group sees out of spell range armed no approach: %+v", w.bookCasts[k])
				}
				return
			}
			if !tc.want {
				if !w.entities[0].HasAttackTarget || w.entities[0].AttackTarget != 2 {
					t.Fatal("ordinary attack fallback changed")
				}
				return
			}
			if w.bookCasts[k].Phase != bookCharging || w.entities[0].HasTarget {
				t.Fatal("in-range mage approached instead of charging")
			}
			if ev, ok := runCast(w, 60); !ok || ev.Caster != 1 || ev.Target != 2 {
				t.Fatalf("cast did not release: %+v", ev)
			}
			if w.entities[0].X != mage.X || w.entities[0].Y != mage.Y || w.entities[1].HP >= target.HP {
				t.Fatal("mage moved or released without applying damage")
			}
		})
	}
}

// A computer mage whose victim only another member sees, beyond its spell
// range, walks until the victim is in range and casts from that cell; it does
// not close to weapon reach first (AI-760, AI-678).
func TestComputerMageWalksIntoSpellRangeOnGroupSightAndCasts(t *testing.T) {
	mage := spMage(1, 2, 8, 30, 100, 100, 1<<1)
	mage.Owner, mage.Group, mage.ScanRange, mage.Speed = 2, 16, 2, 60
	mage.AttackCharge, mage.AttackRelax = 8, 4
	target := spEnt(2, 15, 8)
	target.Owner, target.HP, target.MaxHP = 1, 500, 500
	scout := spEnt(3, 15, 7)
	scout.Owner, scout.Group, scout.ScanRange = mage.Owner, mage.Group, 4
	w := hlWorld(t, 7, acEnemies(t), []SpellRule{hlArrow()}, mage, target, scout)
	if w.actorSeesEntity(0, 1) || w.BookSpellRefusal(1, 2, 1) == "" {
		t.Fatal("fixture: the mage sees its victim or has it in spell range")
	}
	if !w.orderAttack(0, 2) {
		t.Fatal("attack order refused")
	}
	var in cell
	reached := false
	for range 400 {
		events := StepObserved(w, nil)
		m := spAt(t, w, 1)
		if !reached && w.BookSpellRefusal(1, 2, 1) == "" {
			in, reached = cellOf(&m), true
		}
		for _, ev := range events {
			if ev.Caster != 1 {
				continue
			}
			if !reached || ev.Target != 2 || cellOf(&m) != in {
				t.Fatalf("cast %+v from %d,%d; the first cell in spell range was %d,%d (reached %v)", ev, m.X, m.Y, in.x, in.y, reached)
			}
			return
		}
	}
	m := spAt(t, w, 1)
	t.Fatalf("the mage never cast: at %d,%d, first in-range cell %d,%d (reached %v)", m.X, m.Y, in.x, in.y, reached)
}
