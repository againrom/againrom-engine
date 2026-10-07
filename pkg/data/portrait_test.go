package data_test

import (
	"testing"

	"againrom/pkg/data"
)

// UNIT-PICT-035: the compose/load split is a test on the type id, and the
// shipped roster's own partition follows from it rather than being written
// down. Both are asserted — the test at its boundary, and the partition over
// the roster the claim measures.
func TestComposesFigureSplitsAtTheEnginesOwnBoundary(t *testing.T) {
	for _, tc := range []struct {
		id   int32
		want bool
	}{
		{0, true}, {1, true}, {24, true}, {25, true},
		{26, false}, {27, false}, {64, false}, {80, false}, {81, false},
	} {
		if got := data.ComposesFigure(tc.id); got != tc.want {
			t.Errorf("ComposesFigure(%d) = %v, want %v", tc.id, got, tc.want)
		}
	}

	// The shipped roster is 1..27 and 64..80, and over it the partition must
	// come out exactly as the claim states: 1..24 compose, 26, 27 and 64..80
	// load a file. 25 is not in the roster, which is why the boundary above
	// and this walk disagree about it without either being wrong.
	var composed, flat []int32
	for id := int32(1); id <= 80; id++ {
		if (id > 27 && id < 64) || id == 25 {
			continue
		}
		if data.ComposesFigure(id) {
			composed = append(composed, id)
		} else {
			flat = append(flat, id)
		}
	}
	if len(composed) != 24 || composed[0] != 1 || composed[23] != 24 {
		t.Errorf("the composing set is %v, want 1..24", composed)
	}
	if len(flat) != 19 || flat[0] != 26 || flat[1] != 27 || flat[2] != 64 || flat[18] != 80 {
		t.Errorf("the flat-portrait set is %v, want 26, 27 and 64..80", flat)
	}
}

// UNIT-PICT-036, UNIT-PICT-037: two leaves off one name, and the tier-1
// truncation is the whole difference.
func TestPortraitPaths(t *testing.T) {
	if got := data.PortraitPath("Bee"); got != "infowindow/Bee.bmp" {
		t.Errorf("PortraitPath = %q", got)
	}

	for _, tc := range []struct {
		tier int32
		want string
	}{
		// The digit is dropped at tier 1 and at tier 1 alone — which is why the
		// shipped set has no `<name>1`.
		{1, "infowindow/Bee.bmp"},
		{2, "infowindow/Bee2.bmp"},
		{3, "infowindow/Bee3.bmp"},
		{4, "infowindow/Bee4.bmp"},
		// The formatter admits any integer; the bound on the digit is the
		// shipped palette arrays, not this address.
		{0, "infowindow/Bee0.bmp"},
		{5, "infowindow/Bee5.bmp"},
	} {
		if got := data.PortraitTierPath("Bee", tc.tier); got != tc.want {
			t.Errorf("PortraitTierPath(tier %d) = %q, want %q", tc.tier, got, tc.want)
		}
	}

	// A tier-4 class's two addresses differ, which is the consequence worth a
	// criterion of its own: the dialogue window and the info panel show
	// different files for the same actor.
	if a, b := data.PortraitPath("Bee"), data.PortraitTierPath("Bee", 4); a == b {
		t.Errorf("the info-panel and dialogue addresses agree at tier 4: %q", a)
	}

	// An empty name addresses nothing rather than a directory with an
	// extension on it.
	if got := data.PortraitPath(""); got != "" {
		t.Errorf("PortraitPath(\"\") = %q, want the empty string", got)
	}
	if got := data.PortraitTierPath("", 3); got != "" {
		t.Errorf("PortraitTierPath(\"\", 3) = %q, want the empty string", got)
	}
}
