package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/databin"
	"againrom/pkg/mapload"
	"againrom/pkg/vfs"
)

// dataBinPath is where the definition table lives inside its archive, below the
// archive's own identity segment. The identity is derived from the host path the
// caller passed, so no install path is spelled anywhere here (golden rule 3).
const dataBinPath = "data/data.bin"

// A PARSED collection satisfies the interface the definition tier searches
// through. This assertion belongs here because this is the only place in the
// tree that holds both packages: pkg/data may not import the format tier that
// parses the table, so nothing below this tool can state the relationship, and
// without it the two halves would only meet at a caller's compile.
var _ data.Collection = (*databin.Collection)(nil)

// table is one parsed definition table beside the two figures the walk's own
// claim rests on: what the archive node holds, and what the walk read.
type table struct {
	file     *databin.File
	address  string
	nodeSize int
}

// openDataBin resolves the table out of the archive and parses it. Reading is
// the only IO; the walk is a pure function of the bytes.
func openDataBin(archivePath string) (*table, error) {
	identity, err := vfs.Identity(archivePath)
	if err != nil {
		return nil, err
	}
	fsys, err := vfs.Open([]string{archivePath}, nil)
	if err != nil {
		return nil, err
	}
	address := identity + "/" + dataBinPath
	b, err := fsys.ReadFile(address)
	if err != nil {
		return nil, err
	}
	f, err := databin.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", address, err)
	}
	return &table{file: f, address: address, nodeSize: len(b)}, nil
}

// runDataBin is the verb: the table's census, and — when a map is named — every
// one of its placements resolved at a chosen difficulty.
func runDataBin(args []string, w io.Writer) error {
	t, err := openDataBin(args[0])
	if err != nil {
		return err
	}
	if err := reportTable(w, t); err != nil {
		return err
	}
	if len(args) == 1 {
		return nil
	}

	diff := mapload.DifficultyNormal
	if len(args) == 3 {
		n, err := strconv.Atoi(args[2])
		if err != nil {
			return fmt.Errorf("%w: difficulty must be a number, got %q", errUsage, args[2])
		}
		diff = mapload.Difficulty(n)
	}
	return reportMap(w, t, args[1], diff)
}

func reportTable(w io.Writer, t *table) error {
	f := t.file
	left := t.nodeSize - f.Consumed
	if _, err := fmt.Fprintf(w, "%s: %d of %d byte(s) consumed, %d left over\n",
		t.address, f.Consumed, t.nodeSize, left); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%-12s %8s %8s %8s\n", "collection", "written", "titles", "params"); err != nil {
		return err
	}
	for i := range f.Collections {
		c := &f.Collections[i]
		written, params := 0, 0
		for j := range c.Entries {
			if c.OneBased && j == 0 {
				continue // the reserved entry is allocated, never written
			}
			written++
			if len(c.Entries[j].Params) > 0 {
				params++
			}
		}
		if _, err := fmt.Fprintf(w, "%-12s %8d %8d %8d\n", c.ID, written, len(c.Titles), params); err != nil {
			return err
		}
	}
	if err := reportUnitWidths(w, f.Collection(databin.Units)); err != nil {
		return err
	}
	return reportHumansHealth(w, f.Collection(databin.Humans))
}

