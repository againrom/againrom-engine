package ui

import (
	"image"
	"image/color"
	"testing"
)

func TestCharacterPaneDrawsWholeFigureAndKeepsBackground(t *testing.T) {
	background := color.RGBA{R: 7, G: 19, B: 31, A: 255}
	figure := image.NewRGBA(image.Rect(0, 0, 160, 240))
	for y := 0; y < 240; y++ {
		figure.SetRGBA(80, y, color.RGBA{R: 173, G: uint8(y), A: 255})
	}
	for _, origin := range []image.Point{{16, 0}, {480, 238}, {864, 238}} {
		pane := image.Rectangle{Min: origin, Max: origin.Add(image.Pt(160, 242))}
		out := image.NewRGBA(image.Rectangle{Max: pane.Max.Add(image.Pt(1, 1))})
		DrawCharacterPaneBody(out, TownCharacterView{PaneRect: pane, HasSubject: true, Figure: figure,
			FigurePane: TownPane{Body: solidPic(160, 242, background)}})
		for y := 0; y < 240; y++ {
			p := origin.Add(image.Pt(80, y+2))
			if got, want := out.RGBAAt(p.X, p.Y), figure.RGBAAt(80, y); got != want {
				t.Fatalf("pane %v: source row %d = %v, want %v", pane, y, got, want)
			}
			if got := out.RGBAAt(p.X-1, p.Y); got != background {
				t.Fatalf("transparent figure erased background at %v: %v", p, got)
			}
		}
		if got := out.RGBAAt(origin.X+80, pane.Max.Y); got.A != 0 {
			t.Fatal("figure leaked below the pane")
		}
	}
}

func TestCharacterPaneFeetKeepTheirEquipmentMask(t *testing.T) {
	mask := &SlotMask{W: 160, H: 240, Slot: make([]uint8, 160*240)}
	mask.Slot[225*160+80] = 6
	v := shopFullDollTestView(mask)
	if slot, ok := shopDollSlotAt(v, mask, image.Pt(560, 465)); !ok || slot != 5 {
		t.Fatalf("shop foot pixel: slot %d/%v, want 5/true", slot, ok)
	}
	mission, _ := openDollAt(t)
	sub := mission.invSubject
	sub.Figure = solidPic(160, 240, color.RGBA{R: 173, A: 255})
	sub.SlotMask = mask
	mission.SetInventorySubject(sub)
	box := figureBoxFor(t, mission)
	if slot, ok := mission.dollFigureSlotAt(box.Min.X+80, box.Min.Y+227); !ok || slot != 5 {
		t.Fatalf("mission foot pixel: slot %d/%v, want 5/true", slot, ok)
	}
}
