package sim

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func formVersionConstants(t *testing.T) map[string]struct {
	value byte
	file  string
} {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]struct {
		value byte
		file  string
	}{}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				vs := spec.(*ast.ValueSpec)
				for i, id := range vs.Names {
					if !strings.HasSuffix(id.Name, "FormVersion") || i >= len(vs.Values) {
						continue
					}
					lit, ok := vs.Values[i].(*ast.BasicLit)
					if !ok || lit.Kind != token.INT {
						t.Fatalf("%s: %s is not a literal byte", name, id.Name)
					}
					v, err := strconv.ParseUint(lit.Value, 0, 8)
					if err != nil {
						t.Fatalf("%s: %s: %v", name, id.Name, err)
					}
					out[id.Name] = struct {
						value byte
						file  string
					}{byte(v), name}
				}
			}
		}
	}
	return out
}

// Every FormVersion constant is in formWrappers with its encoder's tag.
func TestFormWrappersNameEveryWrapperVersion(t *testing.T) {
	consts := formVersionConstants(t)
	if len(consts) < len(formWrappers) {
		t.Fatalf("%d form version constants, %d wrappers", len(consts), len(formWrappers))
	}
	listed := map[byte]formWrapper{}
	for _, fw := range formWrappers {
		if _, dup := listed[fw.version]; dup {
			t.Fatalf("version %d listed twice", fw.version)
		}
		if fw.decode == nil {
			t.Fatalf("version %d has no decoder", fw.version)
		}
		listed[fw.version] = fw
	}
	for name, c := range consts {
		fw, ok := listed[c.value]
		if !ok {
			t.Errorf("%s (%d) in %s is not in formWrappers", name, c.value, c.file)
			continue
		}
		src, err := os.ReadFile(c.file)
		if err != nil {
			t.Fatal(err)
		}
		var quoted []string
		for _, r := range fw.tag {
			quoted = append(quoted, "'"+string(r)+"'")
		}
		if !strings.Contains(string(src), strings.Join(quoted, ", ")) {
			t.Errorf("%s: %s's encoder does not append tag %s", c.file, name, fw.tag)
		}
		delete(listed, c.value)
	}
	for v := range listed {
		t.Errorf("formWrappers names version %d, which no FormVersion constant declares", v)
	}
}

func TestHasStructureBlockingFormUnderAutoCast(t *testing.T) {
	w, err := NewStructuredWorld(7, Bounds{Width: 8, Height: 8}, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 3, Y: 3, HP: 100, MaxHP: 100}}, nil, Relations{}, nil, nil, nil, GhostTemplate{},
		[]Structure{{ID: 3, Col: 6, Row: 6, Width: 1, Height: 1, Attach: 1, Blocking: 2}})
	if err != nil {
		t.Fatal(err)
	}
	base, err := w.MarshalBinary()
	if err != nil || base[0] != structureBlockingFormVersion || !HasStructureBlockingForm(base) {
		t.Fatalf("base form %d err=%v", base[0], err)
	}
	if !w.SetAutoCast(0, AutoCastLevel(1)) {
		t.Fatal("automatic-casting level refused")
	}
	raw, err := w.MarshalBinary()
	if err != nil || raw[0] != autoCastFormVersion {
		t.Fatalf("form %d err=%v", raw[0], err)
	}
	if !HasStructureBlockingForm(raw) || CheckSaveForm(raw) != nil {
		t.Fatal("automatic-casting wrapper hid the structure blocking section")
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() || cold.Structures()[0].Blocking != 2 {
		t.Fatalf("cold load err=%v", err)
	}
}
