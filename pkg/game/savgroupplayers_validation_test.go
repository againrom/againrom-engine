package game

import (
	"bytes"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func groupPlayersValidationSnapshot1115(t *testing.T) (*FrontEnd, Snapshot) {
	t.Helper()
	front := groupDocumentFront1115(t)
	open, town, err := front.RestoreOriginal(groupDocumentLiteral1115(t, front))
	if err != nil || town {
		t.Fatal("synthetic Player/Group document LOAD", err, town)
	}
	if err := front.App("Player binding validation").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	s := groupDocumentSnapshot1115(t, front)
	b := s.SavedDocument.GroupBindings
	if !b.PlayersPresent || len(b.Players) != 1 || len(b.Groups) != 1 || b.Groups[0].ContainerID != b.Players[0].ID || b.Groups[0].Authored || b.Unavailable != "" {
		t.Fatal("fixture lacks current exact imported Player/Group provenance", b)
	}
	return front, s
}

func groupPlayersValidationClone1115(t *testing.T, source Snapshot) Snapshot {
	t.Helper()
	s := source
	s.World = bytes.Clone(source.World)
	var err error
	s.SavedDocument, err = cloneSavedDocument(source.SavedDocument)
	if err != nil {
		t.Fatal("clone valid Player binding fixture", err)
	}
	return s
}

// Use the complete synthetic wire fixture at every externally supplied
// snapshot seam. A rejected DTO may not leave a partial result or mutate the
// already running session, even when a sender bypasses EncodeSave entirely.
func groupPlayersValidationReject1115(t *testing.T, front *FrontEnd, s Snapshot) {
	t.Helper()
	if out, err := cloneSavedDocument(s.SavedDocument); err == nil || out != nil {
		t.Errorf("clone accepted malformed Player/authored provenance: out=%v err=%v", out != nil, err)
	}
	if out, err := EncodeSave(s, "hostile Player binding"); err == nil || out != nil {
		t.Errorf("EncodeSave accepted malformed Player/authored provenance: bytes=%d err=%v", len(out), err)
	}
	if out, label, err := DecodeSave(uncheckedDocumentEnvelope1115(t, s)); err == nil || label != "" || !reflect.DeepEqual(out, Snapshot{}) {
		t.Errorf("DecodeSave returned malformed/partial snapshot: label=%q err=%v", label, err)
	}
	groupPlayersValidationRestoreReject1115(t, front, s)
}

func groupPlayersValidationRestoreReject1115(t *testing.T, front *FrontEnd, s Snapshot) {
	t.Helper()
	before, _, err := front.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if open, town, err := front.Restore(s); err == nil || open != nil || town {
		t.Errorf("Restore accepted malformed/partial provenance: opener=%v town=%v err=%v", open != nil, town, err)
	}
	after, _, err := front.Snapshot(true)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Errorf("rejected Restore changed the active session: %v", err)
	}
}

func TestSavedGroupPlayersValidation1115RegistryPresence(t *testing.T) {
	for _, present := range []bool{false, true} {
		state := &SnapshotSAVGroupBindings{Version: 1, PlayersPresent: present}
		out, err := cloneSavedGroupBindings(state, &sav.DocumentData{}, nil)
		if err != nil || out == nil || out.PlayersPresent != present || len(out.Players) != 0 {
			t.Fatalf("absent/present-empty distinction lost: present=%v out=%+v err=%v", present, out, err)
		}
		for _, nativePresent := range []bool{false, true} {
			world := &sim.World{}
			if err := world.ImportSavedGroups(nil, nil); err != nil {
				t.Fatal(err)
			}
			if nativePresent {
				if err := world.ImportSavedGroupPlayers(nil, nil); err != nil {
					t.Fatal(err)
				}
			}
			for _, unavailable := range []string{"", "current Group projection unavailable"} {
				out.Unavailable = unavailable
				err := validateSavedGroupBindingWorld(&SnapshotSAVDocument{Document: &sav.DocumentData{}, GroupBindings: out}, world)
				if (err == nil) != (present == nativePresent) {
					t.Fatalf("registry presence: document=%v native=%v unavailable=%q err=%v", present, nativePresent, unavailable, err)
				}
			}
		}
	}
	if err := validateSavedGroupBindingWorld(&SnapshotSAVDocument{Document: &sav.DocumentData{}, GroupBindings: &SnapshotSAVGroupBindings{Version: 1}}, &sim.World{}); err == nil {
		t.Fatal("present document bindings accepted an absent native Group registry")
	}
}

