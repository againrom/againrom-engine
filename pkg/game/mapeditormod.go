package game

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/namefold"
)

type editorModDocument struct {
	Source, Manifest   []byte
	Original, Missions []mod.Mission
	Files              map[string][]byte
	Selected           int
}

func (m *editorMod) document() *editorModDocument {
	d := &editorModDocument{Source: bytes.Clone(m.source), Manifest: bytes.Clone(m.manifest), Files: map[string][]byte{}}
	for i, mission := range m.missions.Missions {
		d.Original = append(d.Original, mod.CloneMission(mission))
		d.Missions = append(d.Missions, mod.CloneMission(mission))
		if mission.RowID() == m.selected {
			d.Selected = i
		}
	}
	if src, err := m.speakersFile(); err == nil {
		if records, err := mod.ReadSpeakers(m.id, m.dir, mod.SpeakersFile, src, nil); err == nil {
			d.Files[mod.SpeakersFile] = bytes.Clone(src)
			for _, r := range records.Speakers {
				if r.Picture != "" {
					d.Files[r.Picture] = bytes.Clone(r.PictureBytes)
				}
			}
		}
	}
	return d
}

func (d *editorModDocument) clone() *editorModDocument {
	if d == nil {
		return nil
	}
	n := *d
	n.Source, n.Manifest = bytes.Clone(d.Source), bytes.Clone(d.Manifest)
	n.Original, n.Missions = nil, nil
	for _, m := range d.Original {
		n.Original = append(n.Original, mod.CloneMission(m))
	}
	for _, m := range d.Missions {
		n.Missions = append(n.Missions, mod.CloneMission(m))
	}
	n.Files = map[string][]byte{}
	for k, v := range d.Files {
		n.Files[k] = bytes.Clone(v)
	}
	return &n
}

func (d *editorModDocument) selected(p editorPackageSpec, word func(string) string) (mod.Mission, error) {
	if d.Selected < 0 || d.Selected >= len(d.Missions) {
		return mod.Mission{}, errors.New(word("editor.mod.err.nomission"))
	}
	m := mod.CloneMission(d.Missions[d.Selected])
	// A mission its map file declares takes its route from the file's name.
	if m.Off || m.ByFileName() {
		return m, nil
	}
	m.Kind, m.Number = p.Kind, p.Number
	m.After, m.Guest, m.Chapter, m.Gold = p.After, p.Guest, p.Chapter, p.Gold
	m.Class, m.Sex = p.Class, p.Sex
	m.AutoGetMission, m.MainParty = p.AutoGetMission, p.MainParty
	m.Party = mod.CloneMission(mod.Mission{Party: p.Party}).Party
	if m.Campaign() {
		m.After = 0
	}
	if len(m.Party) != 0 {
		m.Guest, m.Chapter, m.Carry = 0, 0, ""
	}
	return m, nil
}

// fileMissionFixed refuses a change of the selected mission's route or place
// in the list when its map file declares it: the file's name is the route.
func (s *mapInspectionEdits) fileMissionFixed(w *editorWorkspaceState) error {
	if d := w.Project; d != nil && d.Selected >= 0 && d.Selected < len(d.Missions) && d.Missions[d.Selected].ByFileName() {
		return fmt.Errorf(s.inspector.word("editor.mod.err.filemission"), d.Missions[d.Selected].Map)
	}
	return nil
}

