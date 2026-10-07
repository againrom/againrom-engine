package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentSpellRangeProjectionKeepsLiveValueAndOrdinaryEdits(t *testing.T) {
	for _, edit := range []string{"unchanged", "range", "mana", "defensive", "id"} {
		t.Run(edit, func(t *testing.T) {
			f := currentArchiveFixture(t)
			ms := f.live.mission.state
			var child sim.SavedSpellObject
			var object uint16
			for _, binding := range ms.savedDocument.Objects.Spells {
				for _, v := range ms.World.SavedObjects().Spells {
					if v.ID == binding.ID && v.Value.Range > 0 {
						child, object = v, binding.ObjectIndex
					}
				}
			}
			if child.ID == 0 || object == 0 {
				t.Fatal("range witness has no bound Spell")
			}
			record := &ms.savedDocument.Document.Objects[object-1]
			mustSetValue(record, "S09", uint32(child.Value.Range-1))
			graph, rows, err := captureCurrentObjects(ms.savedDocument, ms.World)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, row := range rows {
				if row.ID == child.ID {
					found = row.Range != nil && row.Range.Value == child.Value.Range
				}
			}
			if !found {
				t.Fatal("projected Range lost its live operand")
			}
			switch edit {
			case "range":
				mustSetValue(record, "S09", 73)
			case "mana":
				mustSetValue(record, "S0C", 54321)
			case "defensive":
				mustSetValue(record, "S0A", 0x83)
			case "id":
				mustSetValue(record, "S08", 2)
			}
			want, err := savedSpellRecord(record)
			if err != nil {
				t.Fatal(err)
			}
			if edit == "unchanged" {
				want.Value.Range = child.Value.Range
			}
			_, err = restoreCurrentObjects(ms, graph, rows, f.Table)
			if err != nil {
				t.Fatal(err)
			}
			matched := false
			for _, v := range ms.World.SavedObjects().Spells {
				if v.ID == child.ID && v.Value != want.Value {
					t.Fatal("range operand replaced ordinary values", v.Value, want.Value)
				}
				matched = matched || v.ID == child.ID
			}
			if !matched {
				t.Fatal("range restoration dropped the Spell")
			}
			if edit == "unchanged" {
				for _, fault := range []string{"kind", "unbound", "null id", "absent wire", "unknown spell", "redundant"} {
					bad := slices.Clone(rows)
					for i := range bad {
						row := &bad[i]
						if row.ID != child.ID {
							continue
						}
						operand := *row.Range
						row.Range = &operand
						switch fault {
						case "kind":
							row.Kind = 2
						case "unbound":
							row.Object = 0
						case "null id":
							row.ID = 0
						case "absent wire":
							operand.Wire.Present = false
						case "unknown spell":
							operand.Wire.ID = 29
						case "redundant":
							operand.Value = operand.Wire.Range
						}
					}
					before := ms.World.Hash()
					if _, err := restoreCurrentObjects(ms, graph, bad, f.Table); err == nil || ms.World.Hash() != before {
						t.Fatal("malformed range operand admitted or changed World", fault, err)
					}
				}
			}
		})
	}
}

func TestCurrentObjectIdentityAbsenceAllowsOrdinaryKeyEdit(t *testing.T) {
	absent, present := false, true
	anchor, changed := uint32(123), uint32(456)
	for _, test := range []struct {
		name     string
		row      currentOwnedObject
		legacy   bool
		ordinary uint32
		want     uint32
	}{
		{"absent", currentOwnedObject{Kind: 1, Object: 1, IdentityPresent: &absent, IdentityAnchor: &anchor}, false, anchor, 0},
		{"edited", currentOwnedObject{Kind: 1, Object: 1, IdentityPresent: &absent, IdentityAnchor: &anchor}, false, changed, changed},
		{"sack", currentOwnedObject{Kind: 4, Object: 1, IdentityPresent: &absent, IdentityAnchor: &anchor}, false, anchor, 0},
		{"sack edited", currentOwnedObject{Kind: 4, Object: 1, IdentityPresent: &absent, IdentityAnchor: &anchor}, false, changed, changed},
		{"present", currentOwnedObject{Kind: 1, Object: 1, IdentityPresent: &present}, false, anchor, anchor},
		{"private", currentOwnedObject{Kind: 1, IdentityPresent: &absent}, false, 0, 0},
		{"legacy", currentOwnedObject{Kind: 1, Object: 1, IdentityPresent: &absent}, true, anchor, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateCurrentObjectIdentityAbsence(test.row, test.legacy); err != nil {
				t.Fatal(err)
			}
			if got := restoreCurrentObjectIdentity(test.row, test.ordinary); got != test.want {
				t.Fatal("ordinary key did not control presence", got, test.want)
			}
		})
	}
}

