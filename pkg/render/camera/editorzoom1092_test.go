package camera

import (
	"math"
	"testing"
)

func TestEditorZoom1092InstanceFloorLeavesDefaultsUnchanged(t *testing.T) {
	custom, ordinary := New(256, 256, 320, 480), New(256, 256, 320, 480)
	custom.SetMinimumZoom(320.0 / 8192)
	custom.SetZoom(0.001)
	custom.Pan(999999, 999999)
	custom.ZoomAbout(100, 100, 0.5)
	if custom.Zoom != 320.0/8192 {
		t.Fatalf("custom floor lost: %v", custom.Zoom)
	}
	for _, c := range []*Camera{ordinary, New(256, 256, 320, 480)} {
		c.SetZoom(0.001)
		c.ZoomAbout(100, 100, 0.5)
		if c.Zoom != 0.125 {
			t.Fatalf("default minimum changed: %v", c.Zoom)
		}
		c.SetZoom(100)
		if c.Zoom != 8 {
			t.Fatalf("default maximum changed: %v", c.Zoom)
		}
	}
	for _, invalid := range []float64{0, -1, 9, math.NaN(), math.Inf(1)} {
		custom.SetMinimumZoom(invalid)
		custom.SetZoom(0.001)
		if custom.Zoom != 0.125 {
			t.Fatalf("invalid floor did not restore default: %v", custom.Zoom)
		}
	}
	custom.SetMinimumZoom(2)
	custom.SetZoom(math.NaN())
	if custom.Zoom != 2 {
		t.Fatal("native fallback escaped the instance floor")
	}
}
