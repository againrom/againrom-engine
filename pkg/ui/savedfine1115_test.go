package ui

import (
	"image"
	"testing"
)

func TestSavedFine1115PositionDoesNotDriftWithPresentationClock(t *testing.T) {
	for _, tc := range []struct {
		x, y uint8
		want image.Point
	}{{128, 128, image.Pt(0, 0)}, {224, 128, image.Pt(12, 0)}, {0, 128, image.Pt(-16, 0)},
		{32, 128, image.Pt(-12, 0)}, {64, 128, image.Pt(-8, 0)}, {96, 128, image.Pt(-4, 0)},
		{255, 1, image.Pt(15, -15)}, {128, 0, image.Pt(0, -16)}} {
		for _, period := range []int{0, 1, shiftPeriod} {
			for _, phase := range []int{-1, 0, 1, shiftPeriod / 2, shiftPeriod, shiftPeriod * 2} {
				v := &Viewer{phaseUS: phase, phasePeriodUS: period}
				e := MapEntity{FinePosition: true, FineX: tc.x, FineY: tc.y,
					Cell: image.Pt(16, 16), Step: image.Pt(-1, 1), Transit: 37, TransitSpan: 99}
				if got := v.entityShift(e); got != tc.want {
					t.Fatalf("fine %d,%d clock %d/%d: %v want %v", tc.x, tc.y, phase, period, got, tc.want)
				}
			}
		}
	}
}
