package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// The mod window's page scopes. Each page of a scope of its own is one row on
// each side: the UI's modScopePages and the game's workspaceScopes. Every
// ui.Mod<Name>Scope constant stands in both tables, and the game names it
// nowhere else, so a page is not a branch of a chain.

// ScopeRowFiles are the UI file that declares the scopes with modScopePages
// and the game file that holds workspaceScopes, whose rows workspaceScopeRows
// builds.
var ScopeRowFiles = [2]string{"pkg/ui/mapeditorworkspace.go", "pkg/game/mapinspectionworkspace.go"}

// ScopeRowFindings reports each page scope missing from a table and each
// game reference to one outside workspaceScopes; files maps path to source.
func ScopeRowFindings(files map[string]string) ([]string, error) {
	parse := func(path string) (*ast.File, error) {
		src, ok := files[path]
		if !ok {
			return nil, fmt.Errorf("%s: not given", path)
		}
		return parser.ParseFile(token.NewFileSet(), path, src, 0)
	}
	uiFile, err := parse(ScopeRowFiles[0])
	if err != nil {
		return nil, err
	}
	gameFile, err := parse(ScopeRowFiles[1])
	if err != nil {
		return nil, err
	}
	scope := func(name string) bool { return strings.HasPrefix(name, "Mod") && strings.HasSuffix(name, "Scope") }
	var scopes []string
	inTable := map[string]map[string]bool{"modScopePages": {}, "workspaceScopes": {}}
	var outside []string
	walk := func(f *ast.File, ident func(n ast.Node) (string, bool)) {
		for _, d := range f.Decls {
			g, isVar := d.(*ast.GenDecl)
			table := ""
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "workspaceScopeRows" {
				table = "workspaceScopes"
			}
			if isVar && g.Tok == token.VAR && len(g.Specs) == 1 {
				if v, ok := g.Specs[0].(*ast.ValueSpec); ok && len(v.Names) == 1 && inTable[v.Names[0].Name] != nil {
					table = v.Names[0].Name
				}
			}
			ast.Inspect(d, func(n ast.Node) bool {
				if name, ok := ident(n); ok {
					if table != "" {
						inTable[table][name] = true
					} else {
						outside = append(outside, name)
					}
				}
				return true
			})
		}
	}
	for _, d := range uiFile.Decls {
		if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.CONST {
			for _, s := range g.Specs {
				for _, n := range s.(*ast.ValueSpec).Names {
					if scope(n.Name) {
						scopes = append(scopes, n.Name)
					}
				}
			}
		}
	}
	walk(uiFile, func(n ast.Node) (string, bool) {
		if id, ok := n.(*ast.Ident); ok && scope(id.Name) {
			return id.Name, true
		}
		return "", false
	})
	outside = nil
	walk(gameFile, func(n ast.Node) (string, bool) {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return "", false
		}
		pkg, ok := sel.X.(*ast.Ident)
		return sel.Sel.Name, ok && pkg.Name == "ui" && scope(sel.Sel.Name)
	})
	var out []string
	for _, s := range scopes {
		for _, table := range []string{"modScopePages", "workspaceScopes"} {
			if !inTable[table][s] {
				out = append(out, fmt.Sprintf("%s is not a row of %s", s, table))
			}
		}
	}
	for _, s := range outside {
		out = append(out, fmt.Sprintf("%s: ui.%s is named outside workspaceScopes", ScopeRowFiles[1], s))
	}
	if len(scopes) == 0 {
		out = append(out, ScopeRowFiles[0]+": declares no page scope")
	}
	return out, nil
}
