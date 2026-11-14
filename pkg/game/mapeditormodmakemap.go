package game

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/namefold"
	"againrom/pkg/ui"
)

// guestRow is the default guest row of mod id among the rows of d: the
// campaign's first guest slot, the lowest number after it that the install
// admits and no row of d takes, the party of its companion and a place after
// the last guest row of d. Add and Make mission both build their guest from
// it, and name the map themselves.
func (s *mapInspectionEdits) guestRow(d *editorModDocument, id string) (mod.Mission, error) {
	c, err := LoadCampaign(s.inspector.archives.Containers)
	if err != nil {
		return mod.Mission{}, err
	}
	m, err := testPlayGuest(c, s.inspector.archives.Containers)
	if err != nil {
		return mod.Mission{}, err
	}
	for ; m.Number <= 65535; m.Number++ {
		used := slices.ContainsFunc(d.Missions, func(v mod.Mission) bool { return v.Number == m.Number })
		if !used && checkGuest(c, s.inspector.archives.Containers, mod.MissionData{Missions: []mod.Mission{m}}, m) == nil {
			break
		}
	}
	if m.Number > 65535 {
		return mod.Mission{}, errors.New(s.inspector.word("editor.mod.err.nonumber"))
	}
	m.Mod = id
	m.Party = []mod.GuestMember{{Character: mod.InstallCharacterKey(int32(m.Guest))}}
	m.Guest, m.Chapter, m.Carry = 0, 0, ""
	for i := len(d.Missions) - 1; i >= 0; i-- {
		if d.Missions[i].Kind == mod.MissionGuest && !d.Missions[i].Off {
			m.After = d.Missions[i].Number
			break
		}
	}
	return m, nil
}

// draftOpenMap puts the open map, its pages and its variable labels into the
// drafts of d at the free place of its stem, and answers that place.
func (s *mapInspectionEdits) draftOpenMap(d *editorModDocument) (mod.MissionPlace, error) {
	place := s.place().Free(s.projectFiles(d))
	pages, err := s.exportPageTree("", nil)
	if err != nil {
		return place, err
	}
	for name, b := range pages {
		d.Files[place.Pages("")+"/"+name] = bytes.Clone(b)
	}
	if labels := s.labelsForSave(); len(labels) > 0 {
		d.Files[place.Labels()] = labels
	}
	d.Files[place.Map] = s.model.Bytes()
	return place, nil
}

// mapList is the maps of project d: the one list over its files and rows.
func (s *mapInspectionEdits) mapList(d *editorModDocument) ([]mod.MapEntry, mod.MissionData) {
	data, _ := s.inspector.modMissions(s.inspector.mod.description(), d.Missions)
	return mod.MapList(s.projectFiles(d), "", data), data
}

