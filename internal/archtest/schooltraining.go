package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The school training scan: rules.SchoolPrice and rules.SchoolTrainedXP in
// pkg/rules/school.go state the school's price and trained experience, and
// production code under pkg/ computes neither a second time. It refuses:
//
//   - math.Pow with the literal base 1.1, the curve both values raise;
//   - a product of a 1.1-curve power (math.Pow(1.1, n), math.Pow(statBase, n)
//     or pow11) and the literal 200 or 1000, the price or experience scale;
//   - a SkillXP* call plus the literal 1, the experience a purchase stores.
const schoolTrainingRuleFile = "pkg/rules/school.go"

// CheckSchoolTraining reports every second school price or trained-experience
// computation. files is keyed by module-relative slash path, as
// LoadDrawnTextSources returns it.
func CheckSchoolTraining(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the school training scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/") || name == schoolTrainingRuleFile {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				if schoolCallName(n.Fun) == "Pow" && len(n.Args) == 2 && schoolLiteral(n.Args[0]) == "1.1" {
					vs = append(vs, Violation{From: at(n), Reason: "math.Pow(1.1, n) outside " + schoolTrainingRuleFile + "; read rules.SchoolPrice or rules.SchoolTrainedXP"})
				}
			case *ast.BinaryExpr:
				switch n.Op {
				case token.MUL:
					for _, pair := range [2][2]ast.Expr{{n.X, n.Y}, {n.Y, n.X}} {
						if scale := schoolLiteral(pair[1]); (scale == "200" || scale == "1000") && raisesSkillCurve(pair[0]) {
							vs = append(vs, Violation{From: at(n), Reason: "a 1.1-curve power times " + scale + " is a second school price or experience; read rules.SchoolPrice or rules.SchoolTrainedXP"})
						}
					}
				case token.ADD:
					for _, pair := range [2][2]ast.Expr{{n.X, n.Y}, {n.Y, n.X}} {
						if schoolLiteral(pair[1]) == "1" && isSkillXPCall(pair[0]) {
							vs = append(vs, Violation{From: at(n), Reason: "a skill experience threshold plus 1 is a second trained experience; read rules.SchoolTrainedXP"})
						}
					}
				}
			}
			return true
		})
	}
	return vs
}

func schoolCallName(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}

// schoolLiteral is the text of a numeric literal, parentheses removed, or "".
func schoolLiteral(e ast.Expr) string {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			break
		}
		e = p.X
	}
	if lit, ok := e.(*ast.BasicLit); ok && (lit.Kind == token.INT || lit.Kind == token.FLOAT) {
		return lit.Value
	}
	return ""
}

// raisesSkillCurve reports whether e holds a power of the 1.1 curve.
func raisesSkillCurve(e ast.Expr) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return !found
		}
		switch schoolCallName(call.Fun) {
		case "pow11":
			found = true
		case "Pow":
			if len(call.Args) == 2 {
				if id, ok := call.Args[0].(*ast.Ident); ok && id.Name == "statBase" || schoolLiteral(call.Args[0]) == "1.1" {
					found = true
				}
			}
		}
		return !found
	})
	return found
}

func isSkillXPCall(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	return ok && strings.HasPrefix(schoolCallName(call.Fun), "SkillXP")
}
