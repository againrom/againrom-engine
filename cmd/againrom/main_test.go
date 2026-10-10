package main

// Tests for the game entry point.
//
// Every fixture is a synthetic install written to a temp dir: three synthetic
// .res archives and synthetic map bytes, all built from the documented format
// contracts. No game file is read, and no window is ever opened — the headless
// mode is exactly the path that proves that.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/game"
	"againrom/pkg/render/menu"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// noEnv is an environment with nothing set.
func noEnv(string) string { return "" }

func TestNormalLaunchIdentityNamesTheCheckout(t *testing.T) {
	want, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	want, err = filepath.Abs(want)
	if err != nil {
		t.Fatalf("Abs(Getwd): %v", err)
	}
	if got := launchSource(); got != want {
		t.Errorf("launchSource() = %q, want current checkout %q", got, want)
	}
	if got := launchStamp(); got == "" || len(strings.TrimSuffix(got, "+dirty")) > 12 {
		t.Errorf("launchStamp() = %q, want a nonempty revision no longer than 12 characters plus optional dirty mark", got)
	}
}

// envWith returns a getenv that answers only for AGAINROM_ASSETS.
func envWith(root string) func(string) string {
	return func(k string) string {
		if k == "AGAINROM_ASSETS" {
			return root
		}
		return ""
	}
}

var hotIndex = [menu.ButtonCount]byte{0x80, 0x90, 0xa0, 0xb0, 0xc0, 0xd0, 0xe0, 0xf0}

// installOptions describes a synthetic install to lay down.
type installOptions struct {
	looseMaps   []string // loose .alm file names
	archiveMaps []string // .alm entry names inside scenario.res
	omitRegions []int    // 1-based buttons whose mask region is left empty
	// menuEdits is keyed by the ADDRESS the front-end resolves the asset by,
	// which is what a case is about; writeInstall turns each key into the entry
	// path inside main.res, because that is what an archive holds.
	menuEdits   map[string][]byte
	omitArchive string // an archive to leave out entirely
	// omitObjectRegistry leaves objects/objects.reg out of graphics.res,
	// keeping the archive itself readable. It is the switch that turns AC-7's
	// failing -check into a CASE: with every install carrying the registry,
	// nothing here could tell a startup that loads the object bundle from one
	// that never asks for it (0017 AC-7).
	omitObjectRegistry bool
	// omitFont leaves the default font's two nodes out of graphics.res. Unlike
	// the two registries below it is NOT a startup failure — the front-end
	// assembles carrying the reason and every map still opens, because the font
	// gates a panel to look at and not a map's content.
	omitFont bool
	// omitChargen leaves one mandatory generator bitmap out while retaining the
	// rest of the synthetic install, so startup failure is attributable to its
	// address rather than to an absent archive.
	omitChargen bool
	// omitFontAdvances leaves the atlas in and the sidecar out, which is the
	// other shape a broken font takes: the pair's two counts cannot agree
	// because one of them is not there.
	omitFontAdvances bool
	// omitCursor leaves the attack cursor's art out of graphics.res. Like the
	// font it is NOT a startup failure -- the map screen falls back to its own
	// authored mark, so the attack mode stays visible -- and it is a switch so
	// that "reports a missing pointer" is a case here rather than the state
	// every fixture is accidentally in (0080 AC-11).
	omitCursor bool
	// omitStartingWeapon leaves the Weapons row character generation names out
	// of the definition table, keeping the table itself readable. Like the font
	// it is NOT a startup failure — the party's hero is then BARE, which the
	// original permits — and it is a switch so that "reports the band" and
	// "reports why there is none" are two cases and not one (0078 AC-15).
	omitStartingWeapon bool
	// withShootWeapon adds a SECOND Weapons row beside the blade's own — plus
	// the shape and material words its own literal names — resolving the
	// shooting skill's literal, "Uncommon Wood Short Bow". Off by default, so
	// every existing fixture in this file keeps resolving one weapon, the
	// blade's own; it exists so -skill's own tests can prove the flag is read
	// BEFORE the install loads, by giving the shoot slot something real to
	// resolve to.
	withShootWeapon bool
	// omitUnitRegistry is its unit-layer sibling: units/units.reg left out of
	// graphics.res, the archive still readable — the second of 0022 AC-10's
	// two install LAYOUTS, and the only way the failing -check is a case rather
	// than an accident (SC-8).
	omitUnitRegistry bool
	// omitStructureRegistry is the third of the same shape:
	// structures/structures.reg left out of graphics.res, the archive still
	// readable, so "an unreadable structure registry fails startup" is a case
	// here rather than something reached by deleting a whole archive.
	omitStructureRegistry bool
	// omitTable leaves data/data.bin out of world.res, keeping the archive
	// itself readable. It is the switch that makes "the table is required"
	// a case rather than something reached only by deleting a whole archive.
	omitTable bool
	// undecodableMaps are .alm entry names written into scenario.res carrying
	// bytes that are not a map.
	undecodableMaps []string
}

// containerEntry turns an address into the entry path inside the archive named by
// archive: the container's identity segment, which every address the front-end
// resolves now carries, stripped off the front.
//
// An install holds containers, so the fixture writes container-relative
// entries; the front-end resolves addresses. The two differ by exactly that
// segment, and this DERIVES it from the archive name the front-end opens
// rather than writing a second prefix literal — a literal could drift
// until the fixture keyed entries under a prefix nothing resolves, and the
// suite would still be green.
func containerEntry(t *testing.T, archive, address string) string {
	t.Helper()
	identity, err := vfs.Identity(archive)
	if err != nil {
		t.Fatalf("vfs.Identity(%q): %v", archive, err)
	}
	rel := strings.TrimPrefix(address, identity+"/")
	if rel == address {
		t.Fatalf("address %q carries no %q identity segment", address, identity)
	}
	return rel
}

// mainEntry and graphicsEntry are containerEntry over the two containers whose
// addresses this fixture writes entries for: the menu art in main.res, and the
// object and unit registries in graphics.res.
//
// world.res needs no such helper: this fixture writes its one entry at the path
// the front-end's own address resolves to, and that path carries no identity
// segment of its own.
func mainEntry(t *testing.T, address string) string {
	t.Helper()
	return containerEntry(t, game.MainArchive, address)
}

func graphicsEntry(t *testing.T, address string) string {
	t.Helper()
	return containerEntry(t, game.GraphicsArchive, address)
}

