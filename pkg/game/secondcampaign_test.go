package game

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/render/menu"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

func secondCampaignFixture(t *testing.T, withTown bool) (*FrontEnd, *ui.App, *secondCampaignScreen) {
	t.Helper()
	return secondCampaignFixtureFiles(t, withTown)
}

func secondCampaignFixtureFiles(t *testing.T, withTown bool, files ...synth.File) (*FrontEnd, *ui.App, *secondCampaignScreen) {
	t.Helper()
	if withTown {
		files = append(files, synth.File{Path: "text/town.txt", Data: []byte("#npc517talk10\r\n<NPC=517,PART=1>\r\nFirst page.\r\n<NPC=21,PART=2>\r\nSecond page.\r\n#other\r\n")})
	}
	path := filepath.Join(t.TempDir(), MainArchive)
	if err := os.WriteFile(path, synth.Archive(files), 0o600); err != nil {
		t.Fatal(err)
	}
	fs, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	profile, ok := base.Find(base.ROM2EN)
	if !ok {
		t.Fatal("missing ROM2 profile")
	}
	mask := image.NewPaletted(image.Rect(0, 0, 640, 480), make(color.Palette, 256))
	for y := 50; y < 60; y++ {
		for x := 50; x < 60; x++ {
			mask.SetColorIndex(x, y, byte(0x80+16*(menu.NewGameButton-1)))
		}
	}
	f := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Containers: fs, Base: base.Match{Profile: profile}},
		Assets: &menu.Assets{Mask: mask}, Font: resolved(missionFont(), nil), Words: ui.AuthoredWords()},
		CampaignSession: CampaignSession{Town: NewTown(Campaign{}), Carried: []mapload.PartyMember{{ID: "old", Name: "old hero"}}}}
	app := f.App("campaign")
	app.Layout(1024, 768)
	screen := f.TownScreen().(*secondCampaignScreen)
	app.SetTown(screen)
	return f, app, screen
}

