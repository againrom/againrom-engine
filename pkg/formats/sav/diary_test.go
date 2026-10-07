package sav

import (
	"bytes"
	"testing"
)

// diaryRecordFixture builds a *Record as the walker would have left it for a
// Diary of n dword/word elements, without going through a byte stream: the
// unit tests below are about diaryFromRecord's own decode and SetDiary's own
// patch, not about the walker that feeds them (program_test.go's own
// TestThePlayerCarriesItsOwnDiary and TestTheWalkReachesTheParticipantsOwnCharacters
// cover that reachability end to end).
func diaryRecordFixture(dwords []uint32, words []uint16, self uint32) *Record {
	r := newRecord("Diary", 0, 0)
	r.Counts["Journal"], r.Counts["JournalWords"] = len(dwords), len(words)
	db := make([]byte, 4*len(dwords))
	for i, v := range dwords {
		put32(db, 4*i, v)
	}
	wb := make([]byte, 2*len(words))
	for i, v := range words {
		put16(wb, 2*i, v)
	}
	r.Raw["Journal"], r.Raw["JournalWords"] = db, wb
	r.Value["D2C"] = self
	return r
}

// TestDiaryFromRecordSkipsConstructionDefaultsAndKeepsTheRest is SAV-667's own
// zero-fill (dword 0, word 1024) treated as "never touched": an element at
// that exact pair is not an entry, and every other element is, whatever its
// value, since only the construction default is indistinguishable from an
// untouched slot.
func TestDiaryFromRecordSkipsConstructionDefaultsAndKeepsTheRest(t *testing.T) {
	r := diaryRecordFixture(
		[]uint32{0, 7, 0, 3},
		[]uint16{1024, 1017, 1024, 1021},
		0xdeadbeef,
	)
	d, err := diaryFromRecord(r)
	if err != nil {
		t.Fatalf("diaryFromRecord: %v", err)
	}
	if d.Length != 4 {
		t.Errorf("length = %d, want 4", d.Length)
	}
	if d.Self != 0xdeadbeef {
		t.Errorf("self = %#08x, want 0xdeadbeef", d.Self)
	}
	want := []DiaryEntry{{Index: 1, Count: 7, Remaining: 1017}, {Index: 3, Count: 3, Remaining: 1021}}
	if len(d.Entries) != len(want) {
		t.Fatalf("entries = %+v, want %+v", d.Entries, want)
	}
	for i, e := range d.Entries {
		if e != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, e, want[i])
		}
	}
}

// TestDiaryFromRecordAllDefaultIsAnEmptyPresentDiary is a Diary the walk
// reached (HasDiary true) whose every element is still the construction
// default: present, Length intact, Entries empty — not the same as absent.
func TestDiaryFromRecordAllDefaultIsAnEmptyPresentDiary(t *testing.T) {
	r := diaryRecordFixture([]uint32{0, 0, 0}, []uint16{1024, 1024, 1024}, 0)
	d, err := diaryFromRecord(r)
	if err != nil {
		t.Fatalf("diaryFromRecord: %v", err)
	}
	if d.Length != 3 || len(d.Entries) != 0 {
		t.Errorf("diary = %+v, want length 3, no entries", d)
	}
}

// TestDiaryFromRecordRefusesMismatchedCounts is SAV-667's own corpus-exact
// dword_count==word_count relationship, enforced as a refusal rather than
// silently reading past one array into the other's own bytes: 542 of 542
// corpus records agree, so a record that does not is hostile or unreadable
// input (B1), not a shape this reader guesses at.
func TestDiaryFromRecordRefusesMismatchedCounts(t *testing.T) {
	r := diaryRecordFixture([]uint32{0, 0, 0, 0}, []uint16{1024, 1024, 1024}, 0)
	r.Counts["Journal"] = 4
	r.Counts["JournalWords"] = 3
	if _, err := diaryFromRecord(r); err == nil {
		t.Fatal("diaryFromRecord accepted a 4-dword/3-word record")
	}
}

