package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/catmullrom"
)

// drawFinalFrame is the one dispatch the six final blits of a native frame to
// the window share, and the scaler of the zoomed map (blitSmoothMap). op is a
// uniform Scale then Translate, the contract drawSharpBilinear states.
// filterSmooth draws through the Catmull-Rom shader (pkg/render/catmullrom)
// at every scale except exactly 1, where it stays the single FilterNearest
// draw. filterSharp, or a shader that did not compile, runs method B
// (drawSharpBilinear) unchanged. filterPixel is FilterNearest below a scale
// of 2 and method B from 2 up. The destination rectangle is op's in every branch.
// A missing image or op takes the stage of scale 0.
func drawFinalFrame(dst, src *ebiten.Image, op *ebiten.DrawImageOptions, buf **ebiten.Image, filter frameFilter) {
	s := 0.0
	if dst != nil && src != nil && op != nil {
		s = op.GeoM.Element(0, 0)
	}
	switch stageOf(filter, s).window {
	case scaleNone:
	case scaleNearest:
		nearestBlit(dst, src, op)
	case scaleCatmull:
		if !catmullRomBlit(dst, src, op) {
			sharpBilinearBlit(dst, src, op, buf)
		}
	default:
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

// frameFilter is what drawFinalFrame does at a scale other than 1.
// filterSmooth is the Catmull-Rom kernel. filterSharp is method B.
// filterPixel keeps game pixels hard: nearest neighbour below a
// scale of 2, method B from 2 up.
type frameFilter uint8

const (
	filterSmooth frameFilter = iota
	filterSharp
	filterPixel
)

// filterOf chooses the filter from Smoothing and the content's text-safe or pixel policy.
func filterOf(smoothingOff, pixels bool) frameFilter {
	if !smoothingOff {
		return filterSmooth
	}
	if pixels {
		return filterPixel
	}
	return filterSharp
}

func (a *App) frameFilter() frameFilter {
	return filterOf(!a.FrameSmoothing(), a.cutscene != nil)
}

// finalFrameFilter carries the drawing App's or Viewer's filter into
// blitFrameCanvas, blitPointerLayer and blitMenuOverMap, whose signatures
// the op-recording tests replace them by. Each caller sets it from its own
// instance immediately before the call.
var finalFrameFilter frameFilter

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
	crKernel    = make([]float32, 2)
	crUniforms  = map[string]any{"Placement": crPlacement, "Kernel": crKernel}
)

// windowKernel is the kernel the window scaler draws: Catmull-Rom unless the
// launch settings chose another. The CPU reference reads it too.
var windowKernel = catmullrom.CatmullRom

// SetWindowKernel chooses the window scaler's kernel: Mitchell-Netravali
// (B = C = 1/3) when soft is true, else Catmull-Rom.
func SetWindowKernel(soft bool) {
	windowKernel = catmullrom.CatmullRom
	if soft {
		windowKernel = catmullrom.Mitchell
	}
}

func drawCatmullRom(dst, src *ebiten.Image, op *ebiten.DrawImageOptions) bool {
	sh := catmullRomProgram()
	if sh == nil {
		return false
	}
	s := float32(op.GeoM.Element(0, 0))
	crPlacement[0], crPlacement[1] = s, s
	crPlacement[2], crPlacement[3] = float32(op.GeoM.Element(0, 2)), float32(op.GeoM.Element(1, 2))
	crKernel[0], crKernel[1] = float32(windowKernel.B), float32(windowKernel.C)
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

// SetFrameSmoothing sets the shared Smoothing preference, or the local value
// when no preference store is connected.
func (a *App) SetFrameSmoothing(enabled bool) {
	if a == nil {
		return
	}
	if a.flow != nil && a.flow.gameOptions.Write != nil {
		value := 0
		if enabled {
			value = 1
		}
		a.flow.setGameOption(GameOptionSmoothing, value)
		return
	}
	a.frameSmoothingOff = !enabled
	if v := a.flow.viewer; v != nil {
		v.SetFrameSmoothing(enabled)
	}
}

// FrameSmoothing reports the current switch.
func (a *App) FrameSmoothing() bool {
	if a == nil {
		return false
	}
	if a.flow != nil {
		if a.flow.gameOptions.Read != nil || a.flow.gameOptions.Current != nil {
			return a.flow.gameOptionValue(GameOptionSmoothing) != 0
		}
		if v := a.flow.viewer; v != nil {
			return v.graphics.Smoothing
		}
	}
	return !a.frameSmoothingOff
}

// SetFrameSmoothing sets the viewer's Smoothing preference.
func (v *Viewer) SetFrameSmoothing(enabled bool) {
	if v == nil {
		return
	}
	v.graphics.Smoothing = enabled
}
