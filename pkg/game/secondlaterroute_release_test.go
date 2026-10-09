package game

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondLaterEntry starts a new ROM2 campaign through the first inn talk,
// then places it in quiet town 2 at a later stage with town 2 and mission n
// available, leaves by GATES and enters mission n through the destination
// rows. The stage and destinations are a constructed campaign position: no
// published entry reaches stage 30 missions from a new game (DIV-2392).
func secondLaterEntry(t *testing.T, f *FrontEnd, app *ui.App, n int, stage int32) {
	t.Helper()
	for _, target := range []string{"new game", "TAVERN", "TALK"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	for pages := 0; app.HeadlessActivate("notice") == nil; pages++ {
		if pages == 64 {
			t.Fatal("initial inn dialogue exceeded 64 pages")
		}
	}
	c := f.Town.second
	c.current, c.available, c.room = secondLocation{2, 2}, []secondLocation{{2, 2}, {1, n}}, secondTownSquare
	c.bank[768] = stage
	// A held selection makes Back rebuild the rows from the placed state.
	c.selected = secondLocation{2, 2}
	if err := app.HeadlessKey("escape"); err != nil || c.selected != (secondLocation{}) {
		t.Fatal("town Back", err)
	}
	if err := app.HeadlessActivate("GATES"); err != nil {
		t.Fatal(err)
	}
	if c.current != (secondLocation{}) {
		t.Fatal("town 2 GATES did not depart", c.current)
	}
	secondLaterEnter(t, f, app, n)
}

func secondLaterEnter(t *testing.T, f *FrontEnd, app *ui.App, n int) {
	t.Helper()
	for _, target := range []string{secondLocationName(secondLocation{1, n}), "ENTER"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(target, err)
		}
	}
	if app.Screen() != ui.ScreenMap || f.liveMission != n || f.live.world.Outcome() != sim.OutcomeUndecided {
		t.Fatalf("mission %d did not open: %v %q", n, app.Screen(), app.HeadlessMessage())
	}
	for range 20 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	dismissSecondMissionDialogue(t, app, f.live)
}

func secondLaterAwaitVictory(t *testing.T, app *ui.App, live *mapWorld, ticks int) {
	t.Helper()
	for tick := 0; tick < ticks && !(live.mission.open && live.mission.kind == ui.NoticeSuccess); tick++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		dismissSecondMissionDialogue(t, app, live)
	}
	won, _ := live.world.ScriptCounters()
	if live.world.Outcome() != sim.OutcomeWon || !live.mission.open || live.mission.kind != ui.NoticeSuccess || won == 0 {
		t.Fatalf("installed victory: outcome=%v notice=%v/%v wins=%d", live.world.Outcome(), live.mission.open, live.mission.kind, won)
	}
}

// secondThirtyOneWin wins installed mission 31 through its own triggers:
// player 4 has no living unit and the hero stands within four cells of the
// unit the register-74 trigger measures, so that trigger raises register 74
// and the victory trigger fires. The kill and the placement are disclosed
// headless steps.
func secondThirtyOneWin(t *testing.T, app *ui.App, live *mapWorld) {
	t.Helper()
	w := live.world
	if _, err := w.HeadlessKillPlayer(4); err != nil {
		t.Fatal(err)
	}
	script := w.Script()
	checks, instants := script.Checks(), script.Instants()
	var distance *sim.ScriptCheck
	for _, tr := range script.Triggers() {
		raises := false
		for _, i := range tr.Instants {
			if i >= 0 && int(i) < len(instants) && instants[i].Op == sim.ScriptInstantIncVariable && instants[i].Args[0] == 74 {
				raises = true
			}
		}
		for _, p := range tr.Pairs {
			for i := range checks {
				if raises && p.Used && checks[i].Register == p.Left && checks[i].Op == sim.ScriptCheckUnitDistance {
					distance = &checks[i]
				}
			}
		}
	}
	if distance == nil {
		t.Fatal("mission 31 distance check absent")
	}
	id := live.mission.ids[0]
	if distance.Unit != id || !distance.HasUnit2 {
		t.Fatal("mission 31 distance check does not measure the hero", *distance)
	}
	target, ok := w.Entity(distance.Unit2)
	if !ok {
		t.Fatal("mission 31 distance subject absent")
	}
	if err := w.HeadlessPlace(id, target.X+1, target.Y); err != nil {
		t.Fatal(err)
	}
	secondLaterAwaitVictory(t, app, live, 400)
}

// TestReleaseSecondLaterDepartureReachesTheNextMap wins installed mission
// 31, acknowledges the victory, and enters the mission its departure adds.
func TestReleaseSecondLaterDepartureReachesTheNextMap(t *testing.T) {
	f := secondGameFront(t)
	app := f.App("later departure")
	app.Layout(1024, 768)
	secondLaterEntry(t, f, app, 31, 30)
	live := f.live
	secondThirtyOneWin(t, app, live)
	bank, _ := live.world.ROM2ScenarioState()
	party := live.mission.party
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatal(err)
	}
	c := f.Town.second
	if app.Screen() != ui.ScreenTown || f.live != nil || c.current != (secondLocation{}) || !reflect.DeepEqual(c.available, []secondLocation{{2, 2}, {1, 32}}) {
		t.Fatalf("mission 31 departure: screen=%v current=%v available=%v message=%q", app.Screen(), c.current, c.available, app.HeadlessMessage())
	}
	want := bank
	want[773] = 0
	for i := 0; i < 20; i++ {
		if bank[532+i] != 0 {
			want[532+i] = 1
			if bank[512+i] != 0 {
				want[532+i] = 2
			}
		}
		want[512+i] = 0
	}
	want[896+31] = 1
	if c.bank != want {
		t.Fatal("mission 31 departure changed a slot outside the prefix and slot 927")
	}
	secondLaterEnter(t, f, app, 32)
	if len(f.liveParty) != len(party) || f.liveParty[0].ID != party[0].ID || f.liveParty[0].Name != party[0].Name {
		t.Fatal("next map replaced the carried party")
	}
	next, _ := f.live.world.ROM2ScenarioState()
	for i := range next {
		expect := c.bank[i]
		if i >= 752 && i < 768 {
			expect = 0
		}
		if next[i] != expect {
			t.Fatalf("mission 32 bank[%d]=%d want %d", i, next[i], expect)
		}
	}
	t.Logf("installed M31 victory tick %d -> departure adds 32 -> M32 opens with the carried party", live.world.Tick())
}

