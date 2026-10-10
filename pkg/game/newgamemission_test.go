package game

import (
	"testing"

	"againrom/pkg/base"
)

// The mission a new game opens is the campaign's own first main mission, and
// a profile that names its first mission keeps it.
func TestNewGameMissionIsTheRegistryFirstMission(t *testing.T) {
	c := Campaign{Main: []int{20, 30}, Side: []int{11}}
	if n, ok := c.FirstMission(); !ok || n != 20 {
		t.Fatalf("FirstMission = %d %t, want 20", n, ok)
	}
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}}
	if got := f.NewGameMission(); got != 20 {
		t.Fatalf("NewGameMission = %d, want the registry's first main mission 20", got)
	}
	none := &FrontEnd{}
	if got := none.NewGameMission(); got != base.DefaultFirstMission {
		t.Fatalf("NewGameMission with no campaign = %d, want the profile default", got)
	}
	var nilFront *FrontEnd
	if got := nilFront.NewGameMission(); got != base.DefaultFirstMission {
		t.Fatalf("nil front end = %d", got)
	}
}
