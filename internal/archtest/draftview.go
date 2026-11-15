package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// The draft view scan. The editor's readers of the project's drafts reach
// them only through projectView: the callers call it, and the others take
// its draftView and no draft map of their own. A reader that deletes a name
// itself, takes a bare draft map or reads a draft by its exact name is a
// second route.

// DraftViewFiles are the files that hold the readers, each with its readers.
var DraftViewFiles = map[string][]string{
	"pkg/game/mapeditormod.go":        {"projectFiles", "rowFiles", "missionFiles", "checkMissionSet"},
	"pkg/game/mapeditorcampaign.go":   {"missionDraft"},
	"pkg/game/modmapsave.go":          {"readInventory"},
	"pkg/game/mapeditormodcontext.go": {"openModContext", "draftTerrain", "draftModContext"},
	"pkg/game/mapeditormodfiles.go":   {"filesOverrides", "modFileOverrides"},
	"pkg/game/modspeakers.go":         {"speakerDraftFiles", "draftSpeakersFile"},
	"pkg/game/mapedittestplay.go":     {"testPlayWith"},
}

// DraftViewCallers must call projectView; the others take a draftView.
var DraftViewCallers = map[string]bool{"projectFiles": true, "readInventory": true, "missionDraft": true,
	"draftModContext": true, "filesOverrides": true, "speakerDraftFiles": true, "draftSpeakersFile": true, "testPlayWith": true}

// DraftViewOverlays put the view over the run's records and so delete
// records; they are held to the other rules.
var DraftViewOverlays = map[string]bool{"draftTerrain": true}

// DraftViewFindings reports each reader in files (path to source) that has a
// second route to the drafts.
func DraftViewFindings(files map[string]string) ([]string, error) {
	var out []string
	for path, readers := range DraftViewFiles {
		src, ok := files[path]
		if !ok {
			return nil, fmt.Errorf("%s: not given", path)
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			return nil, err
		}
		found := map[string]bool{}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			for _, r := range readers {
				if fn.Name.Name != r {
					continue
				}
				found[r] = true
				calls, deletes, exact := false, false, false
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					if ix, ok := n.(*ast.IndexExpr); ok {
						if sel, ok := ix.X.(*ast.SelectorExpr); ok && sel.Sel.Name == "drafts" {
							exact = true
						}
					}
					if c, ok := n.(*ast.CallExpr); ok {
						switch fun := c.Fun.(type) {
						case *ast.Ident:
							calls = calls || fun.Name == "projectView"
						case *ast.SelectorExpr:
							deletes = deletes || fun.Sel.Name == "DeleteFunc" || fun.Sel.Name == "Delete"
						}
					}
					return true
				})
				if deletes && !DraftViewOverlays[r] {
					out = append(out, fmt.Sprintf("%s:%s deletes a name itself", path, r))
				}
				if exact {
					out = append(out, fmt.Sprintf("%s:%s reads a draft by its exact name", path, r))
				}
				if DraftViewCallers[r] && !calls {
					out = append(out, fmt.Sprintf("%s:%s does not call projectView", path, r))
				}
				for _, p := range fn.Type.Params.List {
					if DraftViewCallers[r] {
						break
					}
					if m, ok := p.Type.(*ast.MapType); ok {
						if _, isSlice := m.Value.(*ast.ArrayType); isSlice {
							out = append(out, fmt.Sprintf("%s:%s takes a draft map", path, r))
						}
					}
					if e, ok := p.Type.(*ast.Ellipsis); ok {
						if _, isMap := e.Elt.(*ast.MapType); isMap {
							out = append(out, fmt.Sprintf("%s:%s takes a draft map", path, r))
						}
					}
				}
			}
		}
		for _, r := range readers {
			if !found[r] {
				out = append(out, fmt.Sprintf("%s:%s is not found", path, r))
			}
		}
	}
	return out, nil
}
