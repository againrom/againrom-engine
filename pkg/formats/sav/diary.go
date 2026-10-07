package sav

import "fmt"

// DiaryEntry is one element pair from a Diary's own two parallel arrays
// (SAV-DIARY-042) that departs from their shared construction default:
// dword[Index]==0, word[Index]==1024 (SAV-667's own zero-fill loop). Index is
// the array position, which SAV-667 places as a Units-table row — a unit
// type, from the LIVE installed catalog a fresh Diary sizes itself from —
// once a save has been loaded at all; this package is asset-free and does
// not resolve an index to the catalog entry it names. Count is the raw
// dword and Remaining the raw word exactly as the file carries them:
// SAV-668 measures word==1024-count with no exception across the full
// preserved corpus, and that word is a floor-limited countdown a located
// mutator decrements while the dword array's own writer was not found, but
// this reader carries both as observed rather than deriving one from the
// other — a file where they disagree is a corpus question for research, not
// something to normalise away (B1).
//
// A default element (Count==0, Remaining==1024) is never reported: it carries
// no information the file distinguishes from "never touched" (SAV-667's own
// zero-fill is indistinguishable from a slot no code has reached yet).
type DiaryEntry struct {
	Index     int
	Count     uint32
	Remaining uint16
}

// Diary is one Diary record's complete decoded content. Length is the two
// arrays' own shared declared element count (SAV-667: dword_count==word_count
// on 542 of 542 corpus records) — carried separately from len(Entries)
// because most elements are still at the construction default and are not
// reported as entries, so Length alone says how large the file's own arrays
// were, which a byte-for-byte re-export needs even when Entries is empty.
// Self is the raw D2C reference value (SAV-669: resolves to the Diary's own
// enclosing Player on every nonzero corpus case — 408 of 410 Player-owned
// records, the remaining 2 the same project-written saves SAV-667 names —
// and is always null on the 132 corpus Humanoid-owned records). This project
// does not resolve it to an object: nothing beyond confirming that
// self-identity is known to consume it, so it is carried for completeness
// and is not itself gameplay state.
type Diary struct {
	Length  int
	Entries []DiaryEntry
	Self    uint32
}

// diaryFromRecord decodes a "Diary"-class record's two parallel arrays —
// Journal, the CDWordArray at +0x04, and JournalWords, the CWordArray at
// +0x18 (SAV-DIARY-042) — into the typed, sparse form above.
//
// It refuses a record whose two declared counts disagree, or whose retained
// array bytes do not match their own declared count: SAV-667 finds the two
// counts equal on every one of 542 corpus records and program.go's walker
// already bounds each array's own declared count against maxListElements and
// the stream's remaining length before allocating it (stpDWordArray,
// stpWordArray), so a record failing either check here is hostile or
// unreadable input, not a shape this reader guesses at.
func diaryFromRecord(r *Record) (Diary, error) {
	if r == nil {
		return Diary{}, fmt.Errorf("sav: no diary record")
	}
	n := r.Counts["Journal"]
	if w := r.Counts["JournalWords"]; w != n {
		return Diary{}, fmt.Errorf("sav: diary at %d declares %d dword and %d word elements", r.Off, n, w)
	}
	if n < 0 {
		return Diary{}, fmt.Errorf("sav: diary at %d declares a negative element count", r.Off)
	}
	dwords, words := r.Raw["Journal"], r.Raw["JournalWords"]
	if len(dwords) != 4*n || len(words) != 2*n {
		return Diary{}, fmt.Errorf("sav: diary at %d has %d/%d array bytes for %d elements",
			r.Off, len(dwords), len(words), n)
	}
	d := Diary{Length: n, Self: r.value("D2C")}
	for i := 0; i < n; i++ {
		count, remaining := u32(dwords, 4*i), u16(words, 2*i)
		if count == 0 && remaining == 1024 {
			continue
		}
		d.Entries = append(d.Entries, DiaryEntry{Index: i, Count: count, Remaining: remaining})
	}
	return d, nil
}

