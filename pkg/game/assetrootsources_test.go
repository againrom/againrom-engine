package game

import (
	"os"
	"path/filepath"
	"testing"
)

// assetRootFixture lays out one install per source kind under a temporary
// folder, each holding only empty archive names, and returns the source that
// names each install with the root it must yield.
func assetRootFixture(t *testing.T) (sources map[AssetRootKind]AssetRootSource, roots map[AssetRootKind]string) {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	flagRoot := filepath.Join(base, "flag")
	envRoot := filepath.Join(base, "env")
	processRoot := writeInstallNames(t, filepath.Join(base, "process"), installArchiveNames)
	openedRoot := writeInstallNames(t, filepath.Join(base, "opened"), installArchiveNames)
	starterRoot := writeInstallNames(t, filepath.Join(base, "starter"), installArchiveNames)

	opened := filepath.Join(openedRoot, "maps", "a.alm")
	if err := os.MkdirAll(filepath.Dir(opened), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(opened, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := filepath.Join(base, "tools")
	if err := os.MkdirAll(tools, 0o755); err != nil {
		t.Fatal(err)
	}
	ini := "[starter]\nlast-base=a\n[bases]\na=" + starterRoot + "\n"
	if err := os.WriteFile(filepath.Join(tools, "starter.ini"), []byte(ini), 0o644); err != nil {
		t.Fatal(err)
	}

	sources = map[AssetRootKind]AssetRootSource{
		AssetRootFlag:           {AssetRootFlag, "  " + flagRoot + " "},
		AssetRootEnvironment:    {AssetRootEnvironment, envRoot},
		AssetRootProcessFolders: {AssetRootProcessFolders, filepath.Join(processRoot, "program.exe")},
		AssetRootOpenedFile:     {AssetRootOpenedFile, opened},
		AssetRootStarterBase:    {AssetRootStarterBase, filepath.Join(tools, "program.exe")},
	}
	roots = map[AssetRootKind]string{
		AssetRootFlag:           flagRoot,
		AssetRootEnvironment:    envRoot,
		AssetRootProcessFolders: processRoot,
		AssetRootOpenedFile:     openedRoot,
		AssetRootStarterBase:    starterRoot,
	}
	return sources, roots
}

// Every subset of the five sources, passed in the opposite of the precedence
// order, yields the root of its lowest kind: each source alone names its
// root, and of any two the earlier wins.
func TestFindAssetRootEarlierSourceWins(t *testing.T) {
	sources, roots := assetRootFixture(t)
	const kinds = int(AssetRootStarterBase) + 1
	for mask := 1; mask < 1<<kinds; mask++ {
		var admitted []AssetRootSource
		want := AssetRootKind(-1)
		for k := kinds - 1; k >= 0; k-- {
			if mask&(1<<k) == 0 {
				continue
			}
			admitted = append(admitted, sources[AssetRootKind(k)])
			want = AssetRootKind(k)
		}
		root, kind := FindAssetRoot(admitted...)
		if kind != want || root != roots[want] {
			t.Errorf("sources %05b: got %q from kind %d, want %q from kind %d", mask, root, kind, roots[want], want)
		}
	}
}

// A program admits only the sources it uses: the game's three never reach the
// opened file's install or the starter's base, and with nothing admitted
// there is no root.
func TestFindAssetRootAdmitsOnlyGivenSources(t *testing.T) {
	sources, _ := assetRootFixture(t)
	if root, _ := FindAssetRoot(sources[AssetRootOpenedFile], sources[AssetRootStarterBase]); root == "" {
		t.Fatal("the opened file and the starter base name no root")
	}
	game := []AssetRootSource{
		{AssetRootFlag, ""},
		{AssetRootEnvironment, "  "},
		{AssetRootProcessFolders, filepath.Join(t.TempDir(), "program.exe")},
	}
	if root, _ := FindAssetRoot(game...); root != "" {
		t.Errorf("the game's three sources named %q, want none", root)
	}
	if root, _ := FindAssetRoot(); root != "" {
		t.Errorf("no source named %q, want none", root)
	}
}

// The starter's base names a root only when it is an install, and only for a
// program path.
func TestFindAssetRootStarterBaseNeedsAnInstall(t *testing.T) {
	base := t.TempDir()
	ini := "[bases]\na=" + filepath.Join(base, "missing") + "\n"
	if err := os.WriteFile(filepath.Join(base, "starter.ini"), []byte(ini), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, self := range []string{"", filepath.Join(base, "program.exe")} {
		if root, _ := FindAssetRoot(AssetRootSource{AssetRootStarterBase, self}); root != "" {
			t.Errorf("starter base for %q named %q, want none", self, root)
		}
	}
}
