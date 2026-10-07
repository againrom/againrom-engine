// Command screenshot composes named screens against a lawful game install,
// through production code paths, and writes each as a PNG.
//
//	screenshot -list
//	screenshot [-assets DIR] -out DIR [-screens NAME,NAME,...]
//
// IT REPLACES SIX RE-IMPLEMENTATIONS OF ONE DECISION. Before this tool
// existed, plaquescreens, townsquarecheck, schoolcheck, shopdump, paneldump
// and plaqueseams each picked which pkg/ui Compose* function draws a given
// screen by hand, one call site per tool, so a later change to
// pkg/ui/app.go's own Draw could stop matching what those six produced and
// nothing would say so. This tool selects a composer nowhere in its own code:
// every screen goes through ui.App.HeadlessFrame or ui.ComposeTownScreen.
//
// Draw does not call either of those two. It reaches the same composition
// through its own siblings — (*App).composeTownScreen and (*App)
// .composeTownRoom — and the convergence is one level down: HeadlessFrame
// and Draw both call composeScreen's single switch, and ComposeTownScreen
// and (*App).composeTownRoom both call the package-level composeTownRoom.
// That shared floor is what makes a change to Draw's own selection change
// this tool's output too. An earlier revision of this header said Draw goes
// through the two exported seams, which it does not (pass-1 adversarial
// review).
//
// SIX SCREENS ARE REACHED THROUGH ONE LIVE ui.App, driven the way a
// keyboard session drives it — game.FrontEnd.App(title) plus the Headless*
// dispatch cmd/menuaccelcheck already uses: the root menu; the load window
// (Escape, New Game); the map picker; the generation screen's pre-create and
// detailed pages (the first offered picture, then Forward); the mission
// screen once Play is pressed; and the in-game menu raised over it with
// Escape. A screen this build cannot compose on the CPU is refused, never
// rendered by a second, tool-local decision — see HeadlessFrame's own
// header in pkg/ui/headless.go for the exact boundary.
//
// TWO SCREENS ARE REACHED WITH NO ui.App, through the FrontEnd-level path
// cmd/shopdump and cmd/plaquescreens already use: FinishMission opens the
// first campaign mission the registry marks as offering a town, and
// TownScreen() reads the result. Reaching a live App's own ScreenTown needs
// a mission finished through the sim (pkg/ui/flow.go's NoticeToTown), which
// this tool does not run; composeTownRoom's own header in pkg/ui/app.go
// says why FinishMission is the production path here instead. The square's
// own SHOP door, pressed exactly as a click would press it, opens the shop
// room.
//
// It writes only PNGs, only under -out, and no game data. -assets falls
// back to AGAINROM_ASSETS; the writable save store it opens the load window
// against is a scratch temporary directory this process creates and
// removes, never the install.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"againrom/pkg/game"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// screen is one named capture this tool knows how to attempt: how it is
// reached (session, driven through one live ui.App in sequence, or town,
// reached through FinishMission with no App) and whether this build is
// expected to compose it on the CPU at all. expectCompose false means the
// boundary is already known and documented (composeScreen's own header) —
// a refusal there is the tool working, not a failure; expectCompose true
// means a refusal or drive failure is reported as one.
type screen struct {
	name          string
	kind          string // "session" or "town"
	expectCompose bool
	note          string
}

