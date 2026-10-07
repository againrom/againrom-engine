package ui

import (
	"bytes"
	"image"
	"image/color"
	"reflect"
	"testing"

	"againrom/pkg/render/terrain"
)

func inspectionFixture(t *testing.T, size image.Point) (*App, *Viewer) {
	t.Helper()
	f := structureFrame()
	for i := range f.Pixels {
		f.Pixels[i].Opaque = true
	}
	c := &terrain.StructureClass{ID: 1, Name: "Gate", Picture: "gate", TileWidth: 1, TileHeight: 1, FullHeight: 1,
		Frames: []*terrain.StaticFrame{f}, Portrait: solidPic(160, 240, color.RGBA{G: 177, A: 255})}
	set := &terrain.StructureSet{}
	set.Classes[1] = c
	v := newStructureViewer(t, terrain.Grid{Width: 32, Height: 32, Tiles: make([]uint16, 1024),
		Structures: []terrain.StructureRecord{{ID: 7, X: 6 << 8, Y: 6 << 8, Key: 1}}}, set, true)
	a, _, _ := atOnMap(t)
	a.flow.viewer = v
	a.Layout(size.X, size.Y)
	v.commandMode = true
	v.SetFont(panelFont())
	v.SetPanelLayout(CompactPanelLayout(nil))
	v.SetLocalOwner(0)
	v.SetEntities([]MapEntity{
		{ID: 1, Name: "Selected", Cell: image.Pt(3, 3), Life: LifeAlive, HP: 33, MaxHP: 77},
		{ID: 7, Name: "Visitor", Owner: 2, Cell: image.Pt(8, 6), Life: LifeAlive, HP: 19, MaxHP: 90, Combat: UnitCombat{Known: true, Defence: 42}},
	})
	v.SetStructures([]MapStructure{{ID: 7, Health: 456, MaxHealth: 789, Cell: image.Pt(6, 6)}})
	v.sel = selection{1}
	inv := packOf(1, solidPic(160, 240, color.RGBA{R: 99, A: 255}))
	inv.Figure = solidPic(160, 240, color.RGBA{R: 99, A: 255})
	v.SetInventorySubject(inv)
	v.SetUnitInspectionPictureSource(func(id uint32) *image.RGBA { return solidPic(160, 240, color.RGBA{B: uint8(id), A: 255}) })
	v.Camera().X, v.Camera().Y = 0, 0
	return a, v
}

