package tokenidentity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const withComments = `// Package sample does a thing.
//
// This paragraph is the kind of narrative a comment cleanup removes.
package sample

// Total adds the values and returns the sum.
func Total(values []int) int {
	sum := 0 // running total
	for _, v := range values {
		/* inline */
		sum += v
	}
	return sum
}
`

const withoutComments = `package sample

func Total(values []int) int {
	sum := 0
	for _, v := range values {

		sum += v
	}
	return sum
}
`

const withDirective = `//go:build linux

package sample

//go:embed payload.bin
var payload []byte
`

func TestCommentsDoNotChangeTheTokenHash(t *testing.T) {
	with := HashFile("sample.go", []byte(withComments))
	without := HashFile("sample.go", []byte(withoutComments))
	if with.ScanError != "" || without.ScanError != "" {
		t.Fatalf("scan error: %q / %q", with.ScanError, without.ScanError)
	}
	if with.Tokens != without.Tokens {
		t.Fatalf("token hash moved when only comments changed: %s vs %s", with.Tokens, without.Tokens)
	}
	if with.Directives != without.Directives {
		t.Fatalf("directive hash moved with no directive in either source: %s vs %s", with.Directives, without.Directives)
	}
}

func TestOneChangedOperatorMovesTheTokenHash(t *testing.T) {
	base := HashFile("sample.go", []byte(withComments))
	edited := HashFile("sample.go", []byte(strings.Replace(withComments, "sum += v", "sum -= v", 1)))
	if base.Tokens == edited.Tokens {
		t.Fatal("token hash survived += becoming -=")
	}
}

func TestAChangedLiteralMovesTheTokenHash(t *testing.T) {
	base := HashFile("sample.go", []byte(withComments))
	edited := HashFile("sample.go", []byte(strings.Replace(withComments, "sum := 0", "sum := 1", 1)))
	if base.Tokens == edited.Tokens {
		t.Fatal("token hash survived the literal 0 becoming 1")
	}
}

// A renamed identifier changes the literal text of an IDENT token, so the
// token half catches a rename even though every token kind is unchanged.
func TestARenamedIdentifierMovesTheTokenHash(t *testing.T) {
	base := HashFile("sample.go", []byte(withComments))
	edited := HashFile("sample.go", []byte(strings.ReplaceAll(withComments, "sum", "acc")))
	if base.Tokens == edited.Tokens {
		t.Fatal("token hash survived a local variable rename")
	}
}

// Two token streams whose literals concatenate to the same bytes must not
// collide; the length delimiter in HashFile is what stops them.
func TestAdjacentTokensDoNotCollide(t *testing.T) {
	a := HashFile("a.go", []byte("package p\n\nvar x = \"ab\" + \"c\"\n"))
	b := HashFile("b.go", []byte("package p\n\nvar x = \"a\" + \"bc\"\n"))
	if a.Tokens == b.Tokens {
		t.Fatal("two different string-literal splits hashed the same")
	}
}

func TestADeletedBuildConstraintMovesOnlyTheDirectiveHash(t *testing.T) {
	base := HashFile("sample.go", []byte(withDirective))
	stripped := HashFile("sample.go", []byte(strings.Replace(withDirective, "//go:build linux\n", "", 1)))
	if base.Tokens != stripped.Tokens {
		t.Fatalf("token hash moved when a build constraint was deleted: %s vs %s", base.Tokens, stripped.Tokens)
	}
	if base.Directives == stripped.Directives {
		t.Fatal("directive hash survived a deleted //go:build line, which the token half cannot see at all")
	}
}

func TestADeletedEmbedMovesOnlyTheDirectiveHash(t *testing.T) {
	base := HashFile("sample.go", []byte(withDirective))
	stripped := HashFile("sample.go", []byte(strings.Replace(withDirective, "//go:embed payload.bin\n", "", 1)))
	if base.Tokens != stripped.Tokens {
		t.Fatalf("token hash moved when an embed was deleted: %s vs %s", base.Tokens, stripped.Tokens)
	}
	if base.Directives == stripped.Directives {
		t.Fatal("directive hash survived a deleted //go:embed line")
	}
}

