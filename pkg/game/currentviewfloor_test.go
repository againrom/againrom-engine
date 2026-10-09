package game

import (
	"math"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func TestCurrentViewKeepsLiveYBelowOriginalFloor(t *testing.T) {
	view := ui.SaveApplicationState{ViewX: 12.25, ViewY: 7.34375, Zoom: 1.375, PeriodUS: 31000}
	app := &SnapshotApplicationState{Version: applicationStateVersion, View: view, Baseline: view,
		Original: OriginalStateData{ViewX: 12, ViewY: 9, Pressed: -1, Speed: 4}}
	snapshot := Snapshot{ApplicationState: app}
	raw, err := applicationCurrentRaw(app)
	if err != nil {
		t.Fatal(err)
	}
	if raw.ViewX != 12 || raw.ViewY != viewOriginFloor {
		t.Fatalf("ordinary SAV view = (%d, %d), want (12, %d)", raw.ViewX, raw.ViewY, viewOriginFloor)
	}
	var doc sav.DocumentData
	if err := projectCurrentSession(&doc, snapshot); err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	if actions.Session == nil || actions.Session.View == nil || actions.Session.View.YOriginDelta == nil || *actions.Session.View.YOriginDelta != -1 {
		t.Fatalf("current session did not carry the integer difference from the ordinary floor: %+v", actions.Session)
	}
	loaded := ui.SaveApplicationState{ViewX: float64(raw.ViewX), ViewY: float64(raw.ViewY), Zoom: 1}
	applyCurrentView(&loaded, raw, actions)
	if loaded.ViewX != view.ViewX || loaded.ViewY != view.ViewY || loaded.Zoom != view.Zoom {
		t.Fatalf("current view after SAV LOAD = (%v, %v, %v), want (%v, %v, %v)", loaded.ViewX, loaded.ViewY, loaded.Zoom, view.ViewX, view.ViewY, view.Zoom)
	}

	changed := raw
	changed.ViewX += 4
	changed.ViewY += 3
	loaded = ui.SaveApplicationState{ViewX: float64(changed.ViewX), ViewY: float64(changed.ViewY), Zoom: 1}
	applyCurrentView(&loaded, changed, actions)
	if loaded.ViewX != view.ViewX+4 || loaded.ViewY != view.ViewY+3 {
		t.Fatalf("edited ordinary view = (%v, %v), want (%v, %v)", loaded.ViewX, loaded.ViewY, view.ViewX+4, view.ViewY+3)
	}
}

func TestCurrentViewWithoutOffsetKeepsPriorSAVRules(t *testing.T) {
	raw := OriginalStateData{ViewX: 0, ViewY: 0}
	legacy := &currentActionData{Session: &currentSessionData{View: &currentViewResidue{
		Zoom: 1.25, FractionX: 0.25, FractionY: 0.34375, PeriodUS: 31000,
	}}}
	loaded := ui.SaveApplicationState{ViewX: float64(raw.ViewX), ViewY: float64(raw.ViewY)}
	applyCurrentView(&loaded, raw, legacy)
	if loaded.ViewX != 0.25 || loaded.ViewY != 0.34375 || loaded.Zoom != 1.25 {
		t.Fatalf("prior supplement view = (%v, %v, %v)", loaded.ViewX, loaded.ViewY, loaded.Zoom)
	}
	loaded = ui.SaveApplicationState{ViewX: float64(raw.ViewX), ViewY: float64(raw.ViewY), Zoom: 1}
	applyCurrentView(&loaded, raw, nil)
	if loaded.ViewX != 0 || loaded.ViewY != 0 || loaded.Zoom != 1 {
		t.Fatalf("original zero sentinel view changed: %+v", loaded)
	}
	view := ui.SaveApplicationState{ViewX: 0, ViewY: 0, Zoom: 1, PeriodUS: 31000}
	app := &SnapshotApplicationState{Version: applicationStateVersion, View: view, Baseline: view,
		Original: OriginalStateData{ViewX: 9, ViewY: 9, Pressed: -1, Speed: 4}}
	projected, err := applicationCurrentRaw(app)
	if err != nil || projected.ViewX != 0 || projected.ViewY != 0 {
		t.Fatalf("zero sentinel projected as %+v: %v", projected, err)
	}
	var doc sav.DocumentData
	if err := projectCurrentSession(&doc, Snapshot{ApplicationState: app}); err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil || actions.Session.View.YOriginDelta != nil {
		t.Fatalf("zero sentinel acquired an integer Y offset: %+v: %v", actions, err)
	}
	outOfRange := int64(1) << 32
	actions.Session.View.YOriginDelta = &outOfRange
	if err := validateCurrentSession(actions.Session); err == nil {
		t.Fatal("out-of-range current Y offset was admitted")
	}
}

func TestReleaseMissionTwentyKeepsCameraThroughSAVAndCommonTicks(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	a := f.App("mission camera save")
	a.Layout(1024, 768)
	open := f.NewGameOpener(20, ui.ChargenResult{Name: "Camera", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	want := f.live.view.SaveApplication()
	if want.ViewY != viewOriginFloor || want.ViewX < viewOriginFloor {
		t.Fatalf("fresh mission camera = (%v, %v), want row8 and a legal column", want.ViewX, want.ViewY)
	}
	store, name, wire := menuSAVE(t, f, a, OriginalStore{})
	doc, err := sav.DecodeDocumentData(wire)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := readOriginalApplicationState(&doc)
	if err != nil || raw.ViewY != viewOriginFloor {
		t.Fatalf("ordinary SAV Y=%d, want floor %d: %v", raw.ViewY, viewOriginFloor, err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	if actions.Session == nil || actions.Session.View == nil || actions.Session.View.YOriginDelta != nil {
		t.Fatalf("legal fresh camera acquired an integer Y offset: %+v", actions.Session)
	}
	g := releaseFront(t)
	g.SetDeterministicFrames(true)
	b := g.App("mission camera load")
	b.Layout(1024, 768)
	save, list, load := g.SaveSeams(store, OriginalStore{}, nil)
	b.SetSaveSeams(save, list, load)
	groundAppLoad(t, b, list, localOriginalSaveToken(name))
	if err := a.HeadlessKey("escape"); err != nil || a.Screen() != ui.ScreenMap {
		t.Fatal("saved mission did not resume", a.Screen(), err)
	}
	for tick := 0; tick <= 120; tick++ {
		live, cold := f.live.view.SaveApplication(), g.live.view.SaveApplication()
		if cold.ViewX != live.ViewX || cold.ViewY != live.ViewY || cold.Zoom != live.Zoom {
			t.Fatalf("common tick %d: camera saved (%v, %v, %v), loaded (%v, %v, %v)", tick,
				live.ViewX, live.ViewY, live.Zoom, cold.ViewX, cold.ViewY, cold.Zoom)
		}
		if tick < 120 {
			f.LiveAdvance(1)
			g.LiveAdvance(1)
		}
	}
}

func TestReleaseMissionEightyOneCameraKeepsItsOwnEdge(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := f.NextParty()
	f.Carried = mapload.CloneParty(party)
	a := f.App("mission camera edge")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpenerWith(81, party)); err != nil {
		t.Fatal(err)
	}
	edge := f.live.view.SaveApplication()
	edge.ViewY = 7.25
	if err := f.live.view.RestoreSaveApplication(edge); err != nil {
		t.Fatal(err)
	}
	want := f.live.view.SaveApplication()
	if want.ViewY != 7.25 || want.ViewX < viewOriginFloor {
		t.Fatalf("mission 81 edge camera = (%v, %v)", want.ViewX, want.ViewY)
	}
	store, name, wire := menuSAVE(t, f, a, OriginalStore{})
	doc, err := sav.DecodeDocumentData(wire)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := readOriginalApplicationState(&doc)
	if err != nil || raw.ViewY != originalViewFloor(want.ViewY) {
		t.Fatalf("ordinary mission 81 Y=%d, live=%v: %v", raw.ViewY, want.ViewY, err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	delta := int64(math.Round(want.ViewY)) - int64(raw.ViewY)
	if delta != -1 {
		t.Fatalf("mission 81 edge delta = %d, want -1", delta)
	}
	if actions.Session == nil || actions.Session.View == nil || delta == 0 && actions.Session.View.YOriginDelta != nil ||
		delta != 0 && (actions.Session.View.YOriginDelta == nil || *actions.Session.View.YOriginDelta != delta) {
		t.Fatalf("mission 81 integer camera offset = %+v, want %d", actions.Session, delta)
	}
	g := releaseFront(t)
	g.SetDeterministicFrames(true)
	b := g.App("mission camera edge load")
	b.Layout(1024, 768)
	save, list, load := g.SaveSeams(store, OriginalStore{}, nil)
	b.SetSaveSeams(save, list, load)
	groundAppLoad(t, b, list, localOriginalSaveToken(name))
	cold := g.live.view.SaveApplication()
	if cold.ViewX != want.ViewX || cold.ViewY != want.ViewY || cold.Zoom != want.Zoom {
		t.Fatalf("mission 81 camera saved (%v, %v, %v), loaded (%v, %v, %v)",
			want.ViewX, want.ViewY, want.Zoom, cold.ViewX, cold.ViewY, cold.Zoom)
	}
}
