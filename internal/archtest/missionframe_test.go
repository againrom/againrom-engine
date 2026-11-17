package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckMissionFrameNamesEachViolation drives the scan with planted
// copies of each refused shape.
func TestCheckMissionFrameNamesEachViolation(t *testing.T) {
	plan := "package ui\nvar missionLayers = []int{}\nfunc drawLayers() { for range missionLayers {} }\n" +
		"func headlessMissionFrame() { _ = missionLayers }\n"
	stage := "package ui\nfunc stageOf(f int) int { if f == filterSmooth { return 1 }; switch f { case filterPixel: return 2 }; return 0 }\n"
	cases := []struct {
		name, src, want string
	}{
		{"the plan and the rule", "", ""},
		{"a second filter comparison", "package ui\nfunc blit(f int) { if f != filterSmooth {} }\n", "compared outside framestage.go"},
		{"a filter switch", "package ui\nfunc blit(f int) { switch f { case filterSharp: } }\n", "switched on outside framestage.go"},
		{"a second walk", "package ui\nfunc drawAgain() { for range missionLayers {} }\n", "missionLayers used outside"},
		{"a copy of the plan", "package ui\nvar other = missionLayers\n", "missionLayers used outside"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{"pkg/ui/missionframe.go": plan, "pkg/ui/framestage.go": stage}
			if tc.src != "" {
				files["pkg/ui/a.go"] = tc.src
			}
			var got []string
			for _, v := range CheckMissionFrame(files) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckMissionFrame = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckMissionFrame = %v, want one naming %q", got, tc.want)
			}
		})
	}
	if vs := CheckMissionFrame(nil); len(vs) != 1 {
		t.Fatalf("CheckMissionFrame(nil) = %v, want one violation", vs)
	}
}

// TestMissionFrameHasOnePlanAndOneFilterRule is the live-tree half.
func TestMissionFrameHasOnePlanAndOneFilterRule(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckMissionFrame(files) {
		t.Errorf("%s", v)
	}
}

func TestMissionCardFitOwnsCharacterPaneModeFlag(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "pkg/ui/characterpane.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "characterPaneModeFlag" {
			continue
		}
		sharedFit := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "missionCardBoxRect" {
					sharedFit = true
				}
			}
			if expr, ok := n.(*ast.BinaryExpr); ok && expr.Op == token.SUB {
				t.Error("characterPaneModeFlag must not compute a separate room threshold")
			}
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.INT && lit.Value != "0" {
				t.Error("characterPaneModeFlag must not contain a height constant")
			}
			return true
		})
		if !sharedFit {
			t.Error("characterPaneModeFlag must read missionCardBoxRect")
		}
		return
	}
	t.Fatal("missing characterPaneModeFlag")
}