func inspectionHover(t *testing.T, a *App, v *Viewer, ref InspectionSubject) {
	t.Helper()
	x, y, err := v.InspectionPoint(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("hover", x, y); err != nil {
		t.Fatal(err)
	}
	if got, ok := v.Inspection(); !ok || got != ref {
		t.Fatalf("hover got %+v/%v want %+v", got, ok, ref)
	}
}

func TestInspectionProductionPointerKeepsSelectionAndBothPanelsTogether(t *testing.T) {
	for _, size := range []image.Point{{640, 480}, {800, 600}, {1024, 768}, {1280, 720}, {1920, 1080}} {
		t.Run(size.String(), func(t *testing.T) {
			a, v := inspectionFixture(t, size)
			t.Logf("window=%v frame=%v viewport=%v scale=%v", size, v.FrameSize(), v.ViewportSize(), v.Placement().Scale())
			inventory := v.invSubject
			for _, ref := range []InspectionSubject{{InspectionUnit, 7}, {InspectionStructure, 7}, {InspectionUnit, 7}} {
				inspectionHover(t, a, v, ref)
				s, ok := v.InspectionPanel()
				if !ok || s.ID != 7 || s.Kind != ref.Kind {
					t.Fatalf("subject=%+v,%v", s, ok)
				}
				view := v.characterPaneView(true)
				if view.Subject != s || view.Figure == nil {
					t.Fatal("card and figure subjects disagree")
				}
				if ref.Kind == InspectionStructure {
					if s.Name != "Gate" || s.HP != 456 || s.MaxHP != 789 {
						t.Fatalf("structure stats=%+v", s)
					}
					if _, ok := panelText(s, PanelFieldManaCard); ok {
						t.Fatal("structure invented mana")
					}
					if view.Figure.RGBAAt(50, 50).G != 177 {
						t.Fatal("structure got entity picture")
					}
				} else if s.HP != 19 || s.Combat.Defence != 42 || view.Figure.RGBAAt(50, 50).B != 7 {
					t.Fatal("unit live stats/figure lost")
				}
				if _, _, ok := v.panelPresent(); !ok {
					t.Fatal("pane not composed")
				}
				if _, _, ok := v.missionCardPresent(); !ok {
					t.Fatal("card not composed")
				}
				if v.panelKey.subject != s || v.missionCardKey.subject != s {
					t.Fatal("composed subjects disagree")
				}
				if !reflect.DeepEqual(v.sel, selection{1}) || !reflect.DeepEqual(v.invSubject, inventory) {
					t.Fatal("hover mutated selection/inventory")
				}
			}
			// A live update with no new pointer event must replace both cached cards.
			v.entities[1].HP = 8
			before, _, _ := v.missionCardPresent()
			if v.missionCardKey.subject.HP != 8 {
				t.Fatal("unit health stale")
			}
			v.entities[1].HP = 6
			after, _, _ := v.missionCardPresent()
			if bytes.Equal(before.Pix, after.Pix) {
				t.Fatal("live unit HP did not repaint")
			}
			if err := a.HeadlessPointer("hover", -1, -1); err != nil {
				t.Fatal(err)
			}
			if got, _ := v.Inspection(); got != (InspectionSubject{InspectionUnit, 1}) {
				t.Fatal("pointer leave did not restore selection")
			}
		})
	}
}

func TestInspectionGatesAndFogDoNotLeakOrChangeInventory(t *testing.T) {
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	ref := InspectionSubject{InspectionStructure, 7}
	inspectionHover(t, a, v, ref)
	x, y := v.cursorX, v.cursorY
	for _, gate := range []struct {
		name string
		set  func(bool)
	}{
		{"modal", func(b bool) { v.menuUp = b }}, {"primary", func(b bool) { v.primaryDown = b }},
		{"secondary", func(b bool) { v.secondaryDown = b }}, {"marquee", func(b bool) { v.dragging = b }},
		{"item drag", func(b bool) { v.dragActive = b }}, {"inventory grab", func(b bool) { v.invGrab = b }},
	} {
		t.Run(gate.name, func(t *testing.T) {
			gate.set(true)
			if got, _ := v.Inspection(); got.ID != 1 || got.Kind != InspectionUnit {
				t.Fatalf("gate leaked %+v", got)
			}
			gate.set(false)
		})
	}
	for _, p := range []image.Point{{900, 250}, {900, 550}, {1000, 10}, {20, 20}} {
		v.cursorX, v.cursorY = p.X, p.Y
		if _, ok := v.hoverInspection(); ok {
			t.Fatalf("HUD/outside point %v inspected map", p)
		}
	}
	v.cursorX, v.cursorY = x, y
	fog := make([]byte, 1024)
	for i := range fog {
		fog[i] = FogVisible
	}
	for _, state := range []byte{FogUnseen, FogExplored} {
		fog[6*32+6] = state
		v.SetFog(fog, 32, 32)
		if got, _ := v.Inspection(); got.ID != 1 {
			t.Fatal("hidden structure live stats leaked")
		}
	}
	v.SetFog(nil, 0, 0)
	inspectionHover(t, a, v, InspectionSubject{InspectionUnit, 7})
	fog[6*32+8] = FogExplored
	v.SetFog(fog, 32, 32)
	if got, _ := v.Inspection(); got.ID != 1 {
		t.Fatal("hidden unit leaked")
	}
	v.sel = selection{7}
	if _, ok := v.Inspection(); ok {
		t.Fatal("selected hidden unit leaked through fallback")
	}
	if view := v.characterPaneView(true); view.HasSubject || view.Figure != nil {
		t.Fatal("hidden selection still paints")
	}
	// Missing structure art is a blank figure, never an unrelated entity's doll.
	v.SetFog(nil, 0, 0)
	v.sel = selection{1}
	v.structureInfo[7].Portrait = nil
	inspectionHover(t, a, v, ref)
	if v.characterPaneView(true).Figure != nil {
		t.Fatal("missing portrait substituted")
	}
}

func TestInspectionUsesDrawDepthAndOpaquePixels(t *testing.T) {
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	f := structureFrame()
	for i := range f.Pixels {
		f.Pixels[i].Opaque = true
	}
	// Both IDs are 7, both images cover exactly the same cell. A unit on
	// the front row is the last paint; the tagged result must be the unit.
	v.entities[1].Cell = image.Pt(6, 6)
	v.entities[1].Frame = f
	v.entities[1].Art = &terrain.UnitClass{Width: 32, Height: 32}
	inspectionHover(t, a, v, InspectionSubject{InspectionUnit, 7})
	// Transparent unit pixels expose the building, and only the building.
	for i := range f.Pixels {
		f.Pixels[i].Opaque = false
	}
	inspectionHover(t, a, v, InspectionSubject{InspectionStructure, 7})
	// Hidden units must not occlude visible structure pixels.
	v.entities[1].Cell = image.Pt(6, 5)
	v.entities[1].Art.CenterY = -32
	for i := range f.Pixels {
		f.Pixels[i].Opaque = true
	}
	fog := make([]byte, 1024)
	for i := range fog {
		fog[i] = FogVisible
	}
	fog[5*32+6] = FogUnseen
	v.SetFog(fog, 32, 32)
	inspectionHover(t, a, v, InspectionSubject{InspectionStructure, 7})
}

func TestInspectionLeavesCommandHotkeysOnActualSelection(t *testing.T) {
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	var actors []uint32
	a.flow.stance = func(id uint32, guard bool) { actors = append(actors, id) }
	for _, ref := range []InspectionSubject{{InspectionUnit, 7}, {InspectionStructure, 7}} {
		inspectionHover(t, a, v, ref)
		if !v.inventoryEligible() || !v.dollInventoryActive() {
			t.Fatal("hover changed inventory eligibility")
		}
		in := atFrame(v.winCursorX, v.winCursorY)
		in.Guard = true
		a.step(in, atAt)
		if got, ok := v.Inspection(); !ok || got != ref {
			t.Fatal("hotkey replaced displayed hover")
		}
	}
	if !reflect.DeepEqual(actors, []uint32{1, 1}) {
		t.Fatalf("guard keys acted on %v instead of selected unit1", actors)
	}
}

func TestStructureCardNameIsTheInstalledBuildingName(t *testing.T) {
	_, v := inspectionFixture(t, image.Pt(640, 480))
	ref := InspectionSubject{InspectionStructure, 7}
	if s, ok := v.inspectionPanel(ref); !ok || s.Name != "Gate" {
		t.Fatalf("without an installed name the card reads %q, %v; want the registry name", s.Name, ok)
	}
	w := AuthoredWords()
	w.BuildingNames[1] = "Installed gate"
	v.SetWords(w)
	if s, ok := v.inspectionPanel(ref); !ok || s.Name != "Installed gate" {
		t.Fatalf("with an installed name the card reads %q, %v", s.Name, ok)
	}
}
