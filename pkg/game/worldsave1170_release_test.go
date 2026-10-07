package game

import (
	"bytes"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func world1170Entity(t *testing.T, f *FrontEnd, runtime uint32) sim.Entity {
	return worldEntityByRuntimeID(t, f, runtime)
}

func worldEntityByRuntimeID(t *testing.T, f *FrontEnd, runtime uint32) sim.Entity {
	t.Helper()
	for _, e := range f.live.world.Entities() {
		if e.SourceBinding.RuntimeID == runtime {
			return e
		}
	}
	t.Fatalf("runtime actor %d disappeared", runtime)
	return sim.Entity{}
}

func world1170Position(w *sim.World, e sim.Entity) [2]int32 {
	if x, y, ok := w.ActorFinePosition(e.ID); ok {
		return [2]int32{e.X*256 + int32(x), e.Y*256 + int32(y)}
	}
	if e.Stride.Present && e.Transit > 0 {
		n := int32(e.TransitTotal - e.Transit)
		return [2]int32{e.Stride.FromX*256 + 128 + int32(e.Stride.StepX)*n, e.Stride.FromY*256 + 128 + int32(e.Stride.StepY)*n}
	}
	return [2]int32{e.X*256 + 128, e.Y*256 + 128}
}

func loadSAVWindow(t *testing.T, store SaveStore, name string) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("cold current SAV")
	app.Layout(1024, 768)
	save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, localOriginalSaveToken(name))
	return f, app
}

