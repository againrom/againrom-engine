package game

import (
	"fmt"

	"againrom/pkg/formats/spr16"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// The campaign documents panel's install-backed inputs (B3, MENU-DOC-009,
// MISSION-DOC-021, TEXT-API-007, SPR16A-FONT-018).
//
// Three things are read here and nowhere else in this tree: the panel's eleven
// bitmaps, the two per-element resource paths, and font4.

const (
	docSheetPath = graphicsPrefix + "interface/docs/sheet.bmp"

	// docArrowDir and docOKDir are the two directories the arrays at `+0x74`,
	// `+0x88` and `+0x9c` name.
	docArrowDir = graphicsPrefix + "interface/docs/arrows/"
	docOKDir    = graphicsPrefix + "interface/docs/ok/"

	// docPicturePrefix and docTextPrefix are MISSION-DOC-021's two format
	// strings with the value removed: kind 0 formats
	// `graphics\interface\Docs\%d.bmp` and kind 1 `main\text\Docs\%d.txt`,
	// the first component naming the archive. Address separators are `/` and
	// lower case in this tree, which is the container filesystem's own
	// normalisation and not a change of address.
	docPicturePrefix = graphicsPrefix + "interface/docs/"
	docTextPrefix    = mainPrefix + "text/docs/"
)

// docArrowFiles and docOKFiles are the two arrays in the original's own order.
//
// THE ORDER IS LOAD-BEARING and it is the selection index (MENU-DOC-009:
// element 1 on hover, 2 on press, 3 for OK, 0 on leave). Sorting these by name
// would swap `11` and `01` on the arrows.
var (
	docArrowFiles = [3]string{"00_", "01_", "11_"}
	docOKFiles    = [4]string{"ok_off.bmp", "ok_on.bmp", "ok_l_off.bmp", "ok_l_on.bmp"}
)

// The eleven bitmaps' own pixel sizes, checked at load. MENU-DOC-009 gives
// each control's hit rectangle as EQUAL to its bitmap, so a bitmap that is not
// this size would move a control's picture off its hit test without anything
// else failing.
const (
	docSheetW, docSheetH = 640, 480
	docLeftW, docLeftH   = 56, 40
	docRightW, docRightH = 60, 40
	docOKW, docOKH       = 44, 32
)

// LoadDocumentPanelArt resolves the documents panel's eleven bitmaps.
//
// Cosmetic on LoadTipPanelArt's own rule: a missing or mis-sized node carries
// its own address in the returned error, and the front end then installs no
// panel art at all rather than making the game unusable.
func LoadDocumentPanelArt(src terrain.EntrySource) (*ui.DocumentPanelArt, error) {
	a := &ui.DocumentPanelArt{}
	var err error
	if a.Sheet, err = readChargenBMP(src, docSheetPath); err != nil {
		return nil, err
	}
	if err = chargenSize(a.Sheet, docSheetW, docSheetH, docSheetPath); err != nil {
		return nil, err
	}
	for i, stem := range docArrowFiles {
		lp := docArrowDir + stem + "l.bmp"
		if a.Left[i], err = readChargenBMP(src, lp); err != nil {
			return nil, err
		}
		if err = chargenSize(a.Left[i], docLeftW, docLeftH, lp); err != nil {
			return nil, err
		}
		rp := docArrowDir + stem + "r.bmp"
		if a.Right[i], err = readChargenBMP(src, rp); err != nil {
			return nil, err
		}
		if err = chargenSize(a.Right[i], docRightW, docRightH, rp); err != nil {
			return nil, err
		}
	}
	for i, name := range docOKFiles {
		p := docOKDir + name
		if a.OK[i], err = readChargenBMP(src, p); err != nil {
			return nil, err
		}
		if err = chargenSize(a.OK[i], docOKW, docOKH, p); err != nil {
			return nil, err
		}
	}
	return a, nil
}

// TEXT-API-007
const DocumentFont = "font4"

// FontAtlasPathA is FontAtlasPath for a font whose atlas ships as `.16a`.
//
// FONT4 IS THE ONLY ONE (SPR16A-FONT-018: the construction site names
// font1/2/3 through the `.16` class and font4 through the `.16a` class), and
// both shipped roots carry `font4/font4.16a` and `font4/font4.dat` and no
// `font4/font4.16`. The sidecar's address is unchanged, which is why
// FontAdvancePath is reused rather than restated.
func FontAtlasPathA(base string) string { return graphicsPrefix + base + "/" + base + ".16a" }

// LoadFontA is LoadFont for a `.16a` atlas. Every rule in LoadFont's own
// header holds here unchanged: it does not set the selector, every failure is
// an error and none is a partial font, and an atlas with no records is
// refused.
//
// The output overlay uses .16a levels as alpha, unlike .16 opaque shades.
// Palette indices are still dropped: the installed font uses near-white
// entries 254 and 255. Multicolour atlases remain outside this loader's seam.
// Native Draw retains its raster contract. DIV-303.
func LoadFontA(src terrain.EntrySource, base string) (*text.Font, error) {
	if src == nil {
		return nil, fmt.Errorf("%s: no graphics archive", FontAtlasPathA(base))
	}

	atlasPath := FontAtlasPathA(base)
	atlasBytes, err := src.ReadFile(atlasPath)
	if err != nil {
		return nil, err
	}
	// The stream leads with a 1024-byte palette, declared and never inferred
	// (spr16.DecodeA's own contract). Both shipped roots decode to 224
	// records under this declaration, which is the record count
	// SPR16A-FONT-018 gives for font4 and the count its 896-byte sidecar
	// carries; a wrong declaration does not decode at all.
	sprite, err := spr16.DecodeA(atlasBytes, true)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", atlasPath, err)
	}
	if len(sprite.Frames) == 0 {
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
	if len(advances) != len(sprite.Frames) {
		return nil, fmt.Errorf("%s has %d advances for %s's %d glyph records",
			advPath, len(advances), atlasPath, len(sprite.Frames))
	}

	font := &text.Font{Spacing: FontSpacing, Glyphs: make([]text.Glyph, len(sprite.Frames))}
	for i, f := range sprite.Frames {
		pixels := make([]text.Pixel, len(f.Pixels))
		for j, p := range f.Pixels {
			pixels[j] = text.Pixel{Level: p.Level, Painted: p.Painted}
		}
		font.Glyphs[i] = text.Glyph{
			Width:          f.Width,
			Height:         f.Height,
			Pixels:         pixels,
			Advance:        advances[i],
			CoverageLevels: true,
		}
	}
	return font, nil
}

// DocumentPicturePath and DocumentTextPath are MISSION-DOC-021's two
// per-element addresses, the element's own value formatted into the path.
func DocumentPicturePath(v int) string { return fmt.Sprintf("%s%d.bmp", docPicturePrefix, v) }
func DocumentTextPath(v int) string    { return fmt.Sprintf("%s%d.txt", docTextPrefix, v) }

// LoadDocumentPage resolves one collection element to what the panel draws.
//
// A PICTURE IS NOT RESIZED and a text is not split. The picture is handed
// over at its natural size, which for the one shipped picture is 464x344
// against a 456-px document rectangle, and the text is read whole by the
// same reader every other install text in this tree goes through
// (ReadShopTip's own header: MISSION-DOC-021 decodes that reader). Wrapping
// happens in pkg/ui, where the font is.
//
// AN UNRESOLVABLE ELEMENT ANSWERS false AND IS DROPPED, not drawn as an empty
// page: a value naming no shipped file is not a page the player can be shown,
// and a blank sheet in the middle of the collection is worse than one fewer
// element. The five values the shipped campaign registry carries resolve 5/5
// on both roots (MISSION-DOC-021), so this arm is reached by authored content
// alone.
func LoadDocumentPage(src entrySource, d Document) (ui.DocumentPage, bool) {
	if src == nil {
		return ui.DocumentPage{}, false
	}
	switch d.Kind {
	case DocumentPicture:
		addr := DocumentPicturePath(d.Value)
		b, err := src.ReadFile(addr)
		if err != nil {
			return ui.DocumentPage{}, false
		}
		img, err := chargenRGBA(b, addr)
		if err != nil {
			return ui.DocumentPage{}, false
		}
		return ui.DocumentPage{Picture: img}, true
	case DocumentText:
		s, ok := ReadShopTip(src, DocumentTextPath(d.Value))
		if !ok || s == "" {
			return ui.DocumentPage{}, false
		}
		return ui.DocumentPage{Text: s}, true
	}
	return ui.DocumentPage{}, false
}

// documentSource is the front end's ui.DocumentSource: the collection the town
// carries, resolved to pages against the archives.
//
// IT HOLDS THE FRONT END AND NOT A SLICE, which is the whole point of the seam
// (ui.DocumentSource's own header): the collection grows as the campaign
// advances, and a value captured when the application was assembled would show
// the campaign as it stood then.
type documentSource struct{ f *FrontEnd }

// Documents resolves the town's collection, in the order it was granted, and
// drops every element that does not resolve.
func (s documentSource) Documents() []ui.DocumentPage {
	if s.f == nil || s.f.Town == nil || s.f.Archives == nil {
		return nil
	}
	list := s.f.Town.Documents()
	if len(list) == 0 {
		return nil
	}
	out := make([]ui.DocumentPage, 0, len(list))
	for _, d := range list {
		if page, ok := LoadDocumentPage(s.f.Archives.Containers, d); ok {
			out = append(out, page)
		}
	}
	return out
}

// documentArt is the front end's one copy of the panel's bitmaps, loaded on
// first use and cached the way tipArt already is.
func (f *FrontEnd) documentArt() *ui.DocumentPanelArt {
	if f == nil {
		return nil
	}
	if f.docArtCache.Tried() {
		return f.docArtCache.Value()
	}
	f.docArtCache.begin()
	if f.Archives == nil {
		return nil
	}
	a, err := LoadDocumentPanelArt(f.Archives.Containers)
	if err != nil {
		return nil
	}
	return f.docArtCache.store(a)
}

// documentFont is the front end's one copy of font4, loaded on first use.
//
// IT CARRIES THE INSTALL'S OWN LANGUAGE SELECTOR, set here for the reason
// frontend.go sets it on font1: the document text is the install's bytes, CP866
// on a Russian root, and a font without the selector draws those bytes as the
// wrong glyphs. Reading the selector cannot fail — an install that names no
// language selects 0, the identity rule.
func (f *FrontEnd) documentFont() *text.Font {
	if f == nil {
		return nil
	}
	return f.Presentation.documentFont(&f.InstallResources)
}

func (p *Presentation) documentFont(in *InstallResources) *text.Font {
	if p.docFontCache.Tried() {
		return p.docFontCache.Value()
	}
	p.docFontCache.begin()
	if in.Archives == nil {
		return nil
	}
	font, err := LoadFontA(in.Archives.Containers, DocumentFont)
	if err != nil {
		return nil
	}
	font.Selector = LanguageSelector(in.Archives.Containers)
	return p.docFontCache.store(font)
}
