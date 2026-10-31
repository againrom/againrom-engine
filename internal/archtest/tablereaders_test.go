package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestTableReadersLiveOnlyInTheirBuilders(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	sites, err := TableReaderSites(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range sites {
		t.Errorf("%s reads the terrain cost table or parses a definition table itself; use pkg/game/table.go", s)
	}
	for file := range TableReadersAllowed {
		if _, err := os.Stat(root + "/" + file); err != nil {
			t.Errorf("allowed file %s: %v", file, err)
		}
	}
}

func TestTableReaderScanSeesEachShape(t *testing.T) {
	const src = `package x

import "againrom/pkg/formats/databin"

// data/map.reg in a comment is not a read.
const a = "world/data/map.reg"
var _, _ = databin.Parse(nil)
var _, _ = databin.ParseWith(nil, databin.ROM1Layout)
`
	got := strings.Join(tableReadersInSource("x.go", []byte(src)), " ")
	if got != "x.go:6 x.go:7" {
		t.Fatalf("hits = %s", got)
	}
}
