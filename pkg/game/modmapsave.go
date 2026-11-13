package game

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"againrom/pkg/locale"
	"againrom/pkg/mod"
	"againrom/pkg/namefold"
	"againrom/pkg/ui"
)

// editorMod is the mod folder the map editor's Save writes into, with the
// missions its data/missions.toml declares in the install's language, and
// the context its admitted fork gives the editor: the table and tileset, the
// character catalog, and what applying the mod's data reported. Its files,
// the kinds whose files it holds and its check report make its overview.
type editorMod struct {
	source, manifest []byte
	digest           string
	snapshot         map[string][32]byte
	ownershipRecord  []byte
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
	// flow is the campaign and the missions of the mod's admitted fork, for
	// W's graph; nil when no fork was built.
	flow *campaignFlow
}

// SetMod is the editor's one open route: it opens the mod at dir, a folder or
// a packed file holding any kinds, runs it and makes it the mod the editor
// works in. Save writes the edited map to the path the mod's mission entry
// for that map names. A folder must lie outside the install; a packed mod
// opens read-only.
func (x *MapInspector) SetMod(dir string) error {
	e := mod.Load(dir)
	if e.Err != nil {
		return fmt.Errorf(x.word("editor.mod.err.folder"), dir, e.Err)
	}
	tree, err := mod.OpenTree(dir)
	if err != nil {
		return fmt.Errorf(x.word("editor.mod.err.folder"), dir, err)
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
			return fmt.Errorf(x.word("editor.mod.err.ininstall"), dir, root)
		}
	}
	lang := locale.Fallback
	if root != "" {
		if l, ok := locale.ByBaseSuffix(BaseID(InspectInstall(root))); ok {
			lang = l.Code
		}
	}
	d, src, err := mod.DescribeMissions(dir, e.Manifest.ID, lang)
	if err != nil {
		return fmt.Errorf("%s: %v", fmt.Sprintf(x.word("editor.mod.err.mod"), e.Manifest.ID), err)
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
		return fmt.Errorf("%s: %v", fmt.Sprintf(x.word("editor.mod.err.mod"), e.Manifest.ID), err)
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
	m.snapshot = map[string][32]byte{}
	for _, name := range m.files {
		data, err := tree.ReadFile(name)
		if err != nil {
			return err
		}
		m.snapshot[namefold.ModTree(name)] = sha256.Sum256(data)
	}
	if owned {
		m.ownershipRecord, err = os.ReadFile(filepath.Join(dir, editorPackageRecord))
		if err != nil {
			return err
		}
		m.snapshot[editorPackageRecord] = sha256.Sum256(m.ownershipRecord)
	}
	m.contextIssue = x.openModContext(m, e, root)
	x.setEditorMod(m)
	return nil
}

// readInventory lists the files of the mod's tree and the data kinds whose
// files it holds: what the overview shows.
func (m *editorMod) readInventory(tree mod.Tree, drafts ...map[string][]byte) error {
	files, err := tree.Files()
	if err != nil {
		return err
	}
	if len(drafts) > 0 {
		for name, data := range drafts[0] {
			if data == nil {
				files = slices.DeleteFunc(files, func(path string) bool { return path == name })
				continue
			}
			if !slices.Contains(files, name) {
				files = append(files, name)
			}
		}
		slices.Sort(files)
	}
	m.files, m.kinds = files, nil
	for _, k := range mod.DataKinds() {
		for _, name := range k.Files {
			if slices.Contains(files, name) {
				m.kinds = append(m.kinds, k.Name+": "+name)
			}
		}
	}
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
			if e.RowID() == m.selected {
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

// SaveIntoMod saves the current project through the staged package writer.
func (s *mapInspectionEdits) SaveIntoMod() (string, error) {
	defer func() { s.errorsAllowed = false }()
	s.saveNote = ""
	x, m := s.inspector, s.inspector.mod
	if m == nil {
		return "", errors.New(x.word("editor.mod.err.nofoldername"))
	}
	if err := x.modReadOnly(); err != nil {
		return "", err
	}
	if m.copyOnly {
		return "", errors.New(x.word("editor.mod.err.imported"))
	}
	if s.model == nil {
		return s.saveWorkspacePackage(m.dir)
	}
	if err := s.prepareWorkspace(); err != nil {
		return "", err
	}
	if s.workspace.Project != nil && s.workspace.Project.Selected < 0 && bytes.Equal(s.saved, s.model.Bytes()) {
		return s.saveWorkspacePackage(m.dir)
	}
	fullPackage := !s.errorsAllowed && x.archives != nil && x.archives.Containers != nil
	if m.owned {
		if err := s.gate(true); err != nil {
			return "", err
		}
		if fullPackage {
			target, err := s.saveWorkspacePackage(m.dir)
			var refused *ui.ReportRefusal
			if errors.As(err, &refused) {
				refused.Ask = true
				return "", refused
			}
			return target, err
		}
	}
	e, ok := m.entryFor(s.source)
	if d := s.workspace.Project; d != nil && d.Selected >= 0 {
		var err error
		e, err = d.selected(s.workspace.Package, x.word)
		if err != nil {
			return "", err
		}
		ok = !e.Off
	}
	if !ok {
		return "", fmt.Errorf(x.word("editor.mod.err.noentry"), m.id, s.source, mod.MissionsFile)
	}
	if !m.owned {
		if err := s.gate(true); err != nil {
			return "", err
		}
	}
	if fullPackage {
		return s.saveWorkspacePackage(m.dir)
	}
	// A confirmed map with report errors retains the other pending edits.
	b, err := s.encodeForSave()
	if err != nil {
		return "", fmt.Errorf("%s: %w", fmt.Sprintf(x.word("editor.mod.err.mod"), m.id), err)
	}
	if len(b) > mod.MaxFileBytes {
		return "", fmt.Errorf(x.word("editor.mod.err.mapsize"), len(b), mod.MaxFileBytes)
	}
	if err := extendedName(b, e.Map); err != nil {
		return "", err
	}
	old := []string(nil)
	if m.owned {
		old, err = missionPackageOwnership(m.dir, editorPackageRecord, nil)
		if err != nil {
			return "", err
		}
	}
	plan := missionPackagePlan{Source: m.dir}
	if err := x.writeEditorProject(m.dir, m.id, old, map[string][]byte{e.Map: b}, plan, true); err != nil {
		return "", fmt.Errorf("%s: %w", fmt.Sprintf(x.word("editor.mod.err.mod"), m.id), err)
	}
	if err := x.SetMod(m.dir); err != nil {
		return "", err
	}
	s.joinMod()
	s.saved, s.source, s.savedInside = s.model.Bytes(), filepath.Join(m.dir, filepath.FromSlash(e.Map)), s.TextsInside()
	if s.errorsAllowed {
		s.saveNote = x.word("editor.save.package")
	}
	return s.source, nil
}
