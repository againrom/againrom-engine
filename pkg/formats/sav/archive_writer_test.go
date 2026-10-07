package sav

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func readArchive1115(t *testing.T, b []byte, n int, retain bool) []*Record {
	t.Helper()
	w := &walker{b: b, next: 1, classes: map[uint16]string{}, objects: map[uint16]*Record{}, retainGraph: retain}
	var roots []*Record
	for i := 0; i < n; i++ {
		r, err := w.object(0)
		if err != nil {
			t.Fatal(err)
		}
		roots = append(roots, r)
	}
	if w.p != len(b) {
		t.Fatalf("reader stopped at %d of %d", w.p, len(b))
	}
	return roots
}

func TestArchive1115SparseAliasesDetachedAndReminted(t *testing.T) {
	s := newStream()
	s.obj("Sack") // class1, object2
	s.token(0x1413, 23, 42, 70, 0, 0)
	s.u32(0x11223344)
	s.u32(4)
	s.null()
	s.item(wantPiece{class: "Weapon", code: 0x2345, row: 3, stack: 2})
	weaponIndex := s.classes["Weapon"] + 1
	s.backref(weaponIndex)
	s.null()
	s.u32(0x44556677)
	s.u32(0x8899aabb)
	s.null()
	s.backref(2)
	expected := append([]byte(nil), s.b...)
	roots := readArchive1115(t, s.b, 3, true)
	if len(roots[0].Refs["Contents"]) != 2 {
		t.Fatal("compact projection changed")
	}
	refs := roots[0].RefSlots["Contents"]
	if len(refs) != 4 || refs[0] != nil || refs[3] != nil || refs[1] != refs[2] {
		t.Fatal("sparse alias lost")
	}
	roots = detachArchiveRecords(roots)
	clear(s.b)
	if roots[0] != roots[2] || roots[0].Off != 0 || roots[0].End != 0 || roots[0].Index != 0 {
		t.Fatal("source coordinates retained or alias lost")
	}
	got, err := serializeArchiveReferences(roots)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, expected) {
		t.Fatal("detached graph did not reconstruct literal archive")
	}
	// New root order changes archive indices. The original indices and source
	// bytes are unavailable; both container aliases must name the new object.
	weapon := roots[0].RefSlots["Contents"][1]
	got, err = serializeArchiveReferences([]*Record{weapon, roots[0], weapon})
	if err != nil {
		t.Fatal(err)
	}
	reloaded := readArchive1115(t, got, 3, true)
	if reloaded[0] != reloaded[2] || reloaded[1].RefSlots["Contents"][1] != reloaded[0] || reloaded[1].RefSlots["Contents"][2] != reloaded[0] {
		t.Fatal("reminted alias mismatch")
	}
	weapon.Value["F42"] = 7
	changed, err := serializeArchiveReferences([]*Record{weapon})
	if err != nil {
		t.Fatal(err)
	}
	if readArchive1115(t, changed, 1, true)[0].Value["F42"] != 7 {
		t.Fatal("writer ignored changed named value")
	}
}

func TestArchive1115PlayerGroupsBooksAndInlineDiary(t *testing.T) {
	f := &File{Body: (walkFixture{mapName: "10.alm", mission: 10, groups: 2, chars: []wantChar{
		{name: "one", runtimeID: 1, journal: 2, book: func(s *stream) {
			s.u8(1)
			s.u32(0x12345678)
			s.u32(4)
			s.null()
			s.obj("Spell")
			idx := s.next - 1
			s.u8(6)
			s.u8(9)
			s.u8(10)
			s.u16(0x0c0c)
			s.u32(0xb0000006)
			s.backref(idx)
		}},
		{name: "two", runtimeID: 2, journal: -1},
	}}).body()}
	if err := f.readHead(); err != nil {
		t.Fatal(err)
	}
	w := &walker{b: f.Body, p: f.Head.End, next: 1, classes: map[uint16]string{}, objects: map[uint16]*Record{}, retainGraph: true}
	p, err := w.object(0)
	if err != nil {
		t.Fatal(err)
	}
	expected := append([]byte(nil), f.Body[f.Head.End:w.p]...)
	roots := detachArchiveRecords([]*Record{p})
	clear(f.Body)
	got, err := serializeArchiveReferences(roots)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, expected) {
		t.Fatal("Player/Group/inline Diary programme differs")
	}
	actor := roots[0].Groups[0].RefSlots["Actors"][0]
	if actor.Value["SpellsHeader"] != 0x12345678 || actor.RefSlots["Spells"][0] != nil || actor.RefSlots["Spells"][1] != actor.RefSlots["Spells"][2] {
		t.Fatal("book header/slots lost")
	}
	if len(roots[0].Refs["Diary"][0].Raw["Journal"]) != 4*playerJournal {
		t.Fatal("inline Diary array lost")
	}
	// No default value can make the historical compact-only projection an
	// exact graph, even when a missing array happened to have length zero.
	compact := readArchive1115(t, expected, 1, false)
	if b, err := serializeArchiveReferences(compact); err == nil || b != nil {
		t.Fatal("accepted lossy projection")
	}
}

