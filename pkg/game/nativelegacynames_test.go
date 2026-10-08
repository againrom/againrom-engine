package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

type nativeNamePolicyTest struct {
	Present      bool
	Actors       []currentManifestActor
	NativeActors []currentManifestActor `json:",omitempty"`
}

func nativeNamePolicyRead(t *testing.T, doc *sav.DocumentData) *nativeNamePolicyTest {
	t.Helper()
	leaf, present, err := sav.NativeActions(doc.State)
	if err != nil || !present {
		t.Fatal("native name transport", present, err)
	}
	var input struct{ Manifest *nativeNamePolicyTest }
	if err := json.Unmarshal(leaf, &input); err != nil {
		t.Fatal(err)
	}
	return input.Manifest
}

func nativeNamePolicyWrite(t *testing.T, doc *sav.DocumentData, policy *nativeNamePolicyTest) {
	t.Helper()
	a, err := readCurrentActions(doc)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var input map[string]json.RawMessage
	if err := json.Unmarshal(leaf, &input); err != nil {
		t.Fatal(err)
	}
	input["Manifest"], err = json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err = json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
		t.Fatal(err)
	}
}

func nativeNameManifestCheck(t *testing.T, f *FrontEnd, want map[sim.EntityID]string, namesOnly bool) {
	t.Helper()
	manifest := f.live.mission.state.ActorManifest
	if manifest == nil || len(manifest.Actors) != len(want) {
		t.Fatal("ordinary native names unavailable", manifest, want)
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var presence struct{ NativeNamesOnly bool }
	if err := json.Unmarshal(encoded, &presence); err != nil || presence.NativeNamesOnly != namesOnly {
		t.Fatal("source manifest presence changed", string(encoded), namesOnly, err)
	}
	for _, row := range manifest.Actors {
		name, found := want[row.ID]
		e, live := f.live.world.Entity(row.ID)
		if !found || row.Name != name || row.Constructed || !live || e.SourceBinding != (sim.SourceBinding{}) || e.ActorLoad.Source != (sim.SourceActor{}) || e.ActorLoad.Present || e.NativeClass.Present {
			t.Fatal("ordinary native name or source absence changed", row, name, e)
		}
		if f.live.actorNames[row.ID] != name {
			t.Fatal("native name unavailable to presentation", row, f.live.actorNames[row.ID])
		}
	}
}

func TestLegacyNativeNamesCurrentStateThroughSAVE(t *testing.T) {
	for _, mode := range []string{"omitted", "absent", "present"} {
		t.Run(mode, func(t *testing.T) {
			f, doc, _ := legacyRawFixture(t)
			var policy *nativeNamePolicyTest
			if mode != "omitted" {
				policy = &nativeNamePolicyTest{Present: mode == "present"}
			}
			nativeNamePolicyWrite(t, &doc, policy)
			a, err := readCurrentActions(&doc)
			if err != nil {
				t.Fatal(err)
			}
			want := map[sim.EntityID]string{}
			for _, binding := range a.Bindings {
				if !binding.Structure && !binding.Missing {
					want[binding.ID] = fmt.Sprintf("ordinary native %d ", binding.ID) + string([]byte{0xff, 0xc0})
					mustSetText(&doc.Objects[binding.Object-1], "Name", want[binding.ID])
				}
			}
			raw, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			cold := nativeSubjectCold(t, f, raw)
			nativeNameManifestCheck(t, cold, want, mode != "present")
			for cycle := 0; cycle < 2; cycle++ {
				for i := range cold.live.mission.state.ActorManifest.Actors {
					row := &cold.live.mission.state.ActorManifest.Actors[i]
					row.Name = fmt.Sprintf("current native %d cycle%d", row.ID, cycle)
					want[row.ID] = row.Name
				}
				hash := cold.live.world.Hash()
				_, saved, _ := saveCurrentEffect(t, cold)
				p := nativeNamePolicyRead(t, &saved)
				if p == nil || p.Present != (mode == "present") || p.Actors != nil || len(p.NativeActors) != len(want) {
					t.Fatal("native name membership promoted source manifest", p)
				}
				private, _ := json.Marshal(p)
				if bytes.Contains(private, []byte("\"Name\"")) || bytes.Contains(private, []byte("current native")) {
					t.Fatal("native name acquired a duplicate private value")
				}
				a, err := readCurrentActions(&saved)
				if err != nil {
					t.Fatal(err)
				}
				leaf, _, _ := sav.NativeActions(saved.State)
				for _, binding := range a.Bindings {
					name, found := want[binding.ID]
					if binding.Structure || binding.Missing || !found {
						continue
					}
					record := &saved.Objects[binding.Object-1]
					for _, text := range record.Texts {
						if text.Name == "Name" && text.Value != name {
							t.Fatal("SAVE reused stale ordinary name", text.Value, name)
						}
					}
					want[binding.ID] = name + " ordinary edit"
					mustSetText(record, "Name", want[binding.ID])
				}
				after, _, _ := sav.NativeActions(saved.State)
				if !bytes.Equal(leaf, after) {
					t.Fatal("ordinary Name edit changed private membership")
				}
				raw, err = sav.EncodeDocumentData(saved)
				if err != nil {
					t.Fatal(err)
				}
				cold = nativeSubjectCold(t, f, raw)
				nativeNameManifestCheck(t, cold, want, mode != "present")
				if cold.live.world.Hash() != hash {
					t.Fatal("presentation name changed World state")
				}
			}
		})
	}
}

func TestNativeActorNameModernAbsenceIsPreserved(t *testing.T) {
	f, raw := nativeSubjectFixture(t)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a.NativeHistoryVersion != 1 {
		t.Fatal(err)
	}
	for _, b := range a.Bindings {
		if !b.Structure && !b.Missing {
			mustSetText(&doc.Objects[b.Object-1], "Name", "ordinary modern name")
		}
	}
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	for cycle := 0; cycle < 2; cycle++ {
		cold := nativeSubjectCold(t, f, raw)
		if cold.live.mission.state.ActorManifest != nil {
			t.Fatal("modern absent manifest acquired native observations")
		}
		raw, doc, _ = saveCurrentEffect(t, cold)
		p := nativeNamePolicyRead(t, &doc)
		if p == nil || p.Present || p.Actors != nil || p.NativeActors != nil {
			t.Fatal("modern absent source/native name policy changed", p)
		}
	}
}

func TestLegacyNativeNamesOmittedPolicyKeepsSourceManifest(t *testing.T) {
	f := currentManifestFront(t, false)
	doc, err := sav.DecodeDocumentData(currentRuntimeSave(t, f))
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	a.NativeHistoryVersion, a.Manifest = 0, nil
	leaf, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
		t.Fatal(err)
	}
	want := map[sim.EntityID]string{}
	source, native := 0, 0
	for _, b := range a.Bindings {
		if b.Structure || b.Missing {
			continue
		}
		if a.Values[b.ID].SourceBound {
			source++
		} else {
			native++
		}
		want[b.ID] = fmt.Sprintf("mixed ordinary %d", b.ID)
		mustSetText(&doc.Objects[b.Object-1], "Name", want[b.ID])
	}
	if source == 0 || native == 0 {
		t.Fatal("mixed source/native fixture missing", source, native)
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	cold := actorRegistryFront1111(t)
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal(town, err)
	}
	if err := cold.App("mixed ordinary names").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	manifest := cold.live.mission.state.ActorManifest
	if manifest == nil || len(manifest.Actors) != len(want) {
		t.Fatal("mixed source/native names unavailable", manifest, want)
	}
	for _, row := range manifest.Actors {
		if row.Name != want[row.ID] {
			t.Fatal("mixed ordinary name changed", row, want[row.ID])
		}
		e, live := cold.live.world.Entity(row.ID)
		if !live || (e.SourceBinding.Class != 0) != a.Values[row.ID].SourceBound {
			t.Fatal("mixed source provenance changed", e, a.Values[row.ID])
		}
	}
	_, saved, _ := saveCurrentEffect(t, cold)
	p := nativeNamePolicyRead(t, &saved)
	if p == nil || !p.Present || len(p.Actors) != source || len(p.NativeActors) != native {
		t.Fatal("omitted policy lost the existing source manifest", p, source, native)
	}
}

