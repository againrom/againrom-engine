// Command paneldump composes the unit information panel against a lawful game
// install and reports HOW BIG THE COMPOSED BOX IS, in pixels, for the two kinds
// of subject the window has to serve.
//
//	paneldump [-assets DIR] [-mission N] [-png DIR]
//
// It is the measuring instrument for 0116. "The panel is compact" is a claim
// about pixels, and pixels are a function of the game's own font, which no
// synthetic test may read (golden rule 2) — so the number can only come
// from a developer tool, and its output is what verification.md records.
//
// THE TWO SUBJECTS ARE REAL AND ARE NOT FIXTURES. A campaign mission is
// started exactly as the front end starts it, and the two subjects are read
// out of the world it produces: the party member standing on a start cell,
// and the first unit the MAP placed. The second is the case the owner named,
// and no fixture can stand in for it. Until 0137 the loader knew no
// character at all for that second subject and the panel drew seven fewer
// rows for it; it knows one now, so the two boxes are measured against each
// other rather than one against a stub.
//
// It writes no game data anywhere: the PNGs it can emit are the panel's own
// composed box — our frame, our labels, the install's font glyphs — and they go
// outside the repo, to a directory the caller names.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sort"

	"againrom/pkg/data"
	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "paneldump:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("paneldump", flag.ContinueOnError)
	fs.SetOutput(out)
	assets := fs.String("assets", "", "game install root (or AGAINROM_ASSETS)")
	mission := fs.Int("mission", 10, "campaign mission number")
	pngDir := fs.String("png", "", "write each panel as a PNG into this directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *mission <= 0 {
		return fmt.Errorf("-mission must be positive")
	}
	root := game.ResolveAssetRoot(*assets, os.Getenv("AGAINROM_ASSETS"))
	if root == "" {
		return fmt.Errorf("no asset root: pass -assets or set AGAINROM_ASSETS")
	}
	archives, err := game.OpenArchives(root)
	if err != nil {
		return err
	}
	font, err := game.LoadFont(archives.Containers, game.DefaultFont, game.FontShades)
	if err != nil {
		return err
	}
	units, err := game.LoadUnits(archives.Containers)
	if err != nil {
		return err
	}
	// LoadDefinitions rather than LoadTable, missionrun's own reason: the
	// party's hero is armed out of the SAME walk of the same file the
	// placements resolve against, so the tool and the game start one party.
	defs, err := game.LoadDefinitions(archives.Containers)
	if err != nil {
		return err
	}
	ms, err := game.StartMission(archives.Containers, *mission, defs.Table,
		mapload.DifficultyNormal, game.MissionParty(defs.StartWeapon, defs.Bodies, defs.Table))
	if err != nil {
		return err
	}

	layout := ui.AuthoredPanelLayout()
	fmt.Fprintf(out, "mission %d  %s  font %s  line height %d\n",
		ms.Number, ms.Address, game.DefaultFont, font.Height())
	for _, sub := range panelSubjects(ms, units, defs.Table) {
		report(out, layout, font, sub, *pngDir)
	}
	return nil
}

// subject is one panel subject with the word for which KIND of unit it is.
type subject struct {
	kind string
	s    ui.PanelSubject
}

// panelSubjects reads the two subjects out of a started mission.
//
// THE FIELDS ARE THE SAME FIELDS THE RUNNING GAME PUSHES. Everything the panel
// states about a unit is a copy off the simulation entity — the health and mana
// pairs, the cell, the eight numbers a blow reads, the rate a step reads — plus
// the actor name and class-name fallback the front end handed over and the
// character the loader knew, which is game.MissionCharacters' own answer.
// That is what pkg/game's per-frame entity push does; a tool deriving any of
// it differently would measure a window nobody sees.
//
// IT WAS game.PartyCharacters UNTIL 0137, when the running game began
// stating a character for every placement that resolves to a definition
// entry. A tool left on the older call would have gone on composing a placed
// unit's panel out of the zero character — the exact failure the sentence
// above names, arrived at by standing still. WHICH SUBJECT IS WHICH is
// therefore no longer decidable from "is a character known": both are now,
// so the party half is asked of game.PartyCharacters separately, purely as
// the membership test it has become here.
func panelSubjects(ms *game.Mission, units *terrain.UnitSet, defs *mapload.Table) []subject {
	chars := game.MissionCharacters(ms, defs)
	party := game.PartyCharacters(ms)
	ents := ms.World.Entities()
	sort.Slice(ents, func(i, j int) bool { return ents[i].ID < ents[j].ID })

	var out []subject
	var gotParty, gotPlain bool
	for _, e := range ents {
		c := chars[e.ID]
		_, known := party[e.ID]
		if (known && gotParty) || (!known && gotPlain) {
			continue
		}
		name := ""
		if cl := units.Classes[e.Class]; cl != nil {
			name = cl.Name
		}
		s := ui.PanelSubject{
			ID: uint32(e.ID), Name: name,
			HP: int(e.HP), MaxHP: int(e.MaxHP),
			Mana: int(e.Mana), MaxMana: int(e.MaxMana),
			Cell:     image.Pt(int(e.X), int(e.Y)),
			Selected: 1,
			Combat: ui.UnitCombat{Known: true,
				DamageBase: int(e.DamageBase), DamageSpread: int(e.DamageSpread),
				ToHit: int(e.ToHit), Defence: int(e.Defence), Absorption: int(e.Absorption),
				AttackCharge: int(e.AttackCharge), AttackRelax: int(e.AttackRelax),
				AlwaysHits: e.AlwaysHits},
			Char:  c,
			Speed: int(e.Speed),
		}
		if equipped, ok := ms.World.Equipped(e.ID); ok {
			s.Worn = wornNames(equipped, defs)
		}
		if known {
			out, gotParty = append(out, subject{kind: "party", s: s}), true
		} else {
			out, gotPlain = append(out, subject{kind: "placed", s: s}), true
		}
		if gotParty && gotPlain {
			break
		}
	}
	return out
}

