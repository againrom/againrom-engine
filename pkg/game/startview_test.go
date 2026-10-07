package game

// The cell a mission's view opens on, as a function of what the start
// reported (AC-2, AC-3, AC-4).
//
// The rule is exercised over hand-built start reports. No archive is opened and
// no window is created — the door that consumes it needs a lawful install, so
// what is pinned here is the decision, and the wiring is witnessed by the build
// and by the headless mission drive.

import (
	"image"
	"testing"

	"againrom/pkg/mapload"
)

func TestStartViewCellIsTheFirstMemberOrTheDecidedDrop(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start mapload.Start
		want  image.Point
	}{
		{
			name:  "a party of one stands on the drop",
			start: mapload.Start{Drop: mapload.Cell{X: 43, Y: 46}, Cells: []mapload.Cell{{X: 43, Y: 46}}},
			want:  image.Pt(43, 46),
		},
		{
			name: "a party of several is anchored on its first member",
			start: mapload.Start{
				Drop:  mapload.Cell{X: 43, Y: 46},
				Cells: []mapload.Cell{{X: 43, Y: 46}, {X: 44, Y: 46}, {X: 43, Y: 47}},
			},
			want: image.Pt(43, 46),
		},
		{
			name:  "a party of none opens on the cell the start decided",
			start: mapload.Start{Drop: mapload.Cell{X: 12, Y: 7}},
			want:  image.Pt(12, 7),
		},
		{
			name:  "a drawn fallback is still the start's own answer",
			start: mapload.Start{Drop: mapload.Cell{X: 30, Y: 100}, Fallback: true},
			want:  image.Pt(30, 100),
		},
		{
			name: "the first member wins even where it is not the drop",
			// The engine places the hero at the drop exactly, so this pair does
			// not arise on the shipped path; it is here because the rule must be
			// "read the placed party", not "read Drop and call it the party".
			start: mapload.Start{Drop: mapload.Cell{X: 1, Y: 1}, Cells: []mapload.Cell{{X: 60, Y: 61}}},
			want:  image.Pt(60, 61),
		},
		{
			name:  "a start that decided nothing opens at the origin rather than failing",
			start: mapload.Start{},
			want:  image.Pt(0, 0),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := startViewCell(tc.start); got != tc.want {
				t.Errorf("startViewCell = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStartViewCellReadsTheReportAndNotTheDrop(t *testing.T) {
	drop := mapload.Cell{X: 43, Y: 46}
	a := startViewCell(mapload.Start{Drop: drop, Cells: []mapload.Cell{{X: 10, Y: 20}}})
	b := startViewCell(mapload.Start{Drop: drop, Cells: []mapload.Cell{{X: 70, Y: 80}}})

	if a == b {
		t.Fatalf("two different placed parties gave one answer %v", a)
	}
	if a == image.Pt(int(drop.X), int(drop.Y)) || b == image.Pt(int(drop.X), int(drop.Y)) {
		t.Errorf("the drop cell was preferred to the placed party: %v, %v", a, b)
	}
}