func TestReleaseCurrentWorldSAV1170(t *testing.T) {
	t.Run("uninterrupted_route", func(t *testing.T) { world1170MotionContinuation(t, "crossing") })
	t.Run("current_native_turn", func(t *testing.T) { world1170MotionContinuation(t, "turn") })
	t.Run("pending_native_heal", func(t *testing.T) { world1170MotionContinuation(t, "heal") })
	f := releaseFront(t)
	_, raw := groundCorpusFile(t, "2026-08-15/game0016.sav", "5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345")
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal(err)
	}
	app := f.App("current world SAV")
	if err = app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	// Advance a complete script interval through the normal driver. Results
	// must come from the live register bank, including its saved clock.
	beforeResults := f.live.world.ScriptRegisters()
	for range 16 {
		f.live.tick()
	}
	if f.live.world.ScriptRegisters() == beforeResults {
		t.Fatal("ordinary driver did not change current Results")
	}
	var hero sim.Entity
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	party, err := source.Party()
	if err != nil {
		t.Fatal(err)
	}
	var heroKey uint32
	for _, member := range party {
		if member.Hero {
			heroKey = member.Key
		}
	}
	for _, e := range f.live.world.Entities() {
		if e.SourceBinding.Identity == heroKey && e.SourceBinding.Class != 0 {
			hero = e
			break
		}
	}
	if hero.SourceBinding.RuntimeID == 0 {
		t.Fatal("authentic hero missing")
	}
	pack, _ := f.live.world.CarriedStacks(hero.ID)
	if len(pack) == 0 {
		t.Fatal("authentic hero pack missing")
	}
	beforeSacks := len(f.live.world.Sacks())
	var damaged sim.Entity
	for _, e := range f.live.world.Entities() {
		if e.Alive() && e.HP > 3 && e.Owner != hero.Owner && e.SourceBinding.Class == 1 && e.SourceBinding.RuntimeID != 0 {
			damaged = e
			break
		}
	}
	if damaged.SourceBinding.RuntimeID == 0 {
		t.Fatal("source-backed nonparty damage subject missing")
	}
	f.live.pending = append(f.live.pending, sim.Command{Kind: sim.KindDropCarried, Entity: hero.ID, Spell: 0, X: hero.X + 1, Y: hero.Y})
	headlessDamage(t, f.live.world, damaged.ID, 3)
	f.live.tick()
	if len(f.live.world.Sacks()) != beforeSacks+1 {
		t.Fatal("ordinary drop did not create its real new root")
	}
	_, _, order, _, _, _, _, _, _, _, err := open()
	if err != nil {
		t.Fatal(err)
	}
	order(uint32(hero.ID), int(hero.X+3), int(hero.Y))
	for i := 0; i < 100; i++ {
		f.live.tick()
		hero = world1170Entity(t, f, hero.SourceBinding.RuntimeID)
		if hero.Stride.Present && hero.Transit > 1 && world1170NativeTurnsFinished(f.live.world) {
			break
		}
	}
	if !hero.Stride.Present || hero.Transit <= 1 || !world1170NativeTurnsFinished(f.live.world) {
		t.Fatalf("ordinary move did not reach a mid-crossing: %+v", hero)
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	out, err := f.ExportCurrentWorldSave(s, "1170 current world")
	if err != nil {
		t.Fatal(err)
	}
	world1170LossControls(t, s, raw, f.live.world, hero)
	file, err := sav.Open(out)
	if err != nil || file.World == nil {
		t.Fatal(err)
	}
	t.Logf("current world SAV: %d objects, %d bytes", len(s.SavedDocument.Document.Objects), len(out))
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{}, nil))
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	want := world1170Entity(t, f, hero.SourceBinding.RuntimeID)
	wantDamage := world1170Entity(t, f, damaged.SourceBinding.RuntimeID)
	if wantDamage.HP == damaged.HP {
		t.Fatal("ordinary damage was not present at SAVE")
	}
	t.Logf("changed SAV cut: tick=%d hero runtime=%d transit=%d/%d; damage runtime=%d HP=%d->%d action deadline=%+v; sacks=%d->%d", f.live.world.Tick(), want.SourceBinding.RuntimeID, want.Transit, want.TransitTotal, wantDamage.SourceBinding.RuntimeID, damaged.HP, wantDamage.HP, wantDamage.ActionClock, beforeSacks, len(f.live.world.Sacks()))
	position := world1170Position(f.live.world, want)
	wantPack, _ := f.live.world.CarriedItems(want.ID)
	wantSacks := f.live.world.Sacks()
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 || !strings.HasSuffix(entries[0].Name, ".sav") {
		t.Fatalf("ordinary SAVE requires SAV: %+v / %v", entries, err)
	}
	saved, err := store.Read(entries[0].Name)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(saved, []byte("Asg&")) {
		t.Fatal("ordinary output is not SAV")
	}
	clear(raw)
	clear(out)
	cold, coldApp := loadSAVWindow(t, store, entries[0].Name)
	got := world1170Entity(t, cold, want.SourceBinding.RuntimeID)
	if gotDamage := world1170Entity(t, cold, damaged.SourceBinding.RuntimeID); gotDamage.HP != wantDamage.HP {
		t.Fatal("SAV lost the current damage subject")
	}
	world1170Holdings(t, cold.live.world, got.ID, wantPack, wantSacks)
	if got.HP != want.HP || world1170Position(cold.live.world, got) != position || len(cold.live.world.Sacks()) != beforeSacks+1 {
		t.Fatalf("current world lost: HP %d/%d position %v/%v sacks %d/%d", got.HP, want.HP, world1170Position(cold.live.world, got), position, len(cold.live.world.Sacks()), beforeSacks+1)
	}
	for step := 1; step <= 160; step++ {
		f.live.tick()
		cold.live.tick()
		liveActor := world1170Entity(t, f, want.SourceBinding.RuntimeID)
		coldActor := world1170Entity(t, cold, want.SourceBinding.RuntimeID)
		if world1170Position(f.live.world, liveActor) != world1170Position(cold.live.world, coldActor) || liveActor.HP != coldActor.HP {
			t.Fatalf("current crossing continuation %d: live %v HP%d cold %v HP%d", step, world1170Position(f.live.world, liveActor), liveActor.HP, world1170Position(cold.live.world, coldActor), coldActor.HP)
		}
		liveDamage := world1170Entity(t, f, damaged.SourceBinding.RuntimeID)
		coldDamage := world1170Entity(t, cold, damaged.SourceBinding.RuntimeID)
		if liveDamage.HP != coldDamage.HP {
			t.Fatalf("current damage continuation %d: HP %d/%d", step, liveDamage.HP, coldDamage.HP)
		}
	}
	arrived := world1170Entity(t, f, want.SourceBinding.RuntimeID)
	if world1170Position(f.live.world, arrived) != [2]int32{want.TargetX*256 + 128, want.TargetY*256 + 128} {
		t.Fatal("changed-world route did not complete")
	}
	cold.live.pending = append(cold.live.pending, sim.Command{Kind: sim.KindMoveTo, Entity: got.ID, X: arrived.X + 1, Y: arrived.Y})
	f.live.pending = append(f.live.pending, sim.Command{Kind: sim.KindMoveTo, Entity: want.ID, X: arrived.X + 1, Y: arrived.Y})
	f.live.tick()
	cold.live.tick()
	nextWant := world1170Entity(t, f, want.SourceBinding.RuntimeID)
	nextGot := world1170Entity(t, cold, want.SourceBinding.RuntimeID)
	if nextGot.HP != nextWant.HP || world1170Position(cold.live.world, nextGot) != world1170Position(f.live.world, nextWant) {
		t.Fatalf("same next order diverged: %v/%v HP %d/%d", world1170Position(cold.live.world, nextGot), world1170Position(f.live.world, nextWant), nextGot.HP, nextWant.HP)
	}
	if err := coldApp.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := coldApp.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err = store.List()
	if err != nil || len(entries) != 2 {
		t.Fatalf("second SAV missing: %+v %v", entries, err)
	}
	if !strings.HasSuffix(entries[0].Name, ".sav") {
		t.Fatal("second ordinary SAVE fell back", entries[0])
	}
	again, _ := loadSAVWindow(t, store, entries[0].Name)
	last := world1170Entity(t, again, want.SourceBinding.RuntimeID)
	world1170Holdings(t, again.live.world, last.ID, wantPack, wantSacks)
	if last.HP != nextWant.HP || world1170Position(again.live.world, last) != world1170Position(f.live.world, nextWant) {
		t.Fatal("second SAV lost successor state")
	}
	lastDamage := world1170Entity(t, again, damaged.SourceBinding.RuntimeID)
	nextDamage := world1170Entity(t, f, damaged.SourceBinding.RuntimeID)
	if lastDamage.HP != nextDamage.HP || lastDamage.ActionClock != nextDamage.ActionClock {
		t.Fatal("second SAV lost the current damage subject or idle deadline")
	}
	t.Log("changed damage, full route, new root and next action survive 161 samples and second ordinary SAV")
}

