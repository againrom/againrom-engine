package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// SleepFile is the town's one Sleep: the runner that asks the town's shop to
// restock and the lookup of the town description's Sleep control row.
// Outside it, no production pkg/ code calls a shop's restock or names the
// Sleep control, and only the LOAD arrival draws a second-game stock.
const SleepFile = "pkg/game/sleep.go"

// The shapes SleepSites names.
const (
	ShapeRestockCall = "restock-call" // a call of restock or RestockWith
	ShapeStockDraw   = "stock-draw"   // a call of the second game's drawStock
	ShapeSleepName   = "sleep-name"   // the string "sleep", the control's name
)

// sleepRestockNames are the restock calls only the runner makes.
var sleepRestockNames = map[string]bool{"restock": true, "RestockWith": true}

// SleepAllowList names the sites the guard admits, keyed "file shape", with
// their count and reason.
var SleepAllowList = map[string]struct {
	Count  int
	Reason string
}{
	"pkg/game/campaignsecond.go " + ShapeStockDraw: {1, "a LOAD of a second-game town draws its stock anew before the shop is entered (DIV-3045); it is no Sleep"},
}

// SleepSites reports every restock call, second-game stock draw and Sleep
// control name outside the runner, one finding per site, less the allow
// list's admitted counts.
func SleepSites(files map[string]string) []Violation {
	allow := map[string]int{}
	for key, a := range SleepAllowList {
		allow[key] = a.Count
	}
	return sleepSitesAllowing(files, allow)
}

func sleepSitesAllowing(files map[string]string, allow map[string]int) []Violation {
	if _, ok := files[SleepFile]; !ok {
		return []Violation{{From: SleepFile, Reason: "the Sleep runner is missing; the scan guards nothing"}}
	}
	var vs []Violation
	for _, name := range sortedNames(files) {
		if !strings.HasPrefix(name, "pkg/") || name == SleepFile {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		report := func(n ast.Node, shape, reason string) {
			key := name + " " + shape
			if allow[key] > 0 {
				allow[key]--
				return
			}
			at := name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line)
			vs = append(vs, Violation{From: at + " " + shape, Reason: reason})
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CallExpr:
				fun := ""
				switch c := x.Fun.(type) {
				case *ast.Ident:
					fun = c.Name
				case *ast.SelectorExpr:
					fun = c.Sel.Name
				}
				switch {
				case sleepRestockNames[fun]:
					report(x, ShapeRestockCall, fun+" called outside "+SleepFile+"; a Sleep asks the town's shop through the runner")
				case fun == "drawStock":
					report(x, ShapeStockDraw, "drawStock called outside "+SleepFile+"; a Sleep restocks through the runner")
				}
			case *ast.BasicLit:
				if s, err := strconv.Unquote(x.Value); x.Kind == token.STRING && err == nil && s == "sleep" {
					report(x, ShapeSleepName, "the Sleep control's name outside "+SleepFile+"; ask sleepButton")
				}
			}
			return true
		})
	}
	return vs
}
