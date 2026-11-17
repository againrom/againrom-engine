package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
)

// The drafted kinds scan. A kind the map editor drafts is one row of
// dataKinds: its drafting (Draft) stands in the row beside its draft
// reader, and the editor's readers of the drafts take the kind's files from
// the rows and name none of a drafted kind's files themselves, so a new
// drafted kind is a row and no reader's branch.

// DraftKindTable is the file that declares dataKinds.
const DraftKindTable = "pkg/mod/kinds.go"

// DraftKindReaders are the files that hold the readers, each with its
// readers.
var DraftKindReaders = map[string][]string{
	"pkg/game/mapeditormod.go":           {"document"},
	"pkg/game/mapinspectionworkspace.go": {"contextDraft"},
	"pkg/game/modassetimport.go":         {"importedModFile"},
	"pkg/game/mapeditormodcontext.go":    {"openModContext", "draftTerrain"},
	"pkg/game/mapeditormodterrain.go":    {"editKind", "put", "drop", "heldRows", "editRows"},
}

// DraftKindNames are the package mod names of a drafted kind's files.
var DraftKindNames = []string{"ItemsFile", "TerrainFile", "SpeakersFile", "StringsFile", "FilesDir", "CommonFiles", "ItemSprite"}

// DraftKindKept are the names a reader keeps from before the drafted-kind row:
// document() keeps its own speakers load and the speakers-picture drop, named
// under the story's Open debt.
var DraftKindKept = map[string][]string{"document": {"SpeakersFile"}}

// DraftKindFindings reports each dataKinds row with a draft reader and no
// Draft, and each reader that names a drafted kind's file; files maps path
// to source.
func DraftKindFindings(files map[string]string) ([]string, error) {
	parse := func(path string) (*ast.File, error) {
		src, ok := files[path]
		if !ok {
			return nil, fmt.Errorf("%s: not given", path)
		}
		return parser.ParseFile(token.NewFileSet(), path, src, 0)
	}
	var out []string
	table, err := parse(DraftKindTable)
	if err != nil {
		return nil, err
	}
	rows := 0
	ast.Inspect(table, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok || len(spec.Names) != 1 || spec.Names[0].Name != "dataKinds" || len(spec.Values) != 1 {
			return true
		}
		lit, ok := spec.Values[0].(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, e := range lit.Elts {
			row, ok := e.(*ast.CompositeLit)
			if !ok {
				continue
			}
			rows++
			name, keys := "", map[string]bool{}
			for _, f := range row.Elts {
				kv, ok := f.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				if !ok {
					continue
				}
				keys[key.Name] = true
				if s, ok := kv.Value.(*ast.BasicLit); ok && key.Name == "Name" {
					name, _ = strconv.Unquote(s.Value)
				}
			}
			if (keys["draft"] || keys["lookup"]) && !keys["Draft"] {
				out = append(out, fmt.Sprintf("%s: row %s has a draft reader and no Draft", DraftKindTable, name))
			}
		}
		return false
	})
	if rows == 0 {
		out = append(out, DraftKindTable+": dataKinds is not found")
	}
	for path, readers := range DraftKindReaders {
		f, err := parse(path)
		if err != nil {
			return nil, err
		}
		found := map[string]bool{}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil || !slices.Contains(readers, fn.Name.Name) {
				continue
			}
			found[fn.Name.Name] = true
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "mod" && slices.Contains(DraftKindNames, sel.Sel.Name) && !slices.Contains(DraftKindKept[fn.Name.Name], sel.Sel.Name) {
					out = append(out, fmt.Sprintf("%s:%s names mod.%s", path, fn.Name.Name, sel.Sel.Name))
				}
				return true
			})
		}
		for _, r := range readers {
			if !found[r] {
				out = append(out, fmt.Sprintf("%s:%s is not found", path, r))
			}
		}
	}
	slices.Sort(out)
	return out, nil
}
