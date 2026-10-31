package archtest

import (
	"os"
	"testing"
)

// creatureAimFiles hold the creature slot cast; only creatureAimProfile asks
// the game's identity, and every game rule there is a row of that profile.
var creatureAimFiles = []string{"pkg/sim/creaturecast.go", "pkg/sim/creaturespell.go"}

func TestCreatureCastsReadTheCreatureAimProfile(t *testing.T) {
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
	for _, f := range creatureAimFiles {
		if _, ok := files[f]; !ok {
			t.Fatalf("%s is missing; the scan guards nothing", f)
		}
	}
	if calls := groupRouteIdentityCalls(t, files, creatureAimFiles); len(calls) != 1 {
		t.Errorf("the creature cast asks the game's identity at %v, want only creatureAimProfile", calls)
	}
}
