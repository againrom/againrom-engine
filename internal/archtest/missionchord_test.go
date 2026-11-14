package archtest

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A row's chord is built only in rowChord.
func TestMissionBindingChordHasOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "pkg", "ui")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	row := `(\.(ctrl|shift|alt)\b|modHeld|modAny|modNone)`
	forms := []*regexp.Regexp{
		regexp.MustCompile(`(?s)MissionKeyChord\{[^}]*` + row),
		regexp.MustCompile(`\.(Ctrl|Shift|Alt)\s*=[^=\n][^\n]*` + row),
	}
	hits := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, re := range forms {
			n += len(re.FindAllIndex(src, -1))
		}
		if n == 0 {
			continue
		}
		if entry.Name() != "playermissionkeys.go" {
			t.Errorf("binding row chord built outside rowChord: %s", entry.Name())
		}
		hits += n
	}
	if hits != 1 {
		t.Fatalf("binding row chord builders=%d, want one (rowChord)", hits)
	}
}
