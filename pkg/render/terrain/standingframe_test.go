package terrain

import "testing"

func TestStandingFramesKeepAllSixteenFacings(t *testing.T) {
	for _, stored := range []int{16, 9} {
		for facing := 0; facing < 16; facing++ {
			want, mirrored := facing, false
			if stored == 9 && facing > 8 {
				want, mirrored = 16-facing, true
			}
			frame, mirror := SelectStandingFrame(UnitAnim{S: stored}, stored, facing)
			if frame != want || mirror != mirrored {
				t.Errorf("stored %d facing %d: frame/mirror %d/%v, want %d/%v", stored,
					facing, frame, mirror, want, mirrored)
			}
		}
	}
}

func TestStandingFramesRefuseMissingSheetEntries(t *testing.T) {
	for _, tc := range []struct {
		stored, count, facing int
	}{{16, 0, 0}, {16, 16, -1}, {16, 16, 16}, {16, 7, 7}, {9, 7, 9}, {9, 9, 17}} {
		if frame, mirror := SelectStandingFrame(UnitAnim{S: tc.stored}, tc.count, tc.facing); frame != 0 || mirror {
			t.Errorf("stored/count/facing %d/%d/%d: frame/mirror %d/%v, want 0/false", tc.stored,
				tc.count, tc.facing, frame, mirror)
		}
	}
}