// writeInstall lays a synthetic asset root and returns its path.
func writeInstall(t *testing.T, o installOptions) string {
	t.Helper()
	dir := t.TempDir()

	write := func(name string, data []byte) {
		if name == o.omitArchive {
			return
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	menuEdits := make(map[string][]byte, len(o.menuEdits)+1)
	// The shipped body list startup's party derivation reads. It is written by
	// default so every install this file builds carries one; a case that needs
	// it absent can still say so through o.menuEdits, whose entries are applied
	// after and so win, exactly like every other switch here.
	menuEdits[mainEntry(t, game.HeroPictureAddress)] = installBodyList()
	menuEdits[mainEntry(t, "main/graphics/chrgen/leftup.bmp")] = installChargenBitmap(160, 238)
	menuEdits[mainEntry(t, "main/text/main.txt")] = installChargenText()
	menuEdits[mainEntry(t, "main/text/npcnames.txt")] = installChargenNames()
	for address, data := range o.menuEdits {
		menuEdits[mainEntry(t, address)] = data
	}
	write(game.MainArchive, synth.MenuArchive(synth.MenuOptions{
		Prefix:      mainEntry(t, menu.EntryPrefix),
		Hover:       menu.HoverRects,
		Pressed:     menu.PressedRects,
		MaskIndex:   hotIndex,
		OmitRegions: o.omitRegions,
		Edits:       menuEdits,
	}))

	// graphics.res holds no tile strips; a tileset with absent slots draws
	// placeholders rather than failing, which is the shipped behaviour.
	//
	// It DOES hold a readable objects/objects.reg and units/units.reg, because
	// startup loads the static-object and unit-art bundles out of this same
	// archive and an unreadable registry is a startup failure (0017 AC-7; 0022
	// AC-10). Each registry declares no classes: a front-end needs the files to
	// read, not art to draw, and empty ones keep this fixture about the menu
	// and the map list. omitObjectRegistry and omitUnitRegistry leave one out
	// while the archive stays readable, which is the only way this file can
	// state either failure as a case instead of reaching it by accident.
	graphics := []synth.File{{Path: "terrain.3d/readme", Data: []byte("x")}}
	graphics = append(graphics, installChargenGraphics(t, o.omitChargen)...)
	if !o.omitObjectRegistry {
		graphics = append(graphics, synth.File{Path: graphicsEntry(t, game.ObjectRegistry), Data: synth.ObjectsReg(nil)})
	}
	// The unit registry declares ONE class and the archive carries ONE hero
	// body sheet, and both exist for a single reason: startup resolves the
	// party's own body out of this archive, and with an empty registry that
	// resolution succeeds at doing nothing, so the line that performs it could
	// be deleted with every test here still green. The class is the one the
	// authored body's name resolves to, and the sheet sits at the address the
	// appearance law composes for it.
	if !o.omitUnitRegistry {
		graphics = append(graphics,
			synth.File{Path: graphicsEntry(t, game.UnitRegistry), Data: installUnitsReg()},
			synth.File{Path: installHeroSheetAddress(), Data: installHeroSheet()})
	}
	// And a readable structures/structures.reg, for the same reason: startup
	// loads the structure-art bundle out of this archive too, and an unreadable
	// registry is a startup failure. It declares no classes, so this fixture
	// stays about the menu and the map list.
	if !o.omitStructureRegistry {
		graphics = append(graphics, synth.File{Path: graphicsEntry(t, game.StructureRegistry), Data: synth.StructuresReg()})
	}
	// The default font's two nodes. A COMPLETE install has them, so the
	// summary a complete install prints stays the one it printed before this
	// story; the two switches above are what let a broken font be a case here
	// rather than the accidental state of every fixture.
	if !o.omitFont {
		atlas, advances := synth.Font16(installFontGlyphs())
		graphics = append(graphics, synth.File{Path: graphicsEntry(t, game.FontAtlasPath(game.DefaultFont)), Data: atlas})
		if !o.omitFontAdvances {
			graphics = append(graphics, synth.File{Path: graphicsEntry(t, game.FontAdvancePath(game.DefaultFont)), Data: advances})
		}
	}
	// The attack cursor's art, for the font's own reason: a COMPLETE install has
	// it, so the summary a complete install prints stays the one it printed
	// before this story, and omitCursor is what lets a missing one be a case
	// here rather than the accidental state of every fixture (0080 AC-11).
	//
	// One 1x1 painted cell is enough. Nothing in the check mode looks at the
	// picture — it reports only whether there was one — so a fixture carrying a
	// drawn pointer would be asserting the loader's own tests over again.
	if !o.omitCursor {
		graphics = append(graphics, synth.File{
			Path: graphicsEntry(t, game.AttackCursorPath),
			Data: installCursorSheet(),
		})
	}
	write(game.GraphicsArchive, synth.Archive(graphics))

	entries := make([]synth.File, 0, len(o.archiveMaps))
	for i, name := range o.archiveMaps {
		entries = append(entries, synth.File{
			Path: name,
			Data: synth.ALM(synth.ALMOptions{Width: 4 + i, Height: 4, Name: ""}),
		})
	}
	for _, name := range o.undecodableMaps {
		entries = append(entries, synth.File{Path: name, Data: []byte("this is not a map")})
	}
	if len(entries) == 0 {
		entries = append(entries, synth.File{Path: "scenario.reg", Data: []byte("not a map")})
	}
	// scenario.res also holds npc.reg, which the table load requires: it is the
	// first rung of the humans band, so an install without it resolves fifteen
	// shipped placements to nobody. It declares no sections, for the reason the
	// registries above declare no classes — this fixture is about the menu and
	// the map list.
	entries = append(entries, synth.File{Path: "npc.reg", Data: synth.NPCReg(nil)})
	write(game.ScenarioArchive, synth.Archive(entries))

	// world.res holds the placeable-definition table, which startup requires:
	// what lies behind it is the movement domain and the per-class health of
	// every placed unit, so an install without it is an install the application
	// refuses. The table declares no rows — a front-end needs the file to read,
	// not content to resolve — which keeps this fixture about the menu and the
	// map list.
	//
	// It DOES carry the three collections the starting weapon resolves through,
	// because a table with no `Short Sword` row leaves the party's hero bare and
	// an install a player could use does not.
	world := []synth.File{}
	if !o.omitTable {
		world = append(world, synth.File{Path: "data/data.bin", Data: definitionTable(o)})
	}
	write(game.WorldArchive, synth.Archive(world))

	// movies.res carries the shop merchant's own static picture
	// (SHOP-MERCHANT-046) and is required for the same reason world.res is:
	// an install missing it should say so before the user opens a shop, not
	// after (1009). This fixture carries no shop art — this file is about
	// the menu and the map list — so movies.res is written empty and
	// readable.
	write(game.MoviesArchive, synth.Archive(nil))

	for i, name := range o.looseMaps {
		data := synth.ALM(synth.ALMOptions{Width: 5 + i, Height: 5, Name: "Loose " + name})
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

func installChargenBitmap(size ...int) []byte {
	w, h := 2, 2
	if len(size) == 2 {
		w, h = size[0], size[1]
	}
	return synth.BMP24(image.NewRGBA(image.Rect(0, 0, w, h)))
}

func installChargenMask(w, h int, hot ...uint8) []byte {
	palette := make(color.Palette, 256)
	for i := range palette {
		palette[i] = color.RGBA{R: uint8(i), G: uint8(i), B: uint8(i), A: 255}
	}
	mask := image.NewPaletted(image.Rect(0, 0, w, h), palette)
	for i, index := range hot {
		mask.Pix[i] = index
	}
	return synth.BMP8(mask)
}

// installChargenText is a synthetic main.txt: one non-empty line per slot up to
// the last one the generator requires, each terminated by CRLF.
//
// CRLF AND NOT NUL SINCE 0168. The decoded loader ends a line at a CR and skips
// the byte after it (TEXT-STRTAB-023); the NUL separator this fixture used
// before was readable only by the splitter pkg/game used to carry, and that
// splitter is gone.
func installChargenText() []byte {
	const lastRequiredSlot = 260
	out := make([]byte, 0, (lastRequiredSlot+1)*3)
	for slot := 0; slot <= lastRequiredSlot; slot++ {
		out = append(out, 'x', '\r', '\n')
	}
	return out
}

// installChargenNames is a synthetic npcnames.txt that reaches entry 23, the
// last of the four hero names the pre-create page reads (entries 20..23).
func installChargenNames() []byte {
	const lastHeroEntry = 23
	out := make([]byte, 0, (lastHeroEntry+1)*3)
	for entry := 0; entry <= lastHeroEntry; entry++ {
		out = append(out, 'x', '\r', '\n')
	}
	return out
}

func installChargenGraphics(t *testing.T, omitBackground bool) []synth.File {
	t.Helper()
	files := make([]synth.File, 0, 4*3+2*5*3+4)
	add := func(address string, data []byte) {
		files = append(files, synth.File{Path: graphicsEntry(t, address), Data: data})
	}
	if !omitBackground {
		add("graphics/interface/chrgen/precreate/mainarea.bmp", installChargenBitmap(640, 480))
	}
	add("graphics/interface/chrgen/precreate/mask.bmp", installChargenMask(640, 480, 20, 40, 60, 80, 140, 100, 120, 160, 180))
	add("graphics/interface/chrgen/precreate/amulet.bmp", installChargenBitmap(112, 204))
	add("graphics/interface/chrgen/precreate/buttonok.bmp", installChargenBitmap(100, 56))
	for level, size := range [3][2]int{{60, 74}, {76, 112}, {100, 152}} {
		for _, suffix := range []string{"on", "l", "lon"} {
			add(fmt.Sprintf("graphics/interface/chrgen/precreate/levels/level%d%s.bmp", level, suffix), installChargenBitmap(size[0], size[1]))
		}
	}
	// The command panel is the shop's four-button composition (DIV-2868):
	// its body and the pressed-only plaque of each command.
	add("graphics/interface/shopmenu.bmp", installChargenBitmap(176, 238))
	for i, size := range [4][2]int{{120, 52}, {140, 46}, {140, 46}, {120, 52}} {
		add(fmt.Sprintf("graphics/interface/shopbutton%d.bmp", i+1), installChargenBitmap(size[0], size[1]))
	}
	add("graphics/interface/chrgen/rollstatsr.bmp", installChargenBitmap(16, 238))
	add("graphics/interface/humanbackr.bmp", installChargenBitmap(160, 242))
	add("graphics/interface/humanbackl.bmp", installChargenBitmap(16, 242))
	// fullstatsl.bmp/fullstatsr.bmp are the detailed page's own compact
	// character card background and seam (1022 spec B3), a distinct pair
	// under chrgen/ from the pane body/seam above.
	add("graphics/interface/chrgen/fullstatsl.bmp", installChargenBitmap(160, 242))
	add("graphics/interface/chrgen/fullstatsr.bmp", installChargenBitmap(16, 242))
	for _, hero := range []string{"mf", "mm", "ff", "fm"} {
		for _, state := range []string{"on", "l", "lon"} {
			add("graphics/interface/chrgen/precreate/heroes/"+hero+state+".bmp", installChargenBitmap())
		}
	}
	for _, entry := range []struct {
		class  string
		skills []string
		mask   []uint8
	}{
		{"fighter", []string{"sword", "axe", "mace", "pike", "bow"}, []uint8{255, 191, 152, 127, 102}},
		{"mag", []string{"fire", "water", "air", "earth", "astral"}, []uint8{127, 102, 255, 152, 191}},
	} {
		class, skills := entry.class, entry.skills
		add("graphics/interface/chrgen/"+class+"/column.bmp", installChargenBitmap(320, 480))
		add("graphics/interface/chrgen/"+class+"/mask.bmp", installChargenMask(320, 480, entry.mask...))
		for _, skill := range skills {
			for _, state := range []string{"on", "shine_off", "shine_on"} {
				add("graphics/interface/chrgen/"+class+"/"+skill+"/"+state+".bmp", installChargenBitmap())
			}
		}
	}
	for _, name := range []string{"mnloff", "mloff", "mlon", "mnlon", "mdisable", "pnloff", "ploff", "plon", "pnlon", "pdisable"} {
		add("graphics/interface/chrgen/buttons/"+name+".bmp", installChargenBitmap(20, 20))
	}
	glyphs := []synth.Font16Glyph{{Width: 1, Height: 1, Advance: 1}}
	atlas, advances := synth.Font16(glyphs)
	add(game.FontAtlasPath("font2"), atlas)
	add(game.FontAdvancePath("font2"), advances)
	add(game.FontAtlasPathA(game.DocumentFont), installNameFontSheet())
	add(game.FontAdvancePath(game.DocumentFont), advances)
	return files
}

// defaultInstall is a well-formed install with three loose and two archived maps.
func defaultInstall(t *testing.T) string {
	t.Helper()
	return writeInstall(t, installOptions{
		looseMaps:   []string{"Beast.ALM", "Kids.alm", "Waters.alm"},
		archiveMaps: []string{"10.alm", "20.alm"},
	})
}

func TestCheck(t *testing.T) {
	t.Run("a complete install reports both counts and exits 0", func(t *testing.T) {
		dir := defaultInstall(t)
		var out, errOut bytes.Buffer

		code := run([]string{"-assets", dir, "-check", "-picker"}, noEnv, &out, &errOut)

		if code != 0 {
			t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut.String())
		}
		if errOut.Len() != 0 {
			t.Errorf("stderr = %q, want empty", errOut.String())
		}
		// The summary is the first line; the detected base is named on the one
		// after it, and nothing else follows on this route.
		lines := strings.Split(strings.TrimSpace(out.String()), "\n")
		if len(lines) != 2 || !strings.HasPrefix(lines[1], "againrom: base rom1-en ") {
			t.Fatalf("output = %q, want the summary then one base line", out.String())
		}
		line := lines[0]
		// 3 loose + 2 archived = 5 map rows, plus one mission row per archived map
		// — both "10.alm" and "20.alm" parse as positive mission numbers — for
		// 7. All eight buttons have a region, and the party's hero is armed out of
		// the definition table: 23 and 40 through a shape factor of 0.2 give (5,
		// 3), and the generated hero's own blade skill adds 2 to the base. At Body
		// 43 the stat term is 3, so the band is 10-16. The to-hit is 44 rather
		// than the install's own 49 because this synthetic row carries no to-hit
		// column.
		//
		// THE SYNTHETIC INSTALL CARRIES NO CURSOR ART, so the line names the
		// cursor registry as unresolved. That is the reporting rule working rather
		// than a hole in the fixture: the registry is not fatal, every screen
		// falls back to the operating system's arrow, and a headless run is the
		// one place it could ever be said. Writing 28 synthetic sprite sheets to
		// silence one diagnostic line would put the fixture further from the
		// shipped shape, not closer.
		want := "againrom: 7 map rows, 8 of 8 buttons have a mask region; " +
			"no cursor registry: cursor slot \"default\": " +
			"readfile graphics/cursors/default/sprites.16a: file does not exist; " +
			"hero Body 43, Reaction 26, Mind 15, Spirit 15, Blade 10, " +
			"Iron Short Sword 10-16, to-hit 44, defence 8"
		if line != want {
			t.Errorf("summary = %q, want %q", line, want)
		}
	})

	// AND THE OTHER SHAPE: a table that will not yield the weapon leaves the
	// hero BARE, which the original permits and which must not be silent — the
	// same line says so instead of stating a band.
	t.Run("a table with no starting weapon reports a bare hero and exits 0", func(t *testing.T) {
		var out, errOut bytes.Buffer
		dir := writeInstall(t, installOptions{omitStartingWeapon: true})
		if code := run([]string{"-assets", dir, "-check", "-picker"}, noEnv, &out, &errOut); code != 0 {
			t.Fatalf("exit %d, want 0; stderr %q", code, errOut.String())
		}
		line := strings.TrimSpace(out.String())
		// A bare hero still states his CHARACTER — the spread is his whether or
		// not anything resolved for him to hold — and then says why he holds
		// nothing, naming the row it went looking for.
		if !strings.Contains(line, ", bare:") || !strings.Contains(line, "Short Sword") ||
			!strings.Contains(line, "hero Body 43, Reaction 26, Mind 15, Spirit 15, Blade 10") {
			t.Errorf("summary = %q, want the character, a bare hero and the row it looked for", line)
		}
	})

	t.Run("the asset root comes from the flag, the environment, or the game folder", func(t *testing.T) {
		dir := defaultInstall(t)

		t.Run("flag", func(t *testing.T) {
			var out, errOut bytes.Buffer
			if code := run([]string{"-assets", dir, "-check", "-picker"}, noEnv, &out, &errOut); code != 0 {
				t.Fatalf("exit = %d (stderr: %s)", code, errOut.String())
			}
			if !strings.Contains(out.String(), "7 map rows") {
				t.Errorf("summary = %q", out.String())
			}
		})

		t.Run("environment", func(t *testing.T) {
			var out, errOut bytes.Buffer
			if code := run([]string{"-check", "-picker"}, envWith(dir), &out, &errOut); code != 0 {
				t.Fatalf("exit = %d (stderr: %s)", code, errOut.String())
			}
			if !strings.Contains(out.String(), "7 map rows") {
				t.Errorf("summary = %q", out.String())
			}
		})

		t.Run("the flag wins when both are set", func(t *testing.T) {
			// The environment points at a directory that is not an install at
			// all, so if it were consulted the run would fail.
			var out, errOut bytes.Buffer
			bogus := t.TempDir()
			if code := run([]string{"-assets", dir, "-check", "-picker"}, envWith(bogus), &out, &errOut); code != 0 {
				t.Fatalf("exit = %d, want the flag to win (stderr: %s)", code, errOut.String())
			}
		})

		t.Run("the game folder is used when neither is set", func(t *testing.T) {
			t.Chdir(dir)
			var out, errOut bytes.Buffer
			if code := run([]string{"-check", "-picker"}, noEnv, &out, &errOut); code != 0 {
				t.Fatalf("exit = %d, want the install the process is in to be used (stderr: %s)", code, errOut.String())
			}
			if !strings.Contains(out.String(), "7 map rows") {
				t.Errorf("summary = %q", out.String())
			}
			if !strings.Contains(errOut.String(), "using the install found at "+dir) {
				t.Errorf("stderr = %q, want the discovered root named", errOut.String())
			}
		})

		// The fallback is LAST. A configured root that is not an install must
		// still fail, or a mistyped -assets on a machine whose working directory
		// happens to be an install would silently run the wrong game.
		t.Run("a configured root still loses to nothing", func(t *testing.T) {
			t.Chdir(dir)
			bogus := t.TempDir()
			var out, errOut bytes.Buffer
			if code := run([]string{"-assets", bogus, "-check"}, noEnv, &out, &errOut); code == 0 {
				t.Errorf("exit = 0, want the named root to be used and to fail (stderr: %s)", errOut.String())
			}
			if strings.Contains(errOut.String(), "using the install found at") {
				t.Errorf("stderr = %q, want no discovery when -assets was given", errOut.String())
			}
		})
	})

	t.Run("a button with no mask region is reported, not rejected", func(t *testing.T) {
		// Reporting this is precisely what the mode is for, so it must still
		// exit successfully.
		dir := writeInstall(t, installOptions{
			looseMaps:   []string{"a.alm"},
			omitRegions: []int{3},
		})
		var out, errOut bytes.Buffer

		code := run([]string{"-assets", dir, "-check", "-picker"}, noEnv, &out, &errOut)

		if code != 0 {
			t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut.String())
		}
		if !strings.Contains(out.String(), "7 of 8 buttons") {
			t.Errorf("summary = %q, want it to report 7 of 8", strings.TrimSpace(out.String()))
		}
	})

	t.Run("no asset root is reported and exits non-zero", func(t *testing.T) {
		var out, errOut bytes.Buffer

		code := run([]string{"-check"}, noEnv, &out, &errOut)

		if code == 0 {
			t.Fatalf("exit = 0 with no asset root configured")
		}
		if !strings.Contains(errOut.String(), "no asset root") {
			t.Errorf("stderr = %q, want it to say no asset root is configured", errOut.String())
		}
		if out.Len() != 0 {
			t.Errorf("stdout = %q, want nothing printed", out.String())
		}
	})

	t.Run("an unknown flag is reported and exits non-zero", func(t *testing.T) {
		var out, errOut bytes.Buffer
		if code := run([]string{"-nosuchflag"}, noEnv, &out, &errOut); code == 0 {
			t.Errorf("exit = 0 on an undefined flag")
		}
		if !strings.Contains(errOut.String(), usage) {
			t.Errorf("stderr = %q, want the usage line", errOut.String())
		}
	})
}

func TestProductionHeadlessFlagParsesAndExcludesCheckMode(t *testing.T) {
	o, err := parse([]string{"--headless", `scenarios\0152-save666.json`})
	if err != nil {
		t.Fatal(err)
	}
	if o.headless != `scenarios\0152-save666.json` || o.check {
		t.Fatalf("parsed options = headless %q check %v", o.headless, o.check)
	}
	if _, err := parse([]string{"-check", "--headless", "scenario.json"}); err == nil ||
		!strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("check plus headless = %v, want mutual-exclusion error", err)
	}
}

// TestEveryShippedScenarioParsesAndValidates is 0155 SC-1.
//
// The scenarios directory is the documented input surface of this command, and
// a file there that no longer validates is a broken example whichever install it
// was written for. Reading them needs no game data, so this runs everywhere the
// suite runs.
func TestEveryShippedScenarioParsesAndValidates(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "scenarios", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 2 {
		t.Fatalf("found %d shipped scenario(s); the directory holds one per stage at least", len(files))
	}
	stages := map[string]int{}
	ran, skipped := 0, 0
	for _, path := range files {
		name := filepath.Base(path)
		s, err := game.ReadHeadlessScenario(path)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		stages[s.StageName()]++
		if s.AssetSource() != game.AssetsSynthetic {
			// AN INSTALL-TIER SCENARIO IS SKIPPED HERE AND COUNTED, never
			// run: `go test` is green with no game present (golden rule 2).
			// The count is logged because a tier that cannot run looks
			// exactly like a tier that passed.
			skipped++
			t.Logf("%s: skipped, needs a lawful install (assets %q)", name, s.AssetSource())
			continue
		}
		play, err := s.World.Build()
		if err != nil {
			t.Errorf("%s: build world: %v", name, err)
			continue
		}
		if err := game.RunPlayScenario(play, s, io.Discard, io.Discard); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		ran++
	}
	t.Logf("shipped scenarios: %d ran on synthetic assets, %d skipped for want of an install", ran, skipped)
	if ran == 0 {
		t.Error("no shipped scenario runs without an install; the synthetic tier has no worked example")
	}
	if skipped == 0 {
		t.Error("no shipped scenario needs an install; the install tier has no worked example")
	}
	if stages[game.StageFrontEnd] == 0 || stages[game.StageMission] == 0 {
		t.Errorf("shipped scenarios cover stages %v; both stages need a worked example", stages)
	}
}

func TestAMissionScenarioTakesTheMissionRouteAndNotTheFrontEnd(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	mission := write("mission.json", `{"version":2,"stage":"mission","mission":10,
		"steps":[{"command":"report","name":"placed"}]}`)
	front := write("front.json", `{"version":1,"steps":[{"command":"capture","name":"start"}]}`)

	for _, tc := range []struct {
		name, path string
		want       bool
	}{
		{"mission stage", mission, true},
		{"front-end stage", front, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if code := run([]string{"-assets", dir, "--headless", tc.path}, noEnv, &out, &errOut); code == 0 {
				t.Fatalf("exit = 0 over an asset root holding no archives")
			}
			if got := strings.Contains(errOut.String(), "againrom: headless:"); got != tc.want {
				t.Fatalf("stderr = %q; headless prefix present = %v, want %v", errOut.String(), got, tc.want)
			}
		})
	}
}

