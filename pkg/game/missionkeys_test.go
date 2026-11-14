package game

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestMissionKeysSurviveAProfileReloadPerGame(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	store := OptionsStore{Path: path}
	if err := store.writeAll(map[string]string{"GameSpeed": "4"}); err != nil {
		t.Fatal(err)
	}
	first := ui.PlayerMissionKeys{"inventory": {Key: ebiten.KeyK}, "select-all": {None: true}}
	second := ui.PlayerMissionKeys{"inventory": {Key: ebiten.KeyY, Ctrl: true}, "support": {Key: ebiten.KeyY, Shift: true}}
	if err := store.SetMissionKeys("", first); err != nil {
		t.Fatal(err)
	}
	if err := store.SetMissionKeys("pickup-p", second); err != nil {
		t.Fatal(err)
	}
	reloaded := OptionsStore{Path: path}
	for _, tc := range []struct {
		profile string
		want    ui.PlayerMissionKeys
	}{{"", first}, {"pickup-p", second}} {
		got, err := reloaded.MissionKeys(tc.profile)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%q after reload = %v, %v; want %v", tc.profile, got, err, tc.want)
		}
	}
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"FirstGameKeys.inventory=K", "FirstGameKeys.select-all=none",
		"SecondGameKeys.inventory=Ctrl+Y", "SecondGameKeys.support=Shift+Y", "GameSpeed=4"} {
		if !strings.Contains(string(text), line+"\n") {
			t.Fatalf("profile settings lack %q:\n%s", line, text)
		}
	}
	if err := reloaded.SetMissionKeys("", ui.PlayerMissionKeys{}); err != nil {
		t.Fatal(err)
	}
	if got, _ := reloaded.MissionKeys("pickup-p"); !reflect.DeepEqual(got, second) {
		t.Fatalf("the first game's Defaults changed the second game's keys: %v", got)
	}
	if got, _ := reloaded.MissionKeys(""); len(got) != 0 {
		t.Fatalf("the first game's keys after Defaults = %v", got)
	}
}

func TestMissionKeyRebindLeavesTheSaveUnchanged(t *testing.T) {
	export := func(rebind bool) []byte {
		f := restoredWorldMapFront(t, true, false)
		seed := currentTrainingCity(t)
		f.Table, f.Carried = seed.Table, mapload.CloneParty(seed.Carried)
		f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
		if rebind {
			if err := f.Options.SetMissionKeys("", ui.PlayerMissionKeys{"inventory": {Key: ebiten.KeyK}, "book": {None: true}}); err != nil {
				t.Fatal(err)
			}
		}
		snapshot, label, err := f.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := f.ExportCurrentSave(snapshot, label)
		if err != nil {
			t.Fatal(err)
		}
		return encoded
	}
	if plain, rebound := export(false), export(true); len(plain) == 0 || !bytes.Equal(plain, rebound) {
		t.Fatalf("the SAV written after a rebind differs: %d and %d bytes", len(plain), len(rebound))
	}
	// The mission's SAV after the player rebinds on the Keys page.
	mission := func(rebind bool) []byte {
		f := currentPoolFixtureFront(t, 91, 92)
		f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
		app := f.App("key rebind save")
		if err := app.OpenMission(f.MissionOpener(10)); err != nil {
			t.Fatal(err)
		}
		openMissionGameMenu(t, app)
		if err := app.HeadlessGameMenuAction("keys"); err != nil {
			t.Fatal(err)
		}
		if rebind {
			for _, b := range [][2]string{{"inventory", "K"}, {"attack", "Shift+I"}} {
				if err := app.HeadlessRebindKey(b[0], b[1]); err != nil {
					t.Fatal(err)
				}
			}
			if got, _ := f.Options.MissionKeys(""); got["inventory"] != (ui.MissionKeyChord{Key: ebiten.KeyK}) {
				t.Fatalf("the profile holds %v after the rebind", got)
			}
		}
		for _, name := range []string{"page-return", "return"} {
			if err := app.HeadlessGameMenuAction(name); err != nil {
				t.Fatal(name, err)
			}
		}
		s, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		encoded, _, err := f.playerMissionSave(s, "key rebind")
		if err != nil {
			t.Fatal(err)
		}
		return encoded
	}
	if plain, rebound := mission(false), mission(true); len(plain) == 0 || !bytes.Equal(plain, rebound) {
		t.Fatalf("the mission SAV written after a Keys page rebind differs: %d and %d bytes", len(plain), len(rebound))
	}
}