// TestDiaryFromRecordRefusesShortArrayBytes guards the internal invariant
// retainArray's own skip already establishes on a real walk (the declared
// count bounds the retained byte span): a record built some other way, with
// fewer array bytes than its own declared count, is refused rather than read
// out of bounds.
func TestDiaryFromRecordRefusesShortArrayBytes(t *testing.T) {
	r := newRecord("Diary", 0, 0)
	r.Counts["Journal"], r.Counts["JournalWords"] = 4, 4
	r.Raw["Journal"] = make([]byte, 8) // 2 dwords' worth, not 4
	r.Raw["JournalWords"] = make([]byte, 8)
	if _, err := diaryFromRecord(r); err == nil {
		t.Fatal("diaryFromRecord accepted a short dword array")
	}
}

// TestEncodeDiaryArraysRoundTripsThroughDiaryFromRecord is the exact inverse
// property SetDiary depends on: encoding a Diary this package decoded, then
// decoding the result again, reproduces the same value byte for byte.
func TestEncodeDiaryArraysRoundTripsThroughDiaryFromRecord(t *testing.T) {
	want := Diary{Length: 5, Entries: []DiaryEntry{{Index: 0, Count: 1, Remaining: 1023}, {Index: 4, Count: 6, Remaining: 1018}}, Self: 0x11}
	dwords, words, err := encodeDiaryArrays(want)
	if err != nil {
		t.Fatalf("encodeDiaryArrays: %v", err)
	}
	r := newRecord("Diary", 0, 0)
	r.Counts["Journal"], r.Counts["JournalWords"] = want.Length, want.Length
	r.Raw["Journal"], r.Raw["JournalWords"] = dwords, words
	r.Value["D2C"] = want.Self
	got, err := diaryFromRecord(r)
	if err != nil {
		t.Fatalf("diaryFromRecord: %v", err)
	}
	if got.Length != want.Length || len(got.Entries) != len(want.Entries) {
		t.Fatalf("round trip = %+v, want %+v", got, want)
	}
	for i, e := range got.Entries {
		if e != want.Entries[i] {
			t.Errorf("entry %d = %+v, want %+v", i, e, want.Entries[i])
		}
	}
}

// TestEncodeDiaryArraysRefusesAnOutOfRangeOrRepeatedIndex never allocates or
// writes past the declared Length, and never silently keeps the last of two
// entries claiming the same index.
func TestEncodeDiaryArraysRefusesAnOutOfRangeOrRepeatedIndex(t *testing.T) {
	if _, _, err := encodeDiaryArrays(Diary{Length: 3, Entries: []DiaryEntry{{Index: 3, Count: 1}}}); err == nil {
		t.Error("accepted index 3 in a 3-element diary")
	}
	if _, _, err := encodeDiaryArrays(Diary{Length: 3, Entries: []DiaryEntry{{Index: -1, Count: 1}}}); err == nil {
		t.Error("accepted a negative index")
	}
	if _, _, err := encodeDiaryArrays(Diary{Length: 3, Entries: []DiaryEntry{{Index: 1, Count: 1}, {Index: 1, Count: 2}}}); err == nil {
		t.Error("accepted a repeated index")
	}
}

// TestPlayerDiaryReportsAbsence covers a Player record PartyWalk did not
// attach a Diary to, which SAV-PLDIARY-054 says a corpus save never does but
// a hand-built or forensic partial record still can.
func TestPlayerDiaryReportsAbsence(t *testing.T) {
	rec := newRecord("Player", 0, 0)
	d, ok, err := PlayerDiary(rec)
	if err != nil || ok || d.Length != 0 {
		t.Errorf("PlayerDiary on a bare Player = %+v, %v, %v", d, ok, err)
	}
	if d, ok, err := PlayerDiary(nil); err != nil || ok || d.Length != 0 {
		t.Errorf("PlayerDiary(nil) = %+v, %v, %v", d, ok, err)
	}
}

