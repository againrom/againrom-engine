package game

import (
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
)

// mirrorClass is a class with EVERY phase scalar distinct and positive, and ALL
// THREE tracks non-empty and distinct from each other, so no two fields of the
// derived descriptor share a value: a copy that reads the wrong source field
// cannot come out right by coincidence. Flip 1, so the layout pair is (9, 5)
// rather than the zero-valued-looking (16, 8).
//
// The ATTACK PAIR is here for that property and not for coverage. The class had
// an AttackPhases and no attack arrays, so when the descriptor gained a track and
// a gate for that block the mirror would have carried nil onto nil and false onto
// false — passing for exactly the two fields that had just arrived, which is the
// one case this fixture exists to make impossible.
func mirrorClass() *data.UnitClass {
	return &data.UnitClass{
		Flip:            1,
		MoveBeginPhases: 2,
		MovePhases:      3,
		AttackPhases:    4,
		DyingPhases:     5,
		BonePhases:      6,
		IdlePhases:      7,
		MoveAnimTime:    []int32{2, 1},
		MoveAnimFrame:   []int32{1, 0},
		IdleAnimTime:    []int32{1, 1, 1},
		IdleAnimFrame:   []int32{0, 2, 1},
		AttackAnimTime:  []int32{1, 2, 1, 1},
		AttackAnimFrame: []int32{0, 3, 1, 2},
	}
}

// Every field of the render tier's descriptor equals the same-named field of
// the data tier's, over a class whose fields are pairwise distinct.
func TestUnitAnimMirrorsEveryFieldOfTheDerivation(t *testing.T) {
	src := mirrorClass().Anim()
	got := unitAnim(src)

	from := reflect.ValueOf(src)
	to := reflect.ValueOf(got)
	toType := to.Type()
	for i := 0; i < toType.NumField(); i++ {
		name := toType.Field(i).Name
		want := from.FieldByName(name)
		if !want.IsValid() {
			t.Errorf("%s is on the render tier's descriptor and not on the data tier's — "+
				"the two are one descriptor spelt twice, so a field on one is a field on both", name)
			continue
		}
		if !reflect.DeepEqual(to.Field(i).Interface(), want.Interface()) {
			t.Errorf("%s = %v, want %v — the mirror carries values, it does not derive them",
				name, to.Field(i).Interface(), want.Interface())
		}
	}
}

// The dying slot in particular, read through the bundle's own type at both
// layouts and at an absent phase — the three values the death selection is
// asked to distinguish (SC-1).
func TestUnitAnimCarriesTheDyingSlot(t *testing.T) {
	for _, tc := range []struct {
		label string
		class *data.UnitClass
		want  int
	}{
		{"DY 5 at (9, 5)", mirrorClass(), 5},
		{"DY 3 at (16, 8)", &data.UnitClass{MovePhases: 1, DyingPhases: 3}, 3},
		{"DY absent", &data.UnitClass{MovePhases: 1, DyingPhases: -1}, 0},
	} {
		var a terrain.UnitAnim = unitAnim(tc.class.Anim())
		if a.DyingSlot != tc.want {
			t.Errorf("%s: DyingSlot = %d, want %d", tc.label, a.DyingSlot, tc.want)
		}
		if got := a.DyingBase + a.D*a.DyingSlot; got != a.TailBase {
			t.Errorf("%s: DyingBase + D*DyingSlot = %d, but TailBase is %d — the block "+
				"identity does not survive the mirror", tc.label, got, a.TailBase)
		}
	}
}
