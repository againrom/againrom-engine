package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The owner's mission: its map places both dragons, Wimpy 63
// (AI-WITHDRAW-027), and ranged classes that carry a Withdraw.
const withdrawalWitnessMission = 131

// TestReleaseSavedMonstersKeepWithdrawalThresholds starts the installed
// mission through the player's opener, saves through F2 and reads every
// actor's Order +0x40/+0x44 from the written file. Each holds the actor's
// current Withdraw and Wimpy, so the original's withdrawal test reads the
// authored thresholds after it loads the file. A cold LOAD and a second
// SAVE keep them.
func TestReleaseSavedMonstersKeepWithdrawalThresholds(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "FleeProbe", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("withdrawal thresholds")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	f.ConfigureSaveSeams(app, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
	if err := app.OpenMission(f.missionOpener(withdrawalWitnessMission, party, nil, nil, nil)); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	entities := f.live.world.Entities()
	wimpy, withdraw, dragons := 0, 0, 0 // dragons: actors at Wimpy 63
	for _, e := range entities {
		if e.Wimpy > 0 {
			wimpy++
		}
		if e.Withdraw > 0 {
			withdraw++
		}
		if e.Wimpy == 63 {
			dragons++
		}
	}
	if wimpy == 0 || withdraw == 0 || dragons == 0 {
		t.Fatalf("mission %d starts %d actors with Wimpy, %d with Withdraw, %d with Wimpy 63; want each nonzero", withdrawalWitnessMission, wimpy, withdraw, dragons)
	}

	check := func(when string, raw []byte, world *sim.World) {
		t.Helper()
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		a, err := readCurrentActions(&doc)
		if err != nil || a == nil {
			t.Fatal("action supplement", err)
		}
		savedWimpy, savedWithdraw, either := 0, 0, 0
		for _, e := range world.Entities() {
			order := savedRecordRawForTest(t, castOrderRecord(t, doc, a, e.ID), "U158")
			gotWithdraw, gotWimpy, ok := sav.OrderWithdrawalThresholds(order)
			if !ok || gotWithdraw != e.Withdraw || gotWimpy != e.Wimpy {
				t.Fatalf("%s: actor %d (map unit %d) Order +0x40/+0x44 = %d/%d, want %d/%d", when, e.ID, e.MapUnitID, gotWithdraw, gotWimpy, e.Withdraw, e.Wimpy)
			}
			if gotWimpy > 0 {
				savedWimpy++
			}
			if gotWithdraw > 0 {
				savedWithdraw++
			}
			if gotWimpy != 0 || gotWithdraw != 0 {
				either++
			}
		}
		if savedWimpy != wimpy || savedWithdraw != withdraw {
			t.Fatalf("%s: the file holds %d nonzero Wimpy and %d nonzero Withdraw, want %d and %d", when, savedWimpy, savedWithdraw, wimpy, withdraw)
		}
		t.Logf("%s: %d actors, %d with Wimpy (%d at 63), %d with Withdraw, %d with either", when, len(world.Entities()), savedWimpy, dragons, savedWithdraw, either)
	}
	_, saved := castOrderF2Save(t, f, app, "flee")
	check("first SAVE", saved, f.live.world)

	cold := releaseFront(t)
	open, town, err := cold.RestoreOriginal(saved)
	if err == nil && !town {
		err = cold.App("withdrawal cold LOAD").OpenMission(open)
	}
	if err != nil || town {
		t.Fatal("cold LOAD", err)
	}
	for _, e := range cold.live.world.Entities() {
		for _, before := range entities {
			if before.ID == e.ID && (before.Withdraw != e.Withdraw || before.Wimpy != e.Wimpy) {
				t.Fatalf("cold LOAD: actor %d thresholds %d/%d, want %d/%d", e.ID, e.Withdraw, e.Wimpy, before.Withdraw, before.Wimpy)
			}
		}
	}
	second, _, _ := saveCurrentEffect(t, cold)
	check("second SAVE", second, cold.live.world)
}

// TestReleaseSecondGameSaveKeepsOrderThresholdBytes saves the second game's
// first mission and reads every actor's Order +0x40..+0x47. No second-game
// claim places Withdraw or Wimpy there, so the SAVE leaves those bytes as the
// writer built them before the first-game threshold pass existed: zero for
// each actor the engine placed, though actors carry thresholds.
func TestReleaseSecondGameSaveKeepsOrderThresholdBytes(t *testing.T) {
	secondGameRoot(t)
	f, _ := secondMissionSettled(t)
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, "second-game thresholds")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatal("action supplement", err)
	}
	actors, carrying := 0, 0
	for _, e := range f.live.world.Entities() {
		order := savedRecordRawForTest(t, castOrderRecord(t, doc, a, e.ID), "U158")
		for _, v := range order[0x40:0x48] {
			if v != 0 {
				t.Fatalf("actor %d (Withdraw %d, Wimpy %d) Order +0x40..+0x47 = % x, want the zeros the second game's SAVE writes", e.ID, e.Withdraw, e.Wimpy, order[0x40:0x48])
			}
		}
		actors++
		if e.Withdraw != 0 || e.Wimpy != 0 {
			carrying++
		}
	}
	if carrying == 0 {
		t.Fatal("no actor of the second game's first mission carries a threshold; the control proves nothing")
	}
	t.Logf("%d actors, %d carry thresholds, every Order +0x40..+0x47 zero", actors, carrying)
}
