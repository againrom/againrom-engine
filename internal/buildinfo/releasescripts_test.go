package buildinfo

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// powershell returns the PowerShell the release workflows use, or skips.
func powershell(t *testing.T) string {
	t.Helper()
	for _, name := range []string{"pwsh", "powershell"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	t.Skip("no PowerShell on this machine; the Windows release job runs the scripts")
	return ""
}

// parseErrors returns the PowerShell parser's errors for one script file.
func parseErrors(t *testing.T, shell, path string) string {
	t.Helper()
	command := `$e = $null; $t = $null; [void][System.Management.Automation.Language.Parser]::ParseFile($env:SCRIPT_UNDER_CHECK, [ref]$t, [ref]$e); $e | ForEach-Object { $_.Message + ' at line ' + $_.Extent.StartLineNumber }`
	cmd := exec.Command(shell, "-NoProfile", "-NonInteractive", "-Command", command)
	cmd.Env = append(os.Environ(), "SCRIPT_UNDER_CHECK="+path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("parse %s: %v\n%s", path, err, out)
	}
	return strings.TrimSpace(string(out))
}

// workflowScripts lists the scripts/*.ps1 files the release workflows name.
func workflowScripts(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob("../../.github/workflows/*.yml")
	if err != nil || len(files) == 0 {
		t.Fatalf("no release workflows found: %v", err)
	}
	pattern := regexp.MustCompile(`scripts/[A-Za-z0-9_.-]+\.ps1`)
	seen := map[string]bool{}
	var list []string
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range pattern.FindAllString(string(data), -1) {
			if !seen[name] {
				seen[name] = true
				list = append(list, name)
			}
		}
	}
	return list
}

// A release script that does not parse fails every tag before any build, so
// each PowerShell script a release workflow calls must parse.
func TestWorkflowPowerShellScriptsParse(t *testing.T) {
	shell := powershell(t)
	list := workflowScripts(t)
	found := false
	for _, name := range list {
		if name == "scripts/release-notes.ps1" {
			found = true
		}
		if errs := parseErrors(t, shell, filepath.Join("../..", name)); errs != "" {
			t.Errorf("%s does not parse:\n%s", name, errs)
		}
	}
	if !found {
		t.Fatalf("the workflows no longer name scripts/release-notes.ps1: %v", list)
	}
}

// Loss control: a single-quoted string holding an unescaped apostrophe, the
// defect that stopped every release, is reported by the same parse check.
func TestParseCheckCatchesAnApostropheInASingleQuotedString(t *testing.T) {
	shell := powershell(t)
	broken := filepath.Join(t.TempDir(), "broken.ps1")
	body := "$install = 'opens from the starter's Map editor button.'\n\"$install\"\n"
	if err := os.WriteFile(broken, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if errs := parseErrors(t, shell, broken); errs == "" {
		t.Fatal("the parse check passed a script with an unescaped apostrophe")
	}
	fixed := filepath.Join(t.TempDir(), "fixed.ps1")
	if err := os.WriteFile(fixed, []byte(strings.Replace(body, "starter's", "starter''s", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if errs := parseErrors(t, shell, fixed); errs != "" {
		t.Fatalf("the doubled apostrophe does not parse: %s", errs)
	}
}

// The notes script prints the committed 0.119.0 section and the install text.
func TestReleaseNotesScriptPrintsAVersionSection(t *testing.T) {
	shell := powershell(t)
	cmd := exec.Command(shell, "-NoProfile", "-NonInteractive", "-File", "scripts/release-notes.ps1",
		"-Version", "0.119.0", "-Revision", "0123456789ab")
	cmd.Dir = "../.."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("release-notes.ps1 -Version 0.119.0: %v\n%s", err, out)
	}
	text := string(out)
	head, tail, ok := strings.Cut(text, "\n---\n")
	if !ok || strings.TrimSpace(head) == "" {
		t.Fatalf("no changelog section before the install text:\n%s", text)
	}
	if !strings.Contains(tail, "starter's Map editor button") || !strings.Contains(tail, "Source commit: 0123456789ab") {
		t.Fatalf("install text or revision missing:\n%s", tail)
	}
}
