package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestWithdrawalTriggerGivesWimpyPriorityAndUsesItsOwnRadius(t *testing.T) {
	var rel sim.Relations
	rel.Set(1, 2, 1)
	w, err := sim.NewRelatedWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{
			{ID: 1, X: 10, Y: 10, HP: 8, MaxHP: 100, Owner: 1, Withdraw: 30, Wimpy: 10, ScanRange: 7},
			{ID: 2, X: 16, Y: 10, HP: 100, MaxHP: 100, Owner: 2},
		}, nil, rel)
	if err != nil {
		t.Fatal(err)
	}
	mode, threshold, radius, hostiles := withdrawalTrigger(w, w.Entities()[0])
	if mode != "wimpy" || threshold != 10 || radius != 7 || len(hostiles) != 1 {
		t.Fatalf("trigger = %q %d radius %d with %d hostile(s)", mode, threshold, radius, len(hostiles))
	}
}

func TestWithdrawalWitnessRunsTheProductionPhaseSixTail(t *testing.T) {
	var rel sim.Relations
	rel.Set(1, 2, 1)
	w, err := sim.NewRelatedWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{
			{ID: 1, X: 20, Y: 20, HP: 100, MaxHP: 100, Owner: 1, Withdraw: 30, ScanRange: 5,
				Reach: 2, DamageBase: 1, AlwaysHits: true, Speed: 256, RotationSpeed: 16},
			{ID: 2, X: 18, Y: 20, HP: 100, MaxHP: 100, Owner: 2},
		}, nil, rel)
	if err != nil {
		t.Fatal(err)
	}
	ms := &game.Mission{Number: 70, Address: "scenario/70.alm", World: w}
	var out bytes.Buffer
	if err := driveWithdrawal(ms, nil, &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"mission=70", "mode=withdraw", "before-tail tick=6", "attack=2", "after  tick=7", "target=(23,20)",
		// 1047: RotationSpeed is an input to this fixture, printed verbatim
		// rather than derived, so it is a safe literal to pin here. The
		// facing/desired/remaining fields are printed too but their exact
		// values depend on production direction geometry this test does not
		// independently recompute; TestARaisedGhostInheritsAnInactiveTurn...
		// and pkg/sim's own turn1047_test.go pin the arc arithmetic itself.
		"rotationspeed=16", "facing=", "desired=", "remaining="} {
		if !strings.Contains(got, want) {
			t.Errorf("witness has no %q:\n%s", want, got)
		}
	}
}

// The release gate runs this once for each lawful root. It walks the real
// campaign until one mission supplies the hostile geometry the production tail
// needs, then requires the same witness the -withdrawal flag prints.
func TestReleaseRangedWithdrawalMissionWitness(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: ranged-withdrawal mission witness needs a lawful install")
	}
	archives, err := game.OpenArchives(root)
	if err != nil {
		t.Fatal(err)
	}
	defs, err := game.LoadDefinitions(archives.Containers)
	if err != nil {
		t.Fatal(err)
	}
	party := game.MissionParty(defs.StartWeapon, defs.Bodies, defs.Table)
	var misses []string
	for mission := 10; mission <= 280; mission += 10 {
		ms, err := game.StartMission(archives.Containers, mission, defs.Table, mapload.DifficultyNormal, party)
		if err != nil {
			t.Fatalf("mission %d: %v", mission, err)
		}
		var out bytes.Buffer
		if err := driveWithdrawal(ms, defs.Table, &out); err != nil {
			misses = append(misses, err.Error())
			continue
		}
		t.Log("\n" + out.String())
		return
	}
	t.Fatalf("no mission produced the witness; misses: %s", strings.Join(misses, "; "))
}