func TestSavedGroupPlayersValidation1115AliasesDistinctRootsAndMalformedRegistry(t *testing.T) {
	// This intentionally minimal graph isolates binding identity validation.
	// Both distinct Players have the same semantic slot. The repeated root is
	// one container; the second object remains another, regardless of slot.
	fixture := func() (*sav.DocumentData, *SnapshotSAVGroupBindings) {
		return &sav.DocumentData{
				Players: []uint16{1, 0, 1, 2},
				Objects: []sav.DocumentRecordData{
					{Class: "Player", Values: []sav.DocumentValueData{{Name: "Slot", Value: 1}, {Name: "This", Value: 0x111}}, Groups: []sav.DocumentRecordData{{Class: "Group"}}},
					{Class: "Player", Values: []sav.DocumentValueData{{Name: "Slot", Value: 1}, {Name: "This", Value: 0x222}}},
					{Class: "Player"}, // Not a root: class alone grants no identity.
					{Class: "Unit"},
				},
			}, &SnapshotSAVGroupBindings{
				Version: 1, PlayersPresent: true,
				Players: []SnapshotSAVGroupPlayerBinding{{ID: 7, ObjectIndex: 1}, {ID: 9, ObjectIndex: 2}},
				Groups:  []SnapshotSAVGroupBinding{{ID: 1, PlayerObject: 1, ContainerID: 7}},
			}
	}
	doc, bindings := fixture()
	if out, err := cloneSavedGroupBindings(bindings, doc, nil); err != nil || !reflect.DeepEqual(out, bindings) {
		t.Fatal("alias/distinct same-slot roots were merged or rejected", out, err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*sav.DocumentData, *SnapshotSAVGroupBindings)
	}{
		{"missing root binding", func(_ *sav.DocumentData, b *SnapshotSAVGroupBindings) { b.Players = b.Players[:1] }},
		{"present empty with roots", func(_ *sav.DocumentData, b *SnapshotSAVGroupBindings) { b.Players = nil }},
		{"alias counted twice", func(_ *sav.DocumentData, b *SnapshotSAVGroupBindings) {
			b.Players = append(b.Players, SnapshotSAVGroupPlayerBinding{ID: 11, ObjectIndex: 1})
		}},
		{"zero ID", func(_ *sav.DocumentData, b *SnapshotSAVGroupBindings) { b.Players[0].ID = 0 }},
		{"duplicate ID", func(_ *sav.DocumentData, b *SnapshotSAVGroupBindings) { b.Players[1].ID = b.Players[0].ID }},
		{"unordered ID", func(_ *sav.DocumentData, b *SnapshotSAVGroupBindings) { b.Players[1].ID = 6 }},
		{"reversed distinct root encounter order", func(_ *sav.DocumentData, b *SnapshotSAVGroupBindings) {
			b.Players[0].ObjectIndex, b.Players[1].ObjectIndex = 2, 1
			b.Groups[0].ContainerID = 9 // Keep exact per-Group containment valid.
		}},
		{"zero object", func(_ *sav.DocumentData, b *SnapshotSAVGroupBindings) { b.Players[1].ObjectIndex = 0 }},
		{"duplicate object", func(_ *sav.DocumentData, b *SnapshotSAVGroupBindings) { b.Players[1].ObjectIndex = 1 }},
		{"nonroot Player object", func(_ *sav.DocumentData, b *SnapshotSAVGroupBindings) { b.Players[1].ObjectIndex = 3 }},
		{"nonplayer object", func(_ *sav.DocumentData, b *SnapshotSAVGroupBindings) { b.Players[1].ObjectIndex = 4 }},
		{"out of range object", func(_ *sav.DocumentData, b *SnapshotSAVGroupBindings) { b.Players[1].ObjectIndex = 65535 }},
		{"absent registry with bindings", func(_ *sav.DocumentData, b *SnapshotSAVGroupBindings) { b.PlayersPresent = false }},
		{"null container", func(_ *sav.DocumentData, b *SnapshotSAVGroupBindings) { b.Groups[0].ContainerID = 0 }},
		{"unknown container", func(_ *sav.DocumentData, b *SnapshotSAVGroupBindings) { b.Groups[0].ContainerID = 8 }},
		{"different exact container same slot", func(_ *sav.DocumentData, b *SnapshotSAVGroupBindings) { b.Groups[0].ContainerID = 9 }},
		{"authored owner is different Player same slot", func(_ *sav.DocumentData, b *SnapshotSAVGroupBindings) {
			b.Groups[0].Authored = true
			b.Groups[0].Owner = SnapshotSAVGroupReferenceBinding{ObjectIndex: 2, Key: 0x222, Class: 1, Owner: 1}
			b.Unavailable = "current roster has an export coverage gap"
		}},
		{"nonplayer root", func(d *sav.DocumentData, _ *SnapshotSAVGroupBindings) { d.Players[3] = 4 }},
		{"out of range root", func(d *sav.DocumentData, _ *SnapshotSAVGroupBindings) { d.Players[3] = 65535 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, bindings := fixture()
			tc.mutate(doc, bindings)
			if out, err := cloneSavedGroupBindings(bindings, doc, nil); err == nil || out != nil {
				t.Fatal("accepted malformed exact Player identity/container", out, err)
			}
		})
	}
}

