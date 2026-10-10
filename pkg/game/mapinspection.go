package game

import (
	"fmt"
	"image"
	"os"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/textinput"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// MapInspector opens authored data only: it never constructs a World, mission,
// scheduler, save store or audio device. All installed resources are read-only.
type MapInspector struct {
	archives   *Archives
	tiles      *terrain.Tileset
	statics    *terrain.StaticSet
	structures *terrain.StructureSet
	units      *terrain.UnitSet
	sacks      []*terrain.StaticFrame
	table      *mapload.Table
	font       *text.Font
	Maps       []MapEntry
}

func NewMapInspector(root string) (*MapInspector, error) {
	a, err := OpenArchives(root)
	if err != nil {
		return nil, err
	}
	x := &MapInspector{archives: a, tiles: terrain.LoadTileset(a.Containers)}
	if x.statics, err = LoadStatics(a.Containers); err != nil {
		return nil, err
	}
	if x.structures, err = LoadStructures(a.Containers); err != nil {
		return nil, err
	}
	if x.units, err = LoadUnits(a.Containers); err != nil {
		return nil, err
	}
	if x.table, err = LoadTable(a.Containers); err != nil {
		return nil, err
	}
	x.sacks = LoadSackFrames(a.Containers)
	// Shipped font1 keeps catalogue/body text legible at the small supported
	// window size. It carries the install's byte conversion selector too.
	if x.font, err = LoadFont(a.Containers, DefaultFont, FontShades); err != nil {
		return nil, err
	}
	x.font.Selector = LanguageSelector(a.Containers)
	loose, err := DirMaps(root, a.Loose)
	if err != nil {
		return nil, err
	}
	x.Maps = BuildMapList(loose, ArchiveMaps(a.Containers))
	return x, nil
}

func (x *MapInspector) Editor() *ui.MapEditor {
	rows := make([]ui.EditorMap, len(x.Maps))
	for i, e := range x.Maps {
		key := "loose/" + e.Source
		if e.FromArchive {
			key = "scenario/" + e.Source
		}
		rows[i] = ui.EditorMap{Key: key, Label: x.inspectionUTF8(e.Text())}
	}
	e := ui.NewMapEditor(x.font, rows, x.Open)
	e.SetHostTextEncoder(x.inspectionUTF8)
	return e
}

func (x *MapInspector) inspectionUTF8(s string) string {
	selector := 0
	if x.font != nil {
		selector = x.font.Selector
	}
	var b []byte
	for _, r := range s {
		c, ok := textinput.EncodeRune(r, selector)
		if !ok {
			c = '?'
		}
		b = append(b, c)
	}
	return string(b)
}

// Open accepts a catalogue key or an explicit host ALM path. It returns a
// completely prepared candidate; MapEditor commits it only on success.
func (x *MapInspector) Open(source string) (*ui.InspectionDocument, error) {
	var b []byte
	var err error
	switch {
	case strings.HasPrefix(source, "scenario/"):
		b, err = x.archives.Containers.ReadFile(source)
	case strings.HasPrefix(source, "loose/"):
		b, err = x.archives.Loose.ReadFile(strings.TrimPrefix(source, "loose/"))
	default:
		b, err = os.ReadFile(source)
	}
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", source, err)
	}
	return x.inspect(b, source)
}

func (x *MapInspector) inspect(b []byte, source string) (*ui.InspectionDocument, error) {
	model, err := mapedit.New(b)
	if err != nil {
		return nil, err
	}
	s := &mapInspectionEdits{inspector: x, model: model, source: source, saved: model.Bytes()}
	return s.project(model)
}

