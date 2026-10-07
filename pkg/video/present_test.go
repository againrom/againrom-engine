package video

import (
	"testing"

	"againrom/internal/synth"
)

func identityPalette(r, g, b byte) (p [256][3]byte) {
	for i := range p {
		p[i] = [3]byte{r, g, b}
	}
	return
}

// planeByColumn gives each source column x the palette index x/2 modulo 256
// and pal[i] = (i, 0, 0), so an output pixel's red channel names its source
// column.
func planeByColumn(w, h int) ([]byte, [256][3]byte) {
	plane := make([]byte, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			plane[y*w+x] = byte(x / 2)
		}
	}
	var pal [256][3]byte
	for i := range pal {
		pal[i] = [3]byte{byte(i), 0, 0}
	}
	return plane, pal
}

func TestPresentedSizeFollowsTheDoublingRule(t *testing.T) {
	for _, c := range []struct{ w, h, ow, oh, scale int }{
		{640, 360, 640, 360, 1},
		{800, 360, 640, 360, 1},
		{320, 180, 640, 360, 2},
		{640, 480, 640, 480, 1},
		{641, 181, 640, 360, 1},
		{320, 181, 640, 360, 2},
		{321, 180, 640, 360, 2},
	} {
		ow, oh, scale := presentedSize(c.w, c.h)
		if ow != c.ow || oh != c.oh || scale != c.scale {
			t.Errorf("presentedSize(%d,%d) = %d,%d,%d want %d,%d,%d", c.w, c.h, ow, oh, scale, c.ow, c.oh, c.scale)
		}
	}
}

func TestSmallMovieIsDrawnAsTwoByTwoBlocks(t *testing.T) {
	plane, pal := planeByColumn(320, 180)
	out := newPresenter(320, 180, 3, nil).present(plane, pal, true)
	if out.Rect.Dx() != 640 || out.Rect.Dy() != 360 {
		t.Fatalf("size %v", out.Rect)
	}
	for _, x := range []int{0, 1, 2, 3, 100, 101, 638, 639} {
		want := byte((x / 2) / 2)
		if got := out.Pix[x*4]; got != want {
			t.Errorf("column %d red = %d, want %d", x, got, want)
		}
	}
	if out.Pix[(359*640+639)*4+3] != 255 {
		t.Error("alpha not opaque")
	}
}

func TestWideMovieShowsAWindowAtTheStartOrigin(t *testing.T) {
	plane, pal := planeByColumn(800, 360)
	out := newPresenter(800, 360, 3, &Sidecar{StartX: 100}).present(plane, pal, true)
	if got := out.Pix[0]; got != 50 {
		t.Errorf("window starts at source column 100: red = %d, want 50", got)
	}
	if got := out.Pix[639*4]; got != byte((739/2)%256) {
		t.Errorf("window ends at source column 739: red = %d, want %d", got, (739/2)%256)
	}
}

// A pan adds its step after each frame: the first moved blit is frame start+1
// and frame end is drawn at the start origin plus (end-start) steps.
func TestPanMovesTheSourceOriginAfterEachFrame(t *testing.T) {
	plane, pal := planeByColumn(800, 360)
	p := newPresenter(800, 360, 100, &Sidecar{Pans: []Pan{{Start: 2, End: 10, StepX: 4}}})
	for k := 0; k < 14; k++ {
		out := p.present(plane, pal, true)
		steps := k - 2
		if steps < 0 {
			steps = 0
		}
		if steps > 8 {
			steps = 8
		}
		if want := byte(steps * 4 / 2); out.Pix[0] != want {
			t.Errorf("frame %d: first column red = %d, want %d", k, out.Pix[0], want)
		}
	}
}