// PlayerDiary decodes the Player's own inline Diary (SAV-662, SAV-669: a
// virtual dispatch through Player+0x40's own vtable, never tagged or
// archive-indexed) from rec, PartyWalk's own second return value. It reports
// present=false, with no error, for a Player record whose programme carried
// no Diary — SAV-PLDIARY-054's own contract requires exactly one, so this
// only guards a rec this package did not itself build.
func PlayerDiary(rec *Record) (Diary, bool, error) {
	if rec == nil {
		return Diary{}, false, nil
	}
	refs := rec.Refs["Diary"]
	if len(refs) == 0 {
		return Diary{}, false, nil
	}
	d, err := diaryFromRecord(refs[0])
	if err != nil {
		return Diary{}, false, err
	}
	return d, true, nil
}

// encodeDiaryArrays rebuilds the two parallel arrays' own raw bytes from d,
// reproducing the construction default (0, 1024) at every index d.Entries
// does not name. It is the exact inverse of diaryFromRecord's own sparse
// projection: encoding a Diary this package decoded reproduces the original
// dwords/words bytes exactly, whether or not any index departs from default.
func encodeDiaryArrays(d Diary) (dwords, words []byte, err error) {
	dwords, words = make([]byte, 4*d.Length), make([]byte, 2*d.Length)
	for i := range d.Length {
		put32(dwords, 4*i, 0)
		put16(words, 2*i, 1024)
	}
	seen := make(map[int]bool, len(d.Entries))
	for _, e := range d.Entries {
		if e.Index < 0 || e.Index >= d.Length {
			return nil, nil, fmt.Errorf("sav: diary entry index %d outside its own %d-element arrays", e.Index, d.Length)
		}
		if seen[e.Index] {
			return nil, nil, fmt.Errorf("sav: diary entry index %d repeated", e.Index)
		}
		seen[e.Index] = true
		put32(dwords, 4*e.Index, e.Count)
		put16(words, 2*e.Index, e.Remaining)
	}
	return dwords, words, nil
}

// SetDiary overwrites one Diary record's two array bodies in place, from a
// complete typed value. rec must be the exact *Record diaryFromRecord (or
// PlayerDiary) decoded: this function replays the same count parse
// diaryFromRecord's own walk performed, starting at rec.Off, so it finds the
// identical byte offsets without a separate offset ever being retained on
// Record. d.Length must equal the record's own declared count: SAV-667 gives
// no rebuild that ever changes a Diary's own array length after load, and
// this package's own carried state (docs/DIVERGENCES.md) never mutates entry
// count either, so a length mismatch is a caller error, not a shape to grow
// or shrink into.
func (f *File) SetDiary(rec *Record, d Diary) error {
	if rec == nil || rec.Class != "Diary" {
		return fmt.Errorf("sav: SetDiary requires a Diary record")
	}
	if d.Length != rec.Counts["Journal"] || d.Length != rec.Counts["JournalWords"] {
		return fmt.Errorf("sav: diary length %d does not match the record's own %d/%d",
			d.Length, rec.Counts["Journal"], rec.Counts["JournalWords"])
	}
	dwords, words, err := encodeDiaryArrays(d)
	if err != nil {
		return err
	}
	w := &walker{b: f.Body, p: rec.Off}
	n1, err := w.count()
	if err != nil || n1 != d.Length {
		return fmt.Errorf("sav: diary at %d: cannot relocate its own dword array", rec.Off)
	}
	journalOff := w.p
	if journalOff+len(dwords) > len(f.Body) {
		return fmt.Errorf("sav: diary at %d: dword array overruns the file", rec.Off)
	}
	w.p += len(dwords)
	n2, err := w.count()
	if err != nil || n2 != d.Length {
		return fmt.Errorf("sav: diary at %d: cannot relocate its own word array", rec.Off)
	}
	wordsOff := w.p
	if wordsOff+len(words) > len(f.Body) {
		return fmt.Errorf("sav: diary at %d: word array overruns the file", rec.Off)
	}
	copy(f.Body[journalOff:], dwords)
	copy(f.Body[wordsOff:], words)
	rec.Raw["Journal"] = append([]byte(nil), dwords...)
	rec.Raw["JournalWords"] = append([]byte(nil), words...)
	return nil
}
