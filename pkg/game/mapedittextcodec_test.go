package game

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// openTextMap opens b in an editor whose font converts bytes (selector 1),
// the selector under which a Windows-1251 byte and the font's own byte for
// the same Cyrillic letter differ.
func openTextMap(t *testing.T, b []byte) (*MapInspector, *ui.MapEditor, string) {
	t.Helper()
	x := &MapInspector{tiles: &terrain.Tileset{}, rules: []mapRule{}, font: &text.Font{Selector: text.SelectorConverting}}
	path := filepath.Join(t.TempDir(), "names.alm")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	e.Document().Viewer.SetFlat(true)
	return x, e, path
}

// A trigger and a condition renamed with Cyrillic through the editor's own
// name fields read back, after Save As and a fresh open, as the text typed:
// the name fields offer it, the record list and the popup's name row carry
// the font's bytes for it, the script model the area label reads holds it,
// and the trigger accepts its own name again on the next Apply.
func TestCyrillicScriptNamesReadBackAsTyped(t *testing.T) {
	const trigger, condition = "Победа", "Смерть героя"
	_, e, path := openTextMap(t, goalMap(t, false))
	set := e.Document().Edits.(ui.InspectionFieldSet)
	if _, err := set.SetFields("Trigger", 0, []ui.InspectionFieldValue{{Key: "name", Text: true, TextValue: trigger}}); err != nil {
		t.Fatal(err)
	}
	if _, err := set.SetFields("Condition", 0, []ui.InspectionFieldValue{{Key: "name", Text: true, TextValue: condition}}); err != nil {
		t.Fatal(err)
	}
	out := path + ".out.alm"
	if _, err := e.Document().Edits.SaveAs(out); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(saved)
	if err != nil {
		t.Fatal(err)
	}
	sc, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	if sc.Triggers[0].Name != trigger || sc.Conditions[0].Label != condition {
		t.Errorf("the map reads back trigger %q and condition %q", sc.Triggers[0].Name, sc.Conditions[0].Label)
	}

	x, re, _ := openTextMap(t, saved)
	d := re.Document()
	trig, cond := trigRecord(t, re, "Trigger", 0), trigRecord(t, re, "Condition", 0)
	if got, want := d.Records[trig].Label, x.inspectionUTF8(trigger); got != want {
		t.Errorf("the record list receives trigger % x, want the font's % x", got, want)
	}
	if got, want := d.Records[cond].Label, x.inspectionUTF8(condition); got != want {
		t.Errorf("the record list receives condition % x, want the font's % x", got, want)
	}
	if got := d.Script.Triggers[0].Name; got != trigger {
		t.Errorf("the area label and list row read trigger % x, want %q", got, trigger)
	}
	if got := d.Script.Conditions[0].Label; got != condition {
		t.Errorf("the node popup reads condition % x, want %q", got, condition)
	}
	for _, f := range d.Edits.Fields("Trigger", 0) {
		if f.Key == "name" && f.TextValue != trigger {
			t.Errorf("the name field offers % x, want %q", f.TextValue, trigger)
		}
	}
	re.Select(trig, true)
	if lines := trigLines(re); !strings.Contains(lines, x.inspectionUTF8(trigger)) {
		t.Errorf("the trigger popup's name row lacks the font's bytes for %q:\n% x", trigger, lines)
	}
	before := append([]byte(nil), d.SourceBytes...)
	next, err := d.Edits.(ui.InspectionFieldSet).SetFields("Trigger", 0, []ui.InspectionFieldValue{
		{Key: "name", Text: true, TextValue: d.Script.Triggers[0].Name}, {Key: "once", Value: 0}})
	if err != nil {
		t.Fatalf("the trigger refuses its own name on Apply: %v", err)
	}
	if bytes.Equal(next.SourceBytes, before) {
		t.Fatal("Apply wrote nothing")
	}
}

// Text reaches the install's bytes through textinput.EncodeText alone. The
// per-rune encoder stays where one typed rune is asked about: the name field's
// and the words' rune encoders, and the mod text check.
func TestInstallTextGoesThroughTheOneCodec(t *testing.T) {
	allowed := map[string]bool{"ChargenSetup": true, "App": true, "ModTextDropped": true}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range parsed.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ast.Inspect(fn, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "EncodeRune" {
					return true
				}
				if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "textinput" && !allowed[fn.Name.Name] {
					t.Errorf("%s: %s encodes rune by rune around textinput.EncodeText", file, fn.Name.Name)
				}
				return true
			})
		}
	}
}
