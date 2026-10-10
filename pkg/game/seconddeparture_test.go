package game

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestSecondCampaignTwentyAcknowledgementProductionRoute(t *testing.T) {
	f, app, _ := secondCampaignFixture(t, true)
	c := newSecondCampaign()
	c.bank[768] = 20
	c.current, c.available = secondLocation{1, 20}, []secondLocation{{1, 20}}
	f.Town.second = c
	w := secondGameScriptWorld(t, []sim.ScriptInstant{{Op: sim.ScriptInstantWin}}, []sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	w.SetROM2ScenarioState(c.bank)
	mw, view := missionDriverFor(t, w, nil, nil)
	mw.mission.table = readUnder(base.GameROM2, &mapload.Table{})
	ms := &Mission{Number: 20, World: w}
	advance := continueMission(frontTransitions{f}, 20, ms, mw.advanceNotice)
	f.live, f.liveMission = mw, 20
	if err := app.OpenMission(openPrepared(preparedMap{viewer: view, tick: mw.deterministicFrame, cadence: mw.setCadenceMode, advance: advance}, func() {})); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 32 && !mw.mission.open; tick++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if !mw.mission.open || mw.mission.kind != ui.NoticeSuccess {
		t.Fatal("script victory did not reach production acknowledgement")
	}
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenTown || f.live != nil || c.current != (secondLocation{}) || !reflect.DeepEqual(c.available, []secondLocation{{2, 2}}) || c.bank[916] != 1 || c.bank[768] != 30 {
		t.Fatalf("M20 acknowledgement did not reach town2: screen=%v live=%v current=%v available=%v stage=%d completed=%d message=%q", app.Screen(), f.live != nil, c.current, c.available, c.bank[768], c.bank[916], app.HeadlessMessage())
	}
}

func TestSecondCampaignTwentyDepartureBankAndSelections(t *testing.T) {
	for _, gate := range []int32{0, -7, 1} {
		f, app, screen := secondCampaignFixture(t, true)
		if err := app.HeadlessActivate("new game"); err != nil {
			t.Fatal(err)
		}
		c := &secondCampaign{current: secondLocation{1, 20}, available: []secondLocation{{1, 20}}}
		f.Town.second = c
		var incoming [1024]int32
		for i := range incoming {
			incoming[i] = int32(13*i - 903)
		}
		incoming[768], incoming[772], incoming[775], incoming[917] = 20, gate, 0, 0
		for i := 0; i < 20; i++ {
			incoming[512+i], incoming[532+i] = int32(i%2), int32(i%3-1)
		}
		w := secondGameScriptWorld(t, []sim.ScriptInstant{{Op: sim.ScriptInstantWin}}, []sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
		w.SetROM2ScenarioState(incoming)
		for range 16 {
			sim.Step(w, nil)
		}
		if err := c.finish(20, w, w.Outcome() == sim.OutcomeWon); err != nil {
			t.Fatal(err)
		}
		c.complete(w)
		for slot, got := range c.bank {
			want := incoming[slot]
			switch {
			case slot >= 512 && slot < 532:
				want = 0
			case slot >= 532 && slot < 552 && incoming[slot] != 0:
				want = 1
				if incoming[slot-20] != 0 {
					want = 2
				}
			}
			switch slot {
			case 532, 916:
				want = 1
			case 552:
				want = 2
			case 768:
				want = 30
			case 773:
				want = 0
			}
			if got != want {
				t.Fatalf("gate%d bank[%d]=%d want%d", gate, slot, got, want)
			}
		}
		want := []secondLocation{{2, 2}}
		if gate != 0 {
			want = append(want, secondLocation{1, 21})
		}
		if !reflect.DeepEqual(c.available, want) {
			t.Fatal("native append order", c.available, want)
		}
		if err := c.finish(20, w, w.Outcome() == sim.OutcomeWon); err == nil {
			t.Fatal("duplicate departure admitted")
		}
		completedBank := c.bank
		choose := func(target string) error {
			for i, row := range screen.Rows() {
				if row.Text == target && row.Choosable {
					action := screen.Choose(i)
					if action.Open != nil {
						return app.OpenMission(action.Open)
					}
					return nil
				}
			}
			return fmt.Errorf("unavailable row %s", target)
		}
		for _, target := range []string{"town 2", "CANCEL", "town 2", "ENTER"} {
			if err := choose(target); err != nil {
				t.Fatal(err)
			}
		}
		if c.current != (secondLocation{2, 2}) || !screen.CanSave() || screen.Header() != "ROM2 campaign: town 2" {
			t.Fatal("quiet town2", c)
		}
		if choose("mission 10") == nil {
			t.Fatal("completed mission offered in town2")
		}
		if err := choose("TAVERN"); err != nil {
			t.Fatal(err)
		}
		// The incoming bank holds a nonzero slot 927, which withholds NPC 2108.
		want21 := c.available
		if rows := screen.Rows(); !reflect.DeepEqual(rows, []ui.TownRow{{Text: "TALK 22", Choosable: true}, {Text: "TALK 2110", Choosable: true}, {Text: "GATES", Choosable: true}}) || c.bank[927] == 0 || !screen.CanSave() {
			t.Fatal("stage-30 town2 inn rows", rows)
		}
		if !screen.Back() || c.room != secondTownSquare || !reflect.DeepEqual(c.available, want21) {
			t.Fatal("inn Back changed the campaign")
		}
		if c.bank != completedBank {
			t.Fatal("town entry changed bank")
		}
		raw, err := json.Marshal(captureSecondCampaign(c))
		if err != nil {
			t.Fatal(err)
		}
		var cold currentSecondCampaign
		if err := json.Unmarshal(raw, &cold); err != nil || !reflect.DeepEqual(c, cold.restore()) {
			t.Fatal("quiet town2 restoration", err)
		}
		if err := choose("GATES"); err != nil {
			t.Fatal(err)
		}
		if c.current != (secondLocation{}) || !reflect.DeepEqual(c.available, want) {
			t.Fatal("town2 departure removed available node")
		}
		if gate != 0 {
			for _, target := range []string{"mission 21", "CANCEL", "mission 21"} {
				if err := choose(target); err != nil {
					t.Fatal(err)
				}
			}
			if choose("ENTER") == nil {
				t.Fatal("absent mission map admitted")
			}
			if c.current != (secondLocation{}) || !reflect.DeepEqual(c.available, want) {
				t.Fatal("failed installed entry changed campaign")
			}
		} else if choose("mission 21") == nil {
			t.Fatal("zero772 enabled21")
		}
	}
}

func TestCurrentSecondLaterAdmissionIsAtomic(t *testing.T) {
	c := currentSecondCampaign{Current: currentSecondLocation{1, 21}, Available: []currentSecondLocation{{2, 2}, {1, 21}}}
	c.Bank[768], c.Bank[772], c.Bank[916] = 30, 1, 1
	if err := c.validate(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*currentSecondCampaign){
		func(c *currentSecondCampaign) { c.Bank[775] = 1 },
		func(c *currentSecondCampaign) { c.Current.ID = 30 },
		func(c *currentSecondCampaign) { c.Available = append(c.Available, c.Available[1]) },
	} {
		bad := c.clone()
		mutate(bad)
		raw, err := json.Marshal(bad)
		if err != nil {
			t.Fatal(err)
		}
		before := c.clone()
		if json.Unmarshal(raw, &c) == nil || !reflect.DeepEqual(before, &c) {
			t.Fatal("malformed later state admitted or changed target")
		}
	}
}

func TestSecondCampaignOptionalMissionDepartureRetainsTown(t *testing.T) {
	c := &secondCampaign{current: secondLocation{1, 21}, available: []secondLocation{{2, 2}, {1, 21}}}
	c.bank[768], c.bank[772], c.bank[916] = 30, -4, 1
	c.bank[512], c.bank[513], c.bank[515] = 3, -1, 4
	c.bank[532], c.bank[533], c.bank[534], c.bank[773] = -7, 6, 2, 19
	want := c.bank
	want[532], want[533], want[534], want[773], want[917] = 2, 2, 1, 0, 1
	for i := 512; i < 532; i++ {
		want[i] = 0
	}
	w := secondGameScriptWorld(t, []sim.ScriptInstant{{Op: sim.ScriptInstantWin}}, []sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	w.SetROM2ScenarioState(c.bank)
	for range 16 {
		sim.Step(w, nil)
	}
	if err := c.finish(21, w, w.Outcome() == sim.OutcomeWon); err != nil {
		t.Fatal(err)
	}
	c.complete(w)
	if c.bank != want || c.current != (secondLocation{}) || !reflect.DeepEqual(c.available, []secondLocation{{2, 2}}) {
		t.Fatal("optional ordinary departure invented a successor", c)
	}
	c.current = secondLocation{2, 2}
	if !c.savePoint() {
		t.Fatal("completed optional mission lost quiet town2 save point")
	}
}
