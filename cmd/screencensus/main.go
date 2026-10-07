// Command screencensus prints one census of the screen composition and
// screen testing surface this build carries, contract B4's five columns:
// screens, composers, synthetic witnesses, gated witnesses, and the
// unwitnessed remainder.
//
// It prints what it selected, not only what it found (AGENTS.md: a census
// that only prints a found-count cannot be told apart from one whose
// selector matched nothing). Every row names its own source.
//
//	screencensus [-module DIR]
//
// TWO GRANULARITIES ARE NAMED AND NEITHER IS HIDDEN INSIDE THE OTHER.
// cmd/screenshot's own "-list" names 9 screens (chargen's pre-create and
// detailed pages, and the town square and shop rooms, each counted
// separately). pkg/ui's own screen registry (screenregistry.go) is indexed
// by the 7-value ui.Screen enum, one value per composeScreen dispatch arm —
// ScreenChargen and ScreenTown each cover two of the 9 named screens. The
// SCREENS section below prints cmd/screenshot's 9; the COMPOSERS and
// SYNTHETIC WITNESSES sections print the registry's 7, at the registry's
// own granularity, because that is the granularity composeScreen dispatches
// at and the registry's own fail-closed test polices. Reconciling "9" and
// "7" by force-fitting one table into the other would hide exactly the
// coverage gap the UNWITNESSED REMAINDER section exists to name (round 2
// adversarial review, finding 1: chargen-detailed and chargen-precreate
// share one registry row, and that row's own GeometryTests list currently
// covers only chargen-precreate).
//
// The COMPOSERS and SYNTHETIC WITNESSES sections read pkg/ui's exported
// ScreenCensus, which returns screenRegistry's own rows — not a second copy
// of that table kept in sync by hand. SYNTHETIC WITNESSES prints two kinds
// per screen and does not conflate them: Test names the dispatch-selection
// test (composeScreen reaches the right composer; cannot fail when a
// destination rectangle moves), and Geometry names every test that pins a
// destination rectangle or text origin against a hand literal, proved by
// mutation (can fail when one moves — contract B2's own bar).
//
// THE "GAME-MENU" / "GAMEMENU" CELL NAMES TWO DIFFERENT REACHABILITY
// FRAMES, not a disagreement between the two sections (round 3 punch list,
// finding 3). cmd/screenshot's own "-list" table (section 1) names
// "game-menu" as reached over a running mission and refusing — that is the
// map-showing case, which composeScreen's own switch never reaches at all
// (Draw's mapShowing() branch returns first). The registry's "gamemenu" row
// (sections 2/3) answers about composeScreen's ScreenGameMenu dispatch
// value, which this build only ever reaches when the town, not a mission,
// is beneath the menu (Draw's own ScreenGameMenu case comment: "the only
// surface that reaches here is the town") — and even then composes only the
// town frame beneath the menu, not the menu's own dim/panel, which paint
// through ebiten's vector package with no CPU composite either way
// (HeadlessFrame's note branch, pkg/ui/headless.go). Both statements are
// true of the case each names; section 2's own printed output states the
// frame each row answers for, so the two tables do not have to be read
// side by side to reconcile them.
//
// The gated-test section calls internal/gatedtests.Scan directly (no
// subprocess, no install). Reconciling its count against
// pipeline/check-release-tests.sh's own run-time count is a manual step
// this tool does not perform: that script lives one level up, in the
// againrom/ orchestrator tree, outside this repository and its module, and
// a census binary shipped from this repository must not assume a sibling
// directory exists at all, let alone at a fixed relative path. The two
// counts are reconciled by hand, once, at story landing, and the
// comparison is recorded in the story's own closure document. Contract B4
// calls a disagreement between the two counts a failure rather than a note;
// this binary cannot enforce that on its own, for the module-boundary
// reason above, so it prints a note and the reconciliation is a landing
// duty performed by hand — recorded as a deviation from B4's literal
// wording in the story's own closure document, not silently met.
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"againrom/internal/gatedtests"
	"againrom/pkg/ui"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	moduleDir := "."
	for i := 0; i < len(args); i++ {
		if args[i] == "-module" && i+1 < len(args) {
			moduleDir = args[i+1]
		}
	}

	root, err := gatedtests.ModuleRoot(moduleDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "screencensus: locating module root:", err)
		return 2
	}

	screenLines, screenCount, err := listScreens(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "screencensus: running cmd/screenshot -list:", err)
		return 2
	}
	census := ui.ScreenCensus()

	fmt.Printf("== 1. screens (source: go run ./cmd/screenshot -list, %d named screens) ==\n", screenCount)
	fmt.Printf("selected: cmd/screenshot's own %d known screen(s)\n", screenCount)
	for _, line := range screenLines {
		fmt.Println("  " + line)
	}

	fmt.Println()
	fmt.Printf("== 2. composers (source: pkg/ui.ScreenCensus, %d ui.Screen values) ==\n", len(census))
	fmt.Printf("selected: %d registered ui.Screen value(s)\n", len(census))
	for _, e := range census {
		if e.Composer != "" {
			fmt.Printf("  %-16s composes: %s\n", e.Name, e.Composer)
		} else {
			fmt.Printf("  %-16s refuses:  %s\n", e.Name, e.Reason)
		}
	}
	fmt.Println("note: this section's \"gamemenu\" row and section 1's \"game-menu\" row name")
	fmt.Println("different reachability frames — see this file's own header comment.")

	fmt.Println()
	fmt.Println("== 3. synthetic witnesses (source: pkg/ui.ScreenCensus) ==")
	fmt.Println("selected: each composing screen's own selection test (Test) and destination-rectangle tests (Geometry),")
	fmt.Println("plus any geometry test named by a screen that refuses composeScreen")
	for _, e := range census {
		if e.Composer == "" {
			for _, g := range e.GeometryTests {
				fmt.Printf("  %-16s geometry:  %s (refuses composeScreen: no selection witness)\n", e.Name, g)
			}
			continue
		}
		fmt.Printf("  %-16s selection: %s\n", e.Name, e.Test)
		own := len(e.GeometryTests)
		inherited := e.EffectiveGeometryTests[:len(e.EffectiveGeometryTests)-own]
		if e.SharedWith != "" {
			fmt.Printf("  %-16s geometry:  shares %s's own composer call (composeScreen dispatches both to the same function)\n", e.Name, e.SharedWith)
		}
		for _, g := range inherited {
			fmt.Printf("  %-16s geometry:  %s (from %s)\n", e.Name, g, e.SharedWith)
		}
		for _, g := range e.GeometryTests {
			fmt.Printf("  %-16s geometry:  %s\n", e.Name, g)
		}
		if len(e.EffectiveGeometryTests) == 0 {
			fmt.Printf("  %-16s geometry:  none\n", e.Name)
		}
	}

	fmt.Println()
	fmt.Println("== 4. gated witnesses (source: internal/gatedtests.Scan, this module's own source) ==")
	tests, err := gatedtests.Scan(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "screencensus: scanning gated tests:", err)
		return 2
	}
	sort.Slice(tests, func(i, j int) bool { return tests[i].ID() < tests[j].ID() })
	fmt.Printf("selected: %d test function(s) reaching an AGAINROM_-gated t.Skip/t.Skipf through package-local helper calls\n", len(tests))
	for _, t := range tests {
		via := "direct"
		if t.Via != "" {
			via = "via " + t.Via
		}
		fmt.Printf("  %-70s %-8s %s\n", t.ID(), via, gatedtests.Subject(t))
	}
	fmt.Println()
	fmt.Println("reconcile the gated-test count above against pipeline/check-release-tests.sh's own")
	fmt.Println("printed population (that script is outside this repository's module and is not run here).")
	fmt.Println("contract B4 calls a disagreement between the two counts a failure rather than a")
	fmt.Println("note; this binary cannot enforce that across a module boundary it must not assume")
	fmt.Println("exists, so the reconciliation above is a manual landing duty, recorded as a")
	fmt.Println("deviation from B4's literal wording in the story's own closure document.")

	fmt.Println()
	fmt.Println("== 5. unwitnessed remainder (computed from section 2/3 above) ==")
	remainder := unwitnessedRemainder(census)
	fmt.Printf("selected: %d composing ui.Screen value(s) with a Test but no effective Geometry test\n", len(remainder))
	for _, line := range remainder {
		fmt.Println("  " + line)
	}
	gaps := geometryGaps(census)
	fmt.Printf("known geometry gaps recorded in the registry: %d\n", len(gaps))
	for _, line := range gaps {
		fmt.Println("  " + line)
	}

	return 0
}

