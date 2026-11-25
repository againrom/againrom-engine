package game

import (
	"cmp"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// The one lister keeps the files of its pattern in any letter case, passes
// over folders and other extensions, counts a file its row will not read,
// sorts by its row's order and stops at the first row when asked.
func TestProfileListerListsTheRowsPattern(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.TXT", "a.txt", "c.dat", "bad.txt", "skip.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "d.txt"), 0o700); err != nil {
		t.Fatal(err)
	}
	listing := profileListing[string]{dir: dir, exts: []string{".txt"}, regular: true, order: cmp.Compare[string],
		read: func(dir string, e os.DirEntry) (string, bool, error) {
			switch e.Name() {
			case "bad.txt":
				return "", false, errors.New("will not read")
			case "skip.txt":
				return "", false, nil
			}
			return e.Name(), true, nil
		}}
	rows, left, err := listProfileFiles(listing)
	if err != nil || left != 1 || !slices.Equal(rows, []string{"a.txt", "b.TXT"}) {
		t.Fatalf("rows %q, left %d, err %v", rows, left, err)
	}
	listing.first = true
	if rows, _, _ := listProfileFiles(listing); len(rows) != 1 {
		t.Fatalf("first listed %q", rows)
	}
	listing.dir = filepath.Join(dir, "absent")
	if rows, left, err := listProfileFiles(listing); rows != nil || left != 0 || err != nil {
		t.Fatalf("absent folder: %q %d %v", rows, left, err)
	}
}
