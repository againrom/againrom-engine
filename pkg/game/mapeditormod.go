package game

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"

	"againrom/pkg/mod"
	"againrom/pkg/namefold"
)

type editorModDocument struct {
	Source, Manifest   []byte
	Original, Missions []mod.Mission
	Files              map[string][]byte
	Selected           int
	// TownsSource is the mod's towns file as opened or saved and
	// TownsOriginal its offer records; Towns are the draft's records, which
	// the one row writer writes over the source.
	TownsSource          []byte
	TownsOriginal, Towns []mod.OfferRecord
	// TownRecordsOriginal and TownRecords are the same for the [[town]]
	// records.
	TownRecordsOriginal, TownRecords []mod.TownRecord
	// Loose is the path in the mod of the map the edits hold when no row
	// names it; Selected is then -1.
	Loose string `json:",omitempty"`
}

// current is the mission whose map, pages and labels the edits hold: the
// selected row with the fields of p, or the map no row names as a mission
// of its path alone. row reports the first.
func (d *editorModDocument) current(p editorPackageSpec, word func(string) string) (m mod.Mission, row bool, err error) {
	if d.Selected < 0 && d.Loose != "" {
		return mod.Mission{Map: d.Loose}, false, nil
	}
	m, err = d.selected(p, word)
	return m, true, err
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
	if src, err := modDataFile(m.dir, mod.TownsFile); err == nil && src != nil {
		if records, offers, err := mod.ParseTowns(m.id, mod.TownsFile, src); err == nil {
			d.TownsSource, d.TownsOriginal, d.Towns = src, offers, cloneOffers(offers)
			d.TownRecordsOriginal, d.TownRecords = records, cloneTownRecords(records)
		}
	}
	// The project holds the files of each held kind whole: the folder files
	// among them, the item sprites and the install files with the terrain
	// strips, stay with their bytes, so an undo past a save restores them.
	for _, k := range mod.DraftKinds() {
		if !k.Draft.Held {
			continue
		}
		for _, rel := range k.DraftNames() {
			if src, err := modDataFile(m.dir, rel); err == nil && src != nil {
				d.Files[rel] = src
			}
		}
		for _, rel := range m.files {
			if k.DraftFolderFile(rel) {
				if data, err := os.ReadFile(filepath.Join(m.dir, filepath.FromSlash(rel))); err == nil {
					d.Files[k.DraftFolderName(rel)] = data
				}
			}
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
	n.TownsSource, n.TownsOriginal, n.Towns = bytes.Clone(d.TownsSource), cloneOffers(d.TownsOriginal), cloneOffers(d.Towns)
	n.TownRecordsOriginal, n.TownRecords = cloneTownRecords(d.TownRecordsOriginal), cloneTownRecords(d.TownRecords)
	return &n
}

// data is the draft's records and the ones it was opened with, as the
// writers of missions and towns take them.
func (d *editorModDocument) data() mod.Data {
	return mod.Data{
		Missions: mod.MissionData{Missions: d.Missions},
		Towns:    mod.TownData{Records: d.TownRecords, Offers: d.Towns},
		Before:   mod.DataBefore{Missions: d.Original, Offers: d.TownsOriginal, Towns: d.TownRecordsOriginal},
	}
}

// townsDraft is the towns file the draft's offer records make of the
// source, and whether it differs from the source.
func (d *editorModDocument) townsDraft() ([]byte, bool, error) {
	src, err := writeHeld(mod.TownsFile, d.TownsSource, d.data())
	return src, err == nil && !bytes.Equal(src, d.TownsSource), err
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
			if _, ok := d.Files[name]; !ok && importedModFile(name) {
				d.Files[name] = nil
			}
		}
	}
	finish := func(m mod.Mission, layout mod.LayoutChange) (map[string][]byte, []mod.Mission, mod.Mission, mod.LayoutChange, error) {
		data, err := writeHeld(mod.MissionsFile, d.Source, d.data())
		if err != nil {
			return nil, nil, m, none, err
		}
		// A mod with no missions file and no row to write keeps none.
		if len(data) != 0 || len(d.Source) != 0 {
			d.Files[mod.MissionsFile] = data
		}
		towns, changed, err := d.townsDraft()
		if err != nil {
			return nil, nil, m, none, err
		}
		if changed {
			d.Files[mod.TownsFile] = towns
		}
		d.Files[mod.ManifestName], err = s.workspace.Package.manifestBytes(d.Manifest)
		if err != nil {
			return nil, nil, m, none, err
		}
		return d.Files, d.Missions, m, layout, nil
	}
	if d.Selected < 0 && d.Loose != "" && s.model != nil {
		// A map no row names is written at its own path.
		if err := extendedName(b, d.Loose); err != nil {
			return nil, nil, mod.Mission{}, none, err
		}
		d.Files[d.Loose] = bytes.Clone(b)
		return finish(mod.Mission{}, none)
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
				return c, wordErr("editor.err.projectnofile", mv.From)
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

// draftView is the open project over the mod's files: the names it holds and
// the drafts as written, where a nil draft is a deletion, folded as the mod
// tree matches names.
type draftView struct {
	names  []string
	drafts map[string][]byte
	folded map[string]string
}

// Draft is the draft of the file rel names, matched as the mod tree matches
// names, and whether the project drafts it; nil bytes take the file out.
func (v draftView) Draft(rel string) ([]byte, bool) {
	name, ok := v.folded[namefold.ModTree(rel)]
	return v.drafts[name], ok
}

// Drafted is every drafted name, sorted, one spelling per folded name.
func (v draftView) Drafted() []string { return slices.Sorted(maps.Values(v.folded)) }

// projectView applies drafts over base, the mod's files. A nil draft deletes
// every base file whose namefold.ModTree equals its name's; the deletions come
// first, then each other draft is added under its own spelling, replacing a
// base file that folds equal to it. The folded drafts follow the same order:
// of names that fold equal, the last added is the one the view reads.
func projectView(base []string, drafts map[string][]byte) draftView {
	held := slices.Clone(base)
	names := slices.Sorted(maps.Keys(drafts))
	folded := map[string]string{}
	for _, name := range names {
		if drafts[name] == nil {
			held = slices.DeleteFunc(held, func(f string) bool { return namefold.ModTree(f) == namefold.ModTree(name) })
			folded[namefold.ModTree(name)] = name
		}
	}
	for _, name := range names {
		if drafts[name] != nil {
			held = slices.DeleteFunc(held, func(f string) bool { return f != name && namefold.ModTree(f) == namefold.ModTree(name) })
			if !slices.Contains(held, name) {
				held = append(held, name)
			}
			folded[namefold.ModTree(name)] = name
		}
	}
	return draftView{names: held, drafts: drafts, folded: folded}
}

// projectFiles is the files of the open project: the mod's files with the
// drafts over them.
func (s *mapInspectionEdits) projectFiles(d *editorModDocument) []string {
	var base []string
	if m := s.inspector.mod; m != nil {
		base = m.files
	}
	return projectView(base, d.Files).names
}

func (x *MapInspector) rowFiles(m *mod.Mission, view draftView) error {
	if view.drafts == nil || m.Map == "" {
		return m.ReadFiles(x.mod.dir)
	}
	b, drafted := view.Draft(m.Map)
	missing := fmt.Errorf("%s: %s", m.Key, x.word("editor.mod.err.mapmissing"))
	if drafted && len(b) == 0 {
		return missing
	}
	err := m.ReadFiles(x.mod.dir)
	if _, _, e := mod.MissionMap.ReadIn(x.mod.dir, m.Map); e != nil {
		if !drafted {
			return missing
		}
		err = nil
	}
	if drafted {
		m.MapBytes = bytes.Clone(b)
	}
	return err
}

func (x *MapInspector) missionFiles(rows []mod.Mission, view draftView) error {
	for i := range rows {
		m := &rows[i]
		if m.Off || m.Map == "" {
			continue
		}
		if view.drafts != nil {
			if err := x.rowFiles(m, view); err != nil {
				return err
			}
		} else if len(m.MapBytes) == 0 {
			if err := m.ReadFiles(x.mod.dir); err != nil {
				return err
			}
		}
	}
	return nil
}

func (x *MapInspector) checkMissionSet(missions []mod.Mission, view draftView) error {
	if x.mod != nil && x.mod.loaded != nil && slices.ContainsFunc(missions, func(m mod.Mission) bool { return m.Kind == mod.MissionCampaign || m.Campaign() && m.GuestParty() }) {
		rows := make([]mod.Mission, len(missions))
		for i, m := range missions {
			rows[i] = mod.CloneMission(m)
		}
		if err := x.missionFiles(rows, view); err != nil {
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
			if e := x.rowFiles(&m, view); e != nil {
				return e
			}
		}
		if m.Kind == mod.MissionReplace {
			err = x.checkMissionRoute(m)
		} else {
			err = checkGuest(c, x.archives.Containers, d, m)
		}
		if err != nil {
			return wordErr("editor.err.pathwhy", m.Key, err)
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
