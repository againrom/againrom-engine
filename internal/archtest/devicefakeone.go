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

// The delivery device fake scan: internal/devicefake is the one controlled
// delivery device player for tests. Under pkg/ and cmd/ no test file declares
// a second type with the player's methods, and no file that is not a test
// imports the package.

const deviceFakePackage = ModulePath + "/internal/devicefake"

// LoadDeviceFakeSources reads every Go file under pkg/ and cmd/, tests
// included, keyed by module-relative slash path.
func LoadDeviceFakeSources(moduleRoot string) (map[string]string, error) {
	out := make(map[string]string)
	for _, top := range []string{"pkg", "cmd"} {
		err := filepath.WalkDir(filepath.Join(moduleRoot, top), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
				return err
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(moduleRoot, path)
			if err != nil {
				return err
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

// CheckDeviceFake reports a test type that has both IsPlaying and Seek
// methods, and a non-test file importing internal/devicefake.
func CheckDeviceFake(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the device fake scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		test := strings.HasSuffix(name, "_test.go")
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		for _, imp := range f.Imports {
			if path, _ := strconv.Unquote(imp.Path.Value); path == deviceFakePackage && !test {
				vs = append(vs, Violation{From: name, Reason: "a file that is not a test imports internal/devicefake"})
			}
		}
		if !test {
			continue
		}
		methods := map[string]map[string]bool{}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
				continue
			}
			recv := fn.Recv.List[0].Type
			if star, ok := recv.(*ast.StarExpr); ok {
				recv = star.X
			}
			if id, ok := recv.(*ast.Ident); ok {
				if methods[id.Name] == nil {
					methods[id.Name] = map[string]bool{}
				}
				methods[id.Name][fn.Name.Name] = true
			}
		}
		for typ, set := range methods {
			if set["IsPlaying"] && set["Seek"] {
				vs = append(vs, Violation{From: name, Reason: typ + " is a second delivery device fake in a test; use internal/devicefake.Player"})
			}
		}
	}
	return vs
}
