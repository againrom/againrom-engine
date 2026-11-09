package game

import (
	"bytes"
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

func TestMapEditorObjectGridPaintsMapSpritesWithoutPortraits(t *testing.T) {
	for _, hasPortrait := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing portrait", true: "available portrait"}[hasPortrait], func(t *testing.T) {
			_, e := openRecords(t)
			s := e.Document().Edits.(*mapInspectionEdits)
			before := bytes.Clone(s.model.Bytes())
			c := s.inspector.structures.Classes[5]
			c.Name, c.Picture, c.Indestructible = "Ruins", "ruins", true
			c.TileWidth, c.TileHeight, c.FullHeight = 2, 1, 2
			c.Frames = []*terrain.StaticFrame{{Width: 32, Height: 32, Pixels: make([]terrain.StaticPixel, 32*32)}}
			for _, green := range []uint8{70, 100, 130} {
				c.Frames = append(c.Frames, structurePreviewFrame(color.RGBA{G: green, A: 255}))
			}
			if hasPortrait {
				c.Portrait = structurePreviewFrame(color.RGBA{R: 190, A: 255}).RGBA()
			}
			s.inspector.objectRows = []ui.BrushObject{{Code: 0}, {Code: 7, Name: "Stone", Palette: ui.PaletteObjects}}
			s.inspector.statics = &terrain.StaticSet{}
			s.inspector.statics.Classes[7] = &terrain.StaticClass{Frame: structurePreviewFrame(color.RGBA{B: 180, A: 255})}
			if err := e.Key("objects"); err != nil {
				t.Fatal(err)
			}
			cell, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPlaceGridCell, N: 0})
			if !ok || e.PlaceGridLen() != 1 {
				t.Fatalf("Objects grid has %d cells, first visible: %v", e.PlaceGridLen(), ok)
			}
			pic := e.ChromeImage()
			counts := map[uint8]int{}
			for y := cell.Min.Y; y < cell.Max.Y; y++ {
				for x := cell.Min.X; x < cell.Max.X; x++ {
					ink := pic.RGBAAt(x, y)
					if ink.R == 0 && ink.B == 0 && ink.A == 255 {
						counts[ink.G]++
					}
				}
			}
			for _, green := range []uint8{70, 100, 130} {
				if counts[green] != 32*32 {
					t.Errorf("grid paints %d pixels of map frame green %d, want 1024", counts[green], green)
				}
			}
			if s.NewRecordPreview(ui.KindStructure, "kind", 5) != c.Portrait {
				t.Error("the Add window's separate information picture changed")
			}
			aside := s.NewRecordPreview("brush", "object", 7)
			if aside == nil || !bytes.Equal(aside.Pix, s.BrushObjectIcon(7).Pix) {
				t.Error("the static-object aside does not use BrushObjectIcon")
			}
			if !bytes.Equal(before, s.model.Bytes()) || s.Dirty() {
				t.Error("opening the Objects grid changed the map")
			}
		})
	}
}
