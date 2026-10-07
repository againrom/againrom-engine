// Command wearcheck reads a lawful install and reports how much of the
// shipped equipment data this build resolves.
//
//	wearcheck [-assets DIR] [-spells] [-speakers] [-weights]
//
// -weights resolves the shipped item weight column over the whole shape x
// material cross product, per item collection; see weights.go. -speakers
// joins every `npc<n>` section of the scenario NPC registry that composes a
// figure with the starting outfit its Humans row would carry. That is a
// corpus join, not the runtime's synthetic-speaker wardrobe.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/game"
	"againrom/pkg/mapload"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "wearcheck:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("wearcheck", flag.ContinueOnError)
	fs.SetOutput(out)
	assets := fs.String("assets", "", "game install root (or AGAINROM_ASSETS)")
	spells := fs.Bool("spells", false,
		"print every castSpell= token this install's Humans and Units rows name, and whether it resolves to a Spells row (0139 AC-1)")
	speakers := fs.Bool("speakers", false,
		"print every figure-composing npc record and its Humans-row starting outfit (0160 corpus join)")
	weights := fs.Bool("weights", false,
		"print the shipped item weight column's range, resolved over every shape and material (1025 shipped-content sweep)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root := game.ResolveAssetRoot(*assets, os.Getenv("AGAINROM_ASSETS"))
	if root == "" {
		return fmt.Errorf("no asset root: pass -assets or set AGAINROM_ASSETS")
	}
	archives, err := game.OpenArchives(root)
	if err != nil {
		return err
	}
	// LoadDefinitions is pkg/game/table.go's own one walk of data.bin — the
	// SAME databin.Parse of the SAME archive member the placement loader
	// itself reads (table.go's tableAddress), so this tool measures the
	// install the game would load, never a second read of its own invention.
	defs, err := game.LoadDefinitions(archives.Containers)
	if err != nil {
		return err
	}
	if *spells {
		return printSpellReport(out, scanSpells(defs.Table))
	}
	if *speakers {
		return printSpeakerReport(out, game.LoadNPCFaces(archives.Containers), defs.Table)
	}
	if *weights {
		return printWeightReport(out, sweepWeights(defs.Table))
	}
	printReport(out, scan(defs.Table))
	return nil
}

// printSpeakerReport is 0160's corpus probe: every `npc<n>` section of the
// scenario NPC registry that composes a FIGURE, joined to the starting outfit
// its Humans row would carry.
//
// IT DOES NOT DEFINE THE RUNTIME ARM. A synthetic dialogue speaker is bare;
// this report remains useful only to show why materialising a definition-row
// outfit onto that portrait is visible. It writes no file: stdout only, and a
// row with no outfit is an ordinary census result rather than an error.
func printSpeakerReport(out io.Writer, faces map[int32]data.NPCFace, t *mapload.Table) error {
	subs := make([]int, 0, len(faces))
	for sub, rec := range faces {
		if rec.Kind == data.NPCFigure {
			subs = append(subs, int(sub))
		}
	}
	sort.Ints(subs)

	fmt.Fprintf(out, "wearcheck -speakers: %d of %d npc record(s) compose a figure\n\n",
		len(subs), len(faces))
	dressed, headed, bare := 0, 0, 0
	for _, sub := range subs {
		rec := faces[int32(sub)]
		worn, row, ok := mapload.SpeakerOutfit(int32(sub), rec.Dir.Mage(), rec.Dir.Female(), t)
		if !ok || worn == (emptyWorn) {
			bare++
			fmt.Fprintf(out, "  npc%-4d %-9s face=%-3d row=%-16s -> NO OUTFIT AT ALL\n",
				sub, rec.Dir, rec.Face, row)
			continue
		}
		dressed++
		filled := make([]string, 0, len(worn))
		for slot := 1; slot <= len(worn); slot++ {
			if worn[slot-1] != 0 {
				filled = append(filled, fmt.Sprintf("%d:%04x", slot, worn[slot-1]))
			}
		}
		if worn[headSlot-1] != 0 {
			headed++
		}
		fmt.Fprintf(out, "  npc%-4d %-9s face=%-3d row=%-16s -> %s\n",
			sub, rec.Dir, rec.Face, row, strings.Join(filled, " "))
	}
	fmt.Fprintf(out, "\n%d dressed, %d of them with the head slot filled, %d bare\n",
		dressed, headed, bare)
	return nil
}

