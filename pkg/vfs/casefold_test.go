package vfs

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"againrom/pkg/installpath"
)

// TestLooseTierResolvesWhateverTheCase reads loose files whose names on a
// case-sensitive file system are upper or mixed case through their folded
// address, and reports a directory holding two spellings of one name.
func TestLooseTierResolvesWhateverTheCase(t *testing.T) {
	disk := fstest.MapFS{
		"TOMB.ALM":          {Data: []byte("tomb")},
		"Kids.alm":          {Data: []byte("kids")},
		"Data/SubDir/X.BIN": {Data: []byte("x")},
		"Pair.alm":          {Data: []byte("a")},
		"PAIR.ALM":          {Data: []byte("b")},
	}
	// Loss control: the folded address as a host name does not open here.
	if _, err := disk.Open("tomb.alm"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("loss control: tomb.alm opened: %v", err)
	}
	f := &FS{dirs: []func() *installpath.Resolver{func() *installpath.Resolver { return installpath.NewFS(disk) }}}
	for address, want := range map[string]string{"Tomb.ALM": "tomb", "kids.alm": "kids", `data\subdir\x.bin`: "x"} {
		b, err := f.ReadFile(address)
		if err != nil || string(b) != want {
			t.Fatalf("ReadFile(%q) = %q, %v; want %q", address, b, err, want)
		}
		if src, ok := f.Locate(address); !ok || src.Tier != TierDir {
			t.Fatalf("Locate(%q) = %+v, %v", address, src, ok)
		}
	}
	if _, err := f.ReadFile("absent.alm"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ReadFile(absent) = %v", err)
	}
	if _, err := f.ReadFile("pair.alm"); !errors.Is(err, installpath.ErrAmbiguous) {
		t.Fatalf("ReadFile(pair.alm) = %v, want both spellings reported", err)
	}
	if src, ok := f.Locate("pair.alm"); !ok || src.Tier != TierDir {
		t.Fatalf("Locate(pair.alm) = %+v, %v; the directory holding the pair is the source", src, ok)
	}
}
