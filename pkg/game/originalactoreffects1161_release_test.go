package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestReleaseActorEffectsRestored1161(t *testing.T) {
	_, raw := groundCorpusFile(t, "2026-08-02/game0007.sav", "a7cb35ea5d87c9c7a9b8cfd70f8a1f61b6927ca089e468a9d6fe983da777156e")
	f := releaseFront(t)
	ms, _, err := loadOriginalMission(f, raw)
	if err != nil {
		t.Fatal(err)
	}
	var subject sim.Entity
	found := false
	for _, e := range ms.World.Entities() {
		if e.SourceBinding.ArchiveIndex == 34 {
			subject, found = e, true
		}
	}
	if !found {
		t.Fatal("raw actor34 has no live carrier")
	}
	active := ms.World.ActiveEffects()
	if len(active) != 1 || active[0].Target != subject.ID || active[0].Spell != 0 || active[0].Kind != sim.EffectHealthRegeneration || active[0].Mode != sim.EffectDuration || active[0].Magnitude != 100 || active[0].Remaining != 776 {
		t.Fatalf("raw actor34 Effect36 kind8/mode1/id0/magnitude100/remaining776 lost at original LOAD: %v", active)
	}
	if subject.HealthRegeneration != 100 || binary.LittleEndian.Uint16(subject.ActorLoad.Source.Modifier[10:]) != 100 {
		t.Fatal("LOAD reapplied the already-saved regeneration modifier", subject.HealthRegeneration)
	}
	// A private, explicitly controlled derivative of this lawful input shortens
	// only the retained counter from 776 to 4. No actor stat or reference changes.
	// The unchanged original above proves its real value; the derivative makes
	// ordinary App SAVE / fresh LOAD / expiry observable before battle intervenes.
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	locs, err := source.DocumentObjectLocations()
	if err != nil {
		t.Fatal(err)
	}
	changed := 0
	for _, loc := range locs {
		if loc.ArchiveIndex != 36 {
			continue
		}
		if loc.Class != "Effect" || loc.Off < 0 || loc.Off > len(source.Body)-44 || binary.LittleEndian.Uint32(source.Body[loc.Off+39:]) != 50856036 {
			t.Fatal("independent Effect36 operand location differs")
		}
		binary.LittleEndian.PutUint16(source.Body[loc.Off+41:], 4)
		changed++
	}
	if changed != 1 {
		t.Fatal("controlled counter population", changed)
	}
	actorEffects1161App(t, source.Marshal(), subject.ID)
}

func actorEffects1161Current(t *testing.T, f *FrontEnd, id sim.EntityID, remaining uint16) Snapshot {
	t.Helper()
	active := f.live.world.ActiveEffects()
	if remaining == 0 {
		if len(active) != 0 {
			t.Fatal("expired attachment remains", active)
		}
	} else if len(active) != 1 || active[0].Target != id || active[0].Remaining != remaining || active[0].Magnitude != 100 {
		t.Fatal("current timer differs", remaining, active)
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	state := s.SavedDocument
	if state == nil || state.ActorEffects == nil || state.ActorEffects.Unavailable != "" || len(state.ActorEffects.Rows) != 1 {
		t.Fatal("current effect projection unavailable", state)
	}
	var actor *sav.DocumentRecordData
	for _, binding := range state.Actors {
		if binding.EntityID == id {
			actor = &state.Document.Objects[binding.ObjectIndex-1]
		}
	}
	if actor == nil {
		t.Fatal("current actor binding absent")
	}
	refs, ok := savedObjectRefs(actor, "Effects")
	mask, err := savedStructureValue(actor, "U144")
	if !ok || err != nil {
		t.Fatal("actor effect roots missing", err)
	}
	row := state.ActorEffects.Rows[0]
	entity, ok := f.live.entity(id)
	if !ok {
		t.Fatal("actor carrier disappeared")
	}
	if remaining == 0 {
		if len(refs) != 0 || row.ObjectIndex != 0 || mask&1 != 0 || entity.HealthRegeneration != 0 || binary.LittleEndian.Uint16(entity.ActorLoad.Source.Modifier[10:]) != 0 {
			t.Fatal("expiry lost owner edge/mask/modifier retirement", refs, row, mask, entity.HealthRegeneration)
		}
	} else {
		if len(refs) != 1 || refs[0] != row.ObjectIndex || row.Entity != id || row.Spell != 0 || mask&1 == 0 || entity.HealthRegeneration != 100 {
			t.Fatal("current effect owner/mask/modifier differs", refs, row, mask)
		}
		// Independent current World-to-Document assertion: do not call the
		// production originalActorEffect mapper or trust equal native hashes.
		object := &state.Document.Objects[row.ObjectIndex-1]
		for name, want := range map[string]uint32{"E3C": 8, "E3D": 1, "E0C": 0, "E40": 100 | uint32(remaining)<<16} {
			got, err := savedStructureValue(object, name)
			if err != nil || got != want {
				t.Fatalf("current Effect %s=%d want%d: %v", name, got, want, err)
			}
		}
	}
	return s
}

func actorEffects1161App(t *testing.T, raw []byte, id sim.EntityID) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("Actor effect continuation")
	app.Layout(1024, 768)
	path := filepath.Join(t.TempDir(), "effect.sav")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "effect.sav")
	actorEffects1161Current(t, f, id, 4)
	mover1160Menu(t, app)
	groundAppLoad(t, app, list, "effect.sav")
	actorEffects1161Current(t, f, id, 4)
	f.live.tick()
	actorEffects1161Current(t, f, id, 3)
	mover1160Menu(t, app)
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := listAGS(store)
	if err != nil || len(entries) != 1 {
		t.Fatal(entries, err)
	}
	if err := app.HeadlessGameMenuAction("return"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	fresh := releaseFront(t)
	fresh.SetDeterministicFrames(true)
	app2 := fresh.App("Source free actor effect LOAD")
	app2.Layout(1024, 768)
	save, list, load = nativeContinuationSeams1170(t, fresh, store, OriginalStore{}, nil)
	app2.SetSaveSeams(save, list, load)
	groundAppLoad(t, app2, list, entries[0].Name)
	actorEffects1161Current(t, fresh, id, 3)
	for step := uint16(0); step < 4; step++ {
		if f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatal("source-free pair differs", step)
		}
		remaining := uint16(0)
		if step < 3 {
			remaining = 3 - step
		}
		actorEffects1161Current(t, f, id, remaining)
		actorEffects1161Current(t, fresh, id, remaining)
		if step < 3 {
			f.live.tick()
			fresh.live.tick()
		}
	}
	// Native SAVE after retirement must also remain source-free and must not
	// resurrect the historical attachment from a retained Document child.
	mover1160Menu(t, app2)
	if err := app2.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err = listAGS(store)
	if err != nil {
		t.Fatal(err)
	}
	groundAppLoad(t, app2, list, entries[0].Name)
	actorEffects1161Current(t, fresh, id, 0)
	t.Log("actor effects: unchanged SAV id0/magnitude100/remaining776; controlled counter4; title and map LOAD; changed menu SAVE at3; fresh source-free LOAD; 3 paired ticks; expiry modifier/mask/owner edge and child retirement; post-expiry native LOAD")
}
