package game

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/ui"
)

type releaseMenuPlayer struct {
	settings []audio.Settings
}

func (*releaseMenuPlayer) Play(audio.Sample, audio.Placement) {}
func (*releaseMenuPlayer) RequestSample(audio.Sample, audio.Request) audio.Voice {
	return nil
}
func (p *releaseMenuPlayer) SetSettings(s audio.Settings) {
	p.settings = append(p.settings, s)
}

func drawnMenuLabel(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '~' {
			b.WriteByte(s[i])
			continue
		}
		if i+1 < len(s) && s[i+1] == '~' {
			b.WriteByte('~')
			i++
		}
	}
	return b.String()
}

func requireReleaseMenuRows(t *testing.T, app *ui.App, want []string, enabled []bool) {
	t.Helper()
	rows := app.HeadlessRows()
	if len(rows) != len(want) {
		t.Fatalf("menu rows = %d, want %d: %+v", len(rows), len(want), rows)
	}
	for i := range want {
		if rows[i].Text != want[i] || rows[i].Choosable != enabled[i] {
			t.Errorf("row %d = (% x,%v), want (% x,%v)", i, []byte(rows[i].Text), rows[i].Choosable,
				[]byte(want[i]), enabled[i])
		}
	}
}

// releaseContinuedCampaign opens the production success panel with words from
// the active lawful install, takes Continue through App's real Escape dispatch,
// and stops on the phase-2 End Quest confirmation. The simulation fixture is
// already covered by the story's one-shot completion witness; this helper adds
// the installed labels and production input/menu path that fixture cannot see.
func releaseContinuedCampaign(t *testing.T, f *FrontEnd) *ui.App {
	t.Helper()
	_, mw, v := victoryContinueDriver(t, 10)
	v.SetGameMenuContext(func() ui.GameMenuContext {
		return gameMenuContext(mw, true, "")
	})
	a := f.App("1057-continued-release")
	if err := a.OpenMission(func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence,
		ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
		return v, mw.paced, mw.enqueue, mw.setCadenceMode, mw.affect, mw.advanceNotice,
			mw.attackOrCast, mw.grab, mw.stance, mw.march, nil
	}); err != nil {
		t.Fatalf("open completed campaign: %v", err)
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatalf("take Continue: %v", err)
	}
	if v.NoticeOpen() {
		t.Fatal("Continue left the success panel open")
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatalf("open completed campaign menu: %v", err)
	}
	if err := a.HeadlessGameMenuAction("end"); err != nil {
		t.Fatalf("open completed End Quest confirmation: %v", err)
	}
	return a
}

