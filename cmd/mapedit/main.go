// Command mapedit inspects authored ALM content and saves explicit edited
// copies outside installations, without simulation. Its test play key (F5)
// writes the map as it stands into a test mod in the player profile and starts
// the game beside this program in that mission.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"againrom/internal/appicon"
	"againrom/pkg/game"
	"againrom/pkg/launch"
	"againrom/pkg/pathdialog"
	"againrom/pkg/ui"
)

// desktop is the window system the editor ends in: the call that gives the
// window its icon and the call that runs the editor in it. A test replaces both.
var desktop = struct {
	setIcon func([]image.Image)
	run     func(*ui.MapEditor) error
}{ui.SetWindowIcon, (*ui.MapEditor).Run}

// openWindow gives the editor's window its own icon and runs the editor.
func openWindow(e *ui.MapEditor) error {
	desktop.setIcon(appicon.Images(appicon.MapEditor))
	return desktop.run(e)
}

// newInspector opens the install at a root for the editor. A test replaces it.
var newInspector = game.NewMapInspector

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		userConfigDir, _ := os.UserConfigDir()
		reportFailure(os.Stderr, userConfigDir, err)
		os.Exit(1)
	}
}

// errorLogName is the file beside the player profile that holds the error of a
// start that could not show it.
const errorLogName = "mapedit-error.log"

// reportFailure shows why the program stopped. It writes the error to stderr; a
// program started without a console has no stderr to write to, and then the
// error is written to errorLogName in the Againrom folder of the user
// configuration directory, the folder the player profile falls back to.
func reportFailure(stderr io.Writer, userConfigDir string, err error) {
	line := "mapedit: " + err.Error() + "\n"
	if _, werr := io.WriteString(stderr, line); werr == nil {
		return
	}
	if userConfigDir == "" {
		return
	}
	dir := filepath.Join(userConfigDir, "Againrom")
	if os.MkdirAll(dir, 0o700) == nil {
		_ = os.WriteFile(filepath.Join(dir, errorLogName), []byte(line), 0o600)
	}
}

// assetRoot is the install the editor reads, from the first source that names
// one: -assets, AGAINROM_ASSETS, the install the program's own folders hold
// (the discovery the game makes), the install that holds the map being opened,
// and the base the starter beside the program has selected.
func assetRoot(flagValue, envValue, mapPath, self string) string {
	if root := game.ResolveAssetRoot(flagValue, envValue); root != "" {
		return root
	}
	if root := game.DiscoverAssetRoot(game.ProcessDirs(self)...); root != "" {
		return root
	}
	if root, ok := game.InstallHolding(mapPath); ok {
		return root
	}
	if self != "" {
		if base, ok := launch.StarterBase(filepath.Dir(self)); ok {
			return game.DiscoverAssetRoot(base)
		}
	}
	return ""
}

func run(args []string, out io.Writer) error {
	self, _ := os.Executable()
	return runAs(self, args, out)
}

