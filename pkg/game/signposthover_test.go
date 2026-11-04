package game

import (
	"testing"

	"againrom/pkg/ui"
)

// A second-game signpost the hero has seen and left behind stays drawn on the
// explored map. Over it the cursor is the use cursor and a click walks the
// hero back and opens its message, as when the signpost is in current sight.
func TestSignpostInExploredGroundAnswersHoverAndClick(t *testing.T) {
	f := secondGameFront(t)
	app := f.App("second")
	app.Layout(1024, 768)
	enterSecondCampaignMission(t, app)
	live := f.live
	step := func(n int) {
		t.Helper()
		for range n {
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
	}
	step(20)
	dismissSecondMissionDialogue(t, app, live)
	step(40)
	var sign uint32
	for _, s := range live.world.Structures() {
		if s.Usable() && s.Kind == 0x4a {
			sign = uint32(s.ID)
		}
	}
	x, y, err := live.view.InspectionPoint(ui.InspectionSubject{Kind: ui.InspectionStructure, ID: sign})
	if err != nil {
		t.Fatal("the signpost in sight has no pixel to point at:", err)
	}
	hover := func() string {
		t.Helper()
		if err := app.HeadlessPointer("hover", x, y); err != nil {
			t.Fatal(err)
		}
		step(3)
		name, _ := app.HeadlessMapCursor()
		return name
	}
	if name := hover(); name != "town" {
		t.Fatalf("in sight: cursor %q, want the use cursor", name)
	}
	if err := live.world.HeadlessPlace(live.mission.ids[0], 26, 67); err != nil {
		t.Fatal(err)
	}
	step(70)
	if name := hover(); name != "town" {
		t.Errorf("explored, out of sight: cursor %q, want the use cursor", name)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	opened := false
	for range 4000 {
		step(1)
		if live.mission.open && live.mission.kind == ui.NoticeDialogue {
			opened = true
			break
		}
	}
	if !opened {
		t.Error("a click on the explored signpost opened no message")
	}
}
