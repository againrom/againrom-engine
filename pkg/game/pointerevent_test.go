package game

import (
	"image"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const pointerEventLine = "To the tower"

// pointerEventApp opens a second-game mission with the hero at (3,3) and one
// 1x1 Pointer at (8,8) whose action slot 1 is the source slot of the message
// instant of event 19.
func pointerEventApp(t *testing.T) (*ui.App, *mapWorld, *ui.Viewer, *sim.World) {
	t.Helper()
	_, app, _ := secondCampaignFixture(t, false)
	s, err := sim.NewROM2Script(missionChecks(), []sim.ScriptInstant{
		{Op: sim.ScriptInstantMessage, Args: [10]int32{7}},
		{Slot: 1, Op: sim.ScriptInstantMessage, Args: [10]int32{19}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	hero := sim.Entity{ID: 0, Owner: sim.SelfSlot, X: 3, Y: 3, HP: 100, MaxHP: 100, Reach: 1, Facing: 64,
		Speed: 50, TokenSize: 1, TypeID: 1, HealthRegenPeriod: 1, ManaRegenPeriod: 1, Capacity: data.UnitCapacity()}
	pointer := sim.Structure{ID: 0, Kind: 0x4a, UseAmount: 1, Col: 8, Row: 8, Width: 1, Height: 1, Attach: 1, Blocking: 1}
	w, err := sim.NewStructuredWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{hero}, s, sim.Relations{}, nil, nil, nil, sim.GhostTemplate{}, []sim.Structure{pointer})
	if err != nil {
		t.Fatal(err)
	}
	text := "#event7\r\n<part=1>\r\nwrong event\r\n#event19\r\n<part=1>\r\n" + pointerEventLine + "\r\n"
	mw, view := missionDriverFor(t, w, nil, missionSource{"main/text/mission7.txt": []byte(text)})
	mw.mission.table = readUnder(base.GameROM2, &mapload.Table{})
	view.SetMissionScreen(secondGameFiles.missionScreen)
	if err := app.OpenMission(openPrepared(preparedMap{viewer: view, tick: mw.deterministicFrame,
		cadence: mw.setCadenceMode, advance: mw.advanceNotice}, func() {})); err != nil {
		t.Fatal(err)
	}
	return app, mw, view, w
}

// A use of the Pointer walks the hero to it; arrival opens the event window
// with event 19's text, the mission holds while it is open, and Ok, Enter and
// Escape each close it. The action does not run again after the close.
func TestPointerArrivalOpensItsEventWindowAndHoldsTheMission(t *testing.T) {
	for _, closer := range []string{"ok", "enter", "escape"} {
		t.Run(closer, func(t *testing.T) {
			app, mw, view, w := pointerEventApp(t)
			mw.useStructure(0, 0)
			for steps := 0; steps < 400 && !mw.mission.open; steps++ {
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			body, kind, open := view.NoticeState()
			if !open || kind != ui.NoticeDialogue || body != pointerEventLine+"\r\n" {
				t.Fatalf("notice %q/%v/%v, want event 19's line as a dialogue", body, kind, open)
			}
			if hero := w.Entities()[0]; hero.X < 7 || hero.X > 9 || hero.Y < 7 || hero.Y > 9 {
				t.Fatalf("the event opened with the hero at %d,%d", hero.X, hero.Y)
			}
			held := w.Hash()
			for range 48 {
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			if w.Hash() != held {
				t.Fatal("the mission ran under the open event window")
			}
			switch closer {
			case "ok":
				// The window stands centred on the 1024x768 frame at (268,268)
				// (R2-ENGINE-608); Ok is the layout's button inside it.
				b := ui.AuthoredDialogueLayout().Button
				for _, action := range []string{"press", "release"} {
					if err := app.HeadlessPointer(action, 268+4, 268+4); err != nil {
						t.Fatal(err)
					}
				}
				if !mw.mission.open {
					t.Fatal("a click off Ok closed the event window")
				}
				p := image.Pt(268, 268).Add(b.Min.Add(b.Max).Div(2))
				for _, action := range []string{"press", "release"} {
					if err := app.HeadlessPointer(action, p.X, p.Y); err != nil {
						t.Fatal(err)
					}
				}
			default:
				if err := app.HeadlessKey(closer); err != nil {
					t.Fatal(err)
				}
			}
			if _, _, open := view.NoticeState(); open || mw.mission.open {
				t.Fatalf("%s left the event window open", closer)
			}
			for range 48 {
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			if w.Hash() == held {
				t.Fatal("the mission did not run on after the close")
			}
			if mw.mission.open {
				t.Fatal("the Pointer's action ran again after the close")
			}
		})
	}
}