func TestDirectiveLines(t *testing.T) {
	cases := []struct {
		name string
		lit  string
		want []string
	}{
		{"build", "//go:build linux && amd64", []string{"//go:build linux && amd64"}},
		{"embed", "//go:embed all:assets", []string{"//go:embed all:assets"}},
		{"generate", "//go:generate stringer -type=Kind", []string{"//go:generate stringer -type=Kind"}},
		{"noinline", "//go:noinline", []string{"//go:noinline"}},
		{"oldBuildTag", "// +build linux", []string{"// +build linux"}},
		{"oldBuildTagNoSpace", "//+build linux", []string{"//+build linux"}},
		{"lineDirective", "//line foo.go:7", []string{"//line foo.go:7"}},
		{"prose", "// go:build linux is what a constraint looks like", nil},
		{"proseNamingEmbed", "// The //go:embed below reads the payload.", nil},
		{"plain", "// Total adds the values.", nil},
		{"trailingSpace", "//go:noescape   ", []string{"//go:noescape"}},
		{"groupOfLines", "//go:build linux\n//go:noinline\n// prose", []string{"//go:build linux", "//go:noinline"}},
		{"block", "/*\n//go:build linux\n*/", []string{"//go:build linux"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DirectiveLines(c.lit)
			if len(got) != len(c.want) {
				t.Fatalf("DirectiveLines(%q) = %q, want %q", c.lit, got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("DirectiveLines(%q) = %q, want %q", c.lit, got, c.want)
				}
			}
		})
	}
}

// A prose mention of a directive must not count. If it did, deleting the
// sentence that describes an embed would report a lost directive, and a real
// loss would be indistinguishable from a cleaned-up sentence.
func TestProseNamingADirectiveDoesNotCount(t *testing.T) {
	base := HashFile("sample.go", []byte("package p\n\n// The //go:embed directive is described here.\nvar x int\n"))
	stripped := HashFile("sample.go", []byte("package p\n\nvar x int\n"))
	if base.Directives != stripped.Directives {
		t.Fatal("a prose sentence naming a directive was counted as one")
	}
}

// Reordering two directive lines is a change to the file, so the directive
// hash must not treat them as an unordered set.
func TestDirectiveOrderMatters(t *testing.T) {
	a := HashFile("a.go", []byte("//go:build linux\n//go:noinline\n\npackage p\n"))
	b := HashFile("b.go", []byte("//go:noinline\n//go:build linux\n\npackage p\n"))
	if a.Directives == b.Directives {
		t.Fatal("two directive lines hashed the same in either order")
	}
}

func TestAScanErrorIsReportedNotHashed(t *testing.T) {
	row := HashFile("broken.go", []byte("package p\n\nvar s = \"unterminated\n"))
	if row.ScanError == "" {
		t.Fatal("unterminated string literal produced no scan error")
	}
	if row.Tokens != "" || row.Directives != "" {
		t.Fatalf("a file that failed to scan was still hashed: %+v", row)
	}
	if !strings.Contains(Format(row), "SCANERROR") {
		t.Fatalf("Format hid the scan error: %q", Format(row))
	}
}

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestHashTreeIsDeterministicAndSorted(t *testing.T) {
	root := writeTree(t, map[string]string{
		"b.go":           withComments,
		"a.go":           withDirective,
		"pkg/inner/c.go": withComments,
		"notes.txt":      "not Go source",
		".hidden/d.go":   withComments,
	})

	first, err := HashTree(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := HashTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 3 {
		t.Fatalf("got %d rows, want 3 (the .txt and the hidden directory must be skipped): %+v", len(first), first)
	}
	want := []string{"a.go", "b.go", "pkg/inner/c.go"}
	for i, row := range first {
		if row.Path != want[i] {
			t.Fatalf("row %d is %q, want %q", i, row.Path, want[i])
		}
		if row != second[i] {
			t.Fatalf("a second run of the same tree differs at %q: %+v vs %+v", row.Path, row, second[i])
		}
	}
}

func TestCompareNamesEveryKindOfDifference(t *testing.T) {
	before := writeTree(t, map[string]string{
		"kept.go":    withComments,
		"edited.go":  withComments,
		"build.go":   withDirective,
		"removed.go": withComments,
	})
	after := writeTree(t, map[string]string{
		"kept.go":   strings.Replace(withComments, " // running total", "", 1),
		"edited.go": strings.Replace(withComments, "sum += v", "sum -= v", 1),
		"build.go":  strings.Replace(withDirective, "//go:build linux\n", "", 1),
		"added.go":  withComments,
	})

	beforeRows, err := HashTree(before)
	if err != nil {
		t.Fatal(err)
	}
	afterRows, err := HashTree(after)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]string{}
	for _, d := range Compare(beforeRows, afterRows) {
		got[d.Path] = d.Kind
	}
	want := map[string]string{
		"edited.go":  "tokens",
		"build.go":   "directives",
		"removed.go": "removed",
		"added.go":   "added",
	}
	for path, kind := range want {
		if got[path] != kind {
			t.Errorf("Compare reported %q for %s, want %q", got[path], path, kind)
		}
	}
	if kind, ok := got["kept.go"]; ok {
		t.Errorf("Compare reported %q for a file whose only change was a deleted comment", kind)
	}
	if len(got) != len(want) {
		t.Errorf("Compare returned %d differences, want %d: %v", len(got), len(want), got)
	}
}
