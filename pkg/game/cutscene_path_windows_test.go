package game

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCutsceneWitnessPhysicalOutputFence(t *testing.T) {
	base := t.TempDir()
	root, outside := filepath.Join(base, "install"), filepath.Join(base, "evidence")
	inside := filepath.Join(root, "nested")
	for _, p := range []string{inside, outside} {
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	alias := filepath.Join(base, "alias")
	cmd := exec.Command("cmd", "/c", "mklink", "/J", alias, root)
	quietWitnessProcess(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create owned fixture junction: %v %s", err, out)
	}
	defer os.Remove(alias)
	f := &FrontEnd{}
	for _, target := range []string{root, inside, alias, filepath.Join(alias, "nested")} {
		if err := f.WitnessCutscene(root, "", target, io.Discard); err == nil || !strings.Contains(err.Error(), "output must be outside install") {
			t.Fatalf("unsafe output %s: %v", target, err)
		}
	}
	// A missing bank is reached only after the real directory fence admits
	// this path. No movie, output file, or original process is involved.
	if err := f.WitnessCutscene(root, "", outside, io.Discard); err == nil || !strings.Contains(err.Error(), "no configured bank") {
		t.Fatalf("safe outside directory was not admitted: %v", err)
	}
}
