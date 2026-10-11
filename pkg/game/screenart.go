package game

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	"strings"

	"againrom/pkg/base"
	"againrom/pkg/formats/bmp"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/render/terrain"
	"againrom/pkg/screenart"
)

// The install side of the one screen art loader: the source that decodes the
// install's picture formats for pkg/screenart, the library every screen of
// one install reads its art through, and the screen art descriptions of the
// screens no composer or generator description names. No other production
// file of this package opens an install picture for a screen; the archtest
// screen-art ratchet holds that.

//go:embed screens/rom1.json
var rom1ScreensJSON []byte

//go:embed screens/rom2.json
var rom2ScreensJSON []byte

// Screens is one edition's screen art description: one manifest per screen
// whose art no composer or generator description names. Every entry carries
// the claim, divergence row or owner ruling behind it.
type Screens struct {
	Name      string           `json:"screens"`
	Manifests []ScreenManifest `json:"manifests"`
	Cite      []string         `json:"cite"`
}

// ScreenManifest is one screen's art manifest.
type ScreenManifest struct {
	Screen string            `json:"screen"`
	Art    []screenart.Entry `json:"art"`
	Cite   []string          `json:"cite"`
}

// Art answers screen's manifest, nil when the description names none.
func (s *Screens) Art(screen string) []screenart.Entry {
	if s == nil {
		return nil
	}
	for _, m := range s.Manifests {
		if m.Screen == screen {
			return m.Art
		}
	}
	return nil
}

// screensDescriptions are the decoded descriptions by the name an edition's
// Screens gives. One that does not decode is a build defect.
var screensDescriptions = map[string]*Screens{
	"rom1": mustDecodeScreens(rom1ScreensJSON),
	"rom2": mustDecodeScreens(rom2ScreensJSON),
}

func mustDecodeScreens(data []byte) *Screens {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var s Screens
	if err := dec.Decode(&s); err != nil {
		panic(fmt.Errorf("screen art description: %w", err))
	}
	seen := map[string]bool{}
	for _, m := range s.Manifests {
		if m.Screen == "" || seen[m.Screen] {
			panic(fmt.Errorf("screen art description %s: screen %q is unnamed or named twice", s.Name, m.Screen))
		}
		seen[m.Screen] = true
		if err := screenart.Check(m.Art); err != nil {
			panic(fmt.Errorf("screen art description %s: screen %s: %w", s.Name, m.Screen, err))
		}
	}
	return &s
}

// ScreensDescription is the screen art description the profile's edition
// names. Callers must not change it.
func ScreensDescription(p base.Profile) *Screens { return screensDescriptions[p.Edition().Screens] }

// loadScreen reads a manifest through the install's library. Names selects
// entries in manifest order; no names reads the whole manifest.
func loadScreen(src terrain.EntrySource, d *Screens, screen string, names ...string) (*screenart.Set, error) {
	art := d.Art(screen)
	if art == nil {
		return nil, fmt.Errorf("screen %s: no art manifest", screen)
	}
	if len(names) > 0 {
		wanted := map[string]bool{}
		for _, name := range names {
			wanted[name] = true
		}
		var selected []screenart.Entry
		for _, e := range art {
			if wanted[e.Name] {
				selected = append(selected, e)
				delete(wanted, e.Name)
			}
		}
		for _, name := range names {
			if wanted[name] {
				return nil, fmt.Errorf("screen %s: no art entry %q", screen, name)
			}
		}
		art = selected
	}
	return screenArtOf(src).Load(screen, art)
}

// screenArtSource decodes the install's picture formats for the loader: a
// 24-bit bitmap as RGBA, an 8-bit bitmap as its indices, and a sprite sheet
// as every frame through its own palette.
type screenArtSource struct{ src terrain.EntrySource }

func (s screenArtSource) Picture(key string) (image.Image, error) {
	pic, err := readInstallBMP(s.src, key)
	if err != nil {
		return nil, err
	}
	return pic, nil
}

func (s screenArtSource) Mask(key string) (*image.Paletted, error) {
	return readInstallMask(s.src, key)
}

// Sprites decodes a sheet; the library validates the frames each entry uses.
func (s screenArtSource) Sprites(key string) []image.Image {
	if s.src == nil {
		return nil
	}
	if strings.HasSuffix(strings.ToLower(key), ".256") {
		raw, err := s.src.ReadFile(key)
		if err != nil {
			return nil
		}
		sheet, err := spr256.Decode(raw)
		if err != nil {
			return nil
		}
		table, ok := sheet.Table()
		if !ok {
			return nil
		}
		out := make([]image.Image, len(sheet.Frames))
		for i, f := range sheet.Frames {
			out[i] = f.RGBA(table)
		}
		return out
	}
	frames := effectFrames(s.src, key)
	out := make([]image.Image, 0, len(frames))
	for _, f := range frames {
		if f.Width <= 0 || f.Height <= 0 {
			return nil
		}
		out = append(out, f.RGBA())
	}
	return out
}

// installArt is an install's containers carrying the library every screen
// of that install reads its art through, so a picture two screens share is
// read and validated once.
type installArt struct {
	terrain.EntrySource
	lib *screenart.Library
}

// screenArtOf answers the library src carries, or a library of its own over
// src.
func screenArtOf(src terrain.EntrySource) *screenart.Library {
	if a, ok := src.(installArt); ok {
		return a.lib
	}
	return screenart.NewLibrary(screenArtSource{src})
}

// artSource answers the install's containers with its art library, nil when
// the install has none. Archives built without OpenArchives carry no library
// and answer the bare containers.
func (a *Archives) artSource() terrain.EntrySource {
	if a == nil || a.Containers == nil {
		return nil
	}
	if a.art == nil {
		return a.Containers
	}
	return installArt{EntrySource: a.Containers, lib: a.art}
}

// decodeInstallBMP decodes a 24-bit bitmap read from addr.
func decodeInstallBMP(data []byte, addr string) (*image.RGBA, error) {
	pic, err := bmp.DecodeRGBA(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", addr, err)
	}
	return pic, nil
}

// readInstallBMP reads and decodes the 24-bit bitmap at addr.
func readInstallBMP(src terrain.EntrySource, addr string) (*image.RGBA, error) {
	if src == nil {
		return nil, fmt.Errorf("%s: no archive", addr)
	}
	b, err := src.ReadFile(addr)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", addr, err)
	}
	return decodeInstallBMP(b, addr)
}

// readInstallMask reads the 8-bit bitmap at addr as its palette indices.
func readInstallMask(src terrain.EntrySource, addr string) (*image.Paletted, error) {
	if src == nil {
		return nil, fmt.Errorf("%s: no archive", addr)
	}
	b, err := src.ReadFile(addr)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", addr, err)
	}
	mask, err := bmp.DecodePaletted(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", addr, err)
	}
	return mask, nil
}

// keyBlack clears the alpha of every pure-black pixel, in place, and answers
// the picture it was given.
func keyBlack(pic *image.RGBA) *image.RGBA {
	if pic == nil {
		return nil
	}
	for i := 0; i+3 < len(pic.Pix); i += 4 {
		if pic.Pix[i] == 0 && pic.Pix[i+1] == 0 && pic.Pix[i+2] == 0 {
			pic.Pix[i+3] = 0
		}
	}
	return pic
}

// rgbaOf is a loaded picture as the RGBA the screens hold, nil for none.
func rgbaOf(pic image.Image) *image.RGBA {
	if rgba, ok := pic.(*image.RGBA); ok {
		return rgba
	}
	return nil
}
