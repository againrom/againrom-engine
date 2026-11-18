package game

import (
	"encoding/json"
	"fmt"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/ui"
)

// menuHasAbandon reports whether the in-mission menu of app lists the leave
// entry, and closes the menu again.
func menuHasAbandon(t *testing.T, label string, a *ui.App) bool {
	t.Helper()
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range a.HeadlessRows() {
		if row.Text == label {
			found = true
		}
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	return found
}

// The second game's mission-entered party that saves and loads still finds the
// leave entry in the mission menu, and the entry returns the party to its town,
// whether the save is loaded into the running front end or a cold one.
func TestLeaveEntrySurvivesLoadOfMissionSave(t *testing.T) {
	for _, second := range []bool{true} {
		for _, cold := range []bool{false, true} {
			t.Run(fmt.Sprintf("second=%t/cold=%t", second, cold), func(t *testing.T) {
				f, a, screen, n := abandonBaseTown(t, second)
				if second {
					secondBagEnter(t, f, a, screen)
				} else if err := a.OpenMission(f.MissionOpener(n)); err != nil {
					t.Fatal(err)
				}
				if !menuHasAbandon(t, f.Words.Engine.Text("mission.abandon"), a) {
					t.Fatal("the entry is missing before the save")
				}
				raw := skillExport(t, f, true, "leave after load")
				g, b := f, a
				if cold {
					g, b, _, _ = abandonBaseTown(t, second)
				}
				open, town, err := g.RestoreOriginal(raw)
				if err != nil || town || open == nil {
					t.Fatalf("mission LOAD: town=%t error=%v", town, err)
				}
				if err := b.OpenMission(open); err != nil {
					t.Fatal(err)
				}
				if !menuHasAbandon(t, g.Words.Engine.Text("mission.abandon"), b) {
					t.Fatal("the leave entry is missing after the load")
				}
				dest, line, _ := g.continuity(n, g.live.mission.state, g.live.advanceNotice)(ui.NoticeAbandon)
				if dest != ui.NoticeToTown {
					t.Fatalf("leaving: destination=%v message=%q", dest, line)
				}
				if second && g.Town.second.current != secondTown(1) {
					t.Fatalf("the party stands at %v, want town 1", g.Town.second.current)
				}
			})
		}
	}
}

// The same on an installed first game: the first campaign mission, entered
// from the town.
func TestReleaseLeaveEntrySurvivesLoadOfMissionSave(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	enter := func() (*FrontEnd, *ui.App, int) {
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		f.Town = NewTown(f.Campaign.Value())
		f.Carried = f.NextParty()
		f.arriveInTown()
		n := f.Town.Chapter()
		a := f.App("leave after load")
		t.Cleanup(a.StopAudio)
		a.Layout(1024, 768)
		if err := a.OpenMission(f.MissionOpener(n)); err != nil {
			t.Fatal(err)
		}
		return f, a, n
	}
	f, a, n := enter()
	if !menuHasAbandon(t, abandonLabel(f, f.Words.Engine.Text("mission.abandon")), a) {
		t.Fatal("the entry is missing before the save")
	}
	raw := skillExport(t, f, true, "leave after load")
	for _, cold := range []bool{false, true} {
		g, b := f, a
		if cold {
			g = releaseFront(t)
			g.SetDeterministicFrames(true)
			b = g.App("cold leave after load")
			t.Cleanup(b.StopAudio)
			b.Layout(1024, 768)
		}
		open, town, err := g.RestoreOriginal(raw)
		if err != nil || town || open == nil {
			t.Fatalf("mission LOAD (mission %d): town=%t error=%v", n, town, err)
		}
		if err := b.OpenMission(open); err != nil {
			t.Fatal(err)
		}
		if !menuHasAbandon(t, abandonLabel(g, g.Words.Engine.Text("mission.abandon")), b) {
			t.Fatalf("cold=%t: the leave entry is missing after the load", cold)
		}
	}
}

// A later town's departure is written to the document and read back; a
// document without the field reads as a mission entered from the map.
func TestSecondEntryTownRoundTrips(t *testing.T) {
	c := newSecondCampaign()
	c.current, c.entryTown = secondMission(30), secondTown(2)
	c.available = []partyPlace{secondTown(2), secondMission(30)}
	doc := captureSecondCampaign(c)
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var back currentSecondCampaign
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if got := back.restore().entryTown; got != secondTown(2) {
		t.Fatalf("entry town after the round trip = %v", got)
	}
	back.EntryTown = nil
	if got := back.restore().entryTown; got != (partyPlace{}) {
		t.Fatalf("a document without the field gave entry town %v", got)
	}
}