// wornNames turns e's worn code array into the panel's own row-name list
// (tasks T4): index 0 (slot 1) is read against the Weapons collection, index
// 1 (slot 2) against the Shields collection, and every other index against
// the Armors collection — the SLOT the code sits in decides which
// collection names it, never the code's own class field. data.ItemCode's D()
// accessor is the row field every one of the three collections is read by,
// exactly as pkg/data's own resolvers already read it back (weapon.go's
// WeaponFromCode).
//
// A CODE WHOSE ROW NAMES NO ENTRY, OR AN EMPTY ENTRY, CONTRIBUTES NOTHING
// rather than an empty string standing in for a name — the same "nothing to
// say" rule PanelSubject's own zero value already carries for every other
// field, so a slot this build cannot name composes exactly the picture an
// unfilled one does.
//
// THE ROW NAME ALONE, NOT THE SHAPE OR THE MATERIAL WORD: the panel states
// `Chain Mail`, not `Uncommon Steel Chain Mail` — the row is the piece's
// identity and the two others are scaling words this function never reads
// off the code at all.
func wornNames(equipped [ui.PanelWornSlots]uint16, t *mapload.Table) [ui.PanelWornSlots]string {
	var out [ui.PanelWornSlots]string
	if t == nil {
		return out
	}
	for i, code := range equipped {
		if code == 0 {
			continue
		}
		var coll data.Collection
		switch i + 1 {
		case 1:
			coll = t.Weapons
		case 2:
			coll = t.Shields
		default:
			coll = t.Armors
		}
		if coll == nil {
			continue
		}
		row := data.ItemCode(code).D()
		if row < 0 || row >= coll.Len() {
			continue
		}
		out[i] = coll.EntryName(row)
	}
	return out
}

// reportFit prints the rows this layout could not draw as stated.
//
// A LAYOUT WITH A PINNED WIDTH TRUNCATES (panel.go, layoutLines): where
// PanelLayout.Size.X is above zero, a value wider than its own column is
// shortened rune by rune until it ends inside the box, down to the empty
// string. AuthoredPanelLayout pins Size.X to sidebarWidth, 300, so the
// mission panel IS such a layout and this pass does run on it.
//
// PanelStatement above says what the panel MEANS to state; this says what it
// draws. The two differ exactly on the rows listed here.
func reportFit(out io.Writer, l ui.PanelLayout, f *text.Font, s ui.PanelSubject) {
	if l.Size.X <= 0 {
		fmt.Fprintf(out, "         fit: Size.X=%d, not a pinned width: no row is truncated\n", l.Size.X)
		return
	}
	budget := l.Size.X - 2*l.Pad.X
	fmt.Fprintf(out, "         fit: width %d, pad %d, left budget %d\n", l.Size.X, l.Pad.X, budget)
	cut := 0
	for i, ln := range ui.CharacterPanelReport(l, f, s) {
		if ln.Value != ln.FullValue {
			w, _ := f.Measure(ln.FullValue)
			fmt.Fprintf(out, "         cut: row %d %-11q value %q (%dpx) drawn as %q, budget %d\n",
				i, ln.Label, ln.FullValue, w, ln.Value, budget)
			cut++
		}
		if ln.RightValue != ln.FullRightValue {
			w, _ := f.Measure(ln.FullRightValue)
			fmt.Fprintf(out, "         cut: row %d %-11q right value %q (%dpx) drawn as %q, budget %d\n",
				i, ln.RightLabel, ln.FullRightValue, w, ln.RightValue, budget-ln.RightX)
			cut++
		}
	}
	fmt.Fprintf(out, "         fit: %d truncated cell(s)\n", cut)
}

func report(out io.Writer, l ui.PanelLayout, f *text.Font, sub subject, dir string) {
	img := ui.RenderPanel(l, f, sub.s)
	if img == nil {
		fmt.Fprintf(out, "%-6s id=%d  NO PANEL\n", sub.kind, sub.s.ID)
		return
	}
	b := img.Bounds().Size()
	stated := ui.PanelStatement(l, sub.s)
	fmt.Fprintf(out, "%-6s id=%-4d box=%dx%d  rows=%d  name=%q\n",
		sub.kind, sub.s.ID, b.X, b.Y, len(stated), sub.s.Name)
	for _, r := range stated {
		fmt.Fprintf(out, "         | %s\n", r)
	}
	reportFit(out, l, f, sub.s)
	if dir == "" {
		return
	}
	path := filepath.Join(dir, fmt.Sprintf("panel-%s-%d.png", sub.kind, sub.s.ID))
	fh, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(out, "         png:", err)
		return
	}
	defer fh.Close()
	if err := png.Encode(fh, img); err != nil {
		fmt.Fprintln(out, "         png:", err)
		return
	}
	fmt.Fprintln(out, "         png:", path)
}
