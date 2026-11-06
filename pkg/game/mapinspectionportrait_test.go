package game

import (
	"bytes"
	"image/color"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/ui"
)

func TestEditorDraftPortraitFollowsFaceSexAndNPC(t *testing.T) {
	f, s, raw := nativeIdentityEditor(t)
	for _, dir := range []data.FigureDir{data.FigureDirManFighter, data.FigureDirWomanFighter} {
		for _, face := range []int{2, 7} {
			pixels := make([]synth.Pixel256, 160*240)
			for i := range pixels {
				pixels[i] = synth.Pixel256{Index: 1, Opaque: true}
			}
			green := byte(60)
			if dir.Female() {
				green = 140
			}
			pic := synth.Sheet256(synth.Sheet256Options{Palette: []color.RGBA{{}, {R: byte(face * 20), G: green, A: 255}}, Frames: []synth.Frame256{{Width: 160, Height: 240, Pixels: pixels}}})
			f.Archives.Containers.Overlay(graphicsPrefix+data.ItemFigureBasePath(dir, face), func() ([]byte, error) { return pic, nil })
		}
	}
	f.Table.NPCCatalog.Faces[8] = data.NPCFace{Kind: data.NPCFigure, Dir: data.FigureDirManFighter, Face: 7, HasWindow: true, WindowX: 40, WindowY: 10}
	s.unitViews = nil
	for _, tc := range []struct {
		name   string
		values []ui.InspectionFieldValue
		red    uint8
		green  uint8
	}{
		{name: "inherited", red: 40, green: 140},
		{name: "face", values: []ui.InspectionFieldValue{{Key: "face", Value: 7}}, red: 140, green: 140},
		{name: "sex", values: []ui.InspectionFieldValue{{Key: "sex", Value: 0}}, red: 40, green: 60},
		{name: "face and sex", values: []ui.InspectionFieldValue{{Key: "face", Value: 7}, {Key: "sex", Value: 0}}, red: 140, green: 60},
		{name: "NPC", values: []ui.InspectionFieldValue{{Key: "identity", Value: namedIdentity(8)}}, red: 140, green: 60},
		{name: "return to inherited", red: 40, green: 140},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, preview, err := s.UnitSettings(0, tc.values)
			if err != nil || preview == nil || preview.Doll == nil {
				t.Fatal("draft has no figure", err)
			}
			want := color.RGBA{R: tc.red, G: tc.green, A: 255}
			if got := preview.Doll.RGBAAt(50, 40); got != want {
				t.Fatalf("figure=%v, want %v", got, want)
			}
			if !bytes.Equal(raw, s.model.Bytes()) || s.CanUndo() {
				t.Fatal("portrait preview wrote map bytes or history")
			}
		})
	}
}