// runAs is the command as the program at self: its folders and the starter
// beside it name the install when no flag or variable does.
func runAs(self string, args []string, out io.Writer) error {
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
	newMap := flags.String("new", "", "start a new map of this size, e.g. 32x32 (width and height, 32 to 256 in steps of 8), or 32x32/altitude40 for a starting height of 0 to 126 (64 without it); with -out the blank map is written and the program exits")
	out2 := flags.String("out", "", "with -new: the new .alm file to write the blank map to; the report is printed and the program exits")
	modDir := flags.String("mod", "", "mod folder Save writes into: Ctrl+S writes the map to the path its data/missions.toml entry names, and refuses a map no entry names")
	mcp := flags.String("mcp", "", "optional editor MCP transport: stdio or http")
	mcpAddress := flags.String("mcp-address", "127.0.0.1:0", "MCP HTTP loopback address; port 0 reuses the recorded port when it is free, else chooses a free port")
	mcpNewToken := flags.Bool("mcp-new-token", false, "make a new MCP bearer token and record it; without this the token in the profile's mapedit-mcp.json is reused")
	mcpPace := flags.Duration("mcp-pace", 0, "pause between MCP tool calls (0..1m)")
	version := flags.Bool("version", false, "print the program name, game version and source revision, and exit")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *version {
		fmt.Fprintln(out, versionLine())
		return nil
	}
	if *mcp != "" && *mcp != "stdio" && *mcp != "http" {
		return fmt.Errorf("-mcp must be stdio or http")
	}
	if *mcpPace < 0 || *mcpPace > time.Minute {
		return fmt.Errorf("-mcp-pace must be between 0 and 1m")
	}
	if *mcp != "" && (*list || *check || *snapshot != "" || *out2 != "") {
		return fmt.Errorf("-mcp requires the editor window, excluding -list, -check, -snapshot and -out")
	}
	if flags.NArg() > 1 {
		return fmt.Errorf("unexpected arguments: %v; give one map path, and flags before it", flags.Args())
	}
	if flags.NArg() == 1 {
		if *path != "" || *newMap != "" {
			return fmt.Errorf("a map path argument %q excludes -map and -new", flags.Arg(0))
		}
		*path = flags.Arg(0)
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
	if *newMap != "" && *path != "" {
		return fmt.Errorf("-new and -map exclude each other")
	}
	if *out2 != "" && *newMap == "" {
		return fmt.Errorf("-out requires -new")
	}
	if *newMap != "" {
		*path = game.NewMapSource(*newMap)
	}
	if *check && *path == "" {
		return fmt.Errorf("-check requires -map")
	}
	if *focusRef < 0 || (*focusRef > 0 && *selectRecord == "") {
		return fmt.Errorf("-focus-reference requires -select and a positive reference index")
	}
	root := assetRoot(*assets, os.Getenv("AGAINROM_ASSETS"), *path, self)
	if root == "" {
		return fmt.Errorf("no asset root: pass -assets, set AGAINROM_ASSETS, or keep the program in the game folder")
	}
	x, err := newInspector(root)
	if err != nil {
		return err
	}
	userConfigDir, _ := os.UserConfigDir()
	x.SetTestPlay(testPlayHost(self, userConfigDir, root))
	if *modDir != "" {
		if err := x.SetMod(*modDir); err != nil {
			return err
		}
	}
	if *out2 != "" {
		target, rows, err := x.WriteNewMap(*path, *out2)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "mapedit: wrote %s (%s)\n", target, *newMap)
		return printFindings(out, rows)
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
		e.ScrollDetails(*detailScroll)
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
		return reportFindings(out, d)
	}
	e.SetPathDialog(pathdialog.Show, root)
	e.SetMapsFolder(x.MapsFolder)
	if *mcp != "" {
		stop, err := startMCP(e, *mcp, *mcpAddress, *mcpNewToken, *mcpPace, self, userConfigDir, root, os.Stdin, out, os.Stderr)
		if err != nil {
			return err
		}
		defer stop()
	}
	return openWindow(e)
}

// reportFindings prints the map report the window shows, one line per finding
// and a count line, and fails when the map holds an error.
func reportFindings(out io.Writer, d *ui.InspectionDocument) error {
	c, ok := d.Edits.(ui.InspectionChecker)
	if !ok {
		return nil
	}
	rows := c.Report()
	if err := printFindings(out, rows); err != nil {
		return err
	}
	if errs, _ := ui.CountFindings(rows); errs > 0 {
		return fmt.Errorf("%s holds %d errors", d.Source, errs)
	}
	return nil
}

// printFindings prints one line per finding and the count line.
func printFindings(out io.Writer, rows []ui.InspectionFinding) error {
	for _, f := range rows {
		fmt.Fprintln(out, f.Line())
	}
	errs, warnings := ui.CountFindings(rows)
	fmt.Fprintf(out, "report: %d errors, %d warnings\n", errs, warnings)
	return nil
}

// testPlayHost is the test play of the editor at self over the install root:
// the player profile that program resolves, and the game beside it started
// through the one launch.
func testPlayHost(self, userConfigDir, root string) game.TestPlayHost {
	return game.TestPlayHost{
		Profile: func() (game.RuntimeProfile, error) {
			return game.ResolveRuntimeProfile("", self, userConfigDir, root)
		},
		Game:  launch.Beside(self, "againrom", runtime.GOOS),
		Start: func(program string, args []string) error { return launch.For(program, args).Start() },
	}
}
