package game

import (
	"math"
	"math/big"
	"slices"
	"testing"
)

// The figure tests take their expectations from the published arithmetic
// (knowledge formats/magic/projectiles.md, MAGIC-275..278, MAGIC-282). The
// end-to-end oracle below re-evaluates the published listing in math/big at
// 53 bits, nearest-even, one rounding per instruction; it shares no float64
// code with the generator.

// scriptedRand answers the given draws in order and counts them.
type scriptedRand struct {
	draws []int
	used  int
}

func (r *scriptedRand) next() int {
	v := r.draws[r.used]
	r.used++
	return v
}

func TestBoltConstantsCarryTheirPublishedBits(t *testing.T) {
	for _, c := range []struct {
		name string
		got  float64
		bits uint64
	}{
		{"-6", boltMinusSix, 0xc018000000000000},
		{"0.15", boltBandScale, 0x3fc3333333333333},
		{"0.03", boltVertScale, 0x3f9eb851eb851eb8},
		{"1", boltOne, 0x3ff0000000000000},
		{"-1", boltMinusOne, 0xbff0000000000000},
		{"0.01", boltHundredth, 0x3f847ae147ae147b},
		{"2", boltTwo, 0x4000000000000000},
		{"3", boltThree, 0x4008000000000000},
		{"-3", boltMinusThree, 0xc008000000000000},
		{"0.7", boltWalkEnd, 0x3fe6666666666666},
		{"-0.5", boltMinusHalf, 0xbfe0000000000000},
	} {
		if got := math.Float64bits(c.got); got != c.bits {
			t.Errorf("constant %s has bits %016x, want %016x", c.name, got, c.bits)
		}
	}
	if boltDeflectModulus != 7 || boltStepModulus != 50 {
		t.Errorf("moduli %d/%d, want 7/50", boltDeflectModulus, boltStepModulus)
	}
}

// TestBoltWalkAdmitsClampsAlternatesAndEnds walks a scripted stream through
// every walk rule: two draws per attempt, |v|>2 and d>0.15 strictly, the
// sign flipped only by an accepted step, the [-3,3] clamp, the 0.7 stop and
// the removal of a last abscissa at or past 1.
func TestBoltWalkAdmitsClampsAlternatesAndEnds(t *testing.T) {
	r := &scriptedRand{draws: []int{
		1,     // odd: sign +1
		2, 40, // v=+2: rejected (|v| must exceed 2)
		6, 15, // d=0.15: rejected (d must exceed 0.15)
		13, 16, // 13%7=6 -> v=+6, d=0.16: accepted, q clamps 6 -> 3, sign -> -1
		5, 49, // v=-5, d=0.49: accepted, q=-2, t=0.65, sign -> +1
		3, 66, // v=+3, 66%50=16 -> d=0.16: accepted, q=1, t=0.81 stops the walk
	}}
	tt, q := boltWalk(r.next)
	if r.used != len(r.draws) {
		t.Fatalf("the walk consumed %d draws, want %d", r.used, len(r.draws))
	}
	wantT := []float64{0, float64(16 * boltHundredth), 0, 0, 1}
	wantT[2] = wantT[1] + float64(49*boltHundredth)
	wantT[3] = wantT[2] + float64(16*boltHundredth)
	if !slices.Equal(tt, wantT) || !slices.Equal(q, []float64{0, 3, -2, 1, 0}) {
		t.Fatalf("walk t=%v q=%v, want t=%v q=[0 3 -2 1 0]", tt, q, wantT)
	}

	// A step that ends at or past 1 is removed before the (1,0) end:
	// sign -1, then d=0.20, 0.49, 0.49 reach 1.18.
	r = &scriptedRand{draws: []int{0, 6, 20, 6, 49, 6, 49}}
	tt, q = boltWalk(r.next)
	second := float64(20*boltHundredth) + float64(49*boltHundredth)
	if r.used != len(r.draws) || !slices.Equal(tt, []float64{0, float64(20 * boltHundredth), second, 1}) ||
		!slices.Equal(q, []float64{0, -3, 3, 0}) {
		t.Fatalf("walk t=%v q=%v after %d draws, want the step past 1 removed", tt, q, r.used)
	}
}