func TestReleasePauseMenuVisitsEveryDestination(t *testing.T) {
	f := releaseFront(t)
	root := filepath.Base(strings.TrimRight(strings.ReplaceAll(
		strings.TrimSpace(os.Getenv("AGAINROM_ASSETS")), "\\", "/"), "/"))
	player := &releaseMenuPlayer{}
	f.SoundPlayer = player
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	f.LoadOptions()
	list := func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "witness.ags", Label: "witness town"}} }
	loadTown := func(string) (ui.MapOpener, bool, error) {
		f.Town = NewTown(f.Campaign.Value())
		f.Town.Arrive()
		f.TownScreen().(*townScreen).resetForNewGame()
		return nil, true, nil
	}

	app := f.App("1042-release")
	app.SetSaveSeams(func(bool) (string, error) { return "witness.ags", nil }, list, loadTown)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatalf("open mission 10: %v", err)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	w := f.Words
	requireReleaseMenuRows(t, app, []string{
		drawnMenuLabel(w.MenuSave), drawnMenuLabel(w.MenuLoad), drawnMenuLabel(w.MenuGameOptions),
		drawnMenuLabel(w.MenuSoundOptions), drawnMenuLabel(w.MenuQuestObjectives),
		drawnMenuLabel(w.MenuEndQuest), drawnMenuLabel(w.MenuReturn),
	}, []bool{true, true, true, true, true, true, true})
	t.Logf("%s campaign menu labels: % x", root, []byte(strings.Join([]string{
		w.MenuSave, w.MenuLoad, w.MenuGameOptions, w.MenuSoundOptions,
		w.MenuQuestObjectives, w.MenuEndQuest, w.MenuReturn,
	}, "|")))
	before := f.live.world.Hash()
	for _, action := range []string{"game-options", "sound-options", "objectives"} {
		if err := app.HeadlessGameMenuAction(action); err != nil {
			t.Fatalf("open %s: %v", action, err)
		}
		if len(app.HeadlessRows()) < 2 {
			t.Fatalf("%s destination has no content and return", action)
		}
		if action == "game-options" {
			if err := app.HeadlessGameMenuAction("toggle-tips"); err != nil {
				t.Fatal(err)
			}
			if on, _ := f.Options.TipsMode(); !on {
				t.Fatal("Game Options changed Tips Mode before OK")
			}
			for _, step := range []string{"page-return", "game-options"} {
				if err := app.HeadlessGameMenuAction(step); err != nil {
					t.Fatal(err)
				}
			}
			if on, _ := f.Options.TipsMode(); on {
				t.Fatal("Game Options did not persist Tips Mode off")
			}
			releaseMenuSpeedWitness(t, f, app)
			releaseTooltipMenuWitness(t, f, app)
		}
		if action == "sound-options" {
			if err := app.HeadlessGameMenuAction("volume-down"); err != nil {
				t.Fatal(err)
			}
			if f.Sound.Volume != 75 || len(player.settings) == 0 {
				t.Fatalf("Sound Options produced volume %d, device settings %v", f.Sound.Volume, player.settings)
			}
			if err := app.HeadlessGameMenuAction("toggle-sound"); err != nil {
				t.Fatal(err)
			}
			stored, err := (OptionsStore{Path: f.Options.Path}).SoundOptions(SoundOptions{Enabled: true, Volume: 100})
			if err != nil || stored.Enabled || stored.Volume != 75 {
				t.Fatal("Sound Options did not persist OFF/75", stored, err)
			}
			// Match the command's startup ordering: resolve preferences before
			// creating devices. CLI presence/precedence has separate cmd tests.
			prior := soundOptions
			SetSoundOptions(stored)
			cold, err := NewFrontEnd(os.Getenv("AGAINROM_ASSETS"))
			SetSoundOptions(prior)
			if err != nil {
				t.Fatal("fresh sound startup", err)
			}
			if cold.Sound != stored {
				t.Fatal("fresh front end lost sound preferences", cold.Sound, stored)
			}
			t.Log("Sound Options pointer/action path: OFF/75 persisted, fresh installed front end initialized with OFF/75")
		}
		if err := app.HeadlessGameMenuAction("page-return"); err != nil {
			t.Fatalf("return from %s: %v", action, err)
		}
	}
	if after := f.live.world.Hash(); after != before {
		t.Fatalf("nested campaign pages moved world hash: %#x -> %#x", before, after)
	}
	if err := app.HeadlessGameMenuAction("end"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenGameMenu {
		t.Fatal("End Quest did not raise confirmation")
	}
	requireReleaseMenuRows(t, app, []string{
		drawnMenuLabel(w.MenuVictory), drawnMenuLabel(w.MenuExitMain),
		drawnMenuLabel(w.MenuExitWindows), drawnMenuLabel(w.MenuReturn),
	}, []bool{false, true, true, true})
	t.Logf("%s campaign completion labels: % x", root, []byte(strings.Join([]string{
		w.MenuVictory, w.OutcomeContinue, w.MenuExitMain, w.MenuExitWindows, w.MenuReturn,
	}, "|")))
	if err := app.HeadlessGameMenuAction("victory"); err == nil || !strings.Contains(err.Error(), "not choosable") {
		t.Fatalf("Victory before Continue = %v", err)
	}
	if err := app.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("end"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("exit-main"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMenu {
		t.Fatalf("campaign Exit to Main reached %s", app.Screen())
	}

	if err := app.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("witness town"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenTown || !f.TownScreen().(*townScreen).CanSave() {
		t.Fatal("town fixture did not establish a resolved city save point", app.Screen())
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	requireReleaseMenuRows(t, app, []string{
		drawnMenuLabel(w.MenuSave), drawnMenuLabel(w.MenuLoad), drawnMenuLabel(w.MenuGameOptions), drawnMenuLabel(w.MenuSoundOptions),
		drawnMenuLabel(w.MenuAbort), drawnMenuLabel(w.MenuReturn),
	}, []bool{true, true, true, true, true, true})
	if err := app.HeadlessGameMenuAction("sound-options"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("abort"); err != nil {
		t.Fatal(err)
	}
	requireReleaseMenuRows(t, app, []string{
		drawnMenuLabel(w.MenuExitMain), drawnMenuLabel(w.MenuExitWindows), drawnMenuLabel(w.MenuReturn),
	}, []bool{true, true, true})
	t.Logf("%s town confirmation labels: % x", root, []byte(strings.Join([]string{
		w.MenuExitMain, w.MenuExitWindows, w.MenuReturn,
	}, "|")))
	if err := app.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("abort"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("confirm-abort"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMenu {
		t.Fatalf("confirmed Abort Game reached %s", app.Screen())
	}

	standalone := -1
	for i, entry := range f.Maps {
		if entry.Mission == 0 && entry.Choosable() {
			standalone = i
			break
		}
	}
	if standalone < 0 {
		t.Fatal("lawful install has no standalone map row")
	}
	plain := f.App("1042-standalone-release")
	plain.SetSaveSeams(func(bool) (string, error) { return "witness.ags", nil }, list, loadTown)
	if err := plain.OpenMission(func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
		return f.loadMap(standalone)
	}); err != nil {
		t.Fatalf("open standalone row %d: %v", standalone, err)
	}
	if err := plain.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	requireReleaseMenuRows(t, plain, []string{
		drawnMenuLabel(w.MenuSave), drawnMenuLabel(w.MenuDiplomacy), drawnMenuLabel(w.MenuGameOptions),
		drawnMenuLabel(w.MenuSoundOptions), drawnMenuLabel(w.MenuQuestObjectives),
		drawnMenuLabel(w.MenuEndQuest), drawnMenuLabel(w.MenuReturn),
	}, []bool{false, true, true, true, false, true, true})
	plainHash := f.live.world.Hash()
	if err := plain.HeadlessGameMenuAction("diplomacy"); err != nil {
		t.Fatal(err)
	}
	if len(plain.HeadlessRows()) < 3 {
		t.Fatalf("Diplomacy destination has %d rows", len(plain.HeadlessRows()))
	}
	if err := plain.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	if after := f.live.world.Hash(); after != plainHash {
		t.Fatalf("Diplomacy moved standalone hash: %#x -> %#x", plainHash, after)
	}
}

// TestReleasePauseMenuCoversFullActionPopulation closes the real-install
// witness population: every root action, nested action and terminal
// destination is driven through the production controller. The release gate
// runs this once for each lawful install.
func TestReleasePauseMenuCoversFullActionPopulation(t *testing.T) {
	f := releaseFront(t)
	f.SoundPlayer = &releaseMenuPlayer{}
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	f.LoadOptions()
	list := func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "witness.ags", Label: "witness town"}} }
	loadTown := func(string) (ui.MapOpener, bool, error) {
		f.Town = NewTown(f.Campaign.Value())
		f.Town.Arrive()
		f.TownScreen().(*townScreen).resetForNewGame()
		return nil, true, nil
	}
	serial := 0
	newApp := func() *ui.App {
		serial++
		a := f.App(fmt.Sprintf("1042-population-%d", serial))
		a.SetSaveSeams(func(bool) (string, error) { return "witness.ags", nil }, list, loadTown)
		return a
	}
	campaign := func(t *testing.T) *ui.App {
		t.Helper()
		a := newApp()
		if err := a.OpenMission(f.MissionOpener(10)); err != nil {
			t.Fatalf("open campaign: %v", err)
		}
		if err := a.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		return a
	}
	town := func(t *testing.T) *ui.App {
		t.Helper()
		a := newApp()
		if err := a.HeadlessKey("load"); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessActivate("witness town"); err != nil {
			t.Fatal(err)
		}
		if !f.TownScreen().(*townScreen).CanSave() {
			t.Fatal("town fixture did not establish a resolved city save point")
		}
		if err := a.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		return a
	}
	standaloneIndex := -1
	for i, entry := range f.Maps {
		if entry.Mission == 0 && entry.Choosable() {
			standaloneIndex = i
			break
		}
	}
	if standaloneIndex < 0 {
		t.Fatal("lawful install has no standalone map row")
	}
	standalone := func(t *testing.T) *ui.App {
		t.Helper()
		a := newApp()
		if err := a.OpenMission(func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
			return f.loadMap(standaloneIndex)
		}); err != nil {
			t.Fatalf("open standalone: %v", err)
		}
		if err := a.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		return a
	}

	t.Run("campaign nonterminal actions", func(t *testing.T) {
		a := campaign(t)
		if err := a.HeadlessGameMenuAction("save"); err != nil || a.HeadlessMessage() != f.Words.SaveAcknowledgement {
			t.Fatalf("save: err=%v message=%q", err, a.HeadlessMessage())
		}
		if err := a.HeadlessGameMenuAction("load"); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessKey("escape"); err != nil || a.Screen() != ui.ScreenGameMenu {
			t.Fatalf("load return: err=%v screen=%s", err, a.Screen())
		}
		for _, page := range []string{"game-options", "sound-options", "objectives"} {
			if err := a.HeadlessGameMenuAction(page); err != nil {
				t.Fatal(err)
			}
			switch page {
			case "game-options":
				if err := a.HeadlessGameMenuAction("toggle-tips"); err != nil {
					t.Fatal(err)
				}
			case "sound-options":
				for _, action := range []string{"toggle-sound", "volume-down", "volume-up"} {
					if err := a.HeadlessGameMenuAction(action); err != nil {
						t.Fatalf("%s: %v", action, err)
					}
				}
			}
			if err := a.HeadlessGameMenuAction("page-return"); err != nil {
				t.Fatal(err)
			}
		}
		if err := a.HeadlessGameMenuAction("end"); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessGameMenuAction("confirm-end"); err == nil || !strings.Contains(err.Error(), "no such action") {
			t.Fatalf("campaign Change Map = %v", err)
		}
		if err := a.HeadlessGameMenuAction("victory"); err == nil || !strings.Contains(err.Error(), "not choosable") {
			t.Fatalf("campaign Victory before Continue = %v", err)
		}
		if err := a.HeadlessGameMenuAction("page-return"); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessGameMenuAction("return"); err != nil || a.Screen() != ui.ScreenMap {
			t.Fatalf("campaign return: err=%v screen=%s", err, a.Screen())
		}
	})

	for _, terminal := range []struct {
		action string
		want   ui.Screen
		exit   bool
	}{
		{action: "exit-main", want: ui.ScreenMenu},
		{action: "exit-windows", want: ui.ScreenGameMenu, exit: true},
	} {
		t.Run("campaign "+terminal.action, func(t *testing.T) {
			a := campaign(t)
			if err := a.HeadlessGameMenuAction("end"); err != nil {
				t.Fatal(err)
			}
			err := a.HeadlessGameMenuAction(terminal.action)
			if terminal.exit {
				if err == nil || !strings.Contains(err.Error(), "requested application exit") {
					t.Fatalf("exit-windows error = %v", err)
				}
				return
			}
			if err != nil || a.Screen() != terminal.want {
				t.Fatalf("%s: err=%v screen=%s want=%s", terminal.action, err, a.Screen(), terminal.want)
			}
		})
	}

	t.Run("campaign victory after Continue", func(t *testing.T) {
		a := releaseContinuedCampaign(t, f)
		requireReleaseMenuRows(t, a, []string{
			drawnMenuLabel(f.Words.MenuVictory), drawnMenuLabel(f.Words.MenuExitMain),
			drawnMenuLabel(f.Words.MenuExitWindows), drawnMenuLabel(f.Words.MenuReturn),
		}, []bool{true, true, true, true})
		if err := a.HeadlessGameMenuAction("victory"); err != nil || a.Screen() != ui.ScreenPicker {
			t.Fatalf("Victory after Continue: err=%v screen=%s", err, a.Screen())
		}
	})

	t.Run("standalone confirm-end", func(t *testing.T) {
		a := standalone(t)
		if err := a.HeadlessGameMenuAction("end"); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessGameMenuAction("confirm-end"); err != nil || a.Screen() != ui.ScreenPicker {
			t.Fatalf("standalone Change Map: err=%v screen=%s", err, a.Screen())
		}
	})

	t.Run("standalone population", func(t *testing.T) {
		a := standalone(t)
		for _, disabled := range []string{"save", "objectives"} {
			if err := a.HeadlessGameMenuAction(disabled); err == nil || !strings.Contains(err.Error(), "not choosable") {
				t.Fatalf("disabled %s = %v", disabled, err)
			}
		}
		for _, page := range []string{"diplomacy", "game-options", "sound-options"} {
			if err := a.HeadlessGameMenuAction(page); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessGameMenuAction("page-return"); err != nil {
				t.Fatal(err)
			}
		}
		if err := a.HeadlessGameMenuAction("end"); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessGameMenuAction("page-return"); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessGameMenuAction("return"); err != nil || a.Screen() != ui.ScreenMap {
			t.Fatalf("standalone return: err=%v screen=%s", err, a.Screen())
		}
	})

	t.Run("town nonterminal actions", func(t *testing.T) {
		a := town(t)
		if err := a.HeadlessGameMenuAction("save"); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessGameMenuAction("load"); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessKey("escape"); err != nil || a.Screen() != ui.ScreenGameMenu {
			t.Fatalf("town load return: err=%v screen=%s", err, a.Screen())
		}
		if err := a.HeadlessGameMenuAction("sound-options"); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessGameMenuAction("page-return"); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessGameMenuAction("abort"); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessGameMenuAction("page-return"); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessGameMenuAction("return"); err != nil || a.Screen() != ui.ScreenTown {
			t.Fatalf("town return: err=%v screen=%s", err, a.Screen())
		}
	})

	for _, terminal := range []struct {
		action string
		exit   bool
	}{
		{action: "confirm-abort"},
		{action: "exit-windows", exit: true},
	} {
		t.Run("town "+terminal.action, func(t *testing.T) {
			a := town(t)
			if err := a.HeadlessGameMenuAction("abort"); err != nil {
				t.Fatal(err)
			}
			err := a.HeadlessGameMenuAction(terminal.action)
			if terminal.exit {
				if err == nil || !strings.Contains(err.Error(), "requested application exit") {
					t.Fatalf("exit-windows error = %v", err)
				}
				return
			}
			if err != nil || a.Screen() != ui.ScreenMenu {
				t.Fatalf("confirm-abort: err=%v screen=%s", err, a.Screen())
			}
		})
	}
}
