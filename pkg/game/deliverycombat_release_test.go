package game

import (
	"encoding/json"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func (w *deliveryWitness) installedCombatRoutes() {
	w.t.Helper()
	f, a := w.front, w.app
	previousDraw, previousHold := newViewerVoiceDraw, w.holdOneShots
	newViewerVoiceDraw = func() voiceDraw { return func() int { return 0 } }
	w.holdOneShots = false
	defer func() { newViewerVoiceDraw, w.holdOneShots = previousDraw, previousHold }()
	groups := map[string]audio.Channel{
		"unit-selection": audio.SpeechChannel, "unit-command": audio.SpeechChannel,
		"unit-swing": audio.EffectsChannel, "unit-strike": audio.EffectsChannel,
		"unit-hurt": audio.SpeechChannel, "unit-death": audio.SpeechChannel,
	}
	proof := struct {
		Mission       int
		Qualification []string
		Hero, Victim  sim.Entity
		PlacedVictim  sim.Entity
		Attack        sim.Command
		StartTick     uint64
		EndTick       uint64
		Frames        int
		HealthWrites  []sim.EntityID
		Admissions    []audio.DeliveryReceipt
		Counts        map[string]int
	}{Mission: 20, Qualification: []string{
		"Installed actors, registries, voice banks, FrontEnd, App, Viewer, shared delivery and retained backend are production objects.",
		"The lowest device player has deterministic playback status; completed one-shots are released after each input frame. This does not prove audible sound.",
		"Presentation voice draws are zero. HeadlessPlace brings a genuine hostile beside the hero; presentation fog is exposed for the pointer route.",
		"Explicit script health controls preserve the hero during the bounded fight and force a fall only if combat has not already supplied one.",
	}, Counts: map[string]int{}}
	defer func() {
		raw, err := json.MarshalIndent(proof, "", "  ")
		w.check(err)
		w.check(os.WriteFile(filepath.Join(w.dir, "installed-combat-controls.json"), raw, 0600))
		w.receipt("installed-combat")
	}()
	var sequence uint64
	for _, r := range w.owner.Service.Snapshot().Receipts {
		sequence = max(sequence, r.Sequence)
	}
	collect := func() {
		w.t.Helper()
		for _, r := range w.owner.Service.Snapshot().Receipts {
			if r.Sequence <= sequence {
				continue
			}
			sequence = r.Sequence
			group, wanted := groups[r.Source]
			if !wanted || r.Reason != audio.DeliveryAdmitted {
				continue
			}
			if r.Group != group || r.Recipe != r.Source || r.VolumeTerm != "distance+setting" || !r.SpatialKnown || r.Repeat || r.Frequency != 0 || r.Scope == 0 || r.Sample == 0 {
				w.t.Fatalf("installed combat request metadata: %+v", r)
			}
			cam := f.live.view.Camera()
			geometry := audio.ViewGeometry{Origin: image.Pt(int(math.Floor(cam.X/32)), int(math.Floor(cam.Y/32))),
				Span: image.Pt(int(float64(cam.ViewW)/cam.Zoom/32), int(float64(cam.ViewH)/cam.Zoom/32))}
			if geometry.Span.X <= 0 || geometry.Span.Y <= 0 || r.Geometry != geometry {
				w.t.Fatalf("installed combat geometry: got %+v, production camera %+v", r.Geometry, geometry)
			}
			dx := float64(r.SourceFine.X - r.Geometry.Origin.X*256 - r.Geometry.Span.X*128)
			dy := float64(r.SourceFine.Y - r.Geometry.Origin.Y*256 - r.Geometry.Span.Y*128)
			d := int(math.Max(-10000, -(math.Exp(math.Hypot(dx, dy)/2048)-1)*100))
			pan := int(math.Max(-10000, math.Min(10000, math.Trunc(dx*4000/float64(r.Geometry.Span.X*256)))))
			if r.Attenuation != d || r.Pan != pan || r.Priority != uint8((10000-int(math.Abs(float64(d))))/100) {
				w.t.Fatalf("installed combat spatial terms: %+v; independent D=%d pan=%d", r, d, pan)
			}
			if !strings.HasPrefix(r.Selector, "registry:") && !strings.HasPrefix(r.Selector, "voice:") {
				w.t.Fatalf("installed combat selector: %+v", r)
			}
			proof.Admissions = append(proof.Admissions, r)
			proof.Counts[r.Source]++
		}
		w.finishOneShots()
	}
	w.check(a.OpenMission(f.NewGameOpener(20, ui.ChargenResult{Name: "Delivery fighter", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 20, 20}})))
	a.Layout(1024, 768)
	for n := 0; a.HeadlessNoticeOpen() && n < 32; n++ {
		w.check(a.HeadlessKey("enter"))
		collect()
	}
	if a.HeadlessNoticeOpen() {
		w.t.Fatal("instrument: installed mission notice did not close")
	}
	releasePauseMission(w.t, f, a)
	if len(f.live.mission.ids) == 0 {
		w.t.Fatal("instrument: installed mission has no hero")
	}
	hero, ok := f.live.entity(f.live.mission.ids[0])
	if !ok || !hero.Alive() {
		w.t.Fatal("instrument: installed mission hero is absent")
	}
	victim, ok := hurtVoiceNearestHostile(f, hero)
	if !ok || victim.OffMap {
		w.t.Fatal("instrument: installed mission has no on-map hostile")
	}
	proof.Hero, proof.Victim = hero, victim
	w.check(f.live.world.HeadlessPlace(victim.ID, hero.X+1, hero.Y))
	proof.PlacedVictim, _ = f.live.entity(victim.ID)
	for i := range f.live.fog.visible {
		f.live.fog.visible[i], f.live.fog.explored[i] = 1, 1
	}
	f.live.push()
	inspectionCentre(f.live, int(hero.X), int(hero.Y))
	w.check(a.HeadlessPointer("hover", 512, 200))
	collect()
	w.check(a.HeadlessSelectEntity(uint32(hero.ID)))
	collect()
	if proof.Counts["unit-selection"] == 0 {
		w.t.Fatal("installed App selection did not admit its bank reply")
	}
	for n := 0; n < 160; n++ {
		w.check(a.HeadlessStep())
		collect()
	}
	if f.live.view.AttackArmed() {
		w.t.Fatal("instrument: attack mode armed before actual input")
	}
	w.check(a.HeadlessKey("a"))
	collect()
	x, y, err := a.HeadlessEntityPoint(uint32(victim.ID))
	w.check(err)
	n := len(f.live.pending)
	w.check(a.HeadlessPointer("press", x, y))
	collect()
	w.check(a.HeadlessPointer("release", x, y))
	collect()
	if len(f.live.pending) != n+1 || f.live.pending[n].Kind != sim.KindAttack || f.live.pending[n].Entity != hero.ID || f.live.pending[n].X != int32(victim.ID) {
		w.t.Fatalf("installed App attack input queued %+v", f.live.pending[n:])
	}
	proof.Attack, proof.StartTick = f.live.pending[n], f.live.world.Tick()
	if proof.Counts["unit-command"] == 0 {
		w.t.Fatal("installed App attack did not admit its bank command reply")
	}
	activePauseResume(w.t, a)
	collect()
	for ; proof.Frames < 600; proof.Frames++ {
		if proof.Counts["unit-swing"] > 0 && proof.Counts["unit-strike"] > 0 && proof.Counts["unit-hurt"] > 0 {
			break
		}
		if e, ok := f.live.entity(hero.ID); ok && e.HP > 0 && e.HP < e.MaxHP/2 {
			w.check(f.live.world.HeadlessHeal(hero.ID))
			proof.HealthWrites = append(proof.HealthWrites, hero.ID)
		}
		if a.HeadlessNoticeOpen() {
			w.check(a.HeadlessKey("enter"))
		} else {
			w.check(a.HeadlessStep())
		}
		collect()
	}
	if proof.Counts["unit-death"] == 0 {
		if e, ok := f.live.entity(victim.ID); ok && e.HP > 0 {
			w.check(f.live.world.HeadlessDamage(victim.ID, e.HP+1))
			proof.HealthWrites = append(proof.HealthWrites, victim.ID)
			f.live.push()
			w.check(a.HeadlessStep())
			collect()
		}
	}
	proof.EndTick = f.live.world.Tick()
	for source := range groups {
		if proof.Counts[source] == 0 {
			w.t.Fatalf("installed combat has no admitted %s after %d bounded frames: %v", source, proof.Frames, proof.Counts)
		}
	}
	if proof.EndTick <= proof.StartTick {
		w.t.Fatal("installed combat did not advance production World")
	}
	w.t.Logf("mission20 attack=%+v tick=%d..%d frames=%d admissions=%v health controls=%v", proof.Attack, proof.StartTick, proof.EndTick, proof.Frames, proof.Counts, proof.HealthWrites)
}