// TestMarkerFlag covers the one flag this command owns beyond the asset root
// and the headless mode: the diagnostic placement markers.
func TestMarkerFlag(t *testing.T) {
	t.Run("the parsed default is off", func(t *testing.T) {
		o, err := parse(nil)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if o.markers {
			t.Errorf("markers = true with no flag given, want the shipped default off")
		}
	})

	t.Run("the on form parses", func(t *testing.T) {
		// A boolean flag with no value is true, so the instrument comes back
		// with a bare -markers; -markers=true is the explicit form.
		for _, arg := range []string{"-markers", "-markers=true"} {
			o, err := parse([]string{arg})
			if err != nil {
				t.Fatalf("parse %s: %v", arg, err)
			}
			if !o.markers {
				t.Errorf("markers = false after %s", arg)
			}
		}
	})

	t.Run("the off form still parses", func(t *testing.T) {
		// It is now the default, but the spelling has to keep working: it is
		// what the usage line and the doc comment showed for nine days and what
		// any script written in that window says.
		o, err := parse([]string{"-markers=false"})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if o.markers {
			t.Errorf("markers = true after -markers=false")
		}
	})

	t.Run("the value reaches the front-end, both ways", func(t *testing.T) {
		dir := defaultInstall(t)
		for _, tc := range []struct {
			label string
			on    bool
			want  game.Markers
		}{
			// Three glyphs off one flag: the static-object cross joined the other two
			// on the game's one question rather than gaining a switch of its own.
			{"on", true, game.Markers{Objects: true, Units: true, Statics: true}},
			{"off", false, game.Markers{}},
		} {
			t.Run(tc.label, func(t *testing.T) {
				front, err := frontEnd(dir, options{assets: dir, markers: tc.on}, game.OptionsStore{})
				if err != nil {
					t.Fatalf("frontEnd: %v", err)
				}
				if front.Markers != tc.want {
					t.Errorf("front.Markers = %+v, want %+v", front.Markers, tc.want)
				}
			})
		}
	})

	t.Run("the front-end's own default is on", func(t *testing.T) {
		// The default lives in NewFrontEnd, so a front-end built without going
		// through this command's flags shows the markers too.
		front, err := game.NewFrontEnd(defaultInstall(t))
		if err != nil {
			t.Fatalf("NewFrontEnd: %v", err)
		}
		if want := (game.Markers{Objects: true, Units: true, Statics: true}); front.Markers != want {
			t.Errorf("NewFrontEnd markers = %+v, want %+v", front.Markers, want)
		}
	})

	t.Run("markers change nothing about the headless summary", func(t *testing.T) {
		// -check reports the install, not the diagnostics: AC-6 pins that line's
		// two counts and nothing may be appended to it here.
		dir := defaultInstall(t)
		var on, off bytes.Buffer
		var errOut bytes.Buffer
		if code := run([]string{"-assets", dir, "-check", "-picker"}, noEnv, &on, &errOut); code != 0 {
			t.Fatalf("exit = %d (stderr: %s)", code, errOut.String())
		}
		if code := run([]string{"-assets", dir, "-check", "-picker", "-markers=false"}, noEnv, &off, &errOut); code != 0 {
			t.Fatalf("exit = %d (stderr: %s)", code, errOut.String())
		}
		if on.String() != off.String() {
			t.Errorf("summary differs with the flag:\n on:  %q\n off: %q", on.String(), off.String())
		}
	})
}

