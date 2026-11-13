package game

import (
	"bytes"
	"errors"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/namefold"
	"againrom/pkg/ui"
)

// The open maps of one mod. Each open map has its own edits: its model, its
// pages and labels and its undo history. The mod's project is one: the edits
// of the map shown hold it, with every open map's draft in its files, and
// Shown carries it from the map left to the map shown.

// InMod reports that the mod these edits belong to is open, so another map
// opens beside this one. A map of a mod since left has no map beside it.
func (s *mapInspectionEdits) InMod() bool {
	return s.inspector != nil && s.inspector.mod != nil && s.modDir == s.inspector.mod.dir
}

// modFolder is the open mod's folder, empty for none.
func (x *MapInspector) modFolder() string {
	if x.mod == nil {
		return ""
	}
	return x.mod.dir
}

// joinMod makes these edits and every open map's the open mod's: a save
// into a mod's folder opens that mod.
func (s *mapInspectionEdits) joinMod() {
	s.modDir = s.inspector.modFolder()
	for _, t := range s.inspector.openMaps {
		t.modDir = s.modDir
	}
}

// MissionNumber is the number of the mod's mission this map is; 0: none.
func (s *mapInspectionEdits) MissionNumber() int {
	if m, ok := s.ownMission(s.project0()); ok {
		return m.Number
	}
	return 0
}

// project0 is the project these edits hold, nil with none.
func (s *mapInspectionEdits) project0() *editorModDocument {
	if s.workspace == nil {
		return nil
	}
	return s.workspace.Project
}

// ownMission is the row of project d that declares this map: the row of the
// key these edits have selected. A map of the mod no row names answers a
// mission of its path alone, with no key and no number, so its drafts are
// found as a row's are.
func (s *mapInspectionEdits) ownMission(d *editorModDocument) (mod.Mission, bool) {
	own := s.project0()
	if own == nil || d == nil {
		return mod.Mission{}, false
	}
	if own.Selected < 0 || own.Selected >= len(own.Missions) {
		return mod.Mission{Map: own.Loose}, own.Loose != ""
	}
	key := own.Missions[own.Selected].RowID()
	for _, m := range d.Missions {
		if m.RowID() == key {
			return m, true
		}
	}
	return mod.Mission{}, false
}