// headSlot is the equipment slot the head layer is drawn from
// (`DLG-FIGURE-021`), restated here for cellWeapon's own reason: a developer
// tool cannot reach another tier's unexported constant.
const headSlot = 6

// emptyWorn is the worn set of a person wearing nothing. It is sized from
// data.EquipSlots rather than from pkg/sim's own count, which a cmd/ tool may
// not import; the two are the same twelve slots.
var emptyWorn [data.EquipSlots]uint16

const (
	cellWeapon    = 0
	cellShield    = 1
	cellArmorFrom = 2
	cellCount     = 10
)

const weaponHandsColumn = 0xe

type itemClass int

const (
	classWeapon itemClass = iota
	classShield
	classArmor
)

func (c itemClass) String() string {
	switch c {
	case classWeapon:
		return "weapon"
	case classShield:
		return "shield"
	case classArmor:
		return "armour"
	default:
		return "?"
	}
}

// cellOutcome is what one equipment cell resolved to.
type cellOutcome int

const (
	outcomeEmpty cellOutcome = iota
	outcomeDropped
	outcomeWorn
	outcomeCarried
)

func (o cellOutcome) String() string {
	switch o {
	case outcomeEmpty:
		return "empty"
	case outcomeDropped:
		return "dropped"
	case outcomeWorn:
		return "worn"
	case outcomeCarried:
		return "carried"
	default:
		return "?"
	}
}

// cellResult is one equipment cell of one row, resolved.
type cellResult struct {
	row, cell int
	class     itemClass
	name      string

	outcome cellOutcome
	slot    int
	// reason is populated for outcomeDropped and outcomeCarried: why the
	// cell did not end up worn.
	reason string
}

type classStats struct {
	used, worn, carried, dropped int
}

type report struct {
	totalRows int
	namedRows int
	classes   [3]classStats
	// histogram is the final worn set's own occupancy, summed over every
	// row: index 0 unused, slots 1..data.EquipSlots counted.
	histogram  [data.EquipSlots + 1]int
	collisions int
	anomalies  []cellResult
	samples    []rowSample
}

// rowSample is one ORDINARY row — no dropped cell, no carried cell, no
// destination collision — kept in full as an illustration of what a clean
// resolution looks like (plan D-7).
type rowSample struct {
	row   int
	cells []cellResult
}

// sampleSize is D-7's own "a handful" — few enough that the shipped names it
// prints stay an illustration rather than a transcription of the collection.
const sampleSize = 5

// itemsOK, shieldItemsOK and armorItemsOK RESTATE pkg/mapload.Table's own
// unexported items/shieldItems/armorItems: a table missing Shapes or
// Materials resolves NOTHING of any class, and a table missing one of
// Weapons/Shields/Armors resolves nothing of that ONE class alone,
// independently of the other two. They are restated rather than reused for
// cellWeapon's own reason above — the methods are unexported.
func itemsOK(t *mapload.Table) bool {
	return t != nil && t.Shapes != nil && t.Materials != nil && t.Weapons != nil
}
func shieldItemsOK(t *mapload.Table) bool {
	return t != nil && t.Shapes != nil && t.Materials != nil && t.Shields != nil
}
func armorItemsOK(t *mapload.Table) bool {
	return t != nil && t.Shapes != nil && t.Materials != nil && t.Armors != nil
}

// twoHanded RESTATES pkg/mapload/spawn.go's own unexported twoHanded:
// whether w's own Weapons row states a Hands column of 2. A nil weapon, an
// absent Weapons collection and a row too short for the column all answer
// false — cellWeapon's own reason above for why this is restated rather
// than called.
func twoHanded(w *data.Weapon, t *mapload.Table) bool {
	if w == nil || t == nil || t.Weapons == nil {
		return false
	}
	row := int(w.Row)
	if row < 0 || row >= t.Weapons.Len() {
		return false
	}
	p := t.Weapons.EntryParams(row)
	if len(p) <= weaponHandsColumn {
		return false
	}
	return p[weaponHandsColumn] == 2
}

