package ui

import (
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestPassingKeepsOnlyTheNamedGroups(t *testing.T) {
	in := appInput{
		AnyKey: true, Keys: []ebiten.Key{ebiten.KeyA}, Close: true, Escape: true, Enter: true, Typed: "x",
		CursorX: 3, CursorY: 4, PrimaryPressed: true, SecondaryReleased: true, Help: true, AltLetter: 'a', Unfocused: true,
		Viewer: Input{CursorX: 5, CursorY: 6, Ctrl: true, Shift: true, Alt: true, Unfocused: true},
	}
	for _, c := range []struct {
		name   string
		groups inputGroup
		want   appInput
	}{
		{"chat", passCheatChat, appInput{CursorX: 3, CursorY: 4, Help: true, AltLetter: 'a', Unfocused: true,
			Viewer: Input{CursorX: 5, CursorY: 6, Alt: true, Unfocused: true}}},
		{"keys", passKeysCapture, appInput{AnyKey: true, Keys: in.Keys, Close: true, Escape: true, CursorX: 3, CursorY: 4,
			PrimaryPressed: true, SecondaryReleased: true, Unfocused: true,
			Viewer: Input{CursorX: 5, CursorY: 6, Ctrl: true, Shift: true, Alt: true, Unfocused: true}}},
	} {
		if got := in.passing(c.groups); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %+v want %+v", c.name, got, c.want)
		}
	}
}

// The gold modal passes only pointer, focus and Close; focus is the top-level field.
func TestGoldModalPassesOnlyPointerFocusAndClose(t *testing.T) {
	a, v, _ := openPurseEditor(t)
	in := appInput{
		AnyKey: true, Keys: []ebiten.Key{ebiten.KeyA}, Close: true, Escape: true, Enter: true, Typed: "x",
		CursorX: 3, CursorY: 4, PrimaryPressed: true, Help: true, Unfocused: true,
		Viewer: Input{CursorX: 5, CursorY: 6, Ctrl: true, Alt: true, Unfocused: false},
	}
	want := appInput{CursorX: 3, CursorY: 4, Close: true, Unfocused: true,
		Viewer: Input{CursorX: 5, CursorY: 6, Unfocused: true}}
	if got := a.stepGoldModal(v, in); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v want %+v", got, want)
	}
}