// replaceNumbers are the campaign's mission numbers the install holds and no
// row of data replaces, lowest first.
func (x *MapInspector) replaceNumbers(data mod.MissionData) []int {
	flow, held := x.installFlow(), x.missionAnswer()
	if flow == nil || held == nil {
		return nil
	}
	c := flow.campaign
	var out []int
	for _, n := range slices.Concat(c.Main, c.Side, c.Offered) {
		if _, taken := data.Replacing(n); !taken && held(n) == nil && !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	slices.Sort(out)
	return out
}

// ModMapActions are the entries of the mod's maps that a mission can be made
// of: a map no row names, with the guest row and the first replacement the
// install allows, and a map that replaces a mission by its file name, with
// the row that names it.
func (s *mapInspectionEdits) ModMapActions() ([]ui.ModMapAction, bool) {
	d := s.project0()
	if d == nil {
		return nil, false
	}
	x := s.inspector
	list, data := s.mapList(d)
	var out []ui.ModMapAction
	for _, e := range list {
		name := strings.TrimPrefix(e.Path, mod.MapsFolder+"/")
		a := ui.ModMapAction{Label: name + ": " + strings.Join(x.mapRoles(e), ", ")}
		switch e.Role() {
		case mod.MapUnused:
			a.Buttons = []ui.ModMapButton{{Caption: x.word("editor.mod.makemission"), Key: "makemission", Value: e.Path}}
			if numbers := x.replaceNumbers(data); len(numbers) > 0 {
				a.Buttons = append(a.Buttons, ui.ModMapButton{Caption: fmt.Sprintf(x.word("editor.mod.makereplace"), numbers[0]), Key: "makemission", Value: strconv.Itoa(numbers[0]) + "|" + e.Path})
			}
		case mod.MapByName:
			a.Buttons = []ui.ModMapButton{{Caption: x.word("editor.mod.makerow"), Key: "makerow", Value: e.Path}}
		default:
			continue
		}
		out = append(out, a)
	}
	return out, true
}

// mapOf is the entry of list at path rel.
func (s *mapInspectionEdits) mapOf(list []mod.MapEntry, rel string) (mod.MapEntry, error) {
	if i := mapEntryOf(list, rel); i >= 0 {
		return list[i], nil
	}
	return mod.MapEntry{}, fmt.Errorf(s.inspector.word("editor.mod.err.mapunknown"), rel)
}

// makeMission makes the unused map of value a mission row of w's project:
// value is its path, or "N|path" for a replacement of mission N. A map the
// open edits hold as a loose map becomes the selected row.
func (s *mapInspectionEdits) makeMission(w *editorWorkspaceState, value string) error {
	x, d := s.inspector, w.Project
	if d == nil {
		return errors.New(x.word("editor.mod.err.nomission"))
	}
	replace := 0
	if head, tail, ok := strings.Cut(value, "|"); ok {
		n, err := strconv.Atoi(head)
		if err != nil {
			return fmt.Errorf(x.word("editor.mod.err.mapunknown"), value)
		}
		replace, value = n, tail
	}
	list, data := s.mapList(d)
	e, err := s.mapOf(list, value)
	if err != nil {
		return err
	}
	if e.Role() != mod.MapUnused {
		return fmt.Errorf(x.word("editor.mod.err.maptaken"), e.Path)
	}
	held := x.missionAnswer()
	var guest mod.Mission
	if replace != 0 {
		if held == nil || held(replace) != nil {
			return fmt.Errorf(x.word("editor.mod.err.replacenumber"), replace)
		}
		if _, taken := data.Replacing(replace); taken {
			return fmt.Errorf(x.word("editor.mod.err.replacetaken"), replace)
		}
	} else if guest, err = s.guestRow(d, w.Package.ID); err != nil {
		return err
	}
	row, err := mod.MakeMission(w.Package.ID, data, e, guest, replace, held)
	if err != nil {
		return err
	}
	d.Missions = append(d.Missions, row)
	if d.Selected < 0 && namefold.ModTree(d.Loose) == namefold.ModTree(row.Map) {
		d.Selected, d.Loose = len(d.Missions)-1, ""
		w.Package = modPackageSpec(row, w.Package)
	}
	return nil
}

// makeRow writes the row of the map of value, a map that replaces its mission
// by its file name, in place of the file form's mission.
func (s *mapInspectionEdits) makeRow(w *editorWorkspaceState, value string) error {
	x, d := s.inspector, w.Project
	if d == nil {
		return errors.New(x.word("editor.mod.err.nomission"))
	}
	list, data := s.mapList(d)
	e, err := s.mapOf(list, value)
	if err != nil {
		return err
	}
	if e.Role() != mod.MapByName {
		return fmt.Errorf(x.word("editor.mod.err.notbyname"), e.Path)
	}
	held := x.missionAnswer()
	if held == nil || held(e.Number) != nil {
		return fmt.Errorf(x.word("editor.mod.err.replacenumber"), e.Number)
	}
	row, err := mod.MakeRow(w.Package.ID, data, e, held)
	if err != nil {
		return err
	}
	at := slices.IndexFunc(d.Missions, func(m mod.Mission) bool {
		return m.ByFileName() && namefold.ModTree(m.Map) == namefold.ModTree(e.Path)
	})
	if at < 0 {
		d.Missions = append(d.Missions, row)
		return nil
	}
	d.Missions[at] = row
	if at == d.Selected {
		w.Package = modPackageSpec(row, w.Package)
	}
	return nil
}

// addOpenMap copies the open map into the mod under its own stem as a draft
// no row names.
func (s *mapInspectionEdits) addOpenMap(w *editorWorkspaceState) error {
	if w.Project == nil || s.model == nil {
		return errors.New(s.inspector.word("editor.mod.err.nomaptoadd"))
	}
	_, err := s.draftOpenMap(w.Project)
	return err
}

// folderNames are the files of host folder that can belong to a map: those at
// its top and those inside a folder of text pages, as slash paths.
func folderNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		switch {
		case e.Type().IsRegular():
			names = append(names, e.Name())
		case e.IsDir() && strings.HasSuffix(namefold.ModTree(e.Name()), ".text"):
			root := filepath.Join(dir, e.Name())
			err := filepath.WalkDir(root, func(p string, de fs.DirEntry, err error) error {
				if err != nil || !de.Type().IsRegular() {
					return err
				}
				rel, err := filepath.Rel(dir, p)
				names = append(names, filepath.ToSlash(rel))
				return err
			})
			if err != nil {
				return nil, err
			}
		}
	}
	return names, nil
}

// addFolderMaps adds every map of host folder, with its pages and labels, as
// drafts of w's project. The files go where the mod's one placement rule puts
// each stem, and each name passes the rule a row's map passes.
func (s *mapInspectionEdits) addFolderMaps(w *editorWorkspaceState, folder string) error {
	x, d := s.inspector, w.Project
	if d == nil {
		return errors.New(x.word("editor.mod.err.nomission"))
	}
	folder = strings.TrimSpace(folder)
	names, err := folderNames(folder)
	if err != nil {
		return fmt.Errorf(x.word("editor.mod.err.mapsfolder"), folder, err)
	}
	maps := mod.FolderMaps(names)
	if len(maps) == 0 {
		return fmt.Errorf(x.word("editor.mod.err.foldernomaps"), folder)
	}
	places := mod.PlaceFolderMaps(s.projectFiles(d), maps)
	drafts := map[string][]byte{}
	for _, name := range slices.Sorted(func(yield func(string) bool) {
		for k := range places {
			if !yield(k) {
				return
			}
		}
	}) {
		rel, err := mod.MissionMap.Path(places[name])
		if err != nil {
			return err
		}
		host := filepath.Join(folder, filepath.FromSlash(name))
		if st, err := os.Stat(host); err != nil {
			return err
		} else if st.Size() > mod.MaxFileBytes {
			return fmt.Errorf(x.word("editor.mod.err.filebig"), name, mod.MaxFileBytes)
		}
		b, err := os.ReadFile(host)
		if err != nil {
			return err
		}
		drafts[rel] = b
	}
	for rel, b := range drafts {
		d.Files[rel] = b
	}
	return nil
}
