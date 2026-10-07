package terrain

import "math"

// Light holds the terrain relief-lighting parameters. The daytime defaults are
// the sun globals the engine actually feeds the lighting math at day; a caller
// MAY override them (from a map's stored fields or flags), which per
// TERR-LIGHT-023 is a viewer choice, not engine fidelity.
//
// ShroudObject and ShroudUnit are the per-band shadow/silhouette indices
// skylight.go's schedule carries: the object-path and unit-path selectors
// the shadow pass would key on. Nothing in this tree draws a shadow yet, and
// nothing outside SkyLight and SunAt reads either field -- they exist so
// that pass inherits them from the one place the schedule is written, rather
// than a second copy of it appearing when that pass is built.
type Light struct {
	Theta        float64  // sun angle, radians (enters as cos|Theta| and tan|Theta|)
	Ambient      uint8    // ambient intensity byte (the level's base term)
	Range        uint8    // directional swing byte (the level's slope term)
	SkyTint      [3]uint8 // sky RGB tint, added per channel before the shade multiply
	ShroudObject uint8    // the band's object-path shroud/silhouette index
	ShroudUnit   uint8    // the band's unit-path shroud/silhouette index
}

const (
	cellPitch    = 32.0 // _L10228: horizontal step per cell
	finalScale   = 0.5  // _L10230: 0.5 scale on the axis average
	axisClampMax = 95.0 // _L05651: per-axis clamp maximum
	levelBias    = 0x20 // the +0x20 base added into L
	// LevelCount is the number of attenuation rows: the [0,95] clamp of the
	// per-vertex byte, so 96 (TERR-LIGHT-018).
	LevelCount = 96

	DefaultTheta = 0.78539815
)

// baseIncidence is pi/6 (_L10229), the base incidence angle.
var baseIncidence = math.Pi / 6

// DefaultDaytime is the engine's cycle-off daytime sun: the DefaultTheta
// angle literal, the day ambient 0x0e and range 0x20, and the daytime sky
// tint (0,0,0). Every map is lit identically at this default; flat ground
// levels to 46 at any theta (TERR-LIGHT-013/020/030). The shroud indices are
// the cycle-off arm's, 4 and 2 (TERR-LIGHT-119) -- so a Light built from
// this constant and one built from SkyLight's cycle-off arm stay equal,
// which skylight_test.go's AC-3 compares.
var DefaultDaytime = Light{
	Theta:        DefaultTheta,
	Ambient:      0x0e,
	Range:        0x20,
	SkyTint:      [3]uint8{0, 0, 0},
	ShroudObject: 4,
	ShroudUnit:   2,
}

// LightFromFields builds a Light from a map's stored type-0 fields, reading
// them as their file types: the +0x08 float as the angle, and the low bytes
// of the +0x10/+0x14 scalars as ambient/range (the engine's P+0x1c/0x1d byte
// slots, TERR-LIGHT-023). The sky tint stays (0,0,0) (the daytime tint; the
// dawn/dusk schedule is out of scope). This is an explicit viewer choice,
// not a fidelity claim: the engine overwrites these fields before use
// (TERR-LIGHT-023).
func LightFromFields(angle float32, ambient, rng uint32) Light {
	return Light{
		Theta:   float64(angle),
		Ambient: uint8(ambient),
		Range:   uint8(rng),
		SkyTint: [3]uint8{0, 0, 0},
	}
}