func TestSavedGroupPlayersValidation1115MalformedSnapshotIsAtomic(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Snapshot)
	}{
		{"missing Players", func(s *Snapshot) { s.SavedDocument.GroupBindings.Players = nil }},
		{"absent Players with bindings", func(s *Snapshot) { s.SavedDocument.GroupBindings.PlayersPresent = false }},
		{"zero Player ID", func(s *Snapshot) { s.SavedDocument.GroupBindings.Players[0].ID = 0 }},
		{"duplicate Player", func(s *Snapshot) { b := s.SavedDocument.GroupBindings; b.Players = append(b.Players, b.Players[0]) }},
		{"null Player object", func(s *Snapshot) { s.SavedDocument.GroupBindings.Players[0].ObjectIndex = 0 }},
		{"nonplayer object", func(s *Snapshot) {
			s.SavedDocument.GroupBindings.Players[0].ObjectIndex = s.SavedDocument.Actors[0].ObjectIndex
		}},
		{"out of range Player object", func(s *Snapshot) { s.SavedDocument.GroupBindings.Players[0].ObjectIndex = 65535 }},
		{"null container", func(s *Snapshot) { s.SavedDocument.GroupBindings.Groups[0].ContainerID = 0 }},
		{"unknown container", func(s *Snapshot) { s.SavedDocument.GroupBindings.Groups[0].ContainerID++ }},
		{"container without registry", func(s *Snapshot) { b := s.SavedDocument.GroupBindings; b.PlayersPresent, b.Players = false, nil }},
		{"authored without registry", func(s *Snapshot) {
			b := s.SavedDocument.GroupBindings
			b.PlayersPresent, b.Players = false, nil
			b.Groups[0].ContainerID, b.Groups[0].Authored = 0, true
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			front, source := groupPlayersValidationSnapshot1115(t)
			s := groupPlayersValidationClone1115(t, source)
			tc.mutate(&s)
			groupPlayersValidationReject1115(t, front, s)
		})
	}
}

func TestSavedGroupPlayersValidation1115MalformedAuthoredEvenWhenUnavailable(t *testing.T) {
	for _, unavailable := range []string{"", "new current action has an export coverage gap"} {
		for _, name := range []string{"reference", "unresolved reference key", "missing owner", "actor owner"} {
			t.Run(name+"/"+unavailable, func(t *testing.T) {
				front, source := groupPlayersValidationSnapshot1115(t)
				s := groupPlayersValidationClone1115(t, source)
				b := s.SavedDocument.GroupBindings
				g := &b.Groups[0]
				g.Authored = true
				g.Reference = SnapshotSAVGroupReferenceBinding{}
				g.Owner = groupBindingsSafetyPlayerReference1115(t, &s)
				b.Unavailable = unavailable
				switch name {
				case "reference":
					g.Reference = g.Owner
				case "unresolved reference key":
					g.Reference.Key = 0x1234
				case "missing owner":
					g.Owner = SnapshotSAVGroupReferenceBinding{}
				case "actor owner":
					index := s.SavedDocument.Actors[0].ObjectIndex
					key := actorProjectionValue(t, s.SavedDocument.Document.Objects[index-1], "Identity")
					g.Owner = SnapshotSAVGroupReferenceBinding{ObjectIndex: index, Class: 2, Key: key}
				}
				groupPlayersValidationReject1115(t, front, s)
			})
		}
	}
}

