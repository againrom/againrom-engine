package sim

import "testing"

// An attacker walking up to a victim behind blocked cells, the nearest
// reachable cell five rings from the victim, is not refused at seven cells and
// reaches the bank.
func TestAttackerWalksToTheBankBehindWhichItsVictimStands(t *testing.T) {
	const width, height, bank, victimX = 100, 30, 72, 77
	grid := make([]byte, width*height)
	for y := int32(0); y < height; y++ {
		for x := int32(bank + 1); x < victimX-1; x++ {
			grid[y*width+x] = blockGround
		}
	}
	victim := engFighter(1, 3, victimX, 15)
	victim.TokenSize = 1
	ents := []Entity{victim}
	for i := 0; i < 3; i++ {
		o := engFighter(EntityID(10+i), 2, 64, int32(14+i))
		o.ScanRange, o.Speed, o.TokenSize, o.Reach = 14, 20, 1, 5
		ents = append(ents, o)
	}
	w, err := NewRelatedWorld(1, Bounds{Width: width, Height: height}, ModeCanonical, Terrain{Block: grid}, ents, nil,
		engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 1}))
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	reached := false
	for k := 0; k < 700; k++ {
		Step(w, nil)
		for _, e := range w.entities[1:] {
			if e.PursuitIdle {
				t.Fatalf("tick %d: attacker %d idles at (%d,%d) with its victim kept", k, e.ID, e.X, e.Y)
			}
			reached = reached || e.X >= bank-1
		}
	}
	if !reached {
		t.Fatal("no attacker reached the bank")
	}
}