func TestPanOutsideTheMovieLeavesBlack(t *testing.T) {
	plane, pal := planeByColumn(800, 360)
	out := newPresenter(800, 360, 2, &Sidecar{StartX: 700}).present(plane, pal, true)
	if out.Pix[99*4] == 0 && out.Pix[0] == 0 {
		t.Fatal("window did not show the movie")
	}
	for _, x := range []int{100, 639} {
		if px := out.Pix[x*4 : x*4+4]; px[0] != 0 || px[1] != 0 || px[2] != 0 || px[3] != 255 {
			t.Errorf("column %d past the movie = %v, want opaque black", x, px)
		}
	}
}

// A fade from frame s to e is applied on frames s to e-1: frame s already
// shows startfade plus one step, frame e-1 shows endfade, and frame e is the
// unscaled palette.
func TestFadeInRunsFromStartFrameToEndFrameMinusOne(t *testing.T) {
	plane := make([]byte, 640*360)
	pal := identityPalette(200, 100, 40)
	p := newPresenter(640, 360, 20, &Sidecar{Fades: []Fade{{Start: 2, End: 6, From: 0, To: 1}}})
	want := []byte{200, 200, 50, 100, 150, 200, 200}
	for k, w := range want {
		if got := p.present(plane, pal, k == 0).Pix[0]; got != w {
			t.Errorf("frame %d red = %d, want %d", k, got, w)
		}
	}
}

func TestFadeOutEndsAtZeroOnTheLastFadeFrame(t *testing.T) {
	plane := make([]byte, 640*360)
	pal := identityPalette(200, 100, 40)
	p := newPresenter(640, 360, 20, &Sidecar{Fades: []Fade{{Start: 0, End: 4, From: 1, To: 0}}})
	want := []byte{150, 100, 50, 0, 0}
	for k, w := range want {
		out := p.present(plane, pal, k == 0)
		if out.Pix[0] != w {
			t.Errorf("frame %d red = %d, want %d", k, out.Pix[0], w)
		}
	}
}

// Records are consumed in index order: a record whose start frame is already
// past never matches, and no later record of that family runs.
func TestFadeRecordsConsumeInOrder(t *testing.T) {
	plane := make([]byte, 640*360)
	pal := identityPalette(200, 0, 0)
	p := newPresenter(640, 360, 20, &Sidecar{Fades: []Fade{{Start: 5, End: 7, From: 0, To: 1}, {Start: 2, End: 4, From: 0, To: 1}}})
	var got []byte
	for k := 0; k < 9; k++ {
		got = append(got, p.present(plane, pal, k == 0).Pix[0])
	}
	want := []byte{200, 200, 200, 200, 200, 100, 200, 200, 200}
	for k := range want {
		if got[k] != want[k] {
			t.Fatalf("frames = %v, want %v", got, want)
		}
	}
}

// With no fade active the display palette is set only by a frame that carries
// a new palette: after a fade to black the frames that follow stay black until
// one does.
func TestFrameAfterFadeOutKeepsBlackUntilANewPalette(t *testing.T) {
	plane := make([]byte, 640*360)
	pal := identityPalette(200, 100, 40)
	p := newPresenter(640, 360, 20, &Sidecar{Fades: []Fade{{Start: 0, End: 4, From: 1, To: 0}}})
	for k := 0; k < 4; k++ {
		p.present(plane, pal, k == 0)
	}
	if got := p.present(plane, pal, false).Pix[0]; got != 0 {
		t.Fatalf("frame after the fade without a new palette: red = %d, want 0", got)
	}
	if got := p.present(plane, pal, true).Pix[0]; got != 200 {
		t.Fatalf("frame carrying a palette: red = %d, want 200", got)
	}
}

func TestDegenerateFadeSpanHasNoEffect(t *testing.T) {
	plane := make([]byte, 640*360)
	pal := identityPalette(200, 0, 0)
	p := newPresenter(640, 360, 5, &Sidecar{Fades: []Fade{{Start: 1, End: 1, From: 0, To: 1}}})
	for k := 0; k < 3; k++ {
		if got := p.present(plane, pal, true).Pix[0]; got != 200 {
			t.Errorf("frame %d red = %d, want 200", k, got)
		}
	}
}

