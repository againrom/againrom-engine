package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The host-file scan. Every file a player or a modder keeps is written by
// the one transaction in pkg/hostfile. Outside its transaction and platform
// primitive files, production code in pkg and in the game, editor and
// starter commands names none of the shapes that stage, publish or create a
// file, except a named entry below with its reason.

// hostFileOwners are the transaction file and its platform primitive files.
var hostFileOwners = map[string]bool{
	"pkg/hostfile/hostfile.go":        true,
	"pkg/hostfile/publish_windows.go": true,
	"pkg/hostfile/publish_other.go":   true,
	"pkg/hostfile/publish_link.go":    true,
}

// hostFileCommands are the commands the scan reads besides pkg.
var hostFileCommands = []string{"cmd/againrom/", "cmd/mapedit/", "cmd/starter/"}

// HostFileEntries are the allowed shapes outside the transaction, by file and
// shape, each with its reason. An entry that matches nothing is stale.
var HostFileEntries = map[string]string{
	"cmd/starter/app.go os.CreateTemp":                 "write probe: an empty temporary file is created and removed to test that the profile folder is writable; nothing is kept",
	"pkg/game/runtimeprofile.go os.CreateTemp":         "write probe: one byte is written to a temporary file that is removed to test that the profile folder is writable; nothing is kept",
	"pkg/game/cutscene_witness.go os.MkdirTemp":        "developer output: the cutscene witness's private input folder, removed when it ends",
	"pkg/game/cutscene_witness.go os.WriteFile":        "developer output: the cutscene witness's movie copy in its private folder",
	"pkg/game/cutscene_witness.go os.Create":           "developer output: the cutscene witness's frame pictures in the folder the developer names",
	"pkg/game/routewitness.go os.MkdirTemp":            "developer output: the route witness's scratch folder, removed when it ends",
	"pkg/game/routewitness.go os.WriteFile":            "developer output: the route witness's route.tsv in the folder the developer names",
	"pkg/ui/framelog.go os.OpenFile":                   "developer output: the frame timing log appended where the developer's flag names",
	"pkg/video/audioprobe_windows_386.go os.WriteFile": "developer output: the audio probe's decoded samples at the path the developer names",
	"cmd/mapedit/main.go os.WriteFile":                 "failure report: the editor's last error line in the profile folder when it has no console, overwritten on each failure",
	"cmd/starter/main.go os.Create":                    "developer output: the starter's -shot picture at the path the developer names",
}

// hostFileCalls are the os functions a shape is, with whether only a call
// whose flags create a file counts.
var hostFileCalls = map[string]bool{
	"Rename": false, "CreateTemp": false, "MkdirTemp": false, "Link": false,
	"WriteFile": false, "Create": false, "OpenFile": true,
}

// hostFileSyscalls are the system-call forms of the same shapes.
var hostFileSyscalls = map[string]bool{"Rename": true, "Link": true, "MoveFile": true, "CreateFile": true, "Open": true}

// hostFileCreateFlags are the flags that make OpenFile, Open or CreateFile
// create a file.
var hostFileCreateFlags = map[string]bool{"O_CREATE": true, "O_CREAT": true, "CREATE_NEW": true, "CREATE_ALWAYS": true, "OPEN_ALWAYS": true, "TRUNCATE_EXISTING": true}

// HostFileScope reports whether the scan reads production file name.
func HostFileScope(name string) bool {
	if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
		return false
	}
	if strings.HasPrefix(name, "pkg/") {
		return true
	}
	for _, c := range hostFileCommands {
		if strings.HasPrefix(name, c) {
			return true
		}
	}
	return false
}

