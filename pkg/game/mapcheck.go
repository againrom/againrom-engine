package game

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// mapRule is one row of the map report's rule table. Error says whether a
// finding of the rule refuses a save; a rule moves between kinds, or leaves the
// table, by editing its row. Authority names the claims and the code that
// state the rule.
type mapRule struct {
	ID        string
	Error     bool
	Authority string
	Meaning   string
	run       func(*mapCheck) []mapHit
}

// mapHit is where a rule found its record: Kind and Index address an
// InspectionRecord, an empty Kind the whole map.
type mapHit struct {
	Kind    string
	Index   int
	Cell    image.Point
	HasCell bool
	Text    string
}

// mapRules is the one rule table. The report reads nothing else.
var mapRules = []mapRule{
	{ID: "terrain-planes", Error: true,
		Authority: "TERR-PASS-049, TERR-PASS-050; mapload.OriginalTerrainPlanes",
		Meaning:   "the installed terrain refuses the map: a side outside 8..256 or a terrain pair never initialised",
		run:       (*mapCheck).terrainPlanes},
	{ID: "mission-build", Error: true,
		Authority: "StartMissionFrom, the mission build",
		Meaning:   "the mission build refuses the map",
		run:       (*mapCheck).missionBuild},
	{ID: "record-outside", Error: true,
		Authority: "ALM-UNIT-040, ALM-OBJ-019, ALM-SACK-065",
		Meaning:   "a unit, structure or ground sack lies outside the map",
		run:       (*mapCheck).recordOutside},
	{ID: "script-reference", Error: true,
		Authority: "ALM-TRIG-046, ALM-TRIG-047; the script compile's unresolved references",
		Meaning:   "a script names a unit, structure, group or player the map does not hold",
		run:       (*mapCheck).scriptReference},
	{ID: "owner-slot", Error: true,
		Authority: "ALM-OWN-039",
		Meaning:   "a unit or structure names an owner slot the roster does not hold",
		run:       (*mapCheck).ownerSlot},
	{ID: "item-code", Error: true,
		Authority: "ALM-SACK-065, ITEM-NAMEPOP-038",
		Meaning:   "a sack or stock item has a code the installed item names do not hold",
		run:       (*mapCheck).itemCode},
	{ID: "structure-kind", Error: true,
		Authority: "ALM-CLS-036, ALM-CLS-053",
		Meaning:   "a structure has a kind the installed structures.reg does not hold",
		run:       (*mapCheck).structureKind},
	{ID: "unit-terrain", Error: false,
		Authority: "mapload.Passability; the unit's movement domain",
		Meaning:   "a unit stands on a cell its movement class cannot enter",
		run:       (*mapCheck).unitTerrain},
	{ID: "structure-overlap", Error: false,
		Authority: "mapload.StructureCensus; terrain.StructurePlacements",
		Meaning:   "a structure's footprint takes cells an earlier structure took",
		run:       (*mapCheck).structureOverlap},
	{ID: "sack-duplicate", Error: false,
		Authority: "ALM-SACK-065",
		Meaning:   "two ground sacks share a cell and the world merges them into one",
		run:       (*mapCheck).sackDuplicate},
	{ID: "drop-cell", Error: false,
		Authority: "MISSION-DROP-002",
		Meaning:   "a drop cell lies outside the map or on a cell the party cannot stand on, or the map has none and the fallback cells leave the map",
		run:       (*mapCheck).dropCell},
	{ID: "stock-link", Error: false,
		Authority: "ALM-ORD-068",
		Meaning:   "a unit's stock link names a type-8 record that is not its own stock, or none",
		run:       (*mapCheck).stockLink},
	{ID: "mod-message-text", Error: false,
		Authority: "MISSION-TEXT-005, TRIG-MSGLIMIT-050; the mod's texts folder",
		Meaning:   "the script sends a message number that the mod's texts folder gives no event page for, and the install holds none either; or a text file of the folder holds a character this base cannot show",
		run:       (*mapCheck).modMessageText},
}

