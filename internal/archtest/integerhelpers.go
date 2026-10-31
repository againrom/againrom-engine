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

var integerIdents = map[string]bool{"int": true, "int8": true, "int16": true, "int32": true, "int64": true}

// CheckIntegerHelpers reports a local abs func or minInt under pkg/ and cmd/.
func CheckIntegerHelpers(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the integer helper scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/") && !strings.HasPrefix(name, "cmd/") || strings.HasPrefix(name, "pkg/intmath/") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			bad := fn.Name.Name == "minInt" || strings.Contains(strings.ToLower(fn.Name.Name), "abs") && oneIntegerInOut(fn.Type)
			if bad {
				vs = append(vs, Violation{From: name + ":" + strconv.Itoa(fset.Position(fn.Pos()).Line),
					Reason: "func " + fn.Name.Name + " is a second integer helper; use intmath.Abs or the built-in min"})
			}
		}
	}
	return vs
}

func oneIntegerInOut(t *ast.FuncType) bool {
	if t.Params == nil || t.Results == nil || len(t.Params.List) != 1 || len(t.Params.List[0].Names) != 1 || len(t.Results.List) != 1 {
		return false
	}
	in, ok1 := t.Params.List[0].Type.(*ast.Ident)
	out, ok2 := t.Results.List[0].Type.(*ast.Ident)
	return ok1 && ok2 && in.Name == out.Name && integerIdents[in.Name]
}

// LoadTreeSources reads every .go file, tests included, under the named dirs.
func LoadTreeSources(moduleRoot string, dirs ...string) (map[string]string, error) {
	out := make(map[string]string)
	for _, dir := range dirs {
		err := filepath.WalkDir(filepath.Join(moduleRoot, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
				return nil
			}
			b, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			rel, rerr := filepath.Rel(moduleRoot, path)
			if rerr != nil {
				return rerr
			}
			out[filepath.ToSlash(rel)] = string(b)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
