package ui

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func chord(k ebiten.Key, mods string) MissionKeyChord {
	c := MissionKeyChord{Key: k}
	for _, m := range mods {
		switch m {
		case 'c':
			c.Ctrl = true
		case 's':
			c.Shift = true
		case 'a':
			c.Alt = true
		}
	}
	return c
}

func bindAll(t *testing.T, a *App, name string, c MissionKeyChord) {
	t.Helper()
	if err := a.bindMissionKey(name, c); err != nil {
		t.Fatalf("bind %s to %v: %v", name, c, err)
	}
}

func TestReboundMissionKeyFiresOnExactlyThePressedChord(t *testing.T) {
	a, _ := onMapWithKeys(t, "")
	bindAll(t, a, "autocast", chord(ebiten.KeyW, ""))
	if in := pressOnMap(a, chord(ebiten.KeyW, "")); !in.Autocast || in.WeaponSwap || in.Retreat {
		t.Fatalf("W: autocast %v swap %v retreat %v", in.Autocast, in.WeaponSwap, in.Retreat)
	}
	if in := pressOnMap(a, chord(ebiten.KeyW, "c")); in.Autocast || !in.Retreat {
		t.Fatalf("Ctrl+W: autocast %v retreat %v, want false true", in.Autocast, in.Retreat)
	}

	a, _ = onMapWithKeys(t, "")
	bindAll(t, a, "save-game", chord(ebiten.KeyK, ""))
	if in := pressOnMap(a, chord(ebiten.KeyK, "")); !in.SaveGame {
		t.Fatal("K does not save the game")
	}
	if in := pressOnMap(a, chord(ebiten.KeyK, "c")); in.SaveGame {
		t.Fatal("Ctrl+K saves the game")
	}

	a, _ = onMapWithKeys(t, "")
	bindAll(t, a, "inventory", chord(ebiten.KeyA, ""))
	if in := pressOnMap(a, chord(ebiten.KeyA, "c")); in.Inventory || !in.Autocast {
		t.Fatalf("Ctrl+A: inventory %v autocast %v, want false true", in.Inventory, in.Autocast)
	}
	if in := pressOnMap(a, chord(ebiten.KeyA, "")); !in.Inventory || in.Attack {
		t.Fatalf("A: inventory %v attack %v, want true false", in.Inventory, in.Attack)
	}
	bindAll(t, a, "attack", chord(ebiten.KeyB, ""))
	bindAll(t, a, "book", chord(ebiten.KeyA, ""))
	if in := pressOnMap(a, chord(ebiten.KeyA, "c")); in.Book || !in.Autocast {
		t.Fatalf("Ctrl+A: book %v autocast %v, want false true", in.Book, in.Autocast)
	}
	if in := pressOnMap(a, chord(ebiten.KeyA, "")); !in.Book || in.Inventory {
		t.Fatalf("A: book %v inventory %v, want true false", in.Book, in.Inventory)
	}
}

func TestReboundSpellSlotNeverReachesADebugChord(t *testing.T) {
	for _, tc := range []struct {
		key   ebiten.Key
		debug func(appInput) bool
	}{
		{ebiten.KeyF10, func(in appInput) bool { return in.Grid }},
		{ebiten.KeyF4, func(in appInput) bool { return in.Reveal }},
	} {
		a, _ := onMapWithKeys(t, "")
		bindAll(t, a, shortcutSlotName("", 0), chord(tc.key, ""))
		in := pressOnMap(a, chord(tc.key, "cs"))
		if in.QuickSpell[0] || !tc.debug(in) {
			t.Fatalf("Ctrl+Shift+%v: spell %v debug %v, want false true", tc.key, in.QuickSpell[0], tc.debug(in))
		}
		for _, c := range []MissionKeyChord{chord(tc.key, "")} {
			if in := pressOnMap(a, c); !in.QuickSpell[0] || in.Grid {
				t.Fatalf("%v: spell %v grid %v", c, in.QuickSpell[0], in.Grid)
			}
		}
		if tc.key == ebiten.KeyF4 && !pressOnMap(a, chord(ebiten.KeyF2, "")).QuickSave {
			t.Fatal("F2 no longer quick saves")
		}
	}
}

