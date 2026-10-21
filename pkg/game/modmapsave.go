package game

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"againrom/pkg/locale"
	"againrom/pkg/mod"
)

// editorMod is the mod folder the map editor's Save writes into, with the
// missions its data/missions.toml declares in the install's language.
type editorMod struct {
	dir      string
	id       string
	missions mod.MissionData
	owned    bool
	copyOnly bool
	selected string
	title    string
}

// SetMod names the mod folder dir the editor's Save writes into: Save writes
// the edited map to the path the mod's mission entry for that map names. dir
// must be a mod folder outside the install whose data/missions.toml reads.
func (x *MapInspector) SetMod(dir string) error {
	if t, err := mod.OpenTree(dir); err == nil && t.Packed() {
		return fmt.Errorf("%s is a packed mod; the editor saves into a mod folder", dir)
	}
	e := mod.Load(dir)
	if e.Err != nil {
		return fmt.Errorf("mod folder %s: %v", dir, e.Err)
	}
	root := ""
	if x.archives != nil {
		root = x.archives.Root
	}
	if root != "" {
		inside, err := pathWithin(root, dir)
		if err != nil {
			return err
		}
		if inside {
			return fmt.Errorf("mod folder %s is inside the install %s; the editor never writes into an install", dir, root)
		}
	}
	lang := locale.Fallback
	if root != "" {
		if l, ok := locale.ByBaseSuffix(BaseID(InspectInstall(root))); ok {
			lang = l.Code
		}
	}
	src, _, err := mod.ReadInside(dir, mod.MissionsFile)
	if err != nil {
		return fmt.Errorf("mod %q declares no missions: %v", e.Manifest.ID, err)
	}
	look, err := mod.TextLookup(dir, lang)
	if err != nil {
		return fmt.Errorf("mod %q: %v", e.Manifest.ID, err)
	}
	d, err := mod.ParseMissions(e.Manifest.ID, mod.MissionsFile, src, look, lang)
	if err != nil {
		return fmt.Errorf("mod %q: %v", e.Manifest.ID, err)
	}
	owned := false
	if _, err := os.Lstat(filepath.Join(dir, editorPackageRecord)); err == nil {
		if _, err := missionPackageOwnership(dir, editorPackageRecord, nil); err != nil {
			return err
		}
		owned = true
	}
	x.mod = &editorMod{dir: dir, id: e.Manifest.ID, missions: d, owned: owned, title: e.Manifest.Title}
	return nil
}

// entryFor is the mission of the mod that names the map opened from source:
// the mission whose number an installed scenario map has, or the mission
// whose map path is the opened host file.
func (m *editorMod) entryFor(source string) (mod.Mission, bool) {
	if isNewSource(source) {
		return m.unwrittenEntry()
	}
	if stem, ok := strings.CutPrefix(source, scenarioPrefix); ok {
		n, err := strconv.Atoi(strings.TrimSuffix(stem, ".alm"))
		for _, e := range m.missions.Missions {
			if err == nil && e.Number == n {
				return e, true
			}
		}
		return mod.Mission{}, false
	}
	if strings.HasPrefix(source, "loose/") {
		return mod.Mission{}, false
	}
	opened, err := os.Stat(source)
	if err != nil {
		return mod.Mission{}, false
	}
	entries := m.missions.Missions
	if m.selected != "" {
		entries = append([]mod.Mission(nil), entries...)
		for n, e := range entries {
			if e.Key == m.selected {
				entries[0], entries[n] = entries[n], entries[0]
				break
			}
		}
	}
	for _, e := range entries {
		if st, err := os.Stat(filepath.Join(m.dir, filepath.FromSlash(e.Map))); err == nil && os.SameFile(opened, st) {
			return e, true
		}
	}
	return mod.Mission{}, false
}

// ModTarget names the mod Save writes into; empty when none is named.
func (s *mapInspectionEdits) ModTarget() string {
	if s.inspector == nil || s.inspector.mod == nil {
		return ""
	}
	if s.inspector.mod.copyOnly {
		return s.inspector.mod.id + " (read-only; W: copy)"
	}
	return s.inspector.mod.id
}

