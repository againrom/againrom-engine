package game

import (
	"math"
	"math/big"
)

// The Lightning and Prismatic Spray figure, pass for pass as the original
// builds it (MAGIC-275..278, MAGIC-282). Inputs are integer display points;
// output is the stored 8-byte point list. Every arithmetic result is rounded
// to binary64 at its own instruction: each product is wrapped in an explicit
// float64 conversion, which forbids a fused multiply-add on any target. The
// native entry control word is Unknown; binary64 nearest is used (DIV-2680).
// The length helper alone runs at its own fixed 64-bit precision.

// The stored constants, by their published bits (MAGIC-282).
var (
	boltMinusSix   = math.Float64frombits(0xc018000000000000)
	boltBandScale  = math.Float64frombits(0x3fc3333333333333) // 0.15
	boltVertScale  = math.Float64frombits(0x3f9eb851eb851eb8) // 0.03
	boltOne        = math.Float64frombits(0x3ff0000000000000)
	boltMinusOne   = math.Float64frombits(0xbff0000000000000)
	boltHundredth  = math.Float64frombits(0x3f847ae147ae147b) // 0.01
	boltTwo        = math.Float64frombits(0x4000000000000000)
	boltThree      = math.Float64frombits(0x4008000000000000)
	boltMinusThree = math.Float64frombits(0xc008000000000000)
	boltWalkEnd    = math.Float64frombits(0x3fe6666666666666) // 0.7
	boltMinusHalf  = math.Float64frombits(0xbfe0000000000000)
)

// The two moduli of one walk attempt (MAGIC-276).
const (
	boltDeflectModulus = 7
	boltStepModulus    = 50
)

// boltPoint is one stored record: i16 x, i16 y, u8 tag (MAGIC-275).
type boltPoint struct {
	X, Y int16
	Tag  uint8
}

// boltFigureRunLimit and boltWalkAttemptLimit bound the re-run and the walk.
// No claim gives a bound; these are engine safety stops that the engine's
// generator does not reach in practice (DIV-2682).
const (
	boltFigureRunLimit   = 1024
	boltWalkAttemptLimit = 1 << 16
)

// boltFigure builds one link from the projectile display point (ax, ay) to
// the target display point (bx, by). rand is the stream the walk consumes; a
// rejected figure re-runs on the continued stream (MAGIC-278).
func boltFigure(ax, ay, bx, by int32, tag uint8, rand func() int) []boltPoint {
	fax, fay := float64(ax), float64(ay)
	dx, dy := float64(bx)-fax, float64(by)-fay
	length := boltHypot(dx, dy)
	var xs, ys []float64
	for run := 0; run < boltFigureRunLimit; run++ {
		var accepted bool
		xs, ys, accepted = boltSampleRun(fax, fay, length, rand)
		if accepted {
			break
		}
	}
	return boltRotate(xs, ys, dx, dy, length, tag)
}

// boltHypot is the scaled length (MAGIC-282). The helper installs its own
// control word, so every step is 64-bit-significand nearest arithmetic with a
// binary64 store at a, b, q, h and p; the exponent of the frexp product is
// rebuilt rather than multiplied. Only the later figure arithmetic inherits
// the Unknown incoming word (DIV-2680).
func boltHypot(dx, dy float64) float64 {
	adx, ady := math.Abs(dx), math.Abs(dy)
	m := math.Max(adx, ady)
	if m == 0 {
		return 0
	}
	a := boltS53(boltR64().Quo(boltX(adx), boltX(m)))
	b := boltS53(boltR64().Quo(boltX(ady), boltX(m)))
	q := boltS53(boltR64().Add(boltR64().Mul(boltX(a), boltX(a)), boltR64().Mul(boltX(b), boltX(b))))
	h := boltS53(boltSqrt64(q))
	fm, em := math.Frexp(m)
	fh, eh := math.Frexp(h)
	p := boltS53(boltR64().Mul(boltX(fm), boltX(fh)))
	bits := math.Float64bits(p)
	n := int((bits>>52)&0x7ff) - 1022 + em + eh
	top := uint16(bits>>48)&0x800f | uint16((n+1022)<<4)
	return math.Float64frombits(bits&0x0000ffffffffffff | uint64(top)<<48)
}

