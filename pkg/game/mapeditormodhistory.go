package game

import (
	"bytes"
	"encoding/json"
	"maps"
	"os"
	"path"
	"path/filepath"
	"strings"

	"againrom/pkg/mod"
)

func modPackageSpec(m mod.Mission, previous editorPackageSpec) editorPackageSpec {
	return editorPackageSpec{ID: previous.ID, Title: previous.Title, Manifest: mod.CloneManifest(previous.Manifest), Kind: m.Kind, Number: m.Number, After: m.After, Guest: m.Guest, Chapter: m.Chapter, Gold: m.Gold, Party: mod.CloneMission(m).Party, Class: m.Class, Sex: m.Sex}
}

func sameWorkspace(a, b []byte) bool {
	if bytes.Equal(a, b) {
		return true
	}
	var left, right editorWorkspaceState
	decodeWorkspace(a, &left)
	decodeWorkspace(b, &right)
	if left.Project == nil || right.Project == nil {
		return false
	}
	for _, w := range []*editorWorkspaceState{&left, &right} {
		w.Pages = nil
		w.Package = editorPackageSpec{ID: w.Package.ID, Title: w.Package.Title, Manifest: w.Package.Manifest}
		w.Project.Selected = 0
	}
	x, _ := json.Marshal(left)
	y, _ := json.Marshal(right)
	return bytes.Equal(x, y)
}

// syncModDocument writes the selected row's edits into the project's
// drafts: the map when mapChanged, each page that differs from oldPages,
// and the labels file beside the map when the labels differ from oldLabels.
func (s *mapInspectionEdits) syncModDocument(w *editorWorkspaceState, oldPages map[string][]byte, oldLabels map[uint32]string, mapChanged bool) {
	d := w.Project
	if d == nil || d.Selected < 0 || d.Selected >= len(d.Missions) {
		return
	}
	m, err := d.selected(w.Package, s.inspector.word)
	if err != nil || m.Off {
		return
	}
	if mapChanged {
		d.Files[m.Map] = s.model.Bytes()
	}
	if !maps.Equal(w.Labels, oldLabels) {
		var labels map[uint32]string
		if _, sc, err := wizardScript(s.model); err == nil {
			labels = liveLabels(sc, w.Labels)
		}
		d.Files[mod.MapLabelFile(m.Map)] = encodeMapLabels(labels)
	}
	for name, b := range w.Pages {
		if bytes.Equal(b, oldPages[name]) {
			continue
		}
		folder := strings.ReplaceAll(m.Texts, "{lang}", s.inspector.pageLanguage())
		sibling := path.Join(mapTextFolder(m.Map), s.inspector.pageLanguage())
		if folder == "" {
			folder = sibling
		}
		for _, root := range []string{sibling, mapTextFolder(m.Map)} {
			rel := path.Join(root, name)
			if _, pending := d.Files[rel]; pending {
				folder = sibling
				break
			}
			if s.inspector.mod != nil {
				if info, err := os.Stat(filepath.Join(s.inspector.mod.dir, filepath.FromSlash(rel))); err == nil && !info.IsDir() {
					folder = sibling
					break
				}
			}
		}
		d.Files[path.Join(folder, name)] = bytes.Clone(b)
	}
	d.Missions[d.Selected] = m
}

func (s *mapInspectionEdits) refreshModSelection() {
	if s.inspector.mod == nil || s.workspace.Project == nil {
		return
	}
	d := s.workspace.Project
	if d.Selected < 0 || d.Selected >= len(d.Missions) {
		return
	}
	m := d.Missions[d.Selected]
	s.inspector.mod.selected = m.RowID()
	if b, err := os.ReadFile(filepath.Join(s.inspector.mod.dir, filepath.FromSlash(m.Map))); err == nil {
		s.saved = b
	}
}
