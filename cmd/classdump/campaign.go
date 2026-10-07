package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/databin"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/vfs"
)

// campaignFlag selects the campaign census, matched as a literal first argument
// for the reason the other two verbs are: this tool's invocations are fixed
// argument shapes, not a flag set.
const campaignFlag = "-campaign"

// The two archives the census opens, and the entries it reads out of them. The
// names are the install's own file names; every address below is composed from
// the identity vfs derives from the host path, so no install path is spelled
// here (golden rule 3).
const (
	scenarioArchive = "scenario.res"
	worldArchive    = "world.res"
	npcEntry        = "npc.reg"
	mapExtension    = ".alm"
)

// campaign is everything the census resolves against: the two searched
// collections and the NPC lookup that answers the humans band's first rung.
type campaign struct {
	table *mapload.Table
	fsys  *vfs.FS

	// scenarioPrefix is the campaign container's identity and separator, so a
	// map address is that prefix and the entry's own name.
	scenarioPrefix string
}

// openCampaign opens both archives under root and loads what a placement
// resolves against.
//
// It opens the two host files by name under the root it is given. That root is a
// command-line argument and nothing here has a default, which is what keeps the
// tool honest about golden rule 3 while still being able to walk a whole install.
func openCampaign(root string) (*campaign, error) {
	scenarioPath := filepath.Join(root, scenarioArchive)
	worldPath := filepath.Join(root, worldArchive)
	fsys, err := vfs.Open([]string{scenarioPath, worldPath}, nil)
	if err != nil {
		return nil, err
	}
	scenarioID, err := vfs.Identity(scenarioPath)
	if err != nil {
		return nil, err
	}
	worldID, err := vfs.Identity(worldPath)
	if err != nil {
		return nil, err
	}

	tableAddr := worldID + "/" + dataBinPath
	b, err := fsys.ReadFile(tableAddr)
	if err != nil {
		return nil, err
	}
	f, err := databin.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", tableAddr, err)
	}

	npcAddr := scenarioID + "/" + npcEntry
	nb, err := fsys.ReadFile(npcAddr)
	if err != nil {
		return nil, err
	}
	r, err := reg.Parse(nb)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", npcAddr, err)
	}

	// The three item collections, the two equipment ones and the structure one
	// ride here beside the two a placement resolves against, because -regen
	// and 0137's own sheet report both build an actual WORLD out of a map
	// rather than only resolving its arms, and FromALMWith reads all eight.
	// Resolve reads none of the six, so the campaign census's own arm figures
	// are unmoved by their presence.
	//
	// ARMORS AND SHIELDS ARE NEW: without them a person this verb built wore no
	// armour and carried no shield whatever his row named — armorItems() and
	// shieldItems() (pkg/mapload/spawn.go) answer false for a table missing
	// either, on the SAME per-class rule that already governs a table missing
	// Weapons, so refusing to name them silently bared every person -regen or
	// the sheet report ever built. This verb's own claim is that its numbers
	// are a player's; a table that leaves two of a person's four equipment
	// slots unresolved is not that.
	return &campaign{
		table: &mapload.Table{
			Units:     f.Collection(databin.Units),
			Humans:    f.Collection(databin.Humans),
			NPC:       data.LoadNPCDefs(r),
			Buildings: f.Collection(databin.Buildings),
			Shapes:    f.Collection(databin.Shapes),
			Materials: f.Collection(databin.Materials),
			Weapons:   f.Collection(databin.Weapons),
			Armors:    f.Collection(databin.Armors),
			Shields:   f.Collection(databin.Shields),
		},
		fsys:           fsys,
		scenarioPrefix: scenarioID + "/",
	}, nil
}

// source is one map the census walks: where it came from, and its bytes.
type source struct {
	name string
	read func() ([]byte, error)
}

