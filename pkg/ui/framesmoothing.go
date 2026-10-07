package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/catmullrom"
)

// drawFinalFrame is the one dispatch the six final blits of a native frame to
// the window share. op is a uniform Scale then Translate, the contract
// drawSharpBilinear states. With FrameSmoothing on, the default, the source
// is drawn through the Catmull-Rom shader (pkg/render/catmullrom) at every
// scale except exactly 1, where it stays the single FilterNearest draw. With
// it off, or when the shader did not compile, method B (drawSharpBilinear)
// runs unchanged. The destination rectangle is op's in every branch.
func drawFinalFrame(dst, src *ebiten.Image, op *ebiten.DrawImageOptions, buf **ebiten.Image, smoothingOff bool) {
	if smoothingOff {
		sharpBilinearBlit(dst, src, op, buf)
		return
	}
	if dst == nil || src == nil || op == nil {
		return
	}
	scale := op.GeoM.Element(0, 0)
	if scale <= 0 {
		return
	}
	if scale == 1 {
		nearestBlit(dst, src, op)
		return
	}
	if !catmullRomBlit(dst, src, op) {
		sharpBilinearBlit(dst, src, op, buf)
	}
}

// The three paths drawFinalFrame chooses between. They are variables so the
// dispatch tests can record which one each site reached; production never
// replaces them.
var (
	sharpBilinearBlit = drawSharpBilinear
	catmullRomBlit    = drawCatmullRom
	nearestBlit       = func(dst, src *ebiten.Image, op *ebiten.DrawImageOptions) {
		nop := *op
		nop.Filter = ebiten.FilterNearest
		dst.DrawImage(src, &nop)
	}
)

// frameSmoothingOff carries the drawing App's or Viewer's FrameSmoothing
// switch into blitFrameCanvas, blitPointerLayer and blitMenuOverMap, whose
// signatures the op-recording tests replace them by. Each caller sets it from
// its own instance immediately before the call.
var frameSmoothingOff bool

// catmullRomShader is compiled once, on first use. A compile error leaves it
// nil for the process, and every draw then falls back to method B.
var (
	catmullRomShader      *ebiten.Shader
	catmullRomShaderErr   error
	catmullRomShaderTried bool
)

func catmullRomProgram() *ebiten.Shader {
	if !catmullRomShaderTried {
		catmullRomShaderTried = true
		catmullRomShader, catmullRomShaderErr = ebiten.NewShader([]byte(catmullrom.Shader))
	}
	return catmullRomShader
}

// drawCatmullRom draws src's own rectangle under op's GeoM with the shader,
// so every destination pixel whose centre the placed rectangle covers is
// filled, exactly the pixels op's DrawImage fills. It allocates nothing once
// the shader exists. It answers false, having drawn nothing, when the shader
// is unavailable.
// crPlacement is the shader's Placement uniform, rewritten in place each draw
// so the retained map allocates nothing per frame.
var (
	crPlacement = make([]float32, 4)
	crUniforms  = map[string]any{"Placement": crPlacement}
)

func drawCatmullRom(dst, src *ebiten.Image, op *ebiten.DrawImageOptions) bool {
	sh := catmullRomProgram()
	if sh == nil {
		return false
	}
	s := float32(op.GeoM.Element(0, 0))
	crPlacement[0], crPlacement[1] = s, s
	crPlacement[2], crPlacement[3] = float32(op.GeoM.Element(0, 2)), float32(op.GeoM.Element(1, 2))
	var o ebiten.DrawRectShaderOptions
	o.GeoM = op.GeoM
	o.ColorScale = op.ColorScale
	o.Blend = op.Blend
	o.Images[0] = src
	o.Uniforms = crUniforms
	b := src.Bounds()
	dst.DrawRectShader(b.Dx(), b.Dy(), sh, &o)
	return true
}

// SetFrameSmoothing selects the final-frame scaler: on, the default, is the
// Catmull-Rom path; off is method B. It mirrors into the current Viewer.
func (a *App) SetFrameSmoothing(enabled bool) {
	if a == nil {
		return
	}
	a.frameSmoothingOff = !enabled
	if v := a.flow.viewer; v != nil {
		v.SetFrameSmoothing(enabled)
	}
}

// FrameSmoothing reports the current switch.
func (a *App) FrameSmoothing() bool { return a != nil && !a.frameSmoothingOff }

// SetFrameSmoothing selects the mission frame's final scaler, mirroring
// App.SetFrameSmoothing.
func (v *Viewer) SetFrameSmoothing(enabled bool) {
	if v == nil {
		return
	}
	v.frameSmoothingOff = !enabled
}
