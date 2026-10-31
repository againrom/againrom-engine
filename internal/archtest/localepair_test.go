package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestLocalePairLivesOnlyInTheLocaleTable: no production file outside
// pkg/locale maps one language code to the other.
func TestLocalePairLivesOnlyInTheLocaleTable(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	sites, err := LocalePairSites(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range sites {
		t.Errorf("%s maps a language code to the other; use locale.Other", s)
	}
}

// TestLocalePairScanSeesEachShape runs the scan on a synthetic source.
func TestLocalePairScanSeesEachShape(t *testing.T) {
	const src = `package x

var a = map[string]string{"en": "ru", "ru": "en"}
var b = map[string]string{
	"ru": "en",
}
var c = map[string]string{"en": "en", "de": "ru", "en2": "ru"}
var d = map[string]int{"en": 1}
`
	got := localePairInSource("x.go", []byte(src), map[string]bool{"en": true, "ru": true})
	want := "x.go:3 x.go:3 x.go:5"
	if strings.Join(got, " ") != want {
		t.Fatalf("hits = %v, want %s", got, want)
	}
}
