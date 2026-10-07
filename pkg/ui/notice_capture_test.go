package ui

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed testdata/notice-alpha.json
var noticeCaptureAlpha []byte

func noticeCaptureFont() *text.Font {
	glyphs := make([]text.Glyph, 'A'-text.FirstChar+1)
	for i := range glyphs {
		glyphs[i] = text.Glyph{Width: 3, Height: 3, Advance: 4}
	}
	glyphs['A'-text.FirstChar].Pixels = make([]text.Pixel, 9)
	for _, i := range []int{1, 3, 4, 5, 7} {
		glyphs['A'-text.FirstChar].Pixels[i] = text.Pixel{Level: text.MaxLevel, Painted: true}
	}
	return &text.Font{Glyphs: glyphs}
}

func noticeCaptureLayout(disabled bool) NoticeLayout {
	return NoticeLayout{Box: image.Rect(30, 120, 50, 138), Text: image.Rect(3, 2, 17, 8),
		Button: image.Rect(2, 10, 10, 17), SecondaryButton: image.Rect(11, 10, 19, 17),
		ButtonLabel: "A", SecondaryButtonLabel: "A", SecondaryDisabled: disabled,
		Fill: color.RGBA{40, 50, 60, 255}, Border: color.RGBA{80, 90, 100, 255},
		ButtonFill: color.RGBA{20, 30, 40, 255}, ButtonBorder: color.RGBA{70, 80, 90, 255},
		TextColor: color.RGBA{255, 255, 255, 255}, ButtonText: color.RGBA{255, 255, 255, 255}}
}

func noticeCaptureExpected(disabled, ink bool) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, 20, 18))
	for y := 0; y < 18; y++ {
		for x := 0; x < 20; x++ {
			c := color.RGBA{40, 50, 60, 255}
			if x == 0 || x == 19 || y == 0 || y == 17 {
				c = color.RGBA{80, 90, 100, 255}
			}
			for _, left := range []int{2, 11} {
				if x >= left && x < left+8 && y >= 10 && y < 17 {
					c = color.RGBA{20, 30, 40, 255}
					if x == left || x == left+7 || y == 10 || y == 16 {
						c = color.RGBA{70, 80, 90, 255}
					}
				}
			}
			out.SetRGBA(x, y, c)
		}
	}
	if ink {
		for i, at := range []image.Point{{4, 12}, {13, 12}, {3, 2}} {
			c := color.RGBA{255, 255, 255, 255}
			if disabled && i == 1 {
				c = color.RGBA{128, 128, 128, 255}
			}
			for _, d := range []image.Point{{1, 0}, {0, 1}, {1, 1}, {2, 1}, {1, 2}} {
				out.SetRGBA(at.X+d.X, at.Y+d.Y, c)
			}
		}
	}
	return out
}

// noticeCalls keeps glyph origins inside the 20x18 notice box. Other HUD
// widgets also enter the frame capture; these tests isolate the notice.
func noticeCalls(calls []text.DrawCall, at image.Point) []text.DrawCall {
	box := image.Rect(at.X, at.Y, at.X+20, at.Y+18)
	var out []text.DrawCall
	for _, c := range calls {
		if image.Pt(c.X, c.Y).In(box) {
			out = append(out, c)
		}
	}
	return out
}