// TestSetDiaryPatchesInPlaceAndLeavesUnrelatedBytesUntouched exercises
// SetDiary against a real walked record (hero()'s own per-character Diary,
// program_test.go's fixture, journal length 6, every element already
// non-default): an unchanged round trip through decode/SetDiary/decode is a
// byte-for-byte no-op, and a changed one touches only the record's own
// [Off,End) span.
func TestSetDiaryPatchesInPlaceAndLeavesUnrelatedBytesUntouched(t *testing.T) {
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{hero()}})
	_, rec, err := f.PartyWalk()
	if err != nil {
		t.Fatalf("PartyWalk: %v", err)
	}
	actors := rec.Refs["Actors"]
	if len(actors) != 1 {
		t.Fatalf("%d actors, want 1", len(actors))
	}
	diaryRec := actors[0].Refs["Diary"]
	if len(diaryRec) != 1 {
		t.Fatalf("%d diary records on the hero, want 1", len(diaryRec))
	}
	d, err := diaryFromRecord(diaryRec[0])
	if err != nil {
		t.Fatalf("diaryFromRecord: %v", err)
	}
	if d.Length != 6 || len(d.Entries) != 6 {
		t.Fatalf("hero's own diary = %+v, want 6 elements all non-default", d)
	}

	before := append([]byte(nil), f.Body...)
	if err := f.SetDiary(diaryRec[0], d); err != nil {
		t.Fatalf("SetDiary (no-op): %v", err)
	}
	if !bytes.Equal(f.Body, before) {
		t.Fatal("SetDiary with the exact decoded value changed the file's own bytes")
	}

	changed := d
	changed.Entries = append([]DiaryEntry(nil), d.Entries...)
	changed.Entries[2] = DiaryEntry{Index: changed.Entries[2].Index, Count: 999, Remaining: 25}
	off, end := diaryRec[0].Off, diaryRec[0].End
	if err := f.SetDiary(diaryRec[0], changed); err != nil {
		t.Fatalf("SetDiary (changed): %v", err)
	}
	if bytes.Equal(f.Body[off:end], before[off:end]) {
		t.Error("SetDiary with a changed value left the record's own span unchanged")
	}
	if !bytes.Equal(f.Body[:off], before[:off]) || !bytes.Equal(f.Body[end:], before[end:]) {
		t.Error("SetDiary touched bytes outside the record's own [Off,End) span")
	}

	_, rec2, err := f.PartyWalk()
	if err != nil {
		t.Fatalf("PartyWalk after SetDiary: %v", err)
	}
	got, err := diaryFromRecord(rec2.Refs["Actors"][0].Refs["Diary"][0])
	if err != nil {
		t.Fatalf("diaryFromRecord after SetDiary: %v", err)
	}
	if len(got.Entries) != len(changed.Entries) || got.Entries[2] != changed.Entries[2] {
		t.Errorf("re-walked diary = %+v, want entry 2 = %+v", got, changed.Entries[2])
	}
}

// TestSetDiaryRefusesALengthMismatch never grows or shrinks a Diary's own
// arrays: SAV-667 gives no rebuild that changes their length after load, and
// this package's own carried state never mutates entry count either.
func TestSetDiaryRefusesALengthMismatch(t *testing.T) {
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{hero()}})
	_, rec, err := f.PartyWalk()
	if err != nil {
		t.Fatalf("PartyWalk: %v", err)
	}
	diaryRec := rec.Refs["Actors"][0].Refs["Diary"][0]
	if err := f.SetDiary(diaryRec, Diary{Length: 7}); err == nil {
		t.Fatal("SetDiary accepted a length that disagrees with the record's own declared count")
	}
	if err := f.SetDiary(rec, Diary{}); err == nil {
		t.Fatal("SetDiary accepted a non-Diary record")
	}
}
