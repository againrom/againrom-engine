package data

import (
	"math"
	"testing"
	"time"
)

// poolDist is the distance from v to the nearest integer -- 0 AT an integer
// and at most 0.5 at a half-integer. It is the truncation-boundary measure
// hero_test.go's own margin test uses for pow11 (there a closure local to
// that function, not exported for this file to share): a value that stands
// far from every integer cannot be pushed across a truncation boundary by an
// ULP the way a value one already sits on top of could be.
func poolDist(v float64) float64 {
	d := v - math.Floor(v)
	if d > 1-d {
		d = 1 - d
	}
	return d
}

func TestPoolLogMarginOverTheWholeExperienceRange(t *testing.T) {
	maxExp := 6 * ftol((pow11(100)-1)*skillXPScale)

	start := time.Now()
	const exactFloor = 1e-12
	var exactAt []int32
	worst := math.Inf(1)
	var worstAt int32
	var worstMult int
	for e := int32(0); e <= maxExp; e++ {
		l := logBase11(float64(e)/poolXPDivisor + 1)
		if e > 0 {
			if d := poolDist(l); d < exactFloor {
				exactAt = append(exactAt, e)
			}
		}
		for _, m := range [...]int{1, 2} {
			if d := poolDist(l * float64(m)); d > 0 && d < worst {
				worst, worstAt, worstMult = d, e, m
			}
		}
	}
	elapsed := time.Since(start)
	t.Logf("swept e = 0..%d (%d values) in %s; worst nonzero margin %v at e=%d, multiplier %d",
		maxExp, maxExp+1, elapsed, worst, worstAt, worstMult)
	t.Logf("dist < %g at e = %v", exactFloor, exactAt)

	wantExact := []int32{500, 1050, 1655}
	if len(exactAt) != len(wantExact) {
		t.Fatalf("dist < 1e-12 at %d value(s) %v, want exactly %v", len(exactAt), exactAt, wantExact)
	}
	for i, e := range exactAt {
		if e != wantExact[i] {
			t.Errorf("exact-power experience[%d] = %d, want %d", i, e, wantExact[i])
		}
	}

	const wantWorst = 1.7763568394002505e-15
	if worst != wantWorst || worstAt != 1050 || worstMult != 1 {
		t.Errorf("worst nonzero margin = %v at e=%d, multiplier %d; want %v at e=1050, multiplier 1",
			worst, worstAt, worstMult, wantWorst)
	}
}

func TestPoolLogAtTheThreeExactPowers(t *testing.T) {
	for _, tc := range []struct {
		e        int32
		wantLog  float64
		wantFtol int32
	}{
		{500, 1.0, 1},                 // k=1: exact in float64, no residual
		{1050, 1.9999999999999982, 1}, // k=2: a few ULP short of 2
		{1655, 2.9999999999999969, 2}, // k=3: a few ULP short of 3
	} {
		got := logBase11(float64(tc.e)/poolXPDivisor + 1)
		if got != tc.wantLog {
			t.Errorf("logBase11 at e=%d = %v, want %v", tc.e, got, tc.wantLog)
		}
		if got := ftol(got); got != tc.wantFtol {
			t.Errorf("ftol(logBase11) at e=%d = %d, want %d", tc.e, got, tc.wantFtol)
		}
	}
}

// TestPoolMarginOverConstructibleExperience is AC-13's second half: over the
// experience values a character THIS TREE CAN CONSTRUCT -- generation writes
// exactly one skill slot, at the ordinary level (ChargenSkill, 10) or the
// other arm's (20, hero.go's own doc on which mode reaches it), the other
// five at zero, so the sum a Recompute reads is skillXP(0), skillXP(10) or
// skillXP(20) alone -- the margin stands nowhere near a truncation boundary,
// by more than ten orders of magnitude against the whole-range worst case
// TestPoolLogMarginOverTheWholeExperienceRange found.
func TestPoolMarginOverConstructibleExperience(t *testing.T) {
	exps := []int32{0, 1593, 5727}

	worst := math.Inf(1)
	var worstE int32
	var worstMult int
	for _, e := range exps {
		l := logBase11(float64(e)/poolXPDivisor + 1)
		for _, m := range [...]int{1, 2} {
			if d := poolDist(l * float64(m)); d > 0 && d < worst {
				worst, worstE, worstMult = d, e, m
			}
		}
	}
	t.Logf("worst margin over constructible experience %v at e=%d, multiplier %d", worst, worstE, worstMult)

	const wantWorst = 0.008861348791139534
	if worst != wantWorst || worstE != 5727 || worstMult != 1 {
		t.Errorf("worst margin = %v at e=%d, multiplier %d; want %v at e=5727, multiplier 1",
			worst, worstE, worstMult, wantWorst)
	}

	const wholeRangeWorst = 1.7763568394002505e-15
	if worst/wholeRangeWorst < 1e10 {
		t.Errorf("constructible margin %v stands only %g times the whole-range worst %v; want at least ten orders of magnitude",
			worst, worst/wholeRangeWorst, wholeRangeWorst)
	}
}

func TestPoolGrowthStepMargin(t *testing.T) {
	maxExp := 6 * ftol((pow11(100)-1)*skillXPScale)
	maxLog := logBase11(float64(maxExp)/poolXPDivisor + 1)
	tightCeiling := ftol(classMult * (float64(StatCap) + maxLog))
	const sweepMaxH = 320
	if tightCeiling > sweepMaxH {
		t.Fatalf("derived pool ceiling %d exceeds the swept bound %d", tightCeiling, sweepMaxH)
	}
	t.Logf("derived tight ceiling for h: %d (swept to %d)", tightCeiling, sweepMaxH)

	worst := math.Inf(1)
	var worstH, worstS int32
	for h := int32(0); h <= sweepMaxH; h++ {
		for s := int32(0); s <= StatCap; s++ {
			v := float64(h) * (pow11(s)/poolGrowthDivisor + 1)
			if d := poolDist(v); d > 0 && d < worst {
				worst, worstH, worstS = d, h, s
			}
		}
	}
	t.Logf("worst margin over h=0..%d, s=0..%d: %v at h=%d, s=%d", sweepMaxH, StatCap, worst, worstH, worstS)

	const wantWorst = 2.368530759611076e-05
	if worst != wantWorst || worstH != 79 || worstS != 14 {
		t.Errorf("worst margin = %v at h=%d, s=%d; want %v at h=79, s=14", worst, worstH, worstS, wantWorst)
	}
}
