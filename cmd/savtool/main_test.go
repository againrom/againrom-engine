package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
)

// synthetic builds a whole save in test code from the format contract. No test
// here reads a game install; the evidence against real files is in the story's
// verification.md.
func synthetic() []byte {
	var b []byte
	le32 := func(v uint32) { b = binary.LittleEndian.AppendUint32(b, v) }
	le16 := func(v uint16) { b = binary.LittleEndian.AppendUint16(b, v) }
	le32(0x50)
	le32(0x05)
	b = append(b, 5)
	b = append(b, "t.alm"...)
	for i := 0; i < 11; i++ {
		le32(0)
	}
	le32(7) // mission
	le32(2) // difficulty
	le32(6)
	le32(1) // one player
	le16(0xffff)
	le16(1)
	le16(uint16(len("Player")))
	b = append(b, "Player"...)
	b = append(b, 4)
	b = append(b, "Hero"...)
	le16(1)
	le32(1)
	b = append(b, 1, 2, 3, 4, 5, 6, 7, 8)
	b = append(b, 0x02)
	le32(0) // participant: the human
	le16(2)
	le32(100 ^ 0x5c073f4d) // money
	b = append(b, 0, 1)    // outcome, +0x3d
	le32(0 ^ 0x5c073f4d)
	le32(0)
	le16(0)
	le16(0)
	le32(0)
	le32(0)
	le32(0)
	le32(1) // Player group
	b = append(b, make([]byte, 2+80+2)...)
	le32(10)
	for i := 0; i < 10; i++ {
		if i == 0 {
			le16(0xffff)
			le16(1)
			le16(4)
			b = append(b, "Unit"...)
		} else {
			le16(0x8003)
		}
		le16(uint16(0x0500 + i*0x101))
		le16(uint16(0x0500 + i*0x101))
		b = append(b, 128, 128)
		le16(0x0777)
		le32(0x1234a020)
		le32(uint32(i + 1))
		b = append(b, 1)
		le16(0)
		le32(uint32(20 + i))
		le16(0)
		le32(0)
		le32(uint32(100 + i))
		le32(0)
		start := len(b)
		b = append(b, make([]byte, 4+2+2+462+2+19+2+2+1+55+2+1+1+17)...)
		binary.LittleEndian.PutUint16(b[start+4+2+2+462+2+19+2+2+1+16:], 1)
	}
	b = append(b, make([]byte, 12+32+2+2+4)...)
	le32(0) // dead list
	b = append(b, 1)
	le32(3) // Buildings
	// Three Building instances: a fixed-length class, written consecutively,
	// so the reader can WALK them rather than scan for them.
	le16(0xffff)
	le16(1)
	le16(uint16(len("Building")))
	b = append(b, "Building"...)
	for i := 0; i < 3; i++ {
		if i > 0 {
			le16(0x800e)
		}
		le16(uint16(0x0500 + i*0x101))
		le16(uint16(0x0500 + i*0x101))
		b = append(b, 0x80, 0x80)
		le16(0x0777)
		le32(0x1234a020)
		// Creation order continues PAST the ten heads below: the reader's
		// calibration rejects a group whose runtime ids repeat, and in a
		// real save every object's id is its own.
		le32(uint32(i + 11))
		b = append(b, 0x11)
		le16(0x0022)
		le32(uint32(0x00330000 + i + 1))
		le16(0x0044)
		le32(0x55555555)
		le32(uint32(0x02c90000 + i*0x80))
		le32(0x02c8f4c0)
		for j := 0; j < 22; j++ {
			b = append(b, byte(0x60+j))
		}
		b = append(b, 0x01)
		le16(1000)
		le16(1000)
		le16(7)
		b = append(b, 0x02, 0x03, 0x04)
		le32(506)
		le32(511)
	}
	le32(0) // SpellEffects
	// The world half: two block records, one cell record, the session block.
	le16(2)
	le32(0x0810<<16 | 0x41<<8 | 0x21)
	le32(0x0a20<<16 | 0xc5<<8 | 0x05)
	le16(1)
	rec := make([]byte, 54)
	binary.LittleEndian.PutUint16(rec, 0x0810)
	b = append(b, rec...)
	b = append(b, 0xde, 0xad, 0xbe, 0xef)
	b = append(b, make([]byte, 4374)...)
	le32(0) // Sacks
	le32(0xbadface1)
	le32(0)
	b = append(b, make([]byte, 400)...)
	if len(b)%2 != 0 {
		b = append(b, 0)
	}

	blob := sav.Compress(b)
	out := make([]byte, 16)
	copy(out, sav.Magic)
	binary.LittleEndian.PutUint32(out[4:], uint32(16+len(blob)))
	binary.LittleEndian.PutUint32(out[8:], sav.MinVersion)
	binary.LittleEndian.PutUint32(out[12:], uint32(len(blob)))
	out = append(out, blob...)
	region := make([]byte, 0x100)
	copy(region, "one\x00was longer than this")
	out = append(out, region...)
	return append(out, "&YA1"...)
}

