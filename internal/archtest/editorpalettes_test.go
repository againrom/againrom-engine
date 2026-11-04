package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// paletteTableFile is the one table outside pkg/ui that decides the bar
// palette a structure kind or an object class lists in.
const paletteTableFile = "pkg/game/mapeditorpalettes.go"

// speakerPickFile and speakerLineFile are the places that ask the host which
// speaker record shows a unit: the map editor's parameter pick and the unit
// popup's speaker line.
const (
	speakerPickFile = "pkg/ui/mapeditorparampick.go"
	speakerLineFile = "pkg/game/mapinspectionidentity.go"
)

var paletteNames = map[string]bool{"PaletteStructures": true, "PaletteObjects": true, "PaletteTrees": true}

// checkEditorPalettes reports, in one non-test source file, a palette named
// outside pkg/ui and the palette table, and a speaker asked for a unit
// outside the parameter pick and the unit popup's speaker line.
func checkEditorPalettes(name, src string) []string {
	f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.SkipObjectResolution)
	if err != nil {
		return []string{name + ": " + err.Error()}
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			if paletteNames[n.Sel.Name] && !strings.HasPrefix(name, "pkg/ui/") && name != paletteTableFile {
				out = append(out, name+": names "+n.Sel.Name+" outside the palette table")
			}
		case *ast.CallExpr:
			if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "UnitSpeaker" && name != speakerPickFile && name != speakerLineFile {
				out = append(out, name+": asks UnitSpeaker outside the parameter pick and the unit popup")
			}
		}
		return true
	})
	return out
}

func TestEditorPalettesAndSpeakerPickHaveOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{paletteTableFile, speakerPickFile, speakerLineFile} {
		if _, ok := files[want]; !ok {
			t.Fatalf("%s is missing: the guard has nothing to hold", want)
		}
	}
	for name, src := range files {
		for _, v := range checkEditorPalettes(name, src) {
			t.Error(v)
		}
	}
}

func TestEditorPaletteGuardSeesASecondBuilder(t *testing.T) {
	for _, tc := range []struct {
		name, src string
		want      int
	}{
		{paletteTableFile, "package game\nfunc f() { _ = ui.PaletteTrees }", 0},
		{"pkg/ui/mapeditorplace.go", "package ui\nfunc f() { _ = x.PaletteObjects }", 0},
		{"pkg/game/mapinspectionbrush.go", "package game\nfunc f() { _ = ui.PaletteTrees }", 1},
		{speakerPickFile, "package ui\nfunc f() { h.UnitSpeaker(1) }", 0},
		{speakerLineFile, "package game\nfunc f() { s.UnitSpeaker(1) }", 0},
		{"pkg/ui/mapeditormessage.go", "package ui\nfunc f() { h.UnitSpeaker(1) }", 1},
		{"pkg/game/mapinspectionnpc.go", "package game\nfunc (s *edits) UnitSpeaker(int) {}", 0},
	} {
		if got := checkEditorPalettes(tc.name, tc.src); len(got) != tc.want {
			t.Errorf("%s: %d findings %v, want %d", tc.name, len(got), got, tc.want)
		}
	}
}
