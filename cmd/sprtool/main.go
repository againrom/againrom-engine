// Command sprtool decodes .256, .16a and .16 sprites to PNG (developer tool).
//
// Usage:
//
//	sprtool png <archive> <path> <dir>     decode one .256 entry, one PNG per frame
//	sprtool png16a <archive> <path> <dir>  decode one .16a entry, one PNG per frame
//	sprtool png16 <archive> <path> <dir>   decode one .16 entry, one PNG per frame
//
// sprtool reads the entry through the 0001 .res archive reader, decodes it with
// pkg/formats/spr256 or pkg/formats/spr16, and writes each frame as a PNG under
// <dir>. png maps opaque .256 pixels through the palette to RGBA and
// transparent pixels to alpha 0 (a palette-less sprite falls back to a
// grayscale ramp on the index). The 16-bit commands each carry one viewing
// convention of this tool — a presentation choice, never a format fact —
// printed to stderr on every run: png16a declares the palette present (the
// tool's own fixed declaration) and draws a painted pixel as its palette RGB at
// alpha level*17, a transparent pixel as alpha 0; png16 draws a painted value
// as opaque gray value*17, a transparent pixel as alpha 0. It is a
// developer-run tool for verifying the decoders against a lawful install and is
// never part of the test suite. Point <dir> at a git-ignored folder (the repo
// ignores sprtool-out/): decoded frames are converted game assets and must
// never be committed.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"againrom/pkg/formats/res"
	"againrom/pkg/formats/spr16"
	"againrom/pkg/formats/spr256"
)

// The 16-bit commands' viewing conventions — presentation choices of this
// tool, never format facts. Each is disclosed twice: in the usage text below
// and as one line to stderr on every run. The palette declaration is png16a's
// own fixed choice (presence is the consumer's declaration, so a viewer must
// make one), and the *17 ramps stretch a 4-bit level or value over 0..255
// (15*17 = 255).
const (
	view16A = "palette declared present (this tool's fixed declaration); painted pixel = the game's own resolution, palette RGB at coverage (level+1)/16 premultiplied, transparent = alpha 0"
	view16G = "painted value = opaque gray value*17, transparent = alpha 0"
)

const usageText = "usage: sprtool <png|png16a|png16> <archive> <path> <dir>\n" +
	"  png     decode one .256 entry, one PNG per frame\n" +
	"  png16a  decode one .16a entry, one PNG per frame; view: " + view16A + "\n" +
	"  png16   decode one .16 entry, one PNG per frame; view: " + view16G

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "sprtool:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s", usageText)
	}
	switch args[0] {
	case "png":
		if len(args) != 4 {
			return fmt.Errorf("usage: sprtool png <archive> <path> <dir>")
		}
		return doPNG(args[1], args[2], args[3])
	case "png16a":
		if len(args) != 4 {
			return fmt.Errorf("usage: sprtool png16a <archive> <path> <dir>")
		}
		return doPNG16A(args[1], args[2], args[3])
	case "png16":
		if len(args) != 4 {
			return fmt.Errorf("usage: sprtool png16 <archive> <path> <dir>")
		}
		return doPNG16(args[1], args[2], args[3])
	default:
		return fmt.Errorf("unknown command %q\n%s", args[0], usageText)
	}
}

func doPNG(archivePath, entryPath, dir string) error {
	a, err := res.Open(archivePath)
	if err != nil {
		return err
	}
	data, err := a.ReadFile(entryPath)
	if err != nil {
		return err
	}
	sprite, err := spr256.Decode(data)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	table := viewTable(sprite)
	written := 0
	for i, f := range sprite.Frames {
		if f.Width == 0 || f.Height == 0 {
			fmt.Fprintf(os.Stderr, "sprtool: frame %d is empty (%dx%d); skipped\n", i, f.Width, f.Height)
			continue
		}
		img := f.RGBA(table)
		name := filepath.Join(dir, fmt.Sprintf("frame_%03d.png", i))
		if err := writePNG(name, img); err != nil {
			return err
		}
		written++
	}
	fmt.Fprintf(os.Stderr, "sprtool: wrote %d of %d frames to %s\n", written, len(sprite.Frames), dir)
	return nil
}

