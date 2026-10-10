package game

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"againrom/pkg/render/text"
)

// fontDigest hashes a font's whole Go value.
func fontDigest(font *text.Font) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%#v", *font))))
}

// oneLoaderFontDigests, per language selector, are the fonts the two former
// loaders decoded.
var oneLoaderFontDigests = map[int]map[string]string{
	0: {
		"font1": "cf0743e91b3f082a48786411de0d71e75100865942ea2c3ad6198f291ca88910",
		"font2": "1d88ac0ba0f06e05e0808628f10335293d4a9701d3085f812e70085431cb177b",
		"font3": "53cca12f5e9f278c44e4d5a384d9215a0946f1d6ddd00edc3227f4495b863d78",
		"font4": "344c36c56244ca61d0c2dbafa7df068659867698342a4f7512cd24d9d96e13e5",
	},
	1: {
		"font1": "cf0743e91b3f082a48786411de0d71e75100865942ea2c3ad6198f291ca88910",
		"font2": "d62028c711f3054db56a37503a0b720bab7f2fea9609cfa2a126ee4676e7f213",
		"font3": "53cca12f5e9f278c44e4d5a384d9215a0946f1d6ddd00edc3227f4495b863d78",
		"font4": "344c36c56244ca61d0c2dbafa7df068659867698342a4f7512cd24d9d96e13e5",
	},
}

func TestReleaseOneFontLoaderKeepsEveryGlyph(t *testing.T) {
	root := releaseRoot(t)
	archives, err := OpenArchives(root)
	if err != nil {
		t.Fatal(err)
	}
	selector := LanguageSelector(archives.Containers)
	want := oneLoaderFontDigests[selector]
	for _, base := range []string{"font1", "font2", "font3", DocumentFont} {
		font, err := loadFontForDigest(archives, base)
		if err != nil {
			t.Fatalf("%s: %v", base, err)
		}
		got := fontDigest(font)
		t.Logf("selector %d %s %d glyphs %s", selector, base, len(font.Glyphs), got)
		if want[base] != got {
			t.Errorf("selector %d %s: digest %s, want %s", selector, base, got, want[base])
		}
	}
}

func loadFontForDigest(archives *Archives, base string) (*text.Font, error) {
	class := FontShades
	if base == DocumentFont {
		class = FontCoverage
	}
	return LoadFont(archives.Containers, base, class)
}
