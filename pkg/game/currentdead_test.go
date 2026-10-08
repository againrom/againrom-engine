package game

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentBoundCorpseFixture(t *testing.T, runtime uint32, health int16, references ...uint32) *FrontEnd {
	t.Helper()
	body := &poolFixtureActor{mapID: 92, cell: 0x0807, hp: uint16(health), maxHP: 30, stage: 2, human: true, runtime: &runtime}
	living := &poolFixtureActor{mapID: 91, cell: 0x1211, hp: 23, maxHP: 30, name: "Living binding"}
	f := currentRetainedRuntimeFront(t)
	raw := savedContainer(poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{living}}}, {}}, []*poolFixtureActor{body}))
	raw = completeCurrentDeadDocument(t, f, raw)
	if len(references) != 0 {
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		mustSetValue(&doc.Objects[doc.DeadActors[0]-1], "U40", references[0])
		raw, err = sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
	}
	return openCurrentRetainedRuntime(t, f, raw)
}

func completeCurrentDeadDocument(t *testing.T, f *FrontEnd, raw []byte) []byte {
	t.Helper()
	return completeDocumentTail1115(t, f, raw)
}

func TestCurrentBoundCorpseReferenceUsesOrdinaryFieldAcrossCycles(t *testing.T) {
	const original = uint32(0xabcddcba)
	for _, change := range []struct {
		name string
		key  uint32
	}{{"unchanged", original}, {"ordinary changed", 0x10203040}, {"ordinary zero", 0}} {
		t.Run(change.name, func(t *testing.T) {
			f := currentBoundCorpseFixture(t, 0xabcdef77, -14, original)
			want := currentBoundCorpseFixture(t, 0xabcdef77, -14, change.key).live.world
			if f.live.world.OriginalDeadActors()[0].Source.References[4] != original {
				t.Fatal("fixture lacks its ordinary retained reference")
			}
			raw := currentRuntimeSave(t, f)
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			body := &doc.Objects[doc.DeadActors[0]-1]
			if key, _ := savedStructureValue(body, "U40"); key != original {
				t.Fatalf("retained reference lost its ordinary owner: %x -> %x", original, key)
			}
			policy, _, _ := sav.NativeActions(doc.State)
			mustSetValue(body, "U40", change.key)
			after, _, _ := sav.NativeActions(doc.State)
			if !bytes.Equal(policy, after) {
				t.Fatal("ordinary reference edit changed native policy")
			}
			raw, err = sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			for cycle := 0; cycle < 2; cycle++ {
				cold := openCurrentRetainedRuntime(t, currentRetainedRuntimeFront(t), raw)
				assertCurrentWorldEqual(t, want, cold.live.world, "retained ordinary reference LOAD")
				if cold.live.world.OriginalDeadActors()[0].Source.References[4] != change.key {
					t.Fatal("ordinary retained reference changed", cycle)
				}
				for tick := 0; tick < 64; tick++ {
					sim.Step(want, nil)
					sim.Step(cold.live.world, nil)
					assertCurrentWorldEqual(t, want, cold.live.world, "retained ordinary reference next tick")
				}
				raw = currentRuntimeSave(t, cold)
			}
		})
	}
}

func TestCurrentBoundCorpseReferenceAbsencePromotesOrdinaryEdits(t *testing.T) {
	seed := currentBoundCorpseFixture(t, 0xabcdef77, -14)
	body := seed.live.world.OriginalDeadActors()[0]
	// The reference names the fixture's source-bound living actor: its SAV
	// identity is World state, while a native actor's key is minted per SAVE.
	var living sim.Entity
	for _, e := range seed.live.world.Entities() {
		if e.Alive() && e.SourceBinding.Identity != 0 {
			living = e
		}
	}
	document, err := sav.DecodeDocumentData(currentRuntimeSave(t, seed))
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&document)
	if err != nil {
		t.Fatal(err)
	}
	var livingKey uint32
	for _, binding := range actions.Bindings {
		if binding.ID == living.ID && !binding.Missing && !binding.Structure {
			livingKey, _ = savedStructureValue(&document.Objects[binding.Object-1], "Identity")
		}
	}
	if livingKey == 0 {
		t.Fatal("fixture lacks an exact living reference target")
	}
	for _, change := range []struct {
		name string
		key  uint32
	}{{"unchanged", body.Source.Identity}, {"ordinary bound", livingKey}, {"ordinary zero", 0}} {
		t.Run(change.name, func(t *testing.T) {
			f := currentBoundCorpseFixture(t, 0xabcdef77, -14, body.Source.Identity)
			want := currentBoundCorpseFixture(t, 0xabcdef77, -14, change.key).live.world
			for _, e := range f.live.world.Entities() {
				if e.ID == body.ID && e.HasKillCredit {
					t.Fatal("fixture resolved its retained raw reference before continuation")
				}
			}
			if change.name == "ordinary bound" {
				if err := want.ImportOriginalActorActions([]sim.OriginalActorAction{{Entity: body.ID, HasCredit: true, Credit: living.ID}}); err != nil {
					t.Fatal(err)
				}
			}
			doc, err := sav.DecodeDocumentData(currentRuntimeSave(t, f))
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || a.Values[body.ID].RetainedCreditAbsent == nil || *a.Values[body.ID].RetainedCreditAbsent != body.Source.Identity {
				t.Fatal("missing exact retained reference absence", err)
			}
			policy, _, _ := sav.NativeActions(doc.State)
			mustSetValue(&doc.Objects[doc.DeadActors[0]-1], "U40", change.key)
			after, _, _ := sav.NativeActions(doc.State)
			if !bytes.Equal(policy, after) {
				t.Fatal("ordinary U40 edit changed native absence")
			}
			raw, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			for cycle := 0; cycle < 2; cycle++ {
				cold := openCurrentRetainedRuntime(t, currentRetainedRuntimeFront(t), raw)
				assertCurrentWorldEqual(t, want, cold.live.world, "retained reference resolution LOAD")
				for tick := 0; tick < 64; tick++ {
					sim.Step(want, nil)
					sim.Step(cold.live.world, nil)
					assertCurrentWorldEqual(t, want, cold.live.world, "retained reference resolution next tick")
				}
				raw = currentRuntimeSave(t, cold)
			}
		})
	}
}

