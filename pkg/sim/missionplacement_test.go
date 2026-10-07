package sim

import (
	"reflect"
	"testing"
)

func TestMissionPlacementOriginalCandidateControls(t *testing.T) {
	for _, c := range []struct {
		population int
		radius     int32
	}{
		{0, 5}, {1, 5}, {3, 5}, {4, 6}, {9, 7}, {63, 11}, {63503, 255}, {63504, 0},
	} {
		if got := missionPlacementRadius(c.population); got != c.radius {
			t.Errorf("population %d radius %d, want %d", c.population, got, c.radius)
		}
	}
	draws := 0
	draw := func(limit int32) int32 {
		if limit != 32767 {
			t.Fatalf("random source limit %d", limit)
		}
		draws++
		if draws%2 == 1 {
			return 0
		}
		return 32767
	}
	x, y, ok := missionPlacementCell(20, 30, 5, draw, func(x, y int32) bool { return true })
	if !ok || x != 23 || y != 28 || draws != 2 {
		t.Fatalf("y-first endpoint control: (%d,%d), success %v, draws %d", x, y, ok, draws)
	}

	draws = 0
	var attempts [][2]int32
	_, _, ok = missionPlacementCell(20, 30, 5, draw, func(x, y int32) bool {
		attempts = append(attempts, [2]int32{x, y})
		return false
	})
	if ok || draws != 28 || len(attempts) != 39 {
		t.Fatalf("refusal control: success %v, draws %d, attempts %d", ok, draws, len(attempts))
	}
	want := [][2]int32{{18, 28}, {18, 29}, {18, 30}, {18, 31}, {18, 32}, {19, 28}}
	if !reflect.DeepEqual(attempts[14:20], want) || attempts[38] != [2]int32{22, 32} {
		t.Fatalf("fallback order or odd-radius extent: %v", attempts[14:])
	}
	draws = 0
	attempts = nil
	missionPlacementCell(20, 30, 0, draw, func(x, y int32) bool {
		attempts = append(attempts, [2]int32{x, y})
		return false
	})
	if draws != 0 || !reflect.DeepEqual(attempts, [][2]int32{{20, 30}, {20, 30}}) {
		t.Fatalf("radius zero consumed randomness or searched: %v / %d draws", attempts, draws)
	}
}

func TestMissionPlacementPreservesSavedCellsAndRespectsWholeFootprints(t *testing.T) {
	b := Bounds{Width: 40, Height: 40}
	grid := make([]byte, 1600)
	grid[21*40+21] = blockGround
	ents := []Entity{
		{ID: 0, Owner: 2, X: 20, Y: 20, HP: 100, MaxHP: 100},
		{ID: 1, Owner: SelfSlot, TokenSize: 2, HP: 100, MaxHP: 100},
		{ID: 2, Owner: SelfSlot, X: 21, Y: 21, HP: 100, MaxHP: 100},
		{ID: 3, Owner: SelfSlot, HP: 100, MaxHP: 100},
	}
	original := append([]Entity(nil), ents...)
	if refused := PlaceMissionParty(b, grid, ents, 1, []bool{false, true, false}, 20, 20, NewDraws(1)); refused != 0 {
		t.Fatalf("unexpected refused members: %d", refused)
	}
	if ents[0] != original[0] || ents[2] != original[2] {
		t.Fatal("authored or saved actor was moved, including the saved blocked cell")
	}
	for _, i := range []int{1, 3} {
		e := ents[i]
		if e.OffMap || !terrainOpenFootprintIn(b, grid, e, e.X, e.Y) {
			t.Fatalf("actor %d not placed on fully open ground: %+v", i, e)
		}
		for j, other := range ents {
			if i != j && footprintsOverlap(e.X, e.Y, e.TokenSize, other.X, other.Y, other.TokenSize) {
				t.Fatalf("actor %d overlaps actor %d", i, j)
			}
		}
	}
	PlaceMissionParty(b, grid, original, 1, []bool{false, true, false}, 20, 20, NewDraws(1))
	if !reflect.DeepEqual(original, ents) {
		t.Fatal("same inputs produced different placement")
	}
}
