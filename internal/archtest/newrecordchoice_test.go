package archtest

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewRecordChoiceHasOneTable pins that the create-record choices have one
// table: each refusal word and each retired target flag has one home in
// pkg/ui/mapeditorscriptmodel.go, and no non-test source sets a flag.
func TestNewRecordChoiceHasOneTable(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	const home = "pkg/ui/mapeditorscriptmodel.go"
	words := map[string]int{
		"a new global is added on the Globals page":               0,
		"a new message is written in the window's message editor": 0,
	}
	retired := []string{"NewMessage bool", "NewGlobal bool", "NewMessage: true", "NewGlobal: true"}
	err = filepath.WalkDir(filepath.Join(root, "pkg"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		for w := range words {
			if n := strings.Count(string(b), w); n > 0 {
				if rel != home {
					t.Errorf("%s repeats the create refusal %q; the table in %s is its one home", rel, w, home)
				}
				words[w] += n
			}
		}
		for _, r := range retired {
			if strings.Contains(string(b), r) {
				t.Errorf("%s sets the retired target flag %q; a target carries NewChoice", rel, r)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for w, n := range words {
		if n != 1 {
			t.Errorf("refusal %q appears %d times, want 1", w, n)
		}
	}
}
