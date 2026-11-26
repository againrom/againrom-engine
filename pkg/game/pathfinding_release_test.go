package game

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/sim"
)

// pathFindingMissionHash opens a mission of the installed game and steps it
// a fixed number of frames. With cleared the table names no [Path Finding]
// values, which is the world a build without the reader ran.
func pathFindingMissionHash(t *testing.T, cleared bool) (*sim.PathFinding, uint64) {
	t.Helper()
	f := autoCastLevelFront(t)
	if cleared {
		f.Table.Build.Path = nil
	}
	app := autoCastLevelMission(t, f)
	for range 120 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	return f.live.world.BuildRules().Path, f.live.world.Hash()
}

// The shipped map.reg of either game holds the values the engine ran as
// constants, so a mission under it hashes as it did.
func TestReleaseShippedPathFindingKeepsTheMissionHash(t *testing.T) {
	loaded, withFile := pathFindingMissionHash(t, false)
	if loaded == nil || *loaded != sim.ShippedPathFinding() {
		t.Fatalf("the install's [Path Finding] values %+v, want the shipped %+v", loaded, sim.ShippedPathFinding())
	}
	none, without := pathFindingMissionHash(t, true)
	if none != nil {
		t.Fatalf("cleared table still carries %+v", none)
	}
	if withFile != without {
		t.Fatalf("mission hash %#x with the install's values, %#x without", withFile, without)
	}
}

// A map.reg that changes [Path Finding] values reaches the definitions of
// either game, and a key it leaves out holds its shipped value.
func TestReleaseModdedPathFindingReachesTheDefinitions(t *testing.T) {
	f := releaseFront(t)
	raw := synth.Reg(0, []synth.RegNode{{Name: "Path Finding", Kind: 1, Children: []synth.RegNode{
		{Name: "StaticIsntNeede", Kind: 2, Int: 9},
		{Name: "DynamicByStatic", Kind: 2, Int: 1},
		{Name: "StaticRefreshRa", Kind: 2, Int: 40},
	}}})
	forked := f.Archives.Containers.Fork()
	forked.Overlay(terrainCostsAddress, func() ([]byte, error) { return raw, nil })
	d, err := LoadDefinitionsOf(forked, f.Archives.Base.Profile)
	if err != nil {
		t.Fatal(err)
	}
	want := sim.ShippedPathFinding()
	want.StaticIsntNeeded, want.DynamicByStaticLookup, want.StaticRefreshRate = 9, 1, 40
	if got := d.Table.Build.Path; got == nil || *got != want {
		t.Fatalf("definitions carry %+v, want %+v", got, want)
	}
	if got, profile := d.Table.Build.DeathSackCap, f.Archives.Base.Profile.DeathSackCap; got != profile {
		t.Fatalf("the build's death bag cap %d, profile %d", got, profile)
	}
}
