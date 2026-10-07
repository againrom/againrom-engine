package game

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func TestReleaseGeneratedRuntimeIDsFitCommandMembers(t *testing.T) {
	f := releaseFront(t)
	party := f.ChargenParty(ui.ChargenResult{Name: "Runtime witness", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("runtime witness")
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	path := generatedMissionSave(t, f, app)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil || doc.World == nil || doc.Head.Mission != 20 {
		t.Fatal("ordinary SAVE did not produce mission 20", err)
	}
	counts, wide := map[string]int{}, map[string]int{}
	seen := map[uint32]bool{}
	for _, r := range doc.Objects {
		switch r.Class {
		case "Human", "Unit", "Building", "Sack":
		default:
			continue
		}
		id, err := savedStructureValue(&r, "RuntimeID")
		if err != nil || id == 0 || seen[id] {
			t.Fatal("new placeable has absent or duplicate runtime identity", r.Class, id, err)
		}
		seen[id] = true
		counts[r.Class]++
		if id > 0xffff {
			wide[r.Class]++
		}
	}
	t.Logf("placeables=%v wide=%v", counts, wide)
	if counts["Human"] == 0 || counts["Unit"] == 0 || counts["Building"] == 0 || counts["Sack"] == 0 {
		t.Fatal("fixture must exercise actors, buildings and sacks")
	}
	if len(wide) != 0 {
		t.Fatal("new placeable runtime identities do not fit command member words", wide)
	}
}

func TestReleaseCityGraftAllocatesFreeRuntimeID(t *testing.T) {
	f := releaseFront(t)
	party := f.ChargenParty(ui.ChargenResult{Name: "Graft witness", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	data, err := nativeCityDataApproximate(party, nil, f.Table, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	companion, ok := mapload.CampaignNPCMember(f.Table, 22, 30, party)
	if !ok || companion.Carry != nil {
		t.Fatal("fixture lacks a newly granted companion")
	}
	before, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	used := map[uint32]bool{0: true}
	for _, o := range data.Objects {
		var token []byte
		switch {
		case o.Unit != nil:
			token = o.Unit.Token
		case o.Item != nil:
			token = o.Item.Token
		case o.Effect != nil:
			token = o.Effect.Token
		}
		if len(token) >= 16 {
			used[binary.LittleEndian.Uint32(token[12:16])] = true
		}
	}
	want := uint32(1)
	for used[want] {
		want++
	}
	out, err := graftNativeCityMember(data, companion, party[0], f.Table)
	if err != nil {
		t.Fatal(err)
	}
	last := out.Objects[len(out.Objects)-1]
	if last.Unit == nil || last.Class != "Human" {
		t.Fatal("graft did not append the companion")
	}
	id := binary.LittleEndian.Uint32(last.Unit.Token[12:16])
	key := binary.LittleEndian.Uint32(last.Unit.Token[29:33])
	t.Logf("new runtime=%d saved-address=%#x first-free=%d", id, key, want)
	var old sav.CityData
	if err = json.Unmarshal(before, &old); err != nil {
		t.Fatal(err)
	}
	for i, o := range old.Objects {
		if o.Unit != nil && !reflect.DeepEqual(o.Unit, out.Objects[i].Unit) {
			t.Fatal("graft changed an existing actor", i)
		}
	}
	if id != want {
		t.Fatalf("new companion runtime ID=%d, want lowest free=%d independently of saved-address=%#x", id, want, key)
	}
}

func TestGeneratedRuntimeAllocationKeepsLoadedIDsAndAddressKeys(t *testing.T) {
	var objects []sav.DocumentRecordData
	for _, row := range []struct {
		class string
		id    uint32
	}{{"Human", 1}, {"Building", 3}, {"Item", 5}, {"Effect", 0x01000006}, {"Human", 0x21000007}, {"Unit", 0}} {
		r := mustNewRecord(row.class)
		mustSetValue(&r, "RuntimeID", row.id)
		objects = append(objects, r)
	}
	before, err := json.Marshal(objects)
	if err != nil {
		t.Fatal(err)
	}
	keys := []uint32{0x61000000, 0x61000010}
	b := generatedDocumentBuilder{doc: sav.DocumentData{Objects: objects}, reservedKeys: append([]uint32(nil), keys...)}
	for _, want := range []uint32{2, 4, 6, 8} {
		if got := b.runtime(); got != want {
			t.Fatalf("runtime allocation=%d, want %d", got, want)
		}
	}
	if !reflect.DeepEqual(b.reservedKeys, keys) || b.identity() != keys[0] || b.identity() != keys[1] {
		t.Fatal("runtime allocation consumed or changed saved-address keys")
	}
	after, err := json.Marshal(b.doc.Objects)
	if err != nil || string(before) != string(after) {
		t.Fatal("allocation rewrote loaded IDs or other retained values", err)
	}
}
