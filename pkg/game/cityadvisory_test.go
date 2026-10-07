package game

import "testing"

// TestCityAdvisoryAcceptsNextChapterAndRefusesOtherwise replaces
// TestCityAdvisorySettledBoundary1168 (DIV-1322): SAV has no wire field for
// the advisory mission number and no gameplay path reads it back, so 0, the
// settled current chapter, and the campaign's own next main mission after it
// are all accepted approximations (silently reset, exactly like an ordinary
// Offered-0 LOAD).
func TestCityAdvisoryAcceptsNextChapterAndRefusesOtherwise(t *testing.T) {
	camp := Campaign{Main: []int{10, 20, 30, 40}}
	for _, tc := range []struct {
		name             string
		open             bool
		chapter, offered int
		refuse           bool
	}{
		{"zero", true, 30, 0, false}, {"current", true, 30, 30, false},
		{"next-chapter", true, 30, 40, false}, {"next-chapter-not-open", false, 30, 40, true},
		{"closed", false, 30, 30, true},
		{"terminal", true, 0, 150, true}, {"negative", true, 30, -1, true},
		{"side", true, 30, 31, true}, {"previous", true, 40, 30, true},
		{"arbitrary-future", true, 30, 987, true}, {"unlisted-chapter", true, 25, 35, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := citySaveAdvisory(Snapshot{Open: tc.open, Offered: tc.offered}, camp, tc.chapter)
			if tc.refuse && err == nil {
				t.Fatalf("citySaveAdvisory must refuse offered=%d chapter=%d", tc.offered, tc.chapter)
			}
			if !tc.refuse && err != nil {
				t.Fatalf("citySaveAdvisory must not refuse offered=%d chapter=%d: %v", tc.offered, tc.chapter, err)
			}
		})
	}
}
