package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPresentationProbeHasOneWindowBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(root, "pkg", "game", "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	calls := map[string]int{"ebiten.RunGameWithOptions(": 0, "ebiten.SetWindowPosition(": 0}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for call := range calls {
			n := strings.Count(string(data), call)
			calls[call] += n
			if n != 0 && filepath.Base(file) != "presentationprobe_test.go" {
				t.Errorf("%s has a presentation window builder outside the shared probe", file)
			}
		}
	}
	for call, n := range calls {
		if n != 1 {
			t.Errorf("%s: got %d calls, want one in the shared probe", call, n)
		}
	}
}