func resolveRow(row int, names []string, t *mapload.Table) (cells []cellResult, worn [data.EquipSlots]uint16, carried []uint16, collisions int) {
	cellName := func(i int) string {
		if i >= 0 && i < len(names) {
			return names[i]
		}
		return ""
	}

	slotOwner := map[int]int{} // slot -> index into cells of the cell that last claimed it
	claim := func(slot, idx int) {
		if _, taken := slotOwner[slot]; taken {
			collisions++
		}
		slotOwner[slot] = idx
	}

	// Cell 0: the weapon.
	weaponIdx := -1
	var weapon *data.Weapon
	if n := cellName(cellWeapon); n == "" {
		cells = append(cells, cellResult{row: row, cell: cellWeapon, class: classWeapon, outcome: outcomeEmpty})
	} else if !itemsOK(t) {
		cells = append(cells, cellResult{row: row, cell: cellWeapon, class: classWeapon, name: n,
			outcome: outcomeDropped, reason: "no Shapes, Materials or Weapons collection installed"})
	} else if w, err := data.ResolveWeapon(n, t.Shapes, t.Materials, t.Weapons); err != nil {
		cells = append(cells, cellResult{row: row, cell: cellWeapon, class: classWeapon, name: n,
			outcome: outcomeDropped, reason: err.Error()})
	} else if slot, ok := data.EquipSlotFor(w.Code); ok {
		worn[slot-1] = uint16(w.Code)
		weaponIdx = len(cells)
		weapon = &w
		claim(slot, weaponIdx)
		cells = append(cells, cellResult{row: row, cell: cellWeapon, class: classWeapon, name: n,
			outcome: outcomeWorn, slot: slot})
	} else {
		cells = append(cells, cellResult{row: row, cell: cellWeapon, class: classWeapon, name: n,
			outcome: outcomeDropped, reason: "resolved but its own code names no equip slot"})
	}

	if n := cellName(cellShield); n == "" {
		cells = append(cells, cellResult{row: row, cell: cellShield, class: classShield, outcome: outcomeEmpty})
	} else if !shieldItemsOK(t) {
		cells = append(cells, cellResult{row: row, cell: cellShield, class: classShield, name: n,
			outcome: outcomeDropped, reason: "no Shapes, Materials or Shields collection installed"})
	} else if s, err := data.ResolveShield(n, t.Shapes, t.Materials, t.Shields); err != nil {
		cells = append(cells, cellResult{row: row, cell: cellShield, class: classShield, name: n,
			outcome: outcomeDropped, reason: err.Error()})
	} else {
		if worn[cellWeapon] != 0 && twoHanded(weapon, t) {
			carried = append(carried, worn[cellWeapon])
			worn[cellWeapon] = 0
			delete(slotOwner, cellWeapon+1)
			if weaponIdx >= 0 {
				cells[weaponIdx].outcome = outcomeCarried
				cells[weaponIdx].reason = "FR-7: a two-handed weapon is displaced when the shield takes slot 2"
			}
		}
		if slot, ok := data.EquipSlotFor(s.Code); ok {
			worn[slot-1] = uint16(s.Code)
			claim(slot, len(cells))
			cells = append(cells, cellResult{row: row, cell: cellShield, class: classShield, name: n,
				outcome: outcomeWorn, slot: slot})
		} else {
			cells = append(cells, cellResult{row: row, cell: cellShield, class: classShield, name: n,
				outcome: outcomeDropped, reason: "resolved but its own code names no equip slot"})
		}
	}

	// Cells 2..9: the armour.
	for i := cellArmorFrom; i < cellCount; i++ {
		n := cellName(i)
		switch {
		case n == "":
			cells = append(cells, cellResult{row: row, cell: i, class: classArmor, outcome: outcomeEmpty})
		case !armorItemsOK(t):
			cells = append(cells, cellResult{row: row, cell: i, class: classArmor, name: n,
				outcome: outcomeDropped, reason: "no Shapes, Materials or Armors collection installed"})
		default:
			a, err := data.ResolveArmor(n, t.Shapes, t.Materials, t.Armors)
			if err != nil {
				cells = append(cells, cellResult{row: row, cell: i, class: classArmor, name: n,
					outcome: outcomeDropped, reason: err.Error()})
				continue
			}
			if slot, ok := data.EquipSlotFor(a.Code); ok {
				worn[slot-1] = uint16(a.Code)
				claim(slot, len(cells))
				cells = append(cells, cellResult{row: row, cell: i, class: classArmor, name: n,
					outcome: outcomeWorn, slot: slot})
			} else {
				carried = append(carried, uint16(a.Code))
				cells = append(cells, cellResult{row: row, cell: i, class: classArmor, name: n,
					outcome: outcomeCarried,
					reason:  fmt.Sprintf("FR-6: armour Slot column reads %d, not 1..%d", a.Slot, data.EquipSlots)})
			}
		}
	}

	return cells, worn, carried, collisions
}

