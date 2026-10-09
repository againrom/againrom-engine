package camera

import "testing"

func TestCellClampKeepsTheOriginWhenBoundsAreReapplied(t *testing.T) {
	c := New(64, 64, 647, 605)
	c.SetCellClampBounds(256, 256, 1792, 1792)
	c.Pan(1e9, 1e9)
	if c.X != 1145 || c.Y != 1216 {
		t.Fatalf("far origin (%v,%v), want (1145,1216)", c.X, c.Y)
	}
	c.SetCellClampBounds(256, 256, 1792, 1792)
	if c.X != 1145 || c.Y != 1216 {
		t.Fatalf("reapplied bounds moved origin to (%v,%v)", c.X, c.Y)
	}
}

func TestCellClampCentresAnOversizedView(t *testing.T) {
	c := New(32, 32, 27*32, 24*32)
	c.SetCellClampBounds(8*32, 8*32, 24*32, 24*32)
	for _, delta := range []float64{-1e9, 1e9} {
		c.Pan(delta, delta)
		if c.X != 80 || c.Y != 128 {
			t.Fatalf("crossed band origin (%v,%v), want (80,128)", c.X, c.Y)
		}
	}
	c.ViewH = 513
	c.Clamp()
	if c.Y != 255.5 {
		t.Fatalf("fractional oversized origin %v, want 255.5", c.Y)
	}
	c.ViewH = 24 * 32
	c.Clamp()
	c.SetClampBounds(8*32, 8*32, 24*32, 24*32)
	if c.X != 80 || c.Y != 128 {
		t.Fatalf("ordinary clamp retained cell policy: (%v,%v)", c.X, c.Y)
	}
}
