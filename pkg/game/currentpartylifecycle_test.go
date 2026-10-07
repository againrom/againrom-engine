package game

import (
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentSourceActorStaleStageRestoresAndResaves(t *testing.T) {
	f := currentManifestFront(t, false)
	var source sim.Entity
	for _, e := range f.live.world.Entities() {
		if e.SourceBinding.Class != 0 && e.Alive() {
			source = e
			break
		}
	}
	if source.SourceBinding.Identity == 0 || source.Decay != sim.DecayNone {
		t.Fatal("fixture has no living source actor", source)
	}
	wantHash := f.live.world.Hash()
	raw := currentRuntimeSave(t, f)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	sourceRecord := func(doc *sav.DocumentData) *sav.DocumentRecordData {
		t.Helper()
		for i := range doc.Objects {
			record := &doc.Objects[i]
			if record.Class != sourceActorDocumentClass(source.SourceBinding.Class) {
				continue
			}
			identity, err := savedStructureValue(record, "Identity")
			if err != nil || identity != source.SourceBinding.Identity {
				continue
			}
			health, err := savedStructureValue(record, "Health")
			if err != nil || int32(int16(health)) != source.HP {
				t.Fatalf("ordinary Health=%d, want %d: %v", health, source.HP, err)
			}
			return record
		}
		t.Fatal("source actor has no ordinary SAV record")
		return nil
	}
	if err := savedActorSetValue(sourceRecord(&doc), "Stage", 1); err != nil {
		t.Fatal(err)
	}
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	badDoc, err := sav.DecodeDocumentData(currentRuntimeSave(t, f))
	if err != nil {
		t.Fatal(err)
	}
	if err := savedActorSetValue(sourceRecord(&badDoc), "Stage", 2); err != nil {
		t.Fatal(err)
	}
	bad, err := sav.EncodeDocumentData(badDoc)
	if err != nil {
		t.Fatal(err)
	}
	badFront := actorRegistryFront(t)
	if _, _, err := badFront.RestoreOriginal(bad); err == nil || !strings.Contains(err.Error(), "decay stage") {
		t.Fatalf("positive-HP terminal Stage bypassed lifecycle validation: %v", err)
	}
	for cycle := 0; cycle < 2; cycle++ {
		cold := actorRegistryFront(t)
		open, town, err := cold.RestoreOriginal(raw)
		if err != nil || town {
			t.Fatalf("LOAD cycle %d: town=%t error=%v", cycle, town, err)
		}
		if err := cold.App("current source actor").OpenMission(open); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, e := range cold.live.world.Entities() {
			if e.SourceBinding.Identity == source.SourceBinding.Identity {
				found = e.ID == source.ID && e.HP == source.HP && e.Decay == sim.DecayNone
			}
		}
		if !found || cold.live.world.Hash() != wantHash {
			t.Fatalf("LOAD cycle %d changed actor identity or current World", cycle)
		}
		raw = currentRuntimeSave(t, cold)
		written, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		stage, err := savedStructureValue(sourceRecord(&written), "Stage")
		if err != nil || stage != 0 {
			t.Fatalf("SAVE cycle %d kept ordinary Stage=%d: %v", cycle, stage, err)
		}
	}
}