// TestBoltKnotsInsertMidpoints: the -0.5 operation makes the exact midpoint
// of each interior pair; a positive factor would extrapolate instead.
func TestBoltKnotsInsertMidpoints(t *testing.T) {
	tt := []float64{0, 0.25, 0.5, 0.75, 1}
	q := []float64{0, 2, -2, 3, 0}
	kx, ky := boltKnots(tt, q, 10, 20, 100, 3)
	wantX := []float64{10, 35, 47.5, 60, 72.5, 85, 110}
	wantY := []float64{20, 26, 20, 14, 21.5, 29, 20}
	if !slices.Equal(kx, wantX) || !slices.Equal(ky, wantY) {
		t.Fatalf("knots x=%v y=%v, want x=%v y=%v", kx, ky, wantX, wantY)
	}
}

// TestBoltSamplingIsHalfOpenAtSixUnits: samples start at the first knot, step
// by six and stay strictly below the third knot.
func TestBoltSamplingIsHalfOpenAtSixUnits(t *testing.T) {
	for _, c := range []struct {
		x2   float64
		want []float64
	}{
		{12, []float64{0, 6}},
		{12.5, []float64{0, 6, 12}},
		{5, []float64{0}},
	} {
		xs, ys := boltSampleTriple(nil, nil, []float64{0, c.x2 / 2, c.x2}, []float64{0, 1, 0})
		if !slices.Equal(xs, c.want) || len(ys) != len(xs) {
			t.Errorf("third knot at %v samples x=%v, want %v", c.x2, xs, c.want)
		}
	}
}

// TestBoltQuadraticMatchesTheListing compares the coefficient program with
// the oracle's 43 operations bit for bit, and checks it fits its knots.
func TestBoltQuadraticMatchesTheListing(t *testing.T) {
	for _, k := range [][6]float64{
		{0, 0, 17.5, 4.25, 35, -2},
		{103, 87.31, 151.2, 90.04, 199.4, 85.5},
		{-40, 12, -10, 13.75, 22.5, 11},
	} {
		a, b, c := boltQuadratic(k[0], k[1], k[2], k[3], k[4], k[5])
		wa, wb, wc := boltOracleQuadratic(k)
		if a != wa || b != wb || c != wc {
			t.Errorf("knots %v: coefficients %v %v %v, want %v %v %v", k, a, b, c, wa, wb, wc)
		}
		for i := 0; i < 6; i += 2 {
			if y := (a*k[i]+b)*k[i] + c; math.Abs(y-k[i+1]) > 1e-9 {
				t.Errorf("knots %v: the quadratic misses knot %d by %v", k, i/2, y-k[i+1])
			}
		}
	}
}

func TestBoltHypotMatchesTheScaledSequence(t *testing.T) {
	for _, d := range [][2]float64{{3, 4}, {-120, 35}, {0, -96}, {1, 1}, {2147483647, -2147483648}, {517, 311}} {
		got, want := boltHypot(d[0], d[1]), boltOracleHypot(d[0], d[1])
		if math.Float64bits(got) != math.Float64bits(want) {
			t.Errorf("hypot%v = %v, want %v", d, got, want)
		}
		if exact := math.Hypot(d[0], d[1]); math.Abs(got-exact) > 1e-6*exact {
			t.Errorf("hypot%v = %v, far from %v", d, got, exact)
		}
	}
	if boltHypot(0, 0) != 0 {
		t.Error("a zero segment has nonzero length")
	}
}

// TestBoltTruncationKeepsTheLowWord: toward zero, then the low 16 bits.
func TestBoltTruncationKeepsTheLowWord(t *testing.T) {
	for _, c := range []struct {
		v    float64
		want int16
	}{
		{1.99, 1}, {-1.99, -1}, {-0.5, 0}, {70000.7, 4464}, {-32769, 32767}, {math.NaN(), 0}, {1e300, 0},
	} {
		if got := boltTruncWord(c.v); got != c.want {
			t.Errorf("trunc(%v) = %d, want %d", c.v, got, c.want)
		}
	}
}

// TestBoltRotationTurnsAboutTheFirstSample: a vertical segment turns the
// canonical x axis onto +y about the first sample.
func TestBoltRotationTurnsAboutTheFirstSample(t *testing.T) {
	got := boltRotate([]float64{10, 16, 22}, []float64{5, 7.9, 4}, 0, 50, 50, 34)
	want := []boltPoint{{10, 5, 34}, {7, 11, 34}, {11, 17, 34}}
	if !slices.Equal(got, want) {
		t.Fatalf("rotated %v, want %v", got, want)
	}
}

