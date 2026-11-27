package namefold

import (
	"strings"
	"unicode/utf8"
)

// WindowsReserved reports a Windows device name, whatever its case or extension.
func WindowsReserved(name string) bool {
	stem, _, _ := strings.Cut(name, ".")
	stem = HostUpper(strings.TrimRight(stem, " "))
	switch stem {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return true
	}
	if len(stem) >= 4 && (strings.HasPrefix(stem, "COM") || strings.HasPrefix(stem, "LPT")) {
		r, size := utf8.DecodeRuneInString(stem[3:])
		if 3+size == len(stem) {
			return r >= '0' && r <= '9' || r == '\u00b9' || r == '\u00b2' || r == '\u00b3'
		}
	}
	return false
}