// LevelGrid computes the W*H per-vertex relief level grid from a height field and
// a light, by the exact TERR-LIGHT-013 formula with the TERR-LIGHT-028 gradient.
// Each level is a byte in [0,95].
//
// The gradient is TWO ADJACENT ONE-CELL DIFFERENCES ALONG THE SAME (row) AXIS: a
// forward one to row y+1 and a backward one from row y-1, each laterally
// interpolated within its own row by tan|theta|, each converted to a level
// independently, and only then averaged.
//
//	tanT  = tan|theta| ; stepH = 32.0 / cos|theta|
//	hA    = H[x -+ tanT, y+1]                        forward  row, fractional column
//	hB    = H[x -+ tanT, y-1]                        backward row, fractional column
//	slope = atan2(delta, stepH)                      delta = hA-H[x,y], then H[x,y]-hB
//	axis  = clamp(L - R*sin(pi/6 - slope), 0, 95)    per half, BEFORE averaging
//	level = ftol(0.5 * (axis1 + axis2))              truncating toward zero
//
// with R = Range and L = (Range>>1) + Ambient + 0x20. The fractional column is
// formed from the vertex's own column and its single lateral neighbour:
// H[x,y] - (H[x,y] - H[x-1,y])*tanT for theta >= 0, and
// H[x,y] + (H[x+1,y] - H[x,y])*tanT for theta < 0.
//
// There is NO two-cell central difference and NO perpendicular x gradient in the
// engine: the only x+-1 reads are the lateral operands of the tan|theta| shear
// inside rows y+1 and y-1, so the sample walks along the sun azimuth while stepH
// lengthens the baseline to match. A port using a central difference
// over-contrasts, roughly doubling every gradient. The azimuth therefore enters
// twice, and the shear is not decorative: removing it moves the 38-map corpus
// level ceiling from 70 to 66.
//
// The single 0.5 multiplies two ALREADY-CLAMPED levels, never a height
// difference, so the routine equals no single-difference form once either half
// clamps. That ordering is why the clamp lives inside axisValue and the scale
// outside it.
//
// Heights are read as SIGNED bytes (the engine's MOVSX, TERR-LIGHT-028): a byte
// >= 0x80 counts as negative. That is unobservable on the shipped corpus (0 of
// 880 704 height bytes reach 0x80) but it is the decoded behaviour, so it is
// transcribed rather than smoothed.
//
// The backward half's lateral column is selected by y == 1, not by the sign of
// theta: the engine computes a theta compare and then discards it (DEC EDI
// overwrites the flags before the branch), so the emitted semantics branch on the
// row. That is transcribed as executed. Whether the original source intended the
// theta test is an inference the research explicitly declines to make, and we
// make it no more strongly; the effect is confined to row 1 and was measured as
// nil on the corpus level range.
//
// The formula's own computed region is the strict interior W-2 x H-2 (it
// reads x+-1 and y+-1). The game computes only that and leaves the outer
// vertex ring uncomputed -- no one-sided fallback, no clamp, no edge fill,
// and the allocator does not zero it (TERR-EDGE-025). We still owe callers a
// total, index-safe grid, so every height index is clamped into [0,W-1] x
// [0,H-1] and the same formula is applied. On the top/bottom edge that
// degenerates the forward or backward half to a purely lateral difference;
// on the left/right edge the shear operand collapses and the shear vanishes;
// a degenerate 1xN or Nx1 grid yields zero deltas. That ring fill is OUR
// defined choice for a region the game defines no value for, not a
// transcription of engine behaviour -- and a flat grid still levels to 46
// everywhere, ring included.
//
// The float slope math is the game's own FP model; the result is a byte and
// no float reaches a which-graphic decision.
//
// A nil slice is returned for a non-positive dimension or a height slice shorter
// than W*H, so a caller must supply a full grid; the compositor validates before
// calling.
func LevelGrid(heights []uint8, w, h int, lt Light) []uint8 {
	if w < 1 || h < 1 || len(heights) < w*h {
		return nil
	}

	// L and R are integer combinations of the light bytes; keep them exact.
	L := float64(int(lt.Range)>>1 + int(lt.Ambient) + levelBias)
	R := float64(lt.Range)
	absTheta := math.Abs(lt.Theta)
	stepH := cellPitch / math.Cos(absTheta)
	tanT := math.Tan(absTheta)

	clampIdx := func(v, hi int) int {
		if v < 0 {
			return 0
		}
		if v > hi {
			return hi
		}
		return v
	}
	// height reads a SIGNED height byte with both indices clamped into the grid.
	height := func(x, y int) float64 {
		return float64(int8(heights[clampIdx(y, h-1)*w+clampIdx(x, w-1)]))
	}
	// axisValue turns one one-cell height delta into a clamped level, exactly as
	// the engine does before the average. The NaN arm is ours: it keeps the
	// function total for a hostile theta whose cos/tan is not a number, rather
	// than leaving an undefined float->byte conversion.
	axisValue := func(delta float64) float64 {
		slope := math.Atan2(delta, stepH)
		v := L - R*math.Sin(baseIncidence-slope)
		if math.IsNaN(v) {
			return 0
		}
		if v < 0 {
			return 0
		}
		if v > axisClampMax {
			return axisClampMax
		}
		return v
	}

	out := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			h0 := height(x, y)

			// Forward half: one row to y+1, sheared laterally by tanT.
			hb := height(x, y+1)
			var hA float64
			if lt.Theta >= 0 {
				hA = hb - (hb-height(x-1, y+1))*tanT
			} else {
				hA = hb + (height(x+1, y+1)-hb)*tanT
			}

			// Backward half: one row from y-1, mirrored laterally on row y == 1.
			var hB float64
			if y == 1 {
				ha := height(x, 0)
				hB = ha + (height(x+1, 0)-ha)*tanT
			} else {
				ha := height(x, y-1)
				hB = ha - (ha-height(x-1, y-1))*tanT
			}

			level := finalScale * (axisValue(hA-h0) + axisValue(h0-hB))
			// level lies in [0,95]; the float->int conversion truncates toward
			// zero, matching the game's ftol for the non-negative average.
			out[y*w+x] = uint8(int(level))
		}
	}
	return out
}