// boltR64 is a 64-bit-significand nearest result.
func boltR64() *big.Float { return new(big.Float).SetPrec(64).SetMode(big.ToNearestEven) }

func boltX(v float64) *big.Float { return boltR64().SetFloat64(v) }

// boltS53 is a binary64 store, nearest-even.
func boltS53(v *big.Float) float64 {
	f, _ := v.Float64()
	return f
}

// boltSqrt64 is sqrt(q) rounded once to a 64-bit significand, nearest-even.
func boltSqrt64(q float64) *big.Float {
	if !(q > 0) || math.IsInf(q, 0) {
		return boltX(math.Sqrt(q))
	}
	frac, exp := math.Frexp(q)
	mant := new(big.Int).SetUint64(uint64(math.Ldexp(frac, 53)))
	exp -= 53
	if exp%2 != 0 {
		mant.Lsh(mant, 1)
		exp--
	}
	const scale = 128
	mant.Lsh(mant, scale)
	root := new(big.Int).Sqrt(mant)
	sticky := new(big.Int).Mul(root, root).Cmp(mant) != 0
	drop := uint(root.BitLen() - 64)
	low := new(big.Int).And(root, new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), drop), big.NewInt(1)))
	half := new(big.Int).Lsh(big.NewInt(1), drop-1)
	root.Rsh(root, drop)
	if c := low.Cmp(half); c > 0 || c == 0 && (sticky || root.Bit(0) == 1) {
		root.Add(root, big.NewInt(1))
	}
	z := boltR64().SetInt(root)
	return z.SetMantExp(z, exp/2-scale/2+int(drop))
}

// boltSampleRun is one complete run: walk, knots, quadratic samples and the
// band test. accepted is false when a sample leaves the band.
func boltSampleRun(ax, ay, length float64, rand func() int) (xs, ys []float64, accepted bool) {
	h := ax + length
	d := h - ax
	vertical := float64(length * boltVertScale)
	band := float64(length * boltBandScale)
	t, q := boltWalk(rand)
	kx, ky := boltKnots(t, q, ax, ay, d, vertical)
	for k := 0; k+2 < len(kx); k += 2 {
		xs, ys = boltSampleTriple(xs, ys, kx[k:k+3], ky[k:k+3])
	}
	for _, y := range ys {
		if math.Abs(y-ay) > band {
			return xs, ys, false
		}
	}
	return xs, ys, true
}

// boltWalk is the raw source (MAGIC-276). One draw sets the sign; each
// attempt draws a deflection and a step, admitted only for |v|>2 and d>0.15.
// The walk runs while the last abscissa is below 0.7, then ends at (1,0).
func boltWalk(rand func() int) (t, q []float64) {
	t, q = []float64{0}, []float64{0}
	sign := boltMinusOne
	if rand()%2 != 0 {
		sign = boltOne
	}
	for attempt := 0; t[len(t)-1] < boltWalkEnd && attempt < boltWalkAttemptLimit; attempt++ {
		v := float64(float64(rand()%boltDeflectModulus) * sign)
		step := float64(float64(rand()%boltStepModulus) * boltHundredth)
		if !(math.Abs(v) > boltTwo && step > boltBandScale) {
			continue
		}
		next := q[len(q)-1] + v
		if next > boltThree {
			next = boltThree
		}
		if next < boltMinusThree {
			next = boltMinusThree
		}
		t = append(t, t[len(t)-1]+step)
		q = append(q, next)
		sign = float64(sign * boltMinusOne)
	}
	if t[len(t)-1] >= boltOne {
		t, q = t[:len(t)-1], q[:len(q)-1]
	}
	return append(t, boltOne), append(q, 0)
}

