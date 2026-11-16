// Command editormockup turns a map editor window's description into a
// mockup and back.
//
//	editormockup page -desc triggerwizard -lang ru -size 800x600 -out wizard.html [-assets DIR]
//	editormockup desc -from wizard.html -out wizard.json
//	editormockup png -desc triggerwizard -lang ru -size 800x600 -out wizard.png [-assets DIR]
//
// page places every element at the popup layout's rectangle and carries the
// description; desc reads it back from a page or file; png paints with the
// editor's popup painter. The install under -assets is only read, for fonts.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"

	"againrom/pkg/game"
	"againrom/pkg/locale"
	"againrom/pkg/ui"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "editormockup:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || (args[0] != "page" && args[0] != "desc" && args[0] != "png") {
		return errors.New("usage: editormockup page|desc|png [flags]")
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	desc := fs.String("desc", "triggerwizard", "an embedded window, or a description file or page")
	from := fs.String("from", "", "the page or description file desc reads")
	lang := fs.String("lang", locale.Fallback, "the words' language")
	size := fs.String("size", "800x600", "the editor window's size, WxH")
	out := fs.String("out", "", "the file to write (required)")
	assets := fs.String("assets", "", "the install root, only read (default AGAINROM_ASSETS)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *out == "" {
		return errors.New("-out is required")
	}
	root := game.ResolveAssetRoot(*assets, os.Getenv("AGAINROM_ASSETS"))
	if err := game.RefuseInstallWrite(root, *out); err != nil {
		return err
	}
	if args[0] == "desc" {
		data, err := os.ReadFile(*from)
		if err != nil {
			return err
		}
		d, err := ui.PageDescription(data)
		if err != nil {
			return err
		}
		return os.WriteFile(*out, d, 0o644)
	}
	d, err := description(*desc)
	if err != nil {
		return err
	}
	window, err := ui.ParseWindowSize(*size, image.Pt(800, 600))
	if err != nil {
		return err
	}
	if !locale.Known(*lang) {
		return fmt.Errorf("-lang %q: not a language of the locale table", *lang)
	}
	var m *ui.EditorMockup
	switch {
	case root != "":
		x, err := game.NewMapInspector(root)
		if err != nil {
			return err
		}
		m, err = x.Editor().Mockup(d, *lang, window)
		if err != nil {
			return err
		}
	case args[0] == "png":
		return errors.New("png paints with the install's fonts: pass -assets or set AGAINROM_ASSETS")
	default:
		if m, err = ui.NewEditorMockup(d, *lang, window, nil, nil, nil); err != nil {
			return err
		}
	}
	if args[0] == "page" {
		return os.WriteFile(*out, m.Page(), 0o644)
	}
	var b bytes.Buffer
	if err := png.Encode(&b, m.Picture()); err != nil {
		return err
	}
	return os.WriteFile(*out, b.Bytes(), 0o644)
}

// description is the embedded window name, or the description in the file
// or page at that path.
func description(name string) ([]byte, error) {
	if data, err := os.ReadFile(name); err == nil {
		return ui.PageDescription(data)
	}
	return ui.ScreenDescription(name)
}
