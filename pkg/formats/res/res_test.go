package res_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"againrom/pkg/formats/res"
)

// nodeSpec is a single 32-byte registry node in a synthetic fixture.
//
// For a file node: a = payload byte offset within the archive, b = payload size.
// For a directory node: a = index of the first child node, b = number of children.
type nodeSpec struct {
	field0 uint32
	a      uint32
	b      uint32
	typ    uint32
	name   [16]byte
}

// padName builds a 16-byte name field: the given bytes, a NUL terminator, then
// 0xCD padding to fill the field (matching the on-disk convention).
func padName(b []byte) [16]byte {
	var n [16]byte
	for i := range n {
		n[i] = 0xCD
	}
	copy(n[:], b)
	if len(b) < 16 {
		n[len(b)] = 0x00
	}
	return n
}

func fileName(s string) [16]byte {
	return padName([]byte(s))
}

func fileNode(name string, offset, size uint32) nodeSpec {
	return nodeSpec{a: offset, b: size, typ: 0, name: fileName(name)}
}

func dirNode(name string, firstChild, count uint32) nodeSpec {
	return nodeSpec{a: firstChild, b: count, typ: 1, name: fileName(name)}
}

func concat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// buildArchive assembles a fully synthetic .res archive:
//
//	[ header (0x18) ][ data ][ registry: len(nodes) x 32-byte nodes ]
//
// The header signature, regOffset (= 0x18 + len(data)) and nodeCount
// (= len(nodes)) are all written consistently; callers that want a malformed
// archive corrupt specific bytes of the returned slice afterwards.
func buildArchive(rootCount uint32, data []byte, nodes []nodeSpec) []byte {
	le := binary.LittleEndian
	regOffset := 0x18 + len(data)

	b := make([]byte, 0x18)
	le.PutUint32(b[0x00:], 0x31415926) // signature
	le.PutUint32(b[0x04:], 0)          // opaque
	le.PutUint32(b[0x08:], rootCount)  // rootCount
	le.PutUint32(b[0x0C:], 0)          // opaque
	le.PutUint32(b[0x10:], uint32(regOffset))
	le.PutUint32(b[0x14:], uint32(len(nodes)))

	b = append(b, data...)
	for _, n := range nodes {
		rec := make([]byte, 0x20)
		le.PutUint32(rec[0x00:], n.field0)
		le.PutUint32(rec[0x04:], n.a)
		le.PutUint32(rec[0x08:], n.b)
		le.PutUint32(rec[0x0C:], n.typ)
		copy(rec[0x10:], n.name[:])
		b = append(b, rec...)
	}
	return b
}

// nestedArchive is the AC-1 fixture: a root directory "dir" containing a file
// "a.txt" and a subdirectory "sub" (containing "b.txt"), plus a top-level FILE
// root "root.bin". Data payloads live in the data region starting at 0x18 (24).
func nestedArchive() []byte {
	aData := []byte("AAAA")   // 4 bytes at offset 24
	bData := []byte("BBBBBB") // 6 bytes at offset 28
	rData := []byte("ROOT")   // 4 bytes at offset 34
	data := concat(aData, bData, rData)

	nodes := []nodeSpec{
		dirNode("dir", 1, 2),        // node 0: children nodes 1,2
		fileNode("a.txt", 24, 4),    // node 1
		dirNode("sub", 3, 1),        // node 2: child node 3
		fileNode("b.txt", 28, 6),    // node 3
		fileNode("root.bin", 34, 4), // node 4: top-level file root
	}
	// Roots (not referenced as a child): node 0 and node 4.
	return buildArchive(2, data, nodes)
}

func singleFileArchive() []byte {
	return buildArchive(1, []byte("DATA"), []nodeSpec{fileNode("f.txt", 24, 4)})
}

func emptyArchive() []byte {
	return buildArchive(0, nil, nil)
}

func assertReject(t *testing.T, name string, b []byte) {
	t.Helper()
	a, err := res.OpenBytes(b)
	if err == nil {
		t.Fatalf("%s: expected an error, got nil", name)
	}
	if a != nil {
		t.Fatalf("%s: expected a nil *Archive on error, got non-nil", name)
	}
}

