package archtest

import (
	"go/ast"
	"go/types"
	"slices"
	"sort"
)

// saveCaptureOwners names, for each capture or Group-drop function, the
// production functions that may reference it. Every SAVE route of the game
// captures through captureSave with a mode, and the Group drop rule has one
// applier, so a new route or a new applier is a finding.
var saveCaptureOwners = map[string][]string{
	"againrom/pkg/game.FrontEnd.Snapshot": {"againrom/pkg/game.FrontEnd.captureSave",
		// The scenario fixture tool builds a fixture, not a player's SAVE,
		// and captures without a transform, as the timer mode does.
		"againrom/cmd/scenariofixture.prepare"},
	"againrom/pkg/game.pruneGroupsForSave":          {"againrom/pkg/game.FrontEnd.captureSave"},
	"againrom/pkg/game.Town.pruneCityGroupsForSave": {"againrom/pkg/game.pruneGroupsForSave"},
	"againrom/pkg/sim.World.PruneGroupsForSaveUndo": {"againrom/pkg/game.pruneGroupsForSave", "againrom/pkg/sim.World.PruneGroupsForSave"},
	"againrom/pkg/sim.World.PruneGroupsForSave":     nil,
	"againrom/pkg/sim.SaveKeepsGroup":               {"againrom/pkg/sim.KeepGroupsForSave"},
}

// CheckSaveCapture reports each production reference to a function of
// saveCaptureOwners from a function the table does not name. A recursive
// reference is no route and is not reported.
func CheckSaveCapture(packages []*CheckedPackage) []string {
	var findings []string
	for _, p := range packages {
		for _, file := range p.Files[:p.NumProd] {
			for _, decl := range file.Decls {
				owner := p.ImportPath + ".package scope"
				if fn, ok := decl.(*ast.FuncDecl); ok {
					if obj, _ := p.Info.Defs[fn.Name].(*types.Func); obj != nil {
						owner = saveCaptureKey(obj)
					}
				}
				ast.Inspect(decl, func(node ast.Node) bool {
					id, ok := node.(*ast.Ident)
					if !ok {
						return true
					}
					used, _ := p.Info.Uses[id].(*types.Func)
					if used == nil {
						return true
					}
					key := saveCaptureKey(used)
					allowed, listed := saveCaptureOwners[key]
					if listed && owner != key && !slices.Contains(allowed, owner) {
						findings = append(findings, owner+" references "+key)
					}
					return true
				})
			}
		}
	}
	sort.Strings(findings)
	return findings
}

func saveCaptureKey(fn *types.Func) string {
	path := ""
	if fn.Pkg() != nil {
		path = fn.Pkg().Path()
	}
	return path + "." + savFuncKey(fn)
}
