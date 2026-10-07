package ui

import (
	"image"
	"math"
	"reflect"
	"testing"

	"againrom/pkg/render/terrain"
)

func applicationViewer(t *testing.T) *Viewer {
	t.Helper()
	v, err := NewViewer("application fixture", grid(80, 80), &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	v.Layout(1024, 768)
	v.localOwner, v.commandMode = 1, true
	v.SetFont(panelFont())
	v.SetEntities([]MapEntity{
		{ID: 41, Owner: 1, Cell: image.Pt(20, 19), Life: LifeAlive, MaxMana: 100},
		{ID: 7, Owner: 1, Cell: image.Pt(16, 17), Life: LifeAlive, MaxMana: 100},
	})
	return v
}

func TestSaveApplication1170CurrentControlsAndColdRestore(t *testing.T) {
	v := applicationViewer(t)
	// Use the selection producer and a real book click. Sparse, reverse-order
	// entities make a slice-index or single-primary capture fail independently.
	v.selectAllOwnedUnits()
	book := []SpellEntry{{ID: 1, Name: "First"}, {ID: 23, Name: "Second"}}
	v.SetSpellbook(7, book)
	var slots [4]uint32
	v.SetQuickSpells(&slots)
	x, y := sbEntryPoint(t, v, book, 1)
	sbClick(v, x, y)
	v.quickSpell(2, true, -1, -1)
	v.toggleHudPanel(hudPanelPack)
	v.ToggleShowHealth()
	v.ToggleTimeFlow()
	v.SetPeriod(terrain.SpeedIndexPeriod(6))
	v.Camera().X, v.Camera().Y = 384, 448
	// This expected state is declared before capture or encoding.
	want := SaveApplicationState{Selection: []uint32{7, 41}, ViewX: 12, ViewY: 14, Zoom: 1,
		InventoryOpen: false, SpellBookOpen: true, DollOpen: true, WornOpen: true, MinimapOpen: true,
		ShowHealth: false, FlyingHP: true, TimeFlow: false, PressedSpell: 23, PeriodUS: 41000}
	if slots != [4]uint32{0, 0, 23, 0} {
		t.Fatalf("current F7 = %v", slots)
	}
	got := v.SaveApplication()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("current UI = %+v, want %+v", got, want)
	}
	got.Selection[0] = 999
	if v.SelectedUnits()[0] != 7 {
		t.Fatal("capture leaked a writable selection alias")
	}
	fresh := applicationViewer(t)
	if err := fresh.RestoreSaveApplication(want); err != nil {
		t.Fatal(err)
	}
	if got := fresh.SaveApplication(); !reflect.DeepEqual(got, want) {
		t.Fatalf("cold UI = %+v, want %+v", got, want)
	}
	fresh.Layout(1024, 768)
	if got := fresh.SaveApplication(); !reflect.DeepEqual(got, want) {
		t.Fatalf("first layout replaced restored view: %+v", got)
	}
}

func TestSaveApplication1170RefusesBeforeChangingViewer(t *testing.T) {
	v := applicationViewer(t)
	v.selectAllOwnedUnits()
	before := v.SaveApplication()
	for _, test := range []struct {
		name   string
		mutate func(*SaveApplicationState)
	}{
		{"unresolved actor", func(s *SaveApplicationState) { s.Selection = []uint32{7, 999} }},
		{"duplicate actor", func(s *SaveApplicationState) { s.Selection = []uint32{7, 7} }},
		{"nonfinite view", func(s *SaveApplicationState) { s.ViewX = math.NaN() }},
		{"unknown cadence", func(s *SaveApplicationState) { s.PeriodUS = 777 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			bad := before
			test.mutate(&bad)
			bad.InventoryOpen = !before.InventoryOpen
			if err := v.RestoreSaveApplication(bad); err == nil {
				t.Fatal("invalid application accepted")
			}
			if !reflect.DeepEqual(v.SaveApplication(), before) {
				t.Fatal("refused application partially changed viewer")
			}
		})
	}
}

func TestSaveApplication1170FlowUsesLoadedCadenceWithoutProfileWrite(t *testing.T) {
	v := applicationViewer(t)
	s := v.SaveApplication()
	s.PeriodUS = 35000
	if err := v.RestoreSaveApplication(s); err != nil {
		t.Fatal(err)
	}
	writes, period := 0, 0
	f := flow{preferredRung: terrain.DefaultCadenceRung, preferredRungSet: true, persistRung: func(int) { writes++ }}
	f.enter(v, nil, nil, func(p int, _, _, _ bool) { period = p }, nil, nil, nil, nil, nil, nil)
	if period != 35000 || f.rung != terrain.CadenceShippedLo+7 || writes != 0 || v.SaveApplication().PeriodUS != 35000 {
		t.Fatalf("LOAD cadence period=%d rung=%d profile writes=%d", period, f.rung, writes)
	}
}

func TestOldHiddenMinimapApplicationRestoresVisible(t *testing.T) {
	v := applicationViewer(t)
	old := v.SaveApplication()
	old.MinimapOpen = false
	if err := v.RestoreSaveApplication(old); err != nil {
		t.Fatal(err)
	}
	if _, _, shown := v.minimapPresent(); !shown {
		t.Fatal("old false minimap switch hid the rendered minimap")
	}
	want := old
	want.MinimapOpen = true
	if got := v.SaveApplication(); !reflect.DeepEqual(got, want) {
		t.Fatalf("LOAD application=%+v want=%+v", got, want)
	}
}