// TestOpenIndexesNestedTree - AC-1.
func TestOpenIndexesNestedTree(t *testing.T) {
	a, err := res.OpenBytes(nestedArchive())
	if err != nil {
		t.Fatalf("OpenBytes: unexpected error: %v", err)
	}
	if a == nil {
		t.Fatal("OpenBytes: nil archive without error")
	}

	got := a.Entries()
	want := []res.Entry{
		{Path: "dir/a.txt", Offset: 24, Size: 4},
		{Path: "dir/sub/b.txt", Offset: 28, Size: 6},
		{Path: "root.bin", Offset: 34, Size: 4},
	}
	if len(got) != len(want) {
		t.Fatalf("Entries(): got %d entries, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Entries()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}

	// Offset/Size are correct: reading each entry yields its exact payload.
	for _, tc := range []struct {
		path string
		want []byte
	}{
		{"dir/a.txt", []byte("AAAA")},
		{"dir/sub/b.txt", []byte("BBBBBB")},
		{"root.bin", []byte("ROOT")},
	} {
		b, err := a.ReadFile(tc.path)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", tc.path, err)
		}
		if !bytes.Equal(b, tc.want) {
			t.Fatalf("ReadFile(%q) = %q, want %q", tc.path, b, tc.want)
		}
	}
}

func TestReadFileNormalizesPath(t *testing.T) {
	a, err := res.OpenBytes(nestedArchive())
	if err != nil {
		t.Fatalf("OpenBytes: %v", err)
	}

	// Every spelling below normalizes to "dir/a.txt".
	spellings := []string{
		"dir/a.txt",
		"DIR/A.TXT",
		"Dir\\A.txt",
		"\\dir\\a.txt\\",
		"/DIR/A.TXT/",
	}
	want := []byte("AAAA")
	for _, p := range spellings {
		b, err := a.ReadFile(p)
		if err != nil {
			t.Fatalf("ReadFile(%q): unexpected error: %v", p, err)
		}
		if !bytes.Equal(b, want) {
			t.Fatalf("ReadFile(%q) = %q, want %q", p, b, want)
		}
	}

	// A normalized-equal path resolves to the same entry.
	viaSlash, err := a.ReadFile("dir/a.txt")
	if err != nil {
		t.Fatalf("ReadFile(slash): %v", err)
	}
	viaBackslash, err := a.ReadFile("DIR\\A.TXT")
	if err != nil {
		t.Fatalf("ReadFile(backslash): %v", err)
	}
	if !bytes.Equal(viaSlash, viaBackslash) {
		t.Fatalf("normalized-equal paths disagree: %q vs %q", viaSlash, viaBackslash)
	}
}

// TestReadFileMissingIsNotExist - AC-3.
func TestReadFileMissingIsNotExist(t *testing.T) {
	a, err := res.OpenBytes(nestedArchive())
	if err != nil {
		t.Fatalf("OpenBytes: %v", err)
	}

	_, err = a.ReadFile("no/such/file.bin")
	if err == nil {
		t.Fatal("ReadFile(missing): expected an error, got nil")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ReadFile(missing): errors.Is(err, fs.ErrNotExist) == false; err = %v", err)
	}
}

// TestOpenRejectsHeader - AC-4.
func TestOpenRejectsHeader(t *testing.T) {
	// Bad signature.
	bad := singleFileArchive()
	binary.LittleEndian.PutUint32(bad[0x00:], 0xDEADBEEF)
	assertReject(t, "bad signature", bad)

	// regOffset past EOF.
	past := singleFileArchive()
	binary.LittleEndian.PutUint32(past[0x10:], uint32(len(past)+1000))
	assertReject(t, "regOffset past EOF", past)
}

// TestOpenRejectsGeometry - SC-4a: a regOffset inside the header.
//
// This test used to pin two further rejections, and both were wrong: a registry
// region whose length is not a multiple of 32, and a header nodeCount that
// disagrees with (EOF - regOffset)/32. The registry is the header's nodeCount
// records and the region length is not evidence about it (AC-11, AC-12) - a
// disagreement is a rejection in one direction only, which is
// TestOpenRejectsShortRegistry, and residue in the other, which is
// TestOpenIgnoresRegistryResidue.
func TestOpenRejectsGeometry(t *testing.T) {
	low := singleFileArchive()
	binary.LittleEndian.PutUint32(low[0x10:], 0x10)
	assertReject(t, "regOffset below 0x18", low)
}

// TestOpenRejectsShortRegistry - AC-12 / SC-13. A nodeCount claiming records the
// archive does not hold is rejected: the reader would otherwise slice past EOF.
// This is hardening, declared in spec.md as stricter than the original.
func TestOpenRejectsShortRegistry(t *testing.T) {
	// One record over: the archive holds exactly one, the header claims two.
	oneOver := singleFileArchive()
	binary.LittleEndian.PutUint32(oneOver[0x14:], 2)
	assertReject(t, "nodeCount one record over", oneOver)

	// Far over.
	farOver := singleFileArchive()
	binary.LittleEndian.PutUint32(farOver[0x14:], 5)
	assertReject(t, "nodeCount five records over", farOver)

	// The widest count a u32 can carry: 0xFFFFFFFF records of 32 bytes. The
	// bound must be computed and rejected before any slice or int narrowing.
	widest := singleFileArchive()
	binary.LittleEndian.PutUint32(widest[0x14:], 0xFFFFFFFF)
	assertReject(t, "nodeCount 0xFFFFFFFF", widest)
}

// stalePackerResidue is the shape a re-packer leaves behind when it overwrites an
// archive in place: 23 bytes past the node array that align as a byte suffix of a
// node record, ending in a name field that stops exactly at EOF. Every byte is
// written as hex, never as text: 0x8F would render as a Cyrillic letter under
// CP866 or CP1251, and 0xCD is the name field's padding (golden rule 2).
var stalePackerResidue = []byte{
	0x07, 0x00, 0x00, 0x00, // the tail of a size word
	0x00, 0x00, 0x00, 0x00, // the tail of a type word (0 = file)
	0x6F, 0x6E, 0x6D, 0x61, 0x70, 0x2E, 0x38, 0x8F, 0x36, 0x35, 0x35, // 11 name bytes
	0x00, 0xCD, 0xCD, 0xCD, // the NUL terminator and its padding
}

// nestedEntries is nestedArchive's index, written out rather than read back from
// a parse of the same fixture: the residue cases below have to be pinned to
// literals, or "unchanged" would only say the reader agrees with itself.
var nestedEntries = []res.Entry{
	{Path: "dir/a.txt", Offset: 24, Size: 4},
	{Path: "dir/sub/b.txt", Offset: 28, Size: 6},
	{Path: "root.bin", Offset: 34, Size: 4},
}

func assertEntries(t *testing.T, name string, got, want []res.Entry) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: Entries(): got %d entries, want %d: %+v", name, len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: Entries()[%d] = %+v, want %+v", name, i, got[i], want[i])
		}
	}
}

