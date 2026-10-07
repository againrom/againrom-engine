package terrain

import (
	"image"
	"image/color"
	"math"
)

func ShadowAngle(theta float64) float64 {
	if math.Abs(theta) < 0.05 {
		if theta < 0 {
			theta = -0.05
		} else {
			theta = 0.05
		}
	}
	return theta * 2 / 3
}

func ShadowSlope(theta float64) float64 {
	return math.Tan(ShadowAngle(theta))
}

// ShadowShear16 is ShadowSlope in the engine's own 16.16 fixed point,
// int(ShadowSlope(theta)*65536) truncated toward zero — Go's own
// float64->int conversion.
func ShadowShear16(theta float64) int {
	return int(ShadowSlope(theta) * 65536)
}

func ShadowRowOffset(slope float64, pivotRow, row int) int {
	return int(slope * float64(pivotRow-row))
}

func ShadowPivotRow(f *StaticFrame) int {
	if f == nil {
		return 0
	}
	return f.Height
}

func ShadowPivotShift(theta float64, frameH, anchorY int) int {
	return int(ShadowSlope(theta) * float64(2*(frameH/2)-anchorY))
}

func UnitShadowPlace(body StaticPlacement, theta float64, airLift int) StaticPlacement {
	frameH := 0
	if body.Frame != nil {
		frameH = body.Frame.Height
	}
	shadow := body
	shadow.TopLeft = body.TopLeft.Add(image.Point{
		X: -ShadowPivotShift(theta, frameH, body.Anchor.Y),
		Y: airLift,
	})
	return shadow
}

func StructureShadowShift(theta float64, fullHeight, gridRow, shadowY int) int {
	return int(ShadowSlope(theta) * float64((fullHeight-gridRow)*CellSize-shadowY))
}

// suppressedShadowY is the hotfix boundary (DIV-068) between a real ShadowY
// — the shipped roster's own pixel heights run 28..55 against a maximum
// FullHeight*CellSize of 192 over all 66 classes (TERR-SHDW-136(c)) — and a
// suppression sentinel: 10000 on 4 classes (three wells and "magic"), 20000
// on 11 (four bridges, four graves, a cave, a teleport, a campfire). 1000
// leaves an order of magnitude of headroom above the largest real value this
// roster carries and a full order of magnitude below the smallest sentinel,
// so a future class with a genuinely tall footprint is not caught by it.
const suppressedShadowY = 1000

func StructureShadowPlace(p StructurePlacement, theta float64) (StructurePlacement, bool) {
	if p.Class == nil || p.Class.VariableSize || p.Frame == nil || p.Class.TileWidth <= 0 || p.Class.ShadowY >= suppressedShadowY {
		return StructurePlacement{}, false
	}
	gridRow := p.GridIndex / p.Class.TileWidth
	shift := StructureShadowShift(theta, p.Class.FullHeight, gridRow, p.Class.ShadowY)
	shadow := p
	shadow.TopLeft = image.Point{X: p.TopLeft.X + shift, Y: p.TopLeft.Y}
	return shadow, true
}

func ObjectShadowPlace(p StaticPlacement, theta float64, originY int) (StaticPlacement, bool) {
	if p.Class == nil || p.Frame == nil || len(p.Class.Frames) == 0 || p.Class.Frames[0] == nil {
		return StaticPlacement{}, false
	}
	f0 := p.Class.Frames[0]
	lift := p.Cell.Y*CellSize + CellSize/2 - p.Anchor.Y - originY - p.TopLeft.Y
	destX, destY, anchorX, anchorY := StaticAnchor(
		p.Cell.X, p.Cell.Y, p.Class.Width, p.Class.Height, p.Class.CenterX, p.Class.CenterY,
		f0.Width, f0.Height, lift, originY)
	shadow := p
	shadow.TopLeft = image.Point{
		X: destX - ShadowPivotShift(theta, f0.Height, anchorY),
		Y: destY,
	}
	shadow.Anchor = image.Point{X: anchorX, Y: anchorY}
	return shadow, true
}

// ShadowKind names which of a Light's two shroud indices a caster's shadow
// reads. It is an enum of two rather than a bool so that a call site reads
// ShadowUnit/ShadowObject rather than a bare true/false that says nothing
// about which is which.
type ShadowKind uint8

