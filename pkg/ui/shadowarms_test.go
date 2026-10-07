package ui

import (
	"math"
	"testing"

	"againrom/pkg/render/terrain"
)

// armsViewer is shadowViewer with the fixture unit's art carrying a paired
// spritesb frame of another size (8x4 against the body's 10x6) and Smoothing
// on. The object and structure shadows are switched off so every shadow draw
// is the unit's.
func armsViewer(t *testing.T, mutate func(*MapEntity)) (*Viewer, MapEntity, *terrain.StaticFrame) {
	t.Helper()
	v := shadowViewer(t)
	v.showStaticArt, v.showStructureArt = false, false
	v.SetGraphicsOptions(GraphicsOptions{Smoothing: true})
	art := entityArtA()
	pair := &terrain.StaticFrame{Width: 8, Height: 4, Pixels: make([]terrain.StaticPixel, 32)}
	for i := range pair.Pixels {
		pair.Pixels[i] = terrain.StaticPixel{Index: 5, Opaque: true}
	}
	e := withFrame(shadowUnitCell, art)
	e.Boundary = pair
	if mutate != nil {
		mutate(&e)
	}
	v.SetEntities([]MapEntity{e})
	if s := v.Sun(); s.ShroudObject == s.ShroudUnit {
		t.Fatalf("the sun does not discriminate the two shroud indices: both %d", s.ShroudObject)
	}
	return v, e, pair
}

func TestUnitShadowSecondSilhouetteIsThePairedFrameAtTheUnitPathLevel(t *testing.T) {
	v, e, pair := armsViewer(t, nil)
	sun := v.Sun()
	slope := terrain.ShadowSlope(sun.Theta)
	draws := v.shadowDraws()
	if len(draws) != 2 {
		t.Fatalf("shadowDraws() = %d entries, want the two silhouettes", len(draws))
	}
	first, second := draws[0], draws[1]
	if first.Second || !second.Second || first.Frame != e.Frame || second.Frame != pair {
		t.Fatalf("silhouettes are (second=%v frame ok=%v) and (second=%v frame ok=%v)", first.Second, first.Frame == e.Frame, second.Second, second.Frame == pair)
	}
	if got, want := first.Alpha, float32(sun.ShroudObject)/16; got != want {
		t.Errorf("first silhouette alpha %v, want the object-path level %v", got, want)
	}
	if got, want := second.Alpha, float32(sun.ShroudUnit)/16; got != want {
		t.Errorf("second silhouette alpha %v, want the unit-path level %v", got, want)
	}
	// The body frame is 10x6 on a (64,64,32,60) canvas: anchor (5,31). The
	// pair is 8x4: anchor (4,30). The shear term is the body frame's and is
	// reused by the pair, whose own height is its pivot row.
	shift := math.Trunc(slope * float64(2*(6/2)-31))
	wantX := float64(shadowUnitCell.X*32+16-4) - shift + slope*4
	if math.Abs(second.X-wantX) > 1e-9 {
		t.Errorf("second silhouette origin X %v, want %v", second.X, wantX)
	}
	if want := float64(shadowUnitCell.Y*32 + 16 - 30); second.Y != want {
		t.Errorf("second silhouette Y %v, want %v", second.Y, want)
	}
	if second.Slope != slope || second.PivotRow != 4 {
		t.Errorf("second silhouette slope %v pivot %d, want %v and 4", second.Slope, second.PivotRow, slope)
	}
}

func TestUnitShadowSecondSilhouetteNeedsSmoothing(t *testing.T) {
	v, _, _ := armsViewer(t, nil)
	v.SetGraphicsOptions(GraphicsOptions{})
	if n := len(v.shadowDraws()); n != 1 {
		t.Errorf("with Smoothing off shadowDraws() = %d entries, want the first silhouette alone", n)
	}
}

func TestUnitShadowFlatArmForAirUnitsOnly(t *testing.T) {
	v, e, _ := armsViewer(t, func(e *MapEntity) { e.DrawCategory = terrain.UnitAir })
	sun := v.Sun()
	shift := float64(terrain.ShadowShear16(sun.Theta) / 2000)
	draws := v.shadowDraws()
	if len(draws) != 2 {
		t.Fatalf("shadowDraws() = %d entries, want 2", len(draws))
	}
	for i, d := range draws {
		if !d.Flat || d.Slope != 0 {
			t.Errorf("silhouette %d: flat=%v slope=%v, want the translated arm", i, d.Flat, d.Slope)
		}
	}
	if want := float64(shadowUnitCell.X*32+16-5) + shift; draws[0].X != want {
		t.Errorf("flat first silhouette X %v, want body X plus the translation %v", draws[0].X, want)
	}
	if want := float64(shadowUnitCell.X*32+16-4) + shift; draws[1].X != want {
		t.Errorf("flat second silhouette X %v, want %v", draws[1].X, want)
	}
	if shift == 0 {
		t.Error("the sun gives no translation; the fixture does not discriminate")
	}

	// A hero is always sheared, and an ordinary unit is.
	for name, mutate := range map[string]func(*MapEntity){
		"hero":     func(e *MapEntity) { e.DrawCategory, e.PlayerCharacter = terrain.UnitAir, true },
		"ordinary": func(e *MapEntity) {},
	} {
		v, _, _ := armsViewer(t, mutate)
		for _, d := range v.shadowDraws() {
			if d.Flat || d.Slope == 0 {
				t.Errorf("%s: silhouette flat=%v slope=%v, want sheared", name, d.Flat, d.Slope)
			}
		}
	}
	_ = e
}

func TestUnitShadowOfAnInvisibleUnitIsTheOwnersAlone(t *testing.T) {
	invisible := func(owner uint32) func(*MapEntity) {
		return func(e *MapEntity) { e.Translucent, e.Owner = true, owner }
	}
	v, _, _ := armsViewer(t, invisible(1))
	v.SetLocalOwner(1)
	sun := v.Sun()
	draws := v.shadowDraws()
	if len(draws) != 1 {
		t.Fatalf("an owner's invisible unit casts %d silhouettes, want 1", len(draws))
	}
	if got, want := draws[0].Alpha, float32(sun.ShroudUnit)/16; got != want {
		t.Errorf("owner's invisible silhouette alpha %v, want the unit-path level %v", got, want)
	}

	v, _, _ = armsViewer(t, invisible(2))
	v.SetLocalOwner(1)
	if n := len(v.shadowDraws()); n != 0 {
		t.Errorf("a detected enemy invisible unit casts %d silhouettes, want none", n)
	}
}
