package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseSecondGameMissionTenAuthoredDialogueIsReadable(t *testing.T) {
	archives := secondGameRoot(t)
	defs, err := LoadDefinitionsFor(archives.Containers, archives.Game())
	if err != nil {
		t.Fatal(err)
	}
	party := MissionPartyAs(false, defs.StartWeapon, defs.Bodies, defs.Table)
	mission, err := StartMission(archives.Containers, 10, defs.Table, mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatal(err)
	}
	if len(mission.Raises) == 0 {
		t.Fatal("mission 10 compiled no dialogue announcements")
	}
	raise := mission.Raises[0]
	payload, ok := ReadEventTextFor(archives.Containers, archives.Game(), mission.Number, int(raise.Event))
	if !ok || len(payload) == 0 {
		t.Fatalf("authored dialogue event %d on latch %d has no readable text", raise.Event, raise.Latch)
	}
	t.Logf("game=%s language selector=%d event=%d payload bytes=%d", archives.Game(), LanguageSelector(archives.Containers), raise.Event, len(payload))
}

func TestReleaseSecondGameFailureUsesInstalledFallback(t *testing.T) {
	archives := secondGameRoot(t)
	words := LoadInstallWords(archives.Containers, InstallTextCode(archives.Containers, archives.Game().Edition()))
	for _, tc := range []struct{ reason, slot int }{{3, 283}, {4, 284}} {
		t.Run(fmt.Sprintf("reason%d", tc.reason), func(t *testing.T) {
			want, found := words.Global(tc.slot)
			wrong, wrongFound := words.Global(118 + tc.reason)
			if !found || want == "" || !wrongFound || wrong == want {
				t.Fatal("installed right and wrong fallback slots must distinguish the selection")
			}
			w := secondGameScriptWorld(t,
				[]sim.ScriptInstant{{Op: 4}, {Op: 5, Args: [10]int32{int32(tc.reason)}}},
				[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0, 1)})
			mw, view := missionDriverFor(t, w, nil, nil)
			mw.mission.table = &mapload.Table{Game: archives.Game()}
			mw.mission.failureText = secondGameFailureText(archives.Containers, InstallTextCode(archives.Containers, archives.Game().Edition()), 10)
			missionSteps(mw, 1)
			body, kind, shown := view.NoticeState()
			if !shown || kind != ui.NoticeFailure || body != want || w.Outcome() != sim.OutcomeLost {
				t.Fatalf("failure%d = %q/%v/%v, outcome=%v; want installed slot%d %q", tc.reason, body, kind, shown, w.Outcome(), tc.slot, want)
			}
		})
	}
}

func TestReleaseSecondGameInitialDialogueAppears(t *testing.T) {
	front := secondGameFront(t)
	app := front.App("second")
	app.Layout(1024, 768)
	enterSecondCampaignMission(t, app)
	if app.Screen() != ui.ScreenMap || front.live == nil {
		t.Fatalf("mission did not open: %v %q", app.Screen(), app.HeadlessMessage())
	}
	for i := 0; i < 300; i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if front.live.mission.open && front.live.mission.kind == ui.NoticeDialogue {
			break
		}
	}
	m := front.live.mission
	if m.open && m.kind == ui.NoticeDialogue {
		payload, audience := m.payload, m.dialogueAudience
		for part := 1; part <= 64; part++ {
			body, expected := dialoguePart(payload, part, audience)
			if !expected {
				if m.open && m.kind == ui.NoticeDialogue && string(m.payload) == string(payload) {
					t.Fatal("initial dialogue restarted after acknowledgement")
				}
				return
			}
			got, kind, shown := front.live.view.NoticeState()
			if !shown || kind != ui.NoticeDialogue || got != body || m.part != part {
				t.Fatalf("page %d not delivered: %q/%v/%v", part, got, kind, shown)
			}
			portrait, face := front.live.view.NoticeSpeaker()
			layout := front.Words.OnLayout(ui.AuthoredDialogueLayout()).WithPortrait(portrait)
			layout.Frame = front.gameMenuArt()
			picture := image.NewRGBA(image.Rect(0, 0, 640, 480))
			ui.ComposeDialogueNotice(picture, layout, front.Font.Value(), body, face, layout.Box.Min)
			if out := os.Getenv("AGAINROM_ROM2_DIALOGUE_FRAMES"); out != "" {
				writeFramePNG(t, filepath.Join(out, fmt.Sprintf("%s-page%d.png", filepath.Base(os.Getenv("AGAINROM_ASSETS")), part)), picture)
			}
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
		}
		t.Fatal("initial dialogue exceeded the bounded page count")
	}
	w := front.live.world
	if w.Outcome() != sim.OutcomeUndecided {
		t.Fatalf("mission ended before its initial dialogue: %v", w.Outcome())
	}
	t.Fatalf("initial dialogue is absent after %d ticks; first message latch=%v", w.Tick(), w.ScriptLatched(0))
}

