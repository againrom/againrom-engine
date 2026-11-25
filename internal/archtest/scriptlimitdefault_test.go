package archtest

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScriptLimitDefaultIsOne(t *testing.T) {
	const first = "scriptprofile" + ".First"
	cases := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{"the dialect table", map[string]string{"pkg/sim/gameprofile.go": "package sim\nvar p = scriptprofile.For(scriptprofile.OpsFirst, false)\n"}, 0},
		{"a sim fallback", map[string]string{"pkg/sim/gameprofile.go": "package sim\nvar p = " + first + "\n"}, 1},
		{"a window fallback", map[string]string{"pkg/ui/mapeditorscriptmodel.go": "package ui\nvar p = " + first + "\n"}, 1},
		{"a package-wide editor variable", map[string]string{"pkg/mapedit/arm.go": "package mapedit\nvar script = " + first + "\n"}, 1},
		{"a record form in the editor", map[string]string{"pkg/mapedit/scriptform.go": "package mapedit\nfunc f() { x := " + first + " ; _ = x }\n"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ScriptLimitDefaultViolations(tc.files); len(got) != tc.want {
				t.Fatalf("violations = %q, want %d", got, tc.want)
			}
		})
	}
}

// TestScriptLimitDefaultInTheTree runs the scan over pkg.
func TestScriptLimitDefaultInTheTree(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	err = filepath.WalkDir(filepath.Join(root, "pkg"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := ScriptLimitDefaultViolations(files); len(got) != 0 {
		t.Fatalf("violations = %q", got)
	}
}