func TestOpenIgnoresRegistryResidue(t *testing.T) {
	t.Run("stale bytes after the registry", func(t *testing.T) {
		withResidue := concat(nestedArchive(), stalePackerResidue)
		if len(withResidue)%0x20 == 0 {
			t.Fatalf("fixture is meant to break the 32-byte identity: %d bytes", len(withResidue))
		}
		a, err := res.OpenBytes(withResidue)
		if err != nil {
			t.Fatalf("OpenBytes(residue): unexpected error: %v", err)
		}
		assertEntries(t, "residue", a.Entries(), nestedEntries)

		// The residue reaches no payload either: each entry still reads its own
		// bytes from the data region.
		for _, tc := range []struct {
			path string
			want []byte
		}{
			{"dir/a.txt", []byte("AAAA")},
			{"dir/sub/b.txt", []byte("BBBBBB")},
			{"root.bin", []byte("ROOT")},
		} {
			b, err := a.ReadFile(tc.path)
			if err != nil {
				t.Fatalf("ReadFile(%q): %v", tc.path, err)
			}
			if !bytes.Equal(b, tc.want) {
				t.Fatalf("ReadFile(%q) = %q, want %q", tc.path, b, tc.want)
			}
		}
	})

	t.Run("an uncounted trailing record", func(t *testing.T) {
		// nodeCount one short of the five records present: node 4 - the file root
		// "root.bin" - is outside the registry, so it is not indexed and every
		// other entry is unchanged. The disagreement in the direction the
		// original tolerates, the opposite of TestOpenRejectsShortRegistry's.
		short := nestedArchive()
		binary.LittleEndian.PutUint32(short[0x14:], 4)
		a, err := res.OpenBytes(short)
		if err != nil {
			t.Fatalf("OpenBytes(short count): unexpected error: %v", err)
		}
		assertEntries(t, "uncounted record", a.Entries(), nestedEntries[:2])
		if _, err := a.ReadFile("root.bin"); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("ReadFile(root.bin) on an uncounted record: err = %v, want fs.ErrNotExist", err)
		}
	})

	t.Run("appending any number of bytes changes nothing", func(t *testing.T) {
		for n := 1; n <= 40; n++ {
			pad := make([]byte, n)
			for i := range pad {
				pad[i] = 0xCD
			}
			a, err := res.OpenBytes(concat(nestedArchive(), pad))
			if err != nil {
				t.Fatalf("OpenBytes(+%d bytes): unexpected error: %v", n, err)
			}
			assertEntries(t, fmt.Sprintf("+%d bytes", n), a.Entries(), nestedEntries)
		}
	})
}