func TestReleaseSecondGameAuthoredNewTriggersHaveNoGaps(t *testing.T) {
	archives := secondGameRoot(t)
	defs, err := LoadDefinitionsFor(archives.Containers, archives.Game())
	if err != nil {
		t.Fatal(err)
	}
	party := MissionPartyAs(false, defs.StartWeapon, defs.Bodies, defs.Table)
	opened := 0
	for number := 1; number <= 300; number++ {
		address, _ := MissionMap(number)
		if _, err := archives.Containers.ReadFile(address); err != nil {
			continue
		}
		mission, err := StartMission(archives.Containers, number, defs.Table, mapload.DifficultyNormal, party)
		if err != nil {
			t.Fatal(err)
		}
		opened++
		_, report, err := mapload.CompileROM2Script(mission.Map, campaignScriptRefs(mission.Map, defs.Table, party, number))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("mission=%d omitted actions=%v checks=%v triggers=%v unresolved=%+v", number, report.OmittedActions, report.OmittedChecks, report.OmittedTriggers, report.Unresolved)
		if number == 10 && len(report.OmittedActions)+len(report.OmittedChecks)+len(report.OmittedTriggers) != 0 {
			t.Errorf("mission %d omits authored script references", number)
		}
		for _, gap := range mission.World.Script().Unsupported() {
			if gap.Kind == sim.ScriptGapCheck && gap.Op >= 23 && gap.Op <= 27 ||
				gap.Kind == sim.ScriptGapInstant && gap.Op >= 35 && gap.Op <= 39 {
				t.Errorf("mission %d leaves new trigger arm unsupported: %+v", number, gap)
			}
		}
	}
	if opened != 46 {
		t.Fatalf("opened %d campaign maps, want 46", opened)
	}
}

func TestReleaseSecondGameMissionTenAllDialoguePages(t *testing.T) {
	front := secondGameFront(t)
	app := front.App("second")
	app.Layout(1024, 768)
	enterSecondCampaignMission(t, app)
	for step := 0; step < 16; step++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	mw := front.live
	for page := 0; mw.mission.open; page++ {
		if page >= 64 {
			t.Fatal("initial notices did not finish")
		}
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
	for event := 1; event <= 7; event++ {
		if !mw.openDialogue(event) {
			t.Fatalf("mission10 event%d did not open", event)
		}
		m := mw.mission
		payload, audience := m.payload, m.dialogueAudience
		pages := 0
		for part := 1; part <= 64; part++ {
			body, expected := dialoguePart(payload, part, audience)
			if !expected {
				break
			}
			got, kind, shown := mw.view.NoticeState()
			if !shown || kind != ui.NoticeDialogue || got != body || m.part != part {
				t.Fatalf("event%d page%d not delivered", event, part)
			}
			portrait, face := mw.view.NoticeSpeaker()
			layout := front.Words.OnLayout(ui.AuthoredDialogueLayout()).WithPortrait(portrait)
			layout.Frame = front.gameMenuArt()
			picture := image.NewRGBA(image.Rect(0, 0, 640, 480))
			ui.ComposeDialogueNotice(picture, layout, front.Font.Value(), body, face, layout.Box.Min)
			if out := os.Getenv("AGAINROM_ROM2_DIALOGUE_FRAMES"); out != "" {
				writeFramePNG(t, filepath.Join(out, fmt.Sprintf("%s-event%d-page%d.png", filepath.Base(os.Getenv("AGAINROM_ASSETS")), event, part)), picture)
			}
			pages++
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
		}
		if m.open {
			t.Fatalf("event%d did not close after %d pages", event, pages)
		}
		if event == 7 {
			want := 4
			if LanguageSelector(front.Archives.Containers) == 1 {
				want = 3
			}
			if pages != want {
				t.Fatalf("event7 has %d selected pages, want %d", pages, want)
			}
		}
		t.Logf("event%d delivered %d pages", event, pages)
	}
}
