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

// The cadence scan: in pkg/ui, outside the one cadence file
// (pkg/ui/cadence.go), no function decides "is it time to advance or show"
// with its own clock test. It refuses, in a function the allow list below
// does not name:

//   - a comparison (<, <=, >, >=) with now.Sub(...), time.Since(...) or a
//     variable assigned from one, whatever it is compared with;
//   - a call to After or Before with one argument (a deadline test);
//   - a comparison of a subtraction of clock readings against a period, gap,
//     interval, delay, blink or wait (millisecond clocks);
//   - a modulo of a tick, frame, clock or elapsed count by a period, cycle,
//     interval or blink.

// A presentation cadence is asked through cadence.due, cadence.dueMillis or
// cadenceGate.reached. The allow list holds the elapsed-time measurements:
// ages, lifetimes, timeouts, windows and clock-step guards.

// cadenceMeasurements names the functions that compare elapsed time and are
// not cadences, keyed "file:Func" or "file:Type.Func", with the reason.
var cadenceMeasurements = map[string]string{
	"pkg/ui/doubleclick.go:doubleClick.inside":              "the pair's time window is a measurement of one press against the last",
	"pkg/ui/messageline.go:messageList.tick":                "accumulates the elapsed span a line has lived",
	"pkg/ui/numeral.go:Viewer.stepNumerals":                 "shifts birth stamps by the paused span",
	"pkg/ui/numeral.go:Viewer.expireNumerals":               "a numeral's age against its life",
	"pkg/ui/chargen_sparkle.go:Chargen.advancePresentation": "sums the clamped elapsed span since the last paint",
	"pkg/ui/tooltip.go:tooltipController.observe":           "a clock that stepped back restamps the hover",
	"pkg/ui/widgetedit.go:caretBlink.tick":                  "a clock that stepped back restamps the caret",
	"pkg/ui/viewer.go:Viewer.RestoreScheduledLightClock":    "aligns a simulation sub-tick to its period, not a presentation clock",
	"pkg/ui/framelog.go:frameLog.Draw":                      "the frame log's one-second flush window, a diagnostic measurement",
}

var (
	cadencePeriodName = regexp.MustCompile(`(?i)period|gap|interval|delay|blink|wait`)
	cadenceClockName  = regexp.MustCompile(`(?i)now|millis|last|tick|stamp|^at$|since`)
	cadenceCountName  = regexp.MustCompile(`(?i)tick|frame|clock|elapsed|millis`)
	cadenceCycleName  = regexp.MustCompile(`(?i)period|cycle|interval|blink`)
)

func cadenceFile(name string) bool { return name == "pkg/ui/cadence.go" }

// CheckCadence reports every presentation clock test outside the cadence
// file, and every allow-list entry nothing needs. files is keyed by
// module-relative slash path, as LoadDrawnTextSources returns it.
func CheckCadence(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg/ui", Reason: "no sources scanned - the cadence scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	used := map[string]bool{}
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/ui/") || cadenceFile(name) || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			key := name + ":" + cadenceFuncName(fn)
			for _, hit := range cadenceHits(fn) {
				if cadenceMeasurements[key] != "" {
					used[key] = true
					continue
				}
				line := strconv.Itoa(fset.Position(hit.pos).Line)
				vs = append(vs, Violation{From: name + ":" + line, Reason: hit.what + " in " + cadenceFuncName(fn) + " is a presentation clock test outside pkg/ui/cadence.go; ask a cadence"})
			}
		}
	}
	keys := make([]string, 0, len(cadenceMeasurements))
	for k := range cadenceMeasurements {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !used[k] {
			vs = append(vs, Violation{From: "internal/archtest/cadence.go", Reason: k + " is listed as a measurement but nothing there needs it; remove the entry"})
		}
	}
	return vs
}

type cadenceHit struct {
	pos  token.Pos
	what string
}

func cadenceFuncName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	t := fn.Recv.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name + "." + fn.Name.Name
	}
	return fn.Name.Name
}

// cadenceElapsedCall reports whether e is x.Sub(y) or time.Since(y).
func cadenceElapsedCall(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name == "Sub" && len(call.Args) == 1 {
		return true
	}
	pkg, isPkg := sel.X.(*ast.Ident)
	return isPkg && pkg.Name == "time" && sel.Sel.Name == "Since"
}

func cadenceIdents(e ast.Expr) []string {
	var out []string
	ast.Inspect(e, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			out = append(out, n.Name)
		case *ast.SelectorExpr:
			out = append(out, n.Sel.Name)
		}
		return true
	})
	return out
}

func cadenceAny(names []string, re *regexp.Regexp) bool {
	for _, n := range names {
		if re.MatchString(n) {
			return true
		}
	}
	return false
}

func cadenceHits(fn *ast.FuncDecl) []cadenceHit {
	// Variables assigned from an elapsed call hold elapsed time.
	elapsed := map[string]bool{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if as, ok := n.(*ast.AssignStmt); ok && len(as.Lhs) == len(as.Rhs) {
			for i, r := range as.Rhs {
				if id, ok := as.Lhs[i].(*ast.Ident); ok && cadenceElapsedCall(r) {
					elapsed[id.Name] = true
				}
			}
		}
		return true
	})
	isElapsed := func(e ast.Expr) bool {
		for {
			p, ok := e.(*ast.ParenExpr)
			if !ok {
				break
			}
			e = p.X
		}
		if cadenceElapsedCall(e) {
			return true
		}
		id, ok := e.(*ast.Ident)
		return ok && elapsed[id.Name]
	}
	var hits []cadenceHit
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BinaryExpr:
			switch n.Op {
			case token.LSS, token.LEQ, token.GTR, token.GEQ:
				if isElapsed(n.X) || isElapsed(n.Y) {
					hits = append(hits, cadenceHit{n.Pos(), "an elapsed-time comparison"})
					break
				}
				for _, side := range []ast.Expr{n.X, n.Y} {
					sub, ok := ast.Unparen(side).(*ast.BinaryExpr)
					if ok && sub.Op == token.SUB && cadenceAny(cadenceIdents(sub), cadenceClockName) && cadenceAny(cadenceIdents(n), cadencePeriodName) {
						hits = append(hits, cadenceHit{n.Pos(), "a clock difference compared with a period"})
						break
					}
				}
			case token.REM:
				if cadenceAny(cadenceIdents(n.X), cadenceCountName) && cadenceAny(cadenceIdents(n.Y), cadenceCycleName) {
					hits = append(hits, cadenceHit{n.Pos(), "a count modulo a period"})
				}
			}
		case *ast.CallExpr:
			if sel, ok := n.Fun.(*ast.SelectorExpr); ok && len(n.Args) == 1 && (sel.Sel.Name == "After" || sel.Sel.Name == "Before") {
				hits = append(hits, cadenceHit{n.Pos(), "a deadline test (" + sel.Sel.Name + ")"})
			}
		}
		return true
	})
	return hits
}