func TestSoundFlags(t *testing.T) {
	t.Run("the parsed defaults are on, at full volume", func(t *testing.T) {
		o, err := parse(nil)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if !o.sound {
			t.Errorf("sound = false with no flag given, want the shipped default on")
		}
		if o.volume != defaultVolume {
			t.Errorf("volume = %d with no flag given, want the shipped default %d", o.volume, defaultVolume)
		}
	})

	t.Run("both flags parse explicit values", func(t *testing.T) {
		o, err := parse([]string{"-sound=false", "-volume", "40"})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if o.sound {
			t.Errorf("sound = true after -sound=false")
		}
		if o.volume != 40 {
			t.Errorf("volume = %d after -volume 40", o.volume)
		}
	})

	t.Run("the values reach the front-end", func(t *testing.T) {
		dir := defaultInstall(t)
		front, err := frontEnd(dir, options{assets: dir, sound: false, volume: 17, soundSet: true, volumeSet: true}, game.OptionsStore{})
		if err != nil {
			t.Fatalf("frontEnd: %v", err)
		}
		if want := (game.SoundOptions{Enabled: false, Volume: 17}); front.Sound != want {
			t.Errorf("front.Sound = %+v, want %+v", front.Sound, want)
		}
	})
}

// skillInstall is defaultInstall's own shape plus the shooting slot's own
// bow row, so a case can prove the flag reaches a REAL weapon rather than
// merely a package variable.
func skillInstall(t *testing.T) string {
	t.Helper()
	return writeInstall(t, installOptions{
		looseMaps:       []string{"Beast.ALM"},
		withShootWeapon: true,
	})
}

