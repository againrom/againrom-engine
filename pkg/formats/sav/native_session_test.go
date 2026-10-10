package sav

import "testing"

func TestNativeSessionFramingIsStrict(t *testing.T) {
	var state DocumentStateData
	if _, ok, err := ReadNativeSession(state); ok || err != nil {
		t.Fatal("absence", ok, err)
	}
	want := NativeSession{Seed: 0x0123456789abcdef, Mode: 1, Shared: 0xdeadbeef}
	if err := SetNativeSession(&state, want); err != nil {
		t.Fatal(err)
	}
	got, ok, err := ReadNativeSession(state)
	if err != nil || !ok || got != want || len(state.ValueRecords) != 1 {
		t.Fatal("round trip", got, ok, err)
	}
	for _, which := range []string{"case", "duplicate", "version", "mode", "kind", "length"} {
		var s DocumentStateData
		_ = SetNativeSession(&s, want)
		r := &s.ValueRecords[0]
		switch which {
		case "case":
			r.Path = "/CurrentState/againromSeed"
		case "duplicate":
			s.ValueRecords = append(s.ValueRecords, *r)
		case "version":
			r.Value.Bytes[0] = 2
		case "mode":
			r.Value.Bytes[12] = 2
		case "kind":
			r.Value.Kind = 0
		case "length":
			r.Value.Bytes = r.Value.Bytes[:16]
		}
		if _, _, err := ReadNativeSession(s); err == nil {
			t.Fatalf("accepted malformed %s", which)
		}
	}
	if err := SetNativeSession(&state, NativeSession{Mode: 2}); err == nil {
		t.Fatal("undefined mode accepted")
	}
}

func TestSessionLeafSurvivesCityStateSerializationBesideTheOthers(t *testing.T) {
	data := NewCityStateData("session")
	doc := DocumentStateData{RootKind: data.RootKind, DirectoryRecords: data.DirectoryRecords, ValueRecords: data.ValueRecords}
	want := NativeSession{Seed: 7, Shared: 9}
	if err := SetNativeActions(&doc, []byte(`{"Version":1}`)); err != nil {
		t.Fatal(err)
	}
	if err := SetNativeMods(&doc, []byte(`{"format":1}`)); err != nil {
		t.Fatal(err)
	}
	if err := SetNativeSession(&doc, want); err != nil {
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
	v := back.values[NativeSessionPath]
	got, ok, err := ReadNativeSession(DocumentStateData{ValueRecords: []CityStateRecordData{{Path: NativeSessionPath, Value: CityStateValueData{Kind: v.kind, Bytes: v.bytes}}}})
	if err != nil || !ok || got != want {
		t.Fatal("the session changed in the city state", got, ok, err)
	}
}
