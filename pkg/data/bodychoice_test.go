package data_test

import (
	"testing"

	"againrom/pkg/data"
)

// choiceList is an invented list: row 1 bare-handed, row 2 a one-handed body,
// row 3 a two-handed one, row 4 blank.
var choiceList = data.NewBodyList(data.BodyUnarmed, data.BodySwordsman, data.BodyAxeman2H, "")

func wielding(row int, shield bool) data.Equipment {
	var e data.Equipment
	e.SetCode(1, data.ItemCode(row))
	if shield {
		e.SetCode(2, data.ComposeItemCode(0, 2, 0, 1))
	}
	return e
}

// A mapping changes the body of its own row and of no other, and leaves the
// class key on the shipped entry.
func TestWeaponBodyChangesThatRowOnly(t *testing.T) {
	l := choiceList.WithWeaponBody(3, data.BodyClubman)
	if choiceList.Len() != 4 || l.Len() != 4 || l.Entry(2) != data.BodyAxeman2H {
		t.Fatal("a mapping changed the shipped entries")
	}
	if _, chosen := choiceList.WeaponBody(3); chosen {
		t.Fatal("WithWeaponBody wrote through to the list it was called on")
	}
	for row := 0; row <= 5; row++ {
		before, okBefore := data.HeroBodyFor(choiceList, wielding(row, false))
		after, okAfter := data.HeroBodyFor(l, wielding(row, false))
		want := before
		if row == 3 {
			want = data.BodyClubman
		}
		if after != want || okAfter != okBefore {
			t.Errorf("row %d: %q %v, want %q %v", row, after, okAfter, want, okBefore)
		}
	}
	name, dir, class, ok := data.HeroAppearance(l, wielding(3, false), false, false)
	shippedClass, _ := data.HeroBodyClass(data.BodyAxeman2H)
	if name != data.BodyClubman || dir != data.HeroDirHeroesLight || class != shippedClass || !ok {
		t.Fatalf("HeroAppearance = %q %q %d %v, want clubman, heroes_l, class %d", name, dir, class, ok, shippedClass)
	}
	// The shield form follows the drawn body; the dying body ignores it.
	if name, _, class, _ := data.HeroAppearance(l, wielding(3, true), false, false); name != "clubman_" || class != data.HeroUnmatchedClass {
		t.Fatalf("with a shield: %q class %d, want clubman_ and the class axeman2h_ leaves, %d", name, class, data.HeroUnmatchedClass)
	}
	if name, _, _, _ := data.HeroAppearance(l, wielding(3, false), false, true); name != data.BodyUnarmed {
		t.Fatalf("dying: %q, want unarmed", name)
	}
}

// A mapping on a row the shipped list names nothing for draws nothing: the
// mod cannot make a body appear where the game without it resolves none.
func TestWeaponBodyOnABlankRowResolvesNothing(t *testing.T) {
	l := choiceList.WithWeaponBody(4, data.BodyClubman).WithWeaponBody(9, data.BodyClubman)
	for _, row := range []int{4, 9} {
		if name, ok := data.HeroBodyFor(l, wielding(row, false)); ok || name != "" {
			t.Errorf("row %d: %q %v, want no name", row, name, ok)
		}
	}
}

// The paint order follows the drawn body: a one-handed mapping paints the
// shield last, a supplied body says which it paints last.
func TestFigureHeldLastFollowsTheDrawnBody(t *testing.T) {
	if got := data.FigureHeldLast(choiceList, wielding(3, true)); got != 1 {
		t.Fatalf("axeman2h paints slot %d last, want 1", got)
	}
	if got := data.FigureHeldLast(choiceList.WithWeaponBody(3, data.BodyClubman), wielding(3, true)); got != 2 {
		t.Fatalf("axeman2h drawn as clubman paints slot %d last, want 2", got)
	}
	mod := choiceList.WithWeaponBody(2, "clubman2h")
	for _, tc := range []struct {
		weaponLast bool
		want       int
	}{{true, 1}, {false, 2}} {
		l := mod.WithModBody("clubman2h", data.ModBody{WeaponLast: tc.weaponLast})
		if got := data.FigureHeldLast(l, wielding(2, true)); got != tc.want {
			t.Errorf("weapon-last %v: slot %d last, want %d", tc.weaponLast, got, tc.want)
		}
	}
}

// A supplied body takes the shield suffix only when the mod supplies that
// form too; the class key stays the shipped entry's either way.
func TestModBodyShieldForm(t *testing.T) {
	l := choiceList.WithWeaponBody(2, "clubman2h").WithModBody("clubman2h", data.ModBody{})
	shipped, _ := data.HeroBodyClass("swordsman_")
	if name, _, class, ok := data.HeroAppearance(l, wielding(2, true), false, false); name != "clubman2h" || class != shipped || !ok {
		t.Fatalf("no shield form supplied: %q class %d %v, want clubman2h class %d", name, class, ok, shipped)
	}
	l = l.WithModBody("clubman2h_", data.ModBody{})
	if name, _, _, _ := data.HeroAppearance(l, wielding(2, true), false, false); name != "clubman2h_" {
		t.Fatalf("shield form supplied: %q, want clubman2h_", name)
	}
	if got := l.ModBodies(); len(got) != 2 || got[0] != "clubman2h" || got[1] != "clubman2h_" {
		t.Fatalf("ModBodies = %v", got)
	}
	if rows := l.WeaponRows(); len(rows) != 1 || rows[0] != 2 {
		t.Fatalf("WeaponRows = %v", rows)
	}
}
