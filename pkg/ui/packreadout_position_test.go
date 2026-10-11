package ui

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/text"
)

func TestMissionPackReadoutUsesIconLeftAndMappedBottom(t *testing.T) {
	tall := townShellRosterTestFont()
	for i := range tall.Glyphs {
		tall.Glyphs[i].Height = 12
		tall.Glyphs[i].Pixels = make([]text.Pixel, 9*12)
		for p := range tall.Glyphs[i].Pixels {
			tall.Glyphs[i].Pixels[p] = text.Pixel{Level: text.MaxLevel, Painted: true}
		}
	}
	for _, font := range []*text.Font{messageFont(), townShellRosterTestFont(), tall} {
		for _, bar := range []image.Rectangle{image.Rect(0, 390, 480, 480), image.Rect(0, 678, 864, 768), image.Rect(0, 514, 887, 620)} {
			for _, cell := range []struct {
				index int
				count uint32
				purse bool
				text  string
			}{{0, 12, false, "12"}, {2, 4321, true, "4,321"}} {
				t.Run(fmt.Sprintf("height%d-width%d-cell%d-font%d", bar.Dy(), bar.Dx(), cell.index, font.Height()), func(t *testing.T) {
					icon := image.NewRGBA(image.Rect(0, 0, 80, 80))
					s := InventorySubject{Pack: []*image.RGBA{icon, icon, icon}, PackCount: []uint32{1, 1, 1}, PackPurse: []bool{false, false, false}}
					s.PackCount[cell.index], s.PackPurse[cell.index] = cell.count, cell.purse
					var pic *image.RGBA
					calls := text.Record(func() { pic = renderPackBarArt(s, 0, 3, bar, font, nil, -1, nil) })
					var face, shadow []text.DrawCall
					for _, call := range calls {
						if call.Flat {
							shadow = append(shadow, call)
						} else {
							face = append(face, call)
						}
					}
					if len(face) == 0 || len(face) != len(shadow) || recordedText(font, face) != cell.text {
						t.Fatalf("readout = %q, %d face and %d shadow glyphs, want %q", recordedText(font, face), len(face), len(shadow), cell.text)
					}
					pad := ((bar.Dx() + 160 - 240) % 80) / 2
					want := image.Pt(bar.Min.X+pad+32+80*cell.index+3, bar.Dy()-14)
					if got := image.Pt(face[0].X, face[0].Y); got != want {
						t.Fatalf("readout origin = %v, want icon-left+3 and mapped-bottom-14 at %v", got, want)
					}
					facePixels := map[image.Point]color.RGBA{}
					for i, call := range face {
						if call.Color != (color.RGBA{185, 159, 73, 255}) || shadow[i].Color != (color.RGBA{8, 8, 8, 255}) || shadow[i].X != call.X+1 || shadow[i].Y != call.Y+1 {
							t.Fatalf("glyph %d lost the gold ramp or one-pixel flat shadow: face %+v, shadow %+v", i, call, shadow[i])
						}
						for p, pixel := range call.Glyph.Pixels {
							if !pixel.Painted {
								continue
							}
							at := image.Pt(call.X+p%call.Glyph.Width, call.Y+p/call.Glyph.Width)
							if !at.In(call.Clip) || pic.RGBAAt(at.X, at.Y) != text.Shade(call.Color, pixel.Level) {
								t.Fatalf("readout glyph pixel %v was clipped or not painted", at)
							}
							facePixels[at] = text.Shade(call.Color, pixel.Level)
						}
					}
					for _, call := range shadow {
						for p, pixel := range call.Glyph.Pixels {
							if !pixel.Painted {
								continue
							}
							at := image.Pt(call.X+p%call.Glyph.Width, call.Y+p/call.Glyph.Width)
							want := call.Color
							if face, covers := facePixels[at]; covers {
								want = face
							}
							if !at.In(call.Clip) || pic.RGBAAt(at.X, at.Y) != want {
								t.Fatalf("readout shadow pixel %v was clipped or not painted", at)
							}
						}
					}
				})
			}
		}
	}
}

func TestShopAndSchoolCaptionsUseSharedColors(t *testing.T) {
	want := plaqueInk{Rest: color.RGBA{200, 184, 144, 255}, Hover: color.RGBA{255, 255, 224, 255}, Disabled: color.RGBA{104, 98, 89, 255}}
	if plaqueCommandInk != want {
		t.Fatalf("shared caption colors = %+v, want preserved %+v", plaqueCommandInk, want)
	}
	for _, state := range []struct {
		name                     string
		hover, pressed, disabled bool
		ink                      color.RGBA
	}{{"rest", false, false, false, want.Rest}, {"hover", true, false, false, want.Hover}, {"held", true, true, false, want.Hover}, {"disabled", true, true, true, want.Disabled}} {
		t.Run(state.name, func(t *testing.T) {
			shop := ShopScreenView{Words: Words{ShopUndo: "AB"}, Live: [4]bool{!state.disabled}}
			school := TownSurfaceView{Kind: TownSurfaceSchool, Buttons: []TownSurfaceButton{{Label: "AB", Value: "0", Enabled: !state.disabled}}}
			p := shopButtonRects[0].Min.Add(image.Pt(2, 2))
			school.Hover, school.HasHover = TownSurfaceButtonRect(TownSurfaceSchool, 0).Min.Add(image.Pt(2, 2)), state.hover
			if state.pressed {
				shop.Press = ShopControl{Kind: ShopControlButton}
				school.Press = TownSurfaceControl{Kind: TownSurfaceControlButton}
			}
			for _, panel := range []buttonPanel{shopButtonPanel(shop, p, state.hover), townSurfacePanel(school, false, false, false)} {
				button := panel.buttons[0]
				if button.Face.Ink != plaqueCommandInk || button.ink() != state.ink {
					t.Fatalf("caption palette or state differs: face %+v, state %v", button.Face.Ink, button.ink())
				}
				calls := text.Record(func() { drawPushButton(image.NewRGBA(image.Rect(0, 0, 640, 480)), messageFont(), button) })
				if len(calls) == 0 {
					t.Fatal("caption drew no glyphs")
				}
				for _, call := range calls {
					if call.Color != state.ink {
						t.Fatalf("caption glyph color = %v, want %v", call.Color, state.ink)
					}
				}
			}
		})
	}
}
