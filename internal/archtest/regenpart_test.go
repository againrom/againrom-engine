package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegenPartShareHasOneHome(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "pkg", "sim")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, ent := range entries {
		name := ent.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		src := string(b)
		for _, banned := range []string{"part.last()", ".share(", "part.j+1", "part.j + 1"} {
			if strings.Contains(src, banned) && name != "regen.go" {
				t.Errorf("%s builds a part itself (%q)", name, banned)
			}
		}
		if name == "regen.go" && (strings.Contains(src, ".share(") || strings.Contains(src, "func (p regenPart) last")) {
			t.Errorf("regen.go keeps a second route")
		}
		if name == "regen.go" && strings.Count(src, "func partShare(") != 1 {
			t.Errorf("regen.go must declare the share once as partShare")
		}
		if name == "regen.go" && (strings.Contains(src, "gain*(p.j+1)") || strings.Contains(src, "Mul(&gain, big.NewInt(part.j")) {
			t.Errorf("regen.go keeps a share beside partShare")
		}
		if name != "regen.go" && strings.Contains(src, "func (p regenPart)") {
			t.Errorf("%s declares a regenPart method", name)
		}
	}
}
