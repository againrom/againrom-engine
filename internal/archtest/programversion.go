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

func CheckProgramVersion(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "cmd", Reason: "no sources scanned - the program version scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		if !strings.HasPrefix(name, "cmd/") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "programVersion" {
				vs = append(vs, Violation{From: name + ":" + strconv.Itoa(fset.Position(fn.Pos()).Line),
					Reason: "func programVersion under cmd/ is a second version identity; use buildinfo.Program"})
			}
		}
	}
	return vs
}

func LoadCmdSources(moduleRoot string) (map[string]string, error) {
	out := make(map[string]string)
	err := filepath.WalkDir(filepath.Join(moduleRoot, "cmd"), func(path string, d fs.DirEntry, err error) error {
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
	return out, err
}
