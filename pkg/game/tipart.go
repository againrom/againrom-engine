package game

import (
	"fmt"
	"image"
	"image/draw"

	"againrom/pkg/formats/spr256"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// The tip panel's own three shipped nodes (1018 spec behaviour 1).
//
// interface/t_back.bmp, interface/t_border.bmp and interface/radiob.256 are
// read here for the first time in this tree: no earlier story loads any of
// them. Their pairing as a panel's fill, frame and toggle gem is the
// contract's own inference from the owner's photographs, not a decoded
// fact — ui.TipPanelArt's own doc restates the caveat, this file only reads
// the three nodes and hands them across the seam already keyed and framed
// the way every other room's art is.
const (
	tipBackPath   = graphicsPrefix + "interface/t_back.bmp"
	tipBorderPath = graphicsPrefix + "interface/t_border.bmp"
	tipGemPath    = graphicsPrefix + "interface/radiob.256"
)

// tipGemOffFrame and tipGemOnFrame select two of radiob.256's six shipped
// frames (24x24 round off/on, 24x24 square off/on, 16x16 small-square
// off/on — the contract's own "The shipped texts" table). AUTHORED: no
// claim or photograph says which pair a checkbox this size uses; the
// small-square pair is taken as the closest fit to a line of text (file doc,
// ui.TipPanelArt).
const (
	tipGemOffFrame = 4
	tipGemOnFrame  = 5
)

// LoadTipPanelArt resolves the tip panel's install-backed art. Cosmetic on
// LoadTownSquareArt's own rule: a missing or mis-sized node carries its own
// address in the returned error, and every room this story adds falls back
// to drawing no panel at all rather than making the game unusable.
func LoadTipPanelArt(src terrain.EntrySource) (*ui.TipPanelArt, error) {
	a := &ui.TipPanelArt{}
	var err error
	if a.Fill, err = readChargenBMP(src, tipBackPath); err != nil {
		return nil, err
	}
	if err = chargenSize(a.Fill, 160, 240, tipBackPath); err != nil {
		return nil, err
	}
	// t_border.bmp SHIPS 8-BIT PALETTED, NOT 24-BIT LIKE t_back.bmp: measured
	// directly against both preserved installs (bpp=8, palette at byte 54,
	// pixel data at 1078 = 54 + 256*4), where readChargenBMP's own decoder
	// (pkg/formats/bmp.Decode) accepts only 24-bit BI_RGB and refuses this
	// file outright. terrain.DecodeBMP8 is the reader this tree already uses
	// for every other 8-bit BMP (chargenassets.go's mask.bmp,
	// worldmap.go's PathMap.bmp); this is the first 8-bit BMP in this tree
	// read for its COLOUR rather than its index, so its palette is resolved
	// to RGBA here rather than carried as indices.
	borderMask, err := readChargenMask(src, tipBorderPath)
	if err != nil {
		return nil, err
	}
	if err = chargenSize(borderMask, 88, 108, tipBorderPath); err != nil {
		return nil, err
	}
	border := paletteRGBA(borderMask)
	// KEYED, NOT OPAQUE: the contract's own reading of the shipped picture is
	// "an ornate gold frame ... on a keyed black field" (behaviour 1's own
	// art table), the same shape townsquareart.go's own overlay is keyed
	// for. An opaque nine-patch here would paint the frame's own black field
	// as a dark rectangle over the panel's fill instead of showing it
	// through.
	a.Border = keyBlack(border)
	if a.GemOff, err = loadTipGemFrame(src, tipGemPath, tipGemOffFrame); err != nil {
		return nil, err
	}
	if a.GemOn, err = loadTipGemFrame(src, tipGemPath, tipGemOnFrame); err != nil {
		return nil, err
	}
	return a, nil
}

// paletteRGBA resolves an 8-bit paletted image's own stored indices to RGBA,
// opaque throughout — DecodeBMP8's own doc explains why the terrain and mask
// readers keep the indices instead: those two decoded behaviours key off the
// index itself. This story's border is a picture, not an index source, so
// nothing here needs to survive past the colour.
func paletteRGBA(pic *image.Paletted) *image.RGBA {
	if pic == nil {
		return nil
	}
	b := pic.Bounds()
	out := image.NewRGBA(b)
	draw.Draw(out, b, pic, b.Min, draw.Src)
	return out
}

// loadTipGemFrame reads one frame of a .256 sheet as an RGBA, carrying the
// sheet's own structural transparency into the alpha channel — loadShopSprite
// (shopart.go)'s own reader, generalized to a frame index instead of always
// frame 0, since radiob.256 packs six gem pictures in one stream and this
// story needs two of them.
func loadTipGemFrame(src entrySource, addr string, frame int) (*image.RGBA, error) {
	if src == nil {
		return nil, fmt.Errorf("%s: no archive", addr)
	}
	raw, err := src.ReadFile(addr)
	if err != nil {
		return nil, err
	}
	sheet, err := spr256.Decode(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", addr, err)
	}
	if !sheet.HasPalette {
		return nil, fmt.Errorf("%s: no palette", addr)
	}
	if frame < 0 || frame >= len(sheet.Frames) {
		return nil, fmt.Errorf("%s: frame %d, sheet carries %d", addr, frame, len(sheet.Frames))
	}
	f := sheet.Frames[frame]
	if f.Width <= 0 || f.Height <= 0 || len(f.Pixels) < f.Width*f.Height {
		return nil, fmt.Errorf("%s: frame %d is empty", addr, frame)
	}
	pic := image.NewRGBA(image.Rect(0, 0, f.Width, f.Height))
	for i := 0; i < f.Width*f.Height; i++ {
		p := f.Pixels[i]
		if !p.Opaque || int(p.Index) >= len(sheet.Palette) {
			continue
		}
		e := sheet.Palette[p.Index]
		o := i * 4
		pic.Pix[o], pic.Pix[o+1], pic.Pix[o+2], pic.Pix[o+3] = e.R, e.G, e.B, 0xff
	}
	return pic, nil
}

// tipArt is the front end's one copy, loaded on first use and cached the way
// shopArt already is.
func (f *FrontEnd) tipArt() *ui.TipPanelArt {
	if f == nil {
		return nil
	}
	return f.Presentation.tipArt(&f.InstallResources)
}

func (p *Presentation) tipArt(in *InstallResources) *ui.TipPanelArt {
	if p.tipArtCache.Tried() {
		return p.tipArtCache.Value()
	}
	p.tipArtCache.begin()
	if in.Archives == nil {
		return nil
	}
	a, err := LoadTipPanelArt(in.Archives.Containers)
	if err != nil {
		return nil
	}
	return p.tipArtCache.store(a)
}
