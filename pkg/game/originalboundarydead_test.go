package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestOriginalAttackCycleBoundaryOwesNothing(t *testing.T) {
	for _, c := range []struct {
		phase, complete, action uint32
		progress                byte
		countdown               uint32
		want                    sim.AttackPhase
		count                   int32
		ok                      bool
	}{
		{0, 1, 0, 0, 1, sim.AttackBoundaryOne, 0, true},
		{0, 1, 3, 1, 9, sim.AttackBoundaryOne, 0, true},
		{0, 0, 0, 0, 4, sim.AttackBoundaryTwo, 0, true},
		{0, 0, 3, 1, 4, sim.AttackReady, 4, true},
		{5, 0, 3, 1, 7, sim.AttackCharging, 7, true},
		{7, 0, 3, 1, 6, sim.AttackRelaxing, 6, true},
		{9, 0, 3, 1, 6, sim.AttackReady, 6, false},
	} {
		phase, count, ok := originalAttackCycle(c.phase, c.complete, c.action, c.progress, c.countdown)
		if phase != c.want || count != c.count || ok != c.ok {
			t.Errorf("sub-phase %d completion %d action %d progress %d countdown %d: got %d/%d/%v, want %d/%d/%v",
				c.phase, c.complete, c.action, c.progress, c.countdown, phase, count, ok, c.want, c.count, c.ok)
		}
	}
}

// deadEffectDocument is a document with one dead Human root (object 1) that
// owns one Poison Effect (object 2) and nothing else.
func deadEffectDocument(t *testing.T) (sav.DocumentData, sim.SavedEffectObject) {
	t.Helper()
	effect := sim.SavedEffectObject{E0C: 8, Value: sim.ItemEffect{Kind: 6, Mode: 2, Operand: 0x0080fff8}, Token: sim.SavedObjectToken{
		Position: [12]byte{4: 128, 5: 128, 6: 18, 7: 5, 8: 32, 9: 112, 10: 112, 11: 18}, RuntimeID: 34734080, T0C: 8, T0E: 24, T08: 4292870150, Identity: 50814192}}
	body := mustNewRecord("Human")
	mustSetValue(&body, "Identity", 0x0f6a3530)
	mustSetValue(&body, "U144", 1<<8)
	savedObjectSetRefs(&body, "Effects", []uint16{2}, true)
	doc := sav.DocumentData{Objects: []sav.DocumentRecordData{body, savedCurrentEffectRecord(effect)}, DeadActors: []uint16{1}}
	return doc, effect
}

func TestRetainedDeadEffectsAreReEmitted(t *testing.T) {
	doc, effect := deadEffectDocument(t)
	got, ok := retainedDeadEffects(&doc, 1)
	if !ok || len(got) != 1 || got[0] != effect {
		t.Fatalf("dead root Effects read %+v %v", got, ok)
	}
	b := &generatedDocumentBuilder{loaded: &doc}
	if err := b.reserveRetainedDeadEffectKeys(&sim.World{}); err != nil {
		t.Fatal(err)
	}
	record := mustNewRecord("Human")
	if err := b.retainDeadEffects(&record, 5, sim.OriginalDeadSource{Identity: 0x0f6a3530, Class: 2}); err != nil {
		t.Fatal(err)
	}
	refs, _ := savedObjectRefs(&record, "Effects")
	mask, _ := savedStructureValue(&record, "U144")
	if len(refs) != 1 || mask != 1<<8 {
		t.Fatalf("written dead record Effects %v mask %#x", refs, mask)
	}
	written, err := savedEffectRecord(&b.doc.Objects[refs[0]-1])
	if err != nil || written != effect {
		t.Fatalf("written Effect %+v %v, want the loaded one with its Token", written, err)
	}
	other := mustNewRecord("Human")
	if err := b.retainDeadEffects(&other, 6, sim.OriginalDeadSource{Identity: 0x0f6a3531, Class: 2}); err != nil {
		t.Fatal(err)
	}
	if refs, _ := savedObjectRefs(&other, "Effects"); len(refs) != 0 {
		t.Fatal("a record the loaded document does not hold took Effects", refs)
	}
	// A body the World holds attachments on keeps them as the World's own.
	b.liveEffects = map[sim.EntityID]bool{5: true}
	live := mustNewRecord("Human")
	if err := b.retainDeadEffects(&live, 5, sim.OriginalDeadSource{Identity: 0x0f6a3530, Class: 2}); err != nil {
		t.Fatal(err)
	}
	if refs, _ := savedObjectRefs(&live, "Effects"); len(refs) != 0 {
		t.Fatal("a body with live attachments took retained Effects", refs)
	}
}

func TestRetainedDeadEffectsRefuseOtherShapes(t *testing.T) {
	for name, alter := range map[string]func(*sav.DocumentData){
		"root with a second owner": func(doc *sav.DocumentData) {
			owner := mustNewRecord("Human")
			savedObjectSetRefs(&owner, "Effects", []uint16{1}, true)
			doc.Objects = append(doc.Objects, owner)
		},
		"shared Effect": func(doc *sav.DocumentData) {
			other := mustNewRecord("Human")
			savedObjectSetRefs(&other, "Effects", []uint16{2}, true)
			doc.Objects = append(doc.Objects, other)
		},
		"repeated spell": func(doc *sav.DocumentData) {
			doc.Objects = append(doc.Objects, doc.Objects[1])
			savedObjectSetRefs(&doc.Objects[0], "Effects", []uint16{2, 3}, true)
		},
		"not a dead root": func(doc *sav.DocumentData) { doc.DeadActors = nil },
	} {
		doc, _ := deadEffectDocument(t)
		alter(&doc)
		if got, ok := retainedDeadEffects(&doc, 1); ok {
			t.Errorf("%s: retained %+v", name, got)
		}
	}
}