func TestCurrentBoundCorpseKeepsAbsentBasisAndTwoCycles(t *testing.T) {
	for _, ticks := range []int{0, 33} {
		t.Run(map[int]string{0: "unchanged", 33: "advanced"}[ticks], func(t *testing.T) {
			f := currentBoundCorpseFixture(t, 0xabcdef77, -14)
			for range ticks {
				f.live.tick()
			}
			want := f.live.world
			for cycle := 0; cycle < 2; cycle++ {
				raw := currentRuntimeSave(t, f)
				doc, err := sav.DecodeDocumentData(raw)
				if err != nil {
					t.Fatal(err)
				}
				a, err := readCurrentActions(&doc)
				if err != nil {
					t.Fatal(err)
				}
				body := want.OriginalDeadActors()[0]
				v := a.Values[body.ID]
				if !v.RetainedDead || v.SourceBound || v.LoadPresent || v.ClassAbsent == nil || v.ClassAbsent.Wire != 2 || v.ClassAbsent.Humanoid || v.RetainedOwnerAbsent == nil {
					t.Fatal("fixture did not exercise explicit retained class/owner absence", v)
				}
				cold := openCurrentRetainedRuntime(t, currentRetainedRuntimeFront(t), raw)
				if cold.live.world.Hash() != want.Hash() || !reflect.DeepEqual(cold.live.world.OriginalDeadActors(), want.OriginalDeadActors()) {
					t.Fatalf("cycle%d full World changed: %x -> %x", cycle, want.Hash(), cold.live.world.Hash())
				}
				var left, right sim.World
				before, _ := want.MarshalBinary()
				after, _ := cold.live.world.MarshalBinary()
				if err := left.UnmarshalBinary(before); err != nil {
					t.Fatal(err)
				}
				if err := right.UnmarshalBinary(after); err != nil {
					t.Fatal(err)
				}
				for tick := 0; tick < 64; tick++ {
					sim.Step(&left, nil)
					sim.Step(&right, nil)
					if left.Hash() != right.Hash() {
						t.Fatalf("cycle%d successor%d diverged", cycle, tick)
					}
				}
				if left.OriginalDeadActors()[0].Current.HP >= body.Current.HP {
					t.Fatal("death clock was not reached")
				}
				f = cold
			}
		})
	}
}