func (x *MapInspector) inspectModel(model *mapedit.Editor, source string) (*ui.InspectionDocument, error) {
	m, err := model.Map()
	if err != nil {
		return nil, err
	}
	// No block plane: the editor must show border cells too. No fog plane,
	// scripts, HP filtering, mission party substitution or runtime spawns.
	g := terrain.Grid{Width: m.Width, Height: m.Height, Tiles: terrain.RenderTileWords(m.Tiles),
		Altitudes: m.Altitudes, Overlay: m.Overlay, Structures: StructureRecords(m.Objects)}
	v, err := ui.NewViewerWithStatics(source, g, x.tiles, x.statics, true, false,
		terrain.AnimGateTiles, x.structures, true)
	if err != nil {
		return nil, err
	}
	v.SetAnimated(false)
	v.SetTimeFlow(false)
	v.ShowReadout(false)
	d := &ui.InspectionDocument{Viewer: v, Source: source, Name: m.Name,
		Width: m.Width, Height: m.Height, SourceBytes: model.Bytes()}
	add := func(r ui.InspectionRecord) {
		if r.Spatial && (r.Cell.X < 0 || r.Cell.Y < 0 || r.Cell.X >= m.Width || r.Cell.Y >= m.Height) {
			r.Warning = joinInspectionWarning(r.Warning, "Anchor outside map; marker clamped to edge")
		}
		d.Records = append(d.Records, r)
	}
	var entities []ui.MapEntity
	for i, u := range m.Units {
		cell := image.Pt(int(u.X>>8), int(u.Y>>8))
		r := ui.InspectionRecord{Kind: "Unit", Index: i, Section: 6, Cell: cell, Spatial: true, Size: image.Pt(1, 1), Label: fmt.Sprintf("class %d", u.ClassID)}
		var c *terrain.UnitClass
		if x.units != nil {
			c = x.units.Classes[int32(u.ClassID)]
		}
		e := ui.MapEntity{ID: uint32(i), Cell: cell, Art: c, Owner: u.Owner}
		e.DrawCategory = terrain.UnitCategoryFor(c, 0)
		if c != nil && len(c.Frames) > 0 {
			idx, mirror := terrain.SelectUnitFrame(c.Anim, len(c.Frames), false, 0, 0, 0)
			e.Frame, e.Mirror = c.Frames[idx], mirror
			if palette := x.units.OwnerPalette(c, u.Owner); palette != nil && e.Frame != nil {
				tinted := *e.Frame
				tinted.Palette = *palette
				e.Frame = &tinted
			}
			r.Label, r.Preview = c.Name, e.Frame
			r.Size = image.Pt(max(1, c.TileSize), max(1, c.TileSize))
		}
		if e.Frame == nil {
			r.Warning = "Unresolved unit art; anchor shown"
		}
		resolved := mapload.Resolve(u, x.table)
		if !resolved.Found() {
			r.Warning = joinInspectionWarning(r.Warning, "Definition unresolved")
		}
		r.Details = []string{fmt.Sprintf("X/Y raw: %d / %d", u.X, u.Y), fmt.Sprintf("Class/sub: %d / %d", u.ClassID, u.ClassSubID),
			fmt.Sprintf("Owner: %d  Unit: %d", u.Owner, u.UnitID), fmt.Sprintf("Group: %d  Flags: %#x", u.GroupID, u.Flags),
			fmt.Sprintf("Definition: %d (%s row %d)", u.DefID, resolved.Arm, resolved.Index), fmt.Sprintf("HP raw: %d (-1 = default)", u.CurrentHP),
			"Authored placement; idle pose, no simulation"}
		entities = append(entities, e)
		add(r)
	}
	v.SetEntities(entities)
	for i, o := range m.Objects {
		r := ui.InspectionRecord{Kind: "Structure", Section: 4, Index: i, Cell: image.Pt(int(o.X>>8), int(o.Y>>8)), Spatial: true, Size: image.Pt(1, 1), Label: fmt.Sprintf("kind %d", o.Kind)}
		one := g
		one.Structures = g.Structures[i : i+1]
		places, counts, _ := terrain.StructurePlacements(one, x.structures, nil, 0)
		if len(places) > 0 {
			c := places[0].Class
			r.Label, r.Preview = c.Name, places[0].Frame
			r.Size = image.Pt(c.TileWidth, c.TileHeight)
			if c.VariableSize {
				r.Size = image.Pt(one.Structures[0].VariableWidth, one.Structures[0].VariableHeight)
			}
		} else {
			r.Warning = "Unresolved structure art; anchor shown"
		}
		if counts.Cells.Outside > 0 {
			r.Warning = joinInspectionWarning(r.Warning, "Footprint crosses map boundary")
		}
		r.Details = []string{fmt.Sprintf("X/Y raw: %d / %d", o.X, o.Y), fmt.Sprintf("Kind: %d (art key %d)", o.Kind, byte(o.Kind)),
			fmt.Sprintf("Raw 0C/0E/12: %d / %d / %d", o.Field0C, o.Field0E, o.Field12), fmt.Sprintf("Extension: %x", o.Ext), fmt.Sprintf("Frame entries: %d", len(places))}
		add(r)
	}
	loot, lootErr := m.Loot()
	var sacks []ui.MapSack
	if lootErr == nil {
		for i, l := range loot.Records {
			r := ui.InspectionRecord{Kind: "Stock", Section: 8, Index: i, Label: fmt.Sprintf("owner %d", l.Owner), Size: image.Pt(1, 1)}
			if l.Ground() {
				r.Kind, r.Label, r.Spatial, r.Cell = "Sack", "ground loot", true, image.Pt(int(l.CellX()), int(l.CellY()))
				sacks = append(sacks, ui.MapSack{Cell: r.Cell})
				if len(x.sacks) > 0 {
					r.Preview = x.sacks[0]
				}
				if r.Preview == nil {
					r.Warning = "Unresolved sack art; anchor shown"
				}
			}
			r.Details = []string{fmt.Sprintf("Owner: %d  Gold: %d", l.Owner, l.Gold), fmt.Sprintf("X/Y raw: %d / %d", l.X, l.Y), fmt.Sprintf("Items: %d", len(l.Elements))}
			for n, e := range l.Elements {
				name, known := "", false
				if x.table != nil {
					name, known = x.table.Names.NameFor(data.ItemCode(e.ItemCode()))
				}
				if !known {
					name = "unresolved item name"
					r.Warning = joinInspectionWarning(r.Warning, fmt.Sprintf("Item %d name unresolved", n))
				}
				r.Details = append(r.Details, fmt.Sprintf("%d: %s", n, name))
				r.Details = append(r.Details, fmt.Sprintf("%d: code %#04x class %d index %d", n, e.ItemCode(), alm.ItemClass(e.ItemCode()), alm.ItemIndex(e.ItemCode())), fmt.Sprintf("   raw04 %d enchantment %d", e.Field04, e.TileMarkerIndex))
			}
			add(r)
		}
	}
	v.SetSackFrames(x.sacks)
	v.SetSacks(sacks)
	for i, b := range m.Overlay {
		if b == 0 {
			continue
		}
		r := ui.InspectionRecord{Kind: "Scenery", Section: 3, Index: i, Cell: image.Pt(i%m.Width, i/m.Width), Spatial: true, Size: image.Pt(1, 1), Label: fmt.Sprintf("class %d", b), Details: []string{fmt.Sprintf("Overlay byte: %d", b)}}
		if x.statics != nil && x.statics.Classes[b] != nil {
			r.Preview = x.statics.Classes[b].Frame
		}
		if r.Preview == nil {
			r.Warning = "Unresolved scenery art; anchor shown"
		}
		add(r)
	}
	for i, e := range m.Enchantments {
		r := ui.InspectionRecord{Kind: "Enchantment", Section: 9, Index: i, Label: fmt.Sprintf("tag %d", e.Tag), Warning: "Effect metadata listed only; no invented world placement",
			Details: []string{fmt.Sprintf("X/Y raw: %d / %d", e.X, e.Y), fmt.Sprintf("A/B/C: %d / %d / %d", e.A, e.B, e.C), fmt.Sprintf("Spell raw: %d", e.SpellRaw), fmt.Sprintf("Elements: %v", e.Elements)}}
		// These coordinates can name item metadata, not a world placement.
		// Do not fabricate a spatial marker from an undecided interpretation.
		add(r)
	}
	for i, g := range m.Groups {
		add(ui.InspectionRecord{Kind: "Group", Section: 5, Index: i, Label: g.Name, Details: []string{fmt.Sprintf("Scalar: %d", g.Scalar), fmt.Sprintf("Relations: %v", g.Relation)}})
	}
	script, scriptErr := m.Script()
	if scriptErr == nil {
		inspectScript(d, m, script)
		d.ScriptNotice = fmt.Sprintf("%d triggers / %d conditions / %d actions", len(script.Triggers), len(script.Conditions), len(script.Actions))
		if len(script.Triggers) == 0 {
			d.ScriptNotice = "No authored triggers"
		}
		if !m.Present(7) {
			d.ScriptNotice = "Section 7 absent; no authored script"
		}
	} else {
		d.ScriptNotice = "Section 7 decode error; see section record"
	}
	// Every physical record remains inspectable, including duplicate/unknown
	// sections ignored by the format's last-wins view. Raw bytes remain intact.
	doc, _ := alm.OpenDocument(model.Bytes())
	ids := doc.RecordTypeIDs()
	last := map[uint32]int{}
	for i, id := range ids {
		last[id] = i
	}
	for i, id := range ids {
		payload := doc.RecordPayload(i)
		r := ui.InspectionRecord{Kind: "Section", Section: int(id), Index: i, Label: fmt.Sprintf("type %d, %d bytes", id, len(payload)), Details: []string{fmt.Sprintf("File record: %d", i), fmt.Sprintf("Payload bytes: %d", len(payload)), fmt.Sprintf("First bytes: %x", payload[:min(24, len(payload))])}}
		if id >= 10 {
			r.Warning = "Unknown section; raw bytes preserved"
		}
		if last[id] != i {
			r.Warning = joinInspectionWarning(r.Warning, "Overridden duplicate; raw bytes preserved, not rendered")
		}
		if last[id] == i && id == 8 && lootErr != nil {
			r.Warning = joinInspectionWarning(r.Warning, lootErr.Error())
		}
		if last[id] == i && id == 7 && scriptErr != nil {
			r.Warning = joinInspectionWarning(r.Warning, scriptErr.Error())
		}
		if last[id] == i && id == 7 {
			r.Details = append([]string{d.ScriptNotice}, r.Details...)
		}
		if last[id] == i && id == 9 && len(m.Enchantments) != int(m.TileMarkers.Count) {
			r.Warning = joinInspectionWarning(r.Warning, fmt.Sprintf("Unresolved enchantment grammar: %d authored records", m.TileMarkers.Count))
		}
		if id == 0 {
			r.Details = append(r.Details, x.inspectionUTF8(m.Name), x.inspectionUTF8(m.Description), fmt.Sprintf("Metadata: %+v", m.Meta))
		}
		add(r)
	}
	return d, nil
}

func joinInspectionWarning(a, b string) string {
	if a == "" {
		return b
	}
	return a + "; " + b
}