const (
	// ShadowUnit selects Light.ShroudUnit — a unit's own shadow band index.
	ShadowUnit ShadowKind = iota
	// ShadowObject selects Light.ShroudObject — the index a structure's
	// shadow and an object's shadow both read.
	ShadowObject
)

// TERR-STRUCT-103
func ShadowLevel(lt Light, kind ShadowKind) int {
	if kind == ShadowUnit {
		return int(lt.ShroudUnit)
	}
	return int(lt.ShroudObject)
}

func ShadowChannel(in uint8, level int) uint8 {
	return uint8((int(in) * (16 - level)) >> 4)
}

// ShadowRGBA is ShadowChannel run over the three colour channels, alpha
// carried unchanged: a shadow recolours what is already at a pixel and never
// adds or removes coverage of its own.
func ShadowRGBA(c color.RGBA, level int) color.RGBA {
	return color.RGBA{
		R: ShadowChannel(c.R, level),
		G: ShadowChannel(c.G, level),
		B: ShadowChannel(c.B, level),
		A: c.A,
	}
}

func ShadowAlpha(level int) uint8 {
	return uint8(math.Round(255 * float64(level) / 16))
}

func ShadowMask(f *StaticFrame) *image.RGBA {
	if f == nil || f.Width <= 0 || f.Height <= 0 || len(f.Pixels) < f.Width*f.Height {
		return image.NewRGBA(image.Rectangle{})
	}
	dst := image.NewRGBA(image.Rect(0, 0, f.Width, f.Height))
	for y := 0; y < f.Height; y++ {
		row := f.Pixels[y*f.Width : (y+1)*f.Width]
		o := dst.PixOffset(0, y)
		for x := 0; x < f.Width; x++ {
			if row[x].Opaque {
				dst.Pix[o+3] = 0xff
			}
			o += 4
		}
	}
	return dst
}

// BlitShadow is the reference statement of the recolour law over a real
// destination: ShadowChannel applied under the frame's own opaque pixels,
// row by row, each row placed at destX+ShadowRowOffset(slope, pivotRow,
// row). Every caster now submits its own non-zero slope and its own pivot
// — the drawn frame's own ShadowPivotRow — so this function shears
// whichever pair it is handed rather than distinguishing a sheared caster
// from an unsheared one itself.
//
// mirror decides which source column a destination column within a row reads —
// f.Width-1-col in place of col — never where a row lands: the same "reflected
// about the vertical centreline of its own rectangle" a mirrored body's draw
// already means for this placement (units.go, "Directions and the mirror"),
// carried into a software walk because this function, unlike BlitStatic, is
// the one place a shadow's own opaque test happens on the CPU rather than at
// the window's draw call.
//
// It reads no source colour: darkening comes only from the destination pixel
// already there, so BlitShadow of one frame at one level over two
// backgrounds differing in a pixel gives two results differing in that
// pixel, and two frames sharing an opaque mask but not a palette give the
// identical result over one background (AC-4).
//
// A nil dst or frame, a non-positive dimension, or a short Pixels slice draws
// nothing, matching blitWalkable's own refusals (blit.go).
func BlitShadow(dst *image.RGBA, f *StaticFrame, destX, destY, level int, slope float64, pivotRow int, mirror bool) {
	if !blitWalkable(dst, f) {
		return
	}
	bounds := dst.Bounds()
	for y := 0; y < f.Height; y++ {
		rowY := destY + y
		offset := ShadowRowOffset(slope, pivotRow, y)
		rowX := destX + offset
		clip := image.Rect(rowX, rowY, rowX+f.Width, rowY+1).Intersect(bounds)
		if clip.Empty() {
			continue
		}
		row := f.Pixels[y*f.Width : (y+1)*f.Width]
		o := dst.PixOffset(clip.Min.X, rowY)
		for x := clip.Min.X; x < clip.Max.X; x++ {
			col := x - rowX
			if mirror {
				col = f.Width - 1 - col
			}
			if row[col].Opaque {
				dst.Pix[o+0] = ShadowChannel(dst.Pix[o+0], level)
				dst.Pix[o+1] = ShadowChannel(dst.Pix[o+1], level)
				dst.Pix[o+2] = ShadowChannel(dst.Pix[o+2], level)
			}
			o += 4
		}
	}
}
