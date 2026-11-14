package ui

import (
	"image"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// onKeysPage opens the mission menu's Keys page through its row.
func onKeysPage(t *testing.T, profile string) (*App, *[]PlayerMissionKeys) {
	t.Helper()
	a, written := onMapWithKeys(t, profile)
	a.flow.menuFont = gameMenuTestFont()
	a.flow.openGameMenu(ScreenMap)
	if err := a.HeadlessGameMenuAction("keys"); err != nil {
		t.Fatal(err)
	}
	if a.flow.menuPage != gameMenuKeysPage {
		t.Fatalf("%q: the Keys row opened page %v", profile, a.flow.menuPage)
	}
	return a, written
}

func keysPageRow(t *testing.T, a *App, name string) int {
	t.Helper()
	for i, s := range listedMissionKeys(a.missionKeys) {
		if s.name() == name {
			return i
		}
	}
	t.Fatalf("no listed action %q", name)
	return -1
}

func keyPress(a *App, k ebiten.Key, alt bool) appInput {
	in := a.headlessIdleInput()
	in.AnyKey, in.Keys, in.Viewer.Alt = true, []ebiten.Key{k}, alt
	return in
}

func TestKeysPageRebindsAndMovesATakenKey(t *testing.T) {
	for _, profile := range []string{"", "pickup-p"} {
		a, written := onKeysPage(t, profile)
		if err := a.HeadlessRebindKey("inventory", "K"); err != nil {
			t.Fatal(err)
		}
		if got := a.player.keys["inventory"]; got != (MissionKeyChord{Key: ebiten.KeyK}) || len(*written) != 1 {
			t.Fatalf("%q: inventory = %+v, stored %v", profile, got, *written)
		}
		if got := a.keyRowText(keysPageRow(t, a, "inventory"), 400); !strings.HasSuffix(got, " K") {
			t.Fatalf("%q: inventory row %q, want K at its end", profile, got)
		}
		if err := a.HeadlessRebindKey("attack", "K"); err != nil {
			t.Fatal(err)
		}
		none := a.flow.menuWord("keys.none")
		if got := a.keyRowText(keysPageRow(t, a, "inventory"), 400); !strings.HasSuffix(got, none) {
			t.Fatalf("%q: robbed inventory row %q, want %q", profile, got, none)
		}
		if a.flow.screen != ScreenGameMenu || a.flow.menuPage != gameMenuKeysPage {
			t.Fatalf("%q: a capture left the page", profile)
		}
		if err := a.HeadlessGameMenuAction("key-defaults"); err != nil {
			t.Fatal(err)
		}
		if len(a.player.keys) != 0 || len((*written)[len(*written)-1]) != 0 {
			t.Fatalf("%q: Defaults left %v", profile, a.player.keys)
		}
	}
}

func TestKeysPageCaptureCancelsAndRefusesReservedKeys(t *testing.T) {
	a, written := onKeysPage(t, "")
	a.keysList().Select(keysPageRow(t, a, "patrol"))
	if err := a.HeadlessGameMenuAction("key-list"); err != nil {
		t.Fatal(err)
	}
	if a.player.capture != "patrol" || !strings.HasSuffix(a.keyRowText(keysPageRow(t, a, "patrol"), 400), keyWaitMark) || a.flow.msg != a.flow.word("keys.press") {
		t.Fatalf("Enter on the row did not wait for a key: %q, prompt %q", a.player.capture, a.flow.msg)
	}
	for _, k := range []ebiten.Key{ebiten.KeyArrowUp, ebiten.KeyArrowLeft, ebiten.KeyTab, ebiten.KeyPageUp, ebiten.KeyPageDown, ebiten.KeyF1, ebiten.KeyEnter} {
		a.step(keyPress(a, k, false), a.headlessAt())
		if a.player.capture != "patrol" || a.flow.msg != a.flow.word("keys.reserved") {
			t.Fatalf("%v: capture %q message %q, want the refusal", k, a.player.capture, a.flow.msg)
		}
	}
	a.step(keyPress(a, ebiten.KeyQ, true), a.headlessAt())
	if a.player.capture != "patrol" {
		t.Fatal("Alt+Q was captured")
	}
	a.step(keyPress(a, ebiten.KeyShiftLeft, false), a.headlessAt())
	if a.player.capture != "patrol" {
		t.Fatal("a lone Shift ended the capture")
	}
	in := a.headlessIdleInput()
	in.Escape = true
	a.step(in, a.headlessAt())
	if a.player.capture != "" || a.flow.menuPage != gameMenuKeysPage || len(*written) != 0 {
		t.Fatalf("Escape: capture %q page %v stored %v", a.player.capture, a.flow.menuPage, *written)
	}
	a.startKeyCapture()
	in = a.headlessIdleInput()
	in.PrimaryPressed = true
	a.step(in, a.headlessAt())
	if a.player.capture != "" || len(*written) != 0 {
		t.Fatal("a click did not cancel the capture")
	}
}

func TestReboundShortcutKeyMarksTheBook(t *testing.T) {
	for _, profile := range []string{"", "pickup-p"} {
		a, _ := onMapWithKeys(t, profile)
		p := missionKeyProfiles[profile]
		key := map[string]ebiten.Key{"": ebiten.KeyF4, "pickup-p": ebiten.KeyF4}[profile]
		if got, def := missionShortcutMarks(p, a.player.keys), key.String(); got[0] != def {
			t.Fatalf("%q: default mark %q, want %q", profile, got[0], def)
		}
		g := MissionKeyChord{Key: ebiten.KeyG}
		if err := a.bindMissionKey(shortcutSlotName(profile, 0), g); err != nil {
			t.Fatal(err)
		}
		if got := missionShortcutMarks(p, a.player.keys); got[0] != "G" {
			t.Fatalf("%q: mark after G %q", profile, got[0])
		}
		if !pressOnMap(a, MissionKeyChord{Key: ebiten.KeyG}).QuickSpell[0] {
			t.Fatalf("%q: G does not reach shortcut 1", profile)
		}
		if pressOnMap(a, MissionKeyChord{Key: key}).QuickSpell[0] {
			t.Fatalf("%q: the default key still reaches spell 1", profile)
		}
	}
}

func TestKeysPageWaitKeepsThePointerMoving(t *testing.T) {
	for _, profile := range []string{"", "pickup-p"} {
		a, _ := onKeysPage(t, profile)
		a.keysList().Select(keysPageRow(t, a, "inventory"))
		if err := a.HeadlessGameMenuAction("key-list"); err != nil {
			t.Fatal(err)
		}
		if a.player.capture != "inventory" {
			t.Fatalf("%q: no wait", profile)
		}
		v := a.flow.viewer
		var last image.Point
		for i, p := range []image.Point{{100, 120}, {220, 200}, {330, 260}} {
			in := a.headlessIdleInput()
			in.CursorX, in.CursorY, in.Viewer.CursorX, in.Viewer.CursorY = p.X, p.Y, p.X, p.Y
			a.step(in, a.headlessAt())
			if a.player.capture != "inventory" {
				t.Fatalf("%q: a move ended the wait", profile)
			}
			if a.pointer != p || i > 0 && v.winCursorX == last.X && v.winCursorY == last.Y {
				t.Fatalf("%q tick %d: pointer %v, viewer %d,%d", profile, i, a.pointer, v.winCursorX, v.winCursorY)
			}
			if v.winCursorX != p.X || v.winCursorY != p.Y {
				t.Fatalf("%q tick %d: viewer pointer %d,%d, want %v", profile, i, v.winCursorX, v.winCursorY, p)
			}
			last = p
		}
		in := a.headlessIdleInput()
		in.Escape = true
		a.step(in, a.headlessAt())
		if a.player.capture != "" {
			t.Fatalf("%q: Escape did not end the wait", profile)
		}
	}
}