// TestBoltReRunContinuesTheStream finds a seed whose first run leaves the
// band, then checks the figure is the second run on the continued stream.
func TestBoltReRunContinuesTheStream(t *testing.T) {
	const ax, ay, bx, by = 40, 60, 260, 110
	fax, fay := float64(ax), float64(ay)
	length := boltHypot(float64(bx-ax), float64(by-ay))
	for seed := uint32(1); seed < 4000; seed++ {
		probe := boltRNG{state: seed}
		_, _, ok := boltSampleRun(fax, fay, length, probe.next)
		if ok {
			continue
		}
		xs, ys, ok2 := boltSampleRun(fax, fay, length, probe.next)
		if !ok2 {
			continue
		}
		want := boltRotate(xs, ys, float64(bx-ax), float64(by-ay), length, 34)
		rng := boltRNG{state: seed}
		if got := boltFigure(ax, ay, bx, by, 34, rng.next); !slices.Equal(got, want) {
			t.Fatalf("seed %d: the figure is not the second run on the continued stream", seed)
		}
		if rng.state != probe.state {
			t.Fatalf("seed %d: the figure consumed a different stream than two runs", seed)
		}
		return
	}
	t.Fatal("no seed below 4000 rejects its first run")
}

// TestBoltFigureMatchesThePublishedProgram runs whole figures against the
// big-float oracle: same endpoints, same stream, same integer points.
func TestBoltFigureMatchesThePublishedProgram(t *testing.T) {
	cases := [][4]int32{{100, 100, 340, 180}, {500, 40, 210, 300}, {64, 400, 64, 90}, {0, 0, 37, -5}, {-30, 20, 900, 25}}
	for _, c := range cases {
		for seed := uint32(1); seed <= 25; seed++ {
			got := boltFigure(c[0], c[1], c[2], c[3], 3, (&boltRNG{state: seed}).next)
			want := boltOracleFigure(c, (&boltRNG{state: seed}).next)
			if len(got) != len(want) {
				t.Fatalf("%v seed %d: %d points, want %d", c, seed, len(got), len(want))
			}
			for i := range got {
				if got[i].X != want[i][0] || got[i].Y != want[i][1] || got[i].Tag != 3 {
					t.Fatalf("%v seed %d: point %d is %v, want %v", c, seed, i, got[i], want[i])
				}
			}
		}
	}
}

// TestBoltFigureGolden pins one figure, so a change to any pass shows here
// even if the oracle were changed with it.
func TestBoltFigureGolden(t *testing.T) {
	got := boltFigure(100, 100, 340, 180, 34, (&boltRNG{state: 7}).next)
	want := goldenBoltFigure()
	if len(got) != len(want) {
		t.Fatalf("golden figure has %d points, want %d: %v", len(got), len(want), got)
	}
	for i := range got {
		if got[i].X != want[i][0] || got[i].Y != want[i][1] {
			t.Fatalf("golden point %d is %v, want %v; whole figure %v", i, got[i], want[i], got)
		}
	}
	if got := boltFigure(5, 5, 5, 5, 34, (&boltRNG{state: 7}).next); len(got) != 0 {
		t.Fatalf("a zero-length segment stored %d points, want none", len(got))
	}
}

// ------------------------------------------------------------ the oracle

const boltOraclePrec = 53

func boltBF(v float64) *big.Float { return new(big.Float).SetPrec(boltOraclePrec).SetFloat64(v) }

func boltOp(op byte, a, b *big.Float) *big.Float {
	z := new(big.Float).SetPrec(boltOraclePrec).SetMode(big.ToNearestEven)
	switch op {
	case '+':
		return z.Add(a, b)
	case '-':
		return z.Sub(a, b)
	case '*':
		return z.Mul(a, b)
	default:
		return z.Quo(a, b)
	}
}

func boltF64(z *big.Float) float64 { v, _ := z.Float64(); return v }

