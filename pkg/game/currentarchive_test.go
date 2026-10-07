package game

import (
	"bytes"
	"encoding/json"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentArchiveFixture(t *testing.T) *FrontEnd {
	t.Helper()
	f, _ := itemMutationOpen1115(t, 1, false)
	return f
}

func openCurrentArchive(t *testing.T, raw []byte) *FrontEnd {
	t.Helper()
	f := itemMutationFront1115(t, 1)
	opener, town, err := f.RestoreOriginal(raw)
	if err == nil && !town {
		err = f.App("current archive coordinates").OpenMission(opener)
	}
	if err != nil || town {
		t.Fatal("current archive cold LOAD", town, err)
	}
	return f
}

func TestCurrentArchiveCoordinatesKeepStrictWorldAcrossTwoCycles(t *testing.T) {
	f := currentArchiveFixture(t)
	before := f.live.world
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	for cycle := 0; cycle < 2; cycle++ {
		raw, err := f.ExportCurrentSave(s, "exact current archive")
		if err != nil {
			t.Fatal(err)
		}
		f = openCurrentArchive(t, raw)
		if before.Hash() != f.live.world.Hash() {
			t.Fatalf("cycle%d exact World changed: %x -> %x", cycle, before.Hash(), f.live.world.Hash())
		}
		for tick := 0; tick < 20; tick++ {
			sim.Step(before, nil)
			sim.Step(f.live.world, nil)
			if before.Hash() != f.live.world.Hash() {
				t.Fatal("exact continuation changed", cycle, tick)
			}
		}
		s, _, err = f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestCurrentArchiveCoordinatesFollowOrdinaryGroupEdits(t *testing.T) {
	f := currentArchiveFixture(t)
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, "current Group coordinate controls")
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"move", "repeat and null", "selector", "Token identity", "Player identity"} {
		t.Run(change, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || len(a.ActorGroups) < 2 {
				t.Fatal("fixture lacks source Group coordinates", err)
			}
			row := a.ActorGroups[0]
			actor := row.Object
			site := currentGroupSite{row.Player, row.Inline}
			player := &doc.Objects[site.Player-1]
			group := &player.Groups[site.Inline]
			var target uint32
			for _, b := range a.Groups {
				if b.Object == site.Player && b.Inline != site.Inline && !b.RootOnly {
					target = b.Inline
					break
				}
			}
			wantSite := site
			switch change {
			case "move", "repeat and null":
				if target == site.Inline {
					t.Fatal("fixture lacks another Group under same exact Player")
				}
				refs, _ := savedObjectRefs(&player.Groups[target], "Actors")
				refs = append(append([]uint16(nil), refs...), actor)
				if change == "repeat and null" {
					refs = append(refs, 0, actor)
				}
				literalSavedObjectRefs(t, &player.Groups[target], "Actors", refs, true)
				old, _ := savedObjectRefs(group, "Actors")
				for i, id := range old {
					if id == actor {
						old[i] = 0
					}
				}
				literalSavedObjectRefs(t, group, "Actors", old, true)
				var count uint32
				for _, g := range player.Groups {
					refs, _ := savedObjectRefs(&g, "Actors")
					count += uint32(len(refs))
				}
				mustSetCount(player, "Actors", count)
				wantSite.Inline = target
			case "selector":
				mustSetValue(group, "G1C", 83)
			case "Token identity":
				mustSetValue(&doc.Objects[actor-1], "Identity", 0x76543210)
			case "Player identity":
				key, _ := savedStructureValue(player, "This")
				mustSetValue(player, "This", 0x76543211)
				for i := range doc.Objects {
					r := &doc.Objects[i]
					for j := range r.Values {
						if r.Values[j].Name == "Reference" && r.Values[j].Value == key {
							r.Values[j].Value = 0x76543211
						}
					}
					for j := range r.Groups {
						for k := range r.Groups[j].Values {
							v := &r.Groups[j].Values[k]
							if (v.Name == "G44" || v.Name == "G40") && v.Value == key {
								v.Value = 0x76543211
							}
						}
					}
				}
			}
			doc, _, err = sav.ReindexDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			// Reindex may relocate only the explicit object/site bindings.
			reindexed, err := readCurrentActions(&doc)
			if err != nil {
				t.Fatal(err)
			}
			for _, b := range reindexed.ActorGroups {
				if b.Native == row.Native {
					actor = b.Object
					wantSite.Player = b.Player
					break
				}
			}
			candidate, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			for cycle := 0; cycle < 2; cycle++ {
				cold := openCurrentArchive(t, candidate)
				var entity sim.Entity
				for _, b := range cold.live.mission.state.savedDocument.Actors {
					if b.ObjectIndex == actor {
						for _, e := range cold.live.world.Entities() {
							if e.ID == b.EntityID {
								entity = e
							}
						}
					}
				}
				if change == "Token identity" && entity.SourceBinding.Identity != 0x76543210 {
					t.Fatal("ordinary actor key was replaced", entity.SourceBinding)
				}
				if change == "Player identity" && entity.SourceBinding.GroupOwnerKey != 0x76543211 {
					t.Fatal("ordinary owner key was replaced", entity.SourceBinding)
				}
				if change == "selector" && (entity.SourceBinding.GroupSelector != 83 || entity.Group != 83) {
					t.Fatal("ordinary selector was replaced", entity.SourceBinding, entity.Group)
				}
				if change == "move" || change == "repeat and null" {
					var wanted uint32
					for _, b := range cold.live.mission.state.savedDocument.GroupBindings.Groups {
						if b.PlayerObject == wantSite.Player && b.InlineIndex == wantSite.Inline {
							wanted = b.ID
						}
					}
					if wanted == 0 || entity.SourceBinding.GroupIndex != wanted {
						t.Fatal("ordinary Group move was replaced", entity.SourceBinding, wanted)
					}
				}
				next, _, err := cold.Snapshot(true)
				if err != nil {
					t.Fatal(err)
				}
				candidate, err = cold.ExportCurrentSave(next, "edited current coordinates")
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestCurrentArchiveCoordinateMalformedPolicyIsAtomic(t *testing.T) {
	f := currentArchiveFixture(t)
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, "invalid coordinate controls")
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"duplicate object", "coordinate collision", "null binding", "null Group anchor"} {
		doc, _ := sav.DecodeDocumentData(raw)
		a, _ := readCurrentActions(&doc)
		switch change {
		case "duplicate object":
			a.ArchiveCoordinates = append(a.ArchiveCoordinates, a.ArchiveCoordinates[0])
		case "coordinate collision":
			a.ArchiveCoordinates[1].Native = a.ArchiveCoordinates[0].Native
		case "null binding":
			a.ArchiveCoordinates[0].Object = 0
		case "null Group anchor":
			a.ActorGroups[0].Anchor = [32]byte{}
		}
		value, _ := json.Marshal(a)
		if err := sav.SetNativeActions(&doc.State, value); err != nil {
			t.Fatal(err)
		}
		candidate, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		before, _ := f.live.world.MarshalBinary()
		_, _, err = f.RestoreOriginal(candidate)
		after, _ := f.live.world.MarshalBinary()
		if err == nil || !bytes.Equal(before, after) {
			t.Fatal("malformed coordinate changed session", change, err)
		}
	}
}
