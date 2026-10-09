package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

// secondStageFortyInn plays a new ROM2 campaign to the stage 30 inn and its
// two mission talks, then applies the controller's ordinary departure of
// mission 30 at the destination rows, which raises slot 768 to 40
// (R2-SESSION-111). Installed mission 30 is not won here: its victory
// trigger names a companion the party producer does not yet supply
// (DIV-2391), so the departure is a disclosed constructed step. The
// campaign then enters town 2 and its inn at stage 40.
func secondStageFortyInn(t *testing.T, f *FrontEnd, app *ui.App) *secondCampaign {
	t.Helper()
	c := secondStageThirtyVisit(t, f, app)
	c.current = secondLocation{1, 30}
	if output := c.completeBank(c.bank); output != 2 || c.bank[768] != 40 || c.bank[896+30] != 1 {
		t.Fatalf("mission 30 departure: output=%d stage=%d", output, c.bank[768])
	}
	if !reflect.DeepEqual(c.available, []secondLocation{{2, 2}, {1, 31}}) {
		t.Fatal("mission 30 departure availability", c.available)
	}
	// A held selection makes Back rebuild the rows from the placed state.
	c.selected = secondLocation{2, 2}
	if err := app.HeadlessKey("escape"); err != nil || c.selected != (secondLocation{}) {
		t.Fatal("destination Back", err)
	}
	secondLaterChoose(t, app, "town 2", "ENTER", "TAVERN")
	if c.current != (secondLocation{2, 2}) || c.room != secondTownInn || c.bank[768] != 40 {
		t.Fatalf("town 2 inn: current=%v room=%d stage=%d", c.current, c.room, c.bank[768])
	}
	return c
}

// secondStageFortyTalks talks to every stage 40 speaker in row order and
// returns the availability after each (R2-ENGINE-223, R2-ENGINE-229,
// R2-ENGINE-230).
func secondStageFortyTalks(t *testing.T, f *FrontEnd, app *ui.App) [][]secondLocation {
	t.Helper()
	c := f.Town.second
	var out [][]secondLocation
	for _, target := range []string{"TALK 22", "TALK 2108", "TALK 2015", "TALK 2111", "TALK 2004"} {
		bank := c.bank
		secondLaterChoose(t, app, target)
		pages := 0
		for ; app.HeadlessActivate("notice") == nil; pages++ {
			if pages == 64 {
				t.Fatal(target, "dialogue exceeded 64 pages")
			}
		}
		if c.bank != bank {
			t.Fatal(target, "changed the bank")
		}
		out = append(out, append([]secondLocation(nil), c.available...))
		t.Logf("%s: %d dialogue pages, available %v", target, pages, c.available)
	}
	return out
}

// TestReleaseSecondStageFortyInnOffersItsMissions shows the stage 40 inn's
// speakers, admits missions 40 to 43 by TALK, saves in the inn, cold-loads
// the SAV, and runs the same talks and the entry of mission 40 on both.
func TestReleaseSecondStageFortyInnOffersItsMissions(t *testing.T) {
	f := secondGameFront(t)
	app := f.App("stage 40 inn")
	app.Layout(1024, 768)
	c := secondStageFortyInn(t, f, app)
	rows := []ui.TownRow{{Text: "TALK 22", Choosable: true}, {Text: "TALK 2108", Choosable: true},
		{Text: "TALK 2015", Choosable: true}, {Text: "TALK 2111", Choosable: true}, {Text: "TALK 2004", Choosable: true},
		{Text: "GATES", Choosable: true}}
	if got := f.TownScreen().Rows(); !reflect.DeepEqual(got, rows) {
		t.Fatal("stage 40 inn rows", got)
	}

	out := secondMissionSaveDirectory(t)
	before := secondTownSampleNow(t, f, app)
	raw := secondTownNamedSave(t, f, app, out, "Stage forty inn")
	secondTownShape(t, raw)
	cold, a := secondTownCold(t, out, "Stage forty inn.sav")
	secondTownAssertSample(t, before, secondTownSampleNow(t, cold, a))
	if got := cold.TownScreen().Rows(); !reflect.DeepEqual(got, rows) {
		t.Fatal("cold stage 40 inn rows", got)
	}

	want := [][]secondLocation{
		{{2, 2}, {1, 31}, {1, 40}},
		{{2, 2}, {1, 31}, {1, 40}},
		{{2, 2}, {1, 31}, {1, 40}, {1, 41}},
		{{2, 2}, {1, 31}, {1, 40}, {1, 41}, {1, 42}},
		{{2, 2}, {1, 31}, {1, 40}, {1, 41}, {1, 42}, {1, 43}},
	}
	if got := secondStageFortyTalks(t, f, app); !reflect.DeepEqual(got, want) {
		t.Fatal("source stage 40 talks", got)
	}
	if got := secondStageFortyTalks(t, cold, a); !reflect.DeepEqual(got, want) {
		t.Fatal("cold stage 40 talks", got)
	}
	secondTownAssertSample(t, secondTownSampleNow(t, f, app), secondTownSampleNow(t, cold, a))
	for _, run := range []struct {
		f   *FrontEnd
		app *ui.App
	}{{f, app}, {cold, a}} {
		secondLaterChoose(t, run.app, "GATES")
		secondLaterEnter(t, run.f, run.app, 40)
	}
	// The source camera carries scroll state from missions 10 and 20, so the
	// comparison is the hashed world, the campaign and the party.
	assertCurrentWorldEqual(t, f.live.world, cold.live.world, "stage 40 mission entry")
	source, loaded := secondSaveSampleNow(t, f, app), secondSaveSampleNow(t, cold, a)
	if source.Hash != loaded.Hash || !reflect.DeepEqual(source.Campaign, loaded.Campaign) || !reflect.DeepEqual(source.Party, loaded.Party) {
		t.Fatal("mission 40 entry differs after the cold LOAD")
	}
	bank, _ := f.live.world.ROM2ScenarioState()
	if c.current != (secondLocation{1, 40}) || bank[768] != 40 {
		t.Fatal("mission 40 did not open at stage 40", c.current, bank[768])
	}
	if _, err := os.Stat(filepath.Join(out, "Stage forty inn.sav")); err != nil {
		t.Fatal(err)
	}
	t.Logf("new game -> stage 30 inn -> Leave30 -> stage 40 inn -> %d-byte SAV -> cold LOAD -> TALK 22/2108/2015/2111/2004 -> M40 opens on both", len(raw))
}