// TestSkillFlag covers -skill end to end: parse maps a name to its own slot
// and refuses an unknown one, and the resolved slot reaches the front end
// before the install loads — proven by the fact that it changes which
// weapon actually resolves, not merely by reading back a package variable.
func TestSkillFlag(t *testing.T) {
	t.Run("the parsed default leaves the trained skill alone", func(t *testing.T) {
		o, err := parse(nil)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if o.skill != skillNone {
			t.Errorf("skill = %d with no flag given, want %d (leave it alone)", o.skill, skillNone)
		}
	})

	t.Run("a known name maps to its own slot, case-insensitively", func(t *testing.T) {
		for _, name := range []string{"shooting", "Shooting", "SHOOTING"} {
			o, err := parse([]string{"-skill", name})
			if err != nil {
				t.Fatalf("parse -skill %s: %v", name, err)
			}
			if o.skill != data.SkillShoot {
				t.Errorf("-skill %s parsed to slot %d, want %d", name, o.skill, data.SkillShoot)
			}
		}
	})

	t.Run("an unknown name is an error, not a panic", func(t *testing.T) {
		o, err := parse([]string{"-skill", "wizardry"})
		if err == nil {
			t.Fatalf("parse -skill wizardry = %+v, want an error", o)
		}
		if !strings.Contains(err.Error(), "wizardry") {
			t.Errorf("error %v does not name the bad value", err)
		}
	})

	t.Run("the value reaches the front end before NewFrontEnd loads it", func(t *testing.T) {
		// game.partySkillSlot is PACKAGE STATE; a leaked value here would
		// silently retrain the hero every later test in this file (and in
		// pkg/game's own suite, if the process runs both) generates.
		before := game.PartySkillSlot()
		t.Cleanup(func() { _ = game.SetPartySkill(before) })

		dir := skillInstall(t)
		front, err := frontEnd(dir, options{assets: dir, skill: data.SkillShoot}, game.OptionsStore{})
		if err != nil {
			t.Fatalf("frontEnd: %v", err)
		}
		if game.PartySkillSlot() != data.SkillShoot {
			t.Errorf("PartySkillSlot() = %d after frontEnd, want %d", game.PartySkillSlot(), data.SkillShoot)
		}
		// THE ORDERING CLAIM ITSELF: skillInstall's table carries the
		// ordinary blade sword AND the shoot slot's own bow; if frontEnd
		// applied the skill AFTER NewFrontEnd had already resolved the
		// starting weapon, StartWeapon would still be the blade's sword.
		// This is armed with the bow instead, which only holds if the
		// setting reached LoadDefinitions before it ran.
		wantName, ok := game.StartingWeaponName(false, data.SkillShoot)
		if !ok {
			t.Fatalf("fixture: the shoot slot trains no weapon by StartingWeaponName's own account")
		}
		if front.StartWeapon.Value() == nil || front.StartWeapon.Value().Name != wantName {
			t.Errorf("StartWeapon = %+v, want %q — the flag applied before the load, not after",
				front.StartWeapon, wantName)
		}
	})

	t.Run("an unrecognised name is reported by the command, not a panic", func(t *testing.T) {
		dir := skillInstall(t)
		var out, errOut bytes.Buffer
		code := run([]string{"-assets", dir, "-check", "-skill", "wizardry"}, noEnv, &out, &errOut)
		if code == 0 {
			t.Fatalf("exit 0 with an unknown -skill value")
		}
		if out.Len() != 0 {
			t.Errorf("stdout = %q, want nothing printed", out.String())
		}
		if !strings.Contains(errOut.String(), "wizardry") {
			t.Errorf("stderr = %q, want it to name the bad value", errOut.String())
		}
	})

	t.Run("without the flag the check line is unchanged", func(t *testing.T) {
		// -skill is off by default in every other case in this file; this
		// pins that skillInstall's EXTRA bow row does not itself change the
		// ordinary blade-armed summary when the flag is not given.
		dir := skillInstall(t)
		var out, errOut bytes.Buffer
		if code := run([]string{"-assets", dir, "-check", "-picker"}, noEnv, &out, &errOut); code != 0 {
			t.Fatalf("exit %d; stderr = %q", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Iron Short Sword") {
			t.Errorf("summary = %q, want the blade's own sword with no flag given", out.String())
		}
	})
}

// TestStaticBundle covers the static-object bundle startup loads: it is loaded
// once and kept, and an install whose graphics.res holds no readable
// objects/objects.reg fails the headless check (0017 AC-7, SC-8).
//
// Both halves need the fixture switch to say anything. Every other install this
// file lays down carries the registry, so until one could lack it, deleting the
// load from NewFrontEnd — or swallowing its error — left the whole tree green:
// the failing -check was reachable by accident and by nothing else.
//
// It is asserted in the HEADLESS mode alone, unlike the failures below, and the
// reason is the regression it exists to catch. With the guard gone this install
// starts cleanly, so a windowed row would reach App().Run() and open a window
// rather than report anything — a test that hangs instead of failing is not a
// test of the thing.
func TestStaticBundle(t *testing.T) {
	t.Run("a complete install loads the bundle and keeps it", func(t *testing.T) {
		// Loaded is not enough: a front-end that decodes the classes and drops
		// them draws bare ground with every -check still passing.
		front, err := frontEnd(defaultInstall(t), options{markers: defaultMarkers}, game.OptionsStore{})
		if err != nil {
			t.Fatalf("frontEnd: %v", err)
		}
		if front.Statics == nil {
			t.Errorf("front.Statics is nil; startup carries no object bundle to the maps it opens")
		}
	})

	t.Run("an unreadable objects/objects.reg fails -check", func(t *testing.T) {
		// The archive itself opens; only the entry is gone. An install missing a
		// piece the game needs is reported before a window could open, exactly
		// like a missing archive.
		dir := writeInstall(t, installOptions{
			looseMaps:          []string{"a.alm"},
			omitObjectRegistry: true,
		})
		var out, errOut bytes.Buffer

		code := run([]string{"-assets", dir, "-check", "-picker"}, noEnv, &out, &errOut)

		if code == 0 {
			t.Fatalf("exit = 0 on an install whose graphics.res holds no %s", game.ObjectRegistry)
		}
		if out.Len() != 0 {
			t.Errorf("stdout = %q, want nothing printed", out.String())
		}
		if !strings.Contains(errOut.String(), game.ObjectRegistry) {
			t.Errorf("stderr = %q, want it to name %q", errOut.String(), game.ObjectRegistry)
		}
	})
}

// TestUnitBundle covers the unit-art bundle startup loads —
// TestStaticBundle's sibling, over 0022 AC-10's two install layouts (SC-8):
// the complete install loads the bundle once in NewFrontEnd and keeps it,
// and an install whose graphics.res holds no readable units/units.reg fails
// the headless check, naming the registry.
//
// Both halves need the omitUnitRegistry switch to say anything, for exactly
// the reason TestStaticBundle's doc gives: with every install carrying the
// registry, deleting the load from NewFrontEnd — or swallowing its error —
// would leave the whole tree green. And the failing half is asserted in the
// HEADLESS mode alone for that test's reason too: with the guard gone this
// install starts cleanly, so a windowed run would open a window rather than
// report anything.
//
// -check builds no texture by construction: the whole run is the headless
// path, which has no graphics context to build one in — the same witness the
// bundle loader's own tests state over synthetic archives.
func TestUnitBundle(t *testing.T) {
	t.Run("a complete install loads the bundle and keeps it", func(t *testing.T) {
		// Loaded is not enough: a front-end that decodes the classes and drops
		// them draws every unit as the square with every -check still passing.
		front, err := frontEnd(defaultInstall(t), options{markers: defaultMarkers}, game.OptionsStore{})
		if err != nil {
			t.Fatalf("frontEnd: %v", err)
		}
		if front.Units == nil {
			t.Fatal("front.Units is nil; startup carries no unit bundle to the maps it opens")
		}
		member := game.MissionParty(front.StartWeapon.Value(), front.Bodies, front.Table)[0]
		key := data.HeroBodyKey(member.BodyDir, data.HeroBody(member.Body))
		if front.Units.Bodies[key] == nil {
			t.Errorf("startup resolved no body for %q under %q; the install carries the class and the sheet at %q",
				member.Body, member.BodyDir, installHeroSheetAddress())
		}
	})

	t.Run("an unreadable units/units.reg fails -check", func(t *testing.T) {
		// The archive itself opens; only the entry is gone. An install missing
		// a piece the game needs is reported before a window could open,
		// exactly like a missing archive or object registry.
		dir := writeInstall(t, installOptions{
			looseMaps:        []string{"a.alm"},
			omitUnitRegistry: true,
		})
		var out, errOut bytes.Buffer

		code := run([]string{"-assets", dir, "-check", "-picker"}, noEnv, &out, &errOut)

		if code == 0 {
			t.Fatalf("exit = 0 on an install whose graphics.res holds no %s", game.UnitRegistry)
		}
		if out.Len() != 0 {
			t.Errorf("stdout = %q, want nothing printed", out.String())
		}
		if !strings.Contains(errOut.String(), game.UnitRegistry) {
			t.Errorf("stderr = %q, want it to name %q", errOut.String(), game.UnitRegistry)
		}
	})
}

// TestStartupFailures covers the incomplete and inconsistent asset sets, in both
// modes: each is reported on the error stream, exits non-zero, and prints nothing
// on the output stream.
func TestStartupFailures(t *testing.T) {
	// One case per failure kind the contract lists, and no more. The archive
	// cases beyond this one are already covered where the archives are opened;
	// graphics.res is the one worth repeating here, because it is the archive the
	// menu never reads and so the one whose necessity is not self-evident.
	cases := []struct {
		label string
		opts  installOptions
		names string // a substring the message must name
	}{
		{"a required archive cannot be opened", installOptions{omitArchive: game.GraphicsArchive}, game.GraphicsArchive},
		{
			// The archive opens and the table is not in it. Its own case
			// because "no table" has to be a refusal rather than a world in
			// which every unit is a ground mover at one health constant — which
			// is exactly what the application built before it read the table,
			// and what it must never silently fall back to.
			"the definition table is absent from an archive that opens",
			installOptions{omitTable: true},
			"data.bin",
		},
		{
			"a menu asset is absent",
			installOptions{menuEdits: map[string][]byte{menu.EntryPrefix + menu.BaseEntry: nil}},
			menu.BaseEntry,
		},
		{
			"the base screen is not 640x480",
			installOptions{menuEdits: map[string][]byte{
				menu.EntryPrefix + menu.BaseEntry: synth.BMP24(image.NewRGBA(image.Rect(0, 0, 320, 240))),
			}},
			menu.BaseEntry,
		},
		{
			"an overlay disagrees with its own placement row",
			installOptions{menuEdits: map[string][]byte{
				menu.EntryPrefix + menu.HoverEntry(2): synth.BMP24(image.NewRGBA(image.Rect(0, 0, 7, 9))),
			}},
			menu.HoverEntry(2),
		},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			opts := tc.opts
			opts.looseMaps = []string{"a.alm"}
			dir := writeInstall(t, opts)

			// Both modes must agree: the windowed run fails at exactly the same
			// point, before a window could open.
			for _, mode := range []struct {
				name string
				args []string
			}{
				{"check", []string{"-assets", dir, "-check"}},
				{"windowed", []string{"-assets", dir}},
			} {
				t.Run(mode.name, func(t *testing.T) {
					var out, errOut bytes.Buffer

					code := run(mode.args, noEnv, &out, &errOut)

					if code == 0 {
						t.Fatalf("exit = 0 on a broken install")
					}
					if out.Len() != 0 {
						t.Errorf("stdout = %q, want nothing printed", out.String())
					}
					if !strings.Contains(errOut.String(), tc.names) {
						t.Errorf("stderr = %q, want it to name %q", errOut.String(), tc.names)
					}
				})
			}
		})
	}
}

// installFontGlyphs is a tiny well-formed atlas: four records, each a solid 4x4
// cell with its own advance. Nothing here is drawn — the check run opens no
// window — so it needs only to LOAD, and four records is enough for the two
// counts to agree and for the pair to be a font.
func installFontGlyphs() []synth.Font16Glyph {
	glyphs := make([]synth.Font16Glyph, 4)
	for i := range glyphs {
		glyphs[i] = synth.Font16Glyph{Width: 4, Height: 4, Advance: uint32(i + 1),
			Ink: func(x, y int) (uint8, bool) { return 15, true }}
	}
	return glyphs
}

func TestGeneratorAssetsAreAStartupRequirement(t *testing.T) {
	o := installOptions{omitChargen: true, looseMaps: []string{"Beast.ALM"}}
	_, err := game.NewFrontEnd(writeInstall(t, o))
	if err == nil || !strings.Contains(err.Error(), "graphics/interface/chrgen/precreate/mainarea.bmp") {
		t.Fatalf("NewFrontEnd missing generator background = %v; want its address", err)
	}
}

// AC-13 — the three shapes a font container takes, through NewFrontEnd.
//
// It lives here and not in pkg/game because this is the package with an install
// fixture: pkg/game assembles a FrontEnd directly and so cannot ask what
// NewFrontEnd does with a broken one. The two switches used here exist for this
// test, so a broken font is a case rather than the accidental state of every
// fixture.
func TestTheFontIsLoadedAndIsNeverFatal(t *testing.T) {
	for _, tc := range []struct {
		name string
		o    installOptions
		font bool
	}{
		{"both nodes present", installOptions{}, true},
		{"the atlas missing", installOptions{omitFont: true}, false},
		{"the sidecar missing", installOptions{omitFontAdvances: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := tc.o
			o.looseMaps = []string{"Beast.ALM"}
			f, err := game.NewFrontEnd(writeInstall(t, o))
			// NOT FATAL is the whole point: every shape assembles.
			if err != nil {
				t.Fatalf("NewFrontEnd: %v — a font is not a startup requirement", err)
			}
			if tc.font {
				if f.Font.Value() == nil || f.Font.Err() != nil {
					t.Fatalf("Font = %v, FontErr = %v; want a font and no error", f.Font, f.Font.Err())
				}
				if got := f.CheckLine(); strings.Contains(got, "no font") {
					t.Errorf("the summary names the font on a complete install: %q", got)
				}
				return
			}
			if f.Font.Value() != nil {
				t.Errorf("Font = %v over a broken container, want none", f.Font)
			}
			if f.Font.Err() == nil {
				t.Fatal("a broken font container assembled with no reason recorded")
			}
			// The reason NAMES the failure: the address it could not read.
			if !strings.Contains(f.Font.Err().Error(), game.DefaultFont) {
				t.Errorf("FontErr = %q, which does not name the font", f.Font.Err())
			}
			if got := f.CheckLine(); !strings.Contains(got, "no font") {
				t.Errorf("the summary does not report the missing font: %q", got)
			}
		})
	}
}

