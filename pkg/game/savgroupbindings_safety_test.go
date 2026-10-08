package game

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
)

func groupBindingsSafetyClone1115(t *testing.T, source Snapshot) Snapshot {
	t.Helper()
	next := source
	var err error
	next.SavedDocument, err = cloneSavedDocument(source.SavedDocument)
	if err != nil {
		t.Fatal(err)
	}
	if next.SavedDocument == nil || next.SavedDocument.GroupBindings == nil || len(next.SavedDocument.GroupBindings.Groups) != 1 || len(next.SavedDocument.GroupBindings.Members) != 1 {
		t.Fatal("safety fixture must have one exact Group and member")
	}
	return next
}

// Add only detached synthetic graph records. These helpers distinguish a
// malformed binding from an invalid record shape or an unrelated orphan.
func groupBindingsSafetySecondActor1115(t *testing.T, s *Snapshot) uint16 {
	t.Helper()
	doc := s.SavedDocument.Document
	actor := doc.Objects[s.SavedDocument.Actors[0].ObjectIndex-1]
	doc.Objects = append(doc.Objects, actor)
	index := uint16(len(doc.Objects))
	doc.DeadActors = append(doc.DeadActors, index)
	if _, err := sav.CloneDocumentData(*doc); err != nil {
		t.Fatal("second synthetic actor graph", err)
	}
	return index
}

func groupBindingsSafetySecondGroup1115(t *testing.T, s *Snapshot) {
	t.Helper()
	bindings := s.SavedDocument.GroupBindings
	player := &s.SavedDocument.Document.Objects[bindings.Groups[0].PlayerObject-1]
	player.Groups = append(player.Groups, player.Groups[0])
	for i := range player.Counts {
		switch player.Counts[i].Name {
		case "Groups":
			player.Counts[i].Count++
		case "Actors":
			player.Counts[i].Count *= 2
		}
	}
	second := bindings.Groups[0]
	second.ID++
	second.InlineIndex++
	bindings.Groups = append(bindings.Groups, second)
	if _, err := cloneSavedDocument(s.SavedDocument); err != nil {
		t.Fatal("second synthetic inline Group", err)
	}
}

func groupBindingsSafetyPlayerReference1115(t *testing.T, s *Snapshot) SnapshotSAVGroupReferenceBinding {
	t.Helper()
	index := s.SavedDocument.GroupBindings.Groups[0].PlayerObject
	ref := SnapshotSAVGroupReferenceBinding{ObjectIndex: index, Class: 1}
	for _, value := range s.SavedDocument.Document.Objects[index-1].Values {
		switch value.Name {
		case "This":
			ref.Key = value.Value
		case "Slot":
			ref.Owner = value.Value
		}
	}
	if ref.Key == 0 {
		t.Fatal("synthetic Player has no nonzero source key")
	}
	return ref
}

