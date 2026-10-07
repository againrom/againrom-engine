package game

import (
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseAutoHealing1191MenuNativeAndOriginalSave(t *testing.T) {
	t.Run("carried_original_policy", autoHealingCarriedOriginal)
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	f.SetTipsOff(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "Healing witness", Choices: []int{1, 1, 3}, Stats: []int{31, 27, 24, 29}})
	a := f.App("global healing")
	a.SetCutscenes(nil)
	if err := a.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	a.Layout(640, 480)
	check := func(front *FrontEnd, percent uint32) {
		t.Helper()
		if got, present := front.live.world.AutoHealing(sim.SelfSlot); !present || got != percent {
			t.Fatalf("player policy %d/%v, want%d", got, present, percent)
		}
		casters := 0
		for _, e := range front.live.world.Entities() {
			s := e.ActorLoad.Source
			if e.Owner != sim.SelfSlot {
				continue
			}
			if s.Class != 0 && s.HasOwner {
				want := uint16(int32(int16(e.MaxMana)) * int32(percent) / 100)
				if s.ManaReservePercent != percent || s.ManaFloor != want {
					t.Fatalf("actor%d floor/percentage=%d/%d want%d/%d", e.ID, s.ManaFloor, s.ManaReservePercent, want, percent)
				}
			}
			if e.MaxMana > 0 && e.KnownSpells != 0 {
				casters++
			}
		}
		if casters == 0 {
			t.Fatal("installed party has no mage to witness mana floors")
		}
	}
	check(f, 50)
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	hash := f.live.world.Hash()
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, 420, 200); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.live.pending) != 0 {
		t.Fatal("the No radio queued its choice before OK", f.live.pending)
	}
	if err := a.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	if f.gameOptionValues(true)[ui.GameOptionAutoHealing] != 0 || f.live.world.Hash() != hash {
		t.Fatal("No radio did not queue its choice on the held world")
	}
	if dir := os.Getenv("AGAINROM_AUTOHEAL_REVIEW_DIR"); dir != "" {
		dir = filepath.Join(dir, filepath.Base(os.Getenv("AGAINROM_ASSETS")))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		out, err := os.Create(filepath.Join(dir, "autohealing-options.png"))
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(out, a.GameMenuPanel())
		closeErr := out.Close()
		if err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := EncodeSave(s, "paused healing policy")
	if err != nil {
		t.Fatal(err)
	}
	s, _, err = DecodeSave(raw)
	if err != nil {
		t.Fatal(err)
	}
	cold := releaseFront(t)
	cold.SetDeterministicFrames(true)
	cold.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	cold.SetTipsOff(true)
	if err := cold.Options.setGameOption(ui.GameOptionAutoHealing, 2); err != nil {
		t.Fatal(err)
	}
	ca := cold.App("cold healing")
	open, town, err := cold.Restore(s)
	if err != nil || town {
		t.Fatal(town, err)
	}
	if err = ca.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if f.live.world.Hash() != cold.live.world.Hash() || !reflect.DeepEqual(f.live.pending, cold.live.pending) || cold.gameOptionValues(true)[ui.GameOptionAutoHealing] != 0 {
		t.Fatal("native load used profile instead of the saved policy and queued No")
	}
	for i := 0; i < 64; i++ {
		f.live.tick()
		cold.live.tick()
		if f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatalf("native successor%d differs", i)
		}
	}
	check(f, 100)
	check(cold, 100)
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil || filepath.Ext(name) != ".sav" {
		t.Fatal("ordinary original SAVE", name, err)
	}
	raw, err = store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	imported := releaseFront(t)
	imported.Options = cold.Options
	imported.SetTipsOff(true)
	ia := imported.App("source healing")
	open, town, err = imported.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("original LOAD", town, err)
	}
	if err := ia.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	check(imported, 100)
	if imported.gameOptionValues(true)[ui.GameOptionAutoHealing] != 0 {
		t.Fatal("original F58 lost its No label")
	}
	// Ctrl+U uses the same production route as the radio. Advance only the
	// ordinary map frame, then prove Standard and its persisted profile value.
	if err := ia.HeadlessKey("ctrl-u"); err != nil {
		t.Fatal(err)
	}
	for len(imported.live.pending) != 0 {
		imported.live.tick()
	}
	check(imported, 50)
	stored, present, err := imported.Options.gameOptions()
	if err != nil || !present[ui.GameOptionAutoHealing] || stored[ui.GameOptionAutoHealing] != 1 {
		t.Fatal("Ctrl+U preference", stored, present, err)
	}
}

func autoHealingCarriedOriginal(t *testing.T) {
	f, _ := townReturnImported1168(t)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	var percent uint32
	found := false
	for _, member := range f.Carried {
		if member.Carry != nil && member.Carry.LiveLoad != nil {
			percent = member.Carry.LiveLoad.Inventory.Source.ManaReservePercent
			found = true
		}
	}
	if !found || percent == 50 {
		t.Fatal("fixture must carry its original non-default Player percentage", percent)
	}
	a := f.App("carried original autohealing")
	if err := a.OpenMission(f.MissionOpenerWith(30, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	if got, present := f.live.world.AutoHealing(sim.SelfSlot); !present || got != percent {
		t.Fatal("next mission replaced the saved Player percentage", got, present, percent)
	}
	for _, e := range f.live.world.Entities() {
		if e.Owner == sim.SelfSlot && e.ActorLoad.Source.HasOwner && e.ActorLoad.Source.ManaReservePercent != percent {
			t.Fatal("next mission overwrote the original actor policy", e.ID)
		}
	}
	// An explicit choice must remain writable together with the changed
	// Human floors when the same source party returns to town.
	if err := f.setGameOption(true, ui.GameOptionAutoHealing, 0); err != nil {
		t.Fatal(err)
	}
	f.LiveAdvance(1)
	if _, _, err := f.LiveCompleteCampaign(); err != nil {
		t.Fatal(err)
	}
	snapshot, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, _, err := f.playerCitySave(snapshot, "Changed healing")
	if err != nil {
		t.Fatal("changed healing city SAV", err)
	}
	g := releaseFront(t)
	if _, town, err := g.RestoreOriginal(raw); err != nil || !town {
		t.Fatal(town, err)
	}
	if got, present, err := carriedAutoHealing(g.Carried); err != nil || !present || got != 100 {
		t.Fatal("city Player percentage did not follow the changed actor floors", got, present, err)
	}
}