// mapCheck is one pass of the rule table over one map: the decoded map, the
// installed tables and the derived values several rules read, each made once.
type mapCheck struct {
	x      *MapInspector
	model  *mapedit.Editor
	m      *alm.Map
	source string
	pages  *modPages

	script    alm.Script
	scriptErr error
	loot      alm.Loot
	lootErr   error

	plane      []byte
	terrainRan bool
	terrainErr error
	world      *sim.World
	worldRan   bool
}

// sourceMission is the campaign mission an installed scenario map belongs to,
// and 0 for any other source.
func sourceMission(source string) int {
	stem, ok := strings.CutPrefix(source, scenarioPrefix)
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSuffix(stem, ".alm"))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// modPages is what a mod ships for one mission entry that the rules read: the
// entry and the files of its texts folder, by file name.
type modPages struct {
	id    string
	entry mod.Mission
	files map[string][]byte
}

// editorPages is the pages of the mod folder Save writes into for the map
// opened from source; nil without a mod or an entry for the map.
func (x *MapInspector) editorPages(source string) *modPages {
	if x.mod == nil {
		return nil
	}
	entry, ok := x.mod.entryFor(source)
	if !ok {
		return nil
	}
	p := &modPages{id: x.mod.id, entry: entry, files: map[string][]byte{}}
	if entry.Texts == "" {
		return p
	}
	dir := filepath.Join(x.mod.dir, filepath.FromSlash(entry.Texts))
	files, err := os.ReadDir(dir)
	if err != nil {
		return p
	}
	for _, f := range files {
		if !f.Type().IsRegular() {
			continue
		}
		if b, err := os.ReadFile(filepath.Join(dir, f.Name())); err == nil {
			p.files[f.Name()] = b
		}
	}
	return p
}

// missionPages is the pages the mod loader holds for a mission it declares.
func missionPages(m mod.Mission) *modPages {
	return &modPages{id: m.Mod, entry: m, files: m.TextFiles}
}

// CheckMap runs the rule table over the map the model now holds and answers the
// findings, errors first, each group in table order and then record order.
func (x *MapInspector) CheckMap(model *mapedit.Editor, source string) ([]ui.InspectionFinding, error) {
	return x.checkMapPages(model, source, x.editorPages(source))
}

// CheckModMap is the rule table over the map bytes of a mission a mod declares,
// with the texts that mission ships. It is the one entry the mod load and the
// editor's report share. A map the editor's model cannot open is one
// mission-build error.
func (x *MapInspector) CheckModMap(m mod.Mission, source string) []ui.InspectionFinding {
	model, err := mapedit.New(m.MapBytes)
	if err != nil {
		return []ui.InspectionFinding{{Error: true, Rule: "mission-build", Text: err.Error()}}
	}
	return x.report(model, source, missionPages(m))
}

// report answers the findings for the map the model holds, a failed check being
// one mission-build error.
func (x *MapInspector) report(model *mapedit.Editor, source string, pages *modPages) []ui.InspectionFinding {
	rows, err := x.checkMapPages(model, source, pages)
	if err != nil {
		rows = []ui.InspectionFinding{{Error: true, Rule: "mission-build", Text: err.Error()}}
	}
	if rows == nil {
		rows = []ui.InspectionFinding{}
	}
	return rows
}

func (x *MapInspector) checkMapPages(model *mapedit.Editor, source string, pages *modPages) ([]ui.InspectionFinding, error) {
	m, err := model.Map()
	if err != nil {
		return nil, err
	}
	c := &mapCheck{x: x, model: model, m: m, source: source, pages: pages}
	c.script, c.scriptErr = m.Script()
	c.loot, c.lootErr = m.Loot()
	var errs, warnings []ui.InspectionFinding
	for _, rule := range x.ruleTable() {
		for _, h := range rule.run(c) {
			f := ui.InspectionFinding{Error: rule.Error, Rule: rule.ID, Kind: h.Kind, Index: h.Index, Cell: h.Cell, HasCell: h.HasCell, Text: h.Text}
			if rule.Error {
				errs = append(errs, f)
			} else {
				warnings = append(warnings, f)
			}
		}
	}
	return append(errs, warnings...), nil
}

// ruleTable is the rule table the inspector reports with: mapRules unless the
// inspector was given another.
func (x *MapInspector) ruleTable() []mapRule {
	if x.rules != nil {
		return x.rules
	}
	return mapRules
}