func TestReleaseCurrentWorldDeadRandomRefusal1170(t *testing.T) {
	f := releaseFront(t)
	_, raw := groundCorpusFile(t, "2026-08-15/game0016.sav", "5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345")
	open, _, err := f.RestoreOriginal(raw)
	if err != nil {
		t.Fatal(err)
	}
	app := f.App("current corpse and random state")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	var victim sim.Entity
	for _, e := range f.live.world.Entities() {
		if e.MapUnitID != 0 && e.SourceBinding.Class == 1 && e.Alive() {
			victim = e
			break
		}
	}
	if victim.SourceBinding.Identity == 0 {
		t.Fatal("source-backed victim missing")
	}
	headlessDamage(t, f.live.world, victim.ID, victim.HP+20)
	for i := 0; i < 200; i++ {
		f.live.tick()
		victim = worldEntityByRuntimeID(t, f, victim.SourceBinding.RuntimeID)
		if victim.Decay >= 2 {
			break
		}
	}
	if victim.Decay < 2 {
		t.Fatal("death did not enter corpse list")
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	// DIV-1202 replaced the refusal this test used to require with an authored
	// exact carrier, /CurrentState/AgainromRng: an advanced native random
	// stream now has a lossless SAV leaf and ExportCurrentWorldSave no longer
	// refuses on its account.
	exported, err := f.ExportCurrentWorldSave(s, "current corpse")
	if err != nil {
		t.Fatal("advanced random stream must no longer refuse SAV export", err)
	}
	exportedDoc, err := sav.DecodeDocumentData(exported)
	if err != nil {
		t.Fatal(err)
	}
	if state, present, err := sav.NativeRandomState(exportedDoc.State); err != nil || !present || state != f.live.world.RandomState() {
		t.Fatalf("current corpse SAV lost the AgainromRng leaf: present=%v state=%#x want=%#x err=%v", present, state, f.live.world.RandomState(), err)
	}
	runtime := victim.SourceBinding.RuntimeID
	seam := lateCorpseSeam(f, runtime)
	store, name, written := menuSAVE(t, f, app, OriginalStore{})
	clear(raw)
	cold := loadLocalLegacySave(t, store, name)
	if corpse := worldEntityByRuntimeID(t, cold, runtime); corpse.Alive() || corpse.Decay != victim.Decay || corpse.HP != victim.HP || lateCorpseSeam(cold, runtime) != seam {
		t.Fatalf("new current corpse lost: %s, want %s", lateCorpseSeam(cold, runtime), seam)
	}
	if cold.live.world.RandomState() != f.live.world.RandomState() {
		t.Fatalf("cold SAV lost the current native random state: %#x != %#x", cold.live.world.RandomState(), f.live.world.RandomState())
	}
	for step := 0; step <= 160; step++ {
		if f.live.world.RandomState() != cold.live.world.RandomState() || lateCorpseSeam(f, runtime) != lateCorpseSeam(cold, runtime) {
			t.Fatalf("death/advanced-stream SAV continuation diverged at sample %d: %s | %s", step, lateCorpseSeam(f, runtime), lateCorpseSeam(cold, runtime))
		}
		if step != 160 {
			f.live.tick()
			cold.live.tick()
		}
	}
	lost := loadAlteredSAV(t, written, func(doc *sav.DocumentData) bool { return setSavedActorHealth(doc, runtime, int16(victim.HP)-7) })
	if lateCorpseSeam(lost, runtime) == seam {
		t.Fatal("loss control: a SAV with the corpse HP altered still matches")
	}
	beforeSecond := worldEntityByRuntimeID(t, cold, runtime)
	secondSave, _, _ := cold.SaveSeams(store, OriginalStore{}, nil)
	name, err = secondSave(true)
	if err != nil || !strings.HasSuffix(name, ".sav") {
		t.Fatal("advanced stream must survive second ordinary SAV", name, err)
	}
	again := loadLocalLegacySave(t, store, name)
	if corpse := worldEntityByRuntimeID(t, again, runtime); corpse.Alive() || corpse.Decay != beforeSecond.Decay || corpse.HP != beforeSecond.HP {
		t.Fatal("second death/advanced-stream SAV lost the current corpse")
	}
	if again.live.world.RandomState() != cold.live.world.RandomState() {
		t.Fatal("second death/advanced-stream SAV lost its current native random state")
	}
	// The converter fixture's "to sav" direction (ConvertSave)
	// takes an AGS-encoded native checkpoint, not an already-ordinary SAV
	// (that is DecodeSave's own input contract); build one from cold's present
	// state to exercise that direction with the same advanced stream.
	coldSnapshot, _, err := cold.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	agsBytes, err := EncodeSave(coldSnapshot, "advanced random ags")
	if err != nil {
		t.Fatal(err)
	}
	converted, _, err := cold.ConvertSave(agsBytes, "sav", nil)
	if err != nil {
		t.Fatal("converter must carry the advanced native random stream, not refuse it", err)
	}
	convertedDoc, err := sav.DecodeDocumentData(converted)
	if err != nil {
		t.Fatal(err)
	}
	if state, present, err := sav.NativeRandomState(convertedDoc.State); err != nil || !present || state != cold.live.world.RandomState() {
		t.Fatalf("converter did not carry the exact advanced native random stream: present=%v state=%#x want=%#x err=%v", present, state, cold.live.world.RandomState(), err)
	}
	t.Logf("current corpse and RNG %016x retained through 161 SAV samples, second cold SAVE and the ags->sav converter", cold.live.world.RandomState())
}

func TestReleaseCurrentWorldEffectSAV1170(t *testing.T) {
	f := releaseFront(t)
	_, raw := groundCorpusFile(t, "2027-09-07/game0125.sav", "3a055c8dcef6f053721e1c1478c82552c199b92fb178e9f034fe7500077b4efd")
	open, _, err := f.RestoreOriginal(raw)
	if err != nil {
		t.Fatal(err)
	}
	app := f.App("current Light SAV")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	for range 18 {
		f.live.tick()
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	want := f.live.world.ActiveEffects()
	if len(want) == 0 {
		t.Fatal("final zero pulse lost attachments")
	}
	out, err := f.ExportCurrentWorldSave(s, "zero Light pulse")
	if err != nil {
		t.Fatal(err)
	}
	aliveWire := bytes.Clone(out)
	store := SaveStore{Dir: t.TempDir()}
	name, err := store.WriteOriginal("", out)
	if err != nil {
		t.Fatal(err)
	}
	cold, _ := loadSAVWindow(t, store, name)
	if len(cold.live.world.ActiveEffects()) != len(want) || len(cold.live.world.SavedSpellEffects()) != 1 {
		t.Fatal("current effect graph lost")
	}
	cold.live.tick()
	if len(cold.live.world.SavedSpellEffects()) != 0 {
		t.Fatal("expired area resurrected")
	}
	for range 15 {
		cold.live.tick()
	}
	for _, e := range cold.live.world.ActiveEffects() {
		if e.Spell == 12 {
			t.Fatal("expired attachment resurrected", e)
		}
	}
	s, _, err = cold.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	out, err = cold.ExportCurrentWorldSave(s, "expired Light")
	if err != nil {
		t.Fatal(err)
	}
	world1170ExpiredRootControl(t, aliveWire, out)
	name, err = store.WriteOriginal("", out)
	if err != nil {
		t.Fatal(err)
	}
	cold, _ = loadSAVWindow(t, store, name)
	if len(cold.live.world.SavedSpellEffects()) != 0 {
		t.Fatal("expired area survived second SAV")
	}
	for _, e := range cold.live.world.ActiveEffects() {
		if e.Spell == 12 {
			t.Fatal("expired attachment survived second SAV", e)
		}
	}
}

func TestReleaseCurrentWorldProjectileContinuation(t *testing.T) { originalSpellSAVContinuation(t) }
