package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/archtest"
)

// root is the synthetic module the tests select over.
var root = filepath.Join("testdata", "mod")

// selected runs Select over the synthetic module with one change and returns
// the names as package-qualified words.
func selected(t *testing.T, c Change) []string {
	t.Helper()
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	m, err := archtest.LoadTypeCheckedModule(abs)
	if err != nil {
		t.Fatal(err)
	}
	tests, _, err := Select(m, abs, []Change{c})
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, x := range tests {
		s := strings.TrimPrefix(x.Package, "againrom/") + "." + x.Name
		if x.Release {
			s += "(release)"
		}
		out = append(out, s)
	}
	return out
}

func lineOf(t *testing.T, file, text string) Span {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		t.Fatal(err)
	}
	for i, l := range strings.Split(string(b), "\n") {
		if strings.Contains(l, text) {
			return Span{i + 1, i + 1}
		}
	}
	t.Fatalf("%s has no line with %q", file, text)
	return Span{}
}

func TestSelectedTestsAreTheCallersOfAChangedFunctionAtAnyDepth(t *testing.T) {
	got := selected(t, Change{"a/a.go", []Span{lineOf(t, "a/a.go", "func Leaf")}})
	want := []string{
		"a.TestHelper", "a.TestLeafDirect", "a.TestReleaseTop(release)", "a.TestTopIsTransitive",
		"b.TestWrap",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestAChangedMiddleFunctionSelectsItsCallersAndNotItsCallees(t *testing.T) {
	got := selected(t, Change{"a/a.go", []Span{lineOf(t, "a/a.go", "func Mid")}})
	for _, bad := range []string{"a.TestLeafDirect", "a.TestUnrelated", "b.TestAlone"} {
		for _, g := range got {
			if g == bad {
				t.Fatalf("%s selected by a change of Mid: %v", bad, got)
			}
		}
	}
	if len(got) != 4 {
		t.Fatalf("got %v", got)
	}
}

func TestAChangedMethodSelectsTheCallersOfThatTypeAndOfTheInterface(t *testing.T) {
	got := selected(t, Change{"a/a.go", []Span{lineOf(t, "a/a.go", "func (Sq) Area")}})
	if !reflect.DeepEqual(got, []string{"a.TestShape", "a.TestSquare"}) {
		t.Fatalf("got %v", got)
	}
}

func TestAChangedMethodDoesNotSelectTheCallersOfAMethodOfTheSameNameOnAnotherType(t *testing.T) {
	got := selected(t, Change{"a/a.go", []Span{lineOf(t, "a/a.go", "func (t T) Grow")}})
	if !reflect.DeepEqual(got, []string{"a.TestMethod"}) {
		t.Fatalf("got %v", got)
	}
}

func TestAChangedFieldSelectsTheTestsThatNameItAndTheTestsOfItsType(t *testing.T) {
	got := selected(t, Change{"a/a.go", []Span{lineOf(t, "a/a.go", "Name string")}})
	if !reflect.DeepEqual(got, []string{"a.TestField", "a.TestMethod"}) {
		t.Fatalf("got %v", got)
	}
}

func TestAChangedConstReachesTheVarThatReadsItAndItsTests(t *testing.T) {
	got := selected(t, Change{"a/a.go", []Span{lineOf(t, "a/a.go", "const Limit")}})
	if !reflect.DeepEqual(got, []string{"a.TestTable"}) {
		t.Fatalf("got %v", got)
	}
}

func TestAChangedTestHelperSelectsTheTestsThatCallIt(t *testing.T) {
	got := selected(t, Change{"a/a_test.go", []Span{lineOf(t, "a/a_test.go", "func fixture")}})
	if !reflect.DeepEqual(got, []string{"a.TestHelper"}) {
		t.Fatalf("got %v", got)
	}
}

func TestAChangedTestSelectsItself(t *testing.T) {
	got := selected(t, Change{"a/a_test.go", []Span{lineOf(t, "a/a_test.go", "func TestNothing")}})
	if !reflect.DeepEqual(got, []string{"a.TestNothing"}) {
		t.Fatalf("got %v", got)
	}
}

func TestAChangedTestdataFileSelectsTheTestsOfItsPackageThatNameIt(t *testing.T) {
	got := selected(t, Change{Path: "a/testdata/in.txt"})
	if !reflect.DeepEqual(got, []string{"a.TestReadsFile"}) {
		t.Fatalf("got %v", got)
	}
}

func TestAChangedInitSelectsEveryTestOfItsPackage(t *testing.T) {
	got := selected(t, Change{"c/c.go", []Span{lineOf(t, "c/c.go", "func init")}})
	if !reflect.DeepEqual(got, []string{"c.TestC1", "c.TestC2"}) {
		t.Fatalf("got %v", got)
	}
}

func TestACallThroughAnImportedPackageReachesTheCallersInOtherPackages(t *testing.T) {
	got := selected(t, Change{"b/b.go", []Span{lineOf(t, "b/b.go", "func Wrap")}})
	if !reflect.DeepEqual(got, []string{"b.TestWrap"}) {
		t.Fatalf("got %v", got)
	}
}

func TestAChangedPathThatIsNotGoOrTestdataIsIgnored(t *testing.T) {
	abs, _ := filepath.Abs(root)
	m, err := archtest.LoadTypeCheckedModule(abs)
	if err != nil {
		t.Fatal(err)
	}
	tests, ignored, err := Select(m, abs, []Change{{Path: "go.mod"}, {Path: "docs/x.md"}})
	if err != nil || len(tests) != 0 || !reflect.DeepEqual(ignored, []string{"docs/x.md", "go.mod"}) {
		t.Fatalf("got %v %v %v", tests, ignored, err)
	}
}

func TestParseDiffTakesTheNewLinesOfEveryHunk(t *testing.T) {
	diff := "diff --git a/x.go b/x.go\n--- a/x.go\n+++ b/x.go\n@@ -3,2 +3,4 @@ f\n+a\n@@ -10 +12 @@\n@@ -20,3 +24,0 @@\n" +
		"diff --git a/gone.go b/gone.go\n--- a/gone.go\n+++ /dev/null\n@@ -1,5 +0,0 @@\n"
	got := parseDiff(diff)
	want := []Change{{"x.go", []Span{{3, 6}, {12, 12}, {24, 25}}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
