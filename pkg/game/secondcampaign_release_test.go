package game

import (
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func enterSecondCampaignMission(t *testing.T, app *ui.App) {
	t.Helper()
	for _, target := range []string{"new game", "TAVERN", "TALK 517"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
		if app.Screen() != ui.ScreenTown {
			t.Fatalf("%s reached %v, want initial campaign town", target, app.Screen())
		}
	}
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatalf("initial inn dialogue did not open: %v", err)
	}
	for pages := 0; pages < 64; pages++ {
		if err := app.HeadlessActivate("notice"); err != nil {
			break
		}
		if pages == 63 {
			t.Fatal("initial inn dialogue exceeded 64 pages")
		}
	}
	for _, target := range []string{"GATES", "mission 10", "ENTER"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenMap {
		t.Fatalf("first campaign mission did not open: %v %q", app.Screen(), app.HeadlessMessage())
	}
}

func TestReleaseSecondCampaignFirstRoute(t *testing.T) {
	for _, continued := range []bool{false, true} {
		name := "immediate"
		if continued {
			name = "continued"
		}
		t.Run(name, func(t *testing.T) { secondCampaignFirstRoute(t, continued) })
	}
}

func secondCampaignFirstRoute(t *testing.T, continued bool) {
	t.Helper()
	f := secondGameFront(t)
	app := f.App("second")
	app.Layout(1024, 768)
	enterSecondCampaignMission(t, app)
	enterSecondCampaignNextMission(t, f, app, continued)
	for range 20 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
}

func enterSecondCampaignNextMission(t *testing.T, f *FrontEnd, app *ui.App, continued bool) {
	t.Helper()
	if f.liveMission != 10 {
		t.Fatalf("opened mission %d, want selected ID 10", f.liveMission)
	}
	if f.live.world.Outcome() != sim.OutcomeUndecided {
		t.Fatal("first mission started decided")
	}
	live := f.live
	for tick := 0; tick < 20; tick++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	dismissSecondMissionDialogue(t, app, live)
	id := live.mission.ids[0]
	if err := live.world.HeadlessPlace(id, 60, 36); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	live.view.Camera().CenterOn(60*32, 36*32)
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	x, y := secondCampaignCellPoint(t, app, 63, 36)
	if err := app.HeadlessPointer("press", x, y); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessPointer("release", x, y); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 1200; tick++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if live.mission.open && live.mission.kind == ui.NoticeDialogue {
			dismissSecondMissionDialogue(t, app, live)
		}
		if live.mission.open && live.mission.kind == ui.NoticeSuccess {
			break
		}
	}
	if live.world.Outcome() != sim.OutcomeWon || !live.mission.open || live.mission.kind != ui.NoticeSuccess {
		e, _ := live.world.Entity(id)
		t.Fatalf("ordinary move did not produce first victory: outcome=%v notice=%v/%v actor=%d,%d", live.world.Outcome(), live.mission.open, live.mission.kind, e.X, e.Y)
	}
	bank, _ := live.world.ROM2ScenarioState()
	if f.Town.second.current != (secondLocation{kind: 1, id: 10}) || f.Town.second.has(secondLocation{kind: 1, id: 20}) {
		t.Fatal("campaign left before victory acknowledgement")
	}
	if continued {
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		if live.mission.open || !live.mission.delayedVictory || f.Town.second.has(secondLocation{kind: 1, id: 20}) {
			t.Fatal("Continue completed or retained the victory panel")
		}
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		for _, action := range []string{"end", "victory"} {
			if err := app.HeadlessGameMenuAction(action); err != nil {
				t.Fatal(err)
			}
		}
	} else {
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() == ui.ScreenCutscene {
		if err := app.HeadlessCutsceneStep("key"); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessCutsceneStep(""); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenTown || f.live != nil {
		t.Fatalf("victory did not reach available destinations: %v %q", app.Screen(), app.HeadlessMessage())
	}
	c := f.Town.second
	if c.bank[768] != 20 || c.bank[906] != 1 || c.current != (secondLocation{}) || len(c.available) != 1 || c.available[0] != (secondLocation{kind: 1, id: 20}) {
		t.Fatalf("first Leave state: current=%+v available=%v stage=%d completed=%d", c.current, c.available, c.bank[768], c.bank[906])
	}
	for i := 0; i < 1024; i++ {
		if i >= 512 && i < 552 || i == 768 || i == 773 || i == 906 {
			continue
		}
		if c.bank[i] != bank[i] {
			t.Fatalf("unrelated bank slot %d changed", i)
		}
	}
	carried := f.NextParty()
	if len(carried) != len(live.mission.party) || carried[0].ID != live.mission.party[0].ID || carried[0].Name != live.mission.party[0].Name {
		t.Fatal("first victory lost controlled party identity")
	}
	if err := app.HeadlessActivate("notice"); err == nil {
		t.Fatal("duplicate acknowledgement found another victory")
	}
	if c.selected != (secondLocation{}) {
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		if app.Screen() != ui.ScreenTown || c.selected != (secondLocation{}) {
			t.Fatal("ordinary Back retained the finished destination selection")
		}
	}
	before := *c
	for _, target := range []string{"mission 20", "CANCEL"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenTown || !reflect.DeepEqual(before, *c) {
		t.Fatal("cancelled destination changed the campaign")
	}
	if err := app.HeadlessActivate("mission 10"); err == nil {
		t.Fatal("completed location remains selectable")
	}
	for _, target := range []string{"mission 20", "ENTER"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenMap || f.liveMission != 20 {
		t.Fatalf("next available destination: %v mission%d %q", app.Screen(), f.liveMission, app.HeadlessMessage())
	}
	nextBank, _ := f.live.world.ROM2ScenarioState()
	for i := range nextBank {
		want := c.bank[i]
		if i >= 752 && i < 768 {
			want = 0
		}
		if nextBank[i] != want {
			t.Fatalf("next mission bank[%d]=%d want%d", i, nextBank[i], want)
		}
	}
	if len(f.liveParty) != len(carried) || f.liveParty[0].ID != carried[0].ID || f.liveParty[0].Name != carried[0].Name || !reflect.DeepEqual(f.liveParty[0].CarriedItems, carried[0].CarriedItems) || f.liveParty[0].Worn != carried[0].Worn {
		t.Fatal("selected next destination replaced the carried party")
	}
	t.Logf("initial inn -> mission10 tick%d victory acknowledgement -> available20 -> mission20 tick%d; party=%s", live.world.Tick(), f.live.world.Tick(), carried[0].ID)
}

func secondCampaignCellPoint(t *testing.T, app *ui.App, col, row int) (int, int) {
	t.Helper()
	for y := 64; y < 650; y += 8 {
		for x := 180; x < 950; x += 8 {
			cx, cy, err := app.HeadlessDropCell(x, y)
			if err == nil && cx == col && cy == row {
				return x, y
			}
		}
	}
	t.Fatalf("cell %d,%d is outside the production map hit test", col, row)
	return 0, 0
}

func dismissSecondMissionDialogue(t *testing.T, app *ui.App, live *mapWorld) {
	t.Helper()
	for pages := 0; live.mission.open && live.mission.kind == ui.NoticeDialogue; pages++ {
		if pages >= 64 {
			t.Fatal("dialogue exceeded 64 pages")
		}
		body, _, shown := live.view.NoticeState()
		if !shown || strings.TrimSpace(body) == "" {
			t.Fatal("empty authored dialogue page")
		}
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReleaseSecondCampaignFailureDoesNotLeave(t *testing.T) {
	f := secondGameFront(t)
	app := f.App("second")
	app.Layout(1024, 768)
	enterSecondCampaignMission(t, app)
	live := f.live
	for ticks := 0; ticks < 20; ticks++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	dismissSecondMissionDialogue(t, app, live)
	before := *f.Town.second
	if err := live.world.HeadlessKill(live.mission.ids[0]); err != nil {
		t.Fatal(err)
	}
	for ticks := 0; ticks < 300 && !live.mission.open; ticks++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if !live.mission.open || live.mission.kind != ui.NoticeFailure {
		t.Fatal("ordinary hero death did not show failure")
	}
	if err := app.HeadlessActivate("exit to main menu"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMenu || !reflect.DeepEqual(before, *f.Town.second) || f.Offered != 0 {
		t.Fatal("failure acknowledged a campaign victory")
	}
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenTown || f.live != nil || !reflect.DeepEqual(f.Town.second, newSecondCampaign()) {
		t.Fatal("new game retained a failed campaign")
	}
}