// SaveIntoMod writes the edited map over the file the mod's mission entry for
// this map names, and refuses when no entry names it.
func (s *mapInspectionEdits) SaveIntoMod() (string, error) {
	m := s.inspector.mod
	if m == nil {
		return "", errors.New("no mod folder is named")
	}
	if m.copyOnly {
		return "", errors.New("this imported mod is read-only; choose a new mod ID and folder with W to save an owned copy")
	}
	if m.owned {
		return s.saveWorkspacePackage(m.dir)
	}
	if !bytes.Equal(s.savedWorkspace, s.workspaceImage) {
		return "", errors.New("authored pages need an editor-owned mod package; choose a new folder with W")
	}
	e, ok := m.entryFor(s.source)
	if !ok {
		return "", fmt.Errorf("no mission of mod %q names this map (%s); add its entry to %s", m.id, s.source, mod.MissionsFile)
	}
	if err := s.gate(); err != nil {
		return "", err
	}
	b, err := s.model.SaveBytes()
	if err != nil {
		return "", fmt.Errorf("mod %q: %w", m.id, err)
	}
	root := ""
	if s.inspector.archives != nil {
		root = s.inspector.archives.Root
	}
	target, err := writeModMap(root, m.dir, e.Map, b)
	if err != nil {
		return "", fmt.Errorf("mod %q: %w", m.id, err)
	}
	s.saved, s.source = s.model.Bytes(), target
	return target, nil
}

// checkModMapDirectory refuses, before any directory is created, a target
// directory whose deepest existing ancestor resolves outside the mod folder or
// inside the install at root.
func checkModMapDirectory(root, dir, target, rel string) error {
	folder, err := editorPhysicalDirectory(dir)
	if err != nil {
		return fmt.Errorf("resolve mod folder: %w", err)
	}
	lexical, err := filepath.Rel(dir, target)
	if err != nil || lexical == ".." || strings.HasPrefix(lexical, ".."+string(filepath.Separator)) || filepath.IsAbs(lexical) {
		return fmt.Errorf("%s leads outside the mod folder", rel)
	}
	existing := target
	for {
		if _, err := os.Lstat(existing); err == nil {
			break
		}
		next := filepath.Dir(existing)
		if next == existing {
			return fmt.Errorf("%s leads outside the mod folder", rel)
		}
		existing = next
	}
	physical, err := editorPhysicalDirectory(existing)
	if err != nil {
		return fmt.Errorf("resolve map output directory: %w", err)
	}
	if inside, err := pathWithin(folder, physical); err != nil || !inside {
		return fmt.Errorf("%s leads outside the mod folder", rel)
	}
	return fenceEditorWrite(root, physical, "Save")
}

// writeModMap writes data to the slash path rel inside the mod folder dir,
// replacing the file there. The file stays inside the folder and outside the
// install at root; a map larger than a mod may load is refused.
func writeModMap(root, dir, rel string, data []byte) (string, error) {
	if len(data) > mod.MaxFileBytes {
		return "", fmt.Errorf("the map is %d bytes; a mod map holds at most %d", len(data), mod.MaxFileBytes)
	}
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := checkModMapDirectory(root, dir, filepath.Dir(full), rel); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}
	parent, err := editorPhysicalDirectory(filepath.Dir(full))
	if err != nil {
		return "", fmt.Errorf("resolve map output directory: %w", err)
	}
	folder, err := editorPhysicalDirectory(dir)
	if err != nil {
		return "", fmt.Errorf("resolve mod folder: %w", err)
	}
	if inside, err := pathWithin(folder, parent); err != nil || !inside {
		return "", fmt.Errorf("%s leads outside the mod folder", rel)
	}
	if err := fenceEditorWrite(root, parent, "Save"); err != nil {
		return "", err
	}
	target := filepath.Join(parent, filepath.Base(full))
	if st, err := os.Lstat(target); err == nil && !st.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a file", rel)
	}
	// The new map is written beside the old one and moved over it, so a failed
	// write leaves the old map whole. A leading dot keeps it out of the mod's
	// content digest.
	tmp, err := os.CreateTemp(parent, ".mapedit-*.alm")
	if err != nil {
		return "", err
	}
	_, writeErr := tmp.Write(data)
	if writeErr == nil {
		writeErr = tmp.Sync()
	}
	closeErr := tmp.Close()
	if writeErr == nil {
		writeErr = closeErr
	}
	if writeErr == nil {
		writeErr = os.Rename(tmp.Name(), target)
	}
	if writeErr != nil {
		// This exact file was created by this call, never an input.
		_ = os.Remove(tmp.Name())
		return "", writeErr
	}
	return target, nil
}