// sources is every map the census covers: the campaign container's entries
// first, in the container's own order, then the loose maps at the root, sorted.
//
// BOTH HALVES ARE THE CORPUS. The 28 embedded maps and the 10 loose ones are
// what the published arm figures are measured over, and a census of one half
// cannot be compared with them — which is exactly the gap this verb exists to
// close.
func (c *campaign) sources(root string) ([]source, error) {
	var out []source
	for _, e := range c.fsys.Entries() {
		if !strings.HasPrefix(e.Address, c.scenarioPrefix) {
			continue
		}
		name := strings.TrimPrefix(e.Address, c.scenarioPrefix)
		if !strings.EqualFold(filepath.Ext(name), mapExtension) {
			continue
		}
		addr := e.Address
		out = append(out, source{name: name, read: func() ([]byte, error) { return c.fsys.ReadFile(addr) }})
	}

	ents, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var loose []string
	for _, e := range ents {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), mapExtension) {
			continue
		}
		loose = append(loose, e.Name())
	}
	sort.Strings(loose)
	for _, name := range loose {
		path := filepath.Join(root, name)
		out = append(out, source{name: name, read: func() ([]byte, error) { return os.ReadFile(path) }})
	}
	return out, nil
}

// armTotals counts one map's placements by the arm each took, and how many of
// those reached an entry.
type armTotals struct {
	taken   [4]int
	reached [4]int
}

func (a *armTotals) add(r mapload.Resolution) {
	a.taken[r.Arm]++
	if r.Found() {
		a.reached[r.Arm]++
	}
}

func (a *armTotals) addAll(o armTotals) {
	for i := range a.taken {
		a.taken[i] += o.taken[i]
		a.reached[i] += o.reached[i]
	}
}

// The arm order the report prints in — the order the resolution ladder tries
// them, so the table reads as the ladder does.
var armOrder = [4]mapload.Arm{
	mapload.ArmUnits, mapload.ArmServerID, mapload.ArmNPC, mapload.ArmHumansByType,
}

// runCampaign is the verb: every shipped map's placements resolved by arm, and
// every map's drop cells, over one lawful install.
//
// ITS OWN CENSUS PRINTS COUNTS AND CELLS ONLY — no entry name, no string,
// no parameter value — so that much of its output is a measurement of this
// tree's loader rather than converted game data. THAT IS NO LONGER TRUE OF
// THE WHOLE VERB: named a map, it goes on to print that one map's NPC-arm
// detail and its full character sheets, and both of those DO name an entry
// and a definition row's own values — the same converted game data
// classdump's other verbs already print, on the same rule (never committed;
// golden rule 1).
//
// Any map that will not read, decode or walk is a failure of the whole run. A
// census whose denominator silently shrinks is worse than no census: the figure
// it is compared against is exhaustive over 38 maps, and a run that quietly
// covered 37 would agree with nothing and look like it did.
//
// Named a map, it follows the census with that map's NPC-arm placements read
// down BOTH routes at once — the registry's and the record's own —
// because a count cannot show that two routes disagree, and "the npc arm
// never reads its own definition id" is a claim about exactly that
// disagreement.
func runCampaign(root, only string, w io.Writer) error {
	c, err := openCampaign(root)
	if err != nil {
		return err
	}
	srcs, err := c.sources(root)
	if err != nil {
		return err
	}
	var detail *alm.Map

	fmt.Fprintf(w, "%-16s %8s %8s %8s %8s %8s   %s\n",
		"map", "type6", "units", "server-id", "npc", "humans", "drop cells")

	var total armTotals
	placements, maps, cells := 0, 0, 0
	for _, s := range srcs {
		b, err := s.read()
		if err != nil {
			return fmt.Errorf("%s: %w", s.name, err)
		}
		m, err := alm.Open(b)
		if err != nil {
			return fmt.Errorf("%s: %w", s.name, err)
		}
		var t armTotals
		for _, u := range m.Units {
			t.add(mapload.Resolve(u, c.table))
		}
		drops := mapload.DropCells(m)
		fmt.Fprintf(w, "%-16s %8d %8d %8d %8d %8d   %s\n", s.name, len(m.Units),
			t.taken[mapload.ArmUnits], t.taken[mapload.ArmServerID],
			t.taken[mapload.ArmNPC], t.taken[mapload.ArmHumansByType], formatCells(drops))
		if only != "" && strings.EqualFold(s.name, only) {
			detail = m
		}
		total.addAll(t)
		placements += len(m.Units)
		cells += len(drops)
		maps++
	}

	fmt.Fprintf(w, "%-16s %8d %8d %8d %8d %8d   %d cell(s) over %d map(s)\n", "TOTAL", placements,
		total.taken[mapload.ArmUnits], total.taken[mapload.ArmServerID],
		total.taken[mapload.ArmNPC], total.taken[mapload.ArmHumansByType], cells, maps)
	for _, arm := range armOrder {
		fmt.Fprintf(w, "  arm %-10s taken %6d  reached an entry %6d\n",
			arm, total.taken[arm], total.reached[arm])
	}
	if only == "" {
		return nil
	}
	if detail == nil {
		return fmt.Errorf("%s: no such map under %s", only, root)
	}
	reportNPCArm(w, c, only, detail)

	world, err := mapload.FromALMWith(detail, c.table, mapload.DifficultyNormal)
	if err != nil {
		return fmt.Errorf("%s: %w", only, err)
	}
	ents := world.Entities()
	if len(ents) != len(detail.Units) {
		return fmt.Errorf("%s: the world holds %d entities for %d placements", only, len(ents), len(detail.Units))
	}
	// sheetEntity NAMES NO TYPE OF pkg/sim (sheet.go's own doc); this loop is
	// databin.go's reportMap own conversion, restated here rather than shared,
	// because a shared helper could only be typed either sim.Entity (which
	// would put pkg/sim on this package's row of the import DAG) or a Go
	// generic naming the same fields by another route — Go's generics do not
	// admit that without naming the concrete type either.
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
	return reportSheets(w, detail, c.table, sheetEnts)
}