var knownScreens = []screen{
	{"menu", "session", true, "the root screen, composed by menu.Assets.Compose"},
	{"load", "session", true, "the load list, composed with the installed font"},
	{"picker", "session", false, "ScreenPicker draws only through ebitenutil.DebugPrintAt"},
	{"chargen-precreate", "session", true, "the generation screen's pre-create page"},
	{"chargen-detailed", "session", true, "the generation screen's detailed page, after Forward"},
	{"gameplay", "session", false, "the mission screen: Viewer.Draw paints the ebiten canvas directly, no CPU composite"},
	{"mission-pane-doll", "session", true, "the mission character pane in figure mode, the one part of the mission screen composed on the CPU"},
	{"mission-pane-stats", "session", true, "the statistics card below the mission figure (legacy capture name)"},
	{"mission-column-card", "session", true, "the mission column's own fourth box, the statistics card that stands beside the figure (`DIV-344`)"},
	{"documents", "session", true, "the campaign documents panel, opened by double-clicking the access item in the pack bar"},
	{"game-menu", "session", false, "the in-game menu over a running mission: ebiten vector dim and panel, no CPU composite"},
	{"town-square", "town", true, "the town square, once the campaign's first town-declaring mission finishes"},
	{"town-tavern", "town", true, "the tavern before a candidate is selected"},
	{"town-tavern-selected", "town", true, "the tavern with one mercenary selected and inspected"},
	{"town-tavern-next-frame", "town", true, "the selected mercenary on its next shipped animation frame"},
	{"town-tavern-talk", "town", true, "the selected mercenary's authored Talk modal and portrait"},
	{"town-shop", "town", true, "the shop room, entered through the square's own SHOP door"},
	{"town-shop-stats", "town", true, "the shop room with the character pane's DOLL/STATS toggle pressed"},
}

func run(args []string) int {
	fs := flag.NewFlagSet("screenshot", flag.ContinueOnError)
	assets := fs.String("assets", "", "game install root (or AGAINROM_ASSETS)")
	out := fs.String("out", "", "directory to write PNGs into")
	screensFlag := fs.String("screens", "", "comma-separated screen names to capture (default: every known screen)")
	list := fs.Bool("list", false, "print every screen this tool knows how to reach and exit")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *list {
		printList()
		return 0
	}

	root := game.ResolveAssetRoot(*assets, os.Getenv("AGAINROM_ASSETS"))
	if root == "" {
		fmt.Fprintln(os.Stderr, "screenshot: no asset root: pass -assets or set AGAINROM_ASSETS")
		return 2
	}
	if *out == "" {
		fmt.Fprintln(os.Stderr, "screenshot: -out is required")
		return 2
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "screenshot:", err)
		return 2
	}

	requested, err := selectScreens(*screensFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "screenshot:", err)
		return 2
	}
	fmt.Printf("requested %d screen(s): %s\n", len(requested), joinNames(requested))

	f, err := game.NewFrontEnd(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "screenshot:", err)
		return 2
	}

	results := capture(f, root, *out, requested)
	return report(results)
}

func printList() {
	fmt.Printf("%d known screen(s):\n", len(knownScreens))
	for _, s := range knownScreens {
		compose := "refuses (no CPU composite)"
		if s.expectCompose {
			compose = "composes"
		}
		fmt.Printf("  %-18s %-7s %-24s %s\n", s.name, s.kind, compose, s.note)
	}
}

func selectScreens(flagValue string) ([]screen, error) {
	if strings.TrimSpace(flagValue) == "" {
		return append([]screen(nil), knownScreens...), nil
	}
	byName := make(map[string]screen, len(knownScreens))
	for _, s := range knownScreens {
		byName[s.name] = s
	}
	var out []screen
	for _, name := range strings.Split(flagValue, ",") {
		name = strings.TrimSpace(name)
		s, ok := byName[name]
		if !ok {
			names := make([]string, 0, len(knownScreens))
			for _, k := range knownScreens {
				names = append(names, k.name)
			}
			sort.Strings(names)
			return nil, fmt.Errorf("unknown screen %q (known: %s)", name, strings.Join(names, ", "))
		}
		out = append(out, s)
	}
	return out, nil
}

func joinNames(screens []screen) string {
	names := make([]string, len(screens))
	for i, s := range screens {
		names[i] = s.name
	}
	return strings.Join(names, ", ")
}