// scan walks every row of t.Humans and resolves its ten cells. A nil table
// or a nil Humans collection answers the empty report rather than panicking:
// there is nothing to scan, not a scan that failed.
func scan(t *mapload.Table) report {
	var rep report
	if t == nil || t.Humans == nil {
		return rep
	}
	for i := 1; i < t.Humans.Len(); i++ {
		cells, worn, _, collisions := resolveRow(i, t.Humans.EntryStrings(i), t)

		rep.totalRows++
		rep.collisions += collisions

		named := false
		clean := collisions == 0
		for _, c := range cells {
			switch c.outcome {
			case outcomeEmpty:
				continue
			case outcomeWorn:
				named = true
				rep.classes[c.class].used++
				rep.classes[c.class].worn++
			case outcomeCarried:
				named, clean = true, false
				rep.classes[c.class].used++
				rep.classes[c.class].carried++
				rep.anomalies = append(rep.anomalies, c)
			case outcomeDropped:
				named, clean = true, false
				rep.classes[c.class].used++
				rep.classes[c.class].dropped++
				rep.anomalies = append(rep.anomalies, c)
			}
		}
		if named {
			rep.namedRows++
		}
		for slot := 1; slot <= data.EquipSlots; slot++ {
			if worn[slot-1] != 0 {
				rep.histogram[slot]++
			}
		}
		if clean && named && len(rep.samples) < sampleSize {
			rep.samples = append(rep.samples, rowSample{row: i, cells: cells})
		}
	}
	return rep
}

// printReport writes rep as plain text: the counts first, then the two
// correctness checks, then every anomaly in full, then the bounded sample of
// ordinary rows (plan D-7). It writes to out alone — no file, anywhere.
func printReport(out io.Writer, rep report) {
	fmt.Fprintf(out, "wearcheck: %d Humans rows scanned, %d name at least one item\n\n",
		rep.totalRows, rep.namedRows)

	fmt.Fprintln(out, "per cell class (FR-1: weapon=cell 0, shield=cell 1, armour=cells 2..9):")
	fmt.Fprintln(out, "  class    used  worn  carried  dropped")
	for _, c := range []itemClass{classWeapon, classShield, classArmor} {
		s := rep.classes[c]
		fmt.Fprintf(out, "  %-7s  %-4d  %-4d  %-7d  %-4d\n", c, s.used, s.worn, s.carried, s.dropped)
	}

	fmt.Fprintln(out, "\nslot histogram (pieces landed in the final worn set, by slot):")
	for slot := 1; slot <= data.EquipSlots; slot++ {
		fmt.Fprintf(out, "  slot %2d: %d\n", slot, rep.histogram[slot])
	}

	fmt.Fprintf(out, "\ndestination collisions (two cells of one row resolving to the same slot): %d\n",
		rep.collisions)

	fmt.Fprintf(out, "\nanomalies — every dropped or carried cell, in full (%d):\n", len(rep.anomalies))
	if len(rep.anomalies) == 0 {
		fmt.Fprintln(out, "  none")
	}
	for _, a := range rep.anomalies {
		fmt.Fprintf(out, "  row=%-5d cell=%-2d %-7s name=%-40q %-7s %s\n",
			a.row, a.cell, a.class, a.name, a.outcome, a.reason)
	}

	fmt.Fprintf(out, "\nordinary rows — a bounded sample of %d (the rest are shipped game data this "+
		"repository commits none of):\n", len(rep.samples))
	for _, s := range rep.samples {
		fmt.Fprintf(out, "  row %d:\n", s.row)
		for _, c := range s.cells {
			if c.outcome == outcomeEmpty {
				continue
			}
			fmt.Fprintf(out, "    cell=%-2d %-7s name=%-40q -> %-7s slot=%d\n",
				c.cell, c.class, c.name, c.outcome, c.slot)
		}
	}
}

// castSpellMark is the attachment's own leading literal, tested as a
// substring the way this file's -spells mode finds a candidate cell worth
// resolving at all — a cheap pre-filter over every trailing string in both
// collections, before the one real parse (data.ResolveWeapon) runs.
const castSpellMark = "{castSpell="

