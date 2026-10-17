package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// -new with -out writes the new map to a new file: the constructor's bytes,
// single-player, with its drop cell at the centre (DIV-3597). It prints what
// the map reports; -new alone checks the same map without a file.
func TestReleaseNewWritesTheBlankMapAndPrintsItsReport(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: needs lawful install")
	}
	path := filepath.Join(t.TempDir(), "blank.alm")
	var out bytes.Buffer
	if err := run([]string{"-assets", root, "-new", "48x64", "-out", path}, &out); err != nil {
		t.Fatalf("-new -out: %v", err)
	}
	want, err := mapedit.NewMap(alm.NewMapSpec{Width: 48, Height: 64, Tile: 0x11, Altitude: 64})
	if err != nil {
		t.Fatal(err)
	}
	if err := want.SetWord70(1); err != nil {
		t.Fatal(err)
	}
	if err := want.SetDropCell(24, 32); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, want.Bytes()) {
		t.Fatalf("the file is not the constructor's bytes with the mode and the drop cell (%v)", err)
	}
	for _, want := range []string{"mapedit: wrote ", "warning victory", "report: 0 errors, 1 warnings"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, out.String())
		}
	}

	// The written file opens as any map and reports the same.
	var again bytes.Buffer
	if err := run([]string{"-assets", root, "-map", path, "-check"}, &again); err != nil {
		t.Fatalf("-check of the written map: %v", err)
	}
	if !strings.Contains(again.String(), "report: 0 errors, 1 warnings") || !strings.Contains(again.String(), "48x64") {
		t.Errorf("-check of the written map:\n%s", again.String())
	}

	// The same map checked without a file; its drop action is a record.
	var fresh bytes.Buffer
	if err := run([]string{"-assets", root, "-new", "48x64", "-check"}, &fresh); err != nil {
		t.Fatalf("-new -check: %v", err)
	}
	if !strings.Contains(fresh.String(), "new/48x64 48x64; 13 records") {
		t.Errorf("-new -check:\n%s", fresh.String())
	}

	// A file that exists is not replaced, and a size the constructor refuses
	// writes nothing.
	if err := run([]string{"-assets", root, "-new", "48x64", "-out", path}, &out); err == nil {
		t.Error("a second -out replaced the file")
	}
	other := filepath.Join(t.TempDir(), "bad.alm")
	if err := run([]string{"-assets", root, "-new", "31x32", "-out", other}, &out); err == nil {
		t.Error("a 31x32 map was written")
	}
	if _, err := os.Stat(other); err == nil {
		t.Error("a refused size left a file")
	}
}
