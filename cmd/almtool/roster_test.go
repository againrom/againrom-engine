package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// rosterALM is minimalALM with a type-5 roster and type-6 placements written
// into it: n roster records of 76 bytes and one unit record of 70 bytes per
// entry of owners.
//
// EVERY BYTE IS WRITTEN HERE and nothing comes from a game file. The names are
// written as BYTES rather than as text, which is golden rule 2's rule for a
// CP866 field and is kept here even though these three names are ASCII — a
// fixture that spells one name as literal text teaches the next one to.
func rosterALM(t *testing.T, names [][]byte, rows [][16]uint16, owners []uint32) []byte {
	t.Helper()
	if len(names) != len(rows) {
		t.Fatalf("fixture: %d name(s) against %d relation row(s)", len(names), len(rows))
	}
	const (
		groupRec, unitRec = 76, 70
		nameOff, relOff   = 0x0c, 0x2c
		ownerOff          = 0x14
	)
	type5 := make([]byte, groupRec*len(names))
	for i, n := range names {
		rec := type5[i*groupRec : (i+1)*groupRec]
		if len(n) >= 0x20 {
			t.Fatalf("fixture: name %d is %d bytes, the field is 0x20 with a NUL", i, len(n))
		}
		copy(rec[nameOff:], n)
		for k, w := range rows[i] {
			binary.LittleEndian.PutUint16(rec[relOff+2*k:], w)
		}
	}
	type6 := make([]byte, unitRec*len(owners))
	for i, o := range owners {
		binary.LittleEndian.PutUint32(type6[i*unitRec+ownerOff:], o)
	}
	return rebuildALM(t, map[int][]byte{5: type5, 6: type6},
		map[int]uint32{0x1c: uint32(len(names)), 0x24: uint32(len(owners))})
}

// rebuildALM rewrites minimalALM's container with the payloads given for the
// type ids named, and the metadata counts patched. It walks the record headers
// rather than assuming an offset, so it survives a change to minimalALM.
func rebuildALM(t *testing.T, payloads map[int][]byte, counts map[int]uint32) []byte {
	t.Helper()
	base := minimalALM()
	const fileHdr, recHdr = 20, 20

	var out []byte
	out = append(out, base[:fileHdr]...)
	at := fileHdr
	for at+recHdr <= len(base) {
		hdr := base[at : at+recHdr]
		size := int(binary.LittleEndian.Uint32(hdr[8:12]))
		id := int(binary.LittleEndian.Uint32(hdr[12:16]))
		body := append([]byte(nil), base[at+recHdr:at+recHdr+size]...)
		if p, ok := payloads[id]; ok {
			body = append([]byte(nil), p...)
		}
		if id == 0 {
			for off, v := range counts {
				binary.LittleEndian.PutUint32(body[off:], v)
			}
		}
		newHdr := append([]byte(nil), hdr...)
		binary.LittleEndian.PutUint32(newHdr[8:12], uint32(len(body)))
		out = append(out, newHdr...)
		out = append(out, body...)
		at += recHdr + size
	}
	if at != len(base) {
		t.Fatalf("fixture: the container did not tile — %d of %d bytes walked", at, len(base))
	}
	return out
}

func writeALM(t *testing.T, stream []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "roster.alm")
	if err := os.WriteFile(path, stream, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// TestRosterVerbPrintsTheSlotsAndTheEffectiveMatrix is 0094 AC-12.
//
// The fixture is chosen so that every rule the verb states has a witness that
// only that rule produces:
//
//   - slot 2's row spells its own diagonal as 1, so the forced 2 is visible as an
//     OVERRIDE of the file rather than as agreement with it — five shipped rosters
//     do exactly this;
//   - slot 1 -> slot 3 carries 0x0101, whose low byte is 1 and whose raw word is
//     not, so the narrowing shows in the matrix while the raw column keeps the
//     word the file holds;
//   - the placements are 2 on slot 3, 1 on slot 1 and none on slot 2, so a count
//     that had been taken per record index rather than per owner word would come
//     out in the wrong row.
func TestRosterVerbPrintsTheSlotsAndTheEffectiveMatrix(t *testing.T) {
	names := [][]byte{{'S', 'e', 'l', 'f'}, {'M', 'o', 'b'}, {'F', 'o', 'e'}}
	var rows [][16]uint16
	rows = append(rows,
		[16]uint16{0, 0, 0x0101}, // slot 1: not hostile to 2, hostile to 3 through a wide word
		[16]uint16{1, 1, 0},      // slot 2: hostile to 1, and its own diagonal spelled 1
		[16]uint16{1, 0, 0})      // slot 3: hostile to 1 only — the asymmetry with slot 1
	path := writeALM(t, rosterALM(t, names, rows, []uint32{3, 3, 1}))

	out, err := captureStdout(t, func() error { return cmdRoster(path) })
	if err != nil {
		t.Fatalf("cmdRoster: %v", err)
	}
	for _, want := range []string{
		`roster (type5): 3 record(s), 3 placed unit(s)`,
		`   1  "Self"`,
		`   2  "Mob"`,
		`   3  "Foe"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the readout does not carry %q:\n%s", want, out)
		}
	}
	// The raw column keeps the file's word; the matrix keeps its low byte.
	if !strings.Contains(out, "[0 0 257 0 0 0 0 0 0 0 0 0 0 0 0 0]") {
		t.Errorf("slot 1's raw row lost the word the file holds:\n%s", out)
	}
	matrix := out[strings.Index(out, "<- to"):]
	for _, want := range []string{
		"\n     1   2   0   1\n", // the forced diagonal, the 0, and the narrowed 0x0101
		"\n     2   1   2   0\n", // slot 2's own 1 on the diagonal, overridden to 2
		"\n     3   1   0   2\n", // hostile to 1 while 1 is hostile to it: the mirror is 0
	} {
		if !strings.Contains(matrix, want) {
			t.Errorf("the effective matrix does not carry the row %q:\n%s", want, matrix)
		}
	}
	// The owner census is per owner WORD, so slot 2 owns nothing and slot 3 owns two.
	for _, want := range []string{`"Self"                               1`, `"Mob"                                0`,
		`"Foe"                                2`} {
		if !strings.Contains(out, want) {
			t.Errorf("the owner census does not carry %q:\n%s", want, out)
		}
	}
}

// A document with no roster prints its header and nothing else, and returns nil:
// a map may hold no type-5 record, and that is an answer rather than a failure.
func TestRosterVerbOnADocumentWithNoRoster(t *testing.T) {
	out, err := captureStdout(t, func() error { return cmdRoster(writeALM(t, minimalALM())) })
	if err != nil {
		t.Fatalf("cmdRoster: %v", err)
	}
	if want := "roster (type5): 0 record(s), 0 placed unit(s)\n"; !strings.HasPrefix(out, want) {
		t.Errorf("output %q, want it to open with %q", out, want)
	}
	if strings.Contains(out, "effective matrix") {
		t.Errorf("a rosterless document printed a matrix:\n%s", out)
	}
}

// A stream that will not decode goes through the shared verb error path, and
// nothing reaches standard output — the same contract cmdRoundtrip is held to.
func TestRosterVerbRejectsAnUndecodableStream(t *testing.T) {
	stream := minimalALM()
	stream[0] ^= 0xFF // break the magic
	out, err := captureStdout(t, func() error { return cmdRoster(writeALM(t, stream)) })
	if err == nil {
		t.Fatal("cmdRoster accepted a bad-magic stream")
	}
	if out != "" {
		t.Errorf("a refused stream printed %q", out)
	}
}
