package archtest

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"
)

// commandConstructors is the one non-test file allowed to write a sim.Command
// composite literal. It is where the constructors are, so it is where the
// field assignments for every kind are, and it is the only place a reader has
// to go to learn what a kind's two numbers mean.
const commandConstructors = "pkg/sim/command.go"

// commandLiteralExempt is that file and the test beside it. The test is exempt
// for the opposite reason to every other test file: its literals are not debt
// but the SPECIFICATION, one written-out command per constructor, and the whole
// of what it proves is that the two forms agree field for field. Counting them
// as debt would ask for them to be deleted, which would delete the proof.
var commandLiteralExempt = map[string]bool{
	commandConstructors:       true,
	"pkg/sim/command_test.go": true,
}

// commandPackage is the import path whose Command type this measures.
const commandPackage = "againrom/pkg/sim"

// CommandLiteralReport is every sim.Command composite literal in the module.
//
// Production is the list this guard drives to nothing: a non-test file outside
// the constructor file that builds a Command by naming its fields. Tests is the
// same count over _test.go files, which is a ratchet rather than an absolute -
// nine hundred test literals are not a defect, but they are debt, and the number
// may only fall.
type CommandLiteralReport struct {
	Production []string
	Tests      int
	TestFiles  []CommandLiteralFile
}

// CommandLiteralFile is one test file and how many literals it writes.
type CommandLiteralFile struct {
	File  string
	Count int
}

// CommandLiteralBaseline is the committed record of the ratcheted test count.
type CommandLiteralBaseline struct {
	Tests int
}

// CheckCommandLiterals reports every violation of the two rules: production is
// an absolute zero, and the test count may only fall.
func CheckCommandLiterals(report CommandLiteralReport, base CommandLiteralBaseline) []string {
	var out []string
	for _, site := range report.Production {
		out = append(out, fmt.Sprintf("%s writes a sim.Command composite literal; build it with a constructor in %s",
			site, commandConstructors))
	}
	switch {
	case report.Tests > base.Tests:
		out = append(out, fmt.Sprintf("sim.Command literals in tests rose from %d to %d; a new test builds commands with the constructors",
			base.Tests, report.Tests))
	case report.Tests < base.Tests:
		out = append(out, fmt.Sprintf("sim.Command literals in tests fell from %d to %d; write the new number into commandliteral_baseline.go in this commit",
			base.Tests, report.Tests))
	}
	return out
}

// LoadCommandLiterals walks the module and counts sim.Command composite
// literals.
//
// It is TYPE-AWARE: every file in the module is type-checked with go/types
// (loadTypeCheckedModule, shared with LoadComposition within one process),
// and a composite literal counts when go/types resolves ITS OWN static type -
// not how the source spells it - to the named type sim.Command. That single
// question already covers a bare Command{} written inside package sim, a
// qualified sim.Command{}, a name reached through a dot import or a type
// alias, and an element type elided any number of levels deep inside a
// slice, array or map literal, as a value, a pointer element or a key.
//
// What it cannot see is a file go list does not select for this build:
// another GOOS/GOARCH, a custom build tag, //go:build ignore, and testdata/
// or _-prefixed directories. A type parameter, or a conversion from another
// struct type, is not a Command literal to it either.
func LoadCommandLiterals(root string) (CommandLiteralReport, error) {
	m, err := loadTypeCheckedModule(root)
	if err != nil {
		return CommandLiteralReport{}, err
	}

	var report CommandLiteralReport
	perTest := map[string]int{}
	visit := func(f *ast.File, info *types.Info, rel string, isTest bool) {
		if commandLiteralExempt[rel] {
			return
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok || !isCommandType(info.Types[lit].Type) {
				return true
			}
			if isTest {
				perTest[rel]++
				report.Tests++
			} else {
				report.Production = append(report.Production, fmt.Sprintf("%s:%d", rel, m.Fset.Position(lit.Pos()).Line))
			}
			return true
		})
	}

	paths := make([]string, 0, len(m.Packages))
	for p := range m.Packages {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, path := range paths {
		cp := m.Packages[path]
		for i, f := range cp.Files {
			visit(f, cp.Info, cp.RelPaths[i], i >= cp.NumProd)
		}
		for i, f := range cp.XTestFiles {
			visit(f, cp.XTestInfo, cp.XTestRelPaths[i], true)
		}
	}

	sort.Strings(report.Production)
	for f, n := range perTest {
		report.TestFiles = append(report.TestFiles, CommandLiteralFile{File: f, Count: n})
	}
	sort.Slice(report.TestFiles, func(i, j int) bool {
		if report.TestFiles[i].Count != report.TestFiles[j].Count {
			return report.TestFiles[i].Count > report.TestFiles[j].Count
		}
		return report.TestFiles[i].File < report.TestFiles[j].File
	})
	return report, nil
}

// isCommandType reports whether t, after unwrapping any type alias, is the
// named type sim.Command. A literal's own type is a pointer only when an
// elided &Command{} element produced it, so one pointer is unwrapped.
func isCommandType(t types.Type) bool {
	if t == nil {
		return false
	}
	t = types.Unalias(t)
	if p, ok := t.(*types.Pointer); ok {
		t = types.Unalias(p.Elem())
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == commandPackage && obj.Name() == "Command"
}
