package game

import (
	"fmt"

	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// The campaign documents panel's install-backed inputs (B3, MENU-DOC-009,
// MISSION-DOC-021, TEXT-API-007, SPR16A-FONT-018): the panel's eleven
// bitmaps and the per-element picture, named by the documents manifest of
// the screen art description; the per-element text; and font4.

// docTextPrefix is MISSION-DOC-021's text format string with the value
// removed: kind 1 formats `main\text\Docs\%d.txt`.
const docTextPrefix = mainPrefix + "text/docs/"

// LoadDocumentPanelArt resolves the documents panel's eleven bitmaps from
// the documents manifest of description d. MENU-DOC-009 gives each control's
// hit rectangle as equal to its bitmap, so each is held to its size.
//
// Cosmetic on LoadTipPanelArt's own rule: a missing or mis-sized node carries
// its own address in the returned error, and the front end then installs no
// panel art at all rather than making the game unusable.
func LoadDocumentPanelArt(src terrain.EntrySource, d *Screens) (*ui.DocumentPanelArt, error) {
	set, err := loadScreen(src, d, "documents")
	if err != nil {
		return nil, err
	}
	a := &ui.DocumentPanelArt{Sheet: rgbaOf(set.Picture("sheet"))}
	for i := range a.Left {
		a.Left[i] = rgbaOf(set.Picture(fmt.Sprintf("left/%d", i)))
		a.Right[i] = rgbaOf(set.Picture(fmt.Sprintf("right/%d", i)))
	}
	for i := range a.OK {
		a.OK[i] = rgbaOf(set.Picture(fmt.Sprintf("ok/%d", i)))
	}
	return a, nil
}

// TEXT-API-007
const DocumentFont = "font4"

// DocumentPicturePath and DocumentTextPath are MISSION-DOC-021's two
// per-element addresses, the element's own value formatted into the path;
// the picture's is the bound page entry of description d's documents
// manifest.
func DocumentPicturePath(d *Screens, v int) string {
	for _, e := range d.Art("documents") {
		if e.Name == "page" {
			return fmt.Sprintf(e.Key, v)
		}
	}
	return ""
}
func DocumentTextPath(v int) string { return fmt.Sprintf("%s%d.txt", docTextPrefix, v) }

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
func LoadDocumentPage(src terrain.EntrySource, screens *Screens, d Document, code TextCode) (ui.DocumentPage, bool) {
	if src == nil {
		return ui.DocumentPage{}, false
	}
	switch d.Kind {
	case DocumentPicture:
		frames, err := screenArtOf(src).Bind("documents", screens.Art("documents"), "page", d.Value)
		if err != nil || len(frames) == 0 {
			return ui.DocumentPage{}, false
		}
		return ui.DocumentPage{Picture: rgbaOf(frames[0])}, true
	case DocumentText:
		s, ok := ReadShopTip(src, DocumentTextPath(d.Value), code)
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
		if page, ok := LoadDocumentPage(s.f.Archives.artSource(), ScreensDescription(s.f.profile()), d, s.f.textCode()); ok {
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
	a, err := LoadDocumentPanelArt(f.Archives.artSource(), ScreensDescription(f.profile()))
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
	// Share the generator's name font: same font, same selector.
	if in.ChargenAssets != nil && in.ChargenAssets.Presentation != nil && in.ChargenAssets.Presentation.NameFont != nil {
		return p.docFontCache.store(in.ChargenAssets.Presentation.NameFont)
	}
	font, err := LoadFont(in.Archives.Containers, DocumentFont, FontCoverage)
	if err != nil {
		return nil
	}
	font.Selector = LanguageSelector(in.Archives.Containers)
	return p.docFontCache.store(font)
}
