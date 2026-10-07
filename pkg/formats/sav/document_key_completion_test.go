package sav

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestCompleteDocumentKeysPreservesCurrentValuesAndOrdinaryEdits(t *testing.T) {
	d := documentDataFixture(t)
	for i := range d.Objects {
		for j := range d.Objects[i].Values {
			v := &d.Objects[i].Values[j]
			if v.Name == "Identity" || v.Name == "This" {
				v.Value = 0x71000000 + uint32(i)
			}
		}
	}
	actor := documentRecordByClass(t, &d, "Human")
	player := documentRecordByClass(t, &d, "Player")
	*documentValue1115(t, actor, "Identity") = 0x12345678
	*documentValue1115(t, player, "Hero") = 0x12345678
	*documentValue1115(t, actor, "Reference") = *documentValue1115(t, player, "This")
	*documentValue1115(t, actor, "RuntimeID") = 0x12345678 // A separate namespace may overlap.
	d.World.Cells[0].GroundActor = 0x12345678
	d.World.Cells[0].Layers[0] = 0x01000000 // An unresolved reference remains unresolved.
	for i := range actor.Raw {
		if actor.Raw[i].Name == "U158" {
			binary.LittleEndian.PutUint32(actor.Raw[i].Bytes[0xc:], 0x12345678)
			binary.LittleEndian.PutUint32(actor.Raw[i].Bytes[0x10:], 0x87654321)
		}
	}
	want, err := CloneDocumentData(d)
	if err != nil {
		t.Fatal(err)
	}
	for cycle := 0; cycle < 2; cycle++ {
		got, err := CompleteDocumentKeys(d)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatal("key completion changed existing identity, reference or runtime value", cycle, err)
		}
		raw, err := EncodeDocumentData(got)
		if err != nil {
			t.Fatal(err)
		}
		d, err = DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestCompleteDocumentKeysAllocatesOnlyAbsentObjects(t *testing.T) {
	d := documentDataFixture(t)
	var missing []int
	for i := range d.Objects {
		for j := range d.Objects[i].Values {
			v := &d.Objects[i].Values[j]
			if v.Name != "Identity" && v.Name != "This" {
				continue
			}
			v.Value = 0x71000000 + uint32(i)
			if len(missing) < 2 {
				v.Value = 0
				missing = append(missing, i)
			}
		}
	}
	if len(missing) != 2 {
		t.Fatal("fixture has fewer than two distinct keyed objects")
	}
	actor := documentRecordByClass(t, &d, "Human")
	*documentValue1115(t, actor, "Reference") = 0
	for i := range actor.Raw {
		if actor.Raw[i].Name == "U158" {
			binary.LittleEndian.PutUint32(actor.Raw[i].Bytes[0xc:], 0x01000000)
		}
	}
	d.World.Cells[0].Layers[0] = 0x01000001
	before, _ := documentDataGobCopy(t, d)
	want, err := CloneDocumentData(d)
	if err != nil {
		t.Fatal(err)
	}
	got, err := CompleteDocumentKeys(d)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[uint32]bool{d.World.TerrainIdentity: true, 0x01000000: true, 0x01000001: true}
	for i := range got.Objects {
		for j := range got.Objects[i].Values {
			v := &got.Objects[i].Values[j]
			if v.Name != "Identity" && v.Name != "This" {
				continue
			}
			if v.Value == 0 || keys[v.Value] {
				t.Fatal("allocated key is null, repeated or captures an unresolved reference", v)
			}
			keys[v.Value] = true
			original := want.Objects[i].Values[j].Value
			if original == 0 {
				want.Objects[i].Values[j].Value = v.Value
			} else if original != v.Value {
				t.Fatal("allocation moved an existing current key")
			}
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("key completion rewrote a reference or unrelated field")
	}
	after, _ := documentDataGobCopy(t, d)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("key completion mutated its caller")
	}
}

func TestCompleteDocumentKeysScopesIdentityBySAVClass(t *testing.T) {
	const key = uint32(0x62000020)
	d := documentDataFixture(t)
	var first, second = -1, -1
	for i := range d.Objects {
		found := false
		for j := range d.Objects[i].Values {
			v := &d.Objects[i].Values[j]
			if v.Name != "Identity" && v.Name != "This" {
				continue
			}
			if found {
				t.Fatalf("fixture object %d has multiple identity fields", i+1)
			}
			found = true
			v.Value = 0x71000000 + uint32(i)
		}
		if !found {
			continue
		}
		if first < 0 {
			first = i
		} else if second < 0 && d.Objects[i].Class != d.Objects[first].Class {
			second = i
		}
	}
	if first < 0 || second < 0 || d.World == nil {
		t.Fatal("fixture lacks distinct typed identities and terrain")
	}
	for i := range d.Objects {
		for j := range d.Objects[i].Values {
			v := &d.Objects[i].Values[j]
			if (v.Name == "Identity" || v.Name == "This") && (i == first || i == second) {
				v.Value = key
			}
		}
	}
	d.World.TerrainIdentity = key
	got, err := CompleteDocumentKeys(d)
	if err != nil {
		t.Fatal("distinct SAV classes and terrain role may reuse a numeric key", err)
	}
	for _, object := range []int{first, second} {
		value := uint32(0)
		for _, field := range got.Objects[object].Values {
			if field.Name == "Identity" || field.Name == "This" {
				value = field.Value
				break
			}
		}
		if value != key {
			t.Fatal("typed identity was rewritten", object, value)
		}
	}
	if got.World.TerrainIdentity != key {
		t.Fatal("terrain identity was rewritten", got.World.TerrainIdentity)
	}
}

func TestCompleteDocumentKeysRejectsSameClassCollisionsAtomically(t *testing.T) {
	for _, which := range []string{"objects", "missing terrain"} {
		d := documentDataFixture(t)
		for i := range d.Objects {
			for j := range d.Objects[i].Values {
				v := &d.Objects[i].Values[j]
				if v.Name == "Identity" || v.Name == "This" {
					v.Value = 0x71000000 + uint32(i)
				}
			}
		}
		switch which {
		case "objects":
			d.Objects = append(d.Objects, d.Objects[0])
		case "missing terrain":
			d.World.TerrainIdentity = 0
		}
		before, _ := documentDataGobCopy(t, d)
		if _, err := CompleteDocumentKeys(d); err == nil {
			t.Fatal("ambiguous completed key graph accepted", which)
		}
		after, _ := documentDataGobCopy(t, d)
		if !reflect.DeepEqual(before, after) {
			t.Fatal("invalid key graph changed caller", which)
		}
	}
}