func TestCurrentBoundCorpseOrdinaryRuntimeAndHealthEditsWin(t *testing.T) {
	f := currentBoundCorpseFixture(t, 0xabcdef77, -14)
	want := currentBoundCorpseFixture(t, 60123, -17).live.world
	doc, err := sav.DecodeDocumentData(currentRuntimeSave(t, f))
	if err != nil {
		t.Fatal(err)
	}
	policy, _, _ := sav.NativeActions(doc.State)
	body := &doc.Objects[doc.DeadActors[0]-1]
	savedObjectSetValue(body, "RuntimeID", 60123)
	savedObjectSetValue(body, "Health", uint32(uint16(65536-17)))
	after, _, _ := sav.NativeActions(doc.State)
	if !bytes.Equal(policy, after) {
		t.Fatal("ordinary edit changed native policy")
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	for cycle := 0; cycle < 2; cycle++ {
		cold := openCurrentRetainedRuntime(t, currentRetainedRuntimeFront(t), raw)
		if cold.live.world.Hash() != want.Hash() {
			t.Fatalf("cycle%d ordinary edit lost: %x -> %x", cycle, want.Hash(), cold.live.world.Hash())
		}
		raw = currentRuntimeSave(t, cold)
	}
}

func TestCurrentBoundCorpseMalformedPresenceIsAtomic(t *testing.T) {
	f := currentBoundCorpseFixture(t, 0xabcdef77, -14)
	raw := currentRuntimeSave(t, f)
	id := f.live.world.OriginalDeadActors()[0].ID
	before, _ := f.live.world.MarshalBinary()
	live, town, units := f.live, f.Town, f.Units
	for _, change := range []string{"class zero", "class wide", "class redundant", "source bound", "manager absent", "owner zero", "credit zero", "credit without manager", "body root absent", "wrong identity", "unknown class field"} {
		t.Run(change, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil {
				t.Fatal(err)
			}
			v := a.Values[id]
			switch change {
			case "class zero":
				v.ClassAbsent.Wire = 0
			case "class wide":
				v.ClassAbsent.Wire = 4
			case "class redundant":
				v.ClassAbsent.Humanoid = true
			case "source bound":
				v.SourceBound = true
			case "manager absent":
				v.RetainedDead = false
			case "owner zero":
				*v.RetainedOwnerAbsent = 0
			case "credit zero":
				v.RetainedCreditAbsent = new(uint32)
			case "credit without manager":
				key := uint32(0x12345678)
				v.RetainedCreditAbsent = &key
				v.RetainedDead = false
			case "body root absent":
				doc.DeadActors = nil
			case "wrong identity":
				a.Values[id+100] = v
			}
			a.Values[id] = v
			leaf, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			if change == "unknown class field" {
				leaf = bytes.Replace(leaf, []byte(`"ClassAbsent":{`), []byte(`"ClassAbsent":{"Extra":1,`), 1)
			}
			if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
				t.Fatal(err)
			}
			candidate, err := sav.EncodeDocumentData(doc)
			if err == nil {
				_, _, err = f.RestoreOriginal(candidate)
			}
			after, marshalErr := f.live.world.MarshalBinary()
			if err == nil || marshalErr != nil || !bytes.Equal(before, after) || f.live != live || f.Town != town || f.Units != units {
				t.Fatal("malformed retained presence published state", err, marshalErr)
			}
		})
	}
}

func TestCurrentBoundCorpseOrdinaryClassAndOwnerEditsWin(t *testing.T) {
	f := currentBoundCorpseFixture(t, 0xabcdef77, -14)
	doc, err := sav.DecodeDocumentData(currentRuntimeSave(t, f))
	if err != nil {
		t.Fatal(err)
	}
	id := f.live.world.OriginalDeadActors()[0].ID
	before, _, _ := sav.NativeActions(doc.State)
	body := &doc.Objects[doc.DeadActors[0]-1]
	var owner, slot uint32
	for _, index := range doc.Players {
		p := &doc.Objects[index-1]
		candidate, _ := savedStructureValue(p, "Slot")
		if candidate != 1 {
			owner, _ = savedStructureValue(p, "This")
			slot = candidate
			break
		}
	}
	if owner == 0 {
		t.Fatal("fixture lacks a second explicit Player")
	}
	savedObjectSetValue(body, "Reference", owner)
	var flags []string
	for _, name := range []string{"HasInventory", "HasSpellbook"} {
		if value, _ := savedStructureValue(body, name); value != 0 {
			flags = append(flags, name)
		}
	}
	changed := mustNewRecord("Humanoid", flags...)
	for i := range changed.Values {
		for _, v := range body.Values {
			if v.Name == changed.Values[i].Name {
				changed.Values[i] = v
			}
		}
	}
	for i := range changed.Raw {
		for _, v := range body.Raw {
			if v.Name == changed.Raw[i].Name {
				changed.Raw[i] = v
			}
		}
	}
	for i := range changed.Texts {
		for _, v := range body.Texts {
			if v.Name == changed.Texts[i].Name {
				changed.Texts[i] = v
			}
		}
	}
	for i := range changed.Counts {
		for _, v := range body.Counts {
			if v.Name == changed.Counts[i].Name {
				changed.Counts[i] = v
			}
		}
	}
	for i := range changed.RefSlots {
		for _, v := range body.RefSlots {
			if v.Name == changed.RefSlots[i].Name {
				changed.RefSlots[i] = v
			}
		}
	}
	*body = changed
	after, _, _ := sav.NativeActions(doc.State)
	if !bytes.Equal(before, after) {
		t.Fatal("ordinary class/owner edit changed policy")
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	// The edited Humanoid joins a fixture placement that names no Human row,
	// so the World holds no definition a further SAVE could write; the LOAD
	// alone carries the edit.
	{
		cold := openCurrentRetainedRuntime(t, currentRetainedRuntimeFront(t), raw)
		var e sim.Entity
		for _, entity := range cold.live.world.Entities() {
			if entity.ID == id {
				e = entity
			}
		}
		dead := cold.live.world.OriginalDeadActors()[0]
		if !e.Humanoid || e.Owner != slot || e.SourceBinding != (sim.SourceBinding{}) || e.ActorLoad != (sim.ActorLoad{}) || dead.Source.Class != 3 || dead.Source.OwnerKey != owner {
			t.Fatal("ordinary class/owner edit lost", e, dead)
		}
	}
}
