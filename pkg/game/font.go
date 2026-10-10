package game

import (
	"fmt"

	"againrom/pkg/formats/spr16"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
)

// A font is two nodes, and this is where they are joined: the glyph atlas
// and its advance sidecar are separate archive entries with no reference to
// each other, so the only thing that can say they belong together is
// whatever holds both.

const (
	// DefaultFont is the base name of the font a caller takes when it has no
	// reason to want another: 224 records at a 16x15 cell, the only one of the
	// byte-control atlases that is byte-identical between the two shipped
	// releases. It is a DEFAULT and not a lock — LoadFont takes the base name,
	// and font2 (224 records, 8x10) and font3 (64 records, 8x6) load through the
	// same call.
	DefaultFont = "font1"

	// FontSpacing is the letter spacing the engine's own font construction
	// passes, added after every glyph. It lives here beside the addresses
	// because it comes from the same place they do: how a font object is built.
	FontSpacing = 2
)

// FontAtlasPath and FontAdvancePath are the two nodes of font base, addressed
// through the container filesystem — each carries graphics.res's identity
// segment, so the caller need not know which archive a font comes from.
func FontAtlasPath(base string) string   { return graphicsPrefix + base + "/" + base + ".16" }
func FontAdvancePath(base string) string { return graphicsPrefix + base + "/" + base + ".dat" }

// LanguagePath is the entry an install names its own language in.
var LanguagePath = mainPrefix + "id"

// LanguageSelector is the install's own language selector: the TRAILING ASCII
// DIGIT of LanguagePath's payload, taken as a number. The shipped roots hold
// "english 0" and "russian 1", so the two releases select 0 and 1 and the
// executable behind them is the same bytes — which is the whole reason a
// consumer that ships one program and two data sets is doing what the engine
// does.
//
// EVERY FAILURE IS SELECTOR 0, and none of them is an error. No source, no
// entry, an empty payload and a payload ending in something that is not a digit
// all answer 0, which is the identity conversion. That is deliberate and it is
// the only safe direction: a selector this function got wrong by guessing would
// silently redraw every string in the game, whereas 0 can only ever leave
// drawing exactly as it was. An install that fails to say which language it is
// is therefore treated as the one whose rule changes nothing.
//
// It reads the LAST byte and not the last RUNE. The payload is the game's own
// byte string and the value it carries is a single decimal digit; a multi-digit
// selector is not something either shipped root has, and inventing a parse for
// one would be inventing a fact.
func LanguageSelector(src terrain.EntrySource) int {
	if src == nil {
		return 0
	}
	b, err := src.ReadFile(LanguagePath)
	if err != nil || len(b) == 0 {
		return 0
	}
	c := b[len(b)-1]
	if c < '0' || c > '9' {
		return 0
	}
	return int(c - '0')
}

// textSelector is the running FrontEnd's own install selector, the same value
// frontend.go's SAVE/LOAD wiring already reads off f.Font.Selector — pulled
// out once here so a SAV label writer and reader do not each re-derive it.
// A FrontEnd with no readable font (FontErr, or one hand-built by a test)
// answers 0, LanguageSelector's own "no language named" identity value.
func (f *FrontEnd) textSelector() int {
	if f != nil && f.Font.Value() != nil {
		return f.Font.Value().Selector
	}
	return 0
}

// FontAtlasClass is how a font's atlas ships: its extension and its decoder.
type FontAtlasClass int

const (
	FontShades   FontAtlasClass = iota // `.16` opaque shades
	FontCoverage                       // `.16a` alpha levels as coverage (DIV-303)
)

func (c FontAtlasClass) AtlasPath(base string) string {
	if c == FontCoverage {
		return FontAtlasPathA(base)
	}
	return FontAtlasPath(base)
}

// glyphs decodes an atlas without advances. A `.16a` stream declares its
// 1024-byte palette (SPR16A-FONT-018); its indices are dropped.
func (c FontAtlasClass) glyphs(atlas []byte) ([]text.Glyph, error) {
	var out []text.Glyph
	if c == FontCoverage {
		sprite, err := spr16.DecodeA(atlas, true)
		if err != nil {
			return nil, err
		}
		for _, f := range sprite.Frames {
			pixels := make([]text.Pixel, len(f.Pixels))
			for j, p := range f.Pixels {
				pixels[j] = text.Pixel{Level: p.Level, Painted: p.Painted}
			}
			out = append(out, text.Glyph{Width: f.Width, Height: f.Height, Pixels: pixels, CoverageLevels: true})
		}
		return out, nil
	}
	frames, err := spr16.DecodeG(atlas)
	if err != nil {
		return nil, err
	}
	for _, f := range frames {
		pixels := make([]text.Pixel, len(f.Pixels))
		for j, p := range f.Pixels {
			pixels[j] = text.Pixel{Level: p.Value, Painted: p.Painted}
		}
		out = append(out, text.Glyph{Width: f.Width, Height: f.Height, Pixels: pixels})
	}
	return out, nil
}

// LoadFont is the one font loader: it joins font base's atlas, decoded as
// class, with its advance sidecar. It does not set the selector, which lives
// in another archive; the caller sets it. Every failure is an error, never a
// partial font: a read error comes back unwrapped, a decode error names its
// node, a count mismatch names both, and an atlas with no record is refused.
func LoadFont(src terrain.EntrySource, base string, class FontAtlasClass) (*text.Font, error) {
	atlasPath := class.AtlasPath(base)
	if src == nil {
		return nil, fmt.Errorf("%s: no graphics archive", atlasPath)
	}

	atlasBytes, err := src.ReadFile(atlasPath)
	if err != nil {
		return nil, err
	}
	glyphs, err := class.glyphs(atlasBytes)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", atlasPath, err)
	}
	if len(glyphs) == 0 {
		return nil, fmt.Errorf("%s: font atlas holds no glyph record", atlasPath)
	}

	advPath := FontAdvancePath(base)
	advBytes, err := src.ReadFile(advPath)
	if err != nil {
		return nil, err
	}
	advances, err := spr16.Advances(advBytes)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", advPath, err)
	}
	if len(advances) != len(glyphs) {
		return nil, fmt.Errorf("%s has %d advances for %s's %d glyph records",
			advPath, len(advances), atlasPath, len(glyphs))
	}
	for i := range glyphs {
		glyphs[i].Advance = advances[i]
	}
	return &text.Font{Spacing: FontSpacing, Glyphs: glyphs}, nil
}
