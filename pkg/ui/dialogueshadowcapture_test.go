package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"testing"

	"againrom/pkg/render/backdrop"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestDialoguePartialShadowAfterWashRetainsEveryVisibleGlyphCell(t *testing.T) {
	t.Cleanup(text.ResetCapture)
	for _, tc := range []struct {
		name           string
		wash, complete bool
	}{
		{"no-wash-partial", false, false}, {"wash-complete", true, true}, {"wash-partial", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			room := image.NewRGBA(image.Rect(0, 0, 640, 480))
			draw.Draw(room, room.Bounds(), &image.Uniform{C: color.RGBA{40, 80, 120, 255}}, image.Point{}, draw.Src)
			f := dialogueClaimFont()
			f.Glyphs[int('A')-text.FirstChar] = text.Glyph{Width: 2, Height: 1, Advance: 2, Pixels: []text.Pixel{{Level: 15, Painted: true}, {Level: 15, Painted: true}}}
			art := dialogueClaimFrame()
			for _, p := range art.Pieces {
				clear(p.Pix)
			}
			art.Pieces[3].SetRGBA(47, 12, color.RGBA{A: 255})
			if tc.complete {
				art.Pieces[3].SetRGBA(46, 12, color.RGBA{A: 255})
			}
			v := backdropViewer(t)
			layoutViewport(v, 640, 480)
			v.font = f
			l := AuthoredDialogueLayout()
			l.Frame = art
			l.Button, l.Portrait, l.Text = image.Rectangle{}, image.Rectangle{}, image.Rectangle{}
			v.SetNoticeLayouts(l, l)
			v.SetDialogue(Dialogue{})
			text.ResetCapture()
			text.SetCapture(false)
			f.Draw(room, "A", 562, 144, color.RGBA{255, 255, 255, 255})
			v.canvasLog.reset(room.Bounds())
			snapshot := image.NewRGBA(room.Bounds())
			copy(snapshot.Pix, room.Pix)
			v.canvasLog.upload(snapshot)
			if tc.wash {
				wash := color.RGBA{20, 30, 40, 85}
				draw.Draw(room, room.Bounds(), &image.Uniform{C: wash}, image.Point{}, draw.Over)
				v.canvasLog.overSolid(room.Bounds(), wash)
			}
			v.paintNotice(ebiten.NewImageFromImage(room))
			calls := append([]text.DrawCall(nil), text.Captured()...)
			text.StopCapture()
			kept, certain := textsmooth.DecideWith(calls, room.Bounds(), v.canvasLog.oracle())
			if !certain || len(kept) != 1 {
				t.Fatalf("kept=%d certain=%v", len(kept), certain)
			}
			t.Logf("visible source cells=2, kept mask=%v tint=%v raster=%v", kept[0].Mask, kept[0].Tint, kept[0].RasterColors)
			for n := 0; n < 2; n++ {
				expected, ok := v.canvasLog.value(len(v.canvasLog.ops), 562+n, 144)
				if !ok {
					t.Fatal("final pixel log unknown")
				}
				if kept[0].Mask != nil && !kept[0].Mask[n] {
					t.Errorf("visible source cell %d excluded from smoothing; final=%v", n, expected)
				}
				if actual := kept[0].NativeColor(15, n); actual != expected {
					t.Errorf("cell%d overlay color=%v final=%v", n, actual, expected)
				}
			}
		})
	}
}
func TestDialoguePartialShadowAfterWashReachesViewerDraw(t *testing.T) {
	t.Cleanup(text.ResetCapture)
	for _, tc := range []struct {
		name           string
		wash, complete bool
	}{
		{"no-wash-partial", false, false}, {"wash-complete", true, true}, {"wash-partial", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := backdropViewer(t)
			f := dialogueClaimFont()
			for i := range f.Glyphs {
				f.Glyphs[i] = text.Glyph{Width: 2, Height: 1, Advance: 2, Pixels: []text.Pixel{{Level: 15, Painted: true}, {Level: 15, Painted: true}}}
			}
			v.SetFont(f)
			v.readoutLayout.LabelColor = color.RGBA{255, 255, 255, 255}
			v.readoutLayout.ValueColor = color.RGBA{255, 255, 255, 255}
			v.Layout(1024, 768)
			v.ShowReadout(true)
			v.SetTextSmoothing(true)
			v.DeferPointer(true)
			v.Draw(ebiten.NewImage(1024, 768))
			if v.textSettleFallbacks != 0 {
				t.Fatal("unexpected fallback", v.textSettleFallbacks)
			}
			var original text.DrawCall
			found := false
			for _, c := range v.textCalls {
				if c.X < 0 || c.X+2 >= v.cam.ViewW || c.Y >= v.cam.ViewH {
					continue
				}
				if c.Flat || len(c.Under) != 2 || c.Under[0].A != 255 || c.Under[1].A != 255 {
					continue
				}
				if v.canvasLog.verdict(c.X, c.Y, c.NativeColor(15, 0)) == textsmooth.Matches && v.canvasLog.verdict(c.X+1, c.Y, c.NativeColor(15, 1)) == textsmooth.Matches {
					original = c
					found = true
					break
				}
			}
			if !found {
				t.Fatal("no whole visible production HUD glyph")
			}
			art := dialogueClaimFrame()
			for _, p := range art.Pieces {
				clear(p.Pix)
			}
			art.Pieces[3].SetRGBA(47, 12, color.RGBA{A: 255})
			if tc.complete {
				art.Pieces[3].SetRGBA(46, 12, color.RGBA{A: 255})
			}
			l := AuthoredDialogueLayout()
			place := v.noticePlace()
			ox, oy := place.Origin()
			if place.Scale() != 1 {
				t.Fatal("unexpected notice scale", place.Scale())
			}
			at := image.Pt(original.X+1-487, original.Y-20)
			l.Box = image.Rectangle{Min: at.Sub(image.Pt(int(ox), int(oy))), Max: at.Sub(image.Pt(int(ox), int(oy))).Add(image.Pt(488, 232))}
			l.Frame = art
			l.Button, l.Portrait, l.Text = image.Rectangle{}, image.Rectangle{}, image.Rectangle{}
			v.SetNoticeLayouts(l, l)
			v.SetDialogue(Dialogue{})
			wash := color.RGBA{}
			if tc.wash {
				wash = color.RGBA{20, 30, 40, 85}
			}
			v.SetNoticeBackdrop(wash)
			v.Draw(ebiten.NewImage(1024, 768))
			if v.textSettleFallbacks != 0 {
				t.Fatal("unexpected fallback", v.textSettleFallbacks)
			}
			for _, c := range v.textCalls {
				if c.X == original.X && c.Y == original.Y && c.Glyph == original.Glyph && c.Color == original.Color {
					for n := 0; n < 2; n++ {
						fin, k := v.canvasLog.value(len(v.canvasLog.ops), c.X+n, c.Y)
						t.Logf("current source cell%d ink%v under%v log%v/%v raster%v tint%v", n, c.NativeColor(15, n), c.Under[n], fin, k, c.RasterColors, c.Tint)
					}
				}
			}
			kept, certain := textsmooth.DecideWith(v.textCalls, v.canvas.Bounds(), v.canvasLog.oracle())
			t.Logf("whole-frame CPU oracle certain=%v", certain)
			found = false
			for _, c := range kept {
				if c.X != original.X || c.Y != original.Y || c.Glyph != original.Glyph || c.Color != original.Color {
					continue
				}
				found = true
				t.Logf("Viewer.Draw HUD at %d,%d kept mask=%v tint=%v raster=%v; captured%d kept%d", c.X, c.Y, c.Mask, c.Tint, c.RasterColors, v.textCaptured, v.textKept)
				for n := 0; n < 2; n++ {
					expected, ok := v.canvasLog.value(len(v.canvasLog.ops), c.X+n, c.Y)
					if !ok {
						t.Fatal("production pixel unknown")
					}
					if c.Mask != nil && !c.Mask[n] {
						t.Errorf("visible production HUD cell%d omitted from smoothing final%v", n, expected)
					}
					if actual := c.NativeColor(15, n); actual != expected {
						t.Errorf("production HUD cell%d overlay%v final%v", n, actual, expected)
					}
					under := image.NewRGBA(image.Rect(0, 0, 1, 1))
					under.SetRGBA(0, 0, original.Under[n])
					draw.Draw(under, under.Bounds(), &image.Uniform{C: wash}, image.Point{}, draw.Over)
					wantUnder := under.RGBAAt(0, 0)
					if tc.complete || n == 1 {
						word := uint16(int(wantUnder.R)>>3<<11 | int(wantUnder.G)>>2<<5 | int(wantUnder.B)>>3)
						w := frameWordExpectation(word, backdrop.RGB565, backdrop.Full)
						wantUnder = color.RGBA{uint8(int(w>>11) * 255 / 31), uint8(int(w>>5&63) * 255 / 63), uint8(int(w&31) * 255 / 31), 255}
					}
					if c.Under[n] != wantUnder {
						t.Errorf("production HUD cell%d under%v expected%v", n, c.Under[n], wantUnder)
					}
				}
				break
			}
			if !found {
				t.Fatal("whole production HUD glyph lost")
			}
		})
	}
}

