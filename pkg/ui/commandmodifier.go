package ui

import (
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/ui/heldkey"
)

// commandModifierHeld is the one reader of the mission command modifier, the
// original's Ctrl. Either Control key holds it on every platform. On macOS
// either Command key holds it as well, so a Mac player presses Command where a
// Windows player presses Control (DIV-4686); elsewhere Command is the Windows
// key and does nothing here. goos and pressed are the seam a test sets.
func commandModifierHeld(goos string, pressed func(ebiten.Key) bool) bool {
	if pressed(ebiten.KeyControlLeft) || pressed(ebiten.KeyControlRight) {
		return true
	}
	return goos == "darwin" && (pressed(ebiten.KeyMetaLeft) || pressed(ebiten.KeyMetaRight))
}

// platformCommandModifier reads the command modifier on the running platform.
func platformCommandModifier() bool {
	return commandModifierHeld(runtime.GOOS, heldkey.Pressed)
}
