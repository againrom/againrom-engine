package archtest

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestScriptProfileSelectorIsOne drives the scan with synthetic sources and
// then over the module's non-test Go sources.
func TestScriptProfileSelectorIsOne(t *testing.T) {
	const decl = "package scriptprofile\nfunc For(ops Ops, extended bool) Profile { return First }\n"
	cases := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{"one selector", map[string]string{"pkg/scriptprofile/profile.go": decl}, 0},
		{"a second declaration", map[string]string{"pkg/scriptprofile/profile.go": decl + "func Of(x bool) Profile { return First }\n"}, 1},
		{"a retired call", map[string]string{"pkg/game/a.go": "package game\nvar p = scriptprofile.Of(true)\n"}, 1},
		{"a retired ForOps call", map[string]string{"pkg/sim/a.go": "package sim\nvar p = scriptprofile.ForOps(0)\n"}, 1},
		{"the one selector", map[string]string{"pkg/game/a.go": "package game\nvar p = scriptprofile.For(scriptprofile.OpsFirst, true)\n"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ScriptProfileSelectorViolations(tc.files); len(got) != tc.want {
				t.Fatalf("violations = %q, want %d", got, tc.want)
			}
		})
	}
}

// TestScriptProfileSelectorInTheTree runs the scan over pkg, cmd and internal.
func TestScriptProfileSelectorInTheTree(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, dir := range []string{"pkg", "cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
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
	}
	if got := ScriptProfileSelectorViolations(files); len(got) != 0 {
		t.Fatalf("script profile selector violations: %q", got)
	}
}
