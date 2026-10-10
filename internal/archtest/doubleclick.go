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

// The double-click scan: outside the one detector (pkg/ui/doubleclick.go)
// and the system reading (pkg/ui/systemclick), production code under pkg/
// holds no second detector. It refuses:
//
//   - a field, variable or constant named for a click, tap or double whose
//     type or value is a time: a press time or a window;
//   - a declared name holding "double click", unless doubleClickNotADetector
//     or doubleClickDebt lists it with its reason;
//   - a reference to the system's double-click setting: GetDoubleClickTime,
//     SetDoubleClickTime, the rectangle's GetSystemMetrics indices or NSEvent's
//     doubleClickInterval.

// doubleClickNotADetector names the declarations that hold "double click"
// and pair no presses, keyed "file:name".
var doubleClickNotADetector = map[string]string{
	"pkg/ui/generator.go:DoubleClickForward": "the generator description's datum for what a hero's double click does",
}

// doubleClickDebt names the detectors not yet absorbed. It may only fall.
var doubleClickDebt = map[string]string{
	"pkg/ui/inventory.go:InventoryDoubleClickFrames": "the mission pack's and worn box's double click, counted in map frames",
}

func doubleClickDetectorFile(name string) bool {
	return name == "pkg/ui/doubleclick.go" || strings.HasPrefix(name, "pkg/ui/systemclick/")
}

var (
	doubleClickName  = regexp.MustCompile(`(?i)double[\s_-]*click`)
	clickPairingName = regexp.MustCompile(`(?i)click|tap|double`)
	systemClickNames = map[string]bool{
		"GetDoubleClickTime": true, "SetDoubleClickTime": true,
		"SM_CXDOUBLECLK": true, "SM_CYDOUBLECLK": true,
		"smCXDoubleClk": true, "smCYDoubleClk": true,
		"doubleClickInterval": true,
	}
	timeUnits = map[string]bool{"Nanosecond": true, "Microsecond": true, "Millisecond": true, "Second": true, "Minute": true}
)

// CheckDoubleClick reports every second detector outside the detector files,
// and every list entry nothing matches. files is keyed by module-relative
// slash path, as LoadDrawnTextSources returns it.
func CheckDoubleClick(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the double-click scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	seen := map[string]bool{}
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/") || doubleClickDetectorFile(name) {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		declared := func(id *ast.Ident, timed bool) {
			if id == nil || id.Name == "_" {
				return
			}
			if timed && clickPairingName.MatchString(id.Name) {
				vs = append(vs, Violation{From: at(id), Reason: id.Name + " holds a click time or window outside the one detector; read appInput.PrimaryDouble"})
				return
			}
			if !doubleClickName.MatchString(id.Name) {
				return
			}
			key := name + ":" + id.Name
			seen[key] = true
			if doubleClickNotADetector[key] == "" && doubleClickDebt[key] == "" {
				vs = append(vs, Violation{From: at(id), Reason: id.Name + " is a double-click detector outside pkg/ui/doubleclick.go; read appInput.PrimaryDouble"})
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncDecl:
				declared(n.Name, false)
			case *ast.TypeSpec:
				declared(n.Name, false)
			case *ast.Field:
				timed := isTimeType(n.Type)
				for _, id := range n.Names {
					declared(id, timed)
				}
			case *ast.ValueSpec:
				timed := isTimeType(n.Type)
				for _, v := range n.Values {
					timed = timed || mentionsTimeUnit(v)
				}
				for _, id := range n.Names {
					declared(id, timed)
				}
			case *ast.Ident:
				if systemClickNames[n.Name] {
					vs = append(vs, Violation{From: at(n), Reason: n.Name + " reads the system's double-click setting outside pkg/ui/systemclick"})
				}
			case *ast.BasicLit:
				if n.Kind == token.STRING {
					if s, err := strconv.Unquote(n.Value); err == nil && systemClickNames[s] {
						vs = append(vs, Violation{From: at(n), Reason: s + " reads the system's double-click setting outside pkg/ui/systemclick"})
					}
				}
			}
			return true
		})
	}
	stale := func(list map[string]string, what string) {
		keys := make([]string, 0, len(list))
		for k := range list {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if !seen[k] {
				vs = append(vs, Violation{From: "internal/archtest/doubleclick.go", Reason: k + " is listed as " + what + " but nothing matches it; remove the entry"})
			}
		}
	}
	stale(doubleClickNotADetector, "not a detector")
	stale(doubleClickDebt, "detector debt")
	return vs
}

// isTimeType reports whether e is time.Time or time.Duration, or a pointer
// to one.
func isTimeType(e ast.Expr) bool {
	if star, ok := e.(*ast.StarExpr); ok {
		e = star.X
	}
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "time" && (sel.Sel.Name == "Time" || sel.Sel.Name == "Duration")
}

// mentionsTimeUnit reports whether e names a time unit such as
// time.Millisecond.
func mentionsTimeUnit(e ast.Expr) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "time" && timeUnits[sel.Sel.Name] {
				found = true
			}
		}
		return !found
	})
	return found
}
