package sim

import "testing"

func TestComputerMageCastsWithGroupSightOnlyInsideSpellRange(t *testing.T) {
	for _, tc := range []struct {
		name                               string
		human, second, unseen, otherGroup  bool
		otherOwner, deadScout, offMapScout bool
		x                                  int32
		want                               bool
	}{
		{name: "computer", x: 9, want: true},
		{name: "player", human: true, x: 9},
		{name: "second game", second: true, x: 9},
		{name: "outside spell range", x: 13},
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
			if casting != tc.want {
				t.Fatalf("casts %+v, want cast %v", w.bookCasts, tc.want)
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
