package sav

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestRemintDocumentKeys1170RelationalNamespaces(t *testing.T) {
	d := documentDataFixture(t)
	old := make(map[uint32]int)
	for i := range d.Objects {
		for j := range d.Objects[i].Values {
			v := &d.Objects[i].Values[j]
			if v.Name == "Identity" || v.Name == "This" {
				v.Value = uint32(0x900000 + i)
				old[v.Value] = i
			}
		}
	}
	actor := documentRecordByClass(t, &d, "Human")
	player := documentRecordByClass(t, &d, "Player")
	actorKey := *documentValue1115(t, actor, "Identity")
	playerKey := *documentValue1115(t, player, "This")
	*documentValue1115(t, player, "Hero") = actorKey
	*documentValue1115(t, actor, "Reference") = playerKey
	*documentValue1115(t, actor, "RuntimeID") = 73
	d.World.Cells[0].GroundActor = actorKey
	d.World.Cells[0].Layers[0] = 0x1000000 // retained unresolved key must not become allocated
	for i := range actor.Raw {
		r := &actor.Raw[i]
		if r.Name == "Block12" {
			binary.LittleEndian.PutUint32(r.Bytes[8:], d.World.TerrainIdentity)
		}
		if r.Name == "U158" {
			binary.LittleEndian.PutUint32(r.Bytes[0xc:], actorKey)
			binary.LittleEndian.PutUint32(r.Bytes[0x10:], 0x12345678)
		}
	}
	before, _ := documentDataGobCopy(t, d)
	got, err := RemintDocumentKeys(d)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Players, d.Players) || !reflect.DeepEqual(got.DeadActors, d.DeadActors) || len(got.Objects) != len(d.Objects) {
		t.Fatal("remint changed graph/root multiplicity")
	}
	seen := map[uint32]bool{got.World.TerrainIdentity: true}
	for i := range got.Objects {
		for _, v := range got.Objects[i].Values {
			if v.Name != "Identity" && v.Name != "This" {
				continue
			}
			if v.Value == 0 || seen[v.Value] {
				t.Fatal("bad identity", v)
			}
			if _, existed := old[v.Value]; existed {
				t.Fatal("source key reused")
			}
			seen[v.Value] = true
		}
	}
	a := documentRecordByClass(t, &got, "Human")
	p := documentRecordByClass(t, &got, "Player")
	if *documentValue1115(t, p, "Hero") != *documentValue1115(t, a, "Identity") || *documentValue1115(t, a, "Reference") != *documentValue1115(t, p, "This") || got.World.Cells[0].GroundActor != *documentValue1115(t, a, "Identity") || *documentValue1115(t, a, "RuntimeID") != 73 {
		t.Fatal("reference namespaces split")
	}
	if got.World.Cells[0].Layers[0] != 0x1000000 || seen[0x1000000] {
		t.Fatal("unresolved reference became bound")
	}
	for _, r := range a.Raw {
		if r.Name == "Block12" && binary.LittleEndian.Uint32(r.Bytes[8:]) != got.World.TerrainIdentity {
			t.Fatal("terrain reference split")
		}
		if r.Name == "U158" && (binary.LittleEndian.Uint32(r.Bytes[0xc:]) != *documentValue1115(t, a, "Identity") || binary.LittleEndian.Uint32(r.Bytes[0x10:]) != 0x12345678) {
			t.Fatal("qualified order remint lost bound/missing distinction")
		}
	}
	wire, err := EncodeDocumentData(got)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeDocumentData(wire); err != nil {
		t.Fatal(err)
	}
	after, _ := documentDataGobCopy(t, d)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("remint mutated source")
	}
}

func TestRemintDocumentKeys1170DuplicateIsNotValueEquality(t *testing.T) {
	d := documentDataFixture(t)
	var values []*uint32
	for i := range d.Objects {
		for j := range d.Objects[i].Values {
			v := &d.Objects[i].Values[j]
			if v.Name == "Identity" || v.Name == "This" {
				v.Value = uint32(i + 1)
				values = append(values, &v.Value)
			}
		}
	}
	*values[1] = *values[0]
	if _, err := RemintDocumentKeys(d); err == nil {
		t.Fatal("ambiguous same-key objects were merged")
	}
	*values[0], *values[1] = 0, 0
	if _, err := RemintDocumentKeys(d); err != nil {
		t.Fatal("separate new zero-key objects must get independent identities", err)
	}
}

func TestReserveDocumentKeys1170BoundsAndRawExclusion(t *testing.T) {
	d := documentDataFixture(t)
	a := documentRecordByClass(t, &d, "Human")
	for i := range a.Raw {
		if a.Raw[i].Name == "U158" {
			binary.LittleEndian.PutUint32(a.Raw[i].Bytes[0xc:], 0x01000000)
		}
	}
	before, _ := documentDataGobCopy(t, d)
	keys, err := ReserveDocumentKeys(d, 3)
	if err != nil || len(keys) != 3 {
		t.Fatal(err)
	}
	for i, key := range keys {
		if key == 0 || key == 0x01000000 || i > 0 && key <= keys[i-1] {
			t.Fatal("reservation collided or repeated", keys)
		}
	}
	for _, count := range []int{-1, 1<<16 + 1} {
		if _, err := ReserveDocumentKeys(d, count); err == nil {
			t.Fatal("unbounded reservation accepted", count)
		}
	}
	after, _ := documentDataGobCopy(t, d)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("reservation changed its source")
	}
}
