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

func (s *mapInspectionEdits) mapList(d *editorModDocument) ([]mod.MapEntry, mod.MissionData) {
	data, _ := s.inspector.modMissions(s.inspector.mod.description(), d.Missions)
	return mod.MapList(s.projectFiles(d), "", data), data
}

func (x *MapInspector) replaceNumbers(data mod.MissionData) []int {
	flow := x.installFlow()
	if flow == nil {
		return nil
	}
	c := flow.campaign
	return data.ReplaceNumbers(slices.Concat(c.Main, c.Side, c.Offered), x.missionAnswer())
}

func (x *MapInspector) mapRefusal(err error, rel string, n int) error {
	switch {
	case errors.Is(err, mod.ErrMapNamed):
		return fmt.Errorf(x.word("editor.mod.err.maptaken"), rel)
	case errors.Is(err, mod.ErrNotByName):
		return fmt.Errorf(x.word("editor.mod.err.notbyname"), rel)
	case errors.Is(err, mod.ErrNotCampaign):
		return fmt.Errorf(x.word("editor.mod.err.replacenumber"), n)
	case errors.Is(err, mod.ErrReplaced):
		return fmt.Errorf(x.word("editor.mod.err.replacetaken"), n)
	}
	return err
}

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

func (s *mapInspectionEdits) mapOf(list []mod.MapEntry, rel string) (mod.MapEntry, error) {
	if i := mapEntryOf(list, rel); i >= 0 {
		return list[i], nil
	}
	return mod.MapEntry{}, fmt.Errorf(s.inspector.word("editor.mod.err.mapunknown"), rel)
}

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
		return x.mapRefusal(mod.ErrMapNamed, e.Path, 0)
	}
	var guest mod.Mission
	if replace == 0 {
		if guest, err = s.guestRow(d, w.Package.ID); err != nil {
			return err
		}
	}
	row, err := mod.MakeMission(w.Package.ID, data, e, guest, replace, x.missionAnswer())
	if err != nil {
		return x.mapRefusal(err, e.Path, replace)
	}
	d.Missions = append(d.Missions, row)
	if d.Selected < 0 && namefold.ModTree(d.Loose) == namefold.ModTree(row.Map) {
		d.Selected, d.Loose = len(d.Missions)-1, ""
		w.Package = modPackageSpec(row, w.Package)
	}
	return nil
}

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
	row, err := mod.MakeRow(w.Package.ID, data, e, x.missionAnswer())
	if err != nil {
		return x.mapRefusal(err, e.Path, e.Number)
	}
	at := slices.IndexFunc(d.Missions, func(m mod.Mission) bool {
		return m.Kind == mod.MissionReplace && m.Number == e.Number && namefold.ModTree(m.Map) == namefold.ModTree(e.Path)
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

func (s *mapInspectionEdits) addOpenMap(w *editorWorkspaceState) error {
	if w.Project == nil || s.model == nil {
		return errors.New(s.inspector.word("editor.mod.err.nomaptoadd"))
	}
	_, err := s.draftOpenMap(w.Project)
	return err
}

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
