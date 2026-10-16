package game

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Original mission SAVs read from the environment, never committed.

// originalBoundaryActor is a living actor record whose order sits at
// sub-phase 0 with completion 1 and a nonzero countdown byte.
type originalBoundaryActor struct {
	Identity  uint32
	Countdown uint32
}

func originalBoundaryActors(t *testing.T, doc *sav.DocumentData) []originalBoundaryActor {
	t.Helper()
	var out []originalBoundaryActor
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Human" && r.Class != "Unit" || slices.Contains(doc.DeadActors, uint16(i+1)) {
			continue
		}
		phase, err := savedMotionRaw(r, "U58", 4)
		if err != nil {
			continue
		}
		complete, _ := savedStructureValue(r, "U136")
		countdown, _ := savedStructureValue(r, "U6C")
		target, _ := savedStructureValue(r, "U5C")
		stage, _ := savedStructureValue(r, "Stage")
		identity, _ := savedStructureValue(r, "Identity")
		if binary.LittleEndian.Uint32(phase) == 0 && complete != 0 && countdown != 0 && target != 0 && stage == 0 {
			out = append(out, originalBoundaryActor{Identity: identity, Countdown: countdown})
		}
	}
	return out
}

// originalDeadEffects maps each dead record that keeps Effects to them,
// Tokens included.
func originalDeadEffects(t *testing.T, doc *sav.DocumentData) map[uint32][]sim.SavedEffectObject {
	t.Helper()
	out := map[uint32][]sim.SavedEffectObject{}
	for _, root := range doc.DeadActors {
		r := &doc.Objects[root-1]
		refs, _ := savedObjectRefs(r, "Effects")
		if len(refs) == 0 {
			continue
		}
		identity, _ := savedStructureValue(r, "Identity")
		effects, ok := retainedDeadEffects(doc, root)
		if !ok {
			t.Fatalf("dead actor %#x holds Effects outside the retained shape", identity)
		}
		out[identity] = effects
	}
	return out
}

func originalEntityByIdentity(w *sim.World, identity uint32) (sim.Entity, bool) {
	for _, e := range w.Entities() {
		if e.SourceBinding.Identity == identity {
			return e, true
		}
	}
	return sim.Entity{}, false
}

func requireOriginalFileState(t *testing.T, raw []byte, w *sim.World) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := file.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	living := 0
	for _, a := range graph.Actors {
		if a.Dead() || a.CurrentOffMap {
			continue
		}
		living++
		e, ok := originalEntityByIdentity(w, a.Identity)
		if !ok || !e.Alive() || e.X != int32(a.Col()) || e.Y != int32(a.Row()) {
			t.Fatalf("living actor %#x at %d,%d is not the World's: %+v present %v", a.Identity, a.Col(), a.Row(), e.ID, ok)
		}
	}
	dead, err := file.DeadActors()
	if err != nil {
		t.Fatal(err)
	}
	held := map[uint32]sim.OriginalDeadRecord{}
	for _, d := range w.OriginalDeadActors() {
		held[d.Source.Identity] = d
	}
	for _, d := range dead {
		got, ok := held[d.Identity]
		if d.Stage == 1 {
			continue
		}
		if !ok || got.Source.State.Stage != d.Stage || got.Source.State.HP != d.HP {
			t.Fatalf("dead actor %#x stage %d HP %d is not the World's: %+v", d.Identity, d.Stage, d.HP, got.Source.State)
		}
	}
	t.Logf("file state: %d living actors at their cells, %d dead records", living, len(dead))
}

// originalColdLoad loads a written SAV into a fresh front end and returns its
// App, so the reloaded mission can be saved again.
func originalColdLoad(t *testing.T, store SaveStore, name string) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("cold SAV LOAD")
	app.Layout(1024, 768)
	save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, name)
	t.Cleanup(app.StopAudio)
	return f, app
}

