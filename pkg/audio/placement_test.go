package audio

import "testing"

// AC-9: a sounding cell to the left of the listener yields a left gain
// strictly greater than its right; to the right, the reverse; on the
// listener, the two are equal. A cell beyond the falloff radius yields no
// play at all.
func TestPlaceLeftRightAndOnTheListener(t *testing.T) {
	left, ok := Place(-10, 0)
	if !ok {
		t.Fatal("Place(-10, 0): refused, want a play")
	}
	if left.Left <= left.Right {
		t.Fatalf("cell to the left: Left=%d, Right=%d, want Left > Right", left.Left, left.Right)
	}

	right, ok := Place(10, 0)
	if !ok {
		t.Fatal("Place(10, 0): refused, want a play")
	}
	if right.Left >= right.Right {
		t.Fatalf("cell to the right: Left=%d, Right=%d, want Left < Right", right.Left, right.Right)
	}

	// The left/right pair is symmetric: (-10,0) and (10,0) are mirror images,
	// so one's Left equals the other's Right and vice versa.
	if left.Left != right.Right || left.Right != right.Left {
		t.Fatalf("not mirror-symmetric: left=%+v right=%+v", left, right)
	}

	on, ok := Place(0, 0)
	if !ok {
		t.Fatal("Place(0, 0): refused, want a play")
	}
	if on.Left != on.Right {
		t.Fatalf("on the listener: Left=%d, Right=%d, want them equal", on.Left, on.Right)
	}
	if on.Left != GainUnit {
		t.Fatalf("on the listener at distance zero: Left=%d, want GainUnit (%d)", on.Left, GainUnit)
	}
}

func TestPlaceRefusesAtOrBeyondFalloff(t *testing.T) {
	if _, ok := Place(0, FalloffCells); ok {
		t.Fatalf("Place(0, %d): played, want refused (at the radius)", FalloffCells)
	}
	if _, ok := Place(0, FalloffCells+50); ok {
		t.Fatal("Place beyond the radius: played, want refused")
	}
	if _, ok := Place(0, FalloffCells-1); !ok {
		t.Fatalf("Place(0, %d): refused, want a play (just inside the radius)", FalloffCells-1)
	}
}

// Pan saturates rather than overshooting GainUnit: a cell straight out to one
// side, close enough that the unclamped algebra would exceed GainUnit (see
// clampGain's own comment), still reports a Placement inside [0, GainUnit].
func TestPlaceClampsGainToTheUnit(t *testing.T) {
	p, ok := Place(-1, 0)
	if !ok {
		t.Fatal("Place(-1, 0): refused, want a play")
	}
	if p.Left < 0 || p.Left > GainUnit || p.Right < 0 || p.Right > GainUnit {
		t.Fatalf("Placement out of [0, %d]: %+v", GainUnit, p)
	}
}
