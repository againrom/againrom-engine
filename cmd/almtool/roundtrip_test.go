package main

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// minimalALM builds the smallest accepted synthetic stream from the container
// grammar: a 20-byte file header, then ten tag-7 records in typeId order 0..9,
// each a 20-byte header plus payload, tiling exactly to EOF. The map is 2x2
// with no type-4/5/6 content; type-7/9 hold only their count words and type-8
// is empty. Every byte is written here — nothing comes from a game file.
func minimalALM() []byte {
	const w, h = 2, 2
	var payloads [10][]byte
	payloads[0] = make([]byte, 632) // type-0 metadata: W, H, all counts zero
	binary.LittleEndian.PutUint32(payloads[0][0x00:], w)
	binary.LittleEndian.PutUint32(payloads[0][0x04:], h)
	payloads[1] = make([]byte, 2*w*h) // tiles, u16 cells
	payloads[2] = make([]byte, w*h)   // altitudes
	payloads[3] = make([]byte, w*h)   // overlay
	payloads[7] = make([]byte, 4)     // type-7: the entryCount word alone
	payloads[9] = make([]byte, 4)     // type-9: the count word alone

	var buf []byte
	u32 := func(v uint32) {
		var word [4]byte
		binary.LittleEndian.PutUint32(word[:], v)
		buf = append(buf, word[:]...)
	}
	u32(0x0052374D) // magic "M7R\0"
	u32(20)         // file hdrLen
	u32(0xDEADBEEF) // dataSize: uninterpreted, any value accepted
	u32(10)         // recordCount
	u32(990)        // formatVersion: any value but 1000
	for id, p := range payloads {
		u32(7)              // record tag
		u32(20)             // record hdrLen
		u32(uint32(len(p))) // payloadSize
		u32(uint32(id))     // typeId
		u32(0x3f800000)     // per-map constant word: uninterpreted bits
		buf = append(buf, p...)
	}
	return buf
}

// captureStdout runs f with os.Stdout redirected into a pipe and returns what
// it printed alongside f's error.
func captureStdout(t *testing.T, f func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()
	callErr := f()
	os.Stdout = orig
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return string(out), callErr
}

// TestRoundtripVerbAccept drives the verb over an accepted synthetic stream in
// a temp dir: nil error and the identity report — size and MD5 digest of the
// input bytes.
func TestRoundtripVerbAccept(t *testing.T) {
	stream := minimalALM()
	path := filepath.Join(t.TempDir(), "min.alm")
	if err := os.WriteFile(path, stream, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	out, err := captureStdout(t, func() error { return cmdRoundtrip(path) })
	if err != nil {
		t.Fatalf("cmdRoundtrip: %v", err)
	}
	want := fmt.Sprintf("roundtrip: identical, %d bytes, md5 %x\n", len(stream), md5.Sum(stream))
	if out != want {
		t.Errorf("output %q, want %q", out, want)
	}
}

// TestRoundtripVerbReject drives the verb over a rejected stream (the accepted
// fixture with its magic corrupted): an error through the verb error path and
// nothing on stdout.
func TestRoundtripVerbReject(t *testing.T) {
	stream := minimalALM()
	stream[0] ^= 0xFF // break the magic: the container's first rejection clause
	path := filepath.Join(t.TempDir(), "bad.alm")
	if err := os.WriteFile(path, stream, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	out, err := captureStdout(t, func() error { return cmdRoundtrip(path) })
	if err == nil {
		t.Fatalf("cmdRoundtrip accepted a bad-magic stream")
	}
	if out != "" {
		t.Errorf("rejection wrote %q to stdout, want the error path only", out)
	}
}

// TestFirstDiff pins the comparison the corpus run trusts: -1 on equal
// buffers, the first differing offset on unequal ones, the shorter length on
// length divergence.
func TestFirstDiff(t *testing.T) {
	cases := []struct {
		name string
		a, b []byte
		want int
	}{
		{"equal", []byte{1, 2, 3}, []byte{1, 2, 3}, -1},
		{"equal empty", nil, nil, -1},
		{"unequal mid", []byte{1, 2, 3, 4}, []byte{1, 2, 9, 4}, 2},
		{"unequal first byte", []byte{0}, []byte{1}, 0},
		{"prefix a shorter", []byte{1, 2}, []byte{1, 2, 3}, 2},
		{"prefix b shorter", []byte{1, 2, 3}, []byte{1}, 1},
	}
	for _, tc := range cases {
		if got := firstDiff(tc.a, tc.b); got != tc.want {
			t.Errorf("%s: firstDiff = %d, want %d", tc.name, got, tc.want)
		}
	}
}
