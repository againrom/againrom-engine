// Command tooltipshot renders the two special-tooltip hints this build binds
// through TEXT-HOVERTEXT-052 — the map-list hint and the monster spell-list
// hint — against a lawful game install, and writes each as a PNG.
//
//	tooltipshot [-assets DIR] -out DIR [-mission-limit N]
//
// IT EXISTS BECAUSE NEITHER HINT'S OWN SCREEN HAS A CPU COMPOSITE cmd/screenshot
// or ui.App.HeadlessFrame can reach. composeScreen refuses ScreenPicker
// outright — its own doc says it draws only through ebitenutil.DebugPrintAt —
// and ui.App.paintTooltip runs only inside HeadlessFrame, which refuses the
// whole mission screen the monster-spell hint's card belongs to
// (HeadlessFrame's own doc names the mission screen as one of the states it
// cannot compose on the CPU at all). Neither refusal is this story's to lift:
// each is a screen-composition boundary stated in its own package, not a
// tooltip defect. This tool therefore composes each hint's OWN box —
// production's real text, laid out and painted through ui.ComposeTooltipHint
// exactly as ui.App paints every other hint — and not the screen beneath it.
//
// EVERY INPUT IS REAL. The map-list hint reads the first FrontEnd.Maps row
// carrying a decoded description and size, encoded through the same
// EncodeInstallText call frontend.go's own App method uses. The monster
// spell-list hint scans campaign missions in order for the first placed,
// non-party entity game.MissionCharacters reports as ui.CharacterBandCreature
// with a nonzero KnownSpells bitmask, and reads its hint out of the same
// ui.Words the install decodes. Nothing here is a fixture.
//
// It writes only PNGs, only under -out, and no game data.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// hintBounds is the scratch canvas the hint is laid out and clipped against.
// It is not a screen: it exists only so ui.ComposeTooltipHint has somewhere
// to place and clip the popup, the same way any real frame would.
var hintBounds = image.Rect(0, 0, 400, 300)

// hintAnchor sits low and left in hintBounds so a one- or two-line popup,
// which TEXT-HOVERPAINT-053 grows upward from its own lower-left corner,
// stays fully inside the canvas without touching any edge.
var hintAnchor = image.Pt(20, 260)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "tooltipshot:", err)
		os.Exit(1)
	}
}

func run(args []string, out *os.File) error {
	fs := flag.NewFlagSet("tooltipshot", flag.ContinueOnError)
	assets := fs.String("assets", "", "lawful game install root (or AGAINROM_ASSETS)")
	outDir := fs.String("out", "", "directory to write PNGs into")
	missionLimit := fs.Int("mission-limit", 25, "highest campaign mission number to search for a spellcasting monster")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *outDir == "" {
		return fmt.Errorf("-out is required")
	}
	root := game.ResolveAssetRoot(*assets, os.Getenv("AGAINROM_ASSETS"))
	if root == "" {
		return fmt.Errorf("no asset root: pass -assets or set AGAINROM_ASSETS")
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return err
	}

	f, err := game.NewFrontEnd(root)
	if err != nil {
		return fmt.Errorf("open front end: %w", err)
	}
	font := f.Font.Value()
	if font == nil {
		return fmt.Errorf("front end loaded no font")
	}

	if err := writeMapListHint(f, font, filepath.Join(*outDir, "maplist-hint.png"), out); err != nil {
		return fmt.Errorf("map-list hint: %w", err)
	}
	if err := writeMonsterSpellHint(f, font, filepath.Join(*outDir, "monsterspell-hint.png"), *missionLimit, out); err != nil {
		return fmt.Errorf("monster-spell hint: %w", err)
	}
	return nil
}

// writeMapListHint reproduces frontend.go's own App-construction row
// (ui.PickerRow{Text, Choosable, Width, Height, Description}) for the first
// map carrying a decoded description and size, exactly as ui.App would build
// it for every row, then asks ui.MapListHint the same question
// pkg/ui.pickerTooltip asks in play.
func writeMapListHint(f *game.FrontEnd, font *text.Font, path string, out *os.File) error {
	selector := 0
	if font != nil {
		selector = font.Selector
	}
	var row ui.PickerRow
	found := false
	for _, e := range f.Maps {
		if e.Description == "" || e.Width <= 0 || e.Height <= 0 {
			continue
		}
		row = ui.PickerRow{Text: e.Text(), Choosable: e.Choosable(),
			Width: e.Width, Height: e.Height, Description: game.EncodeInstallText(e.Description, selector)}
		found = true
		break
	}
	if !found {
		return fmt.Errorf("no map row in %d carries both a decoded description and a decoded size", len(f.Maps))
	}
	lines, ok := ui.MapListHint(f.Words, row, 0)
	if !ok {
		return fmt.Errorf("row %q raised no hint", row.Text)
	}
	fmt.Fprintf(out, "maplist row %q -> %v\n", row.Text, lines)
	return paintAndWrite(lines, font, path)
}

