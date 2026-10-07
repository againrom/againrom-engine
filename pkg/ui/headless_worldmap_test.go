package ui

// Regression coverage for the gates screen once it draws the campaign world
// map instead of a row list (1007-world-map). roomGates's Rows() returns nil
// there by design (game.townScreen), so HeadlessActivate's Picker path has no
// rows to match a scenario's target against and every "walk out to mission N"
// step failed with "no row on screen town; available: []". Two shipped
// scenarios, scenarios/0152-save666.json and scenarios/0163-mission-to-town.json,
// crossed exactly this path and both failed on master before the hotfix that
// added headlessActivateWorldMap.
//
// fakeGatesTown reimplements just enough of pkg/game/worldmap.go's selection,
// travel and choose logic to exercise the real dispatch (WorldMapMove cycling
// enabled missions and starting travel, WorldMapTick arriving at the selected
// one) without pkg/ui importing pkg/game, which the layering forbids.

import (
	"fmt"
	"image"
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
)

type fakeGatesTown struct {
	missions []WorldMapMission
	selected int
	// ticks counts WorldMapTick calls since the selection started its travel,
	// which arrives once it reaches travelTicks. A negative travelTicks never
	// arrives.
	ticks, travelTicks int
	// chooses counts Enter presses.
	chooses int
	opened  []int
}

func (f *fakeGatesTown) Header() string        { return "the gates" }
func (f *fakeGatesTown) Rows() []TownRow       { return nil }
func (f *fakeGatesTown) Footer() []string      { return nil }
func (f *fakeGatesTown) Choose(int) TownAction { return TownAction{} }
func (f *fakeGatesTown) Back() bool            { return false }

func (f *fakeGatesTown) AtWorldMap() bool { return true }

func (f *fakeGatesTown) WorldMapView() WorldMapView {
	return WorldMapView{
		Missions:   append([]WorldMapMission(nil), f.missions...),
		Selected:   f.selected,
		RouteShown: f.ticks,
	}
}

func (f *fakeGatesTown) WorldMapHover(image.Point) {}

// WorldMapMove mirrors townScreen.WorldMapMove: it steps by delta through the
// mission slice, modulo its length, stops on the next ENABLED entry and starts
// travel to it.
func (f *fakeGatesTown) WorldMapMove(delta int) {
	if len(f.missions) == 0 || delta == 0 {
		return
	}
	i := f.selected
	if i < 0 && delta < 0 {
		i = 0
	}
	for attempts := 0; attempts < len(f.missions); attempts++ {
		i = (i + delta + len(f.missions)) % len(f.missions)
		if f.missions[i].Enabled {
			if i != f.selected {
				f.selected, f.ticks = i, 0
			}
			return
		}
	}
}

// WorldMapChoose mirrors townScreen.WorldMapChoose as far as the headless walk
// out depends on it: Enter opens nothing, and a selected mission that cannot be
// travelled to reports its Problem.
func (f *fakeGatesTown) WorldMapChoose() TownAction {
	f.chooses++
	if f.selected >= 0 && !f.missions[f.selected].Enabled {
		return TownAction{Msg: f.missions[f.selected].Problem}
	}
	return TownAction{}
}

func (f *fakeGatesTown) WorldMapClick(image.Point) TownAction { return TownAction{} }

// WorldMapTick mirrors townScreen.WorldMapTick: a selected mission's travel
// arrives, opening the mission, once travelTicks ticks have passed.
func (f *fakeGatesTown) WorldMapTick() TownAction {
	if f.selected < 0 {
		return TownAction{}
	}
	f.ticks++
	if f.travelTicks < 0 || f.ticks < f.travelTicks {
		return TownAction{}
	}
	m := f.missions[f.selected]
	f.selected, f.ticks = -1, 0
	f.opened = append(f.opened, m.Number)
	return TownAction{
		Open: func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
			v, err := NewViewer("m", grid(4, 4), &terrain.Tileset{})
			return v, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		},
		Msg: fmt.Sprintf("travelling to mission %d", m.Number),
	}
}

func gatesFlow(t *testing.T, town *fakeGatesTown) *App {
	t.Helper()
	a := newTestApp(t, appRows(0), okLoader(t))
	a.flow.town = town
	if !a.flow.showTown("") {
		t.Fatal("showTown refused a town that is not nil")
	}
	return a
}

