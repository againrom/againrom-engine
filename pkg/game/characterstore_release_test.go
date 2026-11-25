package game

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/a2c"
)

// The installed game's character store row decodes every character file the
// install's root holds, reading only: none is left out, a second-game file's
// identity pair is its file name, and each listed file keeps its bytes.
func TestReleaseCharacterStoreDecodesTheInstalledCharacterFiles(t *testing.T) {
	f := releaseFront(t)
	root := f.Archives.Root
	row := filesOf(f.Archives.Game()).characters
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	before := map[string]string{}
	for _, e := range entries {
		if e.Type().IsRegular() && strings.EqualFold(filepath.Ext(e.Name()), row.ext) {
			b, err := os.ReadFile(filepath.Join(root, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			before[e.Name()] = string(b)
		}
	}
	files, left, err := listCharacters(root, row)
	if err != nil {
		t.Fatal(err)
	}
	if left != 0 || len(files) != len(before) {
		t.Fatalf("%d %s files: listed %d, left out %d", len(before), row.ext, len(files), left)
	}
	for _, file := range files {
		raw := before[file.File]
		if len(file.Name) == 0 {
			t.Errorf("%s: empty name", file.File)
		}
		if row.ext == ".a2c" {
			id, err := a2c.ReadIdentity([]byte(raw))
			if err != nil || fmt.Sprintf("%d%d.a2c", id.ID0, id.ID1) != file.File {
				t.Errorf("%s: identity %d %d, %v", file.File, id.ID0, id.ID1, err)
			}
		}
		if after, err := os.ReadFile(filepath.Join(root, file.File)); err != nil || string(after) != raw {
			t.Errorf("%s changed while listed: %v", file.File, err)
		}
		t.Logf("%s %q unix time %d", file.File, file.Name, file.Time.Unix())
	}
	t.Logf("%s: %d %s files listed, %d left out", f.Archives.Game(), len(files), row.ext, left)
}