func (c *mapCheck) width() int  { return c.m.Width }
func (c *mapCheck) height() int { return c.m.Height }

func (c *mapCheck) inside(cell image.Point) bool {
	return cell.In(image.Rect(0, 0, c.width(), c.height()))
}

func unitCell(u alm.Unit) image.Point { return image.Pt(int(u.X>>8), int(u.Y>>8)) }

// lootKind is the inspection kind of type-8 record r.
func lootKind(r alm.LootRecord) string {
	if r.Ground() {
		return "Sack"
	}
	return "Stock"
}

// terrainCosts reads the installed terrain cost table; ok is false when the
// install carries none.
func (x *MapInspector) terrainCosts() (costs [11]byte, ok bool, err error) {
	if x == nil || x.archives == nil || x.archives.Containers == nil {
		return costs, false, nil
	}
	raw, err := x.archives.Containers.ReadFile(worldPrefix + "data/map.reg")
	if errors.Is(err, fs.ErrNotExist) {
		return costs, false, nil
	}
	if err != nil {
		return costs, false, fmt.Errorf("original terrain parameters: %w", err)
	}
	costs, err = data.OriginalTerrainCostTable(raw)
	if err != nil {
		return costs, false, fmt.Errorf("original terrain parameters: %w", err)
	}
	return costs, true, nil
}

func (c *mapCheck) terrainPlanes() []mapHit {
	c.terrainRan = true
	costs, ok, err := c.x.terrainCosts()
	if err == nil && ok {
		_, err = mapload.OriginalTerrainPlanes(c.m, costs)
	}
	c.terrainErr = err
	if err != nil {
		return []mapHit{{Text: err.Error()}}
	}
	return nil
}

func (c *mapCheck) missionBuild() []mapHit {
	if c.x == nil || c.x.table == nil {
		return nil
	}
	if !c.terrainRan {
		c.terrainPlanes()
	}
	if c.terrainErr != nil {
		return nil // reported by terrain-planes
	}
	// A fresh decode: the build is free to change the map it is given.
	m, err := c.model.Map()
	if err != nil {
		return []mapHit{{Text: err.Error()}}
	}
	if _, err := StartMissionFrom(m, c.source, sourceMission(c.source), c.x.table, mapload.DifficultyNormal, nil); err != nil {
		return []mapHit{{Text: err.Error()}}
	}
	return nil
}

// structureFootprint is the rectangle a structure's footprint covers, at least
// one cell on each side.
func structureFootprint(st sim.Structure) image.Rectangle {
	return image.Rect(int(st.Col), int(st.Row), int(st.Col)+max(1, int(st.Width)), int(st.Row)+max(1, int(st.Height)))
}

func (c *mapCheck) recordOutside() []mapHit {
	var out []mapHit
	for i, u := range c.m.Units {
		if cell := unitCell(u); !c.inside(cell) {
			out = append(out, mapHit{Kind: "Unit", Index: i, Cell: cell, HasCell: true,
				Text: fmt.Sprintf("the unit stands at cell %d,%d, outside the %dx%d map", cell.X, cell.Y, c.width(), c.height())})
		}
	}
	for i, st := range mapload.Structures(c.m, c.x.table) {
		anchor := image.Pt(int(c.m.Objects[i].X>>8), int(c.m.Objects[i].Y>>8))
		r := structureFootprint(st)
		switch {
		case !c.inside(anchor):
			out = append(out, mapHit{Kind: "Structure", Index: i, Cell: anchor, HasCell: true,
				Text: fmt.Sprintf("the structure stands at cell %d,%d, outside the %dx%d map", anchor.X, anchor.Y, c.width(), c.height())})
		case !r.In(image.Rect(0, 0, c.width(), c.height())):
			out = append(out, mapHit{Kind: "Structure", Index: i, Cell: anchor, HasCell: true,
				Text: fmt.Sprintf("the structure's footprint %d,%d %dx%d leaves the %dx%d map", r.Min.X, r.Min.Y, r.Dx(), r.Dy(), c.width(), c.height())})
		}
	}
	for i, r := range c.loot.Records {
		if !r.Ground() {
			continue
		}
		if cell := image.Pt(int(r.CellX()), int(r.CellY())); !c.inside(cell) {
			out = append(out, mapHit{Kind: "Sack", Index: i, Cell: cell, HasCell: true,
				Text: fmt.Sprintf("the sack lies at cell %d,%d, outside the %dx%d map", cell.X, cell.Y, c.width(), c.height())})
		}
	}
	return out
}

