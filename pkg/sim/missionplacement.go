package sim

// PlaceMissionParty seats newly constructed party actors in list order. The
// caller owns entities and supplies the same terrain used to construct World.
// Entries marked preserved retain their saved positions. Refused actors remain
// in the list with OffMap set; they do not occupy the drop cell.
//
// MISSION-SCATTER-053 supplies radius, candidate order, retries and fallback.
// Admission uses our existing full-footprint rules. Draws retains the engine's
// deterministic seed; the original RNG lifecycle is not reconstructed here.
func PlaceMissionParty(b Bounds, grid []byte, entities []Entity, first int, preserved []bool, x, y int32, draws *Draws) int {
	view := World{bounds: b, grid: grid, entities: entities}
	population := 0
	for i := range entities {
		if entities[i].Owner == SelfSlot {
			population++
		}
		if i >= first && !preserved[i-first] {
			entities[i].OffMap = true
		}
	}
	radius := missionPlacementRadius(population)
	refused := 0
	for i := first; i < len(entities); i++ {
		if preserved[i-first] {
			continue
		}
		admit := func(cx, cy int32) bool { return view.placementOpen(i, cx, cy) }
		cx, cy, ok := x, y, false
		if i == first {
			cx, cy, ok = missionPlacementCell(x, y, 0, draws.Upto, admit)
		}
		if !ok {
			cx, cy, ok = missionPlacementCell(x, y, radius, draws.Upto, admit)
		}
		entities[i].X, entities[i].Y, entities[i].OffMap = cx, cy, !ok
		if !ok {
			refused++
		}
	}
	return refused
}

func missionPlacementRadius(population int) int32 {
	// floor(sqrt(n)+4) = floor(sqrt(n))+4. Integer arithmetic keeps
	// this constructor inside the simulation's deterministic numeric boundary.
	n, root := int64(population), int64(0)
	for bit := int64(1) << 31; bit != 0; bit >>= 1 {
		candidate := root | bit
		if candidate <= n/candidate {
			root = candidate
		}
	}
	return int32(uint8(max(5, root+4)))
}

func missionPlacementCell(x, y, radius int32, draw func(int32) int32, admit func(int32, int32) bool) (int32, int32, bool) {
	half := radius / 2
	axis := func() int32 {
		if radius == 0 {
			return 0
		}
		// The original scales a 15-bit draw to inclusive 0..radius.
		return (draw(32767)*(radius+1))>>15 - half
	}
	for attempt := int32(0); attempt < radius*radius/2+2; attempt++ {
		cy := y + axis()
		cx := x + axis()
		if admit(cx, cy) {
			return cx, cy, true
		}
	}
	if radius != 0 {
		for cx := x - half; cx <= x+half; cx++ {
			for cy := y - half; cy <= y+half; cy++ {
				if admit(cx, cy) {
					return cx, cy, true
				}
			}
		}
	}
	return x, y, false
}
