package namefold

import (
	"strings"
	"testing"
)

// Each policy answers what the fold it replaced answered, on ASCII, CP866
// capitals and small letters, other Unicode and bytes above 0x7f.
func TestPolicies(t *testing.T) {
	for _, tc := range []struct{ in, archive, modTree, host string }{
		{"", "", "", ""},
		{"World/Data/MAP.REG", "world/data/map.reg", "world/data/map.reg", "world/data/map.reg"},
		{`A\B`, `a\b`, `a\b`, `a\b`},
		{"already lower 09_-.", "already lower 09_-.", "already lower 09_-.", "already lower 09_-."},
		{"КАРТА", "КАРТА", "карта", "карта"},
		{"карта", "карта", "карта", "карта"},
		{"ЁЄЇЎ", "ЁЄЇЎ", "ёєїў", "ёєїў"},
		{"Ωmega", "Ωmega", "Ωmega", "ωmega"},
		{"Kelvin ſs", "Kelvin ſs", "Kelvin ſs", "kelvin ſs"},
		{"high\x80\xc0\xffA", "high\x80\xc0\xffa", "high���a", "high���a"},
	} {
		if got := Archive(tc.in); got != tc.archive {
			t.Errorf("Archive(%q) = %q, want %q", tc.in, got, tc.archive)
		}
		if got := string(ArchiveBytes([]byte(tc.in))); got != tc.archive {
			t.Errorf("ArchiveBytes(%q) = %q, want %q", tc.in, got, tc.archive)
		}
		if got := ModTree(tc.in); got != tc.modTree {
			t.Errorf("ModTree(%q) = %q, want %q", tc.in, got, tc.modTree)
		}
		if got := Host(tc.in); got != tc.host {
			t.Errorf("Host(%q) = %q, want %q", tc.in, got, tc.host)
		}
	}
}

// ArchiveByte folds exactly A-Z over all 256 byte values, and ArchiveEqual
// compares by it.
func TestArchiveByteAndEqual(t *testing.T) {
	for b := 0; b < 256; b++ {
		want := byte(b)
		if b >= 'A' && b <= 'Z' {
			want = byte(b) + 32
		}
		if got := ArchiveByte(byte(b)); got != want {
			t.Errorf("ArchiveByte(%#x) = %#x, want %#x", b, got, want)
		}
	}
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{"Terrain", "TERRAIN", true},
		{"Terrain", "Terrai", false},
		{"", "", true},
		{"\x80", "\x80", true},
		{"\xC0", "\xE0", false},
		{"k", "K", false},
	} {
		if got := ArchiveEqual(tc.a, tc.b); got != tc.want {
			t.Errorf("ArchiveEqual(%q, %q) = %t", tc.a, tc.b, got)
		}
	}
}

// Archive returns its argument itself when nothing folds.
func TestArchiveKeepsAnUnfoldedString(t *testing.T) {
	s := strings.Repeat("a1_", 8)
	if Archive(s) != s {
		t.Fatal("changed a lower case string")
	}
}
