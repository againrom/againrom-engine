// Command mapedit inspects authored ALM content and saves explicit edited
// copies outside installations, without simulation.
package main

import (
	"flag"
	"fmt"
	"image/png"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"againrom/pkg/game"
	"againrom/pkg/ui"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "mapedit:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("mapedit", flag.ContinueOnError)
	assets := flags.String("assets", "", "read-only installed asset root")
	path := flags.String("map", "", "explicit ALM path or catalogue key, e.g. scenario/10.alm")
	list := flags.Bool("list", false, "list installed map keys and exit")
	check := flags.Bool("check", false, "load and report every record/warning without opening a window")
	snapshot := flags.String("snapshot", "", "write a headless diagnostic PNG and exit (not GPU readback)")
	selectRecord := flags.String("select", "", "select and jump to Kind:index, e.g. Sack:0")
	size := flags.String("size", "1280x800", "window dimensions for a diagnostic snapshot")
	detailScroll := flags.Int("scroll-details", 0, "wheel steps down the selected inspector in a diagnostic snapshot")
	triggers := flags.Bool("triggers", false, "show the authored trigger filter")
	focusRef := flags.Int("focus-reference", 0, "focus a selected record's 1-based map reference without changing selection")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	parts := strings.Split(*size, "x")
	w, h := 0, 0
	if len(parts) == 2 {
		w, _ = strconv.Atoi(parts[0])
		h, _ = strconv.Atoi(parts[1])
	}
	if w < 640 || h < 480 || w > 3840 || h > 2160 {
		return fmt.Errorf("invalid size %q (640x480..3840x2160)", *size)
	}
	if *detailScroll < 0 || *detailScroll > 100000 || (*detailScroll > 0 && (*snapshot == "" || *selectRecord == "")) {
		return fmt.Errorf("-scroll-details requires -snapshot and -select; range 0..100000")
	}
	if *check && *path == "" {
		return fmt.Errorf("-check requires -map")
	}
	if *focusRef < 0 || (*focusRef > 0 && *selectRecord == "") {
		return fmt.Errorf("-focus-reference requires -select and a positive reference index")
	}
	x, err := game.NewMapInspector(game.ResolveAssetRoot(*assets, os.Getenv("AGAINROM_ASSETS")))
	if err != nil {
		return err
	}
	if *list {
		for _, m := range x.Maps {
			prefix := "loose/"
			if m.FromArchive {
				prefix = "scenario/"
			}
			fmt.Fprintf(out, "%s%s\t%s\n", prefix, m.Source, m.Text())
		}
		return nil
	}
	e := x.Editor()
	if *path != "" {
		if err := e.Open(*path); err != nil {
			return err
		}
	}
	e.Layout(w, h)
	if *selectRecord != "" {
		parts := strings.Split(*selectRecord, ":")
		if len(parts) != 2 || e.Document() == nil {
			return fmt.Errorf("-select requires a map and Kind:index")
		}
		n, err := strconv.Atoi(parts[1])
		if err != nil {
			return err
		}
		found := false
		for i, r := range e.Document().Records {
			if strings.EqualFold(parts[0], r.Kind) && r.Index == n {
				e.Select(i, true)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("record %q not found", *selectRecord)
		}
	}
	if *triggers {
		_ = e.Key("triggers")
	}
	if *focusRef > 0 && !e.FocusReference(*focusRef) {
		return fmt.Errorf("map reference %d is unavailable", *focusRef)
	}
	if *snapshot != "" {
		e.Pointer(ui.Input{CursorX: w - 1, CursorY: h - 70, WheelY: -float64(*detailScroll)}, time.Time{})
		pic, note, err := e.HeadlessFrame()
		if err != nil {
			return err
		}
		f, err := os.OpenFile(*snapshot, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		err = png.Encode(f, pic)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		fmt.Fprintln(out, note)
		return nil
	}
	if *check {
		if e.Document() == nil {
			return fmt.Errorf("-check requires -map")
		}
		d := e.Document()
		fmt.Fprintf(out, "mapedit: %s %dx%d; %d records; authored data, no simulation\n", d.Source, d.Width, d.Height, len(d.Records))
		for _, r := range d.Records {
			fmt.Fprintf(out, "%s #%d type=%d spatial=%v cell=%d,%d %s", r.Kind, r.Index, r.Section, r.Spatial, r.Cell.X, r.Cell.Y, r.Label)
			if r.Warning != "" {
				fmt.Fprintf(out, " WARNING: %s", r.Warning)
			}
			fmt.Fprintln(out)
			if *selectRecord != "" && r.Kind == d.Records[e.Selected()].Kind && r.Index == d.Records[e.Selected()].Index {
				for _, line := range r.Lines {
					fmt.Fprintf(out, "  %s", line.Text)
					if line.Reference > 0 {
						fmt.Fprintf(out, " [map reference %d]", line.Reference)
					}
					fmt.Fprintln(out)
				}
			}
		}
		return nil
	}
	return e.Run()
}