// TestTheAttackPointerIsLoadedAndIsNeverFatal is 0080 AC-11 — the font's own
// shape, applied to the one instrument that has a fallback of its own.
//
// The mission still opens either way. What differs is only whether the map
// screen draws the game's own pointer or the authored mark, and the summary is
// the one place a headless run can say which.
func TestTheAttackPointerIsLoadedAndIsNeverFatal(t *testing.T) {
	for _, tc := range []struct {
		name    string
		o       installOptions
		pointer bool
	}{
		{"the art present", installOptions{}, true},
		{"the art missing", installOptions{omitCursor: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := tc.o
			o.looseMaps = []string{"Beast.ALM"}
			f, err := game.NewFrontEnd(writeInstall(t, o))
			// NOT FATAL is the whole point: both shapes assemble.
			if err != nil {
				t.Fatalf("NewFrontEnd: %v -- the cursor art is not a startup requirement", err)
			}
			if tc.pointer {
				if f.AttackPointer.Value() == nil || f.AttackPointer.Err() != nil {
					t.Fatalf("AttackPointer = %v, AttackPointerErr = %v; want a picture and no error",
						f.AttackPointer, f.AttackPointer.Err())
				}
				if got := f.CheckLine(); strings.Contains(got, "no attack pointer") {
					t.Errorf("the summary names the pointer on a complete install: %q", got)
				}
				return
			}
			if f.AttackPointer.Value() != nil {
				t.Errorf("AttackPointer = %v over an install without the art, want none", f.AttackPointer)
			}
			if f.AttackPointer.Err() == nil {
				t.Fatal("an install without the cursor art assembled with no reason recorded")
			}
			if !strings.Contains(f.AttackPointer.Err().Error(), game.AttackCursorPath) {
				t.Errorf("AttackPointerErr = %q, which does not name the address", f.AttackPointer.Err())
			}
			if got := f.CheckLine(); !strings.Contains(got, "no attack pointer") {
				t.Errorf("the summary does not report the missing pointer: %q", got)
			}
		})
	}
}

// gameFlags is the game front-end's COMPLETE flag set, hand-written.
//
// The inventory is not read back off the flag set, so a flag added to run()
// without being added here leaves this file green while the game's documented
// surface and its real one part company. That is the same instrument
// cmd/mapview/flagset_test.go carries, and it exists HERE because of what it
// forbids as much as what it records: the developer diagnostics — -unshaded,
// -flat, -blocked, -objectanim, -ruins — are the standalone viewer's alone, and
// the game must gain none of them. A structure drawn from its ruin block is a
// picture no state in this tree can reach, so a game that could ask for one
// would be a game showing a fact it does not have.
var gameFlags = []struct{ name, arg string }{
	{"assets", "somedir"},
	{"check", ""},
	{"markers", ""},
	{"picker", ""},
	{"skill", "blade"},
}

func TestGameFlagSurface(t *testing.T) {
	t.Run("every shipped flag is still defined", func(t *testing.T) {
		for _, f := range gameFlags {
			args := []string{"-" + f.name}
			if f.arg != "" {
				args = append(args, f.arg)
			}
			var out, errOut bytes.Buffer
			run(args, func(string) string { return "" }, &out, &errOut)
			if strings.Contains(errOut.String(), "not defined") {
				t.Errorf("-%s is no longer a defined flag: %s", f.name, errOut.String())
			}
		}
	})

	t.Run("the developer diagnostics are not reachable here", func(t *testing.T) {
		// Each of these exists on cmd/mapview and must not exist on the game.
		for _, name := range []string{"ruins", "unshaded", "flat", "blocked", "objectanim", "structures", "statics"} {
			var out, errOut bytes.Buffer
			run([]string{"-" + name}, func(string) string { return "" }, &out, &errOut)
			if !strings.Contains(errOut.String(), "not defined") {
				t.Errorf("-%s parsed; it is a developer diagnostic and the game front-end must not offer one", name)
			}
		}
	})
}

// The door: NEW GAME's default arms generation for mission 10, an explicit
// -mission overrides which mission and bypasses the menu to reach it
// directly, and every way the selected mission can fail says which one.

// missionInstall is the default install plus one entry that is present and will
// not decode, so all three failures are reachable over ONE install.
func missionInstall(t *testing.T) string {
	t.Helper()
	return writeInstall(t, installOptions{
		looseMaps:       []string{"Beast.ALM"},
		archiveMaps:     []string{"10.alm", "20.alm"},
		undecodableMaps: []string{"12.alm"},
	})
}

func TestMissionFlag(t *testing.T) {
	t.Run("the check mode accepts it and reports the mission it started", func(t *testing.T) {
		dir := missionInstall(t)
		var out, errOut bytes.Buffer

		code := run([]string{"-assets", dir, "-check", "-mission", "10"}, noEnv, &out, &errOut)
		if code != 0 {
			t.Fatalf("exit %d, want 0; stderr = %q", code, errOut.String())
		}
		got := out.String()
		// The ordinary check line is still printed, so nothing an install said
		// before this story stops being said.
		if !strings.Contains(got, "map rows") {
			t.Errorf("output %q does not carry the ordinary check line", got)
		}
		for _, want := range []string{"mission 10", "scenario/10.alm", "entities", "party at"} {
			if !strings.Contains(got, want) {
				t.Errorf("output %q does not carry %q", got, want)
			}
		}
	})

	t.Run("the three startup failures are three different messages", func(t *testing.T) {
		dir := missionInstall(t)
		cases := []struct {
			name string
			arg  string
			want string
		}{
			{"a number that names no mission", "-3", "not a campaign mission number"},
			{"a mission whose entry is absent", "11", "read scenario/11.alm"},
			{"an entry that will not decode", "12", "scenario/12.alm"},
		}
		seen := make(map[string]string, len(cases))
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var out, errOut bytes.Buffer
				code := run([]string{"-assets", dir, "-check", "-mission", tc.arg}, noEnv, &out, &errOut)
				if code == 0 {
					t.Fatalf("exit 0; stdout = %q", out.String())
				}
				msg := errOut.String()
				if !strings.Contains(msg, tc.want) {
					t.Errorf("message %q does not carry %q", msg, tc.want)
				}
				for other, prev := range seen {
					if prev == msg {
						t.Errorf("%q and %q report the same message %q", tc.name, other, msg)
					}
				}
				seen[tc.name] = msg
			})
		}
		if len(seen) != len(cases) {
			t.Fatalf("only %d of %d failures produced a message", len(seen), len(cases))
		}
	})

	t.Run("mission is unset by default, and picker is explicit", func(t *testing.T) {
		o, err := parse(nil)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if o.mission != 0 || o.picker || o.missionSet {
			t.Errorf("defaults = mission %d, picker %t, missionSet %t; want mission 0, picker false, missionSet false",
				o.mission, o.picker, o.missionSet)
		}
		o, err = parse([]string{"-mission", "7"})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if o.mission != 7 || !o.missionSet {
			t.Errorf("-mission 7 parsed as mission %d, missionSet %t; want 7, true", o.mission, o.missionSet)
		}
		// missionSet is set by GIVING the flag, not by the number it carries --
		// "-mission 10" still bypasses the menu, exactly as any other explicit
		// value does, unlike a bare launch that never mentions -mission at all.
		o, err = parse([]string{"-mission", "10"})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if o.mission != 10 || !o.missionSet {
			t.Errorf("-mission 10 parsed as mission %d, missionSet %t; want 10, true", o.mission, o.missionSet)
		}
		o, err = parse([]string{"-picker"})
		if err != nil {
			t.Fatalf("parse -picker: %v", err)
		}
		if !o.picker {
			t.Error("-picker parsed false")
		}
		if _, err := parse([]string{"-picker", "-mission", "10"}); err == nil {
			t.Error("-picker with an explicit -mission parsed without an ambiguity error")
		}
	})

	// Owner reversal: this is no longer what a bare launch's WINDOW does --
	// TestNewGameOpensGenerationDirectly proves the window itself opens the
	// menu. It is what NEW GAME's own default target is, and -check still
	// reports it unconditionally on o.picker alone (main.go's own comment at
	// the print site carries why), because a player reaches it with one click.
	t.Run("without a flag, -check still reports NEW GAME's default of mission 10", func(t *testing.T) {
		dir := missionInstall(t)
		var out, errOut bytes.Buffer
		if code := run([]string{"-assets", dir, "-check"}, noEnv, &out, &errOut); code != 0 {
			t.Fatalf("exit %d; stderr = %q", code, errOut.String())
		}
		for _, want := range []string{"mission 10", "scenario/10.alm", "chargen offers"} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("output %q does not carry %q", out.String(), want)
			}
		}
	})

	t.Run("-picker's own check line stays the former one-line summary, then names the base", func(t *testing.T) {
		dir := missionInstall(t)
		var out, errOut bytes.Buffer
		if code := run([]string{"-assets", dir, "-check", "-picker"}, noEnv, &out, &errOut); code != 0 {
			t.Fatalf("exit %d; stderr = %q", code, errOut.String())
		}
		if n := strings.Count(strings.TrimSpace(out.String()), "\n"); n != 1 || !strings.Contains(out.String(), "\nagainrom: base rom1-en ") {
			t.Errorf("output is %d lines, want the summary and the base line:\n%s", n+1, out.String())
		}
		if strings.Contains(out.String(), "mission") {
			t.Errorf("output %q mentions a mission on the picker route", out.String())
		}
	})
}

