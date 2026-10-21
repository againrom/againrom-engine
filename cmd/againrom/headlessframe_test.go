package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeScenario(t *testing.T, source string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "scenario.json")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// A scenario with a key the format does not name is refused before any install
// is read, and the refusal lists the keys it accepts.
func TestHeadlessRefusesAnUnknownKeyListingTheAcceptedOnes(t *testing.T) {
	path := writeScenario(t, `{"version": 10, "steps": [{"command": "frame", "file": "a.png"}]}`)
	code, out, stderr := checkRun(t, []string{"--headless", path}, "")
	if code != 2 || out != "" {
		t.Fatalf("exit %d, output %q", code, out)
	}
	for _, want := range []string{`unknown key "file"`, "accepted keys: command, target, key,", "path"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("%q is not in %q", want, stderr)
		}
	}
}

// A frame command naming a file inside the install is refused before the
// first step runs and writes nothing.
func TestHeadlessFrameIntoTheInstallIsRefused(t *testing.T) {
	root := defaultInstall(t)
	inside := filepath.Join(root, "frame.png")
	path := writeScenario(t, `{"version": 10, "steps": [{"command": "frame", "path": `+jsonQuote(inside)+`}]}`)
	code, out, stderr := checkRun(t, []string{"-assets", root, "--headless", path}, "")
	if code != 1 || out != "" || !strings.Contains(stderr, "step 1 (frame)") || !strings.Contains(stderr, "a frame is never written into an install") {
		t.Fatalf("exit %d, output %q, stderr %q", code, out, stderr)
	}
	if _, err := os.Lstat(inside); !os.IsNotExist(err) {
		t.Fatalf("the refused run left %s: %v", inside, err)
	}
}

func jsonQuote(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}