// TestReleaseSecondMovieExitAtLaterDeparture wins installed mission 110 and
// plays the movie its output selects: output 4 with slot 779 zero, output 5
// with it nonzero.
func TestReleaseSecondMovieExitAtLaterDeparture(t *testing.T) {
	for _, gate := range []int32{0, 1} {
		t.Run(map[int32]string{0: "779 zero", 1: "779 nonzero"}[gate], func(t *testing.T) {
			f := secondGameFront(t)
			f.Cutscenes = OpenCutscenes(f.Archives.Root, "video4")
			app := f.App("later movie")
			app.Layout(1024, 768)
			secondLaterEntry(t, f, app, 110, 100)
			live := f.live
			bank, _ := live.world.ROM2ScenarioState()
			bank[779] = gate
			live.world.SetROM2ScenarioState(bank)
			var subject sim.EntityID
			found := false
			for _, c := range live.world.Script().Checks() {
				if c.Op == sim.ScriptCheckAlive && c.HasUnit {
					subject, found = c.Unit, true
					break
				}
			}
			if !found {
				t.Fatal("mission 110 alive subject absent")
			}
			if err := live.world.HeadlessKill(subject); err != nil {
				t.Fatal(err)
			}
			secondLaterAwaitVictory(t, app, live, 400)
			bank, _ = live.world.ROM2ScenarioState()
			output := 4
			if bank[779] != 0 {
				output = 5
			}
			stem, _ := LoadTextTable(f.Archives.Containers, mainPrefix+"text/cutpaths.txt").At(output)
			if stem == "" {
				t.Fatal("installed cutpaths row", output, "is empty")
			}
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
			if app.Screen() != ui.ScreenCutscene || filepath.Dir(app.CutsceneName()) != stem || app.CutsceneError() != nil {
				t.Fatalf("output %d: screen=%s movie=%q want stem %q err=%v", output, app.Screen(), app.CutsceneName(), stem, app.CutsceneError())
			}
			movie := app.CutsceneName()
			for range 4 {
				if app.Screen() != ui.ScreenCutscene {
					break
				}
				if err := app.HeadlessCutsceneStep("key"); err != nil {
					t.Fatal(err)
				}
			}
			c := f.Town.second
			if app.Screen() != ui.ScreenTown || c.bank[896+110] != 1 || c.bank[768] != 110 || c.current != (secondLocation{}) || !reflect.DeepEqual(c.available, []secondLocation{{2, 2}}) {
				t.Fatalf("after movie: screen=%v current=%v available=%v stage=%d", app.Screen(), c.current, c.available, c.bank[768])
			}
			t.Logf("installed M110 victory with 779=%d -> output %d -> %s", bank[779], output, movie)
		})
	}
}

// TestReleaseSecondLaterMissionSaveContinuation saves installed mission 32
// after the route from mission 31 reached it, cold-loads the SAV in a fresh
// front end, and runs the same pointer command and ticks on both.
func TestReleaseSecondLaterMissionSaveContinuation(t *testing.T) {
	f := secondGameFront(t)
	app := f.App("later save")
	app.Layout(1024, 768)
	secondLaterEntry(t, f, app, 31, 30)
	secondThirtyOneWin(t, app, f.live)
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatal(err)
	}
	secondLaterEnter(t, f, app, 32)
	id := f.live.mission.ids[0]
	e, _ := f.live.world.Entity(id)
	secondMissionMove(t, app, id, int(e.X)+1, int(e.Y))
	for range 8 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if f.live.world.Outcome() != sim.OutcomeUndecided || app.HeadlessNoticeOpen() {
		t.Fatal("mission 32 did not reach a settled nonterminal state")
	}
	out := secondMissionSaveDirectory(t)
	before := secondSaveSampleNow(t, f, app)
	if before.Campaign.Aux == nil || before.Campaign.Aux[0][1] != 10000 {
		t.Fatal("stage 30 auxiliary stores absent from the saved campaign")
	}
	raw := secondMissionNamedSave(t, f, app, out, "Later mission")
	cold, a := secondMissionColdAt(t, out, "Later mission.sav", 32)
	secondAssertSample(t, before, secondSaveSampleNow(t, cold, a))
	assertCurrentWorldEqual(t, f.live.world, cold.live.world, "cold LOAD")
	e, _ = f.live.world.Entity(id)
	x, y := int(e.X), int(e.Y)+1
	secondMissionMove(t, app, id, x, y)
	secondMissionMove(t, a, id, x, y)
	for range 24 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		assertCurrentWorldEqual(t, f.live.world, cold.live.world, "post-load ordinary action")
	}
	after := secondSaveSampleNow(t, f, app)
	secondAssertSample(t, after, secondSaveSampleNow(t, cold, a))
	if bytes.Equal(before.World, after.World) {
		t.Fatal("next ordinary action did not change World")
	}
	t.Logf("route M31 -> M32 -> named %d-byte SAV -> cold LOAD -> 24 identical post-action ticks", len(raw))
}
