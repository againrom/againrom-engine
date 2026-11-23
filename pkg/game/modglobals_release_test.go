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

// The globals mill mod's campaign over the lawful install: the mill's win
// carries its globals to mission 20, whose trigger and reply compare them.
func TestReleaseGlobalsMillCarriesToTheYard(t *testing.T) {
	f := releaseFront(t)
	base := BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS")))
	if _, err := launchModsOn(t, f, globalsMillFolder, base); err != nil {
		t.Fatal(err)
	}
	playMill(t, f)
	winIntoYard(t, f)
	if got := f.live.world.ScriptGlobals(); !reflect.DeepEqual(got, millWonGlobals) {
		t.Fatalf("mission 20 seated %v, want %v", got, millWonGlobals)
	}
	playYard(t, f)
	w := f.live.world
	if w.ScriptRegister(50) != 1 || w.ScriptRegister(51) != 1 {
		t.Fatalf("the yard's trigger and reply gave variables 50=%d 51=%d", w.ScriptRegister(50), w.ScriptRegister(51))
	}
	t.Logf("yard globals %v, World hash %016x", w.ScriptGlobals(), w.Hash())
}
