package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/render/terrain"
	"fmt"
	"slices"
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
		layer := f.modFileLayer()
		layer.order = result.Files.Order
		if err := layer.serveInstallFiles(f.Archives, result.Files.Files); err != nil {
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
func (x *MapInspector) openModContext(m *editorMod, e mod.Entry, root string, drafts ...map[string][]byte) string {
	result, err := modrt.Load([]mod.Entry{e}, BaseID(InspectInstall(root)), nil, modrt.Options{})
	if err != nil {
		return err.Error()
	}
	if len(drafts) > 0 {
		files := drafts[0]
		if src, ok := files[mod.SpeakersFile]; ok {
			result.Speakers, err = mod.ReadSpeakers(m.id, m.dir, mod.SpeakersFile, src, files)
			if err != nil {
				return err.Error()
			}
		}

	}
	m.checkLines = ModCheckLines(result)
	issue := ""
	if err := checkMissionProjection(x.word, m.missions.Missions, result.Missions); err != nil {
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
	return issue
}

func (s *mapInspectionEdits) draftModContext(w *editorWorkspaceState) error {
	x := s.inspector
	if x.mod == nil {
		return nil
	}
	next := *x.mod
	next.context, next.report, next.fileOverrides, next.refusal, next.checkLines = nil, ModDataReport{}, nil, "", nil
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
	if issue := x.openModContext(&next, mod.Load(next.dir), root, files); issue != "" {
		return fmt.Errorf("%s: %s", fmt.Sprintf(x.word("editor.mod.err.mod"), next.id), issue)
	}
	x.setEditorMod(&next)
	s.speakers = nil
	return nil
}
