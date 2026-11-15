package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/namefold"
	"againrom/pkg/render/terrain"
	"fmt"
	"slices"
	"strings"
)

// editorContext is the definition table and the tileset the editor reads a
// map by: the install's own, or a mod's admitted fork of them.
type editorContext struct {
	table *mapload.Table
	tiles *terrain.Tileset
}

// laidContext is a mod's context the editor reads by, with the context it
// was laid over.
type laidContext struct {
	mod, under editorContext
}

// contextBase is the context a mod's fork is built from: the one a laid mod
// context covers, else the table and tileset the editor holds now.
func (x *MapInspector) contextBase() editorContext {
	if l := x.laid; l != nil && l.mod.table == x.table && l.mod.tiles == x.tiles {
		return l.under
	}
	return editorContext{table: x.table, tiles: x.tiles}
}

// setEditorMod makes m the mod the editor works in and reads by: m's admitted
// context when it has one, else the context beneath any laid one.
func (x *MapInspector) setEditorMod(m *editorMod) {
	base := x.contextBase()
	x.mod = m
	next, laid := base, (*laidContext)(nil)
	if m != nil && m.context != nil {
		next, laid = *m.context, &laidContext{mod: *m.context, under: base}
	}
	x.laid = laid
	if next.table == x.table && next.tiles == x.tiles {
		return
	}
	x.table, x.tiles = next.table, next.tiles
	// The choices, swatches and tiers read the table and the tileset; they are
	// built again from the context now in use.
	x.swatches, x.recordOptions, x.itemGroups = nil, nil, nil
	x.identities, x.mercenaryLevels, x.identityDraws, x.identityMission = nil, nil, nil, 0
	// The placed people's sheets are drawn from the table's rows.
	x.placedArts = nil
	if x.tiers != nil {
		x.tiers = &tierCache{}
	}
}

// admittedFork is the one mod context the editor builds, the route the
// game's launch takes: a fork of the install's table and archives with the
// mods' install files served, their rules set by SetMods and their data
// applied by SetModData. Admission before a package is written and the
// context a mod opens in both read it. It answers nil with no install table
// or archives open.
func (x *MapInspector) admittedFork(result modrt.Result) (*FrontEnd, ModDataReport, error) {
	base := x.contextBase()
	if base.table == nil || x.archives == nil || x.archives.Containers == nil {
		return nil, ModDataReport{}, nil
	}
	c, err := LoadCampaign(x.archives.Containers)
	if err != nil {
		return nil, ModDataReport{}, err
	}
	table, archives := *base.table, *x.archives
	table.Mods = mapload.ModContext{}
	archives.Containers, archives.mods = archives.Containers.Fork(), nil
	f := &FrontEnd{InstallResources: InstallResources{Table: &table, Archives: &archives, Campaign: resolved(c, nil), Humans: table.Humans}}
	if x.font != nil {
		f.Font = resolved(x.font, nil)
	}
	if !result.Files.Empty() {
		if err := x.serveModFiles(f, result.Files); err != nil {
			return nil, ModDataReport{}, err
		}
		t, err := LoadTable(f.Archives.Containers)
		if err != nil {
			return nil, ModDataReport{}, err
		}
		f.Table, f.Humans = t, t.Humans
	}
	if err := f.SetMods(result.Rules, result.Set, false); err != nil {
		return nil, ModDataReport{}, err
	}
	report, err := f.SetModData(result.Data)
	if err != nil {
		return nil, report, err
	}
	return f, report, nil
}

