package ui

import (
	"image"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

type mapEntryArrival struct {
	*fakeGatesTown
	open MapOpener
}

func (town *mapEntryArrival) WorldMapTick() TownAction {
	action := town.fakeGatesTown.WorldMapTick()
	if action.Open != nil {
		action.Open = town.open
	}
	return action
}

func mapEntryTestViewer(t *testing.T) *Viewer {
	t.Helper()
	v, err := NewViewer("map entry fixture", grid(80, 80), &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestMapEntryWorldMapArrivalHasReadyViewBeforeFirstTick(t *testing.T) {
	town := &fakeGatesTown{missions: []WorldMapMission{{Number: 10, Enabled: true}}, selected: 0, travelTicks: 1}
	a := gatesFlow(t, town)
	v := mapEntryTestViewer(t)
	v.Camera().SetZoom(2)
	v.SetSAVStartView(image.Pt(40, 40))
	ticks, entries := 0, 0
	a.flow.town = &mapEntryArrival{fakeGatesTown: town, open: func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		return v, func() { ticks++ }, nil, nil, nil, nil, nil, nil, nil, nil, nil
	}}
	a.SetMapCadencePreference(6, nil)
	a.SetMapEntryObserver(func(entered *Viewer) {
		entries++
		state := entered.SaveApplication()
		if entered != v || state.Zoom != 1 || !entered.commandMode || ticks != 0 {
			t.Fatalf("accepted view before first tick: %+v command=%v ticks=%d", state, entered.commandMode, ticks)
		}
		if state.PeriodUS != terrain.CadencePeriod(6) {
			t.Fatalf("entry captured stale cadence: %+v", state)
		}
	})
	a.step(appInput{}, time.Unix(100, 0))
	if a.Screen() != ScreenMap || entries != 1 || ticks != 0 || v.SaveApplication().Zoom != 1 {
		t.Fatalf("arrival: screen=%v entries=%d ticks=%d zoom=%v", a.Screen(), entries, ticks, v.SaveApplication().Zoom)
	}
	a.Layout(1280, 960)
	a.step(appInput{}, time.Unix(101, 0))
	if entries != 1 || ticks == 0 {
		t.Fatalf("subsequent map frame: entries=%d ticks=%d", entries, ticks)
	}
}

func TestMapEntryLateObserverWaitsForNextAcceptedViewer(t *testing.T) {
	a := newTestApp(t, appRows(0), okLoader(t))
	v := mapEntryTestViewer(t)
	open := func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		return v, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil
	}
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	entries := 0
	a.SetMapEntryObserver(func(*Viewer) { entries++ })
	a.Layout(1280, 960)
	if entries != 0 {
		t.Fatal("configuration or resize became mission entry")
	}
	v = mapEntryTestViewer(t)
	v.SetSAVStartView(image.Pt(40, 40))
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	a.Layout(1024, 768)
	if entries != 1 {
		t.Fatalf("next accepted viewer entries=%d", entries)
	}
}
