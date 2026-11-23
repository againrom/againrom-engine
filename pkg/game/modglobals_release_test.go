//go:build release

package game

import (
	"math"
	"os"
	"reflect"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The globals mill mod plays over the lawful install: its mission seats the
// declared initials, the setup trigger sets and adds with a wrap, the reply
// adds, and the later trigger compares the globals and acts.
func TestReleaseGlobalsMillPlaysOnTheInstall(t *testing.T) {
	f := releaseFront(t)
	if _, err := launchModsOn(t, f, globalsMillFolder, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS")))); err != nil {
		t.Fatal(err)
	}
	w := openExtendedMission(t, f, 0)
	if got, want := w.ScriptGlobals(), []sim.ScriptGlobal{{ID: 1}, {ID: 2, Value: 5}, {ID: 3, Value: math.MaxInt32 - 1}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("seated %v, want %v", got, want)
	}
	missionSteps(f.live, 3)
	w = f.live.world
	if got, want := w.ScriptGlobals(), []sim.ScriptGlobal{{ID: 1, Value: 1}, {ID: 2, Value: 7}, {ID: 3, Value: math.MinInt32}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("after the setup %v, want %v", got, want)
	}
	if f.live.mission.dialogue == nil || w.ScriptRegister(50) != 0 {
		t.Fatal("the mill dialogue did not open, or the comparison held before the reply")
	}
	f.live.advanceNotice(ui.NoticeReply1)
	missionSteps(f.live, 2)
	w = f.live.world
	if w.ScriptGlobals()[1].Value != 10 || w.ScriptRegister(50) != 1 {
		t.Fatalf("after the reply globals %v, variable 50 at %d", w.ScriptGlobals(), w.ScriptRegister(50))
	}
	t.Logf("globals %v, World hash %016x", w.ScriptGlobals(), w.Hash())
}