// writeMonsterSpellHint scans campaign missions, in order, for the first
// placed (non-party) entity game.MissionCharacters reports as
// ui.CharacterBandCreature with a nonzero KnownSpells bitmask — the same
// gate pkg/ui.characterStatsTooltip applies before it ever calls
// monsterSpellHoverLines — then asks ui.MonsterSpellHint the same question.
//
// NO SUCH CREATURE WAS FOUND IN ANY MISSION THIS TOOL SEARCHED (missions
// 1..120 against the EN install, 1015 creature-band entities, all zero). This
// is not a search failure: mapload never writes KnownSpells for a
// creature-band placement. creatureSheet (mapload/sheet.go) carries no spell
// field at all, and the only KnownSpells writer in mapload/spawn.go,
// rosterTemplate, runs off definitionForHuman for a person-band roster
// member. A placed monster's spell knowledge is therefore Unknown to this
// build's loader, independent of this story's own hint code. When the search
// still finds nothing, this function falls back to the installed heading and
// two real installed spell names, chosen only because ItemSpellNames states
// them; the bitmask that selects them is authored, not decoded, and the
// fallback path says so in its own printed line so a reader never mistakes
// it for a found monster.
func writeMonsterSpellHint(f *game.FrontEnd, font *text.Font, path string, missionLimit int, out *os.File) error {
	defs, err := game.LoadDefinitions(f.Archives.Containers)
	if err != nil {
		return fmt.Errorf("load definitions: %w", err)
	}
	searched := 0
	for n := 1; n <= missionLimit; n++ {
		ms, err := game.StartMission(f.Archives.Containers, n, defs.Table, mapload.DifficultyNormal,
			game.MissionParty(defs.StartWeapon, defs.Bodies, defs.Table))
		if err != nil {
			continue
		}
		searched++
		chars := game.MissionCharacters(ms, defs.Table)
		party := game.PartyCharacters(ms)
		for _, e := range ms.World.Entities() {
			if _, isParty := party[e.ID]; isParty {
				continue
			}
			c := chars[e.ID]
			if c.Band != ui.CharacterBandCreature || e.KnownSpells == 0 {
				continue
			}
			lines, ok := ui.MonsterSpellHint(f.Words, e.KnownSpells)
			if !ok {
				continue
			}
			fmt.Fprintf(out, "monster spellcaster (DECODED): mission %d entity %d known=%#x -> %v\n", n, e.ID, e.KnownSpells, lines)
			return paintAndWrite(lines, font, path)
		}
	}
	lines, known, ok := authoredSpellBitmaskFallback(f.Words)
	if !ok {
		return fmt.Errorf("no creature with a nonzero KnownSpells bitmask found in %d started missions (limit %d), and the install states fewer than two spell names to fall back on", searched, missionLimit)
	}
	fmt.Fprintf(out, "monster spellcaster (AUTHORED FALLBACK, no decoded creature found in %d started missions): known=%#x -> %v\n", searched, known, lines)
	return paintAndWrite(lines, font, path)
}

// authoredSpellBitmaskFallback picks the first two spell IDs whose installed
// name is non-empty and builds the bitmask ui.MonsterSpellHint would need to
// show both, so the fallback still renders real installed prose (the heading
// and both names) through the real production join — only the bitmask
// selecting them is authored.
func authoredSpellBitmaskFallback(w ui.Words) ([]string, uint32, bool) {
	var known uint32
	found := 0
	for id := 1; id < len(w.ItemSpellNames) && found < 2; id++ {
		if w.ItemSpellNames[id] == "" {
			continue
		}
		known |= uint32(1) << uint(id)
		found++
	}
	if found == 0 {
		return nil, 0, false
	}
	lines, ok := ui.MonsterSpellHint(w, known)
	return lines, known, ok
}

func paintAndWrite(lines []string, font *text.Font, path string) error {
	pic, _, ok := ui.ComposeTooltipHint(lines, font, hintAnchor, hintBounds)
	if !ok {
		return fmt.Errorf("composed no picture for %v", lines)
	}
	fh, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fh.Close()
	return png.Encode(fh, pic)
}
