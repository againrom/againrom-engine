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

func missionPackageFiles(title, description string, m mod.Mission, lang string) map[string][]byte {
	if m.Texts != "" {
		m.Texts = testPlayTexts + "/{lang}"
	}
	manifest := fmt.Sprintf("id = %q\ntitle = %q\nversion = \"1.0.0\"\nauthor = \"Againrom\"\ndescription = %q\napi = 1\napplies-to = [\"rom1\"]\nentry = \"main.star\"\n", m.Mod, title, description)
	files := map[string][]byte{mod.ManifestName: []byte(manifest), "main.star": []byte(testPlayEntry), mod.MissionsFile: testPlayMissions(m), testPlayMap: bytes.Clone(m.MapBytes)}
	for name, b := range m.TextFiles {
		files[testPlayTexts+"/"+lang+"/"+name] = bytes.Clone(b)
	}
	return files
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
			return nil, errors.New("save or undo changes before opening another mod package")
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
	x.mod.selected = entry.Key
	d, err := x.Open(filepath.Join(dir, filepath.FromSlash(entry.Map)))
	if err != nil {
		x.mod = old
	}
	return d, err
}

func (s *mapInspectionEdits) selectWorkspaceMission(index int) (*ui.InspectionDocument, error) {
	if s.Dirty() {
		return nil, errors.New("save or undo changes before selecting another source mission")
	}
	x := s.inspector
	if x.mod == nil || index < 1 || index > len(x.mod.missions.Missions) {
		return nil, errors.New("source mission selection is outside the package")
	}
	old := x.mod.selected
	entry := x.mod.missions.Missions[index-1]
	x.mod.selected = entry.Key
	d, err := x.Open(filepath.Join(x.mod.dir, filepath.FromSlash(entry.Map)))
	if err != nil {
		x.mod.selected = old
	}
	return d, err
}

func (s *mapInspectionEdits) saveWorkspacePackage(dir string) (string, error) {
	if err := s.prepareWorkspace(); err != nil {
		return "", err
	}
	if err := s.gate(); err != nil {
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
	c, err := LoadCampaign(x.archives.Containers)
	if err != nil {
		return "", err
	}
	if m.Kind == mod.MissionReplace {
		f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Archives: x.archives}}
		address, _ := MissionMap(m.Number)
		if err := f.checkReplacing(m, address); err != nil {
			return "", err
		}
	} else if m.Kind == mod.MissionGuest {
		if err := checkGuest(c, x.archives.Containers, mod.MissionData{Missions: []mod.Mission{m}}, m); err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("unknown mission route %q", m.Kind)
	}
	address, _ := MissionMap(m.Number)
	if rows := x.CheckModMap(m, address); hasError(rows) {
		return "", &ui.ReportRefusal{Findings: rows}
	}
	files := missionPackageFiles(p.Title, "Map editor mission package", m, x.pageLanguage())
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
	stage, err := stageMissionPackage(parent, folder, x.archives.Root, p.ID, editorPackageRecord, old, files, guard)
	if err != nil {
		return "", err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(stage)
		}
	}()
	backup := filepath.Join(stage, "previous")
	if len(old) > 0 {
		if err := os.Rename(folder, backup); err != nil {
			return "", err
		}
	}
	if err := os.Rename(filepath.Join(stage, p.ID), folder); err != nil {
		if len(old) > 0 {
			if restore := os.Rename(backup, folder); restore != nil {
				cleanup = false
				return "", fmt.Errorf("publish failed: %v; old package retained at %s: %v", err, backup, restore)
			}
		}
		return "", err
	}
	x.mod = &editorMod{dir: folder, id: p.ID, title: p.Title, owned: true, missions: mod.MissionData{Missions: []mod.Mission{m}}}
	s.source = filepath.Join(folder, filepath.FromSlash(m.Map))
	s.saved, s.savedWorkspace = s.model.Bytes(), s.workspaceImage
	return s.source, nil
}
