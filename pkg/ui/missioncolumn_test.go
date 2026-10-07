package ui

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/terrain"
)

// columnLayerRecord is one blitColumnLayer call this file observed, by name.
type columnLayerRecord struct {
	origin image.Point
	size   image.Point
}

// recordColumnLayers replaces blitColumnLayer for the duration of one test and
// records each named layer's own picture size and drawn origin. It does not
// forward to screen.DrawImage: the call itself is the observation.
func recordColumnLayers(t *testing.T) map[string]columnLayerRecord {
	t.Helper()
	rec := map[string]columnLayerRecord{}
	prev := blitColumnLayer
	blitColumnLayer = func(screen, img *ebiten.Image, op *ebiten.DrawImageOptions, layer string) {
		rec[layer] = columnLayerRecord{
			origin: image.Pt(int(op.GeoM.Element(0, 2)), int(op.GeoM.Element(1, 2))),
			size:   image.Pt(img.Bounds().Dx(), img.Bounds().Dy()),
		}
	}
	t.Cleanup(func() { blitColumnLayer = prev })
	return rec
}

// columnViewer is a viewer at the shipped mission frame size (1024x768, the
// only size this build composes a mission at), with a font, one selected
// entity and an equipped pack, so the panel, the doll and the
// minimap+control panel all present something to draw.
func columnViewer(t *testing.T) *Viewer {
	t.Helper()
	v, err := NewViewer("missioncolumn", terrain.Grid{
		Width: 32, Height: 32, Tiles: make([]uint16, 32*32),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.Layout(MissionFrameW, MissionFrameH)
	v.SetFont(panelFont())
	v.SetEntities([]MapEntity{panelEntity(1, "Warrior", 63, 100, 4, 4)})
	v.sel = selection{1}
	v.SetInventorySubject(packOf(1, solidPic(8, 8, color.RGBA{R: 0xff, A: 0xff})))
	// The command panel draws nothing with no art resolved (commandPanelPresent's
	// own doc), so a synthetic, panel-sized fill stands in here for the four
	// shipped bitmaps this test never reads — the same role solidPic already
	// plays for the pack bar above. Its content is never asserted; only that
	// Draw makes the "controlPanel" blitColumnLayer call at all, and where.
	panelFill := solidPic(160, 80, color.RGBA{G: 0x80, A: 0xff})
	v.SetCommandPanelArt(&CommandPanelArt{
		Heads:      panelFill,
		HeadsSeam:  solidPic(characterPaneSeamW, 80, color.RGBA{R: 0x80, A: 0xff}),
		Active:     panelFill,
		ActiveSeam: solidPic(characterPaneSeamW, 80, color.RGBA{R: 0x80, A: 0xff}),
	})
	// The fourth column child's own background draws nothing with no art
	// resolved (columnFillerPresent's own nil check), so a synthetic,
	// slot-sized fill stands in here on panelFill's own precedent — this
	// test only checks that Draw makes the "filler" blitColumnLayer call at
	// all, and where.
	v.SetCharacterPaneFillerArt(TownPane{
		Body: solidPic(160, characterPaneFillerBGH, color.RGBA{B: 0x80, A: 0xff}),
		Seam: solidPic(characterPaneSeamW, characterPaneFillerBGH, color.RGBA{R: 0x80, A: 0xff}),
	})
	return v
}

// MENU-COMBAT-017
func TestMissionColumnDrawsThePanelMinimapAndControlPanelAtTheirOwnSlots(t *testing.T) {
	v := columnViewer(t)
	rec := recordColumnLayers(t)

	v.Draw(ebiten.NewImage(MissionFrameW, MissionFrameH))

	columnX := MissionViewportSize().X
	if columnX != 864 {
		t.Fatalf("MissionViewportSize().X = %d, want 864 (1024-160, SESS-VIEW-028's strip origin)", columnX)
	}

	for _, tc := range []struct {
		layer      string
		wantOrigin image.Point
		wantSize   image.Point
		what       string
	}{
		// hudPanelTopY (238) is SHOP-FIGURE-041's own id-7 top boundary; the
		// panel's own height is compactPanelH (242), so its BODY fills exactly
		// (238..480), decoded. The composed picture is wider by
		// characterPaneSeamW (16) to its left, for the seam townshell.go's
		// drawCharacterPaneBody always paints now (round 3, P-A), so the
		// recorded layer's own origin and width both carry it.
		{"panel", image.Pt(columnX-characterPaneSeamW, 238), image.Pt(160+characterPaneSeamW, 242), "the character panel, widened by the seam"},
		{"filler", image.Pt(columnX-characterPaneSeamW, 480), image.Pt(160+characterPaneSeamW, characterPaneFillerBGH), "the fourth column child's own background and seam"},
		// card = filler + characterPaneFillerBGH (480+46), flush under the
		// background strip, compactPanelH (242) tall, so its own bottom is 768 —
		// the frame's own bottom edge, exactly. It carries the same
		// characterPaneSeamW widening to its left as the panel above, for the same
		// reason: drawCharacterPaneBody always paints the seam.
		{"missionCard", image.Pt(columnX-characterPaneSeamW, 526), image.Pt(160+characterPaneSeamW, 242), "the mission's own statistics card"},
		// authoredMinimapCorner's own margin is (0,0): flush to the frame's right
		// and top edges.
		{"minimap", image.Pt(columnX+2, 0), image.Pt(158, 158), "the minimap, id 5's own decoded slot, inset for its square"},
		// hudMinimapReserve (158) is the bar's own top; hudToggleBarH (80) is
		// id 6's own decoded height, both from SHOP-FIGURE-041.
		{"controlPanel", image.Pt(columnX-characterPaneSeamW, 158), image.Pt(160+characterPaneSeamW, 80), "the control panel, id 6's own decoded slot and seam"},
	} {
		got, ok := rec[tc.layer]
		if !ok {
			t.Errorf("%s: Draw made no blitColumnLayer call named %q", tc.what, tc.layer)
			continue
		}
		if got.origin != tc.wantOrigin {
			t.Errorf("%s: drawn at %v, want %v", tc.what, got.origin, tc.wantOrigin)
		}
		if got.size != tc.wantSize {
			t.Errorf("%s: picture size %v, want %v", tc.what, got.size, tc.wantSize)
		}
	}

	// The worn box is never one of the four calls at the shipped frame:
	// DIV-200's own conclusion, that nothing here fits below the fourth
	// column child's own full height, is a property of Draw's own call list
	// and not only of wornBoxRect's return value.
	if _, ok := rec["worn"]; ok {
		t.Error("Draw named a worn layer; the worn box does not fit in the column and must not draw (DIV-200)")
	}
}

func TestMissionColumnWornBoxUsesTheHookedBlit(t *testing.T) {
	v, err := NewViewer("missioncolumn-worn", terrain.Grid{
		Width: 32, Height: 32, Tiles: make([]uint16, 32*32),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	layoutViewport(v, MenuWindowW, wornBoxFixtureH)
	v.SetEntities([]MapEntity{panelEntity(1, "Warrior", 63, 100, 4, 4)})
	v.sel = selection{1}
	v.SetInventorySubject(packOf(1, solidPic(8, 8, color.RGBA{R: 0xff, A: 0xff})))

	wantBox, ok := v.wornBox()
	if !ok {
		t.Fatal("setup: wornBox answered false at a frame built to fit it")
	}

	rec := recordColumnLayers(t)
	v.Draw(ebiten.NewImage(v.frameW, v.frameH))

	got, ok := rec["worn"]
	if !ok {
		t.Fatal("Draw made no blitColumnLayer call named \"worn\" though wornPresent draws one")
	}
	if got.origin != wantBox.Min {
		t.Errorf("worn drawn at %v, want %v (wornBox's own origin)", got.origin, wantBox.Min)
	}
	if got.size != wantBox.Size() {
		t.Errorf("worn picture size %v, want %v (wornBox's own size)", got.size, wantBox.Size())
	}
}

func TestMissionColumnReplacesTheCardOnlyWithADrawnStructureReadout(t *testing.T) {
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	rec := recordColumnLayers(t)
	screen := ebiten.NewImage(MissionFrameW, MissionFrameH)
	check := func(what string, readout bool) {
		t.Helper()
		clear(rec)
		v.Draw(screen)
		_, gotReadout := rec["structureReadout"]
		_, gotCard := rec["missionCard"]
		if gotReadout != readout || !gotCard {
			t.Errorf("%s: drawn readout=%v card=%v, want readout=%v card=true", what, gotReadout, gotCard, readout)
		}
	}
	check("selected unit", false)
	ref := InspectionSubject{InspectionStructure, 7}
	inspectionHover(t, a, v, ref)
	check("hovered structure", true)
	v.sel, v.selStructure = nil, ref
	if err := a.HeadlessPointer("hover", -1, -1); err != nil {
		t.Fatal(err)
	}
	check("selected structure", true)
	inspectionHover(t, a, v, InspectionSubject{InspectionUnit, 7})
	check("hovered unit over selected structure", false)
	v.sel = selection{1}
	if err := a.HeadlessPointer("hover", -1, -1); err != nil {
		t.Fatal(err)
	}
	check("unit selection with inactive structure", false)
	v.sel, v.selStructure = nil, InspectionSubject{}
	check("no selection or hover", false)
	v.selStructure = ref
	v.SetFont(nil)
	clear(rec)
	v.Draw(screen)
	for _, layer := range []string{"structureReadout", "missionCard"} {
		if _, ok := rec[layer]; ok {
			t.Errorf("%s drawn without a font", layer)
		}
	}
}