func TestArchive1115NonemptyWordAndRawArrays(t *testing.T) {
	s := newStream()
	s.human(wantChar{name: "words", runtimeID: 1, journal: -1})
	// Literal Unit programme: Token37 + effect-count4, then U15C count.
	at := 6 + len("Human") + 37 + 4
	b := append([]byte(nil), s.b[:at]...)
	b = append(b, 2, 0, 0x34, 0x12, 0x78, 0x56)
	b = append(b, s.b[at+2:]...)
	r := readArchive1115(t, b, 1, true)
	if !bytes.Equal(r[0].Raw["U15C"], []byte{0x34, 0x12, 0x78, 0x56}) {
		t.Fatal("Unit words discarded")
	}
	got, err := serializeArchiveReferences(detachArchiveRecords(r))
	if err != nil || !bytes.Equal(got, b) {
		t.Fatalf("word-array inverse: %v", err)
	}
	o := newStream()
	o.obj("Outpost")
	o.token(0x1615, 128, 128, 2, 1, 0)
	o.raw(40, 0x31)
	o.u32(11)
	o.u32(12)
	o.u32(13)
	o.u32(14)
	o.u16(2)
	o.raw(16, 0x72)
	or := readArchive1115(t, o.b, 1, true)
	got, err = serializeArchiveReferences(detachArchiveRecords(or))
	if err != nil || !bytes.Equal(got, o.b) {
		t.Fatalf("raw-array inverse: %v", err)
	}
}

func TestArchive1115RefusesInvalidGraphsWithoutOutput(t *testing.T) {
	s := newStream()
	s.obj("Sack")
	s.token(0x1413, 128, 128, 1, 0, 0)
	s.u32(0)
	s.u32(0)
	s.u32(0)
	s.u32(0)
	original := readArchive1115(t, s.b, 1, true)
	for name, mutate := range map[string]func(*Record){
		"missing scalar": func(r *Record) { delete(r.Value, "S3C") },
		"byte width":     func(r *Record) { r.Value["T0C"] = 256 },
		"word width":     func(r *Record) { r.Value["T0E"] = 0x10000 },
		"count":          func(r *Record) { r.Counts["Contents"] = maxListElements + 1 },
		"lost slot":      func(r *Record) { r.Counts["Contents"] = 1 },
		"class":          func(r *Record) { r.Class = "Missing" },
	} {
		t.Run(name, func(t *testing.T) {
			r := detachArchiveRecords(original)
			mutate(r[0])
			b, err := serializeArchiveReferences(r)
			if err == nil || b != nil {
				t.Fatal("invalid graph produced output")
			}
		})
	}
	w := newArchiveWriter()
	w.next = 0x8000
	if err := w.reference(original[0], 0); err == nil || !strings.Contains(err.Error(), "index") {
		t.Fatal("unbounded archive index")
	}
	w = newArchiveWriter()
	w.b = make([]byte, maxArchiveOutput-1)
	if err := w.reference(nil, 0); err == nil || len(w.b) != maxArchiveOutput-1 {
		t.Fatal("output budget not enforced before append")
	}
	if binary.LittleEndian.Uint16(s.b) != classIntro {
		t.Fatal("source mutated")
	}
}
