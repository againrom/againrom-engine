package sim

import "testing"

func litCount(stamp []byte) int {
	n := 0
	for _, v := range stamp {
		if v != 0 {
			n++
		}
	}
	return n
}

// The fog reader lights bodies below decay stage 3, never a zero scan range.
func TestLocalSightFollowsDecayStageAndScanRange(t *testing.T) {
	t.Parallel()
	const flatDisc = 145
	for _, tc := range []struct {
		name string
		e    Entity
		want int
	}{
		{"living", Entity{ID: 1, X: 22, Y: 22, Owner: 1, ScanRange: 6, HP: 10, MaxHP: 10}, flatDisc},
		{"downed at zero health", Entity{ID: 1, X: 22, Y: 22, Owner: 1, ScanRange: 6, HP: 0, MaxHP: 10, Decay: DecayFallen}, flatDisc},
		{"fallen body", Entity{ID: 1, X: 22, Y: 22, Owner: 1, ScanRange: 6, HP: -5, MaxHP: 10, Decay: DecayFallen}, flatDisc},
		{"second stage body", Entity{ID: 1, X: 22, Y: 22, Owner: 1, ScanRange: 6, HP: -12, MaxHP: 10, Decay: DecayBones}, flatDisc},
		{"third stage body", Entity{ID: 1, X: 22, Y: 22, Owner: 1, ScanRange: 6, HP: -25, MaxHP: 10, Decay: 3}, 0},
		{"fourth stage body", Entity{ID: 1, X: 22, Y: 22, Owner: 1, ScanRange: 6, HP: -45, MaxHP: 10, Decay: 4}, 0},
		{"living with no scan range", Entity{ID: 1, X: 22, Y: 22, Owner: 1, ScanRange: 0, HP: 10, MaxHP: 10}, 0},
		{"fallen body with no scan range", Entity{ID: 1, X: 22, Y: 22, Owner: 1, ScanRange: 0, HP: -5, MaxHP: 10, Decay: DecayFallen}, 0},
		{"another owner's living unit", Entity{ID: 1, X: 22, Y: 22, Owner: 2, ScanRange: 6, HP: 10, MaxHP: 10}, 0},
	} {
		w, err := NewTerrainWorld(1, sightBounds, ModeCanonical, Terrain{}, []Entity{tc.e}, nil)
		if err != nil {
			t.Fatalf("%s: NewTerrainWorld: %v", tc.name, err)
		}
		if got := litCount(w.Sight(1)); got != tc.want {
			t.Errorf("%s: Sight(1) lights %d cells, want %d", tc.name, got, tc.want)
		}
	}
	body := Entity{ID: 1, X: 22, Y: 22, Owner: 1, ScanRange: 6, HP: -5, MaxHP: 10, Decay: DecayFallen}
	w, err := NewTerrainWorld(1, sightBounds, ModeCanonical, Terrain{}, []Entity{body}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := litCount(w.actorSight(0)); got != 0 {
		t.Errorf("actorSight of a body lights %d cells, want 0", got)
	}
}
