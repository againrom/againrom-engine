package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func scopeRowSources(t *testing.T) map[string]string {
	t.Helper()
	root := moduleRootForTest(t)
	files := map[string]string{}
	for _, path := range ScopeRowFiles {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		files[path] = string(b)
	}
	return files
}

func TestModWindowPageScopesAreOneRowOnEachSide(t *testing.T) {
	got, err := ScopeRowFindings(scopeRowSources(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range got {
		t.Error(f)
	}
}

func TestModWindowPageScopeBranchesAreFound(t *testing.T) {
	files := scopeRowSources(t)
	ui, game := ScopeRowFiles[0], ScopeRowFiles[1]
	files[ui] = strings.Replace(files[ui], "\tModFilesScope = \"installfiles\"\n", "\tModFilesScope = \"installfiles\"\n\tModProbeScope = \"probe\"\n", 1)
	files[game] = strings.Replace(files[game], "\tif r, ok := pageScope(scope); ok {\n\t\treturn r.fields(s, scope)\n\t}\n",
		"\tif r, ok := pageScope(scope); ok {\n\t\treturn r.fields(s, scope)\n\t}\n\tif scope == ui.ModProbeScope {\n\t\treturn nil\n\t}\n", 1)
	got, err := ScopeRowFindings(files)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(got, "\n")
	for _, want := range []string{"ModProbeScope is not a row of modScopePages", "ModProbeScope is not a row of workspaceScopes", "ui.ModProbeScope is named outside workspaceScopes"} {
		if !strings.Contains(joined, want) {
			t.Errorf("findings %q lack %q", joined, want)
		}
	}
}
