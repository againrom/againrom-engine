package game

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/cityfixture"
	"againrom/pkg/formats/sav"
)

// milestone2InstrumentCarriers is the WriterCarriers bit that holds an
// admitted Againrom file to each of these corpus instruments
// (docs/SAV-SUPPLEMENT.md, "Milestone-2 admission"). An original file keeps
// every instrument. An instrument the carrier table there does not list has
// no bit, so an admitted Againrom file meets the writer round trip alone.
var milestone2InstrumentCarriers = map[string]uint32{
	"TestMilestone2ActorRoots":      0,
	"TestMilestone2UnitCombat":      0,
	"TestMilestone2UnitScalars":     0,
	"TestCellRecordCorpusAudit1131": currentCarrierCellRecords,
}

// Each instrument above selects its files through the writer admission: an
// original is held to it, an admitted Againrom file only where its
// WriterCarriers holds the instrument's bit, and a malformed supplement is
// an error.
func TestMilestone2InstrumentsSelectAgainromFilesByWriterCarriers(t *testing.T) {
	t.Setenv("AGAINROM_M2_ORIGINAL_CONTROL", "")
	original, err := cityfixture.Original(false)
	if err != nil {
		t.Fatal(err)
	}
	withLeaf := func(leaf []byte) milestone2File {
		doc, err := sav.DecodeDocumentData(original)
		if err != nil {
			t.Fatal(err)
		}
		if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
			t.Fatal(err)
		}
		raw, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		return milestone2File{rel: "synthetic", raw: raw, f: mustMilestone2Open(t, raw)}
	}
	written := func(bits uint32) milestone2File {
		leaf, err := json.Marshal(currentActionData{Version: 1, WriterCarriers: bits})
		if err != nil {
			t.Fatal(err)
		}
		return withLeaf(leaf)
	}
	for name, carrier := range milestone2InstrumentCarriers {
		if !milestone2CarrierApplies(t, milestone2File{rel: "original", raw: original, f: mustMilestone2Open(t, original)}, carrier) {
			t.Errorf("%s: an original file is not held to the instrument", name)
		}
		for _, bits := range []uint32{0, currentCarrierTerrainBlockPlane, currentCarrierAll &^ carrier, currentCarrierAll} {
			want := carrier != 0 && bits&carrier != 0
			if got := milestone2CarrierApplies(t, written(bits), carrier); got != want {
				t.Errorf("%s: Againrom file with WriterCarriers %#x held=%t, want %t", name, bits, got, want)
			}
		}
	}
	for _, leaf := range []string{`{"Version":2,"WriterCarriers":0}`, `{"Version":1,"WriterCarriers":4096}`, `{"Version":1,"WriterCarriers":0} {}`} {
		if _, admitted, err := milestone2WriterCarriers(withLeaf([]byte(leaf))); err == nil || admitted {
			t.Errorf("malformed supplement %s admitted=%t err=%v", leaf, admitted, err)
		}
	}
	milestone2InstrumentsUseAdmission(t)
}

// milestone2InstrumentsUseAdmission requires each instrument's own function
// to read its bit from milestone2InstrumentCarriers and pass its files
// through milestone2CarrierCorpus or milestone2CarrierApplies.
func milestone2InstrumentsUseAdmission(t *testing.T) {
	t.Helper()
	files, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		holds := false
		for name := range milestone2InstrumentCarriers {
			holds = holds || bytes.Contains(src, []byte("func "+name+"("))
		}
		if !holds {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if _, listed := milestone2InstrumentCarriers[fn.Name.Name]; !listed {
				continue
			}
			table, admits := false, false
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch v := n.(type) {
				case *ast.IndexExpr:
					if id, ok := v.X.(*ast.Ident); ok && id.Name == "milestone2InstrumentCarriers" {
						table = true
					}
				case *ast.CallExpr:
					if id, ok := v.Fun.(*ast.Ident); ok && (id.Name == "milestone2CarrierCorpus" || id.Name == "milestone2CarrierApplies") {
						admits = true
					}
				}
				return true
			})
			found[fn.Name.Name] = true
			if !table || !admits {
				t.Errorf("%s in %s does not select its files through the writer admission (carrier table %t, admission call %t)", fn.Name.Name, path, table, admits)
			}
		}
	}
	for name := range milestone2InstrumentCarriers {
		if !found[name] {
			t.Errorf("%s not found", name)
		}
	}
}