func TestReleaseSharedAudioCombat(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("set AGAINROM_ASSETS to a lawful install")
	}
	if !filepath.IsAbs(root) || !filepath.IsAbs(os.Getenv("AGAINROM_ROOT")) || !strings.EqualFold(filepath.Clean(root), filepath.Clean(os.Getenv("AGAINROM_ROOT"))) {
		t.Fatal("instrument: AGAINROM_ASSETS and AGAINROM_ROOT must name the same absolute install")
	}
	out := os.Getenv("AGAINROM_SFX_DELIVERY_WITNESS_DIR")
	if !filepath.IsAbs(out) {
		t.Fatal("instrument: AGAINROM_SFX_DELIVERY_WITNESS_DIR must name an absolute existing directory")
	}
	out, err := editorPhysicalDirectory(out)
	if err != nil {
		t.Fatal("instrument:", err)
	}
	install, err := editorPhysicalDirectory(root)
	if err != nil {
		t.Fatal("instrument:", err)
	}
	rel, err := filepath.Rel(install, out)
	if err != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatal("instrument: witness output is inside the preserved install")
	}
	dir := filepath.Join(out, filepath.Base(install), "combat")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	w := newDeliveryWitness(t, dir, true)
	w.installedCombatRoutes()
	w.app.StopAudio()
	w.app.StopAudio()
	state := w.owner.BackendState()
	if !w.owner.Service.Snapshot().Closed || len(state.Buffers) != 0 || state.Created != state.Destroyed {
		t.Fatal("combat shutdown leaked retained buffers", state)
	}
	for i, p := range w.players {
		if p.closes != 1 || p.playing {
			t.Fatal(fmt.Sprintf("combat lower player %d: closes=%d playing=%v", i+1, p.closes, p.playing))
		}
	}
	w.receipt("combat-shutdown")
}
