package game

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/a2c"
	"againrom/pkg/formats/alch"
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

// Every character file the install's root holds reads whole through its
// format's reader, and the store's writer turns what it read into a file that
// reads back equal, reading only. The game's own files are owner-made
// originals; an install without one logs that it found none.
func TestReleaseInstalledCharacterFilesReadAndWriteBack(t *testing.T) {
	f := releaseFront(t)
	root := f.Archives.Root
	row := filesOf(f.Archives.Game()).characters
	roundTrip := map[string]func([]byte) (any, any, error){
		".chr": func(b []byte) (any, any, error) {
			c, err := alch.Read(b)
			if err != nil {
				return nil, nil, err
			}
			w, err := alch.Write(c)
			if err != nil {
				return nil, nil, err
			}
			again, err := alch.Read(w)
			return c, again, err
		},
		".a2c": func(b []byte) (any, any, error) {
			c, err := a2c.Read(b)
			if err != nil {
				return nil, nil, err
			}
			w, err := a2c.Write(c)
			if err != nil {
				return nil, nil, err
			}
			again, err := a2c.Read(w)
			return c, again, err
		},
	}[row.ext]
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	read := 0
	for _, e := range entries {
		if !e.Type().IsRegular() || !strings.EqualFold(filepath.Ext(e.Name()), row.ext) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		in, out, err := roundTrip(b)
		if err != nil || !reflect.DeepEqual(in, out) {
			t.Errorf("%s: read %+v, written and read %+v, %v", e.Name(), in, out, err)
		}
		read++
	}
	t.Logf("%s: %d %s files read and written back", f.Archives.Game(), read, row.ext)
}
