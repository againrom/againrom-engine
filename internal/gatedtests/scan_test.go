package gatedtests

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func scanSourceFixture(t *testing.T, files map[string]string) []Test {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "pkg", "fixture")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan fixture: %v", err)
	}
	return got
}

func TestScanFollowsPackageLocalHelpers(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   []Test
	}{
		{
			name:   "direct",
			source: `func TestDirect(t *testing.T) { t.Skip("no AGAINROM_ASSETS") }`,
			want:   []Test{{Package: "pkg/fixture", Func: "TestDirect"}},
		},
		{
			name: "transitive with repeated subtest calls",
			source: `
func TestTransitive(t *testing.T) {
    t.Run("child", func(t *testing.T) { outer(t); outer(t) })
}
func outer(t *testing.T) { middle(t) }
func middle(t *testing.T) { requireInstall(t) }
func requireInstall(t *testing.T) { t.Skipf("no %s: AGAINROM_ASSETS", "install") }
`,
			want: []Test{{Package: "pkg/fixture", Func: "TestTransitive", Via: "outer"}},
		},
		{
			name: "cycle with a gate",
			source: `
func TestCycle(t *testing.T) { cycleA(t) }
func cycleA(t *testing.T) { cycleB(t) }
func cycleB(t *testing.T) { cycleA(t); requireInstall(t) }
func requireInstall(t *testing.T) { t.Skip("no AGAINROM_ASSETS") }
`,
			want: []Test{{Package: "pkg/fixture", Func: "TestCycle", Via: "cycleA"}},
		},
		{
			name: "cycle without a gate",
			source: `
func TestCycle(t *testing.T) { cycleA(t) }
func cycleA(t *testing.T) { cycleB(t) }
func cycleB(t *testing.T) { cycleA(t) }
`,
		},
		{
			name: "non-gated helper and unused gated helper",
			source: `
func TestSubject(t *testing.T) { outer(t) }
func outer(t *testing.T) { subject(t) }
func subject(t *testing.T) { t.Log("AGAINROM_ASSETS supplied"); t.Skip("no matching subject") }
func unused(t *testing.T) { t.Skip("no AGAINROM_ASSETS") }
`,
		},
		{
			name: "local function shadows a helper",
			source: `
func TestShadow(t *testing.T) {
    requireInstall := func(t *testing.T) {}
    requireInstall(t)
}
func requireInstall(t *testing.T) { t.Skip("no AGAINROM_ASSETS") }
`,
		},
		{
			name: "silent return is outside the skip census",
			source: `
func TestSilent(t *testing.T) { optional(t) }
func optional(t *testing.T) { if "AGAINROM_ASSETS" != "" { return } }
`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := scanSourceFixture(t, map[string]string{
				"fixture_test.go": "package fixture\nimport \"testing\"\n" + tc.source,
			})
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Scan = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestScanCountsOnlyTheDefaultBuild(t *testing.T) {
	got := scanSourceFixture(t, map[string]string{
		"plain_test.go": `package fixture
import "testing"
func TestPlain(t *testing.T) { requireInstall(t) }
`,
		"helper_test.go": `package fixture
import "testing"
func requireInstall(t *testing.T) { t.Skip("no AGAINROM_ASSETS") }
`,
		"tagged_test.go": `//go:build sessioncorpusaudit

package fixture
import "testing"
func TestTagged(t *testing.T) { requireInstall(t) }
`,
		"system_test.go": "//go:build " + runtime.GOOS + `

package fixture
import "testing"
func TestThisSystem(t *testing.T) { requireInstall(t) }
`,
		"other_test.go": "//go:build !" + runtime.GOOS + `

package fixture
import "testing"
func TestOtherSystem(t *testing.T) { requireInstall(t) }
`,
	})
	want := []Test{
		{Package: "pkg/fixture", Func: "TestPlain", Via: "requireInstall"},
		{Package: "pkg/fixture", Func: "TestThisSystem", Via: "requireInstall"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Scan = %#v, want %#v", got, want)
	}
}

func TestScanKeepsHelperCallsWithinTheirPackage(t *testing.T) {
	got := scanSourceFixture(t, map[string]string{
		"internal_test.go": `package fixture
import "testing"
func TestInternal(t *testing.T) { requireInstall(t) }
`,
		"helper_test.go": `package fixture
import "testing"
func requireInstall(t *testing.T) { t.Skip("no AGAINROM_ASSETS") }
`,
		"external_test.go": `package fixture_test
import "testing"
func TestExternal(t *testing.T) { requireInstall(t) }
func requireInstall(t *testing.T) {}
`,
	})
	want := []Test{{Package: "pkg/fixture", Func: "TestInternal", Via: "requireInstall"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Scan = %#v, want %#v", got, want)
	}
}

func TestScanHelperDepthIsNotAnArbitraryCutoff(t *testing.T) {
	var source strings.Builder
	source.WriteString("package fixture\nimport \"testing\"\nfunc TestDeep(t *testing.T) { helper000(t) }\n")
	const depth = 512
	for i := 0; i < depth-1; i++ {
		fmt.Fprintf(&source, "func helper%03d(t *testing.T) { helper%03d(t) }\n", i, i+1)
	}
	fmt.Fprintf(&source, "func helper%03d(t *testing.T) { t.Skip(\"no AGAINROM_ASSETS\") }\n", depth-1)
	got := scanSourceFixture(t, map[string]string{"deep_test.go": source.String()})
	want := []Test{{Package: "pkg/fixture", Func: "TestDeep", Via: "helper000"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Scan = %#v, want %#v", got, want)
	}
}

// TestScanMatchesTheCheckedInPopulationList is this package's own
// fail-closed check, the same shape as pkg/ui's screen registry
// (screenregistry_test.go): Scan's live result must equal
// testdata/population.txt exactly, in both directions. A gated test added
// without updating the list fails here; a line in the list that Scan no
// longer finds (a test renamed, moved, or ungated) also fails here. Either
// failure is the population changing without this package's manifest
// changing with it — the same failure check-milestone.sh's own history
// warns against (AGENTS.md: "a gate must print what it SELECTED").
func TestScanMatchesTheCheckedInPopulationList(t *testing.T) {
	root, err := ModuleRoot(".")
	if err != nil {
		t.Fatalf("ModuleRoot: %v", err)
	}
	got, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	gotIDs := map[string]bool{}
	for _, test := range got {
		gotIDs[test.ID()] = true
	}

	raw, err := os.ReadFile("testdata/population.txt")
	if err != nil {
		t.Fatalf("reading testdata/population.txt: %v", err)
	}
	wantIDs := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		wantIDs[line] = true
	}

	for id := range gotIDs {
		if !wantIDs[id] {
			t.Errorf("Scan found %s, not listed in testdata/population.txt", id)
		}
	}
	for id := range wantIDs {
		if !gotIDs[id] {
			t.Errorf("testdata/population.txt lists %s, Scan did not find it", id)
		}
	}
	if len(gotIDs) != len(wantIDs) {
		t.Errorf("population size mismatch: Scan found %d, testdata/population.txt lists %d", len(gotIDs), len(wantIDs))
	}
}

// TestSelfTestNameIsDeclared guards selfTestName (scan.go) against exactly
// the drift that produced round 2's finding 3: the constant must name a
// function this package's own _test.go files actually declare. Parses
// source with go/ast rather than trusting the constant's own string, the
// same shape pkg/ui/screenregistry_test.go's
// TestEveryRegisteredSyntheticTestExists uses for the screen registry.
func TestSelfTestNameIsDeclared(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		name := fi.Name()
		return len(name) > 8 && name[len(name)-8:] == "_test.go"
	}, 0)
	if err != nil {
		t.Fatalf("parsing package directory: %v", err)
	}
	declared := map[string]bool{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fd, ok := decl.(*ast.FuncDecl)
				if !ok || fd.Recv != nil {
					continue
				}
				declared[fd.Name.Name] = true
			}
		}
	}
	if !declared[selfTestName] {
		t.Errorf("selfTestName = %q, which no _test.go file in this package declares", selfTestName)
	}
}

// TestSubjectHumanizesEveryFuncName is a narrow direct test of the naming
// function used by cmd/screencensus, independent of a live scan.
func TestSubjectHumanizesEveryFuncName(t *testing.T) {
	cases := []struct {
		test Test
		want string
	}{
		{Test{Func: "TestReleaseSchoolAndTavernButtonArtLoads"}, "release school and tavern button art loads"},
		{Test{Func: "TestTheTenthMissionRunsEndToEndOnALawfulInstall"}, "the tenth mission runs end to end on a lawful install"},
		{Test{Func: "TestInstallWordsOverALawfulInstall"}, "install words over a lawful install"},
	}
	for _, c := range cases {
		if got := Subject(c.test); got != c.want {
			t.Errorf("Subject(%q) = %q, want %q", c.test.Func, got, c.want)
		}
	}
}