// scriptNodeIndex is the index of the first node of nodes whose identity is id.
func scriptNodeIndex(nodes []alm.ScriptNode, id uint32) int {
	for i, n := range nodes {
		if n.ID == id {
			return i
		}
	}
	return -1
}

// scriptGroupParam is the parameter type code of a Target_Group value, and
// scriptHeroBand the first Target_Unit value that names the party or an
// external static unit and no placed unit.
const (
	scriptGroupParam = 2
	scriptHeroBand   = 10001
)

func (c *mapCheck) scriptReference() []mapHit {
	if c.scriptErr != nil {
		return nil
	}
	var out []mapHit
	_, report, err := tableFiles(c.x.table).compileScript(c.m, campaignScriptRefs(c.m, c.x.table, nil, sourceMission(c.source)))
	if err == nil {
		for _, u := range report.Unresolved {
			if u.Kind == mapload.ScriptRefUnit && u.Value >= scriptHeroBand {
				continue // the party and external static units, which no placement holds
			}
			kind, nodes := "Action", c.script.Actions
			if u.Condition {
				kind, nodes = "Condition", c.script.Conditions
			}
			i := scriptNodeIndex(nodes, u.NodeID)
			if i < 0 {
				continue
			}
			out = append(out, mapHit{Kind: kind, Index: i,
				Text: fmt.Sprintf("%s %d (opcode %#x) names %s %d, which the map does not hold", strings.ToLower(kind), u.NodeID, u.Opcode, u.Kind, u.Value)})
		}
	}
	groups := map[uint32]bool{}
	for _, u := range c.m.Units {
		groups[u.GroupID] = true
	}
	for _, family := range []struct {
		kind  string
		nodes []alm.ScriptNode
	}{{"Action", c.script.Actions}, {"Condition", c.script.Conditions}} {
		for i, n := range family.nodes {
			for p, tag := range n.Type {
				if tag == 3 && rosterSlotError(c.m, n.Value[p]) != nil {
					out = append(out, mapHit{Kind: family.kind, Index: i,
						Text: fmt.Sprintf("%s %d (opcode %#x) parameter %d names player %d, which the map does not hold", strings.ToLower(family.kind), n.ID, n.Opcode, p, n.Value[p])})
				}
				if tag == scriptGroupParam && n.Value[p] != 0 && !groups[n.Value[p]] {
					out = append(out, mapHit{Kind: family.kind, Index: i,
						Text: fmt.Sprintf("%s %d (opcode %#x) names group %d, which no unit holds", strings.ToLower(family.kind), n.ID, n.Opcode, n.Value[p])})
				}
			}
		}
	}
	return out
}

// rosterSlotError is why owner is not a slot of the map's roster, or nil. A map
// without a type-5 record plays with one synthesised side (ALM-REQ-056).
func rosterSlotError(m *alm.Map, owner uint32) error {
	slots := len(m.Groups)
	if !m.Present(5) {
		slots = 1
	}
	if owner < 1 || int(owner) > slots {
		return fmt.Errorf("owner slot %d is not in the map's roster of %d", owner, slots)
	}
	return nil
}

func (c *mapCheck) ownerSlot() []mapHit {
	var out []mapHit
	for i, u := range c.m.Units {
		if err := rosterSlotError(c.m, u.Owner); err != nil {
			out = append(out, mapHit{Kind: "Unit", Index: i, Cell: unitCell(u), HasCell: true, Text: "the unit's " + err.Error()})
		}
	}
	for i, o := range c.m.Objects {
		if err := rosterSlotError(c.m, o.Field0E&0xffff); err != nil {
			out = append(out, mapHit{Kind: "Structure", Index: i, Cell: image.Pt(int(o.X>>8), int(o.Y>>8)), HasCell: true, Text: "the structure's " + err.Error()})
		}
	}
	return out
}

