package game

import (
	"encoding/binary"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Synthetic actor, modifier and Effect are constructed independently. This
// fixture supplies a full graph for native validation, not ROM1 evidence.
func actorEffectFixture(t *testing.T) (*FrontEnd, *Mission) {
	t.Helper()
	f := unit1158FixtureFront(t)
	doc, err := sav.DecodeDocumentData(unit1158AppFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	effect := literalSavedEffectRecord(0xabcdef)
	savedObjectSetValue(&effect, "E0C", 0)
	savedObjectSetValue(&effect, "E3C", 8)
	savedObjectSetValue(&effect, "E3D", 1)
	savedObjectSetValue(&effect, "E40", 100|2<<16)
	doc.Objects = append(doc.Objects, effect)
	for i := range doc.Objects {
		actor := &doc.Objects[i]
		if actor.Class != "Human" {
			continue
		}
		savedObjectSetRefs(actor, "Effects", []uint16{uint16(len(doc.Objects))}, true)
		savedObjectSetValue(actor, "U144", 1)
		for j := range actor.Raw {
			if actor.Raw[j].Name == "UD4" {
				binary.LittleEndian.PutUint16(actor.Raw[j].Bytes[10:], 100)
			}
		}
		break
	}
	doc, _, err = sav.ReindexDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, f.Bodies)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms.World.ActiveEffects()) != 1 || ms.savedDocument.ActorEffects == nil || ms.savedDocument.ActorEffects.Unavailable != "" {
		t.Fatal("fixture effect not admitted")
	}
	return f, ms
}

func TestActorEffects1161NativeDocumentLossControls(t *testing.T) {
	_, ms := actorEffectFixture(t)
	good := ms.savedDocument
	if err := validateSavedActorEffectsWorld(good, ms.World); err != nil {
		t.Fatal("positive native baseline", err)
	}
	for _, name := range []string{"dropped binding", "wrong owner", "wrong ID", "missing edge", "wrong magnitude", "stale remaining", "mask cleared", "retired live child"} {
		t.Run(name, func(t *testing.T) {
			bad, err := cloneSavedDocument(good)
			if err != nil {
				t.Fatal(err)
			}
			row := bad.ActorEffects.Rows[0]
			var actor *sav.DocumentRecordData
			for _, a := range bad.Actors {
				if a.EntityID == row.Entity {
					actor = &bad.Document.Objects[a.ObjectIndex-1]
				}
			}
			switch name {
			case "dropped binding":
				bad.ActorEffects.Rows = nil
			case "wrong owner":
				bad.ActorEffects.Rows[0].Entity = 999
			case "wrong ID":
				bad.ActorEffects.Rows[0].Spell = 1
			case "missing edge":
				savedObjectSetRefs(actor, "Effects", nil, true)
			case "wrong magnitude":
				savedObjectSetValue(&bad.Document.Objects[row.ObjectIndex-1], "E40", 101|2<<16)
			case "stale remaining":
				savedObjectSetValue(&bad.Document.Objects[row.ObjectIndex-1], "E40", 100|3<<16)
			case "mask cleared":
				savedObjectSetValue(actor, "U144", 0)
			case "retired live child":
				bad.ActorEffects.Rows[0].ObjectIndex = 0
			}
			_, cloneErr := cloneSavedDocument(bad)
			if cloneErr == nil && validateSavedActorEffectsWorld(bad, ms.World) == nil {
				t.Fatal("accepted native actor-effect loss")
			}
		})
	}
	// Same native World hash is not proof of a current Document. A stale
	// counter must fail the native envelope validator independently.
	encoded, err := ms.World.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	bad, _ := cloneSavedDocument(good)
	row := bad.ActorEffects.Rows[0]
	savedObjectSetValue(&bad.Document.Objects[row.ObjectIndex-1], "E40", 100|3<<16)
	if _, err := savedDocumentFromSnapshot(Snapshot{Mission: int(good.Document.Head.Mission), Difficulty: mapload.Difficulty(good.Document.Head.Difficulty), World: encoded, SavedDocument: bad}); err == nil {
		t.Fatal("same World hash concealed stale Document counter")
	}
}

