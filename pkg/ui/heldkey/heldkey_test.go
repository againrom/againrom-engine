package heldkey

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// TestHeldEndsAtRelease pins the hold a modifier variant had before
// Ebitengine kept it pressed through its release tick.
func TestHeldEndsAtRelease(t *testing.T) {
	cases := []struct {
		pressed, justPressed, justReleased, want bool
	}{
		{false, false, false, false},
		{true, false, false, true},
		{true, true, false, true},
		{true, false, true, false},
		{true, true, true, true},
	}
	for _, c := range cases {
		if got := held(c.pressed, c.justPressed, c.justReleased); got != c.want {
			t.Errorf("held(%v, %v, %v) = %v, want %v", c.pressed, c.justPressed, c.justReleased, got, c.want)
		}
	}
}

func TestModifierKeys(t *testing.T) {
	for _, k := range []ebiten.Key{ebiten.KeyAlt, ebiten.KeyControl, ebiten.KeyMeta, ebiten.KeyShift} {
		left, right, ok := variants(k)
		if !ok || !isModifier(left) || !isModifier(right) || isModifier(k) {
			t.Errorf("key %v: variants %v %v %v", k, left, right, ok)
		}
	}
	for _, k := range []ebiten.Key{ebiten.KeyA, ebiten.KeyEnter, ebiten.KeyCapsLock, ebiten.KeySpace} {
		if isModifier(k) || isCombined(k) {
			t.Errorf("key %v classed as a modifier", k)
		}
	}
}