// TestHeadlessActivateLeavesTheTownAtTheGates is the witness this hotfix was
// written against: headless naming a mission by its historical scenario text
// must open it from the gates' world map, exactly as a player's arrow keys
// and the ticks that follow do (stepTown, app.go). Before
// headlessActivateWorldMap existed, this failed with "no row on screen town;
// available: []" because roomGates draws no rows. Enter opens nothing on the
// map, so the walk sends none and the mission opens on arrival.
func TestHeadlessActivateLeavesTheTownAtTheGates(t *testing.T) {
	town := &fakeGatesTown{
		missions: []WorldMapMission{
			{Number: 10, Enabled: true},
			{Number: 20, Enabled: true},
			{Number: 30, Enabled: true},
		},
		selected:    -1,
		travelTicks: 3,
	}
	a := gatesFlow(t, town)

	if err := a.HeadlessActivate("walk out to mission 30"); err != nil {
		t.Fatalf(`HeadlessActivate("walk out to mission 30") = %v, want nil`, err)
	}
	if got := town.opened; len(got) != 1 || got[0] != 30 {
		t.Fatalf("opened missions = %v, want [30]", got)
	}
	if town.chooses != 0 {
		t.Fatalf("the walk pressed Enter %d times, want none", town.chooses)
	}
	if a.Screen() != ScreenMap {
		t.Fatalf("Screen() after activating a mission = %v, want ScreenMap", a.Screen())
	}
}

// TestHeadlessActivateWorldMapFailsWhenTheTravelNeverArrives checks the walk's
// own bound: a travel that never arrives fails the step, naming the frames it
// waited, and leaves the gates open with nothing opened.
func TestHeadlessActivateWorldMapFailsWhenTheTravelNeverArrives(t *testing.T) {
	town := &fakeGatesTown{
		missions:    []WorldMapMission{{Number: 20, Enabled: true}},
		selected:    -1,
		travelTicks: -1,
	}
	a := gatesFlow(t, town)

	err := a.HeadlessActivate("walk out to mission 20")
	if err == nil {
		t.Fatal(`HeadlessActivate("walk out to mission 20") = nil, want an error for a travel that never arrives`)
	}
	if want := fmt.Sprintf("did not arrive within %d frames", headlessWorldMapArrivalFrames); !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want it to contain %q", err.Error(), want)
	}
	if len(town.opened) != 0 || a.Screen() != ScreenTown {
		t.Fatalf("opened missions = %v, screen %v; want none opened and the gates open", town.opened, a.Screen())
	}
}

// TestHeadlessActivateWorldMapAcceptsItsOwnStatusPhrasing checks the second
// accepted form, the map's own "travelling to mission N" text (the Msg the
// arrival's action carries), alongside the historical scenario form.
func TestHeadlessActivateWorldMapAcceptsItsOwnStatusPhrasing(t *testing.T) {
	town := &fakeGatesTown{
		missions:    []WorldMapMission{{Number: 20, Enabled: true}},
		selected:    -1,
		travelTicks: 2,
	}
	a := gatesFlow(t, town)

	if err := a.HeadlessActivate("travelling to mission 20"); err != nil {
		t.Fatalf(`HeadlessActivate("travelling to mission 20") = %v, want nil`, err)
	}
	if got := town.opened; len(got) != 1 || got[0] != 20 {
		t.Fatalf("opened missions = %v, want [20]", got)
	}
}

// TestHeadlessActivateWorldMapListsAvailableMissionsOnMiss checks the failure
// path names what a reader can retry with, rather than the useless
// "available: []" an empty Picker row list gave before this hotfix.
func TestHeadlessActivateWorldMapListsAvailableMissionsOnMiss(t *testing.T) {
	town := &fakeGatesTown{
		missions: []WorldMapMission{
			{Number: 10, Enabled: true},
			{Number: 20, Enabled: true},
			{Number: 99, Enabled: false, Problem: "not yet offered"},
		},
		selected: -1,
	}
	a := gatesFlow(t, town)

	err := a.HeadlessActivate("walk out to mission 30")
	if err == nil {
		t.Fatal(`HeadlessActivate("walk out to mission 30") = nil, want an error naming what is available`)
	}
	if !strings.Contains(err.Error(), "[10 20]") {
		t.Fatalf("error = %q, want it to list the available mission numbers [10 20]", err.Error())
	}
	if len(town.opened) != 0 {
		t.Fatalf("opened missions = %v, want none", town.opened)
	}
}
