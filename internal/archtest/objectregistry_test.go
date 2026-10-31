package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestObjectRegistryIsReadOnlyByTheCatalog(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	sites, err := ObjectRegistrySites(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range sites {
		t.Errorf("%s names objects.reg; read it through game.LoadObjectCatalog", s)
	}
	for file := range ObjectRegistryAllowed {
		if _, err := os.Stat(root + "/" + file); err != nil {
			t.Errorf("allowed file %s: %v", file, err)
		}
	}
}

func TestObjectRegistryScanSeesEachShape(t *testing.T) {
	const src = `package x

// objects.reg in a comment is not a read.
const a = "gfx/objects/objects.reg"
var b = []string{"x", "OBJECTS.REG"}
const c = "objects"
`
	got := strings.Join(objectRegistryInSource("x.go", []byte(src)), " ")
	if got != "x.go:4 x.go:5" {
		t.Fatalf("hits = %s", got)
	}
}