// retainedPackageFiles is the project's files and rows as the save of map b
// writes them, the selected row, and the move into the tree that save makes.
func (s *mapInspectionEdits) retainedPackageFiles(b []byte) (map[string][]byte, []mod.Mission, mod.Mission, mod.LayoutChange, error) {
	var none mod.LayoutChange
	d := s.workspace.Project.clone()
	var saved editorWorkspaceState
	decodeWorkspace(s.savedWorkspace, &saved)
	if saved.Project != nil {
		if src, ok := saved.Project.Files[mod.SpeakersFile]; ok {
			names := []string{mod.SpeakersFile}
			if data, err := mod.ParseSpeakers(saved.Package.ID, mod.SpeakersFile, src); err == nil {
				for _, row := range data.Speakers {
					if row.Picture != "" {
						names = append(names, row.Picture)
					}
				}
			}
			for _, name := range names {
				if _, ok := d.Files[name]; !ok {
					d.Files[name] = nil
				}
			}
		}
		// An imported file the last save wrote and an undo since took out
		// of the project leaves the folder with this save.
		for name := range saved.Project.Files {
			if _, ok := d.Files[name]; !ok && strings.HasPrefix(name, mod.AssetsFolder+"/") {
				d.Files[name] = nil
			}
		}
	}
	finish := func(m mod.Mission, layout mod.LayoutChange) (map[string][]byte, []mod.Mission, mod.Mission, mod.LayoutChange, error) {
		data, err := mod.WriteMissions(d.Source, d.Original, d.Missions)
		if err != nil {
			return nil, nil, m, none, err
		}
		// A mod with no missions file and no row to write keeps none.
		if len(data) != 0 || len(d.Source) != 0 {
			d.Files[mod.MissionsFile] = data
		}
		d.Files[mod.ManifestName], err = s.workspace.Package.manifestBytes(d.Manifest)
		if err != nil {
			return nil, nil, m, none, err
		}
		return d.Files, d.Missions, m, layout, nil
	}
	if d.Selected < 0 {
		if s.model != nil && (!bytes.Equal(s.saved, s.model.Bytes()) || !reflect.DeepEqual(s.workspace.Pages, saved.Pages) || !reflect.DeepEqual(s.workspace.Labels, saved.Labels)) {
			return nil, nil, mod.Mission{}, none, errors.New(s.inspector.word("editor.mod.err.unnamed"))
		}
		return finish(mod.Mission{}, none)
	}
	m, err := d.selected(s.workspace.Package, s.inspector.word)
	if err != nil {
		return nil, nil, m, none, err
	}
	if m.Off {
		return nil, nil, m, none, errors.New(s.inspector.word("editor.mod.err.disabledmap"))
	}
	if m.Map == "" {
		if s.model == nil || bytes.Equal(b, s.saved) {
			d.Missions[d.Selected] = m
			return finish(m, none)
		}
		m.Map = s.place().Free(s.projectFiles(d)).Map
	}
	if m.ByFileName() {
		// The layout does not rename a map file that is its own declaration.
		if err := extendedName(b, m.Map); err != nil {
			return nil, nil, m, none, err
		}
	}
	d.Files[m.Map] = bytes.Clone(b)
	d.Missions[d.Selected] = m
	extended := map[string]bool{m.Key: s.model != nil && s.model.Extended()}
	layout, err := s.treeLayout(d, extended)
	if err != nil {
		return nil, nil, m, none, err
	}
	m = d.Missions[d.Selected]
	m.MapBytes = bytes.Clone(b)
	m.TextFiles = s.workspace.Pages
	d.Missions[d.Selected] = m
	return finish(m, layout)
}

// treeLayout moves the project's mission rows that lie outside the tree
// into it, and the map of a row extended names to .almx (mod.TreeLayout):
// each moved file becomes a draft at its new path, read from its draft or
// the mod folder, and each old file no row reads a removal. A speaker
// record bound to a moved map names the map's new path.
func (s *mapInspectionEdits) treeLayout(d *editorModDocument, extended map[string]bool) (mod.LayoutChange, error) {
	c := mod.TreeLayout(d.Missions, s.projectFiles(d), extended)
	data := map[string][]byte{}
	for _, mv := range c.Moves {
		b := d.Files[mv.From]
		if b == nil {
			if s.inspector.mod == nil {
				return c, fmt.Errorf("%s: no such file in the project", mv.From)
			}
			var err error
			if b, _, err = mod.ReadInside(s.inspector.mod.dir, mv.From); err != nil {
				return c, err
			}
		}
		data[mv.To] = b
	}
	for _, name := range c.Removed {
		d.Files[name] = nil
	}
	maps.Copy(d.Files, data)
	if len(c.Moves) != 0 && s.inspector.mod != nil {
		src, drafted := d.Files[mod.SpeakersFile]
		var err error
		if !drafted {
			if src, err = s.inspector.mod.speakersFile(); err != nil {
				return c, err
			}
		}
		if err := moveSpeakerDraft(d.Files, src, c.Moves); err != nil {
			return c, err
		}
	}
	for i, m := range c.Missions {
		if b, moved := data[m.Map]; moved && m.Map != d.Missions[i].Map {
			m.MapBytes = bytes.Clone(b)
		}
		d.Missions[i] = m
	}
	return c, nil
}

// moveSpeakerDraft drafts the speakers file src with its records bound to a
// moved map naming the map's new path, when one is.
func moveSpeakerDraft(files map[string][]byte, src []byte, moves []mod.FileMove) error {
	if src == nil {
		return nil
	}
	b, moved, err := mod.MoveSpeakerMaps(src, moves)
	if moved {
		files[mod.SpeakersFile] = b
	}
	return err
}