// TestChargenGeneration covers the default and explicit mission doors as of
// 0140: character generation is not a flag any more, it runs whenever a
// campaign is STARTED (owner), and -mission N is a fresh start. What this
// file can witness is the command line's own door -- the flag surface, and
// the check mode's report of what generation would offer, reached either by
// an explicit -mission or by NEW GAME's own default (owner reversal;
// TestMissionFlag's own header carries the current shape of the door). The
// map list's own door is pkg/game's gate, tested there; that a campaign
// TRANSITION opens no generation screen is pkg/ui's
// TestCampaignAdvanceNeverArmsGeneration.
//
// EVERY CASE HERE GOES THROUGH -check, for TestMissionFlag's own reason spelled
// out above it: the check mode is where every startup outcome is reachable with
// no window, and a case driven through the windowed arm would open one in a
// test -- with TestMissionFlag's own one exception, which this file does not
// need a second copy of.
func TestChargenGeneration(t *testing.T) {
	t.Run("-chargen is still accepted and is not a usage error on its own", func(t *testing.T) {
		// It USED to be one: -chargen without -mission was refused before the
		// asset root was resolved. Nothing about the flag switches anything now,
		// so nothing about it can be wrong -- and an EMPTY directory, not an
		// install at all, is what makes the reason for the exit unambiguous: the
		// run gets as far as reading archives and fails there, on the assets,
		// with the flag having caused nothing.
		dir := t.TempDir()
		var out, errOut bytes.Buffer

		code := run([]string{"-assets", dir, "-check", "-chargen"}, noEnv, &out, &errOut)

		if code == 2 {
			t.Fatalf("exit = 2 (a usage error) for -chargen alone; the flag is inert now "+
				"(stderr: %s)", errOut.String())
		}
		if strings.Contains(errOut.String(), "requires") {
			t.Errorf("stderr = %q, want no requirement between -chargen and -mission", errOut.String())
		}
	})

	t.Run("-chargen changes nothing about a run that gives it", func(t *testing.T) {
		// The whole of "inert", stated as the property that makes it testable:
		// the same command line with and without the flag produces byte-identical
		// output. Nothing in run reads the flag -- it is not even a field of
		// options -- and this is what says so from the outside.
		dir := missionInstall(t)
		var withFlag, withoutFlag, errOut bytes.Buffer

		if code := run([]string{"-assets", dir, "-check", "-mission", "10", "-chargen"}, noEnv, &withFlag, &errOut); code != 0 {
			t.Fatalf("with the flag: exit = %d (stderr: %s)", code, errOut.String())
		}
		if code := run([]string{"-assets", dir, "-check", "-mission", "10"}, noEnv, &withoutFlag, &errOut); code != 0 {
			t.Fatalf("without the flag: exit = %d (stderr: %s)", code, errOut.String())
		}
		if withFlag.String() != withoutFlag.String() {
			t.Errorf("-chargen changed the output: with it %q, without it %q",
				withFlag.String(), withoutFlag.String())
		}
	})

	t.Run("-check -mission N reports the generation screen, flag or no flag", func(t *testing.T) {
		dir := missionInstall(t)
		var out, errOut bytes.Buffer

		code := run([]string{"-assets", dir, "-check", "-mission", "10"}, noEnv, &out, &errOut)

		if code != 0 {
			t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut.String())
		}
		got := out.String()
		// The ordinary check line, the mission line and the advance line are all
		// still there -- the chargen line is ADDED, not swapped in.
		for _, want := range []string{"map rows", "mission 10", "scenario/10.alm", "winning mission 10"} {
			if !strings.Contains(got, want) {
				t.Errorf("output %q does not carry %q", got, want)
			}
		}
		// The row names, in order, and the budget -- built from
		// front.ChargenSetup() itself in main.go, so this asserts its own
		// output rather than a second copy of the setup's content.
		for _, want := range []string{"Sex", "Class", "Skill", "Body", "Reaction", "Mind", "Spirit"} {
			if !strings.Contains(got, want) {
				t.Errorf("output %q does not name the chargen row %q", got, want)
			}
		}
		if want := fmt.Sprintf("budget %d", data.ChargenBudget); !strings.Contains(got, want) {
			t.Errorf("output %q does not carry %q", got, want)
		}
		// The chargen line comes AFTER the mission line -- main.go's own
		// comment at the call site gives the reason: it is its own row, about
		// a different character than the one MissionLine just reported.
		if mi, ci := strings.Index(got, "mission 10"), strings.Index(got, "chargen offers"); mi < 0 || ci < 0 || ci < mi {
			t.Errorf("output %q does not carry the mission line before the chargen line", got)
		}
	})

	t.Run("a windowed -mission naming no mission is refused before any window", func(t *testing.T) {
		// THE ONE WINDOWED CASE THIS FILE CAN DRIVE. Every other case here goes
		// through -check because the windowed arm ends in app.Run(); this one
		// returns before it, which is the property being asserted -- putting a
		// generation screen in front of the mission must not turn "that number
		// names no mission" into something the player learns after spending a
		// spread.
		dir := missionInstall(t)
		var out, errOut bytes.Buffer

		code := run([]string{"-assets", dir, "-mission", "-3"}, noEnv, &out, &errOut)

		if code != 2 {
			t.Fatalf("exit = %d, want 2 (stderr: %s)", code, errOut.String())
		}
		if !strings.Contains(errOut.String(), "names no mission") {
			t.Errorf("stderr = %q, want it to say the number names no mission", errOut.String())
		}
		if out.Len() != 0 {
			t.Errorf("stdout = %q, want nothing printed", out.String())
		}
	})

	t.Run("-check -picker reports no generation screen", func(t *testing.T) {
		// Generation belongs to NEW GAME's own default target. -picker replaces
		// that destination with the debug map list instead, which enters no
		// generation directly, so there is nothing for it to report and the old
		// check line stays unchanged.
		dir := missionInstall(t)
		var out, errOut bytes.Buffer

		code := run([]string{"-assets", dir, "-check", "-picker"}, noEnv, &out, &errOut)

		if code != 0 {
			t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut.String())
		}
		got := out.String()
		if !strings.Contains(got, "map rows") {
			t.Errorf("output %q lost the ordinary check line", got)
		}
		if strings.Contains(got, "chargen") {
			t.Errorf("output %q mentions chargen on the picker route", got)
		}
	})
}

// TestNewGameOpensGenerationDirectly is the owner-reversal hotfix's own proof
// (docs/hotfix/LEDGER.md): a default run's window opens the main menu and
// nothing else; NEW GAME there arms generation for mission 10 directly and a
// legal confirm starts it; the map picker is unreachable without -picker and
// IS reachable with it.
//
// IT DRIVES THE EXACT PRODUCTION WIRING run() INSTALLS -- frontEnd, front.App,
// armNewGameDoor -- through ui.App's own headless surface (App.Layout,
// HeadlessActivate, game.RunHeadlessScenario), the same instrument
// cmd/mapview-free release tests already use to press brooch buttons and drive
// character generation without opening a real window. No window is ever
// opened, on golden rule 2's own grounds.
//
// EVERY CASE HERE FAILED TO COMPILE BEFORE THIS HOTFIX: armNewGameDoor and
// App.SetNewGameChargen are both new, and the run() this file exercised
// before called App.OpenChargen unconditionally, which put the screen on
// ScreenChargen before a test could ever ask what a bare launch shows.
func TestNewGameOpensGenerationDirectly(t *testing.T) {
	dir := missionInstall(t)

	t.Run("a default run opens the menu, and NEW GAME arms generation for mission 10", func(t *testing.T) {
		o, err := parse(nil)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		front, err := frontEnd(dir, o, game.OptionsStore{})
		if err != nil {
			t.Fatalf("frontEnd: %v", err)
		}
		app := front.App("test")
		app.Layout(640, 480)
		if err := armNewGameDoor(app, front, o); err != nil {
			t.Fatalf("armNewGameDoor: %v", err)
		}

		// a. the window opens at the menu, and nothing else.
		if app.Screen() != ui.ScreenMenu {
			t.Fatalf("screen right after wiring = %v, want ui.ScreenMenu", app.Screen())
		}

		// b. NEW GAME opens generation directly -- not the map picker.
		if err := app.HeadlessActivate("new game"); err != nil {
			t.Fatalf("activate NEW GAME: %v", err)
		}
		if app.Screen() != ui.ScreenChargen {
			t.Fatalf("screen after NEW GAME = %v, want ui.ScreenChargen (not the picker)", app.Screen())
		}

		// b, continued: confirming a legal spread starts mission 10, through
		// the SAME production create_character drive the shipped scenarios use
		// (e.g. scenarios/0163-chargen-mission10.json).
		scenario := game.HeadlessScenario{Version: game.HeadlessScenarioVersion, Steps: []game.HeadlessStep{
			{Command: "create_character", Character: &game.HeadlessCharacter{Name: "Witness"}},
		}}
		if err := game.RunHeadlessScenario(front, app, scenario, io.Discard, io.Discard); err != nil {
			t.Fatalf("create_character: %v", err)
		}
		if app.Screen() != ui.ScreenMap {
			t.Fatalf("screen after confirm = %v, want ui.ScreenMap", app.Screen())
		}
		line, err := front.MissionLine(10)
		if err != nil {
			t.Fatalf("MissionLine(10): %v", err)
		}
		if !strings.Contains(line, "mission 10 at") {
			t.Fatalf("MissionLine(10) = %q, does not name mission 10", line)
		}
	})

	// c. the map picker is unreachable without -picker...
	t.Run("without -picker, NEW GAME never reaches the map picker", func(t *testing.T) {
		o, err := parse(nil)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		front, err := frontEnd(dir, o, game.OptionsStore{})
		if err != nil {
			t.Fatalf("frontEnd: %v", err)
		}
		app := front.App("test")
		app.Layout(640, 480)
		if err := armNewGameDoor(app, front, o); err != nil {
			t.Fatalf("armNewGameDoor: %v", err)
		}
		if err := app.HeadlessActivate("new game"); err != nil {
			t.Fatalf("activate NEW GAME: %v", err)
		}
		if app.Screen() == ui.ScreenPicker {
			t.Fatal("NEW GAME reached the map picker without -picker")
		}
	})

	// c, continued: ...and IS reachable with it.
	t.Run("-picker makes NEW GAME open the map picker instead", func(t *testing.T) {
		o, err := parse([]string{"-picker"})
		if err != nil {
			t.Fatalf("parse -picker: %v", err)
		}
		front, err := frontEnd(dir, o, game.OptionsStore{})
		if err != nil {
			t.Fatalf("frontEnd: %v", err)
		}
		app := front.App("test")
		app.Layout(640, 480)
		// run() never calls armNewGameDoor under -picker (the o.picker guard at
		// the call site); not calling it here mirrors that exactly, rather than
		// asserting a state this build never reaches.
		if err := app.HeadlessActivate("new game"); err != nil {
			t.Fatalf("activate NEW GAME: %v", err)
		}
		if app.Screen() != ui.ScreenPicker {
			t.Fatalf("screen with -picker after NEW GAME = %v, want ui.ScreenPicker", app.Screen())
		}
	})
}

