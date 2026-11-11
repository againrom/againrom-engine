package mod

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	packedNamesFile    = "mod-names.json"
	maxPackedNameBytes = 255
)

type packedNames struct {
	Version int          `json:"version"`
	Entries []packedName `json:"entries"`
}

type packedName struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

func checkPackedComponent(name string) error {
	switch {
	case !utf8.ValidString(name):
		return fmt.Errorf("the name is not UTF-8 text")
	case name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`):
		return fmt.Errorf("%q is not a path component", name)
	case len(name) > maxPackedNameBytes:
		return fmt.Errorf("the name %q is %d UTF-8 bytes; a packed mod component is at most %d", name, len(name), maxPackedNameBytes)
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("the name holds the control character %U", r)
		}
	}
	return nil
}