// doPNG16A decodes one .16a entry and writes one PNG per frame. The palette
// declaration and the alpha ramp are this tool's viewing conventions (view16A),
// printed to stderr on every run.
func doPNG16A(archivePath, entryPath, dir string) error {
	fmt.Fprintln(os.Stderr, "sprtool: png16a view: "+view16A)
	data, err := readEntry(archivePath, entryPath)
	if err != nil {
		return err
	}
	sprite, err := spr16.DecodeA(data, true)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	written := 0
	for i, f := range sprite.Frames {
		if f.Width == 0 || f.Height == 0 {
			fmt.Fprintf(os.Stderr, "sprtool: frame %d is empty (%dx%d); skipped\n", i, f.Width, f.Height)
			continue
		}
		img := f.RGBA(sprite.Palette)
		name := filepath.Join(dir, fmt.Sprintf("frame_%03d.png", i))
		if err := writePNG(name, img); err != nil {
			return err
		}
		written++
	}
	fmt.Fprintf(os.Stderr, "sprtool: wrote %d of %d frames to %s\n", written, len(sprite.Frames), dir)
	return nil
}

// doPNG16 decodes one .16 entry and writes one PNG per frame. The gray ramp is
// this tool's viewing convention (view16G), printed to stderr on every run.
func doPNG16(archivePath, entryPath, dir string) error {
	fmt.Fprintln(os.Stderr, "sprtool: png16 view: "+view16G)
	data, err := readEntry(archivePath, entryPath)
	if err != nil {
		return err
	}
	frames, err := spr16.DecodeG(data)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	written := 0
	for i, f := range frames {
		if f.Width == 0 || f.Height == 0 {
			fmt.Fprintf(os.Stderr, "sprtool: frame %d is empty (%dx%d); skipped\n", i, f.Width, f.Height)
			continue
		}
		img := image.NewNRGBA(image.Rect(0, 0, f.Width, f.Height))
		for y := 0; y < f.Height; y++ {
			for x := 0; x < f.Width; x++ {
				px := f.Pixels[y*f.Width+x]
				if !px.Painted {
					continue // leave the pixel fully transparent (alpha 0)
				}
				v := px.Value * 17
				img.SetNRGBA(x, y, color.NRGBA{R: v, G: v, B: v, A: 0xFF})
			}
		}
		name := filepath.Join(dir, fmt.Sprintf("frame_%03d.png", i))
		if err := writePNG(name, img); err != nil {
			return err
		}
		written++
	}
	fmt.Fprintf(os.Stderr, "sprtool: wrote %d of %d frames to %s\n", written, len(frames), dir)
	return nil
}

// readEntry reads one entry's bytes through the 0001 .res archive reader — the
// 16-bit commands' shared plumbing; png keeps its own path unchanged.
func readEntry(archivePath, entryPath string) ([]byte, error) {
	a, err := res.Open(archivePath)
	if err != nil {
		return nil, err
	}
	return a.ReadFile(entryPath)
}

// viewTable is the sheet's own table, or for the no-palette variant (whose real
// table is borrowed elsewhere and is the consumer's concern) a grey ramp, so the
// frame is still inspectable.
func viewTable(s *spr256.Sprite) *[256]color.RGBA {
	if t, ok := s.Table(); ok {
		return t
	}
	var ramp [256]color.RGBA
	for i := range ramp {
		ramp[i] = color.RGBA{R: uint8(i), G: uint8(i), B: uint8(i), A: 0xFF}
	}
	return &ramp
}

func writePNG(name string, img image.Image) error {
	out, err := os.Create(name)
	if err != nil {
		return err
	}
	if err := png.Encode(out, img); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