// TestOpenRejectsRanges - AC-5.
func TestOpenRejectsRanges(t *testing.T) {
	// A directory whose child range runs past nodeCount.
	dirPast := buildArchive(1, []byte("DATA"), []nodeSpec{
		dirNode("root", 1, 5),    // claims children [1,6) but only 2 nodes exist
		fileNode("a.txt", 24, 4), // node 1
	})
	assertReject(t, "dir child range past nodeCount", dirPast)

	// A file whose payload range exceeds the data region [0x18, regOffset).
	filePast := buildArchive(1, []byte("DATA"), []nodeSpec{
		fileNode("big.bin", 24, 1000),
	})
	assertReject(t, "file payload past data region", filePast)
}

// TestOpenRejectsTypeAndCycle - AC-6.
func TestOpenRejectsTypeAndCycle(t *testing.T) {
	// A node with a type outside {0, 1}.
	badType := buildArchive(1, []byte("DATA"), []nodeSpec{
		{a: 0, b: 0, typ: 2, name: fileName("weird")},
	})
	assertReject(t, "unknown node type", badType)

	// A cyclic directory link reachable from a root: node 1 lists itself.
	cyclic := buildArchive(1, nil, []nodeSpec{
		dirNode("root", 1, 1), // node 0 -> node 1
		dirNode("loop", 1, 1), // node 1 -> node 1 (self)
	})
	assertReject(t, "cyclic directory link", cyclic)
}

func TestOpenRejectsNonTree(t *testing.T) {
	shared := buildArchive(2, []byte("DATA"), []nodeSpec{
		dirNode("d0", 2, 1),           // node 0 -> node 2
		dirNode("d1", 2, 1),           // node 1 -> node 2
		fileNode("shared.txt", 24, 4), // node 2, shared by two parents
	})
	assertReject(t, "node shared by two directories", shared)
}

// TestNameDecodeCP866 - AC-7. Node name bytes {'A','B',0x80} + NUL + 0xCD
// padding decode (CP866) to "AB" followed by U+0410; the padding is dropped.
// The decoded name is asserted case-insensitively because Path is normalized
// (lower-cased), and U+0410 lower-cases to U+0430.
func TestNameDecodeCP866(t *testing.T) {
	name := padName([]byte{'A', 'B', 0x80})
	arch := buildArchive(1, []byte("DATA"), []nodeSpec{
		{a: 24, b: 4, typ: 0, name: name},
	})

	a, err := res.OpenBytes(arch)
	if err != nil {
		t.Fatalf("OpenBytes: %v", err)
	}
	entries := a.Entries()
	if len(entries) != 1 {
		t.Fatalf("Entries(): got %d, want 1: %+v", len(entries), entries)
	}

	const want = "AB\u0410"
	got := entries[0].Path
	if !strings.EqualFold(got, want) {
		t.Fatalf("decoded name = %q, want (case-insensitive) %q", got, want)
	}
}