func TestSavedGroupBindings1115MalformedMetadataIsAtomic(t *testing.T) {
	f, source := documentSnapshotFixture(t)
	for _, tc := range []struct {
		name, reason string
		mutate       func(*testing.T, *Snapshot)
	}{
		{"zero version", "version or bounds", func(_ *testing.T, s *Snapshot) { s.SavedDocument.GroupBindings.Version = 0 }},
		{"future version", "version or bounds", func(_ *testing.T, s *Snapshot) { s.SavedDocument.GroupBindings.Version = 2 }},
		{"long reason", "version or bounds", func(_ *testing.T, s *Snapshot) { s.SavedDocument.GroupBindings.Unavailable = strings.Repeat("x", 4097) }},
		{"nul reason", "version or bounds", func(_ *testing.T, s *Snapshot) { s.SavedDocument.GroupBindings.Unavailable = "missing\x00scope" }},
		{"unavailable document", "invalid unavailable state", func(_ *testing.T, s *Snapshot) {
			s.SavedDocument.Document, s.SavedDocument.Actors = nil, nil
			s.SavedDocument.Unavailable = "complete document unavailable"
		}},
		{"missing Group", "cover inline Groups", func(_ *testing.T, s *Snapshot) { s.SavedDocument.GroupBindings.Groups = nil }},
		{"extra Group", "cover inline Groups", func(_ *testing.T, s *Snapshot) {
			g := s.SavedDocument.GroupBindings
			g.Groups = append(g.Groups, g.Groups[0])
		}},
		{"zero Group ID", "identity or container", func(_ *testing.T, s *Snapshot) { s.SavedDocument.GroupBindings.Groups[0].ID = 0 }},
		{"null container", "identity or container", func(_ *testing.T, s *Snapshot) { s.SavedDocument.GroupBindings.Groups[0].PlayerObject = 0 }},
		{"out of range container", "identity or container", func(_ *testing.T, s *Snapshot) { s.SavedDocument.GroupBindings.Groups[0].PlayerObject = 65535 }},
		{"nonplayer container", "identity or container", func(_ *testing.T, s *Snapshot) {
			s.SavedDocument.GroupBindings.Groups[0].PlayerObject = s.SavedDocument.Actors[0].ObjectIndex
		}},
		{"out of range inline", "identity or container", func(_ *testing.T, s *Snapshot) { s.SavedDocument.GroupBindings.Groups[0].InlineIndex = ^uint32(0) }},
		{"duplicate Group ID", "identity or container", func(t *testing.T, s *Snapshot) {
			groupBindingsSafetySecondGroup1115(t, s)
			g := s.SavedDocument.GroupBindings.Groups
			g[1].ID = g[0].ID
		}},
		{"duplicate inline container", "identity or container", func(t *testing.T, s *Snapshot) {
			groupBindingsSafetySecondGroup1115(t, s)
			g := s.SavedDocument.GroupBindings.Groups
			g[1].InlineIndex = g[0].InlineIndex
		}},
		{"null member object", "ambiguous member", func(_ *testing.T, s *Snapshot) { s.SavedDocument.GroupBindings.Members[0].ObjectIndex = 0 }},
		{"out of range member", "ambiguous member", func(_ *testing.T, s *Snapshot) { s.SavedDocument.GroupBindings.Members[0].ObjectIndex = 65535 }},
		{"nonactor member", "non-actor object", func(_ *testing.T, s *Snapshot) {
			s.SavedDocument.GroupBindings.Members[0].ObjectIndex = s.SavedDocument.GroupBindings.Groups[0].PlayerObject
		}},
		{"duplicate member object", "ambiguous member", func(_ *testing.T, s *Snapshot) {
			g := s.SavedDocument.GroupBindings
			g.Members = append(g.Members, g.Members[0])
		}},
		{"duplicate member entity", "ambiguous member", func(t *testing.T, s *Snapshot) {
			index := groupBindingsSafetySecondActor1115(t, s)
			g := s.SavedDocument.GroupBindings
			member := g.Members[0]
			member.ObjectIndex = index
			g.Members = append(g.Members, member)
		}},
		{"bound unresolved handle", "ambiguous member", func(_ *testing.T, s *Snapshot) { s.SavedDocument.GroupBindings.Members[0].UnresolvedHandle = 1 }},
		{"unbound null handle", "ambiguous member", func(_ *testing.T, s *Snapshot) { s.SavedDocument.GroupBindings.Members[0].Bound = false }},
		{"unbound native identity", "ambiguous member", func(_ *testing.T, s *Snapshot) {
			member := &s.SavedDocument.GroupBindings.Members[0]
			member.Bound, member.UnresolvedHandle, member.EntityID = false, 1, 1
		}},
		{"member count exceeds objects", "version or bounds", func(_ *testing.T, s *Snapshot) {
			s.SavedDocument.GroupBindings.Members = make([]SnapshotSAVGroupMemberBinding, len(s.SavedDocument.Document.Objects)+1)
		}},
		{"actor binding conflict", "disagrees with actor binding", func(t *testing.T, s *Snapshot) {
			s.SavedDocument.GroupBindings.Members[0].ObjectIndex = groupBindingsSafetySecondActor1115(t, s)
		}},
		{"unknown reference class", "invalid resolved reference", func(_ *testing.T, s *Snapshot) { s.SavedDocument.GroupBindings.Groups[0].Reference.Class = 3 }},
		{"unknown owner class", "invalid resolved reference", func(_ *testing.T, s *Snapshot) { s.SavedDocument.GroupBindings.Groups[0].Owner.Class = 3 }},
		{"resolved null reference", "invalid resolved reference", func(_ *testing.T, s *Snapshot) {
			s.SavedDocument.GroupBindings.Groups[0].Reference = SnapshotSAVGroupReferenceBinding{Class: 1, Key: 1}
		}},
		{"unresolved nonnull reference", "invalid resolved reference", func(_ *testing.T, s *Snapshot) {
			s.SavedDocument.GroupBindings.Groups[0].Reference = SnapshotSAVGroupReferenceBinding{ObjectIndex: s.SavedDocument.Actors[0].ObjectIndex}
		}},
		{"reference beyond graph", "invalid resolved reference", func(_ *testing.T, s *Snapshot) {
			s.SavedDocument.GroupBindings.Groups[0].Reference = SnapshotSAVGroupReferenceBinding{ObjectIndex: 65535, Class: 2, Key: 1}
		}},
		{"Player reference to actor", "class/key mismatch", func(_ *testing.T, s *Snapshot) {
			s.SavedDocument.GroupBindings.Groups[0].Reference = SnapshotSAVGroupReferenceBinding{ObjectIndex: s.SavedDocument.Actors[0].ObjectIndex, Class: 1, Key: 1}
		}},
		{"object reference to Player", "class/key mismatch", func(_ *testing.T, s *Snapshot) {
			g := &s.SavedDocument.GroupBindings.Groups[0]
			g.Reference = SnapshotSAVGroupReferenceBinding{ObjectIndex: g.PlayerObject, Class: 2, Key: 1}
		}},
		{"reference zero key", "class/key mismatch", func(t *testing.T, s *Snapshot) {
			ref := groupBindingsSafetyPlayerReference1115(t, s)
			ref.Key = 0
			s.SavedDocument.GroupBindings.Groups[0].Reference = ref
		}},
		{"nonplayer reference owner", "invalid resolved reference", func(_ *testing.T, s *Snapshot) {
			s.SavedDocument.GroupBindings.Groups[0].Reference = SnapshotSAVGroupReferenceBinding{ObjectIndex: s.SavedDocument.Actors[0].ObjectIndex, Class: 2, Key: 1, Owner: 1}
		}},
		{"reference source key mismatch", "different source key", func(t *testing.T, s *Snapshot) {
			ref := groupBindingsSafetyPlayerReference1115(t, s)
			ref.Key++
			s.SavedDocument.GroupBindings.Groups[0].Reference = ref
		}},
		{"reference Player slot mismatch", "different Player slot", func(t *testing.T, s *Snapshot) {
			ref := groupBindingsSafetyPlayerReference1115(t, s)
			ref.Owner++
			s.SavedDocument.GroupBindings.Groups[0].Reference = ref
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := groupBindingsSafetyClone1115(t, source)
			tc.mutate(t, &s)
			if encoded, err := EncodeSave(s, "malformed Group binding"); err == nil || !strings.Contains(err.Error(), tc.reason) || encoded != nil {
				t.Fatalf("EncodeSave refusal %v, want %q and no output", err, tc.reason)
			}
			decoded, label, err := DecodeSave(uncheckedDocumentEnvelope1115(t, s))
			if err == nil || !strings.Contains(err.Error(), tc.reason) || label != "" || !reflect.DeepEqual(decoded, Snapshot{}) {
				t.Fatalf("DecodeSave refusal %v, want %q and no partial state", err, tc.reason)
			}
			before, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			open, town, err := f.Restore(s)
			if err == nil || !strings.Contains(err.Error(), tc.reason) || open != nil || town {
				t.Fatalf("Restore refusal %v, want %q and no candidate", err, tc.reason)
			}
			after, _, err := f.Snapshot(true)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("refused Group metadata changed the complete active snapshot", err)
			}
		})
	}
}

