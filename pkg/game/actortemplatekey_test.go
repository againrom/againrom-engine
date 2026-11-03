package game

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestSecondActorTemplateKeysWriteLoadAndPreserveOtherBytes(t *testing.T) {
	raw := secondTownSave(t)
	engine, err := sav.EngineDocument(raw)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := sav.DecodeDocumentData(engine)
	if err != nil {
		t.Fatal(err)
	}
	units, humans := make(dbCollection, 2), make(dbCollection, 3)
	units[1] = dbEntry{name: "creature", params: make([]int32, 56)}
	units[1].params[55] = 623
	humans[1] = dbEntry{name: "person", params: make([]int32, 25)}
	humans[1].params[24] = 2000
	humans[2] = dbEntry{name: "party person", params: make([]int32, 25)}
	humans[2].params[24] = 10220
	table := &mapload.Table{Units: units, Humans: humans}
	marked, unmarked := true, false
	for _, tc := range []struct {
		name, class  string
		before, want uint32
		row          uint32
		mark         *bool
	}{
		{"new unit", "Unit", 0, 623, 1, nil},
		{"unit flags", "Unit", 0x01010000, 0x0101026f, 1, nil},
		{"loaded unit", "Unit", 0x010104d2, 0x010104d2, 1, nil},
		{"new human", "Human", 0, 2000, 1, nil},
		{"loaded human", "Human", 0x010107d1, 0x010107d1, 1, nil},
		{"normalized human", "Human", 0, 22, 2, nil},
		{"missing unit row", "Unit", 0x01010000, 0x01010000, 3, nil},
		{"missing human row", "Human", 0, 0, 3, nil},
		{"current marked", "Unit", 0, 0x0100026f, 1, &marked},
		{"current unmarked", "Unit", 0x0101004d, 0x0001004d, 1, &unmarked},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := sav.CloneDocumentData(seed)
			if err != nil {
				t.Fatal(err)
			}
			r := mustNewRecord(tc.class)
			mustSetValue(&r, "Identity", 0x1234)
			mustSetValue(&r, "T0C", tc.row)
			mustSetValue(&r, "U148", tc.before)
			doc.Objects = append(doc.Objects, r)
			index := uint16(len(doc.Objects))
			player := &doc.Objects[doc.Players[0]-1]
			refs, _ := savedObjectRefs(&player.Groups[0], "Actors")
			savedObjectSetRefs(&player.Groups[0], "Actors", append(slices.Clone(refs), index), true)
			mustSetCount(player, "Actors", uint32(len(refs)+1))
			if tc.mark != nil {
				a := currentActionData{Version: 1, Bindings: []currentActionBinding{{ID: 0x1234, Object: index}},
					Values: map[sim.EntityID]sim.ActorValues{0x1234: {SuppressRewards: *tc.mark}}}
				b, err := json.Marshal(a)
				if err != nil {
					t.Fatal(err)
				}
				if err := sav.SetNativeActions(&doc.State, b); err != nil {
					t.Fatal(err)
				}
			}
			form := savedForm{records: sav.SecondGame, holders: secondFormHolders, leaf: true}
			baseline := form
			baseline.holders = slices.DeleteFunc(slices.Clone(form.holders), func(h savedFormHolder) bool { return h.field == "U148" })
			s := Snapshot{second: &currentSecondCampaign{}}
			control, _, err := baseline.write(&FrontEnd{InstallResources: InstallResources{Table: table}}, s, doc)
			if err != nil {
				t.Fatal(err)
			}
			out, _, err := form.write(&FrontEnd{InstallResources: InstallResources{Table: table}}, s, doc)
			if err != nil {
				t.Fatal(err)
			}
			out.SecondTail, control.SecondTail = seed.SecondTail, seed.SecondTail
			if out.SecondTail == nil {
				out.SecondTail, control.SecondTail = &sav.SecondTail{}, &sav.SecondTail{}
			}
			index = uint16(1 + slices.IndexFunc(out.Objects, func(r sav.DocumentRecordData) bool {
				v, _ := savedStructureValue(&r, "Identity")
				return v == 0x1234
			}))
			if index == 0 {
				t.Fatal("the formed record is absent")
			}
			got, _ := savedStructureValue(&out.Objects[index-1], "U148")
			if got != tc.want {
				t.Errorf("packed state %#08x, want %#08x", got, tc.want)
			}
			encoded, err := sav.EncodeDocumentData(out)
			if err != nil {
				t.Fatal(err)
			}
			loaded, err := sav.DecodeDocumentData(encoded)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, actor := range loaded.Objects {
				row, _ := savedStructureValue(&actor, "T0C")
				if actor.Class == tc.class && row == tc.row {
					key, _ := savedStructureValue(&actor, "U148")
					if key == tc.want {
						found = true
					}
				}
			}
			if !found {
				t.Errorf("LOAD did not restore packed state %#08x", tc.want)
			}
			mustSetValue(&out.Objects[index-1], "U148", tc.before)
			gotBytes, err := sav.EncodeDocumentData(out)
			if err != nil {
				t.Fatal(err)
			}
			wantBytes, err := sav.EncodeDocumentData(control)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(gotBytes, wantBytes) {
				t.Fatal("bytes outside the packed state changed")
			}
		})
	}
}

func TestSecondActorRewardFlagLoadsFromTheRecord(t *testing.T) {
	h := secondMissionFixtureHero()
	h.identity = 0x01000015
	f, _, _ := secondMissionLoadFile(t, secondMissionFile(h, secondMissionStore(4, 0, -1)))
	if !secondMissionActor(t, f.live.world).SuppressRewards {
		t.Fatal("LOAD did not restore the saved reward flag")
	}
}
