package game

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"againrom/pkg/hostfile"
	"againrom/pkg/locale"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// editorMod is the mod folder the map editor's Save writes into, with the
// missions its data/missions.toml declares in the install's language, and
// the context its admitted fork gives the editor: the table and tileset, the
// character catalog, and what applying the mod's data reported. Its files,
// the kinds whose files it holds and its check report make its overview.
type editorMod struct {
	source, manifest []byte
	digest           string
	catalog          []mod.Character
	context          *editorContext
	report           ModDataReport
	fileOverrides    []string
	files, kinds     []string
	checkLines       []string
	refusal          string // why the game refuses the mod's missions; empty when it admits them
	contextIssue     string
	dir              string
	id               string
	missions         mod.MissionData
	owned            bool
	packed           bool
	copyOnly         bool
	selected         string
	title            string
}

// SetMod is the editor's one open route: it opens the mod at dir, a folder or
// a packed file holding any kinds, runs it and makes it the mod the editor
// works in. Save writes the edited map to the path the mod's mission entry
// for that map names. A folder must lie outside the install; a packed mod
// opens read-only.
func (x *MapInspector) SetMod(dir string) error {
	e := mod.Load(dir)
	if e.Err != nil {
		return fmt.Errorf("mod folder %s: %v", dir, e.Err)
	}
	tree, err := mod.OpenTree(dir)
	if err != nil {
		return fmt.Errorf("mod folder %s: %v", dir, err)
	}
	root := ""
	if x.archives != nil {
		root = x.archives.Root
	}
	if root != "" && !tree.Packed() {
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
	var src []byte
	var d mod.MissionData
	if _, err := tree.Lstat(mod.MissionsFile); !errors.Is(err, fs.ErrNotExist) {
		if src, _, err = tree.ReadInside(mod.MissionsFile); err != nil {
			return fmt.Errorf("mod %q: %v", e.Manifest.ID, err)
		}
		look, err := mod.TextLookup(dir, lang)
		if err != nil {
			return fmt.Errorf("mod %q: %v", e.Manifest.ID, err)
		}
		if d, err = mod.ParseMissions(e.Manifest.ID, mod.MissionsFile, src, look, lang); err != nil {
			return fmt.Errorf("mod %q: %v", e.Manifest.ID, err)
		}
	}
	owned := false
	if _, err := os.Lstat(filepath.Join(dir, editorPackageRecord)); err == nil && !tree.Packed() {
		if _, err := missionPackageOwnership(dir, editorPackageRecord, nil); err != nil {
			return err
		}
		owned = true
	}
	m := &editorMod{dir: dir, id: e.Manifest.ID, missions: d, owned: owned, packed: tree.Packed(), title: e.Manifest.Title}
	if err := m.readInventory(tree); err != nil {
		return fmt.Errorf("mod %q: %v", e.Manifest.ID, err)
	}
	m.source = bytes.Clone(src)
	m.manifest, _, err = mod.ReadInside(dir, mod.ManifestName)
	if err != nil {
		x.setEditorMod(nil)
		return err
	}
	m.digest, err = mod.ContentDigest(dir)
	if err != nil {
		x.setEditorMod(nil)
		return err
	}
	m.contextIssue = x.openModContext(m, e, root)
	x.setEditorMod(m)
	return nil
}

// readInventory lists the files of the mod's tree and the data kinds whose
// files it holds: what the overview shows.
func (m *editorMod) readInventory(tree mod.Tree) error {
	files, err := tree.Files()
	if err != nil {
		return err
	}
	m.files, m.kinds = files, nil
	for _, k := range mod.DataKinds() {
		for _, name := range k.Files {
			if _, err := tree.Stat(name); err == nil {
				m.kinds = append(m.kinds, k.Name+": "+name)
			}
		}
	}
	return nil
}

// reopenModContext builds the context of the mod the editor works in again
// from its folder, keeping the one in use when the mod no longer opens.
func (x *MapInspector) reopenModContext() error {
	m := x.mod
	root := ""
	if x.archives != nil {
		root = x.archives.Root
	}
	next := *m
	next.context, next.report, next.fileOverrides, next.refusal, next.checkLines = nil, ModDataReport{}, nil, "", nil
	tree, err := mod.OpenTree(m.dir)
	if err != nil {
		return err
	}
	if err := next.readInventory(tree); err != nil {
		return err
	}
	if issue := x.openModContext(&next, mod.Load(m.dir), root); issue != "" {
		return errors.New(issue)
	}
	*m = next
	x.setEditorMod(m)
	return nil
}

// entryFor is the mission of the mod that names the map opened from source:
// the mission whose number an installed scenario map has, or the mission
// whose map path is the opened host file.
func (m *editorMod) entryFor(source string) (mod.Mission, bool) {
	switch src := ui.ParseMapSource(source); src.Kind {
	case ui.SourceNew:
		return m.unwrittenEntry()
	case ui.SourceScenario:
		for _, e := range m.missions.Missions {
			if src.HasNumber && e.Number == src.Number {
				return e, true
			}
		}
		return mod.Mission{}, false
	case ui.SourceLoose:
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

// modReadOnly is the one test at the write boundary of the mod the editor
// works in: every edit W makes and every Save into the mod passes it. A
// packed mod refuses them all; it answers nil for any other mod and none.
func (x *MapInspector) modReadOnly() error {
	if x.mod == nil || !x.mod.packed {
		return nil
	}
	return fmt.Errorf("%s %s", x.word("editor.mod.packed"), x.mod.dir)
}

// ModTarget names the mod Save writes into; empty when none is named.
func (s *mapInspectionEdits) ModTarget() string {
	if s.inspector == nil || s.inspector.mod == nil {
		return ""
	}
	if s.inspector.mod.packed {
		return s.inspector.mod.id + " (" + s.inspector.word("editor.mod.packedtarget") + ")"
	}
	if s.inspector.mod.copyOnly {
		return s.inspector.mod.id + " (read-only; W: copy)"
	}
	return s.inspector.mod.id
}

// SaveIntoMod writes the edited map over the file the mod's mission entry for
// this map names, and refuses when no entry names it.
func (s *mapInspectionEdits) SaveIntoMod() (string, error) {
	defer func() { s.errorsAllowed = false }()
	s.saveNote = ""
	m := s.inspector.mod
	if m == nil {
		return "", errors.New("no mod folder is named")
	}
	if err := s.inspector.modReadOnly(); err != nil {
		return "", err
	}
	if m.copyOnly {
		return "", errors.New("this imported mod is read-only; choose a folder with the same ID under another parent with W to save a complete copy")
	}
	if m.owned {
		return s.saveIntoOwnedMod(m)
	}
	if !bytes.Equal(s.savedWorkspace, s.workspaceImage) {
		return "", errors.New("authored pages need an editor-owned mod package; choose a new folder with W")
	}
	e, ok := m.entryFor(s.source)
	if !ok {
		return "", fmt.Errorf("no mission of mod %q names this map (%s); add its entry to %s", m.id, s.source, mod.MissionsFile)
	}
	if err := s.gate(true); err != nil {
		return "", err
	}
	b, err := s.encodeForSave()
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

// saveIntoOwnedMod saves into an editor-owned package. A map the report holds
// no error for is written as the whole package. A map with errors is asked
// for first; once confirmed, only the map file of the package's entry is
// written, since the loader would refuse the package, and SaveNote says the
// rest was not written.
func (s *mapInspectionEdits) saveIntoOwnedMod(m *editorMod) (string, error) {
	if err := s.gate(true); err != nil {
		return "", err
	}
	if !s.errorsAllowed {
		target, err := s.saveWorkspacePackage(m.dir)
		var refused *ui.ReportRefusal
		if !errors.As(err, &refused) {
			return target, err
		}
		refused.Ask = true
		return "", refused
	}
	e, ok := m.entryFor(s.source)
	if !ok {
		return "", fmt.Errorf("no mission of mod %q names this map (%s); add its entry to %s", m.id, s.source, mod.MissionsFile)
	}
	b, err := s.encodeForSave()
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
	s.saveNote = words.For(s.inspector.pageLanguage()).Text("editor.save.package")
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
		_, err := os.Lstat(existing)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("inspect map output directory: %w", err)
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
	inside, err := pathWithin(folder, physical)
	if err != nil {
		return fmt.Errorf("resolve map output boundary: %w", err)
	}
	if !inside {
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
	if err := extendedName(data, rel); err != nil {
		return "", err
	}
	return writeModFile(root, dir, rel, data)
}

// writeModFile writes data to the slash path rel inside the mod folder dir,
// replacing the file there. The file stays inside the folder and outside the
// install at root.
func writeModFile(root, dir, rel string, data []byte) (string, error) {
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
	inside, err := pathWithin(folder, parent)
	if err != nil {
		return "", fmt.Errorf("resolve map output boundary: %w", err)
	}
	if !inside {
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
	if _, err := hostfile.Commit(hostfile.Set{
		Files: []hostfile.File{{Path: target, Data: data, Overwrite: hostfile.Replace}},
		Dir:   parent,
		Temp:  ".mapedit-*.alm",
		Sync:  true,
	}); err != nil {
		return "", err
	}
	return target, nil
}
