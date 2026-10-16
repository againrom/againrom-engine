package game

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/sim"
)

type phaseSoundReceipt struct {
	Tick   uint64
	Record sim.SavedProjectile
}

func TestReleaseProjectilePhaseSoundUsesTheInstalledAdapter(t *testing.T) {
	w := newDeliveryWitness(t, t.TempDir(), true)
	f, app := w.front, w.app
	app.Layout(1024, 768)
	party := MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
	w.check(app.OpenMission(f.MissionOpenerWith(41, party)))
	for n := 0; app.HeadlessNoticeOpen() && n < 32; n++ {
		w.check(app.HeadlessKey("enter"))
	}
	releasePauseMission(t, f, app)
	id := f.live.mission.ids[0]
	flightLearn(t, f.live, id, 21)
	f.live.push()
	at, ok := cityMageNearest(cityMageCastCells(t, f, app, id, 21))
	if !ok {
		t.Fatal("no visible cell admits the installed Earthquake")
	}
	cityMageCastInput(t, f, app, id, 21, at)
	for _, edge := range []string{"right-press", "right-release"} {
		w.check(app.HeadlessPointer(edge, 512, 200))
	}
	activePauseResume(t, app)
	var cues []phaseSoundReceipt
	adapter := f.live.semanticSound
	if adapter == nil {
		t.Fatal("mission did not wire Viewer.PlayEffectAtFine")
	}
	f.live.semanticSound = func(source string, slot int, owner uint32, cell, fine image.Point) {
		if source == "storm-phase" {
			if slot != 551 || fine != cell.Mul(256).Add(image.Pt(128, 128)) {
				t.Fatalf("cell phase request has wrong slot or fine point: %d %v %v", slot, cell, fine)
			}
			found := false
			for _, p := range f.live.world.SavedProjectiles().Items {
				if p.Picture == 51 && p.ActionPhase == 8 && p.Phase == 8 && image.Pt(int(p.X/256), int(p.Y/256)) == cell {
					cues = append(cues, phaseSoundReceipt{Tick: f.live.world.Tick(), Record: p})
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("installed phase request at %v has no live actionphase-8 record", cell)
			}
		}
		adapter(source, slot, owner, cell, fine)
	}
	for range 128 {
		w.check(app.HeadlessStep())
		w.finishOneShots()
	}
	if len(cues) == 0 {
		t.Fatal("ordinary installed cast produced no phase cue")
	}
	seen := map[uint16]bool{}
	for _, cue := range cues {
		if seen[cue.Record.ID] {
			t.Fatalf("record %d repeated its phase callback", cue.Record.ID)
		}
		seen[cue.Record.ID] = true
	}
	admitted := 0
	for _, r := range w.owner.Service.Snapshot().Receipts {
		if r.Source != "storm-phase" || r.Reason != audio.DeliveryAdmitted {
			continue
		}
		if r.Selector != "registry:551" || !r.SpatialKnown || r.Scope == 0 || r.Sample == 0 || r.Repeat {
			t.Fatalf("installed adapter request %+v", r)
		}
		admitted++
	}
	if admitted == 0 {
		t.Fatal("installed phase cue never reached an admitted backend request")
	}
	if out := os.Getenv("AGAINROM_PROJECTILE_PHASE_WITNESS_DIR"); out != "" {
		if !filepath.IsAbs(out) {
			t.Fatal("phase witness directory must be absolute")
		}
		w.check(os.MkdirAll(out, 0700))
		raw, err := json.MarshalIndent(struct {
			Cues       []phaseSoundReceipt
			Admissions int
			Delivery   audio.DeliverySnapshot
		}{cues, admitted, w.owner.Service.Snapshot()}, "", "  ")
		w.check(err)
		w.check(os.WriteFile(filepath.Join(out, "phase-sound.json"), raw, 0600))
	}
	t.Logf("installed App cast: %d distinct phase-8 record cues, %d admitted backend requests; deterministic device players", len(cues), admitted)
}
