package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The return of a mission left without completion is one transaction for
// every game, returnToEntry in the campaign return file. It and the functions
// that admit and run it read the state at entry from the session's item-home
// row and choose nothing by game themselves.
const entryReturnFile = "pkg/game/campaignreturn.go"

// entryReturnFuncs are the abandon return's functions that hold no game
// branch.
var entryReturnFuncs = map[string]bool{"leaveLive": true, "returnToEntry": true, "canLeaveMission": true}

// entryReturnGameChoices are the names whose use in those functions chooses
// by game: the second campaign's record, its field, a row of the item home or
// the campaign service picked directly.
var entryReturnGameChoices = map[string]bool{
	"secondOf": true, "second": true, "secondPartyItems": true, "firstPartyItems": true,
	"campaignOf": true, "secondCampaignRules": true, "firstCampaignRules": true, "campaign": true,
}

// CheckEntryReturn reports a game choice inside the abandon return's
// functions, and a returnToEntry declared other than once, in its file.
// files maps module-relative paths to source; only production files of
// pkg/game are read.
func CheckEntryReturn(files map[string]string) []Violation {
	names := make([]string, 0, len(files))
	for name := range files {
		if strings.HasPrefix(name, "pkg/game/") && !strings.Contains(strings.TrimPrefix(name, "pkg/game/"), "/") &&
			strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return []Violation{{From: "pkg/game", Reason: "no sources scanned - the entry return scan found nothing to read"}}
	}
	sort.Strings(names)
	var vs []Violation
	var declared []string
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(p token.Pos) string { return name + ":" + strconv.Itoa(fset.Position(p).Line) }
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || !entryReturnFuncs[fn.Name.Name] {
				continue
			}
			if fn.Name.Name == "returnToEntry" {
				declared = append(declared, at(fn.Pos()))
				if name != entryReturnFile {
					vs = append(vs, Violation{From: at(fn.Pos()), Reason: "returnToEntry outside " + entryReturnFile})
				}
			}
			if fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok && entryReturnGameChoices[id.Name] {
					vs = append(vs, Violation{From: at(id.Pos()),
						Reason: fn.Name.Name + " names " + id.Name + "; the abandon return reads the state at entry from the session's item-home row"})
				}
				return true
			})
		}
	}
	if len(declared) != 1 {
		vs = append(vs, Violation{From: "pkg/game", Reason: "returnToEntry declared " + strconv.Itoa(len(declared)) + " times, want 1: " + strings.Join(declared, ", ")})
	}
	return vs
}