func TestBindTakesOnlyTheChordOfATwoKeyAction(t *testing.T) {
	a, _ := onMapWithKeys(t, "")
	bindAll(t, a, "attack", chord(ebiten.KeyF2, ""))
	if in := pressOnMap(a, chord(ebiten.KeyF2, "")); !in.Attack || in.QuickSave {
		t.Fatalf("F2: attack %v quick save %v", in.Attack, in.QuickSave)
	}
	if pressOnMap(a, chord(ebiten.KeyF4, "")).QuickSave {
		t.Fatal("F4 still quick saves")
	}
	if got := missionKeyCaptions(a.missionKeys, a.player.keys, missionKeySlot{action: missionKeyQuickSave}); got != "" {
		t.Fatalf("quick save row reads %q, want unassigned", got)
	}

	a, _ = onMapWithKeys(t, "")
	bindAll(t, a, "attack", chord(ebiten.KeyB, ""))
	if in := pressOnMap(a, chord(ebiten.KeyQ, "")); !in.Book {
		t.Fatal("Q no longer opens the spellbook")
	}
	if in := pressOnMap(a, chord(ebiten.KeyB, "")); in.Book || !in.Attack {
		t.Fatalf("B: book %v attack %v", in.Book, in.Attack)
	}
	bindAll(t, a, "attack", chord(ebiten.KeyQ, ""))
	if got := missionKeyCaptions(a.missionKeys, a.player.keys, missionKeySlot{action: missionKeyBook}); got != "" {
		t.Fatalf("book row reads %q once both keys are gone", got)
	}
}

func TestProfileReadNeverLetsOneChordFireTwoActions(t *testing.T) {
	settings := map[string]string{
		"FirstGameKeys.attack":    "K",
		"FirstGameKeys.inventory": "K",
		"FirstGameKeys.spell-9":   "F4",
		"FirstGameKeys.book":      "Ctrl+Shift+F10",
		"FirstGameKeys.worn":      "Ctrl+Shift+K",
	}
	a, _ := onMapWithKeys(t, "")
	a.SetPlayerMissionKeys(PlayerMissionKeysFrom("", settings), nil)
	if in := pressOnMap(a, chord(ebiten.KeyK, "")); !in.Attack || in.Inventory {
		t.Fatalf("K: attack %v inventory %v, want true false", in.Attack, in.Inventory)
	}
	if in := pressOnMap(a, chord(ebiten.KeyF4, "")); !in.QuickSpell[0] || in.QuickSave {
		t.Fatalf("F4: spell %v quick save %v, want true false", in.QuickSpell[0], in.QuickSave)
	}
	if in := pressOnMap(a, chord(ebiten.KeyF10, "cs")); !in.Grid || in.Book {
		t.Fatalf("Ctrl+Shift+F10: grid %v book %v, want true false", in.Grid, in.Book)
	}
	if in := pressOnMap(a, chord(ebiten.KeyK, "cs")); !in.Kill || in.Worn {
		t.Fatalf("Ctrl+Shift+K: kill %v worn %v, want true false", in.Kill, in.Worn)
	}
	if !pressOnMap(a, chord(ebiten.KeyI, "")).Inventory {
		t.Fatal("the entry that lost K took the inventory's own keys")
	}
}

func TestScriptedKeyReadsTheGamesTableNotThePlayersLayout(t *testing.T) {
	a, _ := onMapWithKeys(t, "")
	bindAll(t, a, shortcutSlotName("", 0), chord(ebiten.KeyK, ""))
	in := a.headlessIdleInput()
	if !in.missionKeyNamed(a.activeMissionKeys(), "f4") || !in.QuickSpell[0] {
		t.Fatal("the scenario key f4 does not cast spell 1 under a player layout")
	}
	if !pressOnMap(a, chord(ebiten.KeyK, "")).QuickSpell[0] || pressOnMap(a, chord(ebiten.KeyF5, "")).QuickSpell[0] {
		t.Fatal("the player's own press no longer reads the layout")
	}
}

func TestWaitingKeysRowShowsItsWholeName(t *testing.T) {
	for _, profile := range []string{"", "pickup-p"} {
		a, _ := onKeysPage(t, profile)
		width := keysListBox(a.flow.menuFont).Rect.Dx() - 24
		for i, s := range listedMissionKeys(a.missionKeys) {
			a.player.capture = s.name()
			got := a.keyRowText(i, width)
			if !strings.HasPrefix(got, keyActionCaption(a.flow, s)+" ") || !strings.HasSuffix(got, keyWaitMark) {
				t.Fatalf("%q: waiting row %q cuts the name %q", profile, got, keyActionCaption(a.flow, s))
			}
		}
	}
}

func TestWaitingKeysRowEatsItsChord(t *testing.T) {
	a, written := onKeysPage(t, "")
	a.keysList().Select(keysPageRow(t, a, "attack"))
	a.startKeyCapture()
	before := a.FrameSmoothing()
	in := a.headlessIdleInput()
	in.AnyKey, in.Keys = true, []ebiten.Key{ebiten.KeyO}
	in.Viewer.Ctrl = true
	in.readMissionKeys(a.activeMissionKeys(), onlyKey(ebiten.KeyO), true)
	if !in.Smoothing {
		t.Fatal("the probe input does not carry Ctrl+O as the smoothing toggle")
	}
	a.step(in, a.headlessAt())
	if a.FrameSmoothing() != before {
		t.Fatal("Ctrl+O toggled text smoothing while a row waited for its key")
	}
	if got := a.player.keys["attack"]; got != chord(ebiten.KeyO, "c") || len(*written) != 1 {
		t.Fatalf("attack = %+v, stored %v", got, *written)
	}
}
