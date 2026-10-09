package game

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseGameOptions1186PreferencesAndCheckpoints(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	want := ui.GameOptionValues{0, 0, 0, 2, 2}
	seed := func(s OptionsStore, v ui.GameOptionValues) {
		t.Helper()
		for i, value := range v {
			if err := s.setGameOption(ui.GameOption(i), value); err != nil {
				t.Fatal(err)
			}
		}
	}
	seed(f.Options, want)
	f.LoadOptions()
	party := f.ChargenParty(ui.ChargenResult{Name: "Options witness", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	a := f.App("options")
	a.SetCutscenes(nil)
	if err := a.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	a.Layout(640, 480)
	if f.gameOptionValues(true) != want || f.live.world.Tick() != 0 || len(f.live.pending) != 2 {
		t.Fatal("fresh mission did not read preferences without stepping", f.gameOptionValues(true), f.live.world.Tick(), f.live.pending)
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	pendingStore := SaveStore{Dir: t.TempDir()}
	pendingSave, _, _ := f.SaveSeams(pendingStore, OriginalStore{}, nil)
	beforeSave := f.live.world.Hash()
	pendingName, err := pendingSave(true)
	if err != nil || filepath.Ext(pendingName) != ".sav" || f.live.world.Hash() != beforeSave {
		t.Fatal("queued choices did not write ordinary SAV without stepping", pendingName, err)
	}
	pendingRaw, err := pendingStore.Read(pendingName)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := EncodeSave(s, "options checkpoint")
	if err != nil {
		t.Fatal(err)
	}
	s, _, err = DecodeSave(raw)
	if err != nil {
		t.Fatal(err)
	}
	opposite := ui.GameOptionValues{1, 1, 1, 0, 0}
	savedOptions := releaseFront(t)
	savedOptions.SetDeterministicFrames(true)
	savedOptions.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	seed(savedOptions.Options, opposite)
	savedOptions.LoadOptions()
	pendingOpen, pendingTown, err := savedOptions.RestoreOriginal(pendingRaw)
	if err != nil || pendingTown {
		t.Fatal("queued options SAV LOAD", pendingTown, err)
	}
	if err := savedOptions.App("cold SAV options").OpenMission(pendingOpen); err != nil {
		t.Fatal(err)
	}
	if savedOptions.gameOptionValues(true) != want || !reflect.DeepEqual(savedOptions.live.pending, f.live.pending) || savedOptions.live.world.Tick() != 0 {
		t.Fatal("SAV lost pending choices or applied preferences on LOAD", savedOptions.gameOptionValues(true), savedOptions.live.pending)
	}
	cold := releaseFront(t)
	cold.SetDeterministicFrames(true)
	cold.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	seed(cold.Options, opposite)
	cold.LoadOptions()
	ca := cold.App("cold options")
	open, town, err := cold.Restore(s)
	if err != nil || town {
		t.Fatal("restore AGS", town, err)
	}
	if err := ca.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if cold.gameOptionValues(true) != want || !reflect.DeepEqual(cold.live.pending, f.live.pending) {
		t.Fatal("profile overwrote loaded AGS settings or pending orders", cold.gameOptionValues(true), cold.live.pending)
	}
	for _, withProfile := range []bool{false, true} {
		fresh := releaseFront(t)
		fresh.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
		if withProfile {
			seed(fresh.Options, opposite)
		}
		fresh.LoadOptions()
		fa := fresh.App("loaded session then new game")
		fa.SetCutscenes(nil)
		loaded, town, err := fresh.Restore(s)
		if err != nil || town {
			t.Fatal(err)
		}
		if err := fa.OpenMission(loaded); err != nil || fresh.gameOptionValues(true)[ui.GameOptionRetreat] != 2 {
			t.Fatal("setup did not restore Medium retreat", err)
		}
		result := ui.ChargenResult{Name: "New options", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}}
		if err := fa.OpenMission(fresh.NewGameOpener(10, result)); err != nil {
			t.Fatal("actual NEW GAME path", err)
		}
		application := fresh.live.applicationState
		if fresh.gameOptionValues(true)[ui.GameOptionRetreat] != 0 || application == nil || application.Original.Wimpy != 0 || application.WimpyBaseline != 0 {
			t.Fatal("NEW GAME inherited loaded retreat mode", withProfile, fresh.gameOptionValues(true), application)
		}
		for _, c := range fresh.live.pending {
			if c.Kind == sim.KindPlayerParameter && sim.PlayerParameter(c.X) == sim.PlayerParameterRetreat && c.Y != 0 {
				t.Fatal("fresh profile queued old retreat mode", withProfile, c)
			}
		}
	}
	for i := 0; i <= 64; i++ {
		x, _ := f.live.world.MarshalBinary()
		y, _ := cold.live.world.MarshalBinary()
		if !bytes.Equal(x, y) {
			t.Fatal("AGS continuation differs", i)
		}
		mode, present := f.live.world.CommandFormationMode(sim.SelfSlot)
		savedMode, savedPresent := savedOptions.live.world.CommandFormationMode(sim.SelfSlot)
		heal, healPresent := f.live.world.AutoHealing(sim.SelfSlot)
		savedHeal, savedHealPresent := savedOptions.live.world.AutoHealing(sim.SelfSlot)
		if mode != savedMode || present != savedPresent || heal != savedHeal || healPresent != savedHealPresent || savedOptions.gameOptionValues(true) != f.gameOptionValues(true) || !reflect.DeepEqual(savedOptions.live.pending, f.live.pending) {
			t.Fatal("SAV option state differs from uninterrupted control", i)
		}
		for _, actor := range f.live.world.Entities() {
			if actor.Owner != sim.SelfSlot {
				continue
			}
			back := optionsTwin(t, savedOptions, actor)
			if back.Withdraw != actor.Withdraw || back.Wimpy != actor.Wimpy || back.ActorLoad.Source.ManaFloor != actor.ActorLoad.Source.ManaFloor || back.ActorLoad.Source.ManaReservePercent != actor.ActorLoad.Source.ManaReservePercent {
				t.Fatal("SAV option command changed actor thresholds", i, actor.SourceBinding.RuntimeID)
			}
		}
		if i != 64 {
			wantApplied := 0
			if i == 0 {
				wantApplied = 2
			}
			if applied, savedApplied := f.live.tick(), savedOptions.live.tick(); applied != wantApplied || savedApplied != wantApplied {
				t.Fatal("SAV did not consume exactly the original queue once", i, applied, savedApplied)
			}
			cold.live.tick()
		}
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil || filepath.Ext(name) != ".sav" {
		t.Fatal("active retreat thresholds did not use current SAV", name, err)
	}
	retreatCold := loadAreaContinuation(t, filepath.Join(store.Dir, name))
	for _, actor := range f.live.world.Entities() {
		if actor.Owner == sim.SelfSlot {
			back := optionsTwin(t, retreatCold, actor)
			if back.Withdraw != actor.Withdraw || back.Wimpy != actor.Wimpy {
				t.Fatal("SAV lost active retreat thresholds")
			}
		}
	}
	if err := f.setGameOption(true, ui.GameOptionRetreat, 0); err != nil {
		t.Fatal(err)
	}
	f.live.tick()
	want[ui.GameOptionRetreat] = 0
	name, err = save(true)
	if err != nil || filepath.Ext(name) != ".sav" {
		t.Fatal("ordinary SAV", name, err)
	}
	savRaw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	original := releaseFront(t)
	original.Options = cold.Options
	original.LoadOptions()
	oa := original.App("original options")
	open, town, err = original.RestoreOriginal(savRaw)
	if err != nil || town {
		t.Fatal("restore SAV", town, err)
	}
	if err := oa.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	mode, bound := original.live.world.CommandFormationMode(sim.SelfSlot)
	if original.gameOptionValues(true) != want || len(original.live.pending) != 0 || !bound || mode != 1 {
		t.Fatal("profile overwrote SAV flags or queued an order", original.gameOptionValues(true), original.live.pending)
	}
	for _, e := range f.live.world.Entities() {
		other := optionsTwin(t, original, e)
		if e.Withdraw != other.Withdraw || e.Wimpy != other.Wimpy {
			t.Fatalf("saved retreat thresholds changed actor%d owner%d humanoid%v: %d/%d -> %d/%d", e.ID, e.Owner, e.Humanoid, e.Withdraw, e.Wimpy, other.Withdraw, other.Wimpy)
		}
	}
	if v, _, err := cold.Options.gameOptions(); err != nil || v != opposite {
		t.Fatal("LOAD overwrote process preferences", v, err)
	}
	// A new mission still uses the remembered profile after a checkpoint with
	// different settings has been loaded in this same process.
	if err := oa.OpenMission(original.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	if original.gameOptionValues(true) != opposite {
		t.Fatal("fresh mission inherited the checkpoint instead of preferences", original.gameOptionValues(true))
	}
	openMissionGameMenu(t, a)
	if err := a.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	before := f.live.world.Hash()
	beforeTick := f.live.world.Tick()
	stored := f.gameOptionValues(true)
	checkGem := func(picture *image.RGBA, roi image.Rectangle, frame int) {
		t.Helper()
		gem, err := loadTipGemFrame(f.Archives.Containers, tipGemPath, frame)
		if err != nil || picture == nil || gem.Bounds().Empty() {
			t.Fatal("installed rendered option gem absent", frame, err)
		}
		origin := roi.Min.Add(image.Pt((roi.Dx()-gem.Bounds().Dx())/2, (roi.Dy()-gem.Bounds().Dy())/2))
		compared := 0
		for y := range gem.Bounds().Dy() {
			for x := range gem.Bounds().Dx() {
				red, green, blue, alpha := gem.At(gem.Bounds().Min.X+x, gem.Bounds().Min.Y+y).RGBA()
				if alpha != 65535 {
					continue
				}
				point := origin.Add(image.Pt(x, y))
				if !point.In(roi) {
					t.Fatal("installed option gem exceeds independent ROI", frame, point, roi)
				}
				gotRed, gotGreen, gotBlue, gotAlpha := picture.At(point.X, point.Y).RGBA()
				if gotRed != red || gotGreen != green || gotBlue != blue || gotAlpha != alpha {
					t.Fatal("rendered control does not show the intended installed gem", frame, point)
				}
				compared++
			}
		}
		if compared == 0 {
			t.Fatal("installed option gem has no opaque oracle pixels", frame)
		}
	}
	// Choose Medium directly, then the three checkboxes, through drawn hit
	// boxes. The dialog works on a copy: nothing is stored until OK.
	for _, control := range []struct {
		name        string
		point       image.Point
		target, gem image.Rectangle
		off, on     int
	}{
		{"retreat Medium", image.Pt(416, 364), image.Rect(332, 350, 500, 374), image.Rect(333, 350, 357, 374), 0, 1},
		{"day/night", image.Pt(212, 154), image.Rect(116, 140, 326, 164), image.Rect(117, 140, 141, 164), 2, 3},
		{"health", image.Pt(440, 96), image.Rect(332, 84, 548, 108), image.Rect(333, 84, 357, 108), 2, 3},
		{"damage", image.Pt(440, 124), image.Rect(332, 112, 548, 136), image.Rect(333, 112, 357, 136), 2, 3},
	} {
		if !control.point.In(control.target) {
			t.Fatal("pointer outside independent rendered target", control.name)
		}
		previous, err := a.HeadlessGameOptionsFrame()
		if err != nil {
			t.Fatal(err)
		}
		checkGem(previous, control.gem, control.off)
		for _, edge := range []string{"press", "release"} {
			if err := a.HeadlessPointer(edge, control.point.X, control.point.Y); err != nil {
				t.Fatal(err)
			}
		}
		current, err := a.HeadlessGameOptionsFrame()
		if err != nil {
			t.Fatal(err)
		}
		checkGem(current, control.gem, control.on)
		changed := 0
		for y := control.gem.Min.Y; y < control.gem.Max.Y; y++ {
			for x := control.gem.Min.X; x < control.gem.Max.X; x++ {
				if previous.RGBAAt(x, y) != current.RGBAAt(x, y) {
					changed++
				}
			}
		}
		if changed == 0 || f.live.world.Hash() != before || f.live.world.Tick() != beforeTick || f.gameOptionValues(true) != stored || len(f.live.pending) != 0 {
			t.Fatal("installed pointer missed rendered control or changed live state before OK", control.name, changed)
		}
		t.Logf("%s: independent point %v target %v gem ROI %v changed %d pixels", control.name, control.point, control.target, control.gem, changed)
	}
	if f.gameOptionValues(true) != stored || len(f.live.pending) != 0 {
		t.Fatal("a click reached the stored options before OK", f.gameOptionValues(true), f.live.pending)
	}
	if err := a.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	if f.live.world.Hash() != before || f.live.world.Tick() != beforeTick || f.gameOptionValues(true) != (ui.GameOptionValues{1, 1, 1, 2, 2}) {
		t.Fatal("installed pointer controls changed wrong setting or stepped world", f.gameOptionValues(true))
	}
	if err := a.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	pic := a.GameMenuPanel()
	if pic == nil {
		t.Fatal("installed options panel absent")
	}
	for _, frame := range []int{0, 1, 4, 5} {
		p, err := loadTipGemFrame(f.Archives.Containers, tipGemPath, frame)
		if err != nil || p.Bounds().Empty() {
			t.Fatal("original toggle sprite", frame, err)
		}
	}
	if dir := os.Getenv("AGAINROM_OPTIONS_ARTIFACTS"); dir != "" {
		dir = filepath.Join(dir, filepath.Base(os.Getenv("AGAINROM_ASSETS")))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		out, err := os.Create(filepath.Join(dir, "game-options.png"))
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(out, pic)
		closeErr := out.Close()
		if err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
	}
	t.Log("five persisted controls; ordinary queued-choice SAV; opposite-profile cold LOAD and 64 next-tick option samples without replay; legacy AGS successors; SAV flags and retreat thresholds; installed pointer controls and panel")
}

func optionsTwin(t *testing.T, f *FrontEnd, live sim.Entity) sim.Entity {
	t.Helper()
	back, ok := f.live.world.Entity(live.ID)
	if !ok || back.Owner != live.Owner || back.Humanoid != live.Humanoid || back.SourceBinding.RuntimeID != live.SourceBinding.RuntimeID {
		t.Fatalf("actor %d disappeared or changed identity on reload: %v", live.ID, ok)
	}
	return back
}
