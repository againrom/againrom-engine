package game

import (
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func TestLegacyTransitionWaitObservesAutomaticTownArrivalWithoutInput(t *testing.T) {
	f, town := victoryWorldMapFront(t)
	ms, mw, _ := victoryContinueDriver(t, 30)
	f.continuity(30, ms, mw.advanceNotice)(ui.NoticeVictory)
	f.SetDeterministicFrames(true)
	app := f.App("legacy-transition-wait")
	app.SetSaveSeams(nil, func() []ui.SaveEntry {
		return []ui.SaveEntry{{Name: "return", Label: "return"}}
	}, func(string) (ui.MapOpener, bool, error) { return nil, true, nil })
	if err := app.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("return"); err != nil {
		t.Fatal(err)
	}
	until := &HeadlessUntil{Control: "TAVERN"}
	if ok, err := until.frontEndHolds(app); err != nil || ok || !town.WorldMapView().Returning {
		t.Fatalf("returning world-map accepted town control: holds=%v err=%v view=%+v", ok, err, town.WorldMapView())
	}
	if err := assertHeadlessState(headlessAppSnapshot(f, app), HeadlessStateAssertion{
		Screen: "town", TownPlace: "world_map",
	}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := headlessWaitFrontEnd(app, HeadlessStep{Until: until, Ticks: 4000}); err != nil {
		t.Fatal(err)
	}
	if !town.AtTownSquare() {
		t.Fatal("wait activated the tavern or did not arrive at the square")
	}
	if err := assertHeadlessState(headlessAppSnapshot(f, app), HeadlessStateAssertion{
		Screen: "town", TownPlace: "square",
	}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyTransitionObservationAssertionsDiscriminateAndStayVersioned(t *testing.T) {
	mission := 20
	for _, want := range []HeadlessStateAssertion{
		{Mission: &mission}, {Notice: "victory"}, {TownPlace: "world_map"},
	} {
		if err := assertHeadlessState(HeadlessState{}, want, nil); err == nil {
			t.Fatalf("absent transition accepted: %+v", want)
		}
		step := HeadlessStep{Command: "assert_state", State: &want}
		if err := step.validate(6, StageFrontEnd); err == nil || !strings.Contains(err.Error(), "version 7") {
			t.Fatalf("old vocabulary accepted transition assertion: %v", err)
		}
		if err := step.validate(7, StageFrontEnd); err != nil {
			t.Fatal(err)
		}
	}
	wait := HeadlessStep{Command: "wait_until", Until: &HeadlessUntil{Control: "TAVERN"}}
	if err := wait.validate(6, StageFrontEnd); err == nil {
		t.Fatal("version 6 accepted control wait")
	}
	if err := wait.validate(7, StageMission); err == nil {
		t.Fatal("mission stage accepted frontend control")
	}
	if err := (&HeadlessUntil{Control: "TAVERN", Screen: "town"}).validate(StageFrontEnd); err == nil {
		t.Fatal("ambiguous frontend wait accepted")
	}
	if err := assertHeadlessState(HeadlessState{Mission: 20, Notice: "victory", TownPlace: "world_map"},
		HeadlessStateAssertion{Mission: &mission, Notice: "victory", TownPlace: "world_map"}, nil); err != nil {
		t.Fatal(err)
	}
}