func (c *mapCheck) itemCode() []mapHit {
	if c.x.table == nil {
		return nil
	}
	var out []mapHit
	for i, r := range c.loot.Records {
		for n, e := range r.Elements {
			if err := c.x.checkItem(int64(e.ItemCode())); err != nil {
				h := mapHit{Kind: lootKind(r), Index: i, Text: fmt.Sprintf("element %d: %v", n, err)}
				if r.Ground() {
					h.Cell, h.HasCell = image.Pt(int(r.CellX()), int(r.CellY())), true
				}
				out = append(out, h)
			}
		}
	}
	return out
}

func (c *mapCheck) structureKind() []mapHit {
	if c.x.structures == nil {
		return nil
	}
	var out []mapHit
	for i, o := range c.m.Objects {
		if err := c.x.checkStructureKind(int64(byte(o.Kind))); err != nil {
			out = append(out, mapHit{Kind: "Structure", Index: i, Cell: image.Pt(int(o.X>>8), int(o.Y>>8)), HasCell: true, Text: err.Error()})
		}
	}
	return out
}

// builtWorld is the world the mission build makes from the map alone, made once.
func (c *mapCheck) builtWorld() *sim.World {
	if !c.worldRan {
		c.worldRan = true
		if w, err := mapload.FromALMWith(c.m, c.x.table, mapload.DifficultyNormal); err == nil {
			c.world = w
		}
	}
	return c.world
}

func (c *mapCheck) unitTerrain() []mapHit {
	if c.x.table == nil || len(c.m.Units) == 0 {
		return nil
	}
	w := c.builtWorld()
	if w == nil {
		return nil
	}
	if c.plane == nil {
		c.plane = mapload.Passability(c.m)
	}
	ents := w.Entities()
	var out []mapHit
	for i, u := range c.m.Units {
		cell := unitCell(u)
		if i >= len(ents) || !c.inside(cell) || len(c.plane) != c.width()*c.height() {
			continue
		}
		if ents[i].Domain.BlockedBy(c.plane[cell.Y*c.width()+cell.X]) {
			text := fmt.Sprintf("the unit stands on cell %d,%d, which its movement class cannot enter", cell.X, cell.Y)
			if mapload.InBorderRing(c.m, cell.X, cell.Y) {
				text += "; the cell is in the border ring, and the game removes a unit placed there when the mission loads"
			}
			out = append(out, mapHit{Kind: "Unit", Index: i, Cell: cell, HasCell: true, Text: text})
		}
	}
	return out
}

func (c *mapCheck) structureOverlap() []mapHit {
	if c.x.table == nil || len(c.m.Objects) < 2 {
		return nil
	}
	if mapload.StructureCensus(c.m, c.x.table).Refused == 0 {
		return nil
	}
	// A structure overlaps when the census of the map cut after it counts more
	// refused cells than the map cut before it.
	var out []mapHit
	cut := *c.m
	prev := 0
	for i := range c.m.Objects {
		cut.Objects = c.m.Objects[:i+1]
		refused := mapload.StructureCensus(&cut, c.x.table).Refused
		if refused > prev {
			o := c.m.Objects[i]
			out = append(out, mapHit{Kind: "Structure", Index: i, Cell: image.Pt(int(o.X>>8), int(o.Y>>8)), HasCell: true,
				Text: fmt.Sprintf("the structure's footprint takes %d cells an earlier structure took", refused-prev)})
		}
		prev = refused
	}
	return out
}

func (c *mapCheck) sackDuplicate() []mapHit {
	seen := map[image.Point]int{}
	var out []mapHit
	for i, r := range c.loot.Records {
		if !r.Ground() {
			continue
		}
		cell := image.Pt(int(r.CellX()), int(r.CellY()))
		if !c.inside(cell) {
			continue
		}
		if first, ok := seen[cell]; ok {
			out = append(out, mapHit{Kind: "Sack", Index: i, Cell: cell, HasCell: true,
				Text: fmt.Sprintf("the sack shares cell %d,%d with sack %d; the world merges them into one", cell.X, cell.Y, first)})
			continue
		}
		seen[cell] = i
	}
	return out
}