func TestOpenKeepsHighByteFoldASCIIOnly(t *testing.T) {
	nameHigh1 := padName([]byte{'x', 0x80}) // CP866 -> "x" + U+0410
	nameHigh2 := padName([]byte{'x', 0xA0}) // CP866 -> "x" + U+0430

	// Expected/looked-up paths, built from numeric code points (never a
	// literal Cyrillic glyph in source) via rune -> UTF-8 string conversion.
	pathHigh1 := "x" + string(rune(0x0410)) // U+0410, Cyrillic capital letter A
	pathHigh2 := "x" + string(rune(0x0430)) // U+0430, Cyrillic small letter a

	data := concat([]byte("AAAA"), []byte("BBBB"), []byte("CCCC"))
	arch := buildArchive(3, data, []nodeSpec{
		{a: 24, b: 4, typ: 0, name: nameHigh1},
		{a: 28, b: 4, typ: 0, name: nameHigh2},
		fileNode("MixedCase", 32, 4),
	})

	a, err := res.OpenBytes(arch)
	if err != nil {
		t.Fatalf("OpenBytes: %v", err)
	}

	// The ASCII-case path hits regardless of case.
	gotAscii, err := a.ReadFile("mixedcase")
	if err != nil {
		t.Fatalf("ReadFile(ascii, folded case): %v", err)
	}
	if !bytes.Equal(gotAscii, []byte("CCCC")) {
		t.Fatalf("ReadFile(ascii, folded case) = %q, want %q", gotAscii, "CCCC")
	}

	// The two high-byte names stay distinct: each returns its own bytes. A
	// Unicode fold (the old strings.ToLower normalize) would have collapsed
	// both to the same key, "x" + U+0430, and made the second unreachable.
	got1, err := a.ReadFile(pathHigh1)
	if err != nil {
		t.Fatalf("ReadFile(x+U+0410): %v", err)
	}
	if !bytes.Equal(got1, []byte("AAAA")) {
		t.Fatalf("ReadFile(x+U+0410) = %q, want %q", got1, "AAAA")
	}

	got2, err := a.ReadFile(pathHigh2)
	if err != nil {
		t.Fatalf("ReadFile(x+U+0430): %v", err)
	}
	if !bytes.Equal(got2, []byte("BBBB")) {
		t.Fatalf("ReadFile(x+U+0430) = %q, want %q", got2, "BBBB")
	}
}