func noticeCapturePNG(t *testing.T, name string, img *image.RGBA) {
	t.Helper()
	root := os.Getenv("AGAINROM_NOTICE_CAPTURE_OUTPUT")
	if root == "" {
		return
	}
	if !filepath.IsAbs(root) {
		t.Fatal("explicit capture output required")
	}
	f, err := os.Create(filepath.Join(root, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestMissionNoticeCaptureDraw(t *testing.T) {
	raw := noticeCaptureAlpha
	var golden map[string][]uint8
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatal(err)
	}
	for _, size := range []image.Point{{1024, 768}, {1920, 1080}, {2048, 1536}} {
		for _, disabled := range []bool{false, true} {
			t.Run(fmt.Sprintf("%dx%d/disabled=%t", size.X, size.Y, disabled), func(t *testing.T) {
				v := noticeViewer(t)
				v.ShowReadout(false)
				v.DeferPointer(true)
				font := noticeCaptureFont()
				v.SetFont(font)
				layout := noticeCaptureLayout(disabled)
				v.SetNoticeLayouts(layout, layout)
				v.Layout(size.X, size.Y)
				v.SetNotice("A", NoticeDialogue)
				v.SetTextSmoothing(true)
				v.noticePresent()
				builds := v.noticeBuilds
				rec := recordBlits(t)
				var at image.Point
				for frame := 0; frame < 2; frame++ {
					text.ResetCapture()
					v.Draw(ebiten.NewImage(size.X, size.Y))
					at = image.Pt((v.cam.ViewW-640)/2+30, (v.cam.ViewH-480)/2+120)
					all := text.Captured()
					calls := noticeCalls(all, at)
					if len(calls) != 3 || v.textCaptured != len(all) || v.textKept != 3 || v.textSettleFallbacks != 0 {
						t.Fatalf("NOTICE-CAPTURE-MISSING frame=%d captured=%d of %d stats=%d/%d fallback=%d", frame, len(calls), len(all), v.textCaptured, v.textKept, v.textSettleFallbacks)
					}
					if v.noticeBuilds != builds {
						t.Fatal("unchanged notice recomposed instead of replaying")
					}
					if text.Capturing() {
						t.Fatal("notice capture window leaked")
					}
					points := []image.Point{{4, 12}, {13, 12}, {3, 2}}
					blank := noticeCaptureExpected(disabled, false)
					for i, call := range calls {
						want := points[i].Add(at)
						c := color.RGBA{255, 255, 255, 255}
						if disabled && i == 1 {
							c = color.RGBA{128, 128, 128, 255}
						}
						if call.Glyph != &font.Glyphs['A'-text.FirstChar] || call.X != want.X || call.Y != want.Y || call.Color != c || len(call.Under) != 9 {
							t.Fatalf("capture %d wrong: %+v want=%v %v", i, call, want, c)
						}
						for _, p := range []int{1, 3, 4, 5, 7} {
							if call.Under[p] != blank.RGBAAt(points[i].X+p%3, points[i].Y+p/3) {
								t.Fatal("wrong underlying art", i, p)
							}
						}
					}
					if !bytes.Equal(v.noticePic.Pix, noticeCaptureExpected(disabled, true).Pix) {
						t.Fatal("native notice pixels changed")
					}
					erased := image.NewRGBA(image.Rect(0, 0, v.frameW, v.frameH))
					for _, p := range points {
						for _, d := range []image.Point{{1, 0}, {0, 1}, {1, 1}, {2, 1}, {1, 2}} {
							local := p.Add(d)
							cell := local.Add(at)
							erased.SetRGBA(cell.X, cell.Y, blank.RGBAAt(local.X, local.Y))
						}
					}
					if v.textEraser.buf == nil || !bytes.Equal(v.textEraser.buf.Pix, erased.Pix) {
						t.Fatal("baked glyph eraser differs from independent native background")
					}
					logicalW := max(1024, int(math.Ceil(float64(size.X)*768/float64(size.Y))))
					scale := math.Min(float64(size.X)/float64(logicalW), float64(size.Y)/768)
					ox, oy := (float64(size.X)-float64(logicalW)*scale)/2, (float64(size.Y)-768*scale)/2
					if rec.geom.Element(0, 0) != scale || rec.geom.Element(1, 1) != scale {
						t.Fatal("final frame scale changed")
					}
					expected := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
					for i, p := range points {
						p = p.Add(at)
						x := int(math.Round(ox + float64(p.X)*scale))
						y := int(math.Round(oy + float64(p.Y)*scale))
						dx := int(math.Round(ox+float64(p.X+3)*scale)) - x
						dy := int(math.Round(oy+float64(p.Y+3)*scale)) - y
						alpha, ok := golden[fmt.Sprintf("%dx%d", dx, dy)]
						if !ok || len(alpha) != dx*dy {
							t.Fatal("missing independently computed alpha grid", dx, dy)
						}
						for j, a := range alpha {
							rgb := a
							if disabled && i == 1 {
								rgb = uint8(uint32(a) * 128 / 255)
							}
							expected.SetRGBA(x+j%dx, y+j/dx, color.RGBA{rgb, rgb, rgb, a})
						}
					}
					if v.textOverlay.buf == nil || len(v.textOverlay.buf.Pix) != len(expected.Pix) {
						t.Fatal("output glyph overlay absent")
					}
					for i, want := range expected.Pix {
						got := v.textOverlay.buf.Pix[i]
						if int(got)-int(want) > 1 || int(want)-int(got) > 1 {
							t.Fatalf("independent overlay pixel differs at byte %d: %d/%d", i, got, want)
						}
					}
					if frame == 0 {
						name := fmt.Sprintf("%dx%d-disabled-%t", size.X, size.Y, disabled)
						noticeCapturePNG(t, name+"-native", v.noticePic)
						noticeCapturePNG(t, name+"-overlay", v.textOverlay.buf)
					}
					cpu := image.NewRGBA(image.Rect(0, 0, v.frameW, v.frameH))
					for y := 0; y < 18; y++ {
						for x := 0; x < 20; x++ {
							cpu.SetRGBA(at.X+x, at.Y+y, v.noticePic.RGBAAt(x, y))
						}
					}
					cpu.SetRGBA(at.X+4, at.Y+2, color.RGBA{1, 2, 3, 255})
					kept := textsmooth.Settle(cpu, calls)
					if len(kept) != 3 || len(kept[2].Mask) != 9 || kept[2].Mask[1] {
						t.Fatal("partly covered body did not keep only its showing cells")
					}
					for _, i := range []int{3, 4, 5, 7} {
						if !kept[2].Mask[i] {
							t.Fatalf("showing body cell %d was dropped", i)
						}
					}
					for y := range 18 {
						for x := range 20 {
							cpu.SetRGBA(at.X+x, at.Y+y, v.noticePic.RGBAAt(x, y))
						}
					}
					for _, i := range []int{1, 3, 4, 5, 7} {
						cpu.SetRGBA(at.X+3+i%3, at.Y+2+i/3, color.RGBA{1, 2, 3, 255})
					}
					if got := textsmooth.Settle(cpu, calls); len(got) != 2 {
						t.Fatal("wholly covered body glyph survived", len(got))
					}
				}
				v.SetNotice("AA", NoticeDialogue)
				text.ResetCapture()
				v.Draw(ebiten.NewImage(size.X, size.Y))
				if len(noticeCalls(text.Captured(), at)) != 4 || v.textKept != 4 || v.noticeBuilds != builds+1 {
					t.Fatal("text change did not replace cached glyphs")
				}
				v.SetNotice("A", NoticeDialogue)
				text.ResetCapture()
				v.Draw(ebiten.NewImage(size.X, size.Y))
				if len(noticeCalls(text.Captured(), at)) != 3 || v.textKept != 3 || v.noticeBuilds != builds+2 {
					t.Fatal("text restore replayed stale glyphs")
				}
				v.SetTextSmoothing(false)
				text.ResetCapture()
				v.Draw(ebiten.NewImage(size.X, size.Y))
				if text.CapturedLen() != 0 || !bytes.Equal(v.noticePic.Pix, noticeCaptureExpected(disabled, true).Pix) {
					t.Fatal("off control changed native pixels or captured")
				}
				v.SetTextSmoothing(true)
				v.ClearNotice()
				text.ResetCapture()
				v.Draw(ebiten.NewImage(size.X, size.Y))
				if len(noticeCalls(text.Captured(), at)) != 0 {
					t.Fatal("closed notice replayed stale glyphs")
				}
				v.SetNotice("A", NoticeDialogue)
				v.SetFont(nil)
				text.ResetCapture()
				v.Draw(ebiten.NewImage(size.X, size.Y))
				if len(noticeCalls(text.Captured(), at)) != 0 {
					t.Fatal("fontless notice captured")
				}
			})
		}
	}
}

func TestMissionNoticeCaptureClipped(t *testing.T) {
	v := noticeViewer(t)
	v.ShowReadout(false)
	v.DeferPointer(true)
	font := noticeCaptureFont()
	v.SetFont(font)
	v.Layout(1024, 768)
	l := noticeCaptureLayout(false)
	l.Text = image.Rect(19, 2, 23, 8)
	v.SetNoticeLayouts(l, l)
	v.SetNotice("A", NoticeDialogue)
	v.SetTextSmoothing(true)
	recordBlits(t)
	text.ResetCapture()
	v.Draw(ebiten.NewImage(1024, 768))
	at := image.Pt((v.cam.ViewW-640)/2+30, (v.cam.ViewH-480)/2+120)
	calls := noticeCalls(text.Captured(), at)
	if len(calls) != 3 || v.textKept != 3 || v.textSettleFallbacks != 0 {
		t.Fatalf("CLIPPED-CAPTURE-MISSING capture=%d kept=%d fallback=%d", len(calls), v.textKept, v.textSettleFallbacks)
	}
	body := calls[2]
	if body.X != at.X+19 || body.Y != at.Y+2 || body.Glyph != &font.Glyphs['A'-text.FirstChar] || len(body.Under) != 9 {
		t.Fatal("wrong clipped body capture")
	}
	blank := noticeCaptureExpected(false, false)
	expected := noticeCaptureExpected(false, false)
	white := color.RGBA{255, 255, 255, 255}
	for _, p := range []image.Point{{4, 12}, {13, 12}, {19, 2}} {
		for _, d := range []image.Point{{1, 0}, {0, 1}, {1, 1}, {2, 1}, {1, 2}} {
			cell := p.Add(d)
			expected.SetRGBA(cell.X, cell.Y, white)
		}
	}
	if !bytes.Equal(v.noticePic.Pix, expected.Pix) {
		t.Fatal("clipped native picture differs from literal pixels")
	}
	for _, i := range []int{1, 3, 4, 5, 7} {
		want := blank.RGBAAt(19+i%3, 2+i/3)
		if body.Under[i] != want {
			t.Fatalf("clipped Under[%d]=%v want=%v", i, body.Under[i], want)
		}
	}
	if body.Under[1] != (color.RGBA{}) || body.Under[3].A != 255 {
		t.Fatal("clipping control lacks both outside and inside painted cells")
	}
	queries := 0
	kept, certain := textsmooth.Decide(calls[2:], v.canvas.Bounds(), func(x, y int, want color.RGBA) textsmooth.Verdict { queries++; return textsmooth.Unknown })
	if certain || len(kept) != 0 || queries != 1 {
		t.Fatalf("clipped glyph did not query its one showing cell: certain=%t kept=%d queries=%d", certain, len(kept), queries)
	}
	kept, certain = textsmooth.Decide(calls[:1], v.canvas.Bounds(), func(x, y int, want color.RGBA) textsmooth.Verdict { return textsmooth.Unknown })
	if certain || len(kept) != 0 {
		t.Fatal("unknown full glyph was accepted without readback")
	}
	erased := image.NewRGBA(image.Rect(0, 0, v.frameW, v.frameH))
	for _, p := range []image.Point{{4, 12}, {13, 12}, {19, 2}} {
		for _, d := range []image.Point{{1, 0}, {0, 1}, {1, 1}, {2, 1}, {1, 2}} {
			local := p.Add(d)
			cell := local.Add(at)
			erased.SetRGBA(cell.X, cell.Y, blank.RGBAAt(local.X, local.Y))
		}
	}
	if v.textEraser.buf == nil || !bytes.Equal(v.textEraser.buf.Pix, erased.Pix) {
		t.Fatal("showing clipped body cell or button background was not erased")
	}
	raw := noticeCaptureAlpha
	var golden map[string][]uint8
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatal(err)
	}
	alpha := golden["3x3"]
	if len(alpha) != 9 {
		t.Fatal("missing independent native-scale grid")
	}
	overlay := image.NewRGBA(image.Rect(0, 0, 1024, 768))
	for _, p := range []image.Point{{4, 12}, {13, 12}} {
		p = p.Add(at)
		for i, a := range alpha {
			overlay.SetRGBA(p.X+i%3, p.Y+i/3, color.RGBA{a, a, a, a})
		}
	}
	for row := range 3 {
		a := alpha[row*3]
		overlay.SetRGBA(at.X+19, at.Y+2+row, color.RGBA{a, a, a, a})
	}
	if v.textOverlay.buf == nil || len(v.textOverlay.buf.Pix) != len(overlay.Pix) {
		t.Fatal("clipped control overlay absent")
	}
	for i, want := range overlay.Pix {
		got := v.textOverlay.buf.Pix[i]
		if int(got)-int(want) > 1 || int(want)-int(got) > 1 {
			t.Fatalf("clipped overlay differs at byte %d: %d/%d", i, got, want)
		}
	}
	noticeCapturePNG(t, "clipped-native", v.noticePic)
	noticeCapturePNG(t, "clipped-overlay", v.textOverlay.buf)
}

func TestMissionNoticeCaptureSmall(t *testing.T) {
	v := noticeViewer(t)
	v.SetFont(noticeCaptureFont())
	l := noticeCaptureLayout(false)
	v.SetNoticeLayouts(l, l)
	v.SetNotice("A", NoticeDialogue)
	layoutViewport(v, 320, 240)
	v.noticePresent()
	text.ResetCapture()
	start := beginTextCapture(true)
	_, at, scale, ok := v.noticePresent()
	endTextCapture(start, at)
	if !ok || scale != 0.5 || text.CapturedLen() != 0 {
		t.Fatalf("small notice=%t scale=%g captures=%d", ok, scale, text.CapturedLen())
	}
}