func (c *mapCheck) dropCell() []mapHit {
	cells := mapload.DropCells(c.m)
	if c.plane == nil {
		c.plane = mapload.Passability(c.m)
	}
	out := c.fallbackDrop(cells)
	if len(cells) == 0 {
		return out
	}
	var nodes []int
	for i, n := range c.script.Actions {
		if n.Opcode == dropActionOpcode {
			nodes = append(nodes, i)
		}
	}
	for k, d := range cells {
		cell := image.Pt(int(d.X), int(d.Y))
		text := ""
		switch {
		case !c.inside(cell):
			text = fmt.Sprintf("the drop cell %d,%d lies outside the %dx%d map", cell.X, cell.Y, c.width(), c.height())
		case len(c.plane) == c.width()*c.height() && sim.DomainGround.BlockedBy(c.plane[cell.Y*c.width()+cell.X]):
			text = fmt.Sprintf("the drop cell %d,%d is a cell the party cannot stand on", cell.X, cell.Y)
		default:
			continue
		}
		h := mapHit{Kind: "Action", Index: -1, Cell: cell, HasCell: true, Text: text}
		if len(nodes) == len(cells) {
			h.Index = nodes[k]
		} else {
			h.Kind = ""
		}
		out = append(out, h)
	}
	return out
}

// fallbackDrop (DIV-3212) answers the finding for a map whose drop cells give the party no
// cell, or only some (the start picks one at random): the start then draws one cell from the fallback square on each axis
// (MISSION-DROP-002), and the finding counts the cells of that square the party
// cannot stand on, outside the map or blocked.
func (c *mapCheck) fallbackDrop(cells []mapload.Cell) []mapHit {
	zero := 0
	for _, d := range cells {
		if d.X == 0 || d.Y == 0 {
			zero++
		}
	}
	if zero == 0 && len(cells) > 0 {
		return nil
	}
	low, high := mapload.FallbackDropRange()
	lost, total := 0, 0
	for y := low; y <= high; y++ {
		for x := low; x <= high; x++ {
			total++
			cell := image.Pt(int(x), int(y))
			if !c.inside(cell) || len(c.plane) == c.width()*c.height() && sim.DomainGround.BlockedBy(c.plane[cell.Y*c.width()+cell.X]) {
				lost++
			}
		}
	}
	if lost == 0 {
		return nil
	}
	if zero > 0 && zero < len(cells) {
		return []mapHit{{Text: fmt.Sprintf("%d of the map's %d drop cells have a zero coordinate; the start picks one drop cell at random, and such a pick sends the party to a random cell of %d to %d on each axis: %d of its %d cells are outside the %dx%d map or blocked",
			zero, len(cells), low, high, lost, total, c.width(), c.height())}}
	}
	return []mapHit{{Text: fmt.Sprintf("the map has no drop cell, and the party lands on a random cell of %d to %d on each axis: %d of its %d cells are outside the %dx%d map or blocked",
		low, high, lost, total, c.width(), c.height())}}
}

// dropActionOpcode is the script action whose first two values are a drop cell.
const dropActionOpcode = 0x10002

// unitStockLink is the type-6 word at +0x18: the 1-based type-8 record that
// holds the unit's stock, 0 for none.
const unitStockLink = 0x18

func (c *mapCheck) stockLink() []mapHit {
	var out []mapHit
	for i, u := range c.m.Units {
		rec, err := c.model.Record(mapedit.KindUnit, i)
		if err != nil || len(rec) < unitStockLink+4 {
			continue
		}
		link := int(binary.LittleEndian.Uint32(rec[unitStockLink:]))
		if link == 0 {
			continue
		}
		text := ""
		switch {
		case link > len(c.loot.Records):
			text = fmt.Sprintf("the unit's stock link %d names no type-8 record of the %d the map holds", link, len(c.loot.Records))
		case c.loot.Records[link-1].Owner != uint32(u.UnitID):
			text = fmt.Sprintf("the unit's stock link %d names type-8 record %d, which belongs to unit id %d and not to this unit (id %d)",
				link, link-1, c.loot.Records[link-1].Owner, u.UnitID)
		default:
			continue
		}
		out = append(out, mapHit{Kind: "Unit", Index: i, Cell: unitCell(u), HasCell: true, Text: text})
	}
	return out
}