func TestDialogueShadowCaptureTintStagesAndLaterWash(t *testing.T) {
	over := func(c, tint color.RGBA) color.RGBA {
		pic := image.NewRGBA(image.Rect(0, 0, 1, 1))
		pic.SetRGBA(0, 0, c)
		draw.Draw(pic, pic.Bounds(), &image.Uniform{C: tint}, image.Point{}, draw.Over)
		return pic.RGBAAt(0, 0)
	}
	clone := func(src *image.RGBA) *image.RGBA {
		dst := image.NewRGBA(src.Bounds())
		copy(dst.Pix, src.Pix)
		return dst
	}
	for _, layout := range []backdrop.Layout{backdrop.RGB565, backdrop.RGB555} {
		for _, mode := range []backdrop.Mode{backdrop.Full, backdrop.Reduced} {
			for _, sourceOver := range []bool{false, true} {
				for stage := 0; stage < 3; stage++ {
					t.Run(fmt.Sprintf("layout%d-mode%d-over%v-prior%d", layout, mode, sourceOver, stage), func(t *testing.T) {
						bounds, clip := image.Rect(0, 0, 5, 1), image.Rect(0, 0, 4, 1)
						room := image.NewRGBA(bounds)
						for x := 0; x < 5; x++ {
							room.SetRGBA(x, 0, color.RGBA{uint8(35 + x*9), uint8(80 + x*5), uint8(123 - x*3), 255})
						}
						under := clone(room)
						g := text.Glyph{Width: 5, Height: 1, Advance: 5, Pixels: []text.Pixel{{Painted: true, Level: 15}, {Painted: true, Level: 15}, {Painted: true, Level: 15}, {Painted: true, Level: 15}, {Painted: true, Level: 15}}}
						font := dialogueClaimFont()
						font.Glyphs[int('A')-text.FirstChar] = g
						prior := color.RGBA{}
						if stage > 0 {
							prior = color.RGBA{9, 16, 22, 48}
						}
						var raster *image.RGBA
						calls := text.Record(func() {
							sub := room.SubImage(clip).(*image.RGBA)
							if sourceOver {
								text.DrawGlyphOver(sub, &g, 0, 0, color.RGBA{76, 59, 30, 128})
							} else {
								font.DrawFlat(sub, "A", 0, 0, color.RGBA{235, 219, 173, 255})
							}
							raster = clone(room)
							draw.Draw(sub, clip, &image.Uniform{C: prior}, image.Point{}, draw.Over)
							text.TintSince(0, clip, prior)
						})
						var log pixelLog
						log.reset(bounds)
						if stage == 2 {
							log.upload(clone(room))
						} else {
							log.upload(raster)
							log.overSolid(clip, prior)
						}
						wash := color.RGBA{}
						if stage == 0 {
							wash = color.RGBA{20, 30, 40, 85}
						}
						draw.Draw(room, bounds, &image.Uniform{C: wash}, image.Point{}, draw.Over)
						log.overSolid(bounds, wash)
						for x := 0; x < 4; x++ {
							under.SetRGBA(x, 0, over(over(under.RGBAAt(x, 0), prior), wash))
						}
						mask := image.NewRGBA(bounds)
						mask.SetRGBA(1, 0, color.RGBA{A: 1})
						mask.SetRGBA(2, 0, color.RGBA{A: 2})
						mask.SetRGBA(4, 0, color.RGBA{A: 1})
						lookup, err := backdrop.NewLevel(layout, mode, 6)
						if err != nil {
							t.Fatal(err)
						}
						packed := func(c color.RGBA) color.RGBA {
							bits := 6
							if layout == backdrop.RGB555 {
								bits = 5
							}
							green := int(c.G) >> (8 - bits)
							word := uint16(int(c.R)>>3<<(5+bits) | green<<5 | int(c.B)>>3)
							w := frameWordExpectation(word, layout, mode)
							return color.RGBA{uint8(int(w>>(5+bits)&31) * 255 / 31), uint8(int(w>>5&((1<<bits)-1)) * 255 / ((1 << bits) - 1)), uint8(int(w&31) * 255 / 31), c.A}
						}
						remapPixels := func(pic *image.RGBA) {
							for x := 0; x < 5; x++ {
								for n := uint8(0); n < mask.RGBAAt(x, 0).A; n++ {
									pic.SetRGBA(x, 0, packed(pic.RGBAAt(x, 0)))
								}
							}
						}
						remapCapturedMask(calls, bounds, lookup, mask, image.Point{}, true, &log)
						log.remapMask(bounds, lookup, mask, image.Point{})
						remapPixels(room)
						remapPixels(under)
						if calls[0].Under[4] != (color.RGBA{}) {
							t.Fatal("clipped underlay changed", calls[0].Under[4])
						}
						cover := color.RGBA{221, 7, 131, 255}
						room.SetRGBA(3, 0, cover)
						under.SetRGBA(3, 0, cover)
						log.overSolid(image.Rect(3, 0, 4, 1), cover)
						beforeLater := clone(room)
						later := color.RGBA{7, 14, 3, 32}
						draw.Draw(room, bounds, &image.Uniform{C: later}, image.Point{}, draw.Over)
						draw.Draw(under, bounds, &image.Uniform{C: later}, image.Point{}, draw.Over)
						log.overSolid(bounds, later)
						check := func(name string, tint color.RGBA, raster *image.RGBA) {
							t.Helper()
							oracle := textsmooth.Oracle{
								Cell: func(x, y int, want color.RGBA) textsmooth.Verdict {
									if room.RGBAAt(x, y) == want {
										return textsmooth.Matches
									}
									return textsmooth.Differs
								},
								Tint: func(x, y int, want color.RGBA) (color.RGBA, textsmooth.Verdict) {
									if raster.RGBAAt(x, y) == want {
										return tint, textsmooth.Matches
									}
									return color.RGBA{}, textsmooth.Differs
								},
							}
							kept, certain := textsmooth.DecideWith(calls, bounds, oracle)
							if !certain || len(kept) != 1 {
								t.Fatalf("%s kept%d certain%v", name, len(kept), certain)
							}
							if len(kept[0].Mask) != 5 || kept[0].Mask[3] || kept[0].Mask[4] {
								t.Fatal("opaque/clip exclusion lost", kept[0].Mask)
							}
							for n := 0; n < 3; n++ {
								if !kept[0].Mask[n] {
									t.Errorf("%s visible cell%d masked", name, n)
								}
								expected := room.RGBAAt(n, 0)
								if got := kept[0].NativeColor(15, n); got != expected {
									t.Errorf("%s native%d=%v expected%v", name, n, got, expected)
								}
								if got, want := kept[0].Under[n], under.RGBAAt(n, 0); got != want {
									t.Errorf("%s under%d=%v expected%v", name, n, got, want)
								}
							}
							for n := 0; n < 3; n++ {
								single := kept[0]
								single.Glyph = &text.Glyph{Width: 1, Height: 1, Pixels: []text.Pixel{{Painted: true, Level: 15}}}
								single.X, single.Y = 0, 0
								single.Clip, single.Mask = image.Rect(0, 0, 1, 1), nil
								single.Under = []color.RGBA{kept[0].Under[n]}
								single.RasterColors = []color.RGBA{kept[0].RasterColors[n]}
								overlay := image.NewRGBA(image.Rect(0, 0, 1, 1))
								textsmooth.Composite(overlay, []text.DrawCall{single}, 1, 0, 0)
								if got, want := overlay.RGBAAt(0, 0), room.RGBAAt(n, 0); got != want {
									t.Errorf("%s constant-field overlay%d=%v expected%v", name, n, got, want)
								}
							}
						}
						check("later wash", later, beforeLater)
						calls[0].Tint = later
						remapCapturedMask(calls, bounds, lookup, mask, image.Point{}, true)
						log.remapMask(bounds, lookup, mask, image.Point{})
						remapPixels(room)
						remapPixels(under)
						check("second shadow", color.RGBA{}, room)
					})
				}
			}
		}
	}
}
