package archtest

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func localeTestRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	return root
}

// The live tree names no language outside the table.
func TestLocalePolicyLivesOnlyInTheLocaleTable(t *testing.T) {
	sites, err := LocalePolicySites(localeTestRoot(t))
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range sites {
		t.Errorf("%s; read the pkg/locale table instead", s)
	}
}

// No file maps one code to the other.
func TestLocalePairLivesOnlyInTheLocaleTable(t *testing.T) {
	sites, err := LocalePairSites(localeTestRoot(t))
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range sites {
		t.Errorf("%s maps a language code to the other; use locale.Other", s)
	}
}

func TestLocaleExemptPackagesNameNoLanguage(t *testing.T) {
	root := localeTestRoot(t)
	for _, top := range []string{"pkg/sim", "pkg/data"} {
		err := walkProductionGo(root, top, func(rel string, src []byte) {
			f, err := parser.ParseFile(token.NewFileSet(), rel, src, parser.ImportsOnly)
			if err != nil {
				t.Errorf("%s: %v", rel, err)
				return
			}
			for _, im := range f.Imports {
				if strings.Contains(im.Path.Value, "pkg/locale") {
					t.Errorf("%s imports pkg/locale, so its Selector tests are not exempt", rel)
				}
			}
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestLocaleValuesComeFromTheTable(t *testing.T) {
	v := TableLocaleValues()
	for _, w := range []string{"en", "ru", "english", "russian", "rom1-en", "rom1-ru", "-en", "-ru"} {
		if !v.Words[w] {
			t.Errorf("table values lack %q", w)
		}
	}
	if !v.Pages[866] || !v.Pages[1251] || v.Pages[0] || len(v.Pages) != 2 {
		t.Errorf("table pages = %v", v.Pages)
	}
	if v.Words[""] || v.Words["-"] {
		t.Error("an empty value entered the table values")
	}
}

func TestLocalePairScanSeesEachShape(t *testing.T) {
	const src = `package x

var a = map[string]string{"en": "ru", "ru": "en"}
var b = map[string]string{
	"ru": "en",
}
var c = map[string]string{"en": "en", "de": "ru", "en2": "ru"}
var d = map[string]int{"en": 1}
`
	got := localePairInSource("x.go", []byte(src), TableLocaleValues().Codes)
	want := "x.go:3 x.go:3 x.go:5"
	if strings.Join(got, " ") != want {
		t.Fatalf("hits = %v, want %s", got, want)
	}
}

// One planted instance of each shape, and look-alikes that stay clean.
func TestLocalePolicyScanSeesEachShape(t *testing.T) {
	const src = `package x

import "fmt"

var code = "en"
var entry = "russian"
var base = "rom1-ru"
var suffix = "-ru"
var inList = []string{"en", "ru"}
var keyed = map[string]string{"ru": "x"}
var path = "text/ru/strings.toml"
var format = fmt.Sprintf("%s", "text/%s/strings.toml")

func f(p profile, selector int, c TextCode) {
	if p.Language == "russian" {
	}
	if p.Selector == 1 || p.Selector >= 1 {
	}
	if 0 != p.CodePage || p.WindowsCodePage < 1252 {
	}
	if c.From != 1251 || c.To == 866 {
	}
	if selector == 1 {
	}
	switch selector {
	case 0:
	}
	switch p.CodePage {
	case 866:
	}
}

// "en" and "ru" in a comment are not literals.
func clean(p profile, selector int, n int) {
	if selector < 0 || p.Selector == n || n == 865 || p.Name == "enx" {
	}
	switch n {
	case 1:
	}
	_ = "text/enx/strings.toml"
	_ = "a/b"
}
`
	got := localePolicyInSource("x.go", []byte(src), TableLocaleValues())
	var lines []string
	for _, g := range got {
		parts := strings.SplitN(g, ":", 3)
		lines = append(lines, parts[1])
	}
	want := "5 6 7 8 9 9 10 11 15 17 17 19 19 21 21 23 26 29"
	if strings.Join(lines, " ") != want {
		t.Fatalf("lines = %v, want %s\n%s", lines, want, strings.Join(got, "\n"))
	}
}

func TestLocalePolicyScanSkipsExemptDirectoriesForFieldTests(t *testing.T) {
	const src = `package x

func f(d damage, selector int) {
	if d.Selector >= 5 || selector == 1 {
	}
	switch selector {
	case 0:
	}
	_ = "ru"
}
`
	vals := TableLocaleValues()
	if got := localePolicyInSource("pkg/sim/x.go", []byte(src), vals); len(got) != 1 || !strings.HasPrefix(got[0], "pkg/sim/x.go:9:") {
		t.Fatalf("exempt directory hits = %v", got)
	}
	if got := localePolicyInSource("pkg/game/x.go", []byte(src), vals); len(got) != 4 {
		t.Fatalf("ordinary directory hits = %v", got)
	}
}