// boltKnots maps the raw walk onto canonical X and inserts a -0.5 midpoint
// before every interior point after the second (MAGIC-277).
func boltKnots(t, q []float64, ax, ay, d, vertical float64) (kx, ky []float64) {
	px := func(j int) float64 { return float64(t[j]*d) + ax }
	py := func(j int) float64 { return float64(float64(t[j]*0)+float64(q[j]*vertical)) + ay }
	n := len(t)
	kx = append(kx, px(0), px(1))
	ky = append(ky, py(0), py(1))
	for j := 2; j <= n-2; j++ {
		prevX, prevY := kx[len(kx)-1], ky[len(ky)-1]
		x, y := px(j), py(j)
		kx = append(kx, prevX-float64((x-prevX)*boltMinusHalf), x)
		ky = append(ky, prevY-float64((y-prevY)*boltMinusHalf), y)
	}
	return append(kx, px(n-1)), append(ky, py(n-1))
}

// boltQuadratic is the 43-operation coefficient program of y=(A*x+B)*x+C
// through three knots, in the published order (MAGIC-277).
func boltQuadratic(x0, y0, x1, y1, x2, y2 float64) (a, b, c float64) {
	t1 := float64(y1 * x0)
	t2 := y1 - y0
	t3 := float64(x1 * y0)
	t4 := float64(y2 * x1)
	t5 := t1 - t3
	t6 := float64(x2 * y1)
	t7 := y0 - y2
	t8 := float64(x0 * x0)
	t9 := t4 - t6
	t10 := float64(x1 * x1)
	t11 := float64(x2 * x2)
	t12 := x0 - x2
	t13 := x2 - x1
	t14 := float64(x2 * y0)
	t15 := float64(y2 * x0)
	t16 := y2 - y1
	t17 := x1 - x0
	t18 := float64(t12 * t10)
	t19 := float64(t13 * t8)
	t20 := float64(t5 * t11)
	t21 := float64(t9 * t8)
	t22 := t14 - t15
	t23 := float64(x2 * t2)
	t24 := float64(x1 * t7)
	t25 := float64(t17 * t11)
	t26 := t18 + t19
	t27 := float64(t11 * t2)
	t28 := float64(t22 * t10)
	t29 := t23 + t24
	t30 := float64(t10 * t7)
	t31 := t20 + t21
	t32 := float64(x0 * t16)
	t33 := t27 + t30
	t34 := float64(t8 * t16)
	t35 := t26 + t25
	t36 := t29 + t32
	t37 := t31 + t28
	t38 := t33 + t34
	t39 := t36 / t35
	t40 := t37 / t35
	t41 := t38 / t35
	return -t39, t41, -t40
}

// boltSampleTriple appends the samples of one triple: from the first knot's
// x in steps of six while strictly below the third's (MAGIC-277).
func boltSampleTriple(xs, ys, kx, ky []float64) ([]float64, []float64) {
	a, b, c := boltQuadratic(kx[0], ky[0], kx[1], ky[1], kx[2], ky[2])
	for x := kx[0]; x < kx[2]; {
		xs = append(xs, x)
		ys = append(ys, float64(float64(float64(a*x)+b)*x)+c)
		x = x - boltMinusSix
	}
	return xs, ys
}

// boltRotate turns the accepted samples about the first sample onto the
// segment, truncates toward zero and keeps the low 16 bits (MAGIC-275).
func boltRotate(xs, ys []float64, dx, dy, length float64, tag uint8) []boltPoint {
	if len(xs) == 0 {
		return nil
	}
	c, s := dx/length, dy/length
	x0, y0 := xs[0], ys[0]
	out := make([]boltPoint, len(xs))
	for i := range xs {
		u, v := xs[i]-x0, ys[i]-y0
		rx := float64(float64(c*u)+x0) - float64(s*v)
		ry := float64(float64(c*v)+float64(s*u)) + y0
		out[i] = boltPoint{X: boltTruncWord(rx), Y: boltTruncWord(ry), Tag: tag}
	}
	return out
}

// boltTruncWord is the conversion helper's truncating i64 store, low word
// kept. An out-of-range or NaN operand stores the integer indefinite, whose
// low word is zero.
func boltTruncWord(v float64) int16 {
	if !(v > -0x1p63 && v < 0x1p63) {
		return 0
	}
	return int16(int64(v))
}