// spellToken is one castSpell= attachment this install's Humans or Units
// rows name, resolved as far as it goes.
type spellToken struct {
	// source and row say WHERE this token was found: "Humans" or "Units",
	// and that collection's own one-based row index (spec AC-1, "measured,
	// not assumed" — a reader has to be able to find the row again).
	source string
	row    int
	// cell is the whole equipment-cell string, attachment and all, exactly
	// as the row states it.
	cell string

	weaponErr string

	// spellName and spellPower are the resolved Weapon's own pair (FR-1a):
	// the token AS WRITTEN, underscores intact, and the level. Set only
	// when weaponErr is empty.
	spellName  string
	spellPower int32

	// resolved and spellID are FR-1b's own lookup, mapload.SpellIDByToken
	// over spellName: whether the installed Spells collection carries a row
	// this token matches, and that row's own id when it does.
	resolved bool
	spellID  uint16
}

// scanSpells walks EVERY trailing equipment string of t's Humans collection
// and t's Units collection — the two definition tiers UNIT-EQUIP-005 and the
// chargen literal (HERO-START-039) both draw a spell-carrying weapon's name
// from — and resolves every one that carries castSpellMark (spec AC-1).
//
// BOTH COLLECTIONS, NOT HUMANS ALONE, unlike scan above: UNIT-EQUIP-005
// names two Units rows (Catapult, Ballista) that carry the attachment, and a
// units-band placement's weapon reaches an entity by the SAME route a
// humans-band one does — a probe that only walked Humans would silently
// miss exactly the population 0139's own plan names as a risk.
func scanSpells(t *mapload.Table) []spellToken {
	if t == nil {
		return nil
	}
	var out []spellToken
	for _, coll := range [...]struct {
		name string
		c    data.Collection
	}{{"Humans", t.Humans}, {"Units", t.Units}} {
		if coll.c == nil {
			continue
		}
		for i := 1; i < coll.c.Len(); i++ {
			for _, n := range coll.c.EntryStrings(i) {
				if !strings.Contains(n, castSpellMark) {
					continue
				}
				tok := spellToken{source: coll.name, row: i, cell: n}
				if !itemsOK(t) {
					tok.weaponErr = "no Shapes, Materials or Weapons collection installed"
				} else if w, err := data.ResolveWeapon(n, t.Shapes, t.Materials, t.Weapons); err != nil {
					tok.weaponErr = err.Error()
				} else {
					tok.spellName, tok.spellPower = w.SpellName, w.SpellPower
					tok.spellID, tok.resolved = mapload.SpellIDByToken(t, w.SpellName)
				}
				out = append(out, tok)
			}
		}
	}
	return out
}

// printSpellReport writes every token scanSpells found, in the order found,
// and a tally at the end. It returns an error naming the unresolved count
// when one or more tokens did not reach a Spells row — AC-1's own claim is
// that EVERY one does, so this mode is a pass/fail gate as well as a report:
// a non-zero exit is what a caller (or verification.md's own record of
// running it) can point at.
func printSpellReport(out io.Writer, toks []spellToken) error {
	fmt.Fprintf(out, "wearcheck -spells: %d castSpell= token(s) found\n\n", len(toks))
	resolved, unresolved := 0, 0
	for _, tk := range toks {
		switch {
		case tk.weaponErr != "":
			unresolved++
			fmt.Fprintf(out, "  %-6s row=%-4d %-55q -> WEAPON DID NOT RESOLVE: %s\n",
				tk.source, tk.row, tk.cell, tk.weaponErr)
		case !tk.resolved:
			unresolved++
			fmt.Fprintf(out, "  %-6s row=%-4d %-55q token=%-18q level=%-4d -> NO SPELLS ROW\n",
				tk.source, tk.row, tk.cell, tk.spellName, tk.spellPower)
		default:
			resolved++
			fmt.Fprintf(out, "  %-6s row=%-4d %-55q token=%-18q level=%-4d -> spell id %d\n",
				tk.source, tk.row, tk.cell, tk.spellName, tk.spellPower, tk.spellID)
		}
	}
	fmt.Fprintf(out, "\n%d of %d token(s) resolved to a Spells row, %d did not\n",
		resolved, len(toks), unresolved)
	if unresolved > 0 {
		return fmt.Errorf("wearcheck -spells: %d of %d castSpell= token(s) did not resolve to a Spells row",
			unresolved, len(toks))
	}
	return nil
}
