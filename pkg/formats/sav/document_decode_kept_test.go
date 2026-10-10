package sav

import (
	"reflect"
	"testing"
)

func forgetDecodedDocuments() {
	decodedDocuments.mu.Lock()
	decodedDocuments.entries = nil
	decodedDocuments.mu.Unlock()
}

// A decode answered from the kept copy equals a fresh parse of the bytes, in
// both modes, and a reader's edit reaches no later reader.
func TestKeptDocumentDecodeEqualsAFreshParse(t *testing.T) {
	source := cityTestSource(t)
	forgetDecodedDocuments()
	fresh, err := DecodeDocumentData(source)
	if err != nil {
		t.Fatal(err)
	}
	freshData, freshOrigins, err := DecodeDocumentDataWithOrigins(source)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		kept, err := DecodeDocumentData(source)
		if err != nil || !reflect.DeepEqual(kept, fresh) {
			t.Fatal("kept decode differs from a fresh parse", err)
		}
		keptData, keptOrigins, err := DecodeDocumentDataWithOrigins(source)
		if err != nil || !reflect.DeepEqual(keptData, freshData) || !reflect.DeepEqual(keptOrigins, freshOrigins) {
			t.Fatal("kept decode with origins differs from a fresh parse", err)
		}
		kept.Objects[0].Class = "changed"
		if len(kept.Objects[0].Values) > 0 {
			kept.Objects[0].Values[0].Value ^= 1
		}
		keptOrigins[0].ArchiveIndex ^= 1
	}
	again, err := DecodeDocumentData(source)
	if err != nil || !reflect.DeepEqual(again, fresh) {
		t.Fatal("a reader's edit reached a later reader")
	}
	bad := append([]byte(nil), source...)
	bad[len(bad)/2] ^= 0xff
	forgetDecodedDocuments()
	_, freshErr := DecodeDocumentData(bad)
	_, keptErr := DecodeDocumentData(bad)
	if (freshErr == nil) != (keptErr == nil) {
		t.Fatal("a damaged file decoded differently the second time")
	}
}