// openModContext runs the mod m names and builds its admitted fork, for the
// base of the install at root. It answers the issue that keeps the mod in
// the context beneath, empty when the fork was built or none can be.
//
// A mod the game refuses for its missions still opens: the missions are the
// document the editor edits, and Save admits them again. The refusal is kept
// for the overview and the context holds every other kind.
func (x *MapInspector) openModContext(m *editorMod, e mod.Entry, root string, view draftView) string {
	base := BaseID(InspectInstall(root))
	m.base = base
	result, err := modrt.Load([]mod.Entry{e}, base, nil, modrt.Options{})
	if err != nil {
		return err.Error()
	}
	if src, ok := view.drafts[mod.SpeakersFile]; ok {
		result.Speakers, err = mod.ReadSpeakers(m.id, m.dir, mod.SpeakersFile, src, view.drafts)
		if err != nil {
			return err.Error()
		}
	}
	if err := draftTerrain(&result, m, modrt.LanguageFor(base), view.drafts); err != nil {
		return err.Error()
	}
	if err := draftItems(&result, m, modrt.LanguageFor(base), view.drafts); err != nil {
		return err.Error()
	}
	m.checkLines = ModCheckLines(result)
	m.loaded = &result
	issue := ""
	if err := checkMissionProjection(x.word, m.described.Missions, result.Missions); err != nil {
		issue = err.Error()
	}
	f, report, err := x.admittedFork(result)
	if err != nil && !result.Missions.Empty() {
		m.refusal = err.Error()
		others := result
		others.Missions = mod.MissionData{}
		f, report, err = x.admittedFork(others)
	}
	if err != nil {
		return err.Error()
	}
	m.report = report
	if f == nil {
		m.catalog = characterCatalog(x.contextBase().table)
		return issue
	}
	m.context = &editorContext{table: f.Table, tiles: terrain.LoadTileset(f.Archives.Containers)}
	m.catalog = characterCatalog(f.Table)
	m.fileOverrides = f.ModFileOverrides()
	m.flow = forkFlow(f)
	return issue
}

// draftTerrain puts the project's drafts of the terrain file and of files/
// over result, the run of mod m's folder on an install of language lang,
// and counts the strips m holds. With drafts, a group whose four strips are
// not all held stays out of the context with its strips: the brush offers a
// group once it is whole, and the save's admission refuses it in the
// check's words.
func draftTerrain(result *modrt.Result, m *editorMod, lang string, drafts map[string][]byte) error {
	files := result.Files.Files
	if drafts != nil {
		if src, ok := drafts[mod.TerrainFile]; ok {
			result.Terrain = mod.TerrainData{}
			kind, _ := mod.ScriptDataKind(mod.TerrainFile)
			if err := kind.Read(mod.DataReader{Mod: m.id, Dir: m.dir, Lang: lang}, &result.Data, mod.TerrainFile, src); err != nil {
				return err
			}
		}
		var err error
		if files, err = mod.OverlayInstallFiles(m.id, lang, files, drafts); err != nil {
			return err
		}
	}
	m.strips = map[int][4]int{}
	for _, f := range files {
		if g, b, ok := terrain.ModGroupStrip(mod.FoldAddress(f.Address)); ok {
			held := m.strips[g]
			if s, err := terrain.ReadStrip(f.Bytes); err == nil {
				held[b] = len(s.SubCells)
			}
			m.strips[g] = held
		}
	}
	if drafts == nil {
		return nil
	}
	whole := func(g int) bool { held := m.strips[g]; return !slices.Contains(held[:], 0) }
	result.Terrain.Groups = slices.DeleteFunc(slices.Clone(result.Terrain.Groups), func(g mod.TerrainGroup) bool { return !whole(g.Group) })
	result.Files.Files = slices.DeleteFunc(files, func(f mod.InstallFile) bool {
		g, _, ok := terrain.ModGroupStrip(mod.FoldAddress(f.Address))
		_, row := result.Terrain.Group(g)
		return ok && !row
	})
	return nil
}

