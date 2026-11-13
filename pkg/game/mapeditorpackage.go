package game

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/namefold"
	"againrom/pkg/ui"
)

const editorPackageRecord = ".editor-files.json"

// missionPackageFiles is the files of a package of mission m, whose map
// path is its place: the manifest, the mission table, the map, the pages
// beside it in lang's folder (or the page tree, keyed below the pages
// folder), the labels when there are any, and the files m carries.
func missionPackageFiles(title, description string, m mod.Mission, lang string, labels []byte, trees ...map[string][]byte) map[string][]byte {
	place := mod.MissionPlace{Map: m.Map}
	m.Texts = ""
	p := editorPackageSpec{ID: m.Mod, Title: title, Manifest: defaultPackageManifest()}
	p.Manifest.Description = description
	manifest, _ := p.manifestBytes(nil)
	files := map[string][]byte{mod.ManifestName: manifest, mod.MissionsFile: testPlayMissions(m), m.Map: bytes.Clone(m.MapBytes)}
	for name, b := range m.TextFiles {
		files[place.Pages(lang)+"/"+name] = bytes.Clone(b)
	}
	if len(trees) != 0 {
		for name := range files {
			if strings.HasPrefix(name, place.Pages("")+"/") {
				delete(files, name)
			}
		}
		for name, b := range trees[0] {
			files[place.Pages("")+"/"+name] = bytes.Clone(b)
		}
	}
	if len(labels) > 0 {
		files[place.Labels()] = bytes.Clone(labels)
	}
	for name, b := range m.Carried {
		files[name] = bytes.Clone(b)
	}
	return files
}

// checkMissionRoute refuses the route of mission m, a guest or a replacing
// one, that the open install does not bear out. Save mod package and test play
// both stop on it before they write a file.
func (x *MapInspector) checkMissionRoute(m mod.Mission) error {
	c, err := LoadCampaign(x.archives.Containers)
	if err != nil {
		return err
	}
	switch m.Kind {
	case mod.MissionReplace:
		f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Archives: x.archives}}
		address, _ := MissionMap(m.Number)
		return f.checkReplacing(m, address)
	case mod.MissionGuest:
		return checkGuest(c, x.archives.Containers, mod.MissionData{Missions: []mod.Mission{m}}, m)
	}
	return fmt.Errorf(x.word("editor.mod.err.route"), m.Kind)
}