// reportUnitWidths says which Units rows carry a parameter array and how wide
// each is — as a histogram plus the row indices, so a single row out of step
// with the rest is a number a reader can go and look at rather than a summary
// that hides it.
func reportUnitWidths(w io.Writer, c *databin.Collection) error {
	widths := make(map[int]int)
	var rows []int
	for i := range c.Entries {
		if n := len(c.Entries[i].Params); n > 0 {
			widths[n]++
			rows = append(rows, i)
		}
	}
	keys := make([]int, 0, len(widths))
	for k := range widths {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	if _, err := fmt.Fprintf(w, "Units parameter widths:"); err != nil {
		return err
	}
	for _, k := range keys {
		if _, err := fmt.Fprintf(w, " %d x%d", k, widths[k]); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "\nUnits rows with parameters (%d):", len(rows)); err != nil {
		return err
	}
	for _, i := range rows {
		if _, err := fmt.Fprintf(w, " %d", i); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}

func reportHumansHealth(w io.Writer, c *databin.Collection) error {
	var rows int
	var columnSum, derivedSum int64
	var above, below, equal int
	var zeroColumn, zeroDerived int

	haveRatio := false
	var bestRatio float64
	var bestIndex int
	var bestName string
	var bestDerived, bestColumn int32

	for i := range c.Entries {
		if c.OneBased && i == 0 {
			continue // the reserved entry is allocated, never written
		}
		e := &c.Entries[i]
		if len(e.Params) == 0 {
			continue // a written-but-paramless row states no statistics
		}
		d, err := data.NewHumanDef(e.Name, e.Params)
		if err != nil {
			continue // too short to stream; refused everywhere a map could place it
		}

		rows++
		column, derived := d.HealthMax, d.DerivedMaximum()
		columnSum += int64(column)
		derivedSum += int64(derived)
		switch {
		case derived > column:
			above++
		case derived < column:
			below++
		default:
			equal++
		}
		if derived == 0 {
			zeroDerived++
		}
		if column == 0 {
			zeroColumn++
			continue
		}
		if ratio := float64(derived) / float64(column); !haveRatio || ratio > bestRatio {
			haveRatio, bestRatio = true, ratio
			bestIndex, bestName, bestDerived, bestColumn = i, e.Name, derived, column
		}
	}

	if _, err := fmt.Fprintf(w, "humans health: %d row(s), column sum %d, derived sum %d\n",
		rows, columnSum, derivedSum); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  derived above column %d, below %d, equal %d; zero column %d, zero derived %d\n",
		above, below, equal, zeroColumn, zeroDerived); err != nil {
		return err
	}
	if !haveRatio {
		_, err := fmt.Fprintln(w, "  largest ratio: none - every row's column is 0")
		return err
	}
	label := quote(bestName)
	if bestName == "" {
		label = fmt.Sprintf("entry %d, written and unnamed", bestIndex)
	}
	_, err := fmt.Fprintf(w, "  largest ratio: %s derived %d against column %d\n", label, bestDerived, bestColumn)
	return err
}

// domainNames are the three movement domains a world's entity can be in, by the
// numeric value the simulation gives each. The world constructor admits no other
// value, so this table is total over anything that can arrive here.
//
// A name is spelled here rather than asked of the simulation tier, which this
// tool does not import and gains no reason to: what a domain is CALLED is a
// property of this report, and the report's job is to say which of the three a
// placement landed in, not to be the place their names live.
var domainNames = [...]string{"ground", "ghost", "air"}

func domainName(d uint8) string {
	if int(d) >= len(domainNames) {
		return fmt.Sprintf("domain(%d)", d)
	}
	return domainNames[d]
}

// reportMap resolves every placement of one map against the table and prints the
// arm it took, the entry it reached, and the health maximum and movement domain
// the world it builds carries.
//
// The arm comes from the resolution and the health and the domain from the
// WORLD, not from a second application of the arithmetic or a second call to the
// mapping: a report that recomputed either would agree with itself rather than
// with what a player's world holds. That is why the domain census below counts
// entities and not resolutions — it is a witness of the world, and a join that
// failed between the definition and the entity would show here.
func reportMap(w io.Writer, t *table, mapPath string, diff mapload.Difficulty) error {
	b, err := os.ReadFile(mapPath)
	if err != nil {
		return err
	}
	m, err := alm.Open(b)
	if err != nil {
		return fmt.Errorf("%s: %w", mapPath, err)
	}

	// Buildings joins the two collections a unit placement searches: it is what
	// a placed STRUCTURE resolves against, and this tool is the one place in the
	// tree that holds a map and a parsed table at once.
	// The three item collections ride here too, or the report would witness a
	// world no player has: a scenario human names his own equipment, and a table
	// that cannot say what those names are worth arms nobody. This tool's whole
	// claim is that its rows say what a player's world holds.
	tbl := &mapload.Table{
		Units:     t.file.Collection(databin.Units),
		Humans:    t.file.Collection(databin.Humans),
		Buildings: t.file.Collection(databin.Buildings),
		Shapes:    t.file.Collection(databin.Shapes),
		Materials: t.file.Collection(databin.Materials),
		Weapons:   t.file.Collection(databin.Weapons),
	}
	// BEFORE the world, deliberately. FromALMWith refuses an entry whose damage
	// selector takes an arm the loader does not model, and the structure census
	// depends on neither the world nor the difficulty — sequenced after it, one
	// refused UNIT entry would take a whole corpus run's structure figures with
	// it, for a reason that has nothing to do with structures.
	if err := reportStructures(w, mapPath, m, tbl); err != nil {
		return err
	}

	world, err := mapload.FromALMWith(m, tbl, diff)
	if err != nil {
		return fmt.Errorf("%s: %w", mapPath, err)
	}
	ents := world.Entities()
	if len(ents) != len(m.Units) {
		return fmt.Errorf("%s: the world holds %d entities for %d placements", mapPath, len(ents), len(m.Units))
	}

	if _, err := fmt.Fprintf(w, "\n%s: %d placement(s) at difficulty %d\n",
		mapPath, len(m.Units), int32(diff)); err != nil {
		return err
	}

	census := make(map[mapload.Arm][2]int) // {taken, reached an entry}
	domains := make(map[uint8]int)
	// The SPEED ALPHABET this map's placements actually reach a world with, which
	// is the figure the movement rate has to be checked against: a column value
	// nothing places is not one a player ever meets. Counted off the ENTITIES and
	// never off the table, for the reason the domain census is — a join that
	// failed between the definition and the entity would show here.
	speeds := make(map[int32]int)
	// The combat report's rows are collected HERE, in the walk that already
	// resolves every placement, rather than in a second walk of its own — a
	// second resolution could disagree with this one and the report would be
	// describing a world nobody built.
	rows := make([]templateRow, 0, len(m.Units))
	units, humans := tbl.Units, tbl.Humans
	for i, u := range m.Units {
		r := mapload.Resolve(u, tbl)
		c := census[r.Arm]
		c[0]++
		if r.Found() {
			c[1]++
		}
		census[r.Arm] = c
		d := uint8(ents[i].Domain)
		domains[d]++
		speeds[ents[i].Speed]++

		// Only the units arm reaching an entry yields a definition, which is
		// the same test the loader itself applies — a humans entry is not a
		// unit definition however well the search did.
		row := templateRow{index: i, classID: u.ClassID, classSubID: u.ClassSubID,
			arm: r.Arm.String(), entryIndex: r.Index,
			charge: ents[i].AttackCharge, relax: ents[i].AttackRelax,
			toHit: ents[i].ToHit, defence: ents[i].Defence, absorb: ents[i].Absorption,
			dmgBase: ents[i].DamageBase, dmgSpread: ents[i].DamageSpread,
			alwaysHits: ents[i].AlwaysHits, sight: ents[i].ScanRange}
		// The index is a subscript into the collection the ARM searched, and the
		// two collections are different lengths — a humans index routinely runs
		// past the units collection's end, so the branch has to come before the
		// subscript and not after it.
		if r.Found() {
			row.resolved = true
			if r.Arm == mapload.ArmUnits {
				row.entry = units.EntryName(r.Index)
			} else {
				row.entry = humans.EntryName(r.Index)
			}
		}
		rows = append(rows, row)
		// THE OWNER IS READ OFF THE ENTITY, beside the health and the speed this
		// line already reads off it: the map's own record carries it whether or
		// not the placement resolved, so it is not gated on r.Found() the way the
		// health and the eight are. A placement the map gives no owner reads 0,
		// which the field prints as the 0 it is rather than blanking it, because 0
		// names no roster entry and is a fact about the placement worth showing
		// rather than hiding.
		if _, err := fmt.Fprintf(w, "  %5d key %#04x/%#04x  arm %-9s entry %4d  owner %4d  healthMax %5d  speed %4d  domain %s\n",
			i, uint16(u.ClassID), u.ClassSubID, r.Arm, r.Index, ents[i].Owner, ents[i].MaxHP, ents[i].Speed,
			domainName(d)); err != nil {
			return err
		}
	}

	for _, a := range []mapload.Arm{mapload.ArmNPC, mapload.ArmServerID, mapload.ArmHumansByType, mapload.ArmUnits} {
		c := census[a]
		if _, err := fmt.Fprintf(w, "  arm %-9s taken %5d  reached an entry %5d\n", a, c[0], c[1]); err != nil {
			return err
		}
	}

	// All three are printed, zeros included, so a map placing no flyer says so
	// rather than saying nothing — and the three sum to the placement count,
	// which is what makes the census checkable against the line above it.
	total := 0
	for d := range domainNames {
		n := domains[uint8(d)]
		total += n
		if _, err := fmt.Fprintf(w, "  domain %-8s %5d\n", domainNames[d], n); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "  domain %-8s %5d of %d placement(s)\n", "total", total, len(m.Units)); err != nil {
		return err
	}

	// The speed alphabet, ASCENDING, so one map prints one line per value in one
	// order however a Go map ranges — nothing nondeterministic reaches this
	// output. The count of distinct values is printed beside them, which is the
	// figure a corpus run is summed over.
	values := make([]int32, 0, len(speeds))
	for s := range speeds {
		values = append(values, s)
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	for _, s := range values {
		if _, err := fmt.Fprintf(w, "  speed %5d  %5d placement(s)\n", s, speeds[s]); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "  speed %5s  %5d distinct value(s)\n", "--", len(values)); err != nil {
		return err
	}
	if err := reportCombat(w, mapPath, m, tbl, rows, diff); err != nil {
		return err
	}

	// The character sheets, built from the same ents this walk already holds
	// — a second call to World.Entities() would still witness the same world,
	// but copying the one slice already in hand keeps this the walk's only read
	// of it. sheetEntity NAMES NO TYPE OF pkg/sim (sheet.go's own doc): e's
	// type is inferred from ents, and every field on the right of a colon below
	// is read off it by name, never by naming sim.Entity itself.
	sheetEnts := make([]sheetEntity, len(ents))
	for i, e := range ents {
		sheetEnts[i] = sheetEntity{
			hp: e.HP, maxHP: e.MaxHP,
			mana: e.Mana, maxMana: e.MaxMana,
			dmgBase: e.DamageBase, dmgSpread: e.DamageSpread,
			absorption: e.Absorption,
			toHit:      e.ToHit, defence: e.Defence,
			scanRange: e.ScanRange,
			speed:     e.Speed,
			xpValue:   e.XPValue,
		}
	}
	return reportSheets(w, m, tbl, sheetEnts)
}

// templateRow is one placement's row in the combat-numbers report: which
// placement it is, what it resolved to, and the eight numbers the world gave it.
//
// It is a type of THIS TOOL and not the simulation's entity, and that is
// deliberate rather than incidental: this package's row in the import DAG does
// not include the simulation tier, so the report is assembled out of the numbers
// themselves and this file never names it. The values are still READ OFF THE
// WORLD and never recomputed here — a report that applied the difficulty a
// second time would agree with itself rather than with what a player's world
// holds.
type templateRow struct {
	index      int
	classID    int16
	classSubID uint16

	// resolved says the placement reached a UNIT DEFINITION, which is the only
	// thing that can carry the eight. It is a field of its own and not "entry is
	// empty": a shipped table carries nameless written entries, so an empty name
	// and no entry at all are two different outcomes that must not print alike.
	// arm names which of the four paths the placement took either way, so an
	// unresolved row says WHY.
	resolved   bool
	entry      string
	entryIndex int
	arm        string

	charge, relax      int32
	toHit, defence     int32
	absorb             int32
	dmgBase, dmgSpread int32
	alwaysHits         bool

	// sight is how far the placement sees, in whole cells. It is NOT one of the
	// eight and is read off the world beside them, on the same terms: the value a
	// player's world holds, whichever of the three arms the placement took,
	// rather than a second reading of the table. It is a byte there and is
	// carried as one here, so a row stating a column outside a byte prints what
	// the world carries and not what the table said.
	sight uint8
}

// reportCombat prints one row per placement of the named map: the class key
// pair, the entry the placement resolved to or that it resolved to none, and the
// eight numbers a fight reads.
//
// ITS HEADING SAYS WHAT THESE NUMBERS ARE NOT, and that disclosure is the reason
// the report is safe to read. Equipment ASSIGNS over the cadence pair and ADDS to
// the damage pair, the absorption and the defence, and it moves the reach, on a
// substantial minority of the classes the game ships — none of which this tree
// models. A stat block printed without that sentence invites exactly one reading,
// that the tree fights the way the class does, and that reading is false.
//
// The two cadence numbers are called out separately because they are worse than
// unverified: the game's own unit information panel does not display them, so no
// run of any kind — this one included — can be compared against anything for
// them.
//
// THE SIGHT COLUMN IS NOT ONE OF THE EIGHT and carries no such caveat: it is a
// column of both bands, equipment does not move it, and it is what the
// placement's own march is seeded with. It rides on this report rather than on
// one of its own because it is a per-placement number read off the same world,
// and a second table would be a second walk of the same map.
func reportCombat(w io.Writer, mapPath string, m *alm.Map, tbl *mapload.Table,
	rows []templateRow, diff mapload.Difficulty) error {
	if _, err := fmt.Fprintf(w, "\n%s: template combat numbers, %d placement(s) at difficulty %d\n",
		mapPath, len(m.Units), int32(diff)); err != nil {
		return err
	}
	// THE HEADING SPLITS BY BAND, because the two bands are no longer alike and
	// one warning over both is worse than the old blanket one: a reader who
	// trusts a blanket "equipment is not applied" reads an armed person's damage
	// as his template's. A creature's equipment cell is still unread, so the
	// caveat stands for that band and is stated for it alone.
	if _, err := fmt.Fprint(w,
		"  READ THE BAND COLUMN: THE TWO ARE NOT ALIKE.\n"+
			"  A CREATURE row is its TEMPLATE'S NUMBERS AND EQUIPMENT IS NOT APPLIED. Such a class\n"+
			"  fights at its weapon's cadence and at a damage, absorption, defence and reach its\n"+
			"  equipment moves; none of that is read here, so the row is NOT what it fights at.\n"+
			"  A PERSON row IS ARMED: his equipment is resolved and his weapon's damage, to-hit,\n"+
			"  defence and cadence are already in these numbers, so the row IS what he fights at,\n"+
			"  less the armour this build does not model.\n"+
			"  The charge and relax columns are on no panel the game displays and are compared\n"+
			"  against nothing.\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  %5s %13s %8s %7s %6s %6s %8s %7s %8s %10s %7s %6s  %s\n",
		"index", "key", "band", "charge", "relax", "toHit", "defence", "absorb",
		"dmgBase", "dmgSpread", "always", "sight", "entry"); err != nil {
		return err
	}
	for _, r := range rows {
		// The entry column is LAST on the line, because an entry name is the
		// install's own bytes and may hold anything at all — including a space,
		// which would otherwise walk a column into the next one.
		if _, err := fmt.Fprintf(w, "  %5d %#04x/%#04x %8s %7d %6d %6d %8d %7d %8d %10d %7v %6d  %s\n",
			r.index, uint16(r.classID), r.classSubID, bandOf(r),
			r.charge, r.relax, r.toHit, r.defence, r.absorb, r.dmgBase, r.dmgSpread,
			r.alwaysHits, r.sight, describeEntry(r)); err != nil {
			return err
		}
	}
	return nil
}

// bandOf names which of the two kinds of row this is, which is what the heading
// now splits on.
//
// A row that reached NO entry is neither: it carries the constructor's numbers
// and there is no equipment question to answer about it, so calling it a
// creature would extend the caveat to a row the caveat is not about.
func bandOf(r templateRow) string {
	if !r.resolved {
		return "-"
	}
	if r.arm == mapload.ArmUnits.String() {
		return "creature"
	}
	return "person"
}

// describeEntry renders what a placement resolved to. A placement that reached
// NO entry is said so in words, and named with the arm it took, so it reads as
// unresolved rather than as a row that happens to be at the constructor's
// numbers — which is a thing a reader cannot tell from the numbers alone,
// because those numbers are a real class's numbers too.
//
// It used to say "no unit definition" for every arm but one, which stopped being
// true when the humans band gained a definition of its own: a person's row is
// where his numbers come from, and calling it nothing would have made this
// report deny the very thing it prints.
func describeEntry(r templateRow) string {
	if !r.resolved {
		return "none - the " + r.arm + " arm reached no entry"
	}
	if r.entry == "" {
		return fmt.Sprintf("entry %d, written and unnamed", r.entryIndex)
	}
	return quote(r.entry)
}

// reportStructures prints the structure pass's census over one map and table, at
// both levels, on two lines.
//
// The figures come from the SAME derivation the world's plane comes out of, so
// they describe what a player would walk on and not a second reading of the
// contract written beside it. What they are for is disagreement: a census
// computed from this contract cannot test this contract, but it can be set
// against independently published figures for the same pass, and a difference
// there is worth more than any agreement here.
//
// It prints counts and nothing else — no class name, no entry, no cell. Every
// number is a measurement over the caller's own install and none of it is game
// content.
func reportStructures(w io.Writer, mapPath string, m *alm.Map, tbl *mapload.Table) error {
	c := mapload.StructureCensus(m, tbl)
	if _, err := fmt.Fprintf(w, "\n%s: %d placed structure(s)\n"+
		"  structures: %d resolved, %d unresolved, %d short, %d zero-extent\n",
		mapPath, len(m.Objects), c.Resolved, c.Unresolved, c.Short, c.ZeroExtent); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  cells: %d attached (%d closed, %d opened), %d dropped, %d refused, %d abandoned\n",
		c.Attached, c.Closed, c.Opened, c.Dropped, c.Refused, c.Abandoned); err != nil {
		return err
	}
	return reportStructureField(w, m, tbl)
}

// reportStructureField prints the SEEDED SCRIPT FIELD structure+0x42 over one
// map's placements (1033 B3): how many were seeded, how many carry zero, the
// range, and the distinct values when there are few enough to name.
//
// Check opcode 21 and instant opcode 26 read and write that field, and
// mapload.Structures seeds it from the definition table's healthMax position
// (`ALM-CLS-053`). The line is read off mapload.Structures itself rather than
// recomputed here: a reporter carrying its own copy of the seeding rule would be
// testing that copy and not the rule a player's world is built from.
//
// It reports a RANGE and a population rather than one number, because the
// field's meaning is Unknown (`TRIG-CHECK-053`) and a single figure invites a
// label. A map whose whole population is one value says so, which is the case
// this line exists to make visible: mission 101's placements are all 1, against
// twelve script nodes testing the field below 1.
//
// A map with no placements prints the line with a zero population rather than
// printing nothing, so a reader can tell "no structures" from "this instrument
// said nothing".
func reportStructureField(w io.Writer, m *alm.Map, tbl *mapload.Table) error {
	structs := mapload.Structures(m, tbl)
	seen := map[int]int{}
	zeros, min, max := 0, 0, 0
	for i, st := range structs {
		v := int(st.Field42)
		seen[v]++
		if v == 0 {
			zeros++
		}
		if i == 0 || v < min {
			min = v
		}
		if i == 0 || v > max {
			max = v
		}
	}
	values := make([]int, 0, len(seen))
	for v := range seen {
		values = append(values, v)
	}
	sort.Ints(values)
	list := ""
	if len(values) > 0 && len(values) <= 8 {
		list = ", values"
		for _, v := range values {
			list += fmt.Sprintf(" %dx%d", v, seen[v])
		}
	}
	if _, err := fmt.Fprintf(w, "  script field +0x42: %d seeded, %d zero, range %d..%d%s\n",
		len(structs), zeros, min, max, list); err != nil {
		return err
	}
	return reportStructureFieldReaders(w, m, tbl)
}

// reportStructureFieldReaders names every script node that reads or writes the
// seeded field, the structure each one references, and that structure's seeded
// value. Check opcode 21 reads the field; instant opcode 26 writes it.
//
// The summary line above gives a population and says nothing about whether any
// node reaches it: a build seeding the wrong value and a build seeding the right
// one print the same summary whenever no trigger fires. This line separates
// them. Mission 101 prints twelve check nodes each naming a structure seeded to
// 1, against twelve authored comparisons against the constant 1, which is the
// arrangement under which an authored zero completed that mission's counter
// objective at tick 0.
//
// The selection and the values both come from mapload.StructureFieldRefs, which
// is the tier that owns the seed. A reporter carrying its own opcode list and
// its own copy of the seeding rule would be testing those copies.
func reportStructureFieldReaders(w io.Writer, m *alm.Map, tbl *mapload.Table) error {
	refs, err := mapload.StructureFieldRefs(m, tbl)
	return printStructureFieldReaders(w, refs, err)
}

// printStructureFieldReaders is the rendering, with the derivation already done,
// so each of its four line shapes can be witnessed without a map that produces
// it. Three of the four are unreached on the shipped campaign: every one of the
// 16 authored check-21 nodes resolves to a structure the map placed.
func printStructureFieldReaders(w io.Writer, refs []mapload.StructureFieldRef, err error) error {
	if err != nil {
		_, e := fmt.Fprintf(w, "    +0x42 readers: the script did not compile (%v)\n", err)
		return e
	}
	if len(refs) == 0 {
		_, e := fmt.Fprintf(w, "    +0x42 readers: none authored on this map\n")
		return e
	}
	for _, r := range refs {
		kind, verb := "check", "reads"
		if r.Writes {
			kind, verb = "instant", "writes"
		}
		switch {
		case !r.HasRef:
			_, err = fmt.Fprintf(w, "    %s node %d %s no structure: its reference did not resolve\n",
				kind, r.Node, verb)
		case !r.Placed:
			_, err = fmt.Fprintf(w, "    %s node %d %s structure %d, which this map did not place\n",
				kind, r.Node, verb, r.Ref)
		default:
			_, err = fmt.Fprintf(w, "    %s node %d %s structure %d, seeded +0x42 = %d\n",
				kind, r.Node, verb, r.Ref, r.Value)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
