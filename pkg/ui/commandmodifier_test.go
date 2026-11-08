package ui

import (
	"image"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func held(keys ...ebiten.Key) func(ebiten.Key) bool {
	return func(k ebiten.Key) bool { return slices.Contains(keys, k) }
}

// On macOS Command+digit assigns a squad exactly as Control+digit does; on any
// other platform Command does nothing and Control still assigns.
func TestCommandDigitAssignsSquadOnDarwinOnly(t *testing.T) {
	cases := []struct {
		goos string
		keys []ebiten.Key
		want bool
	}{
		{"darwin", []ebiten.Key{ebiten.KeyMetaLeft}, true},
		{"darwin", []ebiten.Key{ebiten.KeyMetaRight}, true},
		{"darwin", []ebiten.Key{ebiten.KeyControlLeft}, true},
		{"darwin", nil, false},
		{"windows", []ebiten.Key{ebiten.KeyMetaLeft}, false},
		{"linux", []ebiten.Key{ebiten.KeyMetaRight}, false},
		{"windows", []ebiten.Key{ebiten.KeyControlRight}, true},
		{"linux", []ebiten.Key{ebiten.KeyControlLeft}, true},
	}
	for _, c := range cases {
		for _, digit := range []int{1, 2} {
			_, v := mkOnMap(t)
			v.localOwner = 1
			v.SetEntities([]MapEntity{{ID: 4, Cell: image.Pt(3, 3), Life: LifeAlive, Owner: 1}})
			v.sel = selection{4}
			ctrl := commandModifierHeld(c.goos, held(c.keys...))
			if ctrl != c.want {
				t.Fatalf("%s %v: modifier = %v, want %v", c.goos, c.keys, ctrl, c.want)
			}
			v.groupKey(digit, ctrl, false, false)
			if got := slices.Equal(v.groups[digit], selection{4}); got != c.want {
				t.Errorf("%s %v digit %d: assigned = %v, want %v", c.goos, c.keys, digit, got, c.want)
			}
		}
	}
}