// definitionTable is the fixture's placeable-definition table: no placements,
// and the three item collections the party's starting weapon resolves through.
//
// The numbers are the SHAPE of the shipped file and not its content — a shape
// factor, a material factor and a row with a physical pair — chosen so the band
// they compose is one no other arithmetic in this fixture produces.
func definitionTable(o installOptions) []byte {
	scale := func(name string, damage float64) synth.DataBinRow {
		d := make([]float64, 9)
		d[4], d[5], d[6] = damage, 1, 1
		return synth.DataBinRow{Name: name, Doubles: d}
	}
	var d synth.DataBin
	if o.omitStartingWeapon {
		return d.Bytes()
	}
	d.Rows[synth.DataBinShapes] = []synth.DataBinRow{scale("Common", 0.2)}
	d.Rows[synth.DataBinMaterials] = []synth.DataBinRow{scale("Iron", 1)}
	d.Rows[synth.DataBinWeapons] = []synth.DataBinRow{{
		Name: "Short Sword",
		//    0   1   2   3   4  kind min max toHit def  10 rng chg rlx
		Params: []int32{-1, -1, -1, -1, -1, 1, 23, 40, 0, 0, -1, 1, 9, 5, -1},
	}}
	if o.withShootWeapon {
		// A second shape and a second material, additive over the blade's
		// own: "Iron Short Sword" still resolves through "Iron" exactly as
		// it did, because "Uncommon"/"Wood" are not a prefix of it. The
		// attack type at slot 5 is 10 — data's own ranged threshold — and
		// the range cell at slot 0xb is 6, so a hero holding this weapon
		// derives a reach above 1 (spec 0120 AC-5).
		d.Rows[synth.DataBinShapes] = append(d.Rows[synth.DataBinShapes], scale("Uncommon", 1))
		d.Rows[synth.DataBinMaterials] = append(d.Rows[synth.DataBinMaterials], scale("Wood", 1))
		d.Rows[synth.DataBinWeapons] = append(d.Rows[synth.DataBinWeapons], synth.DataBinRow{
			Name: "Short Bow",
			//    0   1   2   3   4  kind min max toHit def  10 rng chg rlx  14
			Params: []int32{-1, -1, -1, -1, -1, 10, 10, 20, 5, 0, -1, 6, 8, 4, -1},
		})
	}
	return d.Bytes()
}

// installHeroBody is the ONE entry this fixture's own body list needs a real
// name for, and it is INVENTED here rather than transcribed from anything
// shipped (SC-3).
//
// It is entry 0 of installBodyList, and entry 0 is the only one either party
// shape this file ever builds lands on: definitionTable writes at most ONE
// Weapons row, so a resolved starting weapon's own Row is always 1 — the
// collection's reserved entry 0 plus that one write — and a BARE hero's
// empty first slot answers row 1 too (data.HeroBodyFor's own "the two live
// arms meet at row 1"). Both routes ask the list for the entry one before
// row 1, which is entry 0, so this fixture never needs a second name.
const installHeroBody data.HeroBody = "fixture-hero"

// installBodyList is the shipped body list this fixture writes into the
// install, at the address ReadBodyList reads. It carries the one name
// installHeroBody's own note explains, and nothing more.
func installBodyList() []byte {
	return []byte(string(installHeroBody) + "\n")
}

// installUnitsReg is a units.reg declaring the ONE class installHeroBody's
// name resolves to, with a canvas and no art address of its own — the
// hero's pixels do not come through a class record's File, so the record is here
// for the geometry alone.
func installUnitsReg() []byte {
	i := func(name string, v int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x02, Int: v}
	}
	class, _ := data.HeroBodyClass(installHeroBody)
	return synth.UnitsReg([]string{`absent\none`},
		[]synth.RegNode{
			i("ID", class), i("File", 0),
			i("Width", 16), i("Height", 16), i("CenterX", 8), i("CenterY", 14),
		})
}

// installHeroSheetAddress is the entry the appearance law composes for that
// body, taken FROM the law rather than written out, so this fixture cannot drift
// away from the address startup actually reads.
//
// It needs no graphicsEntry: the composed address carries no identity segment of
// its own — the loader prefixes one at the read — so it is already the path
// inside graphics.res.
//
// THE DIRECTORY IS THE LAW'S OWN NO-MAGE, NO-ARMOUR ARM, called directly
// rather than through the retired game.PartyBodyDir (0134 T4): this fixture's
// party is bare-handed with no base row and no table, so its worn set is
// empty and the directory arm this composes has to be — mage false, slot 8
// unoccupied — is the one the preload and the party assembly both land on
// for it too.
func installHeroSheetAddress() string {
	dir, _ := data.HeroBodyDir(false, 0, false)
	return data.HeroSheetPath(dir, installHeroBody)
}

// installHeroSheet is one painted frame — enough to resolve, and nothing here
// looks at the picture.
func installHeroSheet() []byte {
	return synth.Sheet256(synth.Sheet256Options{
		Palette: []color.RGBA{{}, {R: 0xff, G: 0xff, B: 0xff}},
		Frames: []synth.Frame256{{Width: 1, Height: 1,
			Pixels: []synth.Pixel256{{Index: 1, Opaque: true}}}},
	})
}

// installCursorSheet is the smallest well-formed .16a the cursor loader accepts:
// a palette block, one 1x1 record painting a single literal pixel, and the
// palette-bearing trailer. Its colour is never read here -- the check mode
// reports only whether a picture loaded.
func installCursorSheet() []byte {
	out := make([]byte, 1024)
	out[4*7+2] = 0xff // palette entry 7, R
	out = binary.LittleEndian.AppendUint32(out, 1)
	out = binary.LittleEndian.AppendUint32(out, 1)
	out = binary.LittleEndian.AppendUint32(out, 4)
	// One literal word, then the pixel: palette index 7 at the top level.
	out = binary.LittleEndian.AppendUint16(out, 1)
	out = binary.LittleEndian.AppendUint16(out, uint16(7<<1)|uint16(15<<9))
	return binary.LittleEndian.AppendUint32(out, 1|0x80000000)
}

// installNameFontSheet is the pre-create prompt and name font's atlas: the
// 1024-byte palette the font loader declares, one 1x1 record painting a single
// literal pixel at level 15, and a plain record-count trailer.
func installNameFontSheet() []byte {
	out := make([]byte, 1024)
	out = binary.LittleEndian.AppendUint32(out, 1)
	out = binary.LittleEndian.AppendUint32(out, 1)
	out = binary.LittleEndian.AppendUint32(out, 4)
	out = binary.LittleEndian.AppendUint16(out, 1)
	out = binary.LittleEndian.AppendUint16(out, uint16(255<<1)|uint16(15<<9))
	return binary.LittleEndian.AppendUint32(out, 1)
}

func TestTheOriginalSavesAreReadFromTheRESOLVEDRootAndNotTheFlag(t *testing.T) {
	root := filepath.Join(t.TempDir(), "install")
	executable := filepath.Join(t.TempDir(), "againrom.exe")
	profile, orig, err := loadSources(root, options{}, executable, "")
	if err != nil {
		t.Fatalf("loadSources: %v", err)
	}
	if orig.Dir != root {
		t.Errorf("the original saves would be read from %q, want the resolved root %q", orig.Dir, root)
	}
	// The halves are DIFFERENT directories, and this is the sentence that says
	// so: nothing this build writes may land in the install it reads.
	if profile.Saves.Dir == "" {
		t.Fatal("no save directory was resolved at all")
	}
	if profile.Saves.Dir == orig.Dir || strings.HasPrefix(profile.Saves.Dir, orig.Dir) {
		t.Errorf("saves would be written at %q, inside the install at %q", profile.Saves.Dir, orig.Dir)
	}
	// An override is still the override, so a player who names a directory
	// gets it and the install half is unaffected by his naming one.
	want := filepath.Join(t.TempDir(), "elsewhere")
	profile, orig, err = loadSources(root, options{saves: want}, executable, "")
	if err != nil || profile.Saves.Dir != want || orig.Dir != root {
		t.Errorf("loadSources(-saves) = (%q, %q, %v), want (%q, %q, nil)", profile.Saves.Dir, orig.Dir, err, want, root)
	}
}
