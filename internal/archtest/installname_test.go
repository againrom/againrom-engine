package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestNoProductionJoinBuildsAnInstallPath(t *testing.T) {
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
	if _, ok := files["pkg/game/archives.go"]; !ok {
		t.Fatal("the scan does not reach pkg/game/archives.go")
	}
	for _, f := range CheckInstallNames(files) {
		t.Error(f)
	}
}

func TestCheckInstallNamesFindsHandBuiltInstallPaths(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		bad          bool
	}{
		{"archive constant", `package p; import "path/filepath"; const SfxArchive = "sfx.res"; func f(root string) string { return filepath.Join(root, SfxArchive) }`, true},
		{"qualified constant", `package p; import ("path/filepath"; "x/game"); func f(root string) string { return filepath.Join(root, game.MainArchive) }`, true},
		{"archive literal", `package p; import "path/filepath"; func f(root string) string { return filepath.Join(root, "Speech.RES") }`, true},
		{"install root field", `package p; import "path/filepath"; type A struct{ Root string }; func f(a A, n string) string { return filepath.Join(a.Root, n) }`, true},
		{"profile file", `package p; import "path/filepath"; func f(dir string) string { return filepath.Join(dir, "options.txt") }`, false},
		{"other join", `package p; import "strings"; func f() string { return strings.Join([]string{"a.res"}, "/") }`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckInstallNames(map[string]string{"pkg/p/p.go": tc.source})
			if bad := len(got) > 0; bad != tc.bad {
				t.Fatalf("findings %q, want bad=%v", got, tc.bad)
			}
			if tc.bad && !strings.Contains(got[0], "pkg/installpath") {
				t.Fatalf("finding %q does not name the resolver", got[0])
			}
		})
	}
	if got := CheckInstallNames(map[string]string{"pkg/installpath/r.go": `package installpath; import "path/filepath"; func f(r string) string { return filepath.Join(r, "x.res") }`}); len(got) != 0 {
		t.Fatalf("the resolver itself was flagged: %q", got)
	}
}
