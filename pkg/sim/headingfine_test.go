package sim

import "testing"

// The heading is the axis when one magnitude exceeds twice the other and the
// diagonal otherwise, on the fine centres of both footprints (MAGIC-242). The
// controls separate it from a 22.5 degree octant rule, which splits 7/3 the
// other way, and from a whole-cell anchor delta.
func TestHeadingIsAnAxisPastTwoToOneOnFineCentres(t *testing.T) {
	for _, tc := range []struct {
		name   string
		dx, dy int64
		want   uint8
	}{
		{"exactly two to one is a diagonal", 512, 256, 96},
		{"just past two to one is the axis", 513, 256, 64},
		{"seven to three is the axis", 7 * 256, 3 * 256, 64},
		{"three to seven north is the axis", -3 * 256, -7 * 256, 0},
		{"a fine offset moves it across", 4*256 + 1, -2 * 256, 64},
		{"coincident centres", 0, 0, 224},
	} {
		if got := headingOf(tc.dx, tc.dy); got != tc.want {
			t.Errorf("%s: heading %d, want %d", tc.name, got, tc.want)
		}
	}
}

// A footprint is measured from its centre: a candidate two cells wide whose
// anchor is level with the caster lies half a cell further south than its
// anchor says.
func TestHeadingMeasuresFootprintCentres(t *testing.T) {
	w := &World{}
	caster := Entity{X: 10, Y: 10, TokenSize: 1}
	one := Entity{X: 14, Y: 12, TokenSize: 1}
	two := Entity{X: 14, Y: 10, TokenSize: 2}
	// Anchors 4 east and 2 south of the caster for the first is exactly two
	// to one, a diagonal. The wide body's centre is 4.5 east and 0.5 south:
	// the axis.
	if got := w.headingBetween(caster, one); got != 96 {
		t.Errorf("one-cell candidate: heading %d, want 96", got)
	}
	if got := w.headingBetween(caster, two); got != 64 {
		t.Errorf("two-cell candidate: heading %d, want 64", got)
	}
}

// Two foes at the same edge distance: the one whose heading from the caster is
// the primary's axis wins on turn cost. Seven east and three south is an axis
// heading at past two to one; the octant rule it replaces called it a
// diagonal and left the tie to list order (MAGIC-242).
func TestPrismaticSpraySecondaryTurnUsesTheTwoToOneHeading(t *testing.T) {
	w := prismaticWorld(t, 0, // cap 2
		prismaticFoeAt(2, 12, 1),
		prismaticFoeAt(3, 8, 5), // seven east, four south: a diagonal, listed first
		prismaticFoeAt(4, 8, 4), // seven east, three south: the east axis
	)
	if _, ids := prismaticCast(t, w, 2); len(ids) != 2 || ids[0] != 2 || ids[1] != 4 {
		t.Fatalf("victims %v, want [2 4]", ids)
	}
}
