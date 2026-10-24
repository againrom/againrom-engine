package ui

import (
	"time"

	"againrom/pkg/ui/heldkey"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type editorNativeInput struct {
	pressed, justPressed func(ebiten.Key) bool
	chars                string
	canvas               Input
	keysDown, closing    bool
}

func readEditorInput() editorNativeInput {
	return editorNativeInput{
		pressed: heldkey.Pressed, justPressed: inpututil.IsKeyJustPressed,
		chars: string(ebiten.AppendInputChars(nil)), canvas: readInput(),
		keysDown: len(heldkey.AppendPressed(nil)) > 0, closing: ebiten.IsWindowBeingClosed(),
	}
}

func (e *MapEditor) updateTextInput(in editorNativeInput, now time.Time) (bool, editorInputPolicy) {
	m, policy := e.inputModel()
	if m == nil {
		return false, policy
	}
	ctrl, shift := in.pressed(ebiten.KeyControl) && !in.pressed(ebiten.KeyAlt), in.pressed(ebiten.KeyShift)
	if !ctrl {
		e.TextInput(in.chars)
	}
	held := map[string]bool{}
	for _, k := range []struct {
		key  ebiten.Key
		name string
	}{
		{ebiten.KeyBackspace, "backspace"}, {ebiten.KeyDelete, "delete"},
		{ebiten.KeyArrowLeft, "left"}, {ebiten.KeyArrowRight, "right"},
		{ebiten.KeyArrowUp, "up"}, {ebiten.KeyArrowDown, "down"},
		{ebiten.KeyHome, "home"}, {ebiten.KeyEnd, "end"},
	} {
		held[k.name] = in.pressed(k.key)
	}
	for _, key := range m.repeat(now, held) {
		if policy.panArrows && (key == "left" || key == "right" || key == "up" || key == "down") {
			continue
		}
		_ = e.Key(policy.modifiedKey(key, ctrl, shift))
	}
	if ctrl {
		for _, k := range []struct {
			key  ebiten.Key
			name string
		}{
			{ebiten.KeyA, "selectall"}, {ebiten.KeyC, "copy"}, {ebiten.KeyX, "cut"}, {ebiten.KeyV, "paste"},
		} {
			if in.justPressed(k.key) {
				_ = e.Key(k.name)
			}
		}
	}
	return true, policy
}
