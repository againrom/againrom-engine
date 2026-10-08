package game

import (
	"encoding/binary"
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Distinct equal-valued Unit objects, repeated archive references, different
// CStrings and all three classes. Literal writer positions below are independent
// of the structural locator. Zero/duplicate Token identities are legal document
// values here, not a claim that the live registry admits them.
func unit1156Literal(t *testing.T, identity uint32) ([]byte, unitScalarSet, *SnapshotSAVDocument) {
	t.Helper()
	a := &poolFixtureActor{name: "A"}
	b := &poolFixtureActor{name: "A"}
	c := &poolFixtureActor{name: "Human with a longer CString", human: true}
	d := &poolFixtureActor{name: "Z", humanoid: true}
	body := poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{a, b, a, c, nil, d, b}}}}, nil)
	for _, actor := range []*poolFixtureActor{a, b, c, d} {
		// No effects, routes or held references in this literal fixture:
		// Token37 + counts8 + raw462 + patrol count2 = control at509;
		// control19 + two nil archive tags4 = CString at532.
		off, control, state := actor.off, actor.off+509, actor.off+532
		for i := range 37 {
			body[off+i] = byte(0x21 + i)
		}
		binary.LittleEndian.PutUint32(body[off+29:], identity)
		for i := range 19 {
			body[control+i] = byte(0x51 + i)
		}
		if int(body[state]) != len(actor.name) {
			t.Fatal("literal CString writer position changed")
		}
		for i := range 55 {
			body[state+1+len(actor.name)+i] = byte(0x71 + i)
		}
	}
	f := poolFixtureFront(t)
	f.Campaign = resolved(saveCampaign(), nil)
	raw := completeDocumentTail1115(t, f, savedContainer(body))
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := unitScalarExpected(source, raw)
	if err != nil {
		t.Fatal(err)
	}
	state, _ := decodeSavedDocument(raw)
	if differences := want.documentDifferences(state); len(differences) != 0 {
		t.Fatal("positive literal baseline", differences)
	}
	if len(want.records) != 4 || want.records[0].values["Body"] != 0x7271 || want.records[0].values["ManaRegen"] != 0x8c8b || want.records[0].values["T18"] != 0x3938 || want.records[0].values["U148"] != 0xa3a2a1a0 || want.records[0].name != "A" || want.records[2].name != c.name {
		t.Fatal("literal independent scalar anchors changed", want.records)
	}
	return raw, want, state
}

func cloneSavedDocumentFixture(t *testing.T, state *SnapshotSAVDocument) *SnapshotSAVDocument {
	t.Helper()
	copy, err := cloneSavedDocument(state)
	if err != nil {
		t.Fatal(err)
	}
	return copy
}

func TestUnit1156DocumentControls(t *testing.T) {
	for _, identity := range []uint32{0, 0x12345678} {
		t.Run(fmt.Sprintf("identity-%x", identity), func(t *testing.T) {
			_, want, state := unit1156Literal(t, identity)
			index := want.origins[want.records[0].archive] - 1
			for _, name := range spell1152Keys(want.records[0].values) {
				t.Run("omit-"+name, func(t *testing.T) {
					bad := cloneSavedDocumentFixture(t, state)
					values := bad.Document.Objects[index].Values
					for i, field := range values {
						if field.Name == name {
							bad.Document.Objects[index].Values = append(values[:i], values[i+1:]...)
							break
						}
					}
					if len(want.documentDifferences(bad)) == 0 {
						t.Fatal("oracle accepted a missing scalar", name)
					}
				})
			}
			for _, name := range spell1152Keys(want.records[0].raw) {
				for byteIndex := range want.records[0].raw[name] {
					t.Run(fmt.Sprintf("raw-%s-%d", name, byteIndex), func(t *testing.T) {
						bad := cloneSavedDocumentFixture(t, state)
						for i, field := range bad.Document.Objects[index].Raw {
							if field.Name == name {
								bad.Document.Objects[index].Raw[i].Bytes[byteIndex] ^= 1
							}
						}
						if len(want.documentDifferences(bad)) == 0 {
							t.Fatal("oracle accepted a wrong raw byte")
						}
					})
				}
			}
			for _, control := range []string{"wrong-scalar", "duplicate-scalar", "cstring", "class", "missing-document", "collapsed-equal-actors"} {
				t.Run(control, func(t *testing.T) {
					bad := cloneSavedDocumentFixture(t, state)
					changed := want
					switch control {
					case "wrong-scalar":
						for i, field := range bad.Document.Objects[index].Values {
							if field.Name == "Identity" {
								bad.Document.Objects[index].Values[i].Value ^= 1
							}
						}
					case "duplicate-scalar":
						bad.Document.Objects[index].Values = append(bad.Document.Objects[index].Values, sav.DocumentValueData{Name: "T18", Value: want.records[0].values["T18"]})
					case "cstring":
						bad.Document.Objects[index].Texts[0].Value = "different"
					case "class":
						bad.Document.Objects[index].Class = "Human"
					case "missing-document":
						bad.Document = nil
					case "collapsed-equal-actors":
						changed.origins = map[uint16]uint16{}
						for archive, object := range want.origins {
							changed.origins[archive] = object
						}
						changed.origins[want.records[1].archive] = want.origins[want.records[0].archive]
					}
					if len(changed.documentDifferences(bad)) == 0 {
						t.Fatal("oracle accepted control", control)
					}
				})
			}
		})
	}
}

