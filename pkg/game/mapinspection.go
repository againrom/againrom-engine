package game

import (
	"fmt"
	"image"
	"os"
	"slices"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/textinput"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// MapInspector opens authored data. It builds a World only to read what the
// game spawns from a map (the unit sheet, the report's mission build) and never
// steps one; it opens no scheduler, save store or audio device. All installed
// resources are read-only.
type MapInspector struct {
	modOutput      string
	archives       *Archives
	tiles          *terrain.Tileset
	statics        *terrain.StaticSet
	objectRows     []ui.BrushObject // the objects list: the clear row, then each drawable class
	swatches       []*image.RGBA    // the terrain list's pictures, by Palette row; nil until first asked
	structures     *terrain.StructureSet
	units          *terrain.UnitSet
	sacks          []*terrain.StaticFrame
	table          *mapload.Table
	font           *text.Font
	body           *text.Font // the documents font: rows, labels, values and report text
	Maps           []MapEntry
	mod            *editorMod    // the mod folder Save writes into; nil for Save As
	testPlay       *TestPlayHost // the game the test play key starts; nil: no test play
	rules          []mapRule     // the map report's rule table; nil is mapRules
	recordOptions  map[string][]ui.InspectionChoice
	recordPictures map[string]map[int64]*image.RGBA
	tiers          *tierCache // tiersOf's answers; nil computes each time
	panelWords     *ui.Words
	cardFont       *text.Font
}

func NewMapInspector(root string) (*MapInspector, error) {
	a, err := OpenArchives(root)
	if err != nil {
		return nil, err
	}
	x := &MapInspector{archives: a, tiles: terrain.LoadTileset(a.Containers), tiers: &tierCache{}}
	if x.statics, err = LoadStatics(a.Containers); err != nil {
		return nil, err
	}
	x.objectRows = objectBrushRows(a.Containers, x.statics)
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
	panelWords := LoadInstallWords(a.Containers, InstallTextCode(a.Containers, a.Game().Edition())).Words()
	x.panelWords = &panelWords
	// Shipped font1 keeps catalogue/body text legible at the small supported
	// window size. It carries the install's byte conversion selector too.
	if x.font, err = LoadFont(a.Containers, DefaultFont, FontShades); err != nil {
		return nil, err
	}
	x.font.Selector = LanguageSelector(a.Containers)
	x.cardFont = x.font
	if d := GeneratorDescription(a.Base.Profile); !a.Base.Profile.Limits.NoCharacterGeneration && d != nil && d.Words.TipFont == "text" {
		card, err := chargenFont(a.artSource(), d.Words.TextFont)
		if err != nil {
			return nil, err
		}
		card.Selector = x.font.Selector
		x.cardFont = card
	}
	// The documents font is the install's plain reading face; the editor's
	// rows, labels, values and reports use it, its titles font1.
	// An install without it keeps font1 throughout.
	if body, err := LoadFont(a.Containers, DocumentFont, FontCoverage); err == nil {
		body.Selector = x.font.Selector
		x.body = body
	}
	loose, err := DirMaps(root, a.Loose)
	if err != nil {
		return nil, err
	}
	x.Maps = BuildMapList(loose, ArchiveMaps(a.Containers))
	return x, nil
}

func (x *MapInspector) Editor() *ui.MapEditor {
	rows := newMapRows()
	for _, e := range x.Maps {
		key := "loose/" + e.Source
		if e.FromArchive {
			key = "scenario/" + e.Source
		}
		e.Name = alm.MapNameText.Decode([]byte(e.Name))
		rows = append(rows, ui.EditorMap{Key: key, Label: x.inspectionUTF8(e.Text())})
	}
	e := ui.NewMapEditor(x.font, rows, x.Open)
	e.SetBodyFont(x.body)
	e.SetCardFont(x.cardFont)
	if x.archives != nil {
		e.SetWords(words.For(x.archives.Base.Profile.Language))
	}
	e.SetHostTextEncoder(x.inspectionUTF8, func(value string) string { return x.messageUTF8([]byte(value)) })
	return e
}

// inspectionUTF8 is the editor's host text in the font's bytes: the install
// text codec under its Substitute policy.
func (x *MapInspector) inspectionUTF8(s string) string {
	selector := 0
	if x.font != nil {
		selector = x.font.Selector
	}
	b, _ := textinput.EncodeText(s, selector, textinput.Substitute)
	return b
}

// Open accepts a catalogue key or an explicit host ALM path. It returns a
// completely prepared candidate; MapEditor commits it only on success.
func (x *MapInspector) Open(source string) (*ui.InspectionDocument, error) {
	var b []byte
	var err error
	switch {
	case isNewSource(source):
		return x.openNew(source)
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

func (x *MapInspector) inspectModel(model *mapedit.Editor, source string, pages map[string][]byte) (*ui.InspectionDocument, error) {
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
	v.SetOwnArt(true)
	if x.panelWords != nil {
		v.SetWords(*x.panelWords)
	}
	v.SetTimeFlow(false)
	v.ShowReadout(false)
	d := &ui.InspectionDocument{Viewer: v, Source: source, Name: alm.MapNameText.Decode([]byte(m.Name)),
		Width: m.Width, Height: m.Height, SourceBytes: model.Bytes()}
	for _, c := range mapload.DropCells(m) {
		d.Drops = append(d.Drops, image.Pt(int(c.X), int(c.Y)))
	}
	d.Passability = passArms(mapload.GroundArms(m, x.table))
	feet := x.feet(m)
	add := func(r ui.InspectionRecord) {
		if r.Spatial && (r.Cell.X < 0 || r.Cell.Y < 0 || r.Cell.X >= m.Width || r.Cell.Y >= m.Height) {
			r.Warning = joinInspectionWarning(r.Warning, "Anchor outside map; marker clamped to edge")
		}
		d.Records = append(d.Records, r)
	}
	var entities []ui.MapEntity
	// The tier each placement draws at, and the actor it spawns as, are the
	// game's: entityTiers and the world built at Normal difficulty. The build
	// takes a fresh decode, being free to change the map it is given.
	tiers := entityTiers(m, x.table)
	var sheets, templates []unitSheet
	sheetErr := fmt.Errorf("no installed tables")
	if x.table != nil {
		if built, err := model.Map(); err != nil {
			sheetErr = err
		} else {
			sheets, sheetErr = unitSheets(built, x.table)
		}
		// The template sheets beside the override section: built only for a
		// map that has a placement with an override.
		if slices.ContainsFunc(m.Units, hasTail) {
			if built, err := model.Map(); err == nil {
				templates, _ = templateSheets(built, x.table)
			}
		}
	}
	for i, u := range m.Units {
		cell := image.Pt(int(u.X>>8), int(u.Y>>8))
		r := ui.InspectionRecord{Kind: "Unit", Index: i, Section: 6, Cell: cell, Spatial: true, Size: image.Pt(1, 1), Footprint: feet.units[i], Label: fmt.Sprintf("class %d", u.ClassID), Player: int(u.Owner), Group: int(u.GroupID)}
		var c *terrain.UnitClass
		if x.units != nil {
			c = x.units.Classes[int32(u.ClassID)]
		}
		e := ui.MapEntity{ID: uint32(i), Cell: cell, Art: c, Owner: u.Owner}
		if c != nil {
			e.TokenSize = c.TileSize
		}
		e.DrawCategory = terrain.UnitCategoryFor(c, 0)
		tier := tiers[sim.EntityID(i)]
		if frames := c.TierFrames(tier); len(frames) > 0 {
			idx, mirror := terrain.SelectUnitFrame(c.Anim, len(frames), false, 0, 0, 0)
			e.Frame, e.Mirror = frames[idx], mirror
			shade := u.Owner
			if u.Owner > 0 && uint64(u.Owner) <= uint64(len(m.Groups)) {
				shade = m.Groups[u.Owner-1].Color
			}
			if palette := x.units.OwnerPalette(c, shade); palette != nil && e.Frame != nil {
				tinted := *e.Frame
				tinted.Palette = *palette
				e.Frame = &tinted
			}
			r.Label, r.Preview = c.Name, e.Frame
			r.Size = image.Pt(max(1, c.TileSize), max(1, c.TileSize))
			if i < len(sheets) && sheets[i].Life != ui.LifeAlive {
				x.fallenDraw(&e, c, tier, sheets[i], shade)
			}
		}
		multi := len(x.tiersOf(u)) > 1
		if multi {
			r.Label += x.inspectionUTF8(fmt.Sprintf("  %s %d", x.word("editor.unit.tier"), u.ClassSubID))
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
		if i < len(sheets) {
			r.Lines = append(r.Lines, x.sheetLines(u, resolved, sheets[i])...)
			template := sheets[i]
			if i < len(templates) {
				template = templates[i]
			}
			r.Lines = append(r.Lines, x.overrideLines(model, i, template)...)
			v := sheets[i].Values
			r.Hover = r.Label + x.inspectionUTF8("  "+fmt.Sprintf(x.word("editor.unit.hover"), v.HP, v.MaxHP))
		} else {
			r.Lines = append(r.Lines, ui.InspectionLine{Text: x.inspectionUTF8(fmt.Sprintf(x.word("editor.unit.unavailable"), sheetErr)), Role: "warning"})
		}
		entities = append(entities, e)
		add(r)
	}
	v.SetEntities(entities)
	for i, o := range m.Objects {
		r := ui.InspectionRecord{Kind: "Structure", Section: 4, Index: i, Cell: image.Pt(int(o.X>>8), int(o.Y>>8)), Spatial: true, Size: image.Pt(1, 1), Footprint: feet.structures[i], Label: fmt.Sprintf("kind %d", o.Kind)}
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
			r.Details = []string{fmt.Sprintf("Owner: %d  Gold: %d", l.Owner, l.Gold), fmt.Sprintf("X/Y raw: %d / %d", l.X, l.Y), fmt.Sprintf("Items: %d / I: recipe by item number", len(l.Elements))}
			for n, e := range l.Elements {
				name, known := mapload.ShippedItemName(x.table, data.ItemCode(e.ItemCode()))
				if !known {
					name = "unresolved item name"
					r.Warning = joinInspectionWarning(r.Warning, fmt.Sprintf("Item %d name unresolved", n))
				}
				r.Details = append(r.Details, fmt.Sprintf("Item %d: %s", n+1, name))
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
		// These coordinates can name item metadata, not a world placement.
		// Do not fabricate a spatial marker from an undecided interpretation.
		add(x.inspectionEnchantment(i, e, m))
	}
	for i, g := range m.Groups {
		add(inspectionPlayer(model, i, x.inspectionUTF8(alm.PlayerNameText.Decode([]byte(g.Name)))))
	}
	script, scriptErr := m.Script()
	if scriptErr == nil {
		inspectScript(d, m, script, x.inspectionUTF8)
		d.Script.Targets = x.scriptTargets(d, m, script, pages)
		inspectScriptUses(d)
		d.ScriptNotice = fmt.Sprintf("%d triggers / %d conditions / %d actions", len(script.Triggers), len(script.Conditions), len(script.Actions))
		if len(script.Triggers) == 0 {
			d.ScriptNotice = "No authored triggers"
		}
		if !m.Present(7) {
			d.ScriptNotice = "Section 7 absent; no authored script"
		}
		if d.AgainromOnly = mapload.ScriptAgainromOnly(script); d.AgainromOnly {
			d.ScriptNotice += " / Againrom-only map"
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
			r.Details = append(r.Details, x.inspectionUTF8(alm.MapNameText.Decode([]byte(m.Name))), x.inspectionUTF8(m.Description), fmt.Sprintf("Metadata: %+v", m.Meta))
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
