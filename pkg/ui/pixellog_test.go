package ui

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
)

func logTestPic(r image.Rectangle, c color.RGBA) *image.RGBA {
	pic := image.NewRGBA(r)
	draw.Draw(pic, r, image.NewUniform(c), image.Point{}, draw.Src)
	return pic
}

// pixelLogScene composes one frame twice, on the CPU and into a pixelLog: a
// filled canvas holding an uploaded picture with text, laid into a wider
// present, a tooltip with text over part of it, a translucent dim over the
// top and a region the log cannot model. It answers the CPU frame, the log
// and the glyphs captured.
func pixelLogScene(t *testing.T) (*image.RGBA, *pixelLog, []text.DrawCall) {
	t.Helper()
	font := smoothingTestFont()
	room := logTestPic(image.Rect(0, 0, 40, 20), color.RGBA{90, 60, 30, 255})
	tip := logTestPic(image.Rect(0, 0, 10, 6), color.RGBA{250, 240, 200, 255})
	text.ResetCapture()
	text.SetCapture(false)
	for x := 0; x < 36; x += 3 {
		font.Draw(room, "A", x, 2, smoothingTestColor)
		dark := color.RGBA{A: 255}
		if x%2 == 1 {
			dark = color.RGBA{60, 40, 20, 255}
		}
		font.Draw(room, "A", x, 12, dark)
	}
	start := markCapture()
	font.Draw(tip, "AAA", 1, 1, smoothingTestColor)
	shiftCapture(start, image.Pt(20, 0))
	text.StopCapture()
	calls := append([]text.DrawCall(nil), text.Captured()...)

	var canvas, present pixelLog
	canvas.reset(room.Rect)
	canvas.fill(room.Rect, pickerBackground)
	canvas.upload(room)
	canvas.end()
	present.reset(image.Rect(0, 0, 48, 20))
	present.fill(present.bounds, wideFrameFill)
	present.overLayer(&canvas, room.Rect.Add(image.Pt(4, 0)), image.Pt(4, 0))
	present.over(tip, image.Pt(24, 0))
	present.overSolid(image.Rect(0, 8, 48, 20), AuthoredNoticeBackdrop())
	present.unknown(image.Rect(40, 0, 48, 4))
	present.end()

	frame := logTestPic(present.bounds, wideFrameFill)
	draw.Draw(frame, room.Rect.Add(image.Pt(4, 0)), room, image.Point{}, draw.Over)
	draw.Draw(frame, tip.Rect.Add(image.Pt(24, 0)), tip, image.Point{}, draw.Over)
	for y := 8; y < 20; y++ {
		for x := 0; x < 48; x++ {
			out := blendChannels(AuthoredNoticeBackdrop(), frame.RGBAAt(x, y))
			frame.SetRGBA(x, y, color.RGBA{uint8(out[0] + 0.5), uint8(out[1] + 0.5), uint8(out[2] + 0.5), uint8(out[3] + 0.5)})
		}
	}
	for i := range calls {
		calls[i].X += 4
	}
	return frame, &present, calls
}

func TestPixelLogVerdictNeverContradictsTheFrame(t *testing.T) {
	frame, log, _ := pixelLogScene(t)
	for y := frame.Rect.Min.Y; y < frame.Rect.Max.Y; y++ {
		for x := frame.Rect.Min.X; x < frame.Rect.Max.X; x++ {
			c := frame.RGBAAt(x, y)
			if v := log.verdict(x, y, c); v == textsmooth.Differs {
				t.Fatalf("cell %d,%d holds %v, the log says it differs", x, y, c)
			}
			other := c
			other.R ^= 0x40
			if v := log.verdict(x, y, other); v == textsmooth.Matches {
				t.Fatalf("cell %d,%d holds %v, the log says it holds %v", x, y, c, other)
			}
			if x < 40 && y < 8 && log.verdict(x, y, c) != textsmooth.Matches {
				t.Fatalf("cell %d,%d is exactly known, the log cannot tell it", x, y)
			}
		}
	}
}

// TestPixelLogSettleMatchesReadbackSettle is the pixel identity of the
// change: the glyphs the log keeps, and the frame the eraser leaves, equal
// what textsmooth.Settle keeps and leaves from the frame itself.
func TestPixelLogSettleMatchesReadbackSettle(t *testing.T) {
	frame, log, calls := pixelLogScene(t)
	kept, certain := textsmooth.Decide(calls, frame.Rect, log.verdict)
	if !certain {
		t.Fatal("the log could not decide a scene whose unknown region holds no glyph")
	}
	settled := image.NewRGBA(frame.Rect)
	copy(settled.Pix, frame.Pix)
	want := textsmooth.Settle(settled, calls)
	if !sameCalls(kept, want, true) || len(kept) == 0 || len(kept) == len(calls) {
		t.Fatalf("log kept %d glyphs, Settle kept %d of %d; want the same set, neither empty nor all", len(kept), len(want), len(calls))
	}
	eraser := image.NewRGBA(frame.Rect)
	textsmooth.Erase(eraser, kept)
	erased := image.NewRGBA(frame.Rect)
	copy(erased.Pix, frame.Pix)
	draw.Draw(erased, erased.Rect, eraser, image.Point{}, draw.Over)
	if string(erased.Pix) != string(settled.Pix) {
		t.Fatal("the eraser drawn over the frame differs from Settle's erased frame")
	}
	if b := glyphBounds(kept); b.Empty() || !eraserInside(eraser, b) {
		t.Fatalf("the eraser paints outside the kept glyphs' bounds %v", b)
	}
}

func eraserInside(img *image.RGBA, r image.Rectangle) bool {
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			if img.RGBAAt(x, y) != (color.RGBA{}) && !image.Pt(x, y).In(r) {
				return false
			}
		}
	}
	return true
}

func TestPixelLogUnknownCellDefersTheGlyph(t *testing.T) {
	_, log, calls := pixelLogScene(t)
	log.active = true
	log.unknown(image.Rect(0, 0, 48, 20))
	if _, certain := textsmooth.Decide(calls, log.bounds, log.verdict); certain {
		t.Fatal("a glyph under an unknown draw was decided without the frame")
	}
}

func TestPixelLogDrawWhileClosedMakesTheImageUnknown(t *testing.T) {
	var l pixelLog
	pic := logTestPic(image.Rect(0, 0, 4, 4), color.RGBA{1, 2, 3, 255})
	l.begin(pic.Rect)
	l.upload(pic)
	l.end()
	l.over(pic, image.Point{})
	l.begin(pic.Rect)
	if v := l.verdict(1, 1, color.RGBA{1, 2, 3, 255}); v != textsmooth.Unknown {
		t.Fatalf("after an unrecorded draw the log answered %v, want Unknown", v)
	}
}
