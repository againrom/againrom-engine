package sav

import (
	"bytes"
	"testing"
)

func TestNativeModsFramingIsStrict(t *testing.T) {
	var state DocumentStateData
	if _, ok, err := NativeMods(state); ok || err != nil {
		t.Fatal("absence", ok, err)
	}
	for _, value := range [][]byte{[]byte("x"), []byte("a mod mark"), bytes.Repeat([]byte{7}, 129)} {
		if err := SetNativeMods(&state, value); err != nil {
			t.Fatal(err)
		}
		back, ok, err := NativeMods(state)
		if err != nil || !ok || !bytes.Equal(value, back) || len(state.ValueRecords) != 1 {
			t.Fatal("round trip", err)
		}
	}
	ClearNativeMods(&state)
	if _, ok, _ := NativeMods(state); ok || len(state.ValueRecords) != 0 {
		t.Fatal("clear left the mark")
	}
	for _, which := range []string{"case", "duplicate", "version", "length", "padding", "kind"} {
		var s DocumentStateData
		_ = SetNativeMods(&s, []byte("x"))
		r := &s.ValueRecords[0]
		switch which {
		case "case":
			r.Path = "/CurrentState/againromMods"
		case "duplicate":
			s.ValueRecords = append(s.ValueRecords, *r)
		case "version":
			r.Value.Bytes[0] = 2
		case "length":
			r.Value.Bytes[4] = 255
		case "padding":
			r.Value.Bytes[11] = 1
		case "kind":
			r.Value.Kind = 0
		}
		if _, _, err := NativeMods(s); err == nil {
			t.Fatalf("accepted malformed %s", which)
		}
	}
	if err := SetNativeMods(&state, nil); err == nil {
		t.Fatal("empty mark accepted")
	}
}

func TestModMarkSurvivesCityStateSerializationBesideTheContinuation(t *testing.T) {
	data := NewCityStateData("mark")
	doc := DocumentStateData{RootKind: data.RootKind, DirectoryRecords: data.DirectoryRecords, ValueRecords: data.ValueRecords}
	mark := []byte(`{"format":1}`)
	if err := SetNativeActions(&doc, []byte(`{"Version":1}`)); err != nil {
		t.Fatal(err)
	}
	if err := SetNativeMods(&doc, mark); err != nil {
		t.Fatal(err)
	}
	data.ValueRecords = doc.ValueRecords
	state, err := cityStateFromData(data, CityDataVersion)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := serializeCityState(state)
	if err != nil {
		t.Fatal(err)
	}
	back, err := parseCityState(raw)
	if err != nil {
		t.Fatal(err)
	}
	v := back.values[NativeModsPath]
	got, ok, err := NativeMods(DocumentStateData{ValueRecords: []CityStateRecordData{{Path: NativeModsPath, Value: CityStateValueData{Kind: v.kind, Bytes: v.bytes}}}})
	if err != nil || !ok || !bytes.Equal(got, mark) {
		t.Fatal("the mark changed in the city state", got, ok, err)
	}
}

func TestModMarkSurvivesWorldStateSerializationBesideRngAndContinuation(t *testing.T) {
	dirs := literalWorldDirectories()
	dirs[1].leaves = append(dirs[1].leaves, literalStateLeaf{name: "AgainromRng", kind: 6, data: literalStateWords(1, 3, 4)})
	state, err := parseWorldState(literalStateStore(dirs))
	if err != nil {
		t.Fatal(err)
	}
	mark := []byte("the mark")
	var doc DocumentStateData
	if err := SetNativeMods(&doc, mark); err != nil {
		t.Fatal(err)
	}
	if err := SetNativeActions(&doc, []byte("actions")); err != nil {
		t.Fatal(err)
	}
	for _, r := range doc.ValueRecords {
		state.values[r.Path] = cityStateValue{kind: r.Value.Kind, bytes: r.Value.Bytes}
	}
	raw, err := serializeWorldState(state)
	if err != nil {
		t.Fatal(err)
	}
	back, err := parseWorldState(raw)
	if err != nil {
		t.Fatal(err)
	}
	v := back.values[NativeModsPath]
	got, ok, err := NativeMods(DocumentStateData{ValueRecords: []CityStateRecordData{{Path: NativeModsPath, Value: CityStateValueData{Kind: v.kind, Bytes: v.bytes}}}})
	if err != nil || !ok || !bytes.Equal(got, mark) {
		t.Fatal("the mark changed in the world state", got, ok, err)
	}
	if _, ok := back.values[NativeRandomPath]; !ok {
		t.Fatal("the random state was lost")
	}
}
