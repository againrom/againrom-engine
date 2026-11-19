package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestScriptOpHasOneDispatch keeps the script op bodies in one table: the
// second game's route names no banked flag and no second switch, and the
// table is read only by the two dispatches in script.go.
func TestScriptOpHasOneDispatch(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "pkg", "sim", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	gone := []string{"bankedScript", "rom2Check(", "rom2Instant(", "rom2CheckSupported", "rom2InstantSupported"}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		text := string(src)
		for _, name := range gone {
			if strings.Contains(text, name) {
				t.Errorf("%s holds %q, the second route the one dispatch replaced", filepath.Base(f), name)
			}
		}
		if strings.Contains(text, "scriptRows.") && filepath.Base(f) != "script.go" {
			t.Errorf("%s reads the script op table outside script.go", filepath.Base(f))
		}
	}
}
