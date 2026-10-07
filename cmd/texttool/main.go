// Command texttool measures and renders the game's own bitmap fonts against a
// lawful install.
//
//	texttool census [-assets DIR] [-font font1]
//	texttool render [-assets DIR] [-font font1] -out FILE (-text STR | -hex HH..) [-scale N]
//
// census prints FIGURES ONLY — counts, extents, a level histogram. render is the
// one verb that produces a picture, and it writes it only to the path -out
// names: there is no default output path, so it cannot write a converted asset
// by accident. Point -out outside the repository (or at the git-ignored
// texttool-out/).
//
// The asset root comes from -assets or AGAINROM_ASSETS and is never compiled in.
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"againrom/pkg/game"
	"againrom/pkg/render/text"
)

// graphicsArchive is the archive the fonts live in, resolved under the asset
// root the caller configures.
const graphicsArchive = "graphics.res"

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "texttool:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: texttool <census|render> [flags]")
	}
	switch args[0] {
	case "census":
		return census(args[1:], out)
	case "render":
		return render(args[1:])
	default:
		return fmt.Errorf("unknown verb %q (want census or render)", args[0])
	}
}

// fontFlags adds the two flags every verb shares and returns the accessors.
func fontFlags(fs *flag.FlagSet) (assets, base *string) {
	assets = fs.String("assets", "", "game asset root (default $AGAINROM_ASSETS)")
	base = fs.String("font", game.DefaultFont, "font base name: font1, font2 or font3")
	return assets, base
}

// loadFont resolves the asset root, opens the container filesystem over the
// graphics archive under it, and loads the named font.
func loadFont(assets, base string) (*text.Font, error) {
	root := game.ResolveAssetRoot(assets, os.Getenv("AGAINROM_ASSETS"))
	if root == "" {
		return nil, fmt.Errorf("no asset root configured; pass -assets or set AGAINROM_ASSETS")
	}
	src, err := game.OpenContainers(filepath.Join(root, graphicsArchive))
	if err != nil {
		return nil, err
	}
	return game.LoadFont(src, base)
}

func census(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("census", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	assets, base := fontFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	font, err := loadFont(*assets, *base)
	if err != nil {
		return err
	}
	TakeCensus(font).Write(out, *base)
	return nil
}

func render(args []string) error {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	assets, base := fontFlags(fs)
	outPath := fs.String("out", "", "PNG to write (REQUIRED; point it outside the repository)")
	str := fs.String("text", "", "the string to draw, as text")
	hexStr := fs.String("hex", "", "the string to draw, as hex bytes — the only way to reach a byte >= 0x80")
	scale := fs.Int("scale", 1, "integer nearest-neighbour magnification")
	fg := fs.String("color", "ffffff", "text colour, RRGGBB")
	bg := fs.String("bg", "000000", "background colour, RRGGBB")
	compare := fs.Bool("compare", false, "draw a second row spaced by the CELL, for comparison")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *outPath == "" {
		return fmt.Errorf("render needs -out: there is no default output path")
	}
	if (*str == "") == (*hexStr == "") {
		return fmt.Errorf("render needs exactly one of -text or -hex")
	}
	if *scale < 1 {
		return fmt.Errorf("-scale must be at least 1")
	}
	s := *str
	if *hexStr != "" {
		b, err := hex.DecodeString(strings.NewReplacer(" ", "", ":", "", ",", "").Replace(*hexStr))
		if err != nil {
			return fmt.Errorf("-hex: %w", err)
		}
		s = string(b)
	}
	fgc, err := parseColor(*fg)
	if err != nil {
		return fmt.Errorf("-color: %w", err)
	}
	bgc, err := parseColor(*bg)
	if err != nil {
		return fmt.Errorf("-bg: %w", err)
	}

	font, err := loadFont(*assets, *base)
	if err != nil {
		return err
	}

	w, h := font.Measure(s)
	rows := 1
	if *compare {
		rows = 2
		if cw := cellWidth(font, s); cw > w {
			w = cw
		}
	}
	if w <= 0 || h <= 0 {
		return fmt.Errorf("%q measures %dx%d in %s: nothing to draw", s, w, h, *base)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h*rows))
	for i := range img.Pix {
		img.Pix[i] = 0
	}
	fill(img, bgc)
	font.Draw(img, s, 0, 0, fgc)
	if *compare {
		drawByCell(img, font, s, 0, h, fgc)
	}

	return writePNG(*outPath, magnify(img, *scale))
}

// cellWidth is how wide the string would be if a consumer advanced by the CELL
// instead of the advance — the defect this tool exists to make visible, and the
// only place in the program that computes it.
func cellWidth(f *text.Font, s string) int {
	w := 0
	for i := 0; i < len(s); i++ {
		if g := f.GlyphFor(s[i]); g != nil {
			w += g.Width
		}
	}
	return w
}

// drawByCell draws s one byte at a time at cell pitch, so a reviewer can see
// what the wrong pitch looks like beside the right one.
func drawByCell(dst *image.RGBA, f *text.Font, s string, x, y int, c color.RGBA) {
	for i := 0; i < len(s); i++ {
		g := f.GlyphFor(s[i])
		if g == nil {
			continue
		}
		f.Draw(dst, s[i:i+1], x, y, c)
		x += g.Width
	}
}

func fill(img *image.RGBA, c color.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func magnify(src *image.RGBA, n int) *image.RGBA {
	if n == 1 {
		return src
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx()*n, b.Dy()*n))
	for y := 0; y < b.Dy()*n; y++ {
		for x := 0; x < b.Dx()*n; x++ {
			dst.SetRGBA(x, y, src.RGBAAt(b.Min.X+x/n, b.Min.Y+y/n))
		}
	}
	return dst
}

func parseColor(s string) (color.RGBA, error) {
	v, err := strconv.ParseUint(strings.TrimPrefix(s, "#"), 16, 32)
	if err != nil || len(strings.TrimPrefix(s, "#")) != 6 {
		return color.RGBA{}, fmt.Errorf("want RRGGBB, got %q", s)
	}
	return color.RGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}, nil
}

func writePNG(name string, img image.Image) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