// reportNPCArm prints one line per NPC-arm placement of m: the npc subscript,
// the server id the registry names for it, the definition id the record itself
// carries, and the entry each of the two reaches.
//
// BOTH INDICES COME FROM THE SEARCH THE LOADER USES, not from a second walk
// written here — the npc route through Resolve, the record's own through the
// same humans search the server-id arm takes. A rewritten search could only ever
// agree with itself.
func reportNPCArm(w io.Writer, c *campaign, name string, m *alm.Map) {
	fmt.Fprintf(w, "\n%s: placements on the npc arm\n", name)
	fmt.Fprintf(w, "  %6s %8s %10s %8s %10s %8s\n",
		"record", "npc", "serverID", "entry", "own defID", "entry")
	for i, u := range m.Units {
		r := mapload.Resolve(u, c.table)
		if r.Arm != mapload.ArmNPC {
			continue
		}
		sid, ok := c.table.NPC.ServerID(int32(u.ClassSubID))
		own := data.FindHumanByServerID(c.table.Humans, int32(u.DefID))
		fmt.Fprintf(w, "  %6d %8d %10s %8d %10d %8d\n",
			i, u.ClassSubID, optional(sid, ok), r.Index, int32(u.DefID), own)
	}
}

// optional renders a value the registry may not have named at all, so an absent
// server id reads as absent rather than as zero — which is a valid index.
func optional(v int32, ok bool) string {
	if !ok {
		return "-"
	}
	return strconv.FormatInt(int64(v), 10)
}

// formatCells renders a map's drop cells, or says it authorised none. A map with
// no cell is the case the published census says never happens, so it is printed
// as words rather than as an empty column that reads like a formatting slip.
func formatCells(cells []mapload.Cell) string {
	if len(cells) == 0 {
		return "none"
	}
	parts := make([]string, len(cells))
	for i, c := range cells {
		parts[i] = fmt.Sprintf("(%d,%d)", c.X, c.Y)
	}
	return strings.Join(parts, " ")
}