func boltOracleHypot(dx, dy float64) float64 {
	adx, ady := math.Abs(dx), math.Abs(dy)
	m := math.Max(adx, ady)
	if m == 0 {
		return 0
	}
	a, b := boltOp('/', boltBF(adx), boltBF(m)), boltOp('/', boltBF(ady), boltBF(m))
	q := boltOp('+', boltOp('*', a, a), boltOp('*', b, b))
	h := new(big.Float).SetPrec(boltOraclePrec).SetMode(big.ToNearestEven).Sqrt(q)
	// p*2^(em+eh) is the rebuilt exponent: fm*fh*2^em*2^eh.
	fm, em := math.Frexp(m)
	fh, eh := math.Frexp(boltF64(h))
	return math.Ldexp(boltF64(boltOp('*', boltBF(fm), boltBF(fh))), em+eh)
}

func boltOracleQuadratic(k [6]float64) (a, b, c float64) {
	x0, y0, x1, y1, x2, y2 := boltBF(k[0]), boltBF(k[1]), boltBF(k[2]), boltBF(k[3]), boltBF(k[4]), boltBF(k[5])
	t := make([]*big.Float, 44)
	t[1] = boltOp('*', y1, x0)
	t[2] = boltOp('-', y1, y0)
	t[3] = boltOp('*', x1, y0)
	t[4] = boltOp('*', y2, x1)
	t[5] = boltOp('-', t[1], t[3])
	t[6] = boltOp('*', x2, y1)
	t[7] = boltOp('-', y0, y2)
	t[8] = boltOp('*', x0, x0)
	t[9] = boltOp('-', t[4], t[6])
	t[10] = boltOp('*', x1, x1)
	t[11] = boltOp('*', x2, x2)
	t[12] = boltOp('-', x0, x2)
	t[13] = boltOp('-', x2, x1)
	t[14] = boltOp('*', x2, y0)
	t[15] = boltOp('*', y2, x0)
	t[16] = boltOp('-', y2, y1)
	t[17] = boltOp('-', x1, x0)
	t[18] = boltOp('*', t[12], t[10])
	t[19] = boltOp('*', t[13], t[8])
	t[20] = boltOp('*', t[5], t[11])
	t[21] = boltOp('*', t[9], t[8])
	t[22] = boltOp('-', t[14], t[15])
	t[23] = boltOp('*', x2, t[2])
	t[24] = boltOp('*', x1, t[7])
	t[25] = boltOp('*', t[17], t[11])
	t[26] = boltOp('+', t[18], t[19])
	t[27] = boltOp('*', t[11], t[2])
	t[28] = boltOp('*', t[22], t[10])
	t[29] = boltOp('+', t[23], t[24])
	t[30] = boltOp('*', t[10], t[7])
	t[31] = boltOp('+', t[20], t[21])
	t[32] = boltOp('*', x0, t[16])
	t[33] = boltOp('+', t[27], t[30])
	t[34] = boltOp('*', t[8], t[16])
	t[35] = boltOp('+', t[26], t[25])
	t[36] = boltOp('+', t[29], t[32])
	t[37] = boltOp('+', t[31], t[28])
	t[38] = boltOp('+', t[33], t[34])
	t[39] = boltOp('/', t[36], t[35])
	t[40] = boltOp('/', t[37], t[35])
	t[41] = boltOp('/', t[38], t[35])
	return -boltF64(t[39]), boltF64(t[41]), -boltF64(t[40])
}