// draftItems puts the project's draft of the items file over result through
// the items kind's reader, its names read from the drafted strings, and
// hands the item layer the drafted sprites in place of the folder's. A
// project that holds no items file has none, whatever the folder holds: an
// undo past a save takes the saved file out. Records an older mod reads
// from another file stay.
func draftItems(result *modrt.Result, m *editorMod, lang string, drafts map[string][]byte) error {
	if drafts == nil {
		return nil
	}
	src := drafts[mod.ItemsFile]
	fixed := func(file string) bool { return namefold.ModTree(file) == namefold.ModTree(mod.ItemsFile) }
	d := &result.Items
	d.Rows = slices.DeleteFunc(d.Rows, func(r mod.ItemRow) bool { return fixed(r.File) })
	d.Changes = slices.DeleteFunc(d.Changes, func(c mod.ItemChange) bool { return fixed(c.File) })
	d.Materials = slices.DeleteFunc(d.Materials, func(r mod.ItemMaterial) bool { return fixed(r.File) })
	d.Weapons = slices.DeleteFunc(d.Weapons, func(r mod.WeaponRow) bool { return fixed(r.File) })
	if d.Empty() {
		d.Mods = nil
	}
	if src != nil {
		look, err := (&editorModDocument{Files: drafts}).textLookup(lang)
		if err != nil {
			return err
		}
		kind, _ := mod.WrittenKind(mod.ItemsFile)
		if err := kind.Read(mod.DataReader{Mod: m.id, Dir: m.dir, Lang: lang, Text: look}, &result.Data, mod.ItemsFile, src); err != nil {
			return err
		}
	}
	for rel, b := range drafts {
		if strings.HasPrefix(rel, mod.AssetsFolder+"/"+mod.ItemSprite.Kind+"/") && b != nil {
			if result.Items.Pictures == nil {
				result.Items.Pictures = map[string][]byte{}
			}
			result.Items.Pictures[rel] = b
		}
	}
	return nil
}

// modMissions is the one mission set of a mod the editor works in: rows and
// described's map files, joined by mod.MissionData.Listed as the install
// answers. A refusal answers the rows alone.
func (x *MapInspector) modMissions(described mod.MissionData, rows []mod.Mission) (mod.MissionData, error) {
	d := described.WithMissions(rows)
	listed, _, err := d.Listed(x.missionAnswer())
	if err != nil {
		return d, err
	}
	return listed, nil
}

// missionAnswer is the game's test (campaignMap) over the install's
// archives; nil with none open, so every number-named map is listed.
func (x *MapInspector) missionAnswer() func(n int) error {
	flow := x.installFlow()
	if flow == nil {
		return nil
	}
	f := &FrontEnd{InstallResources: InstallResources{Archives: x.archives, Campaign: resolved(flow.campaign, nil)}}
	return func(n int) error {
		if err, ok := x.missionHeld[n]; ok {
			return err
		}
		if x.missionHeld == nil {
			x.missionHeld = map[int]error{}
		}
		x.missionHeld[n] = f.campaignMap(n)
		return x.missionHeld[n]
	}
}

// description is the description m read at open; empty with no mod.
func (m *editorMod) description() mod.MissionData {
	if m == nil {
		return mod.MissionData{}
	}
	return m.described
}

// forkFlow is an admitted fork's campaign flow, at open and for the draft.
func forkFlow(f *FrontEnd) *campaignFlow {
	return newCampaignFlow(f.Campaign.Value(), tableMissions(f.Table))
}

func (s *mapInspectionEdits) draftModContext(w *editorWorkspaceState) error {
	x := s.inspector
	if x.mod == nil {
		return nil
	}
	next := *x.mod
	next.context, next.report, next.fileOverrides, next.refusal, next.checkLines = nil, ModDataReport{}, nil, "", nil
	next.flow, next.filesCheck = nil, nil
	next.files = slices.Clone(x.mod.files)
	tree, err := mod.OpenTree(next.dir)
	if err != nil {
		return err
	}
	files := map[string][]byte(nil)
	if w != nil && w.Project != nil {
		files = w.Project.Files
	}
	if err := next.readInventory(tree, files); err != nil {
		return err
	}
	root := ""
	if x.archives != nil {
		root = x.archives.Root
	}
	if issue := x.openModContext(&next, mod.Load(next.dir), root, draftView{drafts: files}); issue != "" {
		return fmt.Errorf("%s: %s", fmt.Sprintf(x.word("editor.mod.err.mod"), next.id), issue)
	}
	x.setEditorMod(&next)
	s.speakers = nil
	return nil
}
