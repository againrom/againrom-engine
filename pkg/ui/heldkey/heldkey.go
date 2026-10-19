// Package heldkey reads held keyboard keys with a modifier key's hold ending
// in the tick it is released.
//
// Ebitengine reports a modifier key (Alt, Control, Shift, Meta and their left
// and right variants) as pressed through the tick it is released. The game
// and the launcher read a modifier as released in that tick unless it was
// also pressed in it, so a chord ends with its modifier's release.
package heldkey

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Pressed reports whether key is held. A combined modifier such as
// ebiten.KeyShift is held when either of its variants is.
func Pressed(key ebiten.Key) bool {
	if left, right, ok := variants(key); ok {
		return Pressed(left) || Pressed(right)
	}
	if !ebiten.IsKeyPressed(key) {
		return false
	}
	if !isModifier(key) {
		return true
	}
	return held(true, inpututil.IsKeyJustPressed(key), inpututil.IsKeyJustReleased(key))
}

// AppendPressed appends the held keys to keys, as inpututil.AppendPressedKeys
// does, without a modifier released in the current tick.
func AppendPressed(keys []ebiten.Key) []ebiten.Key {
	start := len(keys)
	keys = inpututil.AppendPressedKeys(keys)
	out := keys[:start]
	for _, k := range keys[start:] {
		if !isModifier(k) && !isCombined(k) || Pressed(k) {
			out = append(out, k)
		}
	}
	return out
}

// held reports a modifier variant's hold from the engine's pressed state and
// the tick's press and release edges: a release in the tick ends the hold
// unless the key was also pressed in that tick.
func held(pressed, justPressed, justReleased bool) bool {
	return pressed && (!justReleased || justPressed)
}

func isModifier(key ebiten.Key) bool {
	switch key {
	case ebiten.KeyAltLeft, ebiten.KeyAltRight,
		ebiten.KeyControlLeft, ebiten.KeyControlRight,
		ebiten.KeyMetaLeft, ebiten.KeyMetaRight,
		ebiten.KeyShiftLeft, ebiten.KeyShiftRight:
		return true
	}
	return false
}

func isCombined(key ebiten.Key) bool {
	_, _, ok := variants(key)
	return ok
}

func variants(key ebiten.Key) (left, right ebiten.Key, ok bool) {
	switch key {
	case ebiten.KeyAlt:
		return ebiten.KeyAltLeft, ebiten.KeyAltRight, true
	case ebiten.KeyControl:
		return ebiten.KeyControlLeft, ebiten.KeyControlRight, true
	case ebiten.KeyMeta:
		return ebiten.KeyMetaLeft, ebiten.KeyMetaRight, true
	case ebiten.KeyShift:
		return ebiten.KeyShiftLeft, ebiten.KeyShiftRight, true
	}
	return 0, 0, false
}