// unwitnessedRemainder names every composing screen whose EffectiveGeometryTests
// list is empty — a selection witness with no geometry witness of its own or
// shared through another screen's composer, printed explicitly rather than
// left to be inferred from section 3's silence. A screen that shares its
// composed frame with an already-witnessed screen (ScreenCensusEntry.SharedWith)
// is not printed here even when its own GeometryTests is empty: its frame is
// witnessed, by construction, through the shared composer call.
func unwitnessedRemainder(census []ui.ScreenCensusEntry) []string {
	var out []string
	for _, e := range census {
		if e.Composer == "" {
			continue
		}
		if len(e.EffectiveGeometryTests) == 0 {
			out = append(out, fmt.Sprintf("%s: composes (%s), selection witnessed (%s), no geometry witness", e.Name, e.Composer, e.Test))
		}
	}
	return out
}

// geometryGaps names every composing screen carrying a registry
// GeometryGapNote — a known, explicit coverage gap inside a multi-page
// composer (screenregistry.go's own data, not a sentence in this binary).
// Reported separately from unwitnessedRemainder: a screen here can still
// have a non-empty EffectiveGeometryTests (ScreenChargen's pre-create page
// is witnessed; its detailed page is the gap named here).
func geometryGaps(census []ui.ScreenCensusEntry) []string {
	var out []string
	for _, e := range census {
		if e.GeometryGapNote != "" {
			out = append(out, fmt.Sprintf("%s: %s", e.Name, e.GeometryGapNote))
		}
	}
	return out
}

var listCountRe = regexp.MustCompile(`^(\d+) known screen\(s\):$`)

// listScreens runs "go run ./cmd/screenshot -list" in root and returns its
// per-screen lines (already formatted by that command) plus the count it
// printed on its own header line. It does not re-parse the per-screen
// fields: cmd/screenshot's own -list output is passed through verbatim, so
// a future change to that table's columns needs no matching change here.
func listScreens(root string) ([]string, int, error) {
	cmd := exec.Command("go", "run", "./cmd/screenshot", "-list")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, 0, fmt.Errorf("%v: %s", err, ee.Stderr)
		}
		return nil, 0, err
	}

	var lines []string
	count := -1
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if m := listCountRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			fmt.Sscanf(m[1], "%d", &count)
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}
	if count < 0 {
		return nil, 0, fmt.Errorf("did not find a %q header line in cmd/screenshot -list output", `N known screen(s):`)
	}
	return lines, count, nil
}