func TestCurrentObjectIdentityAbsenceRejectsMalformedPolicyAtomically(t *testing.T) {
	f := itemObjectsOpen(t, false)
	ms := f.live.mission.state
	g, rows, err := captureCurrentObjects(ms.savedDocument, ms.World)
	if err != nil {
		t.Fatal(err)
	}
	chosen := -1
	for i, row := range rows {
		if row.Kind == 1 && row.Object != 0 {
			chosen = i
			break
		}
	}
	if chosen < 0 {
		t.Fatal("fixture has no ordinary Item")
	}
	before := ms.World.Hash()
	metadata := ms.savedDocument.Objects
	absent, present := false, true
	anchor, zero := uint32(123), uint32(0)
	for _, mutate := range []func(*currentOwnedObject){
		func(r *currentOwnedObject) { r.IdentityPresent, r.IdentityAnchor = &absent, nil },
		func(r *currentOwnedObject) { r.IdentityPresent, r.IdentityAnchor = &absent, &zero },
		func(r *currentOwnedObject) { r.IdentityPresent, r.IdentityAnchor = &present, &anchor },
		func(r *currentOwnedObject) { r.IdentityPresent, r.IdentityAnchor = nil, &anchor },
		func(r *currentOwnedObject) { r.Kind, r.IdentityPresent, r.IdentityAnchor = 5, &absent, &anchor },
	} {
		bad := slices.Clone(rows)
		mutate(&bad[chosen])
		if _, err := restoreCurrentObjects(ms, g, bad, f.Table); err == nil {
			t.Fatal("malformed absence was accepted")
		}
		if ms.World.Hash() != before || !reflect.DeepEqual(metadata, ms.savedDocument.Objects) {
			t.Fatal("malformed absence partially changed current state")
		}
	}
}

func TestCurrentChildIdentityAbsenceAndOrdinaryEdits(t *testing.T) {
	for _, kind := range []uint8{2, 3} {
		f := currentArchiveFixture(t)
		ms := f.live.mission.state
		graph, rows, err := captureCurrentObjects(ms.savedDocument, ms.World)
		if err != nil {
			t.Fatal(err)
		}
		var chosen currentOwnedObject
		for i := range rows {
			row := &rows[i]
			if row.Kind != kind || row.Object == 0 {
				continue
			}
			field := "Identity"
			if kind == 3 {
				field = "This"
			}
			anchor, err := savedStructureValue(&ms.savedDocument.Document.Objects[row.Object-1], field)
			if err != nil {
				t.Fatal(err)
			}
			absent := false
			row.IdentityPresent, row.IdentityAnchor = &absent, &anchor
			chosen = *row
			break
		}
		if chosen.ID == 0 {
			t.Fatal("missing child")
		}
		if _, err := restoreCurrentObjects(ms, graph, rows, f.Table); err != nil {
			t.Fatal(err)
		}
		before := f.live.world.Hash()
		for cycle := 0; cycle < 2; cycle++ {
			snapshot, label, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := f.ExportCurrentSave(snapshot, label)
			if err != nil {
				t.Fatal(err)
			}
			cold := openCurrentArchive(t, raw)
			if cold.live.world.Hash() != before {
				t.Fatal("child acquired ordinary transport identity", kind, cycle)
			}
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			actions, err := readCurrentActions(&doc)
			if err != nil {
				t.Fatal(err)
			}
			for _, row := range actions.Ownership {
				if row.ID == chosen.ID {
					field := "Identity"
					if kind == 3 {
						field = "This"
					}
					mustSetValue(&doc.Objects[row.Object-1], field, 0x76543210)
				}
			}
			edited, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			changed := openCurrentArchive(t, edited).live.world.SavedObjects()
			expected := cold.live.world.SavedObjects()
			if kind == 2 {
				for i := range expected.Effects {
					if expected.Effects[i].ID == chosen.ID {
						expected.Effects[i].Token.Identity = 0x76543210
					}
				}
			} else {
				for i := range expected.Spells {
					if expected.Spells[i].ID == chosen.ID {
						expected.Spells[i].This = 0x76543210
					}
				}
			}
			if !reflect.DeepEqual(expected, changed) {
				t.Fatal("ordinary child key edit was overwritten or changed another value", kind, cycle)
			}
			f = cold
		}
	}
}
