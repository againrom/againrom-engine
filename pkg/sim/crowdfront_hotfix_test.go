package sim

import "testing"

// A crowd closing on one victim across a river fills the free bank cells
// nearest the victim; no attacker stands behind a taken cell while a free
// bank cell lies nearer the victim.
func TestAttackersFillTheFreeBankCellsNearestTheirVictim(t *testing.T) {
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
	gap := func(x, y int32) int32 { return (x-26)*(x-26) + (y-15)*(y-15) }
	taken := map[int32]bool{}
	for _, e := range w.entities[1:] {
		if e.X == bank {
			taken[e.Y] = true
		}
	}
	for _, e := range w.entities[1:] {
		if e.X >= bank {
			continue
		}
		for y := int32(0); y < height; y++ {
			if !taken[y] && gap(bank, y) < gap(e.X, e.Y) {
				t.Fatalf("attacker %d stands at (%d,%d) while the bank cell (%d,%d) nearer the victim is free", e.ID, e.X, e.Y, bank, y)
			}
		}
	}
}
