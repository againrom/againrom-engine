package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestResourceNamesFoldThroughNamefold(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	sites, err := HandFoldSites(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range sites {
		t.Errorf("%s folds ASCII case by hand; use pkg/namefold", s)
	}
	for file := range HandFoldAllowed {
		if _, err := os.Stat(root + "/" + file); err != nil {
			t.Errorf("allowed file %s: %v", file, err)
		}
	}
}

// Production code under pkg and cmd folds case through pkg/namefold, except in
// the functions CaseFoldAllowed names, each for something that is not a
// resource or host name.
func TestCaseFoldOnlyThroughNamefoldOrAllowed(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	sites, err := CaseFoldSites(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range CaseFoldUnallowed(sites, CaseFoldAllowed) {
		t.Errorf("%s:%d %s uses %s; fold a resource or host name through pkg/namefold, or name the function in CaseFoldAllowed with its reason", s.File, s.Line, s.Func, s.Use)
	}
	for _, k := range CaseFoldStale(sites, CaseFoldAllowed) {
		t.Errorf("CaseFoldAllowed entry %s folds nothing; remove it", k)
	}
	for k, reason := range CaseFoldAllowed {
		if strings.TrimSpace(reason) == "" || strings.Contains(reason, "\n") {
			t.Errorf("CaseFoldAllowed entry %s needs a one-line reason", k)
		}
	}
}

func caseFoldHits(name, src string) string {
	var out []string
	for _, s := range caseFoldInSource(name, []byte(src)) {
		out = append(out, s.Key()+" "+s.Use)
	}
	return strings.Join(out, ", ")
}

// One planted instance per shape: each is named, and a look-alike is not.
func TestCaseFoldScanSeesEachShape(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{"strings.ToLower", `package x
import "strings"
func f(s string) string { return strings.ToLower(s) }`, "x.go#f strings.ToLower"},
		{"strings.ToUpper", `package x
import "strings"
func f(s string) string { return strings.ToUpper(s) }`, "x.go#f strings.ToUpper"},
		{"strings.EqualFold", `package x
import "strings"
func f(a, b string) bool { return strings.EqualFold(a, b) }`, "x.go#f strings.EqualFold"},
		{"strings.ToTitle", `package x
import "strings"
func f(s string) string { return strings.ToTitle(s) }`, "x.go#f strings.ToTitle"},
		{"bytes.ToLower", `package x
import "bytes"
func f(b []byte) []byte { return bytes.ToLower(b) }`, "x.go#f bytes.ToLower"},
		{"bytes.ToUpper", `package x
import "bytes"
func f(b []byte) []byte { return bytes.ToUpper(b) }`, "x.go#f bytes.ToUpper"},
		{"bytes.EqualFold", `package x
import "bytes"
func f(a, b []byte) bool { return bytes.EqualFold(a, b) }`, "x.go#f bytes.EqualFold"},
		{"unicode.ToLower", `package x
import "unicode"
func f(r rune) rune { return unicode.ToLower(r) }`, "x.go#f unicode.ToLower"},
		{"unicode.ToUpper", `package x
import "unicode"
func f(r rune) rune { return unicode.ToUpper(r) }`, "x.go#f unicode.ToUpper"},
		{"unicode.SimpleFold", `package x
import "unicode"
func f(r rune) rune { return unicode.SimpleFold(r) }`, "x.go#f unicode.SimpleFold"},
		{"unicode.To", `package x
import "unicode"
func f(r rune) rune { return unicode.To(unicode.LowerCase, r) }`, "x.go#f unicode.To"},
		{"strings.Map fold table", `package x
import "strings"
func f(s string) string { return strings.Map(func(r rune) rune { return r | 0x20 }, s) }`, "x.go#f strings.Map"},
		{"function value", `package x
import "strings"
var fold = strings.ToLower`, "x.go#(package) strings.ToLower"},
		{"function passed on", `package x
import "strings"
func g(f func(string) string) {}
func h() { g(strings.ToUpper) }`, "x.go#h strings.ToUpper"},
		{"aliased import", `package x
import str "strings"
func f(s string) string { return str.ToLower(s) }`, "x.go#f strings.ToLower"},
		{"dot import", `package x
import . "strings"
func f(s string) string { return ToLower(s) }`, "x.go#(imports) dot import of strings"},
		{"x/text cases", `package x
import "golang.org/x/text/cases"
var c = cases.Fold()`, "x.go#(imports) golang.org/x/text/cases"},
		{"method", `package x
import "strings"
type T struct{}
func (t *T) f(s string) bool { return strings.EqualFold(s, "a") }`, "x.go#T.f strings.EqualFold"},
		{"closure counts with its declaration", `package x
import "strings"
func f() func(string) string { return func(s string) string { return strings.ToLower(s) } }`, "x.go#f strings.ToLower"},
		{"not a fold", `package x
import "strings"
func f(s string) bool { return strings.HasPrefix(s, "a") || strings.Contains(s, "b") || strings.TrimSpace(s) == "" }`, ""},
		{"local value called like a package", `package x
type fake struct{}
func (fake) ToLower(s string) string { return s }
func f(strings fake) string { return strings.ToLower("A") }`, ""},
		{"another package with the name", `package x
import "example.com/other/strings"
func f(s string) string { return strings.ToLower(s) }`, ""},
	} {
		if got := caseFoldHits("x.go", tc.src); got != tc.want {
			t.Errorf("%s: hits = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// An allow-list entry holds only in its own file and its own function.
func TestCaseFoldAllowListIsPerFileAndFunction(t *testing.T) {
	const src = `package x
import "strings"
func a(s string) string { return strings.ToLower(s) }
func b(s string) string { return strings.ToLower(s) }
`
	sites := caseFoldInSource("pkg/one/x.go", []byte(src))
	sites = append(sites, caseFoldInSource("pkg/two/x.go", []byte(src))...)
	allowed := map[string]string{"pkg/one/x.go#a": "test"}
	var got []string
	for _, s := range CaseFoldUnallowed(sites, allowed) {
		got = append(got, s.Key())
	}
	if want := "pkg/one/x.go#b, pkg/two/x.go#a, pkg/two/x.go#b"; strings.Join(got, ", ") != want {
		t.Fatalf("unallowed = %s, want %s", strings.Join(got, ", "), want)
	}
	if stale := CaseFoldStale(sites, map[string]string{"pkg/one/x.go#a": "test", "pkg/one/x.go#gone": "test"}); strings.Join(stale, ", ") != "pkg/one/x.go#gone" {
		t.Fatalf("stale = %v", stale)
	}
}

func TestHandFoldScanSeesEachShape(t *testing.T) {
	const src = `package x

func a(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b - 'A' + 'a'
	}
	return b
}

func b(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

func c(s string) string {
	t := []byte(s)
	for i, c := range t {
		switch {
		case c >= 'A' && c <= 'Z':
			t[i] = c + 32
		}
	}
	return string(t)
}

func d(b byte) bool { return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' }

func e(b byte) byte { return b + 1 }
`
	got := strings.Join(handFoldInSource("x.go", []byte(src)), ", ")
	if got != "x.go:3 a, x.go:10 b, x.go:17 c" {
		t.Fatalf("hits = %s", got)
	}
}
