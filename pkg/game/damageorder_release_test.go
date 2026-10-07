package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseOrderedHurtMessagesKeepWorldAndSAV(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("ordered hurt")
	app.SetCutscenes(nil)
	party := f.ChargenParty(ui.ChargenResult{Name: "Hurt", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 20, 20}})
	var messages []sim.DamageEvent
	var ticks int
	opener := f.MissionOpenerWith(20, party)
	if err := app.OpenMission(func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
		v, _, order, cadence, affect, advance, attack, grab, stance, march, err := opener()
		tick := func() {
			f.live.frameAdvance(func() {
				if !f.live.stopped {
					f.live.tickStep(nil, true, func(report sim.Report) { messages = append(messages, report.Damages...); ticks++ })
				}
			})
		}
		return v, tick, order, cadence, affect, advance, attack, grab, stance, march, err
	}); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	for n := 0; app.HeadlessNoticeOpen() && n < 32; n++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if app.HeadlessNoticeOpen() {
		t.Fatal("mission notice did not close")
	}
	releasePauseMission(t, f, app)
	id := f.live.mission.ids[0]
	if err := f.live.world.HeadlessHeal(id); err != nil {
		t.Fatal(err)
	}
	for i := range f.live.fog.visible {
		f.live.fog.visible[i], f.live.fog.explored[i] = 1, 1
	}
	f.live.push()
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	hero := releaseEntity(t, f.live, id)
	if hero.HP <= 20 {
		t.Fatalf("fixture hero has only %d HP", hero.HP)
	}
	heard := observeReleaseAudio(t, f)
	before, err := f.live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	plain := *f.live.world
	if err := plain.UnmarshalBinary(before); err != nil {
		t.Fatal(err)
	}
	commands := []sim.Command{sim.Damage(id, 7), sim.Damage(id, 13)}
	sim.Step(&plain, commands)
	messages, ticks = nil, 0
	f.live.pending = append(f.live.pending, commands...)
	f.live.stopped = false
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	f.live.stopped = true
	want := []sim.DamageEvent{{Target: id, BeforeHP: hero.HP, AfterHP: hero.HP - 7}, {Target: id, BeforeHP: hero.HP - 7, AfterHP: hero.HP - 20}}
	if ticks != 1 || !slices.Equal(messages, want) {
		t.Fatalf("App route tick=%d messages=%+v want %+v", ticks, messages, want)
	}
	after, err := f.live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	plainBytes, err := plain.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, plainBytes) || f.live.world.Hash() != plain.Hash() {
		t.Fatal("observed App tick changed World bytes/hash")
	}
	var sources []string
	for i, play := range heard.plays {
		if releaseAudioFromActor(t, heard, i, id) {
			sources = append(sources, play.src)
		}
	}
	if !slices.Equal(sources, []string{"mf_hero/easy.wav"}) {
		t.Fatalf("actual installed voice requests=%v", sources)
	}
	saved := fpsCurrentSAV(t, f)
	plays := len(heard.plays)
	f.live.push()
	f.live.push()
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if len(heard.plays) != plays || !bytes.Equal(saved, fpsCurrentSAV(t, f)) || f.live.world.Hash() != plain.Hash() {
		t.Fatal("repeated snapshots replayed messages or changed World/SAV")
	}
	proof := struct {
		Qualification          string
		Install                string
		Messages               []sim.DamageEvent
		Requests               []string
		WorldSHA256, SAVSHA256 [32]byte
	}{"Installed mission20 actors, App, simulation, projection, Viewer, archive voice and shared audio service; explicit engine damage controls7/13; observed/plain World equality and repeated-snapshot SAV equality. No native audibility claim.", f.Archives.Root, messages, sources, sha256.Sum256(after), sha256.Sum256(saved)}
	if out := os.Getenv("AGAINROM_ORDERED_HURT_WITNESS_DIR"); out != "" {
		dir, err := effectRimOutputPath(f.Archives.Root, out)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		raw, err := json.MarshalIndent(proof, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "ordered-hurt.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("App→sim→game→Viewer messages=%+v requests=%v World=%x SAV=%x", messages, sources, proof.WorldSHA256, proof.SAVSHA256)
}
