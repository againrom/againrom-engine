// Package namefold holds the three case-folding policies for resource names.
// Each caller names the policy whose rule it keeps: the archives compare
// names under the archive policy, a packed mod under the mod tree policy, and
// the host's files under the host policy. The package imports only the
// standard library.
package namefold

import "strings"

// ArchiveByte folds ASCII 'A'-'Z' to 'a'-'z' and leaves every other byte,
// every byte of 0x80 and above included, unchanged. An archive entry name
// decodes as CP866, so a Unicode fold would join names the original keeps
// apart.
func ArchiveByte(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

// Archive is s under ArchiveByte.
func Archive(s string) string {
	i := 0
	for i < len(s) && ArchiveByte(s[i]) == s[i] {
		i++
	}
	if i == len(s) {
		return s
	}
	b := []byte(s)
	for ; i < len(b); i++ {
		b[i] = ArchiveByte(b[i])
	}
	return string(b)
}

// ArchiveBytes is a copy of b under ArchiveByte.
func ArchiveBytes(b []byte) []byte {
	out := make([]byte, len(b))
	for i, c := range b {
		out[i] = ArchiveByte(c)
	}
	return out
}

// ArchiveEqual reports whether a and b are equal under ArchiveByte: the same
// length, and the same byte at every position after folding.
func ArchiveEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if ArchiveByte(a[i]) != ArchiveByte(b[i]) {
			return false
		}
	}
	return true
}

// ModTreeRune folds r as a Windows folder does for every letter a RES name can
// hold: ASCII A-Z and the Cyrillic capitals of CP866 (А-Я, Ё, Є, Ї, Ў).
func ModTreeRune(r rune) rune {
	switch {
	case r >= 'A' && r <= 'Z':
		return r - 'A' + 'a'
	case r >= 0x410 && r <= 0x42F:
		return r + 0x20
	case r == 0x401 || r == 0x404 || r == 0x407:
		return r + 0x50
	case r == 0x40E:
		return 0x45E
	}
	return r
}

// ModTree is s under ModTreeRune, the way a packed mod matches names.
func ModTree(s string) string { return strings.Map(ModTreeRune, s) }

// Host is the comparison form of a name on the host file system: Unicode
// lower case.
func Host(s string) string { return strings.ToLower(s) }
