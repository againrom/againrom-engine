package sav

import (
	"bytes"
	"testing"
)

func TestCityStateCurrentContinuationSurvivesPoolOffsets(t *testing.T) {
	data := NewCityStateData("Name occupies the pool before the continuation")
	doc := DocumentStateData{RootKind: data.RootKind, DirectoryRecords: data.DirectoryRecords, ValueRecords: data.ValueRecords}
	want := []byte(`{"Version":1,"Session":{"Won":[111]}}`)
	if err := SetNativeActions(&doc, want); err != nil {
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
	value := back.values[NativeActionsPath]
	got, present, err := NativeActions(DocumentStateData{ValueRecords: []CityStateRecordData{{Path: NativeActionsPath, Value: CityStateValueData{Kind: value.kind, Bytes: value.bytes}}}})
	if err != nil || !present || !bytes.Equal(got, want) {
		t.Fatal("city continuation changed during serialization", got, present, err)
	}
}
