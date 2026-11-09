package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestClosureScanNamesEachForbiddenPackage(t *testing.T) {
	graph := map[string][]string{
		"cmd/x": {"pkg/a", "pkg/b"},
		"pkg/a": {"pkg/c", "pkg/b"},
		"pkg/b": {},
		"pkg/c": {"pkg/bad"},
		"pkg/d": {"pkg/bad2"},
	}
	vs := checkClosure("cmd/x", []string{"pkg/bad", "pkg/bad2"}, func(pkg string) ([]string, error) { return graph[pkg], nil })
	if len(vs) != 1 || !strings.Contains(vs[0].Reason, "cmd/x -> pkg/a -> pkg/c -> pkg/bad") {
		t.Fatalf("checkClosure = %v", vs)
	}
}

func TestStarterLinksNoGamePackage(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	direct, err := packageImports(root, starterRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(direct) == 0 {
		t.Fatalf("%s imports nothing of this module; the scan guards nothing", starterRoot)
	}
	for _, v := range CheckStarterClosure(root) {
		t.Errorf("%s: %s", v.From, v.Reason)
	}
}