func write(t *testing.T, name string, b []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestVerbsReportWhatIsInTheFile(t *testing.T) {
	path := write(t, "game0000.sav", synthetic())
	for _, c := range []struct {
		args []string
		want []string
	}{
		{[]string{"info", path}, []string{"mission 7", "players 1", "HUMAN",
			"money 100", "blocks 2", "10 owner-graph actors"}},
		{[]string{"blocks", path}, []string{"DELTA", "dyn 0xc5"}},
		{[]string{"actors", path}, []string{"10 owner-graph actors", "unit 20"}},
		{[]string{"session", path}, []string{"won 0", "of 1000 fire-once"}},
		{[]string{"verify", path}, []string{"identical", "1/1 byte-identical"}},
		{[]string{"objects", path}, []string{"classes introduced: Player Unit Building",
			"3 distinct of 3", "3 Building instance(s), 3 re-encoded identically"}},
		{[]string{"campaign", path}, []string{"no campaign"}},
	} {
		t.Run(c.args[0], func(t *testing.T) {
			var out bytes.Buffer
			if err := run(c.args, &out); err != nil {
				t.Fatalf("%v: %v", c.args, err)
			}
			for _, w := range c.want {
				if !strings.Contains(out.String(), w) {
					t.Fatalf("output does not mention %q:\n%s", w, out.String())
				}
			}
		})
	}
}

func TestSetWritesOutOfPlaceAndOnlyWhereTold(t *testing.T) {
	in := synthetic()
	path := write(t, "game0000.sav", in)
	outPath := filepath.Join(filepath.Dir(path), "out.sav")
	var out bytes.Buffer
	err := run([]string{"set", "-out", outPath, "-money", "4242",
		"-label", "two", "-actor", "3:40:41", path}, &out)
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	// The input is untouched — the corpus this reads is evidence and a tool
	// that could overwrite it is one slip from destroying it.
	again, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(again, in) {
		t.Fatal("set wrote over its input")
	}
	b, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	f, err := sav.Open(b)
	if err != nil {
		t.Fatalf("the written file does not read back: %v", err)
	}
	if f.Players[0].Money != 4242 || string(f.Label) != "two" {
		t.Fatalf("money %d label %q", f.Players[0].Money, f.Label)
	}
	// -actor is INDEX:COL:ROW, so 3:40:41 is column 40 and row 41.
	if a := f.Actors[3]; a.Col() != 40 || a.Row() != 41 {
		t.Fatalf("actor 3 at (%d,%d)", a.Col(), a.Row())
	}
	if len(b) != len(in) {
		t.Fatalf("the file changed length: %d -> %d", len(in), len(b))
	}
}

func TestRunRefusesWhatItCannotDo(t *testing.T) {
	path := write(t, "game0000.sav", synthetic())
	for _, args := range [][]string{
		nil,
		{"nonsense", path},
		{"info"},
		{"info", path, path},
		{"objects"},
		{"objects", "-class", "Unit", path},
		{"verify"},
		{"set", path}, // no -out
		{"set", "-out", "x", "-actor", "1:2", path},         // too few fields
		{"set", "-out", "x", "-actor", "1:2:3:4:5:6", path}, // too many
		{"set", "-out", "x", "-latch", "1:x", path},         // not a number
		{"set", "-out", "x", "-diplomacy", "1:-2:3", path},
		{"set", "-out", "x", "-outcome", "9", path},
		{"info", filepath.Join(t.TempDir(), "absent.sav")},
	} {
		var out bytes.Buffer
		if err := run(args, &out); err == nil {
			t.Fatalf("%v was accepted", args)
		}
	}
}

func TestVerifyReportsAFileThatWillNotOpen(t *testing.T) {
	bad := append([]byte(nil), synthetic()...)
	bad[0] = 'X'
	var out bytes.Buffer
	err := run([]string{"verify", write(t, "bad.sav", bad)}, &out)
	if err == nil {
		t.Fatal("verify reported success over a file it could not open")
	}
	if !strings.Contains(out.String(), "OPEN FAILED") {
		t.Fatalf("verify said nothing about the failure:\n%s", out.String())
	}
}

func TestFieldsIsPureAndTotal(t *testing.T) {
	for _, c := range []struct {
		in     string
		lo, hi int
		want   []int
	}{
		{"1:2:3", 3, 3, []int{1, 2, 3}},
		{"7:0", 2, 5, []int{7, 0}},
	} {
		got, err := fields(c.in, c.lo, c.hi)
		if err != nil {
			t.Fatalf("%q: %v", c.in, err)
		}
		if len(got) != len(c.want) {
			t.Fatalf("%q gave %v", c.in, got)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("%q gave %v, want %v", c.in, got, c.want)
			}
		}
	}
	for _, in := range []string{"", "1", "1:2:3:4", "1:-1", "a:1"} {
		if _, err := fields(in, 2, 3); err == nil {
			t.Fatalf("%q was accepted", in)
		}
	}
}

func TestFirstDiff(t *testing.T) {
	for _, c := range []struct {
		a, b []byte
		want int
	}{
		{[]byte{1, 2, 3}, []byte{1, 2, 3}, -1},
		{[]byte{1, 2, 3}, []byte{1, 9, 3}, 1},
		{[]byte{1, 2}, []byte{1, 2, 3}, 2},
		{nil, nil, -1},
	} {
		if got := firstDiff(c.a, c.b); got != c.want {
			t.Fatalf("firstDiff(% x, % x) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
