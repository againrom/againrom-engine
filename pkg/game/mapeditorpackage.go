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
	"againrom/pkg/ui"
)

const editorPackageRecord = ".editor-files.json"

func missionPackageFiles(title, description string, m mod.Mission, lang string, trees ...map[string][]byte) map[string][]byte {
	if m.Texts != "" {
		m.Texts = testPlayTexts + "/{lang}"
	}
	manifest := fmt.Sprintf("id = %q\ntitle = %q\nversion = \"1.0.0\"\nauthor = \"Againrom\"\ndescription = %q\napi = 1\napplies-to = [\"rom1\"]\nentry = \"main.star\"\n", m.Mod, title, description)
	files := map[string][]byte{mod.ManifestName: []byte(manifest), "main.star": []byte(testPlayEntry), mod.MissionsFile: testPlayMissions(m), testPlayMap: bytes.Clone(m.MapBytes)}
	for name, b := range m.TextFiles {
		files[testPlayTexts+"/"+lang+"/"+name] = bytes.Clone(b)
	}
	if len(trees) != 0 {
		for name := range files {
			if strings.HasPrefix(name, testPlayTexts+"/") {
				delete(files, name)
			}
		}
		for name, b := range trees[0] {
			files[testPlayTexts+"/"+name] = bytes.Clone(b)
		}
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
	return fmt.Errorf("unknown mission route %q", m.Kind)
}

func (s *mapInspectionEdits) openOrCreateWorkspace(dir string) (*ui.InspectionDocument, error) {
	if dir == "" {
		return nil, errors.New("choose an explicit new mod folder or an editor-owned package")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
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
	if len(x.mod.missions.Missions) == 0 {
		x.mod = old
		return nil, fmt.Errorf("%s declares no mission", dir)
	}
	x.mod.copyOnly = !x.mod.owned
	entry := x.mod.missions.Missions[0]
	for _, candidate := range x.mod.missions.Missions {
		if !candidate.Off {
			entry = candidate
			break
		}
	}
	if entry.Off {
		x.mod = old
		return nil, errors.New("mod has no enabled mission")
	}
	x.mod.selected = entry.Key
	d, err := x.Open(filepath.Join(dir, filepath.FromSlash(entry.Map)))
	if err != nil {
		x.mod = old
	}
	return d, err
}

func (s *mapInspectionEdits) selectWorkspaceMission(index int) (*ui.InspectionDocument, error) {
	return s.activateWorkspaceMission(index, true)
}

func (s *mapInspectionEdits) activateWorkspaceMission(index int, flush bool) (*ui.InspectionDocument, error) {
	x := s.inspector
	if s.workspace.Project == nil || index < 1 || index > len(s.workspace.Project.Missions) {
		return nil, errors.New("source mission selection is outside the package")
	}
	w := cloneWorkspace(s.workspace)
	if flush {
		s.syncModDocument(w, w.Pages, !bytes.Equal(s.saved, s.model.Bytes()))
	}
	entry := w.Project.Missions[index-1]
	if entry.Off {
		return nil, errors.New("disabled declaration has no map to open")
	}
	source := entry.Map
	if x.mod != nil {
		source = filepath.Join(x.mod.dir, filepath.FromSlash(entry.Map))
	}
	previous := ""
	if x.mod != nil {
		previous = x.mod.selected
		x.mod.selected = entry.Key
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
	w.Package = modPackageSpec(entry, s.workspace.Package.ID, s.workspace.Package.Title)
	w.Pages = next.workspace.Pages
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
	if err := s.gate(false); err != nil {
		return "", err
	}
	b, err := s.model.SaveBytes()
	if err != nil {
		return "", err
	}
	x := s.inspector
	if x.archives == nil || x.archives.Containers == nil {
		return "", errors.New("a mod package needs an open ROM1 install")
	}
	if err := campaignOf(x.archives.Game()).testPlayRefusal(); err != nil {
		return "", err
	}
	p := s.workspace.Package
	if err := s.unchangedModSource(); err != nil {
		return "", err
	}
	if !mod.ValidID(p.ID) || p.ID == TestPlayModID || !editorOrdinaryFilename(p.ID) {
		return "", fmt.Errorf("invalid author mod ID %q", p.ID)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(filepath.Base(abs), p.ID) {
		return "", fmt.Errorf("folder must be named %s, the mod ID", p.ID)
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
	if x.mod != nil && x.mod.owned && sameSaveName(x.mod.dir, folder) {
		protected = nil
	} else if _, err := os.Lstat(folder); err == nil {
		return "", fmt.Errorf("%s already exists; open it before saving", folder)
	}
	old, err := missionPackageOwnership(folder, editorPackageRecord, protected)
	if err != nil {
		return "", err
	}
	m := p.mission(b, s.workspace.Pages)
	missions := []mod.Mission{m}
	files := map[string][]byte{}
	if s.workspace.Project != nil {
		files, missions, m, err = s.retainedPackageFiles(b)
		if err != nil {
			return "", err
		}
	} else {
		files = missionPackageFiles(p.Title, "Map editor mission package", m, x.pageLanguage())
	}
	if err := x.checkMissionSet(missions); err != nil {
		return "", err
	}
	address, _ := MissionMap(m.Number)
	if rows := x.CheckModMap(m, address); hasError(rows) {
		return "", s.loaderRefusal(rows)
	}
	if s.workspace.Project == nil {
		pages, err := s.exportPageTree(folder, old)
		if err != nil {
			return "", err
		}
		for name := range files {
			if strings.HasPrefix(name, testPlayTexts+"/") {
				delete(files, name)
			}
		}
		for name, b := range pages {
			files[testPlayTexts+"/"+name] = b
		}
		if len(pages) > 0 {
			m.Texts = testPlayTexts + "/{lang}"
		} else {
			m.Texts = ""
		}
		files[mod.MissionsFile] = testPlayMissions(m)
	}
	for name, b := range files {
		if !strings.HasPrefix(name, testPlayTexts+"/") || slices.Contains(old, name) {
			continue
		}
		original, err := os.ReadFile(filepath.Join(folder, filepath.FromSlash(name)))
		if err == nil && bytes.Equal(original, b) {
			delete(files, name)
		}
	}
	if err := testPlayTargets(folder, old, files); err != nil {
		return "", err
	}
	plan := missionPackagePlan{Check: x.admitMissionPackage, Expected: missions}
	if s.workspace.Project != nil {
		plan.Source = s.retainedSource(folder)
	}
	stage, err := stageMissionPackage(parent, folder, x.archives.Root, p.ID, editorPackageRecord, old, files, guard, plan)
	if err != nil {
		return "", err
	}
	if err := installMissionPackage(stage, folder, p.ID, len(old) > 0); err != nil {
		return "", err
	}
	if err := x.SetMod(folder); err != nil {
		return "", err
	}
	x.mod.selected = m.Key
	s.workspace.Project = x.mod.document()
	s.source = filepath.Join(folder, filepath.FromSlash(m.Map))
	s.workspaceImage = workspaceBytes(s.workspace)
	s.saved, s.savedWorkspace = s.model.Bytes(), s.workspaceImage
	return s.source, nil
}