// TestOpenIndexesANamelessRecord - AC-13. A name field empty at its first byte
// is not a rejection, and this test exists to make that a pinned decision rather
// than an accident of the walk: pkg/vfs relies on the answer (its enumeration
// drops the empty path because no address of its grammar reaches one), so a
// refactor here that quietly stopped indexing such a record would surface as an
// unrelated-looking VFS failure instead of as this test.
//
// Three shapes, because the empty name reaches the index three ways. Fixture
// names go through the same padName helper as every other name, so a nameless
// field is a NUL at byte 0 followed by 0xCD padding - the on-disk convention,
// not a special case invented here. No archive in either shipped release holds
// any of the three.
func TestOpenIndexesANamelessRecord(t *testing.T) {
	// (a) A nameless FILE at the top level: nothing precedes it in the path, so
	// it keys the index under the empty string and is reachable through it.
	t.Run("root file keys the empty path", func(t *testing.T) {
		named := []byte("NAMED")
		unnamed := []byte("UNNAMED!")
		arch := buildArchive(2, concat(named, unnamed), []nodeSpec{
			fileNode("x.bin", 24, uint32(len(named))),
			fileNode("", 24+uint32(len(named)), uint32(len(unnamed))),
		})

		a, err := res.OpenBytes(arch)
		if err != nil {
			t.Fatalf("OpenBytes: %v", err)
		}
		entries := a.Entries()
		if len(entries) != 2 {
			t.Fatalf("Entries() = %+v, want 2 entries", entries)
		}
		if entries[1].Path != "" {
			t.Errorf("Entries()[1].Path = %q, want the empty path", entries[1].Path)
		}
		// Reachable, and by every spelling that normalizes to the empty key.
		for _, name := range []string{"", "/", `\`, "//"} {
			got, err := a.ReadFile(name)
			if err != nil {
				t.Fatalf("ReadFile(%q): %v", name, err)
			}
			if !bytes.Equal(got, unnamed) {
				t.Errorf("ReadFile(%q) = %q, want %q", name, got, unnamed)
			}
		}
	})

	// (b) A nameless file INSIDE a directory contributes nothing to the joined
	// path, so it keys under the directory's own path - colliding with a root
	// file of that name. Entries() lists both, in node order; the index binds
	// the first, and the second is indexed but unreachable.
	t.Run("nameless child collides with its directory name", func(t *testing.T) {
		rootFile := []byte("ROOT-FILE-data")
		child := []byte("CHILD-of-data")
		arch := buildArchive(2, concat(rootFile, child), []nodeSpec{
			dirNode("data", 2, 1), // node 0: its one child is node 2
			fileNode("data", 24, uint32(len(rootFile))),
			fileNode("", 24+uint32(len(rootFile)), uint32(len(child))),
		})
		assertCollision(t, arch, "data", rootFile, child)
	})

	// (c) The empty name at an INTERIOR node: a nameless directory whose child
	// shares a root file's name. Neither leaf name is empty, so no inspection of
	// leaf names would find this one - it is the shape the other two do not
	// cover, and the reason all three are here.
	t.Run("nameless directory collides through its child", func(t *testing.T) {
		rootFile := []byte("ROOT-x")
		nested := []byte("NESTED-x")
		arch := buildArchive(2, concat(rootFile, nested), []nodeSpec{
			dirNode("", 2, 1), // node 0: nameless dir, its one child is node 2
			fileNode("x.bin", 24, uint32(len(rootFile))),
			fileNode("x.bin", 24+uint32(len(rootFile)), uint32(len(nested))),
		})
		assertCollision(t, arch, "x.bin", rootFile, nested)
	})
}

// assertCollision pins the duplicate-key half of AC-13: an archive whose two
// file nodes normalize to one path lists both entries in node order, and the
// index binds that path to the FIRST, leaving the second indexed but with no
// path that reaches it.
func assertCollision(t *testing.T, arch []byte, path string, first, second []byte) {
	t.Helper()
	a, err := res.OpenBytes(arch)
	if err != nil {
		t.Fatalf("OpenBytes: %v", err)
	}
	entries := a.Entries()
	if len(entries) != 2 {
		t.Fatalf("Entries() = %+v, want 2 entries", entries)
	}
	for i, e := range entries {
		if e.Path != path {
			t.Errorf("Entries()[%d].Path = %q, want %q", i, e.Path, path)
		}
	}
	if int(entries[0].Size) != len(first) || int(entries[1].Size) != len(second) {
		t.Fatalf("Entries() sizes = %d, %d, want %d, %d (node order)",
			entries[0].Size, entries[1].Size, len(first), len(second))
	}
	got, err := a.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if !bytes.Equal(got, first) {
		t.Errorf("ReadFile(%q) = %q, want the FIRST record's bytes %q", path, got, first)
	}
	// The second record has no key of its own: nothing distinguishes it from the
	// first at the lookup, which is what "indexed but unreachable" means.
	if bytes.Equal(got, second) {
		t.Errorf("ReadFile(%q) returned the second record's bytes; the index must bind the first", path)
	}
}

// TestOpenEmptyArchive - AC-9.
func TestOpenEmptyArchive(t *testing.T) {
	a, err := res.OpenBytes(emptyArchive())
	if err != nil {
		t.Fatalf("OpenBytes(empty): unexpected error: %v", err)
	}
	if a == nil {
		t.Fatal("OpenBytes(empty): nil archive without error")
	}
	entries := a.Entries()
	if entries == nil {
		t.Fatal("Entries(): got nil, want a non-nil length-0 slice")
	}
	if len(entries) != 0 {
		t.Fatalf("Entries(): got %d entries, want 0: %+v", len(entries), entries)
	}
}

func FuzzOpenBytes(f *testing.F) {
	f.Add(nestedArchive())
	f.Add(concat(nestedArchive(), stalePackerResidue)) // a registry with residue
	f.Add(singleFileArchive())
	f.Add(emptyArchive())
	f.Add([]byte{})
	f.Add([]byte{0x26, 0x59, 0x41, 0x31}) // signature only, truncated

	corrupt := singleFileArchive()
	binary.LittleEndian.PutUint32(corrupt[0x00:], 0) // wrong signature
	f.Add(corrupt)

	f.Fuzz(func(t *testing.T, data []byte) {
		a, err := res.OpenBytes(data)
		if err != nil {
			if a != nil {
				t.Fatalf("error result must carry a nil *Archive, got non-nil")
			}
			return
		}
		if a == nil {
			t.Fatal("nil error must carry a non-nil *Archive")
		}
		// Exercising the public API must not panic either.
		for _, e := range a.Entries() {
			_, _ = a.ReadFile(e.Path)
		}
		_, _ = a.ReadFile("definitely/missing/path")
	})
}
