package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
)

// The script verb reads the extended script record: its head, and each
// condition's group.
func TestScriptVerbReadsTheExtendedRecord(t *testing.T) {
	// The type-7 payload grows to its three empty counted arrays.
	base := minimalALM()
	for off := 20; off+20 <= len(base); off += 20 + int(binary.LittleEndian.Uint32(base[off+8:])) {
		if binary.LittleEndian.Uint32(base[off+12:]) == 7 {
			binary.LittleEndian.PutUint32(base[off+8:], 12)
			base = slices.Concat(base[:off+20], make([]byte, 12), base[off+24:])
			break
		}
	}
	stream, err := alm.AppendScriptRecord(base, alm.ScriptRecord{Variables: 150, Triggers: []alm.ScriptTrigger{{
		Name: "T", Group: []uint32{0, 1}, Left: []uint32{1, 2}, Right: []uint32{1, 2}, Cmp: []uint32{0, 0}, Acts: []uint32{1},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "x.almx")
	if err := os.WriteFile(path, stream, 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := captureStdout(t, func() error { return cmdScript(path) })
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"extended script record typeId 100: 150 variable(s), 1 trigger(s)", "pair 0 (group 0)", "pair 1 (group 1)"} {
		if !strings.Contains(out, want) {
			t.Errorf("the listing lacks %q:\n%s", want, out)
		}
	}
}