// rebaseHistory applies the move c that a save of the mod at from into to
// made to the project states undo and redo hold, so each names the files the
// save left: rows declaring a moved map take its new place (Rebase), the
// table source and its rows are the saved ones, drafts and the speakers draft
// follow the moves, and an opened map that moved opens at its new path. A
// save that moved nothing leaves the history as it is. A speakers draft that
// does not read stays as it is; the next save's admission names it.
func (s *mapInspectionEdits) rebaseHistory(c mod.LayoutChange, from, to string) {
	saved := s.workspace.Project
	if len(c.Moves) == 0 || saved == nil {
		return
	}
	for _, history := range [][]editorWorkspaceSnapshot{s.undoWorkspace, s.redoWorkspace} {
		for i := range history {
			h := &history[i]
			if h.workspace == nil || h.workspace.Project == nil {
				continue
			}
			w := cloneWorkspace(h.workspace)
			d := w.Project
			d.Missions, d.Source, d.Original = c.Rebase(d.Missions), bytes.Clone(saved.Source), saved.clone().Original
			for _, mv := range c.Moves {
				if b, ok := d.Files[mv.From]; ok {
					delete(d.Files, mv.From)
					d.Files[mv.To] = b
				}
			}
			_ = moveSpeakerDraft(d.Files, d.Files[mod.SpeakersFile], c.Moves)
			h.workspace = w
			if rel, err := filepath.Rel(from, h.source); err == nil {
				for _, mv := range c.Moves {
					if namefold.ModTree(filepath.ToSlash(rel)) == namefold.ModTree(mv.From) {
						h.source = filepath.Join(to, filepath.FromSlash(mv.To))
					}
				}
			}
		}
	}
}

// projectFiles is the files of the open project: the mod's files with the
// drafts over them.
func (s *mapInspectionEdits) projectFiles(d *editorModDocument) []string {
	var held []string
	if m := s.inspector.mod; m != nil {
		held = slices.Clone(m.files)
	}
	for _, name := range slices.Sorted(maps.Keys(d.Files)) {
		if d.Files[name] == nil {
			held = slices.DeleteFunc(held, func(f string) bool { return namefold.ModTree(f) == namefold.ModTree(name) })
		} else if !slices.Contains(held, name) {
			held = append(held, name)
		}
	}
	return held
}

func (x *MapInspector) missionFiles(rows []mod.Mission, pending map[string][]byte) error {
	for i := range rows {
		m := &rows[i]
		if m.Off || m.Map == "" {
			continue
		}
		if len(m.MapBytes) == 0 {
			if err := m.ReadFiles(x.mod.dir); err != nil {
				return err
			}
		}
		if b, ok := pending[m.Map]; ok {
			m.MapBytes = bytes.Clone(b)
		}
	}
	return nil
}

func (x *MapInspector) checkMissionSet(missions []mod.Mission) error {
	if x.mod != nil && x.mod.loaded != nil && slices.ContainsFunc(missions, func(m mod.Mission) bool { return m.Kind == mod.MissionCampaign || m.Campaign() && m.GuestParty() }) {
		rows := make([]mod.Mission, len(missions))
		for i, m := range missions {
			rows[i] = mod.CloneMission(m)
		}
		if err := x.missionFiles(rows, nil); err != nil {
			return err
		}
		result := *x.mod.loaded
		result.Missions = mod.MissionData{Missions: rows}
		_, _, err := x.admittedFork(result)
		return err
	}
	if len(missions) == 1 {
		return x.checkMissionRoute(missions[0])
	}
	c, err := LoadCampaign(x.archives.Containers)
	if err != nil {
		return err
	}
	d, _, err := mod.MergeMissions(mod.MissionData{Missions: missions})
	if err != nil {
		return err
	}
	for _, m := range d.Missions {
		if len(m.MapBytes) == 0 && x.mod != nil {
			if e := m.ReadFiles(x.mod.dir); e != nil {
				return e
			}
		}
		if m.Kind == mod.MissionReplace {
			err = x.checkMissionRoute(m)
		} else {
			err = checkGuest(c, x.archives.Containers, d, m)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", m.Key, err)
		}
	}
	return nil
}

func (s *mapInspectionEdits) retainedSource(folder string) string {
	if s.workspace.Project != nil && s.inspector.mod != nil {
		return s.inspector.mod.dir
	}
	return folder
}

func (s *mapInspectionEdits) unchangedModSource() error {
	if m := s.inspector.mod; m != nil && s.workspace.Project != nil {
		stamp, err := mod.ContentDigest(m.dir)
		if err != nil {
			return err
		}
		if stamp != m.digest {
			return errors.New(s.inspector.word("editor.mod.err.changed"))
		}
	}
	return nil
}
