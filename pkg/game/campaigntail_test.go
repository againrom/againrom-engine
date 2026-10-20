package game

import (
	"image"
	"testing"
)

// The first-MapPoint word follows the map object's point, not its index: a
// mission whose object shares MapPoint 0's point sets it, a mission elsewhere
// does not, and a mission the registry does not map takes object 0
// (SAV-1331).
func TestMissionAtFirstMapPointComparesPoints(t *testing.T) {
	d := &globalMapData{
		Objects: []globalMapObject{
			{Point: image.Pt(215, 234), Valid: true},
			{Point: image.Pt(215, 234), Valid: true},
			{Point: image.Pt(90, 40), Valid: true},
			{Point: image.Pt(215, 234)},
		},
		Missions: map[int]int{10: 2, 20: 1, 30: 3},
	}
	for _, tc := range []struct {
		mission int
		want    bool
	}{{10, false}, {20, true}, {30, false}, {40, true}} {
		if got := d.missionAtFirstMapPoint(tc.mission); got != tc.want {
			t.Errorf("mission %d at MapPoint 0 = %v, want %v", tc.mission, got, tc.want)
		}
	}
	if (*globalMapData)(nil).missionAtFirstMapPoint(20) {
		t.Error("a missing map places a mission on MapPoint 0")
	}
}