func regInt(name string, v int32) synth.RegNode { return synth.RegNode{Name: name, Kind: 0x02, Int: v} }
func regFloat(name string, v float64) synth.RegNode {
	return synth.RegNode{Name: name, Kind: 0x04, Float: v}
}

func sidecarReg(nFade, nPan int32, extra ...synth.RegNode) []byte {
	common := synth.RegNode{Name: "Common", Kind: 0x01, Children: []synth.RegNode{
		regInt("startx", 3), regInt("starty", 4), regInt("nFadings", nFade), regInt("nPanaramings", nPan)}}
	return synth.Reg(17, append([]synth.RegNode{common}, extra...))
}

func TestParseSidecarReadsRecords(t *testing.T) {
	data := sidecarReg(2, 1,
		synth.RegNode{Name: "Fading1", Kind: 0x01, Children: []synth.RegNode{regInt("startframe", 0), regInt("endframe", 5), regFloat("startfade", 0), regFloat("endfade", 1)}},
		synth.RegNode{Name: "Fading2", Kind: 0x01, Children: []synth.RegNode{regInt("startframe", 84), regInt("endframe", 89), regFloat("startfade", 1), regFloat("endfade", 0)}},
		synth.RegNode{Name: "Panaraming1", Kind: 0x01, Children: []synth.RegNode{regInt("startframe", 0), regInt("endframe", 80), regInt("stepx", 2), regInt("stepy", -1)}},
	)
	sc, err := ParseSidecar(data)
	if err != nil {
		t.Fatal(err)
	}
	if sc.StartX != 3 || sc.StartY != 4 || len(sc.Fades) != 2 || len(sc.Pans) != 1 {
		t.Fatalf("sidecar = %+v", sc)
	}
	if sc.Fades[1] != (Fade{Start: 84, End: 89, From: 1, To: 0}) {
		t.Errorf("fade 2 = %+v", sc.Fades[1])
	}
	if sc.Pans[0] != (Pan{Start: 0, End: 80, StepX: 2, StepY: -1}) {
		t.Errorf("pan 1 = %+v", sc.Pans[0])
	}
}

// A declared record whose section is missing reads as zeros, and a negative
// count as none.
func TestParseSidecarDefaultsAndBounds(t *testing.T) {
	sc, err := ParseSidecar(sidecarReg(1, -3))
	if err != nil {
		t.Fatal(err)
	}
	if len(sc.Fades) != 1 || sc.Fades[0] != (Fade{}) || len(sc.Pans) != 0 {
		t.Errorf("sidecar = %+v", sc)
	}
	if _, err := ParseSidecar(sidecarReg(maxSidecarRecords+1, 0)); err == nil {
		t.Error("count above the bound was accepted")
	}
	if _, err := ParseSidecar([]byte("not a registry")); err == nil {
		t.Error("garbage was accepted")
	}
}

func TestSidecarNameReplacesTheLastExtension(t *testing.T) {
	for in, want := range map[string]string{
		"m10/01.smk":       "m10/01.reg",
		"part.one/a.b.smk": "part.one/a.b.reg",
		"logos/buka.smk":   "logos/buka.reg",
	} {
		if got := SidecarName(in); got != want {
			t.Errorf("SidecarName(%q) = %q, want %q", in, got, want)
		}
	}
}

// The decoder sets its new-palette flag at open, so the first frame sets the
// display palette even when its own flag is clear (VIDEO-084).
func TestFirstFrameAlwaysSetsThePalette(t *testing.T) {
	plane := make([]byte, 640*360)
	pal := identityPalette(200, 0, 0)
	p := newPresenter(640, 360, 3, nil)
	if got := p.present(plane, pal, false).Pix[0]; got != 200 {
		t.Fatalf("first frame red = %d, want 200", got)
	}
}