func TestSavedGroupBindings1115EmptyPresenceAndOwnership(t *testing.T) {
	for _, present := range []bool{false, true} {
		t.Run(fmt.Sprintf("present-%t", present), func(t *testing.T) {
			var source *SnapshotSAVGroupBindings
			if present {
				source = &SnapshotSAVGroupBindings{Version: 1}
			}
			cloned, err := cloneSavedGroupBindings(source, &sav.DocumentData{}, nil)
			if err != nil || (cloned != nil) != present || !reflect.DeepEqual(source, cloned) {
				t.Fatal("absent and present-empty binding metadata were conflated", err)
			}
			if present && cloned == source {
				t.Fatal("present-empty metadata is not independently owned")
			}
		})
	}
	_, snapshot := documentSnapshotFixture(t)
	source := snapshot.SavedDocument.GroupBindings
	baseline := *source
	baseline.Groups = append([]SnapshotSAVGroupBinding(nil), source.Groups...)
	baseline.Members = append([]SnapshotSAVGroupMemberBinding(nil), source.Members...)
	cloned, err := cloneSavedDocument(snapshot.SavedDocument)
	if err != nil {
		t.Fatal(err)
	}
	cloned.GroupBindings.Groups[0].ID++
	cloned.GroupBindings.Groups[0].Reference.Key++
	cloned.GroupBindings.Members[0].ObjectIndex++
	cloned.GroupBindings.Unavailable = "changed detached clone"
	if !reflect.DeepEqual(*source, baseline) {
		t.Fatal("clone's Group/member slices alias the source metadata")
	}
	encoded, err := EncodeSave(snapshot, "owned Group bindings")
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(encoded)
	if err != nil || !reflect.DeepEqual(decoded.SavedDocument.GroupBindings, source) {
		t.Fatal("native encoding lost exact Group binding values", err)
	}
	decoded.SavedDocument.GroupBindings.Groups[0].InlineIndex++
	decoded.SavedDocument.GroupBindings.Members[0].UnresolvedHandle++
	if !reflect.DeepEqual(*source, baseline) {
		t.Fatal("decoded Group bindings alias the source")
	}
}
