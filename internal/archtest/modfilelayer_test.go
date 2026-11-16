package archtest

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestModFileLayerHasOneBuilder reads the production sources of pkg/game. The
// layer is one composite literal, in newModFileLayer, and the only production
// files that build a layer are the launch share and the front end's getter.
func TestModFileLayerHasOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	dir := filepath.Join(root, "pkg", "game")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var literal, builds []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "&modFiles{") {
			literal = append(literal, name)
		}
		if strings.Contains(string(data), "newModFileLayer(") {
			builds = append(builds, name)
		}
	}
	if !slices.Equal(literal, []string{"modfiles.go"}) {
		t.Errorf("the mod file layer literal is written in %v; it is written only in modfiles.go", literal)
	}
	if !slices.Equal(builds, []string{"installshare.go", "modfiles.go"}) {
		t.Errorf("the layer is built in %v; the launch share and modFileLayer call newModFileLayer and nothing else builds one", builds)
	}
}