func (s *mapInspectionEdits) openOrCreateWorkspace(dir string) (*ui.InspectionDocument, error) {
	if dir == "" {
		return nil, errors.New(s.inspector.word("editor.mod.err.nonew"))
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	// A chosen mod.toml names the mod folder that holds it.
	if fi, err := os.Lstat(abs); err == nil && fi.Mode().IsRegular() && namefold.HostEqual(filepath.Base(abs), mod.ManifestName) {
		abs = filepath.Dir(abs)
	}
	if _, err := os.Lstat(abs); err == nil {
		if s.Dirty() {
			return nil, &ui.WorkspaceLeave{Folder: abs}
		}
		return s.openWorkspace(abs)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if _, err := s.saveWorkspacePackage(abs); err != nil {
		return nil, err
	}
	return s.project(s.model)
}

func (s *mapInspectionEdits) openWorkspace(dir string) (*ui.InspectionDocument, error) {
	x := s.inspector
	old := x.mod
	if err := x.SetMod(dir); err != nil {
		return nil, err
	}
	x.mod.copyOnly = !x.mod.owned
	// The mission list is one view of the opened mod: its first enabled
	// mission's map opens. A mod with none, or a packed one, opens over the
	// map in the editor as last saved.
	var d *ui.InspectionDocument
	var err error
	if entry, ok := x.mod.firstEnabled(); ok && !x.mod.packed {
		x.mod.selected = entry.RowID()
		d, err = x.Open(filepath.Join(dir, filepath.FromSlash(entry.Map)))
	} else if s.model == nil {
		// No map is open to show it over; the window shows the mod.
		s.workspace = nil
		s.undoWorkspace, s.redoWorkspace, s.workspaceHistory = nil, nil, false
	} else {
		d, err = x.inspect(bytes.Clone(s.saved), s.source)
	}
	if err != nil {
		x.setEditorMod(old)
	}
	return d, err
}

// firstEnabled is the mod's first mission that is not switched off.
func (m *editorMod) firstEnabled() (mod.Mission, bool) {
	for _, candidate := range m.missions.Missions {
		if !candidate.Off {
			return candidate, true
		}
	}
	return mod.Mission{}, false
}

func (s *mapInspectionEdits) selectWorkspaceMission(index int) (*ui.InspectionDocument, error) {
	return s.activateWorkspaceMission(index, true)
}

func (s *mapInspectionEdits) activateWorkspaceMission(index int, flush bool) (*ui.InspectionDocument, error) {
	x := s.inspector
	if s.workspace.Project == nil || index < 1 || index > len(s.workspace.Project.Missions) {
		return nil, errors.New(x.word("editor.mod.err.outside"))
	}
	w := cloneWorkspace(s.workspace)
	if flush {
		s.syncModDocument(w, w.Pages, w.Labels, !bytes.Equal(s.saved, s.model.Bytes()))
	}
	entry := w.Project.Missions[index-1]
	if entry.Off {
		return nil, errors.New(x.word("editor.mod.err.disabledopen"))
	}
	source := entry.Map
	if x.mod != nil {
		source = filepath.Join(x.mod.dir, filepath.FromSlash(entry.Map))
	}
	previous := ""
	if x.mod != nil {
		previous = x.mod.selected
		x.mod.selected = entry.RowID()
	}
	var candidate *ui.InspectionDocument
	var err error
	if b, ok := w.Project.Files[entry.Map]; ok {
		candidate, err = x.inspect(b, source)
	} else {
		candidate, err = x.Open(source)
	}
	if err != nil {
		if x.mod != nil {
			x.mod.selected = previous
		}
		return nil, err
	}
	next := candidate.Edits.(*mapInspectionEdits)
	w.Project.Selected = index - 1
	w.Package = modPackageSpec(entry, s.workspace.Package)
	w.Pages = next.workspace.Pages
	w.Labels = next.workspace.Labels
	if b, drafted := w.Project.Files[mod.MapLabelFile(entry.Map)]; drafted {
		if labels, err := decodeMapLabels(mod.MapLabelFile(entry.Map), b); err == nil && len(b) > 0 {
			w.Labels = labels
		} else {
			w.Labels = nil
		}
	}
	for _, folder := range []string{strings.ReplaceAll(entry.Texts, "{lang}", x.pageLanguage()), filepath.ToSlash(filepath.Join(mapTextFolder(entry.Map), x.pageLanguage()))} {
		for name, b := range w.Project.Files {
			if folder != "" && filepath.ToSlash(filepath.Dir(name)) == folder {
				w.Pages[filepath.Base(name)] = bytes.Clone(b)
			}
		}
	}
	s.model, s.source, s.workspace = next.model, source, w
	s.saved = next.saved
	s.workspaceImage = workspaceBytes(w)
	s.refreshModSelection()
	candidate.Edits = s
	return candidate, nil
}

func (s *mapInspectionEdits) saveWorkspacePackage(dir string) (string, error) {
	if err := s.prepareWorkspace(); err != nil {
		return "", err
	}
	var b []byte
	var err error
	if s.model != nil && (s.workspace.Project == nil || s.workspace.Project.Selected >= 0) {
		if err := s.gate(false); err != nil {
			return "", err
		}
		b, err = s.model.SaveBytes()
		if err != nil {
			return "", err
		}
	}
	x := s.inspector
	if x.archives == nil || x.archives.Containers == nil {
		return "", errors.New(x.word("editor.mod.err.noinstall"))
	}
	if err := campaignOf(x.archives.Game()).testPlayRefusal(); err != nil {
		return "", err
	}
	p := s.workspace.Package
	if !mod.ValidID(p.ID) || p.ID == TestPlayModID || !editorOrdinaryFilename(p.ID) {
		return "", fmt.Errorf(x.word("editor.mod.err.badid"), p.ID)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if !namefold.HostEqual(filepath.Base(abs), p.ID) {
		if err := checkMissionPackageOverlap(abs, editorPackageRecord, s.packageProtected()); err != nil {
			return "", err
		}
		return "", fmt.Errorf(x.word("editor.mod.err.foldername"), p.ID)
	}
	parent, err := editorPhysicalDirectory(filepath.Dir(abs))
	if err != nil {
		return "", err
	}
	folder := filepath.Join(parent, filepath.Base(abs))
	guard, _ := admittedOnce(func(path string) error { return fenceEditorWrite(x.archives.Root, path, "Save mod package") })
	if err := guard(parent); err != nil {
		return "", err
	}
	protected := s.packageProtected()
	if x.mod != nil && !x.mod.copyOnly && sameSaveName(x.mod.dir, folder) {
		protected = nil
	} else if _, err := os.Lstat(folder); err == nil {
		return "", fmt.Errorf(x.word("editor.mod.err.exists"), folder)
	}
	var old []string
	if x.mod == nil || x.mod.owned || !sameSaveName(x.mod.dir, folder) {
		old, err = missionPackageOwnership(folder, editorPackageRecord, protected)
	}
	if err != nil {
		return "", err
	}
	m := p.mission(b, s.workspace.Pages, s.place())
	missions := []mod.Mission{m}
	files := map[string][]byte{}
	var layout mod.LayoutChange
	if s.workspace.Project != nil {
		files, missions, m, layout, err = s.retainedPackageFiles(b)
		if err != nil {
			return "", err
		}
	} else {
		files = missionPackageFiles(p.Title, p.Manifest.Description, m, x.pageLanguage(), s.labelsForSave())
		files[mod.ManifestName], err = p.manifestBytes(nil)
		if err != nil {
			return "", err
		}
	}
	if err := x.checkMissionSet(missions); err != nil {
		return "", err
	}
	address, _ := MissionMap(m.Number)
	if m.Map != "" {
		if rows := x.CheckModMap(m, address); hasError(rows) {
			return "", s.loaderRefusal(rows)
		}
	}
	pagesRoot := mod.MissionPlace{Map: m.Map}.Pages("") + "/"
	if s.workspace.Project == nil {
		pages, err := s.exportPageTree(folder, old)
		if err != nil {
			return "", err
		}
		for name := range files {
			if strings.HasPrefix(name, pagesRoot) {
				delete(files, name)
			}
		}
		for name, b := range pages {
			files[pagesRoot+name] = b
		}
	}
	for name, b := range files {
		if !strings.HasPrefix(name, pagesRoot) || slices.Contains(old, name) {
			continue
		}
		original, err := os.ReadFile(filepath.Join(folder, filepath.FromSlash(name)))
		if err == nil && bytes.Equal(original, b) {
			delete(files, name)
		}
	}
	if err := testPlayTargets(folder, old, files); err != nil && !(x.mod != nil && !x.mod.copyOnly && sameSaveName(x.mod.dir, folder)) {
		return "", err
	}
	plan := missionPackagePlan{Check: x.admitMissionPackage, Expected: missions, Word: x.word}
	if s.workspace.Project != nil {
		plan.Source = s.retainedSource(folder)
	}
	from := folder
	if x.mod != nil {
		from = x.mod.dir
	}
	if err := x.writeEditorProject(folder, p.ID, old, files, plan, plan.Source == folder || len(old) > 0); err != nil {
		return "", err
	}
	if err := x.SetMod(folder); err != nil {
		return "", err
	}
	x.mod.selected = m.RowID()
	s.workspace.Project = x.mod.document()
	s.workspace.Package.ID, s.workspace.Package.Title = x.mod.id, x.mod.title
	s.workspace.Package.Manifest, _ = mod.ParseManifest(x.mod.manifest)
	s.workspace.Project.Selected = -1
	for i, entry := range x.mod.missions.Missions {
		if m.Map != "" && entry.RowID() == m.RowID() {
			s.workspace.Project.Selected = i
			s.workspace.Package = modPackageSpec(entry, s.workspace.Package)
			break
		}
	}
	if m.Map != "" {
		s.source = filepath.Join(folder, filepath.FromSlash(m.Map))
	}
	s.rebaseHistory(layout, from, folder)
	s.workspaceImage = workspaceBytes(s.workspace)
	if s.model != nil {
		s.saved = s.model.Bytes()
	}
	s.savedWorkspace = s.workspaceImage
	return s.source, nil
}
