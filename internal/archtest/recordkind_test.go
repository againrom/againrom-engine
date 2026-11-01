package archtest

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLiveEditorWritesNoRecordKindWord measures the real tree: no editor
// root names a record kind by a string literal.
func TestLiveEditorWritesNoRecordKindWord(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	uses, err := LoadRecordKindUses(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, u := range uses {
		t.Error(u)
	}
	src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(recordKindFile)))
	if err != nil {
		t.Fatal(err)
	}
	words, err := RecordKindWords(src)
	if err != nil || len(words) < 10 {
		t.Fatalf("read %d kind words (%v); the constants moved", len(words), err)
	}
}

// TestRecordKindScanFindsEachShape plants one instance of each shape in a
// source and expects the scan to report that shape.
func TestRecordKindScanFindsEachShape(t *testing.T) {
	words := map[string]bool{"Unit": true, "Sack": true}
	cases := []struct{ shape, body string }{
		{"compare", `func f(r R) bool { return r.Kind == "Unit" }`},
		{"compare", `func f(kind string) bool { return "Unit" != kind }`},
		{"case", `func f(r R) { switch r.Kind { case "Unit": } }`},
		{"case", `func f(kind K) { switch kind { case "Sack", "Unit": } }`},
		{"field", `var v = R{Kind: "Unit"}`},
		{"assign", `func f() { kind := "Unit"; _ = kind }`},
		{"assign", `func f(r *R) { r.Kind = "Sack" }`},
		{"return", `func f() ui.RecordKind { return "Unit" }`},
		{"mapkey", `var m = map[K]int{"Unit": 1}`},
		{"argument", `func f(s S) { s.Fields("Unit", 0) }`},
	}
	for _, c := range cases {
		src := "package p\n" + c.body + "\n"
		got, err := ScanRecordKindSource("p.go", []byte(src), words)
		if err != nil {
			t.Fatalf("%s: %v", c.shape, err)
		}
		if len(got) == 0 || got[0].Shape != c.shape {
			t.Errorf("%s %q: found %v", c.shape, c.body, got)
		}
	}
}

// TestRecordKindScanLeavesTextAlone holds the scan against false positives.
func TestRecordKindScanLeavesTextAlone(t *testing.T) {
	words := map[string]bool{"Unit": true}
	for _, body := range []string{
		`func f(s S) string { return "Unit" }`,
		`func f(r R) bool { return r.Arm == "Unit" }`,
		`func f() { copy(b, "Unit") }`,
		`func f() error { return fmt.Errorf("Unit") }`,
		`func f(r R) bool { return r.Kind == ui.KindUnit }`,
		`var v = R{Label: "Unit"}`,
	} {
		got, err := ScanRecordKindSource("p.go", []byte("package p\n"+body+"\n"), words)
		if err != nil || len(got) != 0 {
			t.Errorf("%q: %v %v", body, got, err)
		}
	}
}
