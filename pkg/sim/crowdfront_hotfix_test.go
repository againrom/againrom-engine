package sim

import "testing"

// A crowd closing on one victim across a river fills the bank cells of the
// first ring around the victim it can reach, and the rest stand on the next
// ring: picker B takes the lowest label of the first ring that holds one, and
// a pursuer whose own cell is that lowest stops there (MOVE-099, MOVE-100).
func TestACrowdAtABankFillsTheFirstRingItCanReach(t *testing.T) {
	const width, height, bank = 50, 30, 23
	grid := make([]byte, width*height)
	for y := int32(0); y < height; y++ {
		for x := int32(bank + 1); x < bank+3; x++ {
			grid[y*width+x] = blockGround
		}
	}
	victim := engFighter(1, 3, 26, 15)
	victim.TokenSize = 1
	ents := []Entity{victim}
	for i := 0; i < 14; i++ {
		o := engFighter(EntityID(10+i), 2, int32(15+i/7), int32(12+i%7))
		o.ScanRange, o.Speed, o.TokenSize, o.Reach = 14, 20, 1, 1
		ents = append(ents, o)
	}
	w, err := NewRelatedWorld(1, Bounds{Width: width, Height: height}, ModeCanonical, Terrain{Block: grid}, ents, nil,
		engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 1}))
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	for k := 0; k < 600; k++ {
		Step(w, nil)
	}
	taken := map[int32]bool{}
	for _, e := range w.entities[1:] {
		ring := (cell{x: e.X, y: e.Y}).chebyshevTo(cell{x: 26, y: 15})
		if ring != 3 && ring != 4 || !e.PursuitIdle {
			t.Errorf("attacker %d stands at (%d,%d), ring %d, idle %v; want it idle on ring 3 or 4", e.ID, e.X, e.Y, ring, e.PursuitIdle)
		}
		if e.X == bank {
			taken[e.Y] = true
		}
	}
	for y := int32(12); y <= 18; y++ {
		if !taken[y] {
			t.Errorf("the ring-3 bank cell (%d,%d) is free", bank, y)
		}
	}
}