func TestUnit1156LocatorControls(t *testing.T) {
	raw, _, _ := unit1156Literal(t, 0)
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	locations, _ := f.DocumentActorLocations()
	objects, _ := f.DocumentObjectLocations()
	_, origins, _ := sav.DecodeDocumentDataWithOrigins(raw)
	for _, control := range []string{"missing-locator", "duplicate-locator", "wrong-class", "wrong-token", "negative-control", "control-beyond-end", "overlap-state", "truncated-state", "missing-origin", "collapsed-origin", "duplicate-origin", "wrong-origin"} {
		t.Run(control, func(t *testing.T) {
			locs, ids := slices.Clone(locations), slices.Clone(origins)
			switch control {
			case "missing-locator":
				locs = locs[1:]
			case "duplicate-locator":
				locs = append(locs, locs[0])
			case "wrong-class":
				locs[0].Class = "Human"
			case "wrong-token":
				locs[0].Off++
			case "negative-control":
				locs[0].ControlOff = -1
			case "control-beyond-end":
				locs[0].ControlOff = len(f.Body)
			case "overlap-state":
				locs[0].StateOff = locs[0].ControlOff
			case "truncated-state":
				locs[0].StateOff = len(f.Body) - 1
			case "missing-origin":
				ids = ids[1:]
			case "collapsed-origin":
				ids[0].ObjectIndex = ids[1].ObjectIndex
			case "duplicate-origin":
				ids[0].ArchiveIndex = ids[1].ArchiveIndex
			case "wrong-origin":
				ids[0].ArchiveIndex = 0xffff
			}
			if _, err := unit1156Read(f.Body, locs, objects, ids); err == nil {
				t.Fatal("reader accepted invalid structural input", control)
			}
		})
	}
}

func TestUnit1156LiveControls(t *testing.T) {
	raw := unit1156AppFixture(t)
	f := unit1156FixtureFront(t)
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := unitScalarExpected(source, raw)
	if err != nil {
		t.Fatal(err)
	}
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, f.Bodies)
	if err != nil {
		t.Fatal(err)
	}
	unit1156InitialCheck(t, want, ms)
	entities := ms.World.Entities()
	index := -1
	for i, e := range entities {
		if e.SourceBinding.ArchiveIndex == want.records[0].archive {
			index = i
		}
	}
	if index < 0 {
		t.Fatal("fixture has no bound actor")
	}
	for i, name := range unit1156Stats {
		t.Run("basis-"+name, func(t *testing.T) {
			bad := slices.Clone(entities)
			bad[index].ActorLoad.Source.Stats[i] ^= 1
			if differences, _, _ := want.entityDifferences(bad, ms.ActorManifest); len(differences) == 0 {
				t.Fatal("raw oracle accepted wrong World stat word", name)
			}
		})
	}
	for _, control := range []struct {
		name string
		edit func(*sim.Entity)
	}{
		{"sight-low-byte", func(e *sim.Entity) { e.ActorLoad.Source.Sight ^= 1 }},
		{"mana-floor", func(e *sim.Entity) { e.ActorLoad.Source.ManaFloor ^= 1 }},
		{"experience", func(e *sim.Entity) { e.ActorLoad.Source.Experience ^= 1 }},
		{"health-remainder", func(e *sim.Entity) { e.HealthHundredths ^= 1 }},
		{"mana-remainder", func(e *sim.Entity) { e.ManaHundredths ^= 1 }},
		{"display-backing", func(e *sim.Entity) { e.SourceBinding.DisplayBacking ^= 1 }},
		{"stage", func(e *sim.Entity) { e.Decay = sim.DecayFallen }},
		{"reach", func(e *sim.Entity) { e.Reach ^= 1 }},
		{"attack-charge", func(e *sim.Entity) { e.AttackCharge ^= 1 }},
		{"attack-relax", func(e *sim.Entity) { e.AttackRelax ^= 1 }},
		{"domain", func(e *sim.Entity) { e.Domain = sim.DomainAir }},
		{"identity", func(e *sim.Entity) { e.SourceBinding.Identity ^= 1 }},
		{"cell", func(e *sim.Entity) { e.X++ }},
		{"token-footprint", func(e *sim.Entity) { e.TokenSize++ }},
	} {
		t.Run(control.name, func(t *testing.T) {
			bad := slices.Clone(entities)
			control.edit(&bad[index])
			if differences, _, _ := want.entityDifferences(bad, ms.ActorManifest); len(differences) == 0 {
				t.Fatal("raw oracle accepted wrong World counterpart", control.name)
			}
		})
	}
}
