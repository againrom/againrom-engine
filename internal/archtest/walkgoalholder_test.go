package archtest

import (
	"os"
	"strings"
	"testing"
)

// The walk goal's holder is found by the shared occupant query, which reads
// the footprint and the motion cell; walkGoalHeldByEnemy never compares an
// entity's anchor to the goal cell itself.
func TestWalkGoalHolderUsesTheOccupantQuery(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	src, ok := files["pkg/sim/standground.go"]
	if !ok {
		t.Fatal("pkg/sim/standground.go is missing; the scan guards nothing")
	}
	start := strings.Index(src, "func (w *World) walkGoalHeldByEnemy(")
	if start < 0 {
		t.Fatal("walkGoalHeldByEnemy is missing; the scan guards nothing")
	}
	body := src[start:]
	if end := strings.Index(body[1:], "\nfunc "); end >= 0 {
		body = body[:end+1]
	}
	if !strings.Contains(body, "w.cellLayerOccupants(") {
		t.Error("walkGoalHeldByEnemy must ask cellLayerOccupants for the goal cell")
	}
	for _, bad := range []string{"o.X ==", "o.Y ==", ".X == e.TargetX", ".Y == e.TargetY"} {
		if strings.Contains(body, bad) {
			t.Errorf("walkGoalHeldByEnemy compares an anchor to the goal (%q); use the occupant query", bad)
		}
	}
}