func TestNativeActorNameMalformedMembershipIsAtomic(t *testing.T) {
	f, doc, _ := legacyRawFixture(t)
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	cold := nativeSubjectCold(t, f, raw)
	raw, _, _ = saveCurrentEffect(t, cold)
	before, _ := cold.live.world.MarshalBinary()
	live := cold.live
	manifest := cloneActorManifest(cold.live.mission.state.ActorManifest)
	for _, change := range []string{"duplicate", "unknown", "source membership", "wrong class", "missing name"} {
		t.Run(change, func(t *testing.T) {
			candidate, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			p := nativeNamePolicyRead(t, &candidate)
			if p == nil || len(p.NativeActors) != 2 {
				t.Fatal("native name control fixture unavailable", p)
			}
			a, err := readCurrentActions(&candidate)
			if err != nil {
				t.Fatal(err)
			}
			var object uint16
			for _, b := range a.Bindings {
				if !b.Structure && b.ID == p.NativeActors[0].Entity {
					object = b.Object
				}
			}
			switch change {
			case "duplicate":
				p.NativeActors = append(p.NativeActors, p.NativeActors[0])
			case "unknown":
				p.NativeActors[0].Entity = 999
			case "source membership":
				p.Present = true
				p.Actors, p.NativeActors = p.NativeActors, nil
			case "wrong class":
				candidate.Objects[object-1].Class = "Human"
			case "missing name":
				for i := range candidate.Objects[object-1].Texts {
					if candidate.Objects[object-1].Texts[i].Name == "Name" {
						candidate.Objects[object-1].Texts[i].Name = "MissingName"
					}
				}
			}
			nativeNamePolicyWrite(t, &candidate, p)
			candidateRaw, err := sav.EncodeDocumentData(candidate)
			if err != nil {
				if change == "wrong class" || change == "missing name" {
					return
				}
				t.Fatal(err)
			}
			_, _, err = cold.RestoreOriginal(candidateRaw)
			after, marshalErr := cold.live.world.MarshalBinary()
			if err == nil || marshalErr != nil || !bytes.Equal(before, after) || cold.live != live || !reflect.DeepEqual(manifest, cold.live.mission.state.ActorManifest) {
				t.Fatal("invalid native name membership published", err, marshalErr)
			}
		})
	}
}

