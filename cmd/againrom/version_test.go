package main

import (
	"bytes"
	"strings"
	"testing"

	"againrom/internal/buildinfo"
)

func TestVersionFileIsValid(t *testing.T) {
	if _, err := buildinfo.ParseVersion(versionFile); err != nil {
		t.Fatal(err)
	}
	if programVersion() == "unknown" {
		t.Fatal("the embedded VERSION file is not readable as a version")
	}
}

func TestVersionFlagPrintsOneLineAndExits(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"-version"}, noEnv, &out, &errOut)
	want := "againrom " + programVersion() + " " + buildinfo.Revision() + "\n"
	if code != 0 || out.String() != want || errOut.Len() != 0 {
		t.Fatalf("exit %d, stdout %q (want %q), stderr %q", code, out.String(), want, errOut.String())
	}
}

func TestMenuLabelNamesVersionAndRevision(t *testing.T) {
	label := menuLabel()
	if !strings.HasPrefix(label, "Againrom "+programVersion()+" (") || !strings.HasSuffix(label, ")") {
		t.Fatalf("label %q", label)
	}
}