// boltOracleFigure is the format page's whole listing: walk, knots, triples,
// band test with re-run, rotation and truncation.
func boltOracleFigure(c [4]int32, rand func() int) [][2]int16 {
	Ax, Ay := float64(c[0]), float64(c[1])
	dx, dy := float64(c[2])-Ax, float64(c[3])-Ay
	L := boltOracleHypot(dx, dy)
	D := boltF64(boltOp('-', boltOp('+', boltBF(Ax), boltBF(L)), boltBF(Ax)))
	vs := boltF64(boltOp('*', boltBF(L), boltBF(0.03)))
	band := boltF64(boltOp('*', boltBF(L), boltBF(0.15)))
	var X, Y []float64
	for {
		// walk
		tt, q := []float64{0}, []float64{0}
		sign := -1.0
		if rand()%2 == 1 {
			sign = 1
		}
		for tt[len(tt)-1] < 0.7 {
			v := float64(rand()%7) * sign
			d := boltF64(boltOp('*', boltBF(float64(rand()%50)), boltBF(0.01)))
			if math.Abs(v) > 2 && d > 0.15 {
				nq := math.Min(math.Max(q[len(q)-1]+v, -3), 3)
				tt = append(tt, boltF64(boltOp('+', boltBF(tt[len(tt)-1]), boltBF(d))))
				q = append(q, nq)
				sign = -sign
			}
		}
		if tt[len(tt)-1] >= 1 {
			tt, q = tt[:len(tt)-1], q[:len(q)-1]
		}
		tt, q = append(tt, 1), append(q, 0)
		// knots
		P := func(j int) (float64, float64) {
			x := boltF64(boltOp('+', boltOp('*', boltBF(tt[j]), boltBF(D)), boltBF(Ax)))
			y := boltF64(boltOp('+', boltOp('+', boltOp('*', boltBF(tt[j]), boltBF(0)), boltOp('*', boltBF(q[j]), boltBF(vs))), boltBF(Ay)))
			return x, y
		}
		var kx, ky []float64
		for j := 0; j < 2; j++ {
			x, y := P(j)
			kx, ky = append(kx, x), append(ky, y)
		}
		for j := 2; j <= len(tt)-2; j++ {
			x, y := P(j)
			px, py := kx[len(kx)-1], ky[len(ky)-1]
			mx := boltF64(boltOp('-', boltBF(px), boltOp('*', boltOp('-', boltBF(x), boltBF(px)), boltBF(-0.5))))
			my := boltF64(boltOp('-', boltBF(py), boltOp('*', boltOp('-', boltBF(y), boltBF(py)), boltBF(-0.5))))
			kx, ky = append(kx, mx, x), append(ky, my, y)
		}
		x, y := P(len(tt) - 1)
		kx, ky = append(kx, x), append(ky, y)
		// samples
		X, Y = nil, nil
		for k := 0; k+2 < len(kx); k += 2 {
			a, b, cc := boltOracleQuadratic([6]float64{kx[k], ky[k], kx[k+1], ky[k+1], kx[k+2], ky[k+2]})
			for h := kx[k]; h < kx[k+2]; h = boltF64(boltOp('-', boltBF(h), boltBF(-6))) {
				X = append(X, h)
				Y = append(Y, boltF64(boltOp('+', boltOp('*', boltOp('+', boltOp('*', boltBF(a), boltBF(h)), boltBF(b)), boltBF(h)), boltBF(cc))))
			}
		}
		ok := true
		for _, y := range Y {
			if math.Abs(boltF64(boltOp('-', boltBF(y), boltBF(Ay)))) > band {
				ok = false
			}
		}
		if ok {
			break
		}
	}
	if len(X) == 0 {
		return nil
	}
	cs, sn := boltF64(boltOp('/', boltBF(dx), boltBF(L))), boltF64(boltOp('/', boltBF(dy), boltBF(L)))
	out := make([][2]int16, len(X))
	for i := range X {
		u, v := boltOp('-', boltBF(X[i]), boltBF(X[0])), boltOp('-', boltBF(Y[i]), boltBF(Y[0]))
		rx := boltOp('-', boltOp('+', boltOp('*', boltBF(cs), u), boltBF(X[0])), boltOp('*', boltBF(sn), v))
		ry := boltOp('+', boltOp('+', boltOp('*', boltBF(cs), v), boltOp('*', boltBF(sn), u)), boltBF(Y[0]))
		out[i] = [2]int16{int16(int64(math.Trunc(boltF64(rx)))), int16(int64(math.Trunc(boltF64(ry))))}
	}
	return out
}

func goldenBoltFigure() [][2]int16 {
	return [][2]int16{
		{100, 99}, {103, 107}, {107, 114}, {112, 120}, {116, 125}, {121, 129}, {126, 133}, {132, 135},
		{138, 137}, {144, 138}, {150, 138}, {157, 137}, {163, 135}, {171, 133}, {178, 129}, {182, 127},
		{190, 124}, {197, 121}, {204, 119}, {211, 117}, {217, 116}, {224, 116}, {230, 117}, {236, 118},
		{241, 120}, {247, 123}, {252, 126}, {257, 130}, {262, 135}, {266, 141}, {269, 144}, {274, 148},
		{279, 151}, {284, 155}, {289, 158}, {295, 161}, {300, 164}, {305, 167}, {311, 169}, {316, 172},
		{322, 174}, {328, 176}, {333, 178}, {339, 179},
	}
}
