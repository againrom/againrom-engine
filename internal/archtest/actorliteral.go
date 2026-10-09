package archtest

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"
)

// actorConstructor is the one non-test file allowed to build an actor Entity
// by composite literal: sim.NewActor.
const actorConstructor = "pkg/sim/actorconstructor.go"

// actorOrigins are the files that create actors. Each calls sim.NewActor and
// holds no Entity literal at all, whatever the baseline says.
var actorOrigins = []string{
	"pkg/mapload/fromalm.go",
	"pkg/mapload/start.go",
	"pkg/mapload/siegehire.go",
	"pkg/mapload/sourcebinding.go",
	"pkg/mapload/cheatfactory.go",
	"pkg/game/scenario.go",
	"pkg/sim/spell.go",
}

// ActorLiteralReport counts, per non-test file, the sim.Entity composite
// literals that name at least one field. An empty Entity{} is the zero value
// a lookup returns and builds nothing.
type ActorLiteralReport struct {
	Files map[string]int
	Sites []string
}

// CheckActorLiterals reports every file whose count rose above its baseline,
// every file whose count fell (the baseline is rewritten in that commit), and
// any literal in an actor origin.
func CheckActorLiterals(report ActorLiteralReport, base map[string]int) []string {
	var out []string
	files := map[string]bool{}
	for f := range report.Files {
		files[f] = true
	}
	for f := range base {
		files[f] = true
	}
	names := make([]string, 0, len(files))
	for f := range files {
		names = append(names, f)
	}
	sort.Strings(names)
	origin := map[string]bool{}
	for _, f := range actorOrigins {
		origin[f] = true
	}
	for _, f := range names {
		got, want := report.Files[f], base[f]
		switch {
		case origin[f] && got > 0:
			out = append(out, fmt.Sprintf("%s creates actors and writes %d sim.Entity literal(s); build them with sim.NewActor", f, got))
		case got > want:
			out = append(out, fmt.Sprintf("%s writes %d sim.Entity literal(s), baseline %d; build an actor with sim.NewActor in %s",
				f, got, want, actorConstructor))
		case got < want:
			out = append(out, fmt.Sprintf("%s sim.Entity literals fell from %d to %d; write the new number into actorliteral_baseline.go in this commit",
				f, want, got))
		}
	}
	return out
}

// LoadActorLiterals walks the module's non-test files and counts sim.Entity
// composite literals with at least one element, by each literal's own static
// type, as LoadCommandLiterals does for sim.Command.
func LoadActorLiterals(root string) (ActorLiteralReport, error) {
	m, err := loadTypeCheckedModule(root)
	if err != nil {
		return ActorLiteralReport{}, err
	}
	report := ActorLiteralReport{Files: map[string]int{}}
	for _, cp := range m.Packages {
		for i, f := range cp.Files {
			if i >= cp.NumProd {
				break
			}
			rel := cp.RelPaths[i]
			if rel == actorConstructor {
				continue
			}
			ast.Inspect(f, func(n ast.Node) bool {
				lit, ok := n.(*ast.CompositeLit)
				if !ok || len(lit.Elts) == 0 || !isSimType(cp.Info.Types[lit].Type, "Entity") {
					return true
				}
				report.Files[rel]++
				report.Sites = append(report.Sites, fmt.Sprintf("%s:%d", rel, m.Fset.Position(lit.Pos()).Line))
				return true
			})
		}
	}
	sort.Strings(report.Sites)
	return report, nil
}

// isSimType reports whether t, after unwrapping an alias and one pointer, is
// the named type pkg/sim.<name>.
func isSimType(t types.Type, name string) bool {
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
	return obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == commandPackage && obj.Name() == name
}