func TestSecondCampaignInputUnlockCancelAndFailedEntry(t *testing.T) {
	f, app, screen := secondCampaignFixture(t, true)
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenTown || f.live != nil {
		t.Fatal("new campaign bypassed initial location")
	}
	c := f.Town.second
	for slot, value := range c.bank {
		want := int32(0)
		if slot == 768 {
			want = 10
		}
		if value != want {
			t.Fatalf("initial bank[%d]=%d want%d", slot, value, want)
		}
	}
	if err := app.HeadlessActivate("GATES"); err == nil {
		t.Fatal("locked gate was enabled")
	}
	for _, target := range []string{"TAVERN", "TALK 517"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if !c.has(secondLocation{kind: 1, id: 10}) || c.current != (secondLocation{kind: 2, id: 1}) {
		t.Fatal("TALK did not unlock before acknowledgement")
	}
	if body, ok := screen.dialogueBody(); !ok || !strings.Contains(body, "First page") {
		t.Fatal("TALK lost the installed section")
	}
	if picture, ok := screen.TownDialogue(); !ok || picture == nil {
		t.Fatal("TALK has no composed dialogue")
	}
	for i := 0; i < 2; i++ {
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
	if screen.state().payload != nil {
		t.Fatal("conversation did not close")
	}
	if err := app.HeadlessActivate("TALK 517"); err != nil {
		t.Fatal(err)
	}
	if len(c.available) != 2 {
		t.Fatal("duplicate TALK duplicated availability")
	}
	for i := 0; i < 2; i++ {
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessActivate("GATES"); err != nil {
		t.Fatal(err)
	}
	beforeBank, beforeCurrent := c.bank, c.current
	beforeAvailable := append([]secondLocation(nil), c.available...)
	beforeParty := mapload.CloneParty(f.Carried)
	for _, target := range []string{"mission 10", "CANCEL", "mission 10", "ENTER"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenTown || !strings.Contains(app.HeadlessMessage(), "scenario/10.alm") || f.live != nil {
		t.Fatalf("failed entry lost destination screen: %v %q", app.Screen(), app.HeadlessMessage())
	}
	if c.bank != beforeBank || c.current != beforeCurrent || !reflect.DeepEqual(c.available, beforeAvailable) || !reflect.DeepEqual(f.Carried, beforeParty) {
		t.Fatal("failed or cancelled entry mutated campaign state")
	}
	if err := app.HeadlessActivate("ENTER"); err != nil {
		t.Fatal("failed destination cannot be retried", err)
	}
	if err := app.OpenMission(f.MissionOpener(20)); err == nil {
		t.Fatal("unavailable destination was admitted")
	}
}

func TestSecondCampaignFailedNewGameRetainsSession(t *testing.T) {
	f, app, _ := secondCampaignFixture(t, false)
	town, carried := f.Town, mapload.CloneParty(f.Carried)
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMenu || f.Town != town || !reflect.DeepEqual(f.Carried, carried) || !strings.Contains(app.HeadlessMessage(), "town.txt") {
		t.Fatal("failed initial town replaced the existing session")
	}
}

func TestSecondCampaignLeaveBankAlgebraAndUnknownBoundary(t *testing.T) {
	w := secondGameScriptWorld(t, []sim.ScriptInstant{{Op: sim.ScriptInstantWin}}, []sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	var incoming [1024]int32
	for i := range incoming {
		incoming[i] = int32(i*31 - 700)
	}
	incoming[768], incoming[775] = 10, 0
	for i := 0; i < 20; i++ {
		incoming[512+i] = int32(i % 2)
		incoming[532+i] = int32(i%3 - 1)
	}
	if !w.SetROM2ScenarioState(incoming) {
		t.Fatal("missing bank")
	}
	for i := 0; i < 16; i++ {
		sim.Step(w, nil)
	}
	c := newSecondCampaign()
	c.current, c.available = secondLocation{kind: 1, id: 10}, []secondLocation{{kind: 1, id: 10}}
	if err := c.finish(10, w, w.Outcome() == sim.OutcomeWon); err != nil {
		t.Fatal(err)
	}
	c.complete(w)
	for i, value := range c.bank {
		want := incoming[i]
		switch {
		case i >= 512 && i < 532:
			want = 0
		case i >= 532 && i < 552 && incoming[i] != 0:
			want = 1
			if incoming[i-20] != 0 {
				want = 2
			}
		case i == 768:
			want = 20
		case i == 773:
			want = 0
		case i == 906:
			want = 1
		}
		if value != want {
			t.Fatalf("Leave bank[%d]=%d want%d", i, value, want)
		}
	}
	if err := c.finish(10, w, w.Outcome() == sim.OutcomeWon); err == nil {
		t.Fatal("duplicate Leave admitted")
	}
	c.current = secondLocation{kind: 1, id: 30}
	if err := c.finish(30, w, w.Outcome() == sim.OutcomeWon); err == nil {
		t.Fatal("unclaimed successor inferred")
	}
	c.current = secondLocation{kind: 1, id: 10}
	incoming[775] = 1
	w.SetROM2ScenarioState(incoming)
	if err := c.finish(10, w, w.Outcome() == sim.OutcomeWon); err == nil {
		t.Fatal("unknown restored availability branch admitted")
	}
}

func TestSecondCampaignUnclaimedVictoryRetainsPanel(t *testing.T) {
	for _, restored := range []bool{false, true} {
		t.Run(fmt.Sprint(restored), func(t *testing.T) {
			f, app, _ := secondCampaignFixture(t, true)
			c := newSecondCampaign()
			// 128 is outside the bank, so no ordinary departure applies.
			n := 128
			if restored {
				n = 10
			}
			c.current, c.available = secondLocation{kind: 1, id: n}, []secondLocation{{kind: 1, id: n}}
			f.Town.second = c
			w := secondGameScriptWorld(t, []sim.ScriptInstant{{Op: sim.ScriptInstantWin}}, []sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
			if restored {
				bank, _ := w.ROM2ScenarioState()
				bank[775] = 1
				w.SetROM2ScenarioState(bank)
			}
			mw, view := missionDriverFor(t, w, nil, nil)
			mw.mission.table = &mapload.Table{Game: base.GameROM2}
			view.SetGameMenuContext(func() ui.GameMenuContext { return gameMenuContext(mw, true, "") })
			ms := &Mission{Number: n, World: w}
			advance := continueMission(frontTransitions{f}, n, ms, mw.advanceNotice)
			f.live, f.liveMission = mw, n
			if err := app.OpenMission(openPrepared(preparedMap{viewer: view, tick: mw.deterministicFrame, cadence: mw.setCadenceMode, advance: advance}, func() {})); err != nil {
				t.Fatal(err)
			}
			for ticks := 0; ticks < 32 && !mw.mission.open; ticks++ {
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			if !mw.mission.open || mw.mission.kind != ui.NoticeSuccess {
				t.Fatal("fixture victory did not reach production panel")
			}
			before := w.Hash()
			for presses := 0; presses < 2; presses++ {
				if err := app.HeadlessActivate("notice"); err != nil {
					t.Fatal(err)
				}
				if app.Screen() != ui.ScreenMap || !mw.mission.open || mw.mission.kind != ui.NoticeSuccess || c.current.id != n || c.bank[906] != 0 || w.Hash() != before || app.HeadlessMessage() == "" {
					t.Fatalf("unclaimed victory state: screen=%v open=%v kind=%v current=%v completed=%d hash=%v message=%q", app.Screen(), mw.mission.open, mw.mission.kind, c.current, c.bank[906], w.Hash() == before, app.HeadlessMessage())
				}
				if lines := view.MessageLines(); len(lines) == 0 || !strings.Contains(lines[len(lines)-1].Text, "unavailable") && !strings.Contains(lines[len(lines)-1].Text, "bank775") {
					t.Fatalf("continuation refusal absent from visible message line: %+v", lines)
				}
			}
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if mw.mission.open || !mw.mission.delayedVictory || mw.mission.victoryTaken || c.current.id != n || c.bank[906] != 0 {
				t.Fatal("Continue completed or retained the unclaimed victory panel")
			}
			for attempts := 0; attempts < 2; attempts++ {
				if err := app.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
				for _, action := range []string{"end", "victory"} {
					if err := app.HeadlessGameMenuAction(action); err != nil {
						t.Fatal(err)
					}
				}
				if app.Screen() != ui.ScreenMap || !mw.mission.delayedVictory || mw.mission.victoryTaken || c.current.id != n || c.bank[906] != 0 || app.HeadlessMessage() == "" {
					t.Fatal("refused End Quest Victory consumed completion or campaign state")
				}
			}
		})
	}
}