func TestReleaseOriginalBoundaryAndDeadEffectsLoad(t *testing.T) {
	for _, env := range []string{"AGAINROM_ORIGINAL_BOUNDARY_SAV", "AGAINROM_ORIGINAL_DEAD_EFFECT_SAV", "AGAINROM_ORIGINAL_DEAD_EFFECT_RESAVE_SAV"} {
		t.Run(env, func(t *testing.T) {
			path := os.Getenv(env)
			if path == "" {
				t.Skipf("no %s: the original save is not committed", env)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			boundary, deadEffects := originalBoundaryActors(t, &doc), originalDeadEffects(t, &doc)
			if len(boundary) == 0 && len(deadEffects) == 0 {
				t.Fatal("the file holds neither a stale boundary countdown nor a dead actor's Effect")
			}
			f := releaseFront(t)
			app, source := openOriginalSAVApp(t, f, raw, "source.sav")
			t.Cleanup(app.StopAudio)
			w := f.live.world
			requireOriginalFileState(t, raw, w)
			var watched []sim.EntityID
			for _, b := range boundary {
				e, ok := originalEntityByIdentity(w, b.Identity)
				if !ok || !e.HasAttackTarget || e.AttackPhase != sim.AttackBoundaryOne || e.AttackCountdown != 0 {
					t.Fatalf("boundary actor %#x (countdown byte %d) loaded as %+v", b.Identity, b.Countdown, e)
				}
				watched = append(watched, e.ID)
				t.Logf("boundary actor %#x: entity %d, countdown byte %d, World phase %d count %d", b.Identity, e.ID, b.Countdown, e.AttackPhase, e.AttackCountdown)
			}
			for identity, effects := range deadEffects {
				t.Logf("dead actor %#x keeps %d Effect(s): %+v", identity, len(effects), effects)
			}
			left := map[sim.EntityID]bool{}
			for range 96 {
				f.live.tick()
				for _, id := range watched {
					if e, ok := f.live.entity(id); ok && e.AttackPhase != sim.AttackBoundaryOne && e.AttackPhase != sim.AttackBoundaryTwo {
						left[id] = true
					}
				}
			}
			for _, id := range watched {
				if e, ok := f.live.entity(id); ok && e.Alive() && !left[id] {
					t.Fatalf("boundary actor %d never left the boundary in 96 ticks", id)
				}
			}
			store, name, written := menuSAVE(t, f, app, OriginalStore{Dir: filepath.Dir(source)})
			writtenDoc, err := sav.DecodeDocumentData(written)
			if err != nil {
				t.Fatal(err)
			}
			kept := originalDeadEffects(t, &writtenDoc)
			for identity, effects := range deadEffects {
				if !slices.Equal(kept[identity], effects) {
					t.Fatalf("SAVE lost dead actor %#x Effects: %+v, want %+v", identity, kept[identity], effects)
				}
			}
			fresh, freshApp := originalColdLoad(t, store, name)
			if fresh.live.world.Hash() != f.live.world.Hash() {
				t.Fatal("cold LOAD of the SAVE differs from the saved World")
			}
			_, _, again := menuSAVE(t, fresh, freshApp, OriginalStore{})
			againDoc, err := sav.DecodeDocumentData(again)
			if err != nil {
				t.Fatal(err)
			}
			if kept := originalDeadEffects(t, &againDoc); !maps.EqualFunc(kept, deadEffects, slices.Equal) {
				t.Fatalf("a second SAVE changed the dead actors' Effects: %+v", kept)
			}
			for range 8 {
				f.live.tick()
				fresh.live.tick()
			}
			if fresh.live.world.Hash() != f.live.world.Hash() {
				t.Fatal("the reloaded World diverges on its next ticks")
			}
			t.Logf("loaded, 96 ticks run, SAVE and cold LOAD keep the World and %d dead actors' Effects", len(deadEffects))
		})
	}
}

// The control is an original mission SAV that loaded before this change; its
// no-change SAVE is the one the base engine wrote.
func TestReleaseOriginalControlResaveUnchanged(t *testing.T) {
	path := os.Getenv("AGAINROM_ORIGINAL_CONTROL_SAV")
	if path == "" {
		t.Skip("no AGAINROM_ORIGINAL_CONTROL_SAV: the original save is not committed")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	app, source := openOriginalSAVApp(t, f, raw, "control.sav")
	t.Cleanup(app.StopAudio)
	_, _, written := menuSAVE(t, f, app, OriginalStore{Dir: filepath.Dir(source)})
	sum := sha256.Sum256(written)
	t.Logf("control resave %d bytes sha256 %x", len(written), sum)
	if want := os.Getenv("AGAINROM_ORIGINAL_CONTROL_RESAVE_SHA256"); want != "" && want != fmt.Sprintf("%x", sum) {
		t.Fatalf("control resave sha256 %x, want %s", sum, want)
	}
}

// A native SAV whose dead body carries a live attachment, named by the
// current-action casters under an ID other than the body's import ID, loads
// with that attachment, runs, and keeps it through SAVE and cold LOAD.
func TestReleaseNativeDeadBodyAttachmentLoads(t *testing.T) {
	path := os.Getenv("AGAINROM_NATIVE_DEAD_ATTACHMENT_SAV")
	if path == "" {
		t.Skip("no AGAINROM_NATIVE_DEAD_ATTACHMENT_SAV: the owner save is not committed")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	live, err := liveEffectObjects(&doc)
	if err != nil {
		t.Fatal(err)
	}
	bodies := 0
	for _, root := range doc.DeadActors {
		if _, ok := retainedDeadEffects(&doc, root); ok && live[root] {
			bodies++
		}
	}
	if bodies == 0 {
		t.Fatal("the file holds no dead body with a live attachment")
	}
	f := releaseFront(t)
	app, source := openOriginalSAVApp(t, f, raw, "native.sav")
	t.Cleanup(app.StopAudio)
	onDead := 0
	for _, e := range f.live.world.ActiveEffects() {
		if body, ok := f.live.entity(e.Target); ok && !body.Alive() {
			onDead++
		}
	}
	if onDead == 0 {
		t.Fatalf("%d dead bodies with live attachments loaded with none", bodies)
	}
	for range 32 {
		f.live.tick()
	}
	store, name, _ := menuSAVE(t, f, app, OriginalStore{Dir: filepath.Dir(source)})
	fresh, _ := originalColdLoad(t, store, name)
	if fresh.live.world.Hash() != f.live.world.Hash() {
		t.Fatal("cold LOAD of the SAVE differs from the saved World")
	}
	for range 8 {
		f.live.tick()
		fresh.live.tick()
	}
	if fresh.live.world.Hash() != f.live.world.Hash() {
		t.Fatal("the reloaded World diverges on its next ticks")
	}
	t.Logf("%d dead bodies with live attachments, %d attachments on dead bodies at load; 32 ticks, SAVE and cold LOAD keep the World", bodies, onDead)
}
