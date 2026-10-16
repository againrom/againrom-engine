package sim

import (
	"bytes"
	"encoding/binary"
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

// Every FormVersion constant is in formWrappers, and its codec writes and
// reads its envelope through the shared helpers.
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
		_, ok := listed[c.value]
		if !ok {
			t.Errorf("%s (%d) in %s is not in formWrappers", name, c.value, c.file)
			continue
		}
		src, err := os.ReadFile(c.file)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(src), "endWrapper(") || !strings.Contains(string(src), "readWrapper(data, ") {
			t.Errorf("%s: %s's codec does not use the shared wrapper envelope", c.file, name)
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

// An envelope reads back what endWrapper wrote and refuses a changed tag,
// span or base version; the terminal actor wrapper admits any base.
func TestWrapperEnvelopeRoundTrip(t *testing.T) {
	plain := make([]byte, headerLen+3)
	plain[0] = formatVersion
	section := []byte{1, 2, 3, 4, 5, 6}
	for _, fw := range formWrappers {
		ws := beginWrapper(plain)
		b := endWrapper(append(bytes.Clone(plain), section...), ws, fw.version)
		e, ok := readWrapper(b, fw.version)
		if b[0] != fw.version || !ok || !bytes.Equal(e.section(), section) || !bytes.Equal(e.base(), plain) {
			t.Fatalf("%s: envelope %v did not read back", fw.tag, b)
		}
		if v, inner, ok := fw.inner(b); !ok || v != formatVersion || len(inner) != len(plain) {
			t.Fatalf("%s: inner form lost", fw.tag)
		}
		tag := bytes.Clone(b)
		tag[len(tag)-1] ^= 0x20
		span := bytes.Clone(b)
		switch fw.trailer {
		case trailerSpan32:
			binary.LittleEndian.PutUint32(span[len(span)-9:], uint32(len(span)-headerLen-8))
		case trailerPairs8:
			span[len(span)-6] = byte((len(span)-headerLen-6)/2 + 1)
		case trailerSpan16:
			binary.LittleEndian.PutUint16(span[len(span)-6:], uint16(len(span)-headerLen-5))
		}
		for name, bad := range map[string][]byte{"tag": tag, "span": span} {
			if _, ok := readWrapper(bad, fw.version); ok {
				t.Errorf("%s: changed %s admitted", fw.tag, name)
			}
		}
		if fw.trailer == trailerSpan16 {
			continue
		}
		base := bytes.Clone(b)
		base[len(base)-5] = fw.version
		if _, ok := readWrapper(base, fw.version); ok != (fw.tag == "CTA1") {
			t.Errorf("%s: own version as base admitted=%t", fw.tag, ok)
		}
	}
}
