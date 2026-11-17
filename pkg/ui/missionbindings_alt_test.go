package ui

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// altSystemKey is the set of keys that act with Alt held (MENU-082): the
// digit row's 1..9, the numpad's 0..9, S and B..Y.
func altSystemKey(k ebiten.Key) bool {
	switch {
	case k >= ebiten.KeyDigit1 && k <= ebiten.KeyDigit9:
		return true
	case k >= ebiten.KeyNumpad0 && k <= ebiten.KeyNumpad9:
		return true
	case k >= ebiten.KeyB && k <= ebiten.KeyY:
		return true
	}
	return false
}

func TestMissionBindingsAltActsOnlyOnTheSystemKeys(t *testing.T) {
	for _, profile := range []string{"", "pickup-p"} {
		rows := missionBindings(missionKeyProfiles[profile])
		for _, b := range rows {
			if !altSystemKey(b.key) && b.alt.admits(true) {
				t.Errorf("profile %q: row %s (action %d) acts with Alt held", profile, b.name, b.action)
			}
		}
		for _, k := range []struct {
			name string
			key  ebiten.Key
			want bool
		}{
			{"alt-f12", ebiten.KeyF12, false},
			{"alt-backspace", ebiten.KeyBackspace, false},
			{"alt-f2", ebiten.KeyF2, false},
			{"alt-0", ebiten.KeyDigit0, false},
			{"alt-1", ebiten.KeyDigit1, true},
			{"alt-numpad-0", ebiten.KeyNumpad0, true},
			{"alt-s", ebiten.KeyS, true},
		} {
			acts := false
			for _, b := range rows {
				if b.key == k.key && b.alt.admits(true) {
					acts = true
				}
			}
			if acts != k.want {
				t.Errorf("profile %q: %s acts = %v, want %v", profile, k.name, acts, k.want)
			}
		}
	}
}