// result is one screen's outcome: reached names whether the drive got far
// enough to ask composeScreen/ComposeTownScreen at all. composed and err
// are mutually exclusive once reached is true; unmet is true exactly when
// the outcome did not match the screen's own expectCompose, which is what
// report below turns into a non-zero exit.
type result struct {
	screen   screen
	reached  bool
	composed bool
	note     string
	err      error
	unmet    bool
}

func capture(f *game.FrontEnd, root, outDir string, requested []screen) []result {
	// driveSession and driveTown each run their whole fixed sequence no
	// matter what was requested — the sequence is a chain (menu -> ... ->
	// chargen-detailed -> gameplay), so reaching one requested screen deep
	// in it means passing through every screen before it. Only the loop
	// below, over requested, turns a frame into a PNG or a printed line.
	raw := make(map[string]capturedFrame, len(knownScreens))
	for name, c := range driveSession(f, root) {
		raw[name] = c
	}
	townFront, err := game.NewFrontEnd(root)
	if err != nil {
		failure := fmt.Errorf("open an independent town drive: %w", err)
		for _, name := range []string{"town-square", "town-tavern", "town-tavern-selected", "town-tavern-next-frame", "town-tavern-talk", "town-shop", "town-shop-stats"} {
			raw[name] = capturedFrame{err: failure}
		}
	} else {
		for name, c := range driveTown(townFront) {
			raw[name] = c
		}
	}

	out := make([]result, 0, len(requested))
	for _, s := range requested {
		var r result
		if c, ok := raw[s.name]; ok {
			r = finishResult(s.name, outDir, c)
		} else {
			r = result{reached: false, err: fmt.Errorf("this drive never reached %q", s.name)}
		}
		r.screen = s
		r.unmet = r.screen.expectCompose != r.composed
		out = append(out, r)
		printResult(r)
	}
	return out
}

// capturedFrame is one screen's own outcome before it is matched against
// what the caller requested. tried is true once the drive actually asked
// HeadlessFrame or ComposeTownScreen for this screen; it is false when an
// earlier stage the drive needed never produced the state this screen
// depends on, or when this screen's own precondition (a shop door on the
// square, for town-shop) was absent before any compose was attempted. pix
// is nil and err names why whenever this build refused, or the drive did
// not reach this screen at all.
type capturedFrame struct {
	pix   *image.RGBA
	note  string
	tried bool
	err   error
}

func finishResult(name, outDir string, c capturedFrame) result {
	r := result{reached: c.tried}
	if c.err != nil {
		r.err = c.err
		return r
	}
	r.composed = true
	r.note = c.note
	path := filepath.Join(outDir, name+".png")
	if err := writePNG(path, c.pix); err != nil {
		r.composed = false
		r.err = fmt.Errorf("write %s: %w", path, err)
		return r
	}
	return r
}

func printResult(r result) {
	switch {
	case !r.reached:
		fmt.Printf("%-18s NOT REACHED: %v\n", r.screen.name, r.err)
	case r.composed:
		fmt.Printf("%-18s captured -> %s.png\n", r.screen.name, r.screen.name)
		if r.note != "" {
			fmt.Printf("%-18s note: %s\n", r.screen.name, r.note)
		}
	default:
		fmt.Printf("%-18s refused: %v\n", r.screen.name, r.err)
	}
	if r.unmet {
		fmt.Printf("%-18s UNEXPECTED: wanted compose=%v, got compose=%v\n", r.screen.name, r.screen.expectCompose, r.composed)
	}
}

func report(results []result) int {
	captured, refused, unmet := 0, 0, 0
	for _, r := range results {
		switch {
		case r.composed:
			captured++
		default:
			refused++
		}
		if r.unmet {
			unmet++
		}
	}
	fmt.Printf("%d requested, %d captured, %d refused, %d unmet expectation(s)\n",
		len(results), captured, refused, unmet)
	if unmet > 0 {
		return 1
	}
	return 0
}

func writePNG(path string, pix *image.RGBA) error {
	fh, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fh.Close()
	return png.Encode(fh, pix)
}
