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

// The text and host policies answer exactly what the standard function they
// replaced answered, on ASCII, mixed case, digits, separators, Cyrillic as
// UTF-8 and as CP1251 bytes, invalid UTF-8 and the folds only Unicode knows.
func TestTextAndHostPoliciesEqualTheirStandardFunction(t *testing.T) {
	winCyrillic := string([]byte{0xcf, 0xe0, 0xef, 0xea, 0xe0, 0x2f, 0xd4, 0xe0, 0xe9, 0xeb, 0x2e, 0x42, 0x4d, 0x50}) // Папка/Файл.BMP
	inputs := []string{
		"", "a", "A", "Main/Text/INN.TXT", `Scenario\M12.ALM`, "file.Sav", ".SAV", "x1_2-3.4",
		"Mission12", "a b\tc", "КАРТА", "карта", "Карта/Файл.BMP", "ЁЄЇЎ ёєїў",
		winCyrillic, "high\x80\xc0\xffA", "Ωmega ωmega", "İstanbul", "ǅ", "Kelvin ſs", "ß ẞ",
	}
	for _, in := range inputs {
		if got, want := ArchiveText(in), strings.ToLower(in); got != want {
			t.Errorf("ArchiveText(%q) = %q, want %q", in, got, want)
		}
		if got, want := Host(in), strings.ToLower(in); got != want {
			t.Errorf("Host(%q) = %q, want %q", in, got, want)
		}
		if got, want := HostUpper(in), strings.ToUpper(in); got != want {
			t.Errorf("HostUpper(%q) = %q, want %q", in, got, want)
		}
		for _, other := range inputs {
			if got, want := ArchiveTextEqual(in, other), strings.EqualFold(in, other); got != want {
				t.Errorf("ArchiveTextEqual(%q, %q) = %t, want %t", in, other, got, want)
			}
			if got, want := HostEqual(in, other), strings.EqualFold(in, other); got != want {
				t.Errorf("HostEqual(%q, %q) = %t, want %t", in, other, got, want)
			}
		}
	}
}

// The text policies differ from the ASCII archive policy on the names where
// the choice of a site matters, which is why a site keeps the one it had.
func TestArchiveTextDiffersFromArchiveOnNonASCII(t *testing.T) {
	for _, s := range []string{"КАРТА", "Ωmega", "high\x80A"} {
		if ArchiveText(s) == Archive(s) {
			t.Errorf("%q folds alike under ArchiveText and Archive", s)
		}
	}
	if !ArchiveTextEqual("ſ", "s") || ArchiveEqual("ſ", "s") {
		t.Error("U+017F is equal to s under the text policy only")
	}
	for _, s := range []string{"Main/Text", "a1_.BMP", ""} {
		if ArchiveText(s) != Archive(s) {
			t.Errorf("%q folds apart on ASCII", s)
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
