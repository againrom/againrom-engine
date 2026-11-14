package game

import "testing"

// TestOpensTownUnfollowed pins the one town-win answer the editor, the mod
// check and the test play share: a main mission whose win opens a town, and
// the declared successor that follows one instead.
func TestOpensTownUnfollowed(t *testing.T) {
	plain := Campaign{Main: []int{10, 20}, Offered: []int{20}}
	followed := Campaign{Main: []int{10, 20, 30}, Offered: []int{20}, Auto: map[int]int{10: 20}}
	cases := []struct {
		name string
		c    Campaign
		n    int
		want bool
		next int
	}{
		{"town win, nothing follows", plain, 10, true, -1},
		{"town win, declared successor", followed, 10, false, 20},
		{"no town after the last offer", followed, 20, false, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, next, got := tc.c.opensTown(tc.n); got != tc.want || next != tc.next {
				t.Fatalf("opensTown(%d) = %d, %t, want %d, %t", tc.n, next, got, tc.next, tc.want)
			}
		})
	}
}

// TestRestoreAdvanceAsksTheWinDecision pins the restore of a town that names
// no main mission: each won main mission moves it to the record after names.
func TestRestoreAdvanceAsksTheWinDecision(t *testing.T) {
	c := Campaign{Main: []int{10, 20, 30, 40}, Offered: []int{30, 40}, Auto: map[int]int{10: 20}}
	for _, tc := range []struct {
		won  []int
		want int
	}{
		{nil, 10}, {[]int{10}, 20}, {[]int{10, 20}, 30}, {[]int{10, 20, 30, 40}, 40},
	} {
		if got := testRestoreTown(c, Snapshot{Won: tc.won}).currentMain(); got != tc.want {
			t.Fatalf("restored main after %v = %d, want %d", tc.won, got, tc.want)
		}
	}
}
