package archtest

import (
	"go/token"
	"go/types"
	"os"
	"strings"
	"testing"
)

// TestLiveCommandLiteralsMatchTheirBaseline measures the real tree. A non-test
// file that builds a sim.Command by naming its fields fails here whatever the
// counts say, and a test file that adds one fails the ratchet.
func TestLiveCommandLiteralsMatchTheirBaseline(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	report, err := LoadCommandLiterals(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	if report.Tests == 0 {
		t.Fatal("measured no command literals at all; the loader or the module root is wrong")
	}
	for _, v := range CheckCommandLiterals(report, CommittedCommandLiterals) {
		t.Error(v)
	}
}

// TestCommandConstructorFileIsExempt proves the exemption is a real hole and
// not an accident of the walk: the constructors themselves are literals, and
// the loader has to see them somewhere or the production zero would hold over a
// tree with no constructors in it.
func TestCommandConstructorFileIsExempt(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	report, err := LoadCommandLiterals(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, site := range report.Production {
		if commandLiteralExempt[strings.SplitN(site, ":", 2)[0]] {
			t.Fatalf("%s is counted as production; it is the constructor file", site)
		}
	}
}

// TestCheckCommandLiteralsRatchets holds the evaluator against a table, so the
// direction of each number is decided here rather than by whichever tree
// happens to be checked out.
func TestCheckCommandLiteralsRatchets(t *testing.T) {
	base := CommandLiteralBaseline{Tests: 936}
	cases := []struct {
		name    string
		report  CommandLiteralReport
		want    int
		wantSub string
	}{
		{"equal", CommandLiteralReport{Tests: 936}, 0, ""},
		{"a test literal added", CommandLiteralReport{Tests: 937}, 1, "rose from 936 to 937"},
		{"a test literal removed", CommandLiteralReport{Tests: 935}, 1, "fell from 936 to 935"},
		{"one production literal", CommandLiteralReport{Tests: 936, Production: []string{"pkg/game/world.go:12"}}, 1, "pkg/game/world.go:12"},
		{"production and a rise", CommandLiteralReport{Tests: 937, Production: []string{"cmd/x/main.go:3"}}, 2, "cmd/x/main.go:3"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CheckCommandLiterals(c.report, base)
			if len(got) != c.want {
				t.Fatalf("violations = %v, want %d", got, c.want)
			}
			if c.wantSub != "" && !contains(got, c.wantSub) {
				t.Fatalf("violations = %v, want one naming %q", got, c.wantSub)
			}
		})
	}
}

// An elided &Command{} element, as in []*sim.Command{{...}}, has type
// *sim.Command; the walk must still count it.
func TestIsCommandTypeUnwrapsOnePointer(t *testing.T) {
	pkg := types.NewPackage(commandPackage, "sim")
	command := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Command", nil), types.NewStruct(nil, nil), nil)
	if !isCommandType(command) || !isCommandType(types.NewPointer(command)) {
		t.Fatal("sim.Command or *sim.Command is not recognised")
	}
	if isCommandType(types.NewPointer(types.NewPointer(command))) {
		t.Fatal("**sim.Command is recognised; no literal has that type")
	}
}
