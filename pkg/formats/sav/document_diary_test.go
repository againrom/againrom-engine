package sav

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestReadDocumentDiaryCurrentIndependentPairs(t *testing.T) {
	r, err := NewDocumentRecord("Diary")
	if err != nil {
		t.Fatal(err)
	}
	want := Diary{Length: 3, Entries: []DiaryEntry{{Index: 1, Count: 19, Remaining: 701}}}
	next, err := ProjectDocumentDiary(r, want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadDocumentDiary(next)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("%+v / %v", got, err)
	}
	for _, raw := range next.Raw {
		if raw.Name == "Journal" && (len(raw.Bytes) != 12 || binary.LittleEndian.Uint32(raw.Bytes[4:]) != 19) {
			t.Fatal("current dword missing")
		}
		if raw.Name == "JournalWords" && (len(raw.Bytes) != 6 || binary.LittleEndian.Uint16(raw.Bytes[2:]) != 701) {
			t.Fatal("independent current word missing")
		}
	}
	for _, count := range r.Counts {
		if count.Count != 0 {
			t.Fatal("projection mutated its source")
		}
	}
	next.Counts[0].Count++
	if _, err = ReadDocumentDiary(next); err == nil {
		t.Fatal("mismatched array count admitted")
	}
}
