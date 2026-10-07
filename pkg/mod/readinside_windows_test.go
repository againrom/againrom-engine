package mod

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func modJunction(t *testing.T, link, target string) {
	t.Helper()
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
		t.Fatalf("junction: %v: %s", err, out)
	}
	t.Cleanup(func() { _ = os.Remove(link) })
}

func TestReadInsideResolvesWindowsJunctionTargets(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "mod")
	sibling := filepath.Join(base, "mod-sibling")
	for _, dir := range []string{filepath.Join(root, "data"), sibling} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "main.star"), []byte(filepath.Base(dir)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	modJunction(t, filepath.Join(base, "root-link"), root)
	modJunction(t, filepath.Join(root, "inside-link"), filepath.Join(root, "data"))
	modJunction(t, filepath.Join(root, "outside-link"), sibling)
	for _, name := range []string{"data/main.star", "inside-link/main.star"} {
		data, rel, err := ReadInside(filepath.Join(base, "root-link"), name)
		if err != nil || string(data) != "data" || rel != name {
			t.Fatalf("inside %s: %q %q %v", name, data, rel, err)
		}
	}
	for _, name := range []string{"outside-link/main.star", "../mod-sibling/main.star", "C:/outside/main.star", `data\main.star`} {
		data, _, err := ReadInside(root, name)
		if err == nil || data != nil {
			t.Fatalf("outside %s read %q: %v", name, data, err)
		}
		if name == "outside-link/main.star" && !strings.Contains(err.Error(), "outside the mod folder") {
			t.Fatalf("physical escape: %v", err)
		}
	}
}

func TestReadInsideResolvesWindowsFileSymlinkTargets(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "mod")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(root, "actual.star")
	outside := filepath.Join(base, "outside.star")
	for _, target := range []string{inside, outside} {
		if err := os.WriteFile(target, []byte(filepath.Base(target)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, link := range []struct{ name, target string }{{"inside.star", inside}, {"outside.star", outside}} {
		if err := os.Symlink(link.target, filepath.Join(root, link.name)); err != nil {
			t.Skipf("file symlinks unavailable: %v", err)
		}
	}
	if data, rel, err := ReadInside(root, "inside.star"); err != nil || string(data) != "actual.star" || rel != "inside.star" {
		t.Fatalf("inside symlink: %q %q %v", data, rel, err)
	}
	if data, _, err := ReadInside(root, "outside.star"); err == nil || data != nil || !strings.Contains(err.Error(), "outside the mod folder") {
		t.Fatalf("outside symlink: %q %v", data, err)
	}
}
