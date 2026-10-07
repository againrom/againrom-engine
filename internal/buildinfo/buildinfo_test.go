package buildinfo

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestParseVersion(t *testing.T) {
	for in, want := range map[string]string{"0.9.0\n": "0.9.0", " 12.0.345 ": "12.0.345", "1.2.3\r\n": "1.2.3"} {
		if got, err := ParseVersion(in); err != nil || got != want {
			t.Errorf("%q: %q %v", in, got, err)
		}
	}
	for _, in := range []string{"", "1.2", "1.2.3.4", "01.2.3", "a.b.c", "1.2.3-rc", "1.2.70000", "1.2.3\n4.5.6"} {
		if got, err := ParseVersion(in); err == nil {
			t.Errorf("%q accepted as %q", in, got)
		}
	}
}

func TestLineAndShort(t *testing.T) {
	line := Line("tool", "1.2.3")
	if !strings.HasPrefix(line, "tool 1.2.3 ") || strings.TrimPrefix(line, "tool 1.2.3 ") != Revision() {
		t.Fatalf("%q", line)
	}
	if s := Short(7); len(strings.TrimSuffix(s, "+dirty")) > 7 {
		t.Fatalf("%q", s)
	}
}

func TestStampOverridesToolchainRevision(t *testing.T) {
	seat := []debug.BuildSetting{
		{Key: "vcs.revision", Value: "76fe8a482e8480fe5f5cfc0242d2dcaff342643c"},
		{Key: "vcs.modified", Value: "true"},
	}
	own := "bd8b75723f963f4983939aa41af188dfba68261d"
	if got := resolve(own, seat); got != "bd8b75723f96" {
		t.Errorf("stamped clean: %q", got)
	}
	if got := resolve(own+"+dirty", seat); got != "bd8b75723f96+dirty" {
		t.Errorf("stamped dirty: %q", got)
	}
	if got := resolve(own, nil); got != "bd8b75723f96" {
		t.Errorf("stamp without toolchain settings: %q", got)
	}
	if got := resolve("", seat); got != "76fe8a482e84+dirty" {
		t.Errorf("unstamped: %q", got)
	}
	if got := resolve("", nil); got != "devel" {
		t.Errorf("no revision: %q", got)
	}
}
