package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"againrom/pkg/render/text"
)

// The frame-rate readout, toggled by F12 and off at load (MENU-060). The box is
// 90x24 at (width-120, 0) and the text is right-aligned at width-38 from y 0
// (MENU-081).
const (
	fpsBoxLeft   = 0x78
	fpsBoxRight  = 0x1e
	fpsBoxHeight = 0x18
	fpsTextRight = 0x26
	fpsRecompute = 1000
)

var (
	// The box is the value 8 on every channel; the text is the white ramp's top
	// entry over the dialogue shadow at +1,+1.
	fpsBoxInk  = color.RGBA{8, 8, 8, 255}
	fpsTextInk = color.RGBA{255, 255, 255, 255}
)

// fpsMeter samples draw calls over elapsed milliseconds (MENU-081/083).
type fpsMeter struct {
	calls   int
	elapsed int64
	last    int64
	seen    bool
	value   float64
}

// frame counts one draw call at clock reading ms and returns the rate.
func (m *fpsMeter) frame(ms int64) float64 {
	m.calls++
	if m.seen {
		m.elapsed += ms - m.last
	}
	m.last, m.seen = ms, true
	if m.elapsed > fpsRecompute {
		m.value = float64(m.calls) * 1000 / float64(m.elapsed)
		m.elapsed -= fpsRecompute
		m.calls = 0
	}
	return m.value
}

// fpsKey is everything the composed readout picture is a function of.
type fpsKey struct {
	tenths int
	area   image.Point
}

// ToggleFPS flips the frame-rate readout and reports where it landed.
func (v *Viewer) ToggleFPS() bool {
	v.fpsShown = !v.fpsShown
	return v.fpsShown
}

// FPSShown reports whether the frame-rate readout is on.
func (v *Viewer) FPSShown() bool { return v.fpsShown }

// FPSLine is the text the readout states for a frame rate: one decimal, at
// least three characters wide, then the word fps.
func FPSLine(fps float64) string { return fmt.Sprintf("%3.1f fps", fps) }

// FPSReadout samples one draw and composes its readout, including text capture.
func (v *Viewer) FPSReadout(ms int64) (rate float64, pic *image.RGBA, at image.Point, shown bool) {
	rate = v.fps.frame(ms)
	start := beginTextCapture(v.textSmoothingEnabled)
	pic, at, shown = v.fpsPresent(rate)
	endTextCapture(start, at)
	return
}

// fpsBoxRect is the readout's box in view pixels for a view viewW wide.
func fpsBoxRect(viewW int) image.Rectangle {
	return image.Rect(viewW-fpsBoxLeft, 0, viewW-fpsBoxRight, fpsBoxHeight)
}

// fpsPresent caches the visible readout by shown tenth and view size.
func (v *Viewer) fpsPresent(fps float64) (*image.RGBA, image.Point, bool) {
	if !v.fpsShown || v.font == nil {
		return nil, image.Point{}, false
	}
	box := fpsBoxRect(v.cam.ViewW)
	if box.Min.X < 0 || box.Dx() <= 0 {
		return nil, image.Point{}, false
	}
	if fps < 0 {
		fps = 0
	}
	key := fpsKey{tenths: int(fps*10 + 0.5), area: image.Pt(v.cam.ViewW, v.cam.ViewH)}
	if v.fpsPic == nil || key != v.fpsKey {
		pic := image.NewRGBA(image.Rect(0, 0, box.Dx(), box.Dy()))
		draw.Draw(pic, pic.Bounds(), image.NewUniform(fpsBoxInk), image.Point{}, draw.Src)
		v.fpsText = text.Record(func() {
			line := FPSLine(float64(key.tenths) / 10)
			x := box.Dx() - (fpsTextRight - fpsBoxRight) - v.font.Advance(line)
			v.font.DrawFlat(pic, line, x+dialogueTextShadow, dialogueTextShadow, messageShadowColor)
			v.font.Draw(pic, line, x, 0, fpsTextInk)
		})
		v.fpsPic, v.fpsFresh, v.fpsKey = pic, true, key
	}
	text.Append(v.fpsText, 0, 0)
	return v.fpsPic, box.Min, true
}