func TestSavedGroupPlayersValidation1115NativeDocumentPairing(t *testing.T) {
	for _, unavailable := range []string{"", "current Group roster cannot be projected"} {
		for _, name := range []string{"registry presence", "Player ID", "Player slot"} {
			t.Run(name+"/"+unavailable, func(t *testing.T) {
				front, source := groupPlayersValidationSnapshot1115(t)
				s := groupPlayersValidationClone1115(t, source)
				b := s.SavedDocument.GroupBindings
				b.Unavailable = unavailable
				switch name {
				case "registry presence":
					b.PlayersPresent, b.Players = false, nil
					b.FormationsPresent = false
					b.Groups[0].ContainerID = 0
					s.SavedDocument.PlayerPurses = nil
				case "Player ID":
					b.Players[0].ID++
					b.Groups[0].ContainerID = b.Players[0].ID
					s.SavedDocument.PlayerPurses.Players[0].PlayerID = b.Players[0].ID
				case "Player slot":
					index := b.Players[0].ObjectIndex
					p := &s.SavedDocument.Document.Objects[index-1]
					var slot uint32
					for i := range p.Values {
						if p.Values[i].Name == "Slot" {
							p.Values[i].Value++
							slot = uint32(uint16(p.Values[i].Value))
						}
					}
					s.SavedDocument.PlayerPurses.Players[0].Slot = slot
					for i := range b.Groups {
						for _, ref := range []*SnapshotSAVGroupReferenceBinding{&b.Groups[i].Reference, &b.Groups[i].Owner} {
							if ref.Class == 1 && ref.ObjectIndex == index {
								ref.Owner = slot
							}
						}
					}
				}
				if _, err := cloneSavedDocument(s.SavedDocument); err != nil {
					t.Fatal("pairing probe must be structurally valid on its own", err)
				}
				groupPlayersValidationRestoreReject1115(t, front, s)
			})
		}
	}
	// Unavailable deliberately permits a previous Group roster. Without that
	// marker an imported native Group may not be relabelled as command-authored.
	t.Run("covered authored provenance mismatch", func(t *testing.T) {
		front, source := groupPlayersValidationSnapshot1115(t)
		s := groupPlayersValidationClone1115(t, source)
		g := &s.SavedDocument.GroupBindings.Groups[0]
		g.Authored, g.Reference = true, SnapshotSAVGroupReferenceBinding{}
		g.Owner = groupBindingsSafetyPlayerReference1115(t, &s)
		if _, err := cloneSavedDocument(s.SavedDocument); err != nil {
			t.Fatal("valid authored shape needed to isolate native mismatch", err)
		}
		groupPlayersValidationRestoreReject1115(t, front, s)
	})
}

func TestSavedGroupPlayersValidation1115DetachedCloneAndNativeRoundTrip(t *testing.T) {
	_, source := groupPlayersValidationSnapshot1115(t)
	want := groupPlayersValidationClone1115(t, source)
	cloned := groupPlayersValidationClone1115(t, source)
	cloned.SavedDocument.GroupBindings.Players[0].ID++
	cloned.SavedDocument.GroupBindings.Players[0].ObjectIndex++
	cloned.SavedDocument.GroupBindings.Groups[0].ContainerID++
	cloned.SavedDocument.GroupBindings.Groups[0].Authored = true
	if !reflect.DeepEqual(source, want) {
		t.Fatal("Player/container/authored clone aliases its source")
	}
	encoded, err := EncodeSave(source, "exact Player containers")
	if err != nil {
		t.Fatal(err)
	}
	decoded, label, err := DecodeSave(encoded)
	if err != nil || label != "exact Player containers" || !reflect.DeepEqual(decoded.SavedDocument.GroupBindings, source.SavedDocument.GroupBindings) {
		t.Fatal("native round trip changed Player/container/authored provenance", label, err)
	}
	front := groupDocumentFront1115(t)
	open, town, err := front.Restore(decoded)
	if err != nil || town {
		t.Fatal("exact native/document pairing failed LOAD", err, town)
	}
	if err := front.App("Player binding round trip").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	before := groupDocumentSnapshot1115(t, front)
	decoded.SavedDocument.GroupBindings.Players[0].ID++
	decoded.SavedDocument.GroupBindings.Groups[0].ContainerID++
	after := groupDocumentSnapshot1115(t, front)
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(source, want) {
		t.Fatal("decoded Player slice aliases source or restored session")
	}
}