func TestActorEffects1161LegacyNativeDoesNotRestartAndNewAttachmentIsProjected(t *testing.T) {
	_, ms := actorEffectFixture(t)
	legacy, _ := cloneSavedDocument(ms.savedDocument)
	legacy.ActorEffects = nil
	// Recreate a predecessor's already-expired native world while retaining
	// its historical Document. This route MUST NOT call the original importer.
	sim.Step(ms.World, nil)
	sim.Step(ms.World, nil)
	if len(ms.World.ActiveEffects()) != 0 {
		t.Fatal("fixture timer did not expire")
	}
	ms.savedDocument = legacy
	current, err := snapshotSavedDocument(ms)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms.World.ActiveEffects()) != 0 || current.ActorEffects == nil || current.ActorEffects.Unavailable == "" || len(current.ActorEffects.Rows) != 0 {
		t.Fatal("legacy native restarted timers or concealed projection gap")
	}
	if _, err := cloneSavedDocument(current); err != nil {
		t.Fatal("explicit legacy metadata did not round trip", err)
	}
	// Exercise the native Document restore door itself with a historical
	// effect subtree and no attachment metadata, not only Snapshot handling.
	current.ActorEffects = nil
	encoded, err := ms.World.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var freshWorld sim.World
	if err := freshWorld.UnmarshalBinary(encoded); err != nil {
		t.Fatal(err)
	}
	legacyMission := &Mission{World: &freshWorld}
	if err := restoreSavedDocument(legacyMission, current); err != nil {
		t.Fatal(err)
	}
	if len(freshWorld.ActiveEffects()) != 0 || legacyMission.savedDocument.ActorEffects != nil {
		t.Fatal("native restore restarted historic timers")
	}
	_, ms = actorEffectFixture(t)
	sim.Step(ms.World, nil)
	sim.Step(ms.World, nil)
	state, err := snapshotSavedDocument(ms)
	if err != nil {
		t.Fatal(err)
	}
	id := state.ActorEffects.Rows[0].Entity
	if err := ms.World.ImportOriginalAttachedEffects([]sim.ActiveEffect{{Target: id, Spell: 18, Kind: sim.EffectAbsorption, Mode: sim.EffectDuration, Magnitude: 3, Remaining: 12}}); err != nil {
		t.Fatal(err)
	}
	if err := projectSavedActorEffects(state, ms.World); err != nil {
		t.Fatal(err)
	}
	if state.ActorEffects.Unavailable != "" || len(state.ActorEffects.Rows) != 2 || state.ActorEffects.Rows[0].Spell != 0 || state.ActorEffects.Rows[0].ObjectIndex != 0 || state.ActorEffects.Rows[1].ObjectIndex == 0 {
		t.Fatalf("current attachment was not projected: %+v", state.ActorEffects)
	}
	row := state.ActorEffects.Rows[1]
	value, err := savedEffectRecord(&state.Document.Objects[row.ObjectIndex-1])
	if err != nil || row.Entity != id || row.Spell != 18 || value.E0C != 18 || value.Value.Kind != 16 || value.Value.Mode != 1 || value.Value.Operand != 3|12<<16 {
		t.Fatalf("new attachment raw fields/binding = %+v / %+v: %v", row, value, err)
	}
	if err := validateSavedActorEffectsWorld(state, ms.World); err != nil {
		t.Fatal("new attachment differs from current World", err)
	}
}

func TestActorEffects1161RetirementReindexesAndPreservesIdentity(t *testing.T) {
	_, ms := actorEffectFixture(t)
	before, _ := cloneSavedDocument(ms.savedDocument)
	removed := before.ActorEffects.Rows[0].ObjectIndex
	sim.Step(ms.World, nil)
	sim.Step(ms.World, nil)
	after, err := snapshotSavedDocument(ms)
	if err != nil {
		t.Fatal(err)
	}
	if after.ActorEffects.Unavailable != "" || after.ActorEffects.Rows[0].ObjectIndex != 0 || len(after.Document.Objects) != len(before.Document.Objects)-1 {
		t.Fatal("exclusive child not retired exactly")
	}
	for _, binding := range after.Actors {
		var prior SnapshotSAVActor
		for _, b := range before.Actors {
			if b.EntityID == binding.EntityID {
				prior = b
			}
		}
		old := before.Document.Objects[prior.ObjectIndex-1]
		next := after.Document.Objects[binding.ObjectIndex-1]
		for _, name := range []string{"Identity", "RuntimeID"} {
			a, _ := savedStructureValue(&old, name)
			b, _ := savedStructureValue(&next, name)
			if a != b {
				t.Fatal("actor key changed during retirement", name)
			}
		}
	}
	for _, binding := range after.ActorEffects.Rows {
		if binding.ObjectIndex == removed && removed != 0 {
			t.Fatal("retired archive index reused as current identity")
		}
	}
	// An explicit remap may move bindings but must never silently erase one.
	bad, _ := cloneSavedDocument(before)
	permutation := make([]uint16, len(before.Document.Objects)+1)
	for i := range permutation {
		permutation[i] = uint16(i)
	}
	permutation[removed] = 0
	if err := remapSavedActorEffects(bad, permutation); err == nil {
		t.Fatal("implicit effect retirement accepted")
	}
	bad, _ = cloneSavedDocument(before)
	bad.ActorEffects.Rows = append(bad.ActorEffects.Rows, slices.Clone(bad.ActorEffects.Rows)...)
	if _, err := cloneSavedDocument(bad); err == nil {
		t.Fatal("aliased native bindings accepted")
	}
}
