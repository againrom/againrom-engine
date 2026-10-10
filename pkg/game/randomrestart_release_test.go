package game

import (
	"fmt"
	"image"
	"testing"
	"time"

	"againrom/pkg/base"
	"againrom/pkg/random"
	"againrom/pkg/ui"
)

// townFront is a front end in the town square under launch seed in the given
// mode, on a synthetic town clock, with its App open on the square.
func townFront(t *testing.T, seed uint64, original bool) (*FrontEnd, *ui.App, *townTrace, image.Point) {
	t.Helper()
	f, err := NewFrontEnd(releaseRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if p := f.Base().Profile; p.Limits.NoCharacterGeneration || p.Edition().Campaign == base.CampaignDestinations || TownDescription(p) == nil {
		t.Skip("the base has no composed town square to dwell in")
	}
	cleanupFrontAudio(t, f)
	f.SetRandomLaunch(seed, true, original)
	t.Cleanup(func() { ui.SetOriginalItemStars(false) })
	f.SetDeterministicFrames(true)
	now := time.Unix(5000, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.PersistenceContext.tipsOff = true
	sounds := &traceSounds{}
	f.SoundPlayer, f.SpeechPlayer, f.AmbientPlayer = sounds, sounds, sounds
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Dwell fighter", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 20, 20}})
	f.arriveInTown()
	_, off := traceMaskPoints(t, f)
	a := f.App("town dwell")
	t.Cleanup(a.StopAudio)
	a.Layout(640, 480)
	a.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town.sav", Label: "Town"}} },
		func(string) (ui.MapOpener, bool, error) { return nil, true, nil })
	if err := a.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	return f, a, &townTrace{t: t, f: f, a: a, now: &now, sound: sounds}, off
}

// dwellThenMission dwells frames town frames in original mode at seed 7, then
// opens mission 41 from the town in the running session and runs it. It
// answers the shared state the dwell left and the World hash at the start
// and after 300 ticks.
func dwellThenMission(t *testing.T, frames int) (uint32, []uint64) {
	t.Helper()
	f, a, tr, off := townFront(t, 7, true)
	if f.randomService().Mode() != random.Original {
		t.Skip("this game runs the default mode under the original switch (DIV-2748)")
	}
	tr.hover(off, 41*time.Millisecond, frames)
	shared := f.randomService().SharedState()
	if err := a.OpenMission(f.MissionOpener(41)); err != nil {
		t.Fatal(err)
	}
	for n := 0; a.HeadlessNoticeOpen() && n < 16; n++ {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	world := []uint64{f.live.world.Hash()}
	for i := 0; i < 300; i++ {
		f.live.tick()
	}
	return shared, append(world, f.live.world.Hash())
}

// TestReleaseOriginalRandomTownTimeStaysOutOfTheWorld proves that the time a
// player spends in town does not reach the next mission in original mode: two
// dwells of different length move the shared stream differently, and the
// mission opened after each starts and runs to the same World hashes
// (DIV-2730).
func TestReleaseOriginalRandomTownTimeStaysOutOfTheWorld(t *testing.T) {
	shortShared, short := dwellThenMission(t, 3)
	longShared, long := dwellThenMission(t, 400)
	if shortShared == longShared {
		t.Fatal("the two dwells left one shared state; the witness is empty")
	}
	if fmt.Sprint(short) != fmt.Sprint(long) {
		t.Fatalf("the town dwell reached the World: %x against %x", short, long)
	}
	t.Logf("dwells of 3 and 400 frames left shared states %#x and %#x; World %016x at start, %016x after 300 ticks", shortShared, longShared, short[0], short[1])
}

// wildlifeDraws answers n draws of the town square's wildlife source.
func wildlifeDraws(f *FrontEnd, n int) []int {
	v := f.TownScreen().(*townScreen).squareView()
	out := make([]int, n)
	for i := range out {
		out[i] = v.Pick("wildlife", "scaled", 2000)
	}
	return out
}

// TestReleaseTownWildlifeRestartsWithTheSession proves that a LOAD inside a
// running process restarts the town composer's wildlife generator as a cold
// LOAD of the same SAV starts it: after the running session has drawn, the
// draws after its LOAD are the cold process's.
func TestReleaseTownWildlifeRestartsWithTheSession(t *testing.T) {
	source, _, _, _ := townFront(t, 7, false)
	snapshot, _, err := source.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	saved, _, err := source.playerCitySave(snapshot, "Wildlife")
	if err != nil {
		t.Fatal(err)
	}

	running, _, _, _ := townFront(t, 7, false)
	before := wildlifeDraws(running, 40)
	if _, town, err := running.RestoreOriginal(saved); err != nil || !town {
		t.Fatalf("LOAD in the running process: town %t, %v", town, err)
	}
	inProcess := wildlifeDraws(running, 16)

	cold, _, _, _ := townFront(t, 99, false)
	if _, town, err := cold.RestoreOriginal(saved); err != nil || !town {
		t.Fatalf("cold LOAD: town %t, %v", town, err)
	}
	fresh := wildlifeDraws(cold, 16)

	if fmt.Sprint(inProcess) != fmt.Sprint(fresh) {
		t.Fatalf("a LOAD in a running process drew %v, a cold LOAD %v", inProcess, fresh)
	}
	t.Logf("after 40 draws from %v, a LOAD in the running process drew %v, as the cold LOAD did", before[:4], inProcess[:4])
}
