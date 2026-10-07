// Package buildinfo names the program a binary is: its version, read from the
// program's embedded VERSION file, and the source revision the Go toolchain
// stamped into it.
package buildinfo

import (
	"fmt"
	"regexp"
	"runtime/debug"
	"strings"
)

var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,4})\.(0|[1-9][0-9]{0,4})\.(0|[1-9][0-9]{0,4})$`)

// ParseVersion reads the text of a VERSION file: one line, MAJOR.MINOR.PATCH,
// each a decimal number below 65536.
func ParseVersion(text string) (string, error) {
	v := strings.TrimSpace(text)
	m := versionPattern.FindStringSubmatch(v)
	if m == nil {
		return "", fmt.Errorf("version %q is not MAJOR.MINOR.PATCH", v)
	}
	for _, part := range m[1:] {
		var n int
		fmt.Sscanf(part, "%d", &n)
		if n > 65535 {
			return "", fmt.Errorf("version %q has a part above 65535", v)
		}
	}
	return v, nil
}

var stamp string

// Revision is the source revision the binary was built from: the first 12
// characters of the VCS revision, "+dirty" appended when the tree had
// uncommitted changes, or "devel" when the toolchain stamped none.
func Revision() string {
	var settings []debug.BuildSetting
	if info, ok := debug.ReadBuildInfo(); ok {
		settings = info.Settings
	}
	return resolve(stamp, settings)
}

func resolve(stamp string, settings []debug.BuildSetting) string {
	revision, modified := "devel", false
	for _, setting := range settings {
		switch setting.Key {
		case "vcs.revision":
			if setting.Value != "" {
				revision = setting.Value
			}
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if stamp != "" {
		revision, modified = strings.TrimSuffix(stamp, "+dirty"), strings.HasSuffix(stamp, "+dirty")
	}
	if len(revision) > 12 {
		revision = revision[:12]
	}
	if modified {
		revision += "+dirty"
	}
	return revision
}

// Short is Revision cut to n characters, keeping a "+dirty" mark.
func Short(n int) string {
	r := Revision()
	dirty := strings.HasSuffix(r, "+dirty")
	r = strings.TrimSuffix(r, "+dirty")
	if len(r) > n {
		r = r[:n]
	}
	if dirty {
		r += "+dirty"
	}
	return r
}

// Line is the text -version prints: "<name> <version> <revision>".
func Line(name, version string) string {
	return name + " " + version + " " + Revision()
}