// mapDraftNames is the drafts of project d that belong to the map of row m:
// the map, the labels beside it and the pages of its folders.
func (x *MapInspector) mapDraftNames(d *editorModDocument, m mod.Mission) []string {
	lang := x.pageLanguage()
	folders := []string{mapTextFolder(m.Map), path.Join(mapTextFolder(m.Map), lang)}
	if texts := strings.ReplaceAll(m.Texts, "{lang}", lang); texts != "" {
		folders = append(folders, path.Clean(texts))
	}
	var names []string
	for name := range d.Files {
		if name == m.Map || name == mod.MapLabelFile(m.Map) || slices.Contains(folders, path.Dir(name)) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// ownDraftNames is the drafts of project d that belong to this map.
func (s *mapInspectionEdits) ownDraftNames(d *editorModDocument) []string {
	if m, ok := s.ownMission(d); ok {
		return s.inspector.mapDraftNames(d, m)
	}
	return nil
}

// openEdits is the edits of the open maps: the ones the window last named,
// while these are among them, else these alone.
func (s *mapInspectionEdits) openEdits() []*mapInspectionEdits {
	if slices.Contains(s.inspector.openMaps, s) {
		return s.inspector.openMaps
	}
	return []*mapInspectionEdits{s}
}

// mapDirtyKeep is the last MapDirty answer of the pages and labels, for the
// workspace and the saved state it was made of.
type mapDirtyKeep struct {
	workspace *editorWorkspaceState
	saved     *byte
	dirty     bool
}

// MapDirty reports this map's own unsaved changes: its cells and records,
// its pages and its labels. A change to the mod, or to another open map, is
// not one.
func (s *mapInspectionEdits) MapDirty() bool {
	if s.model != nil && !bytes.Equal(s.saved, s.model.Bytes()) {
		return true
	}
	d := s.project0()
	if d == nil {
		return s.Dirty()
	}
	m, ok := s.ownMission(d)
	if !ok || len(s.savedWorkspace) == 0 {
		return false
	}
	if k := &s.dirtyKeep; k.workspace == s.workspace && k.saved == &s.savedWorkspace[0] {
		return k.dirty
	}
	var saved editorWorkspaceState
	decodeWorkspace(s.savedWorkspace, &saved)
	var base map[string][]byte
	if saved.Project != nil {
		base = saved.Project.Files
	}
	dirty := false
	for _, name := range s.inspector.mapDraftNames(d, m) {
		if was, held := base[name]; name != m.Map && (!held || !bytes.Equal(was, d.Files[name])) {
			dirty = true
		}
	}
	s.dirtyKeep = mapDirtyKeep{workspace: s.workspace, saved: &s.savedWorkspace[0], dirty: dirty}
	return dirty
}

// openModMap is the one switch to a map of the mod: it opens the map of the
// project's mission index, or with loose the map at that path no row names,
// beside the open maps: a new edits of the one opener, holding the project
// as it stands. A map already open answers ui.MapOpen with its edits; this
// map's own other declaration is selected in place.
func (s *mapInspectionEdits) openModMap(index int, loose string) (*ui.InspectionDocument, error) {
	x := s.inspector
	d := s.project0()
	if d == nil || loose == "" && (index < 1 || index > len(d.Missions)) {
		return nil, errors.New(x.word("editor.mod.err.outside"))
	}
	rel := loose
	if rel == "" {
		rel = d.Missions[index-1].Map
	}
	if x.mod != nil {
		source := filepath.Join(x.mod.dir, filepath.FromSlash(rel))
		for _, t := range s.openEdits() {
			if t.model == nil || !sameSaveName(t.source, source) {
				continue
			}
			if t != s {
				return nil, &ui.MapOpen{Edits: t}
			}
			if loose != "" || index-1 == d.Selected {
				return nil, nil
			}
			return s.activateWorkspaceMission(index, true)
		}
	}
	next := &mapInspectionEdits{inspector: x, model: s.model, source: s.source, saved: s.saved, clock: s.clock, modDir: s.modDir,
		workspace: s.workspace, savedWorkspace: s.savedWorkspace, workspaceImage: s.workspaceImage}
	doc, err := next.activateWorkspaceMission(index, true, loose)
	if err != nil {
		s.refreshModSelection()
		return nil, err
	}
	return doc, nil
}

// Shown makes these edits the ones the window shows. open is the edits of
// every open map, these among them; left is the edits shown before, nil for
// none. The mod's project comes with the map left: its rows, its manifest
// and every open map's draft, the left map's cells flushed into it, and the
// drafts of a left map that is no longer open dropped.
func (s *mapInspectionEdits) Shown(left ui.InspectionEdits, open []ui.InspectionEdits) error {
	x := s.inspector
	x.openMaps = x.openMaps[:0]
	for _, o := range open {
		if t, ok := o.(*mapInspectionEdits); ok {
			x.openMaps = append(x.openMaps, t)
		}
	}
	from, _ := left.(*mapInspectionEdits)
	if from == nil || from == s || from.project0() == nil || x.mod == nil {
		if m, ok := s.ownMission(s.project0()); ok && x.mod != nil {
			x.mod.selected = m.RowID()
		}
		return nil
	}
	if err := s.prepareWorkspace(); err != nil {
		return err
	}
	if s.project0() == nil {
		return nil
	}
	p := from.workspace.Project.clone()
	if m, ok := from.ownMission(p); ok && from.model != nil {
		if bytes.Equal(from.saved, from.model.Bytes()) {
			delete(p.Files, m.Map)
		} else {
			p.Files[m.Map] = from.model.Bytes()
		}
	}
	if !slices.Contains(x.openMaps, from) {
		for _, name := range from.ownDraftNames(p) {
			if !slices.Contains(s.ownDraftNames(p), name) {
				delete(p.Files, name)
			}
		}
	}
	s.carry(p, from.workspace.Package)
	var saved, base editorWorkspaceState
	decodeWorkspace(s.savedWorkspace, &saved)
	decodeWorkspace(from.savedWorkspace, &base)
	if base.Project != nil {
		saved.Project = base.Project
		saved.Project.Selected = s.workspace.Project.Selected
		saved.Package.ID, saved.Package.Title, saved.Package.Manifest = base.Package.ID, base.Package.Title, base.Package.Manifest
		s.savedWorkspace = workspaceBytes(&saved)
	}
	s.refreshModSelection()
	return nil
}

// carry makes project p, with the mod's fields of pack, the project these
// edits hold: this map's row selected in it and its mission fields read
// from that row. Undo and redo steps recorded before it keep the project.
func (s *mapInspectionEdits) carry(p *editorModDocument, pack editorPackageSpec) {
	w := cloneWorkspace(s.workspace)
	key, keyed, loose := "", false, ""
	if m, ok := s.ownMission(p); ok {
		key, keyed = m.RowID(), m.RowID() != ""
		if !keyed {
			loose = m.Map
		}
	}
	p.Selected, p.Loose = -1, loose
	w.Package.ID, w.Package.Title, w.Package.Manifest = pack.ID, pack.Title, mod.CloneManifest(pack.Manifest)
	for i, m := range p.Missions {
		if keyed && m.RowID() == key {
			p.Selected = i
			w.Package = modPackageSpec(m, w.Package)
			break
		}
	}
	w.Project = p
	s.workspace, s.workspaceImage = w, workspaceBytes(w)
	s.carried++
}

// carriedOver is the workspace an undo or redo step recorded before the
// project was carried restores: the step's own map, pages and labels, in the
// project as it stands now with this map's mission row and drafts as the
// step held them.
func (s *mapInspectionEdits) carriedOver(step *editorWorkspaceState) *editorWorkspaceState {
	now := s.project0()
	if step == nil || step.Project == nil || now == nil {
		return step
	}
	w := cloneWorkspace(step)
	p := now.clone()
	for _, name := range s.ownDraftNames(p) {
		delete(p.Files, name)
	}
	for _, name := range s.ownDraftNames(step.Project) {
		p.Files[name] = bytes.Clone(step.Project.Files[name])
	}
	if row, ok := s.ownMission(step.Project); ok && row.RowID() != "" {
		for i := range p.Missions {
			if p.Missions[i].RowID() == row.RowID() {
				p.Missions[i] = row
			}
		}
	}
	w.Project = p
	w.Package.ID, w.Package.Title, w.Package.Manifest = s.workspace.Package.ID, s.workspace.Package.Title, mod.CloneManifest(s.workspace.Package.Manifest)
	if p.Selected >= 0 && p.Selected < len(p.Missions) {
		w.Package = modPackageSpec(p.Missions[p.Selected], w.Package)
	}
	return w
}

// wholeStep is the workspace an undo or redo step that changes which map
// these edits hold restores: the step's project, rows and selection as it
// held them, with the drafts the other open maps have now.
func (s *mapInspectionEdits) wholeStep(step *editorWorkspaceState) *editorWorkspaceState {
	now := s.project0()
	if step == nil || step.Project == nil || now == nil {
		return step
	}
	w := cloneWorkspace(step)
	p := step.Project.clone()
	for _, t := range s.openEdits() {
		if t == s {
			continue
		}
		for _, name := range t.ownDraftNames(now) {
			if b, ok := now.Files[name]; ok {
				p.Files[name] = bytes.Clone(b)
			} else {
				delete(p.Files, name)
			}
		}
	}
	w.Project = p
	return w
}

// heldDrafts takes the other open maps' drafts out of the project these
// edits hold, so a save writes this map and the mod and leaves the other
// maps' files as they are. It answers the drafts taken and the workspace as
// it was.
func (s *mapInspectionEdits) heldDrafts() (map[string][]byte, *editorWorkspaceState) {
	was := s.workspace
	d := s.project0()
	if d == nil {
		return nil, was
	}
	own := s.ownDraftNames(d)
	held := map[string][]byte{}
	for _, t := range s.openEdits() {
		if t == s {
			continue
		}
		for _, name := range t.ownDraftNames(d) {
			if !slices.Contains(own, name) {
				held[name] = d.Files[name]
			}
		}
	}
	if len(held) == 0 {
		return nil, was
	}
	w := cloneWorkspace(was)
	for name := range held {
		delete(w.Project.Files, name)
	}
	s.workspace = w
	return held, was
}

// MapSource is the path the map is saved at.
func (s *mapInspectionEdits) MapSource() string { return s.source }

// returnDrafts puts the held drafts back into the project a save left, at
// the paths the save's move c gave their files. The save of the mod at from
// went into to: every other open map, and each step of every open map's
// history, that names a file of from names that file's place in to.
func (s *mapInspectionEdits) returnDrafts(held map[string][]byte, c mod.LayoutChange, from, to string) {
	moved := func(name string) string {
		for _, mv := range c.Moves {
			if namefold.ModTree(name) == namefold.ModTree(mv.From) {
				return mv.To
			}
		}
		return name
	}
	follow := func(path string) string {
		rel, err := filepath.Rel(from, path)
		if err != nil || !filepath.IsLocal(rel) || sameSaveName(from, to) && moved(filepath.ToSlash(rel)) == filepath.ToSlash(rel) {
			return path
		}
		return filepath.Join(to, filepath.FromSlash(moved(filepath.ToSlash(rel))))
	}
	for _, t := range s.openEdits() {
		if t != s {
			t.source = follow(t.source)
		}
		for _, history := range [][]editorWorkspaceSnapshot{t.undoWorkspace, t.redoWorkspace} {
			for i := range history {
				history[i].source = follow(history[i].source)
			}
		}
	}
	if len(held) == 0 {
		return
	}
	for name, b := range held {
		s.workspace.Project.Files[moved(name)] = b
	}
	s.workspaceImage = workspaceBytes(s.workspace)
}
