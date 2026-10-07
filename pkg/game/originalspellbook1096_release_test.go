package game

import (
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// The literal offsets come from pinned research/tools/savreplay's first
// Player programme, not Party(): Human@249, book Spell bodies@1119+11*i.
// The original source stores every ID 1..28. Its Humans template knows four.
// Both installs read the same owner source; these are not two recordings.
func TestReleaseOriginalSpellbook1096SourceMembershipAppAndNative(t *testing.T) {
	f := releaseFront(t)
	path, payload := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 28; i++ {
		off := 1119 + 11*i
		if source.Body[off] != byte(i+1) {
			t.Fatalf("source Spell+08 at %d = %d, want%d", off, source.Body[off], i+1)
		}
	}
	const known = uint32(0x1ffffffe)
	const oldTemplate = uint32(0x00041042)
	if got := uint32(f.Table.Humans.EntryParams(28)[25]); got != oldTemplate {
		t.Fatalf("control template changed: %#x", got)
	}
	f.SetDeterministicFrames(true)
	app := f.App("1096-source-spellbook")
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, filepath.Base(path))
	if len(f.liveParty) != 1 || f.liveParty[0].Name != "Fergard" || f.liveParty[0].KnownSpells != known {
		t.Fatalf("loaded party = %+v", f.liveParty)
	}
	id := f.live.mission.ids[0]
	e := releaseEntity(t, f.live, id)
	rows := spellbookOf(sim.Rules{}, e, f.live.world.Spells(), f.live.spellNames, f.live.view.Words(), nil)
	if e.KnownSpells != known || len(rows) != 28 {
		t.Fatalf("book mask=%#x rows=%d, want28", e.KnownSpells, len(rows))
	}
	// Use a restored non-template spell through the production command seam.
	// Protection from Air (ID16) is absent from the control template mask.
	if reason := f.live.world.BookSpellRefusal(id, id, 16); reason != "" {
		t.Fatalf("source cast16 refused: %s", reason)
	}
	f.live.attackOrCast(uint32(id), uint32(id), 16, 0, 0, false)
	if len(f.live.pending) != 1 || f.live.pending[0].Kind != sim.KindCast {
		t.Fatal("spell selection did not enqueue CAST")
	}
	events := sim.StepObserved(f.live.world, f.live.pending)
	f.live.pending = nil
	for i := 0; len(events) == 0 && i < 255; i++ {
		events = sim.StepObserved(f.live.world, nil)
	}
	cast := false
	for _, event := range events {
		if event.Caster == id && event.Spell == 16 {
			cast = true
		}
	}
	if !cast {
		t.Fatalf("restored non-template spell did not cast; actor=%+v events=%+v", e, events)
	}
	wantEffects := areaAttachments(f.live.world)
	found := false
	for _, effect := range wantEffects {
		found = found || effect.Spell == 16 && effect.Remaining > 0
	}
	if !found {
		t.Fatal("cast did not leave a timed Protection from Air attachment")
	}
	name, err := save(true)
	if err != nil || filepath.Ext(name) != ".sav" {
		t.Fatalf("current save=%q err=%v", name, err)
	}
	groundAppLoad(t, app, list, localOriginalSaveToken(name))
	id = f.live.mission.ids[0]
	if releaseEntity(t, f.live, id).KnownSpells != known {
		t.Fatal("SAV continuation lost saved spell membership")
	}
	if got := areaAttachments(f.live.world); !reflect.DeepEqual(got, wantEffects) {
		t.Fatalf("SAV attachment=%+v want %+v", got, wantEffects)
	}
	t.Logf("source=%s mask=%08x template=%08x rows=28 non-template cast=16 SAV attachments=%+v", path, known, oldTemplate, wantEffects)
}