// CheckHostFiles reports each shape outside the transaction that no entry
// names, and each entry that names nothing. hits counts the shapes read,
// the transaction's own included.
func CheckHostFiles(files map[string]string, entries map[string]string) (hits int, vs []string) {
	used := map[string]bool{}
	names := make([]string, 0, len(files))
	for name := range files {
		if HostFileScope(name) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, name+": "+err.Error())
			continue
		}
		for _, shape := range hostFileShapes(fset, f) {
			hits++
			if hostFileOwners[name] {
				continue
			}
			key := name + " " + shape.what
			if _, ok := entries[key]; ok {
				used[key] = true
				continue
			}
			vs = append(vs, name+":"+strconv.Itoa(shape.line)+": "+shape.what+" stages, publishes or creates a file; a kept file is a row of hostfile.Commit")
		}
	}
	for key := range entries {
		if !used[key] {
			vs = append(vs, "host-file entry "+key+" names nothing; remove it")
		}
	}
	sort.Strings(vs)
	return hits, vs
}

type hostFileShape struct {
	what string
	line int
}

// hostFileShapes lists the shapes in f: an os function of hostFileCalls (a
// call or a value; OpenFile only when its flags create or are not plain
// constants), O_EXCL, a system call of hostFileSyscalls, a MoveFileEx name
// in any identifier or string, and a dot import of os or syscall.
func hostFileShapes(fset *token.FileSet, f *ast.File) []hostFileShape {
	osName, sysNames := "", map[string]bool{}
	var out []hostFileShape
	add := func(n ast.Node, what string) {
		out = append(out, hostFileShape{what, fset.Position(n.Pos()).Line})
	}
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		local := path[strings.LastIndex(path, "/")+1:]
		if imp.Name != nil {
			local = imp.Name.Name
		}
		switch path {
		case "os":
			osName = local
		case "syscall", "golang.org/x/sys/windows", "golang.org/x/sys/unix":
			sysNames[local] = true
		default:
			continue
		}
		if local == "." {
			add(imp, "dot import of "+path)
		}
	}
	calls := map[*ast.SelectorExpr]*ast.CallExpr{}
	ast.Inspect(f, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			if sel, ok := c.Fun.(*ast.SelectorExpr); ok {
				calls[sel] = c
			}
		}
		return true
	})
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			pkg, ok := x.X.(*ast.Ident)
			if !ok {
				return true
			}
			name := x.Sel.Name
			switch {
			case name == "O_EXCL":
				add(x, pkg.Name+".O_EXCL")
			case pkg.Name == osName && osName != "":
				flagged, ok := hostFileCalls[name]
				if !ok {
					return true
				}
				if flagged {
					if c := calls[x]; c != nil && len(c.Args) > 1 && !createsFile(c.Args[1]) {
						return true
					}
				}
				add(x, "os."+name)
			case sysNames[pkg.Name] && hostFileSyscalls[name]:
				if c := calls[x]; (name == "CreateFile" || name == "Open") && c != nil && !anyArgCreates(c.Args) {
					return true
				}
				add(x, pkg.Name+"."+name)
			}
		case *ast.BasicLit:
			if x.Kind == token.STRING && strings.Contains(x.Value, "MoveFileEx") {
				add(x, "MoveFileEx")
			}
		case *ast.Ident:
			if strings.Contains(strings.ToLower(x.Name), "movefileex") {
				add(x, "MoveFileEx")
			}
		}
		return true
	})
	return out
}

// createsFile reports whether an OpenFile flag expression may create a file:
// it names a creating flag, or something other than plain flag constants.
func createsFile(e ast.Expr) bool {
	plain := true
	ast.Inspect(e, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if hostFileCreateFlags[x.Sel.Name] || !strings.HasPrefix(x.Sel.Name, "O_") {
				plain = false
			}
			return false
		case *ast.Ident:
			plain = false
		case *ast.CallExpr:
			plain = false
		}
		return true
	})
	return !plain
}

// anyArgCreates reports whether a system call's arguments name a creating
// flag or disposition.
func anyArgCreates(args []ast.Expr) bool {
	for _, a := range args {
		found := false
		ast.Inspect(a, func(n ast.Node) bool {
			if s, ok := n.(*ast.SelectorExpr); ok && hostFileCreateFlags[s.Sel.Name] {
				found = true
			}
			return !found
		})
		if found {
			return true
		}
	}
	return false
}