func TestNativeObservedClassNameUsesInstalledCaption(t *testing.T) {
	for _, tc := range []struct {
		name   string
		raw    string
		label  string
		want   string
		source bool
	}{
		{"native default", "definition name", "localized creature", "localized creature", false},
		{"native custom", "current instance name", "localized creature", "current instance name", false},
		{"native default without label", "definition name", "", "definition name", false},
		{"source instance", "definition name", "localized creature", "definition name", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actor := sim.Entity{ID: 17, Class: 69, TypeID: 69, X: 1, Y: 1,
				Owner: sim.SelfSlot, HP: 9, MaxHP: 31, Speed: 100, RotationSpeed: 255}
			if tc.source {
				actor.SourceBinding = sim.SourceBinding{Class: 1, ArchiveIndex: 18, TypeID: 69, TokenRow: 4, Face: 3}
				actor.ActorLoad = sim.ActorLoad{Present: true, Source: sim.SourceActor{Class: 1, TypeID: 69}}
			} else {
				actor.NativeBasis = sim.NativeActorBasis{}.WithBody(5)
			}
			world, err := sim.NewWorld(17, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, make([]byte, 64), []sim.Entity{actor})
			if err != nil {
				t.Fatal(err)
			}
			if e, found := world.Entity(17); !found || e.Class != 69 || e.TypeID != 69 || e.Humanoid || sim.InPersistBand(e.TypeID) {
				t.Fatal("native name control must present an ordinary creature", e, found)
			}
			manifest := &SnapshotActorManifest{Version: actorManifestVersion, NativeNamesOnly: !tc.source,
				Actors: []SnapshotActor{{ID: 17, Name: tc.raw}}}
			before := cloneActorManifest(manifest)
			if err := validateActorManifest(manifest, world); err != nil {
				t.Fatal(err)
			}
			m := &alm.Map{Width: 8, Height: 8, Tiles: make([]uint16, 64), Altitudes: make([]uint8, 64)}
			art := worldFixtureArt(16, 16, 8, 16, 3, 3, 1)
			art.Name = "definition name"
			mw := newMapWorld(world, nil, &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{69: art}}, worldFixtureViewer(t, m))
			mw.installActorManifest(manifest)
			words := mw.view.Words()
			words.UnitNames[69] = tc.label
			mw.view.SetWords(words)
			hash := world.Hash()
			draws := mw.entityDraws()
			if len(draws) != 1 || draws[0].Name != tc.want {
				t.Fatal("observed native name changed the installed card caption", draws, tc.want)
			}
			if mw.actorNames[17] != tc.raw || !reflect.DeepEqual(manifest, before) || world.Hash() != hash {
				t.Fatal("presentation changed the ordinary name carrier or World")
			}
			if e, found := world.Entity(17); !found || !tc.source &&
				(e.SourceBinding != (sim.SourceBinding{}) || e.ActorLoad.Source != (sim.SourceActor{}) || e.ActorLoad.Present || e.NativeClass.Present) {
				t.Fatal("native caption promoted source state", e, found)
			}
		})
	}
}
