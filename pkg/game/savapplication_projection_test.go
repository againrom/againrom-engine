package game

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func TestApplicationSelectionOriginalResaveShortIDs(t *testing.T) {
	for _, tc := range []struct {
		name     string
		raw      []uint32
		bindings map[uint32]uint32
		want     []uint32
		fail     bool
	}{
		{"short original resave", []uint32{137}, map[uint32]uint32{9: 0x21000089}, []uint32{9}, false},
		{"wide legacy selection", []uint32{0x21000089}, map[uint32]uint32{9: 0x21000089}, []uint32{9}, false},
		{"exact match wins", []uint32{137}, map[uint32]uint32{9: 0x21000089, 10: 137}, []uint32{10}, false},
		{"short collision", []uint32{137}, map[uint32]uint32{9: 0x21000089, 10: 0x22000089}, nil, true},
		{"duplicate exact", []uint32{137}, map[uint32]uint32{9: 137, 10: 137}, nil, true},
		{"missing short", []uint32{138}, map[uint32]uint32{9: 0x21000089}, nil, true},
		{"missing wide stays invalid", []uint32{0x22000089}, map[uint32]uint32{9: 0x21000089}, nil, true},
		{"zero is not an alias", []uint32{0}, map[uint32]uint32{9: 0x21000000}, nil, true},
		{"duplicate mixed selection", []uint32{137, 0x21000089}, map[uint32]uint32{9: 0x21000089}, nil, true},
		{"two members", []uint32{183, 137}, map[uint32]uint32{9: 0x21000089, 10: 0x051000b7}, []uint32{9, 10}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := make(map[uint32]uint32, len(tc.bindings))
			for key, value := range tc.bindings {
				before[key] = value
			}
			got, err := applicationSelection(tc.raw, tc.bindings)
			if (err != nil) != tc.fail || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("selection=%v error=%v want=%v fail=%v", got, err, tc.want, tc.fail)
			}
			if !reflect.DeepEqual(tc.bindings, before) {
				t.Fatal("selection remapped an actor runtime ID")
			}
		})
	}
}

func TestSAVApplicationProjectsNativeCameraPanelsAndSpeed(t *testing.T) {
	for _, tc := range []struct {
		name         string
		x, y         float64
		period       int
		speed        int32
		wantX, wantY int32
	}{
		// viewOriginFloor (SESS-VIEW-030): a projected axis below 8 floors
		// to 8, so "fractional"'s negative Y lands on the floor rather
		// than its plain rounded value. "fastest"'s exact (0, 0) pair is
		// left alone: originalViewOrigin treats an exact zero/zero view
		// as this build's own "camera never captured" sentinel (no
		// genuine corpus save sits at 0 on either axis, let alone both),
		// not a real position the original's own floor was ever meant to
		// correct.
		{"fractional", 12.49, -13.5, 62000, 4, 12, 8},
		{"rounded_up", 12.5, 13.51, 250000, 0, 13, 14},
		{"fastest", 0, 0, 15625, 8, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			view := ui.SaveApplicationState{ViewX: tc.x, ViewY: tc.y, Zoom: 2, PeriodUS: tc.period, Unpaced: true}
			app := &SnapshotApplicationState{Version: 1, View: view, Baseline: view,
				Original: OriginalStateData{Speed: 4, Pressed: -1, Formation: 31, Wimpy: 7}}
			before := cloneApplicationState(app)
			got, err := applicationCurrentRaw(app)
			if err != nil {
				t.Fatal(err)
			}
			if got.ViewX != tc.wantX || got.ViewY != tc.wantY || got.Speed != tc.speed || got.Formation != 31 || got.Wimpy != 7 {
				t.Fatalf("projected application = %+v", got)
			}
			if !reflect.DeepEqual(app, before) {
				t.Fatal("SAV projection changed the live application state")
			}
		})
	}
}

func TestSAVLocalSettingsKeepUnrecordedApplicationFields(t *testing.T) {
	_, snapshot, world := applicationSnapshotFixture(t)
	doc, err := sav.CloneDocumentData(*snapshot.SavedDocument.Document)
	if err != nil {
		t.Fatal(err)
	}
	want, err := readOriginalApplicationState(&doc)
	if err != nil {
		t.Fatal(err)
	}
	view := localOnlyApplicationView(ui.SaveApplicationState{ShowHealth: want.ShowHP == 0, FlyingHP: want.FlyingHP == 0, TimeFlow: want.ShowTimeFlow == 0})
	snapshot.ApplicationState = &SnapshotApplicationState{LocalOnly: true, Version: 1, View: view, Baseline: view,
		Original: OriginalStateData{Wimpy: 0, Formation: 2}}
	snapshot.CameraSet, snapshot.CameraX, snapshot.CameraY, snapshot.CameraZoom = true, 12.75, 13.25, 2
	before := cloneApplicationState(snapshot.ApplicationState)
	if err := projectApplicationState(&doc, snapshot, world); err != nil {
		t.Fatal(err)
	}
	want.ShowHP, want.FlyingHP, want.ShowTimeFlow = boolToInt32(view.ShowHealth), boolToInt32(view.FlyingHP), boolToInt32(view.TimeFlow)
	want.Formation, want.Wimpy, want.ViewX, want.ViewY = 2, 0, 13, 13
	got, err := readOriginalApplicationState(&doc)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("local settings projection = %+v, want %+v: %v", got, want, err)
	}
	if !reflect.DeepEqual(before, snapshot.ApplicationState) {
		t.Fatal("export changed the local-only checkpoint")
	}
}

func TestSAVAbsentApplicationUsesDocumentAndRecordedCamera(t *testing.T) {
	_, snapshot, world := applicationSnapshotFixture(t)
	doc, err := sav.CloneDocumentData(*snapshot.SavedDocument.Document)
	if err != nil {
		t.Fatal(err)
	}
	want, err := readOriginalApplicationState(&doc)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.ApplicationState = nil
	snapshot.CameraSet, snapshot.CameraX, snapshot.CameraY, snapshot.CameraZoom = true, 12.75, 13.25, 2
	if err := projectApplicationState(&doc, snapshot, world); err != nil {
		t.Fatal(err)
	}
	want.ViewX, want.ViewY = 13, 13
	got, err := readOriginalApplicationState(&doc)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("absent application projection = %+v, want %+v: %v", got, want, err)
	}
}
