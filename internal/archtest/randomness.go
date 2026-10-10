package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// RandomnessPackages are the module-relative packages whose production
// sources may draw only through pkg/random.
var RandomnessPackages = []string{"pkg/sim", "pkg/mapload", "pkg/game", "pkg/ui", "pkg/town"}

// lcgConstants are the original generator's multiplier and increment. Written
// anywhere but pkg/random they are a second copy of the recurrence.
var lcgConstants = map[uint64]bool{214013: true, 2531011: true}

// randomSinks are the pkg/random calls that take a seed or a session.
var randomSinks = map[string]bool{
	"NewGo": true, "NewStream": true, "NewService": true, "SetStreamSeed": true,
	"SetLaunch": true, "Fresh": true, "Begin": true, "Prepare": true,
	"ReseedValue": true, "DeriveSeed": true, "MissionLoadState": true,
	"NewDraws": true, "NewOriginalDraws": true,
}

// RandomnessDebt counts, by file, the violations that could not yet move into
// pkg/random. A count may only fall; a fall lowers it here in the same commit.
var RandomnessDebt = map[string]int{}

// SanctionedClockSeeds names the one function, by file, that may read the
// clock to choose a session seed. Every stream then derives from that seed.
var SanctionedClockSeeds = map[string]string{"pkg/game/randomsession.go": "clockSessionSeed"}

// CheckRandomness scans production sources, keyed by module-relative slash
// path, for randomness that bypasses the random service. It reports, per file
// in path order and then in source order:
//
//   - an import of math/rand or a package under it;
//   - an integer literal equal to the original generator's multiplier or
//     increment;
//   - a seed or session argument to a pkg/random call that reads package
//     time, and a function named for a seed that reads package time, other
//     than the sanctioned clock seed.
//
// It judges parsed syntax: a name in a comment or a string is none of these.
func CheckRandomness(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "randomness", Reason: "no sources scanned"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(p token.Pos) string { return name + ":" + strconv.Itoa(fset.Position(p).Line) }
		ast.Inspect(f, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.ImportSpec:
				imp, _ := strconv.Unquote(node.Path.Value)
				if imp == "math/rand" || strings.HasPrefix(imp, "math/rand/") {
					vs = append(vs, Violation{From: at(node.Pos()), Import: imp, Reason: "imports a generator outside pkg/random"})
				}
				return false
			case *ast.BasicLit:
				if node.Kind == token.INT {
					if v, err := strconv.ParseUint(strings.ReplaceAll(node.Value, "_", ""), 0, 64); err == nil && lcgConstants[v] {
						vs = append(vs, Violation{From: at(node.Pos()), Reason: "writes the original generator's constant " + node.Value})
					}
				}
			case *ast.FuncDecl:
				if strings.Contains(strings.ToLower(node.Name.Name), "seed") && node.Body != nil &&
					readsTime(node.Body) && SanctionedClockSeeds[name] != node.Name.Name {
					vs = append(vs, Violation{From: at(node.Pos()), Reason: "seed function " + node.Name.Name + " reads the clock"})
				}
			case *ast.CallExpr:
				if sinkName(node.Fun) != "" {
					for _, arg := range node.Args {
						if readsTime(arg) {
							vs = append(vs, Violation{From: at(arg.Pos()), Reason: "seeds " + sinkName(node.Fun) + " from the clock"})
						}
					}
				}
			}
			return true
		})
	}
	return vs
}

// sinkName answers the pkg/random seed call fun names, or "".
func sinkName(fun ast.Expr) string {
	switch x := fun.(type) {
	case *ast.SelectorExpr:
		if randomSinks[x.Sel.Name] {
			return x.Sel.Name
		}
	case *ast.Ident:
		if randomSinks[x.Name] {
			return x.Name
		}
	}
	return ""
}

// readsTime reports whether n selects from package time or calls a Unix
// time accessor.
func readsTime(n ast.Node) bool {
	found := false
	ast.Inspect(n, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "time" {
				found = true
			}
			switch sel.Sel.Name {
			case "UnixNano", "UnixMilli", "UnixMicro":
				found = true
			}
		}
		return !found
	})
	return found
}

// LoadRandomnessSources reads the production sources of RandomnessPackages
// under moduleRoot, keyed by module-relative slash path. Only each package's
// own directory is read; a nested directory is another package.
func LoadRandomnessSources(moduleRoot string) (map[string]string, error) {
	out := map[string]string{}
	for _, pkg := range RandomnessPackages {
		dir := filepath.Join(moduleRoot, filepath.FromSlash(pkg))
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			b, err := fs.ReadFile(os.DirFS(dir), e.Name())
			if err != nil {
				return nil, err
			}
			out[pkg+"/"+e.Name()] = string(b)
		}
	}
	return out, nil
}
