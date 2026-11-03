package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// The cover-screen scan: a screen shown over a running mission is one row of
// coverRows (pkg/ui/cover.go), raised through one request on the Viewer. In
// pkg/ui production code it refuses:
//
//   - a field of the flow struct, of type bool or Screen, named for a screen
//     over the map (shopOverMap, docBack and the same shape);
//   - a bool field of the Viewer named for a documents or Shop request;
//   - more than one Raise method, or more than one Take...Request method, on
//     the Viewer.

var (
	coverFlowFieldName = regexp.MustCompile(`(?i)over[-_ ]?map|(doc|document|shop)\w*(back|over)\b|^(doc|document|shop)back$`)
	coverRequestName   = regexp.MustCompile(`(?i)^(doc|document|shop)\w*request$`)
	coverRaiseName     = regexp.MustCompile(`^Raise[A-Z]\w*$`)
	coverTakeName      = regexp.MustCompile(`^Take[A-Z]\w*Request$`)
)

// CheckCoverScreens reports every second path to a screen over the mission.
// files is keyed by module-relative slash path, as LoadDrawnTextSources
// returns it.
func CheckCoverScreens(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the cover-screen scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		if strings.HasPrefix(name, "pkg/ui/") && !strings.Contains(strings.TrimPrefix(name, "pkg/ui/"), "/") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var vs []Violation
	var raises, takes []string
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.GenDecl:
				for _, sp := range d.Specs {
					ts, ok := sp.(*ast.TypeSpec)
					if !ok {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok || (ts.Name.Name != "flow" && ts.Name.Name != "Viewer") {
						continue
					}
					for _, fld := range st.Fields.List {
						typ, ok := fld.Type.(*ast.Ident)
						if !ok {
							continue
						}
						for _, id := range fld.Names {
							switch {
							case ts.Name.Name == "flow" && (typ.Name == "bool" || typ.Name == "Screen") && coverFlowFieldName.MatchString(id.Name):
								vs = append(vs, Violation{From: at(id), Reason: "flow." + id.Name + " names a screen over the mission; it belongs in the cover record"})
							case ts.Name.Name == "Viewer" && typ.Name == "bool" && coverRequestName.MatchString(id.Name):
								vs = append(vs, Violation{From: at(id), Reason: "Viewer." + id.Name + " is a request for one screen over the mission; raise it through RaiseCover"})
							}
						}
					}
				}
			case *ast.FuncDecl:
				if d.Recv == nil || len(d.Recv.List) != 1 {
					continue
				}
				recv := d.Recv.List[0].Type
				if star, ok := recv.(*ast.StarExpr); ok {
					recv = star.X
				}
				if id, ok := recv.(*ast.Ident); !ok || id.Name != "Viewer" {
					continue
				}
				switch {
				case coverRaiseName.MatchString(d.Name.Name):
					raises = append(raises, at(d.Name)+" "+d.Name.Name)
				case coverTakeName.MatchString(d.Name.Name):
					takes = append(takes, at(d.Name)+" "+d.Name.Name)
				}
			}
		}
	}
	if len(raises) > 1 {
		vs = append(vs, Violation{From: "pkg/ui", Reason: "more than one Raise method on the Viewer: " + strings.Join(raises, ", ")})
	}
	if len(takes) > 1 {
		vs = append(vs, Violation{From: "pkg/ui", Reason: "more than one Take...Request method on the Viewer: " + strings.Join(takes, ", ")})
	}
	return vs
}
