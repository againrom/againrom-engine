package sim

import "testing"

// The first tick after a native load, before any area pass.

func acLoaded(t *testing.T, w *World) *World {
	t.Helper()
	var back World
	if err := back.UnmarshalBinary(acMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	return &back
}

func acStepOnto(t *testing.T, w *World) uint16 {
	t.Helper()
	Step(w, []Command{acMove(1, 2, 1)})
	e := w.Entities()[0]
	if e.X != 2 || e.Y != 1 {
		t.Fatalf("the mover stands at (%d,%d), want the layered cell (2,1)", e.X, e.Y)
	}
	return e.TransitTotal
}

func acPair(t *testing.T, reads int) (continuing, loaded *World) {
	t.Helper()
	key := cellKey(2, 1)
	w := srWorld(t, srPlane(8), nil, []Entity{{ID: 1, X: 1, Y: 1, Speed: 16}})
	w.effects = []cellEffect{{Key: key, Spell: 3, Remaining: 200, Mode: areaModeCloud, Cells: []uint16{key}}}
	w.syncAreaCosts()
	acReads(w, cell{x: 2, y: 1}, reads)
	return w, acLoaded(t, w)
}

func TestAreaCostFirstTickAfterNativeLoadMatchesContinuing(t *testing.T) {
	t.Parallel()
	bare := srWorld(t, srPlane(8), nil, []Entity{{ID: 1, X: 1, Y: 1, Speed: 16}})
	bareTransit := acStepOnto(t, bare)
	for _, reads := range []int{0, 1, 2} {
		continuing, loaded := acPair(t, reads)
		if loaded.areaCostLive {
			t.Fatalf("%d prior reads: a loaded world claims a first area pass it has not run", reads)
		}
		c, l := acStepOnto(t, continuing), acStepOnto(t, loaded)
		if c != l {
			t.Errorf("%d prior reads: the continuing mover's transit is %d, the loaded mover's %d", reads, c, l)
		}
		// Loss control: a dropped layer reads the bare baseline; only a prior
		// read tells a layer from bare ground.
		if reads > 0 && l == bareTransit {
			t.Errorf("%d prior reads: the loaded transit %d equals bare ground's, so the layer was lost or the witness is blind", reads, l)
		}
	}
}

// A layer first present during a tick is read as no layer by that tick's
// movers, in a continuing world and in a loaded one alike. A loaded world holds
// an entry for the layer already standing, so the second layer's cast in the
// first tick after LOAD does not reach the byte its movers read. Without the
// entry the second layer is read at once.
func TestAreaCostSecondLayerInFirstTickAfterLoadIsReadAsNoLayer(t *testing.T) {
	t.Parallel()
	key := cellKey(2, 1)
	second := cellEffect{Key: key, Spell: 12, Remaining: 200, Mode: areaModeCloud, Cells: []uint16{key}}
	bareDiffers := false
	for _, reads := range []int{0, 1, 2} {
		build := func(kind string) *World {
			w, _ := acPair(t, reads)
			switch kind {
			case "reset":
				w = acLoaded(t, w)
				w.ResetLoadedAreaCosts()
			case "bare":
				w = acLoaded(t, w)
			}
			return w
		}
		plain := acStepOnto(t, build("continuing"))
		for _, kind := range []string{"continuing", "reset", "bare"} {
			w := build(kind)
			w.effects = append(w.effects, second)
			got := acStepOnto(t, w)
			switch {
			case kind != "bare" && got != plain:
				t.Errorf("%d prior reads, %s world: the first-tick second layer changed the transit from %d to %d", reads, kind, plain, got)
			case kind == "bare" && got != plain:
				bareDiffers = true
			}
		}
	}
	if !bareDiffers {
		t.Error("a restore without the reset reads like the others, so the witness is blind")
	}
}
