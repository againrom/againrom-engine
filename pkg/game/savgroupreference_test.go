package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestSavedGroupDocument1115ResolvedAndMissingKeys(t *testing.T) {
	for _, reference := range []uint32{1004, 0xdeadbeef} {
		t.Run(map[bool]string{true: "resolved", false: "unresolved"}[reference == 1004], func(t *testing.T) {
			front := groupDocumentFront1115(t)
			doc, err := sav.DecodeDocumentData(groupDocumentLiteral1115(t, front))
			if err != nil {
				t.Fatal(err)
			}
			player := &doc.Objects[doc.Players[0]-1]
			set := func(record *sav.DocumentRecordData, name string, value uint32) {
				t.Helper()
				for i := range record.Values {
					if record.Values[i].Name == name {
						record.Values[i].Value = value
						return
					}
				}
				t.Fatal("fixture has no field", name)
			}
			set(player, "This", 0xface1234)
			set(player, "Slot", 1)
			set(&player.Groups[0], "G40", reference)
			set(&player.Groups[0], "G44", 0xface1234)
			raw, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			mission, _, err := loadOriginalMission(front, raw)
			if err != nil {
				t.Fatal(err)
			}
			clear(raw)
			state, err := cloneSavedDocument(mission.savedDocument)
			if err != nil {
				t.Fatal(err)
			}
			priorRef := state.GroupBindings.Groups[0].Reference.ObjectIndex
			groups, orders, _ := mission.World.SavedGroups()
			slices.Reverse(groups[0].Members)
			if err := mission.World.ImportSavedGroups(groups, orders); err != nil {
				t.Fatal(err)
			}
			if err := projectSavedGroups(state, mission.World); err != nil || state.GroupBindings.Unavailable != "" {
				t.Fatal("project references", err, state.GroupBindings.Unavailable)
			}
			g := state.GroupBindings.Groups[0]
			if g.Owner.ObjectIndex != state.Document.Players[0] || g.Owner.Owner != 1 {
				t.Fatal("Player reference binding changed", g)
			}
			if reference == 1004 {
				if g.Reference.ObjectIndex == priorRef || actorProjectionValue(t, state.Document.Objects[g.Reference.ObjectIndex-1], "Identity") != 1004 {
					t.Fatal("actor reference was not atomically reindexed", priorRef, g)
				}
			} else if g.Reference.ObjectIndex != 0 || g.Reference.Key != 0xdeadbeef {
				t.Fatal("missing source lookup became an object", g)
			}
			encoded, err := sav.EncodeDocumentData(*state.Document)
			if err != nil {
				t.Fatal(err)
			}
			file, err := sav.Open(encoded)
			if err != nil {
				t.Fatal(err)
			}
			graph, err := file.ActorGraph()
			if err != nil {
				t.Fatal(err)
			}
			got := graph.Groups[0]
			if !got.Owner.Resolved || got.Owner.Class != "Player" || got.Owner.Key != 0xface1234 || got.Owner.PlayerSlot != 1 {
				t.Fatal("serialized Player reference changed", got.Owner)
			}
			if reference == 1004 {
				if !got.Reference.Resolved || got.Reference.Key != 1004 {
					t.Fatal("resolved current actor reference changed", got.Reference)
				}
			} else if got.Reference.Resolved || got.Reference.Key != 0 {
				t.Fatal("unresolved source address replayed", got.Reference)
			}
			// A changed native reference is a coverage gap, not a partially
			// published graph or a failure of ordinary native persistence.
			baseline, _ := cloneSavedDocument(state)
			groups[0].Reference.Key++
			if err := mission.World.ImportSavedGroups(groups, orders); err != nil {
				t.Fatal(err)
			}
			if err := projectSavedGroups(state, mission.World); err != nil || state.GroupBindings.Unavailable == "" {
				t.Fatal("changed reference not disclosed", err)
			}
			if !reflect.DeepEqual(state.Document, baseline.Document) || !reflect.DeepEqual(state.Actors, baseline.Actors) || !reflect.DeepEqual(state.GroupBindings.Groups, baseline.GroupBindings.Groups) {
				t.Fatal("coverage failure published partial reference changes")
			}
		})
	}
}