// Report answers the findings for the map as it now stands, errors first. The
// last answer is kept while the map bytes stay the same.
func (s *mapInspectionEdits) Report() []ui.InspectionFinding {
	b := s.model.Bytes()
	stamp := s.textsStamp()
	if s.reported != nil && bytes.Equal(s.reportedFor, b) && s.reportedTexts == stamp {
		return s.reported
	}
	rows := s.inspector.report(s.model, s.source, s.inspector.editorPages(s.source))
	s.reported, s.reportedFor, s.reportedTexts = rows, append([]byte(nil), b...), stamp
	return rows
}

// textsStamp names the file names and sizes of the texts folder of the mod's
// entry for this map, the part of the report that does not come from the map
// bytes. It is empty without a mod or a folder.
func (s *mapInspectionEdits) textsStamp() string {
	m := s.inspector.mod
	if m == nil {
		return ""
	}
	entry, ok := m.entryFor(s.source)
	if !ok || entry.Texts == "" {
		return ""
	}
	files, err := os.ReadDir(filepath.Join(m.dir, filepath.FromSlash(entry.Texts)))
	if err != nil {
		return ""
	}
	var sb strings.Builder
	for _, f := range files {
		if info, err := f.Info(); err == nil && info.Mode().IsRegular() {
			fmt.Fprintf(&sb, "%s:%d;", f.Name(), info.Size())
		}
	}
	return sb.String()
}

// gate refuses a save of a map whose report holds an error.
func (s *mapInspectionEdits) gate() error {
	rows := s.Report()
	if errs, _ := ui.CountFindings(rows); errs > 0 {
		return &ui.ReportRefusal{Findings: rows}
	}
	return nil
}

// modMessageText is the rule for a map that Save writes into a mod: each
// message number the script sends needs an eventNN.txt page in the texts
// folder of the mod's mission entry. A replacing mission keeps the install's
// page for a number the folder leaves out, so only a number neither holds is
// reported. A send of the reserved number 255 is the lost-mission panel and
// takes no page.
func (c *mapCheck) modMessageText() []mapHit {
	if c.pages == nil {
		return nil
	}
	return append(c.modMissingPages(c.pages), c.modDroppedCharacters(c.pages)...)
}

// modDroppedCharacters names, once per file, a character of a UTF-8 file in the
// entry's texts folder that the running base cannot show after the plain-form
// table of the mod text encoder.
func (c *mapCheck) modDroppedCharacters(p *modPages) []mapHit {
	if p.entry.Texts == "" || c.x.archives == nil || c.x.archives.Containers == nil {
		return nil
	}
	selector := LanguageSelector(c.x.archives.Containers)
	names := make([]string, 0, len(p.files))
	for name := range p.files {
		names = append(names, name)
	}
	slices.Sort(names)
	var out []mapHit
	for _, name := range names {
		if r, dropped := ModTextDropped(p.files[name], selector); dropped {
			out = append(out, mapHit{Text: fmt.Sprintf("mod %q: %s/%s holds the character %q (U+%04X), which this base cannot show; it is dropped from the page", p.id, p.entry.Texts, name, r, r)})
		}
	}
	return out
}

// modMissingPages is the message-number half of the rule.
func (c *mapCheck) modMissingPages(p *modPages) []mapHit {
	if c.scriptErr != nil {
		return nil
	}
	_, report, err := tableFiles(c.x.table).compileScript(c.m, campaignScriptRefs(c.m, c.x.table, nil, sourceMission(c.source)))
	if err != nil {
		return nil
	}
	seen := map[int32]bool{}
	var out []mapHit
	for _, r := range report.Raises {
		if r.Event <= 0 || r.Event == ReservedMessageNumber || seen[r.Event] {
			continue
		}
		seen[r.Event] = true
		name := fmt.Sprintf("event%02d.txt", r.Event)
		if p.entry.Texts != "" {
			if _, ok := p.files[name]; ok {
				continue
			}
		}
		if p.entry.Kind == mod.MissionReplace && c.x.archives != nil && c.x.archives.Containers != nil {
			if _, shipped := ReadEventText(c.x.archives.Containers, p.entry.Number, int(r.Event)); shipped {
				continue
			}
		}
		out = append(out, mapHit{Text: fmt.Sprintf("the script sends message %d, and mod %q gives no %s", r.Event, p.id, name)})
	}
	return out
}
