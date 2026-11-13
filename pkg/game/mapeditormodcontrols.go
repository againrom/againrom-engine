package game

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

func (s *mapInspectionEdits) newModDocument(w *editorWorkspaceState) error {
	if w.Project != nil {
		return nil
	}
	m := w.Package.mission(s.model.Bytes(), w.Pages, s.place())
	pages, err := s.exportPageTree("", nil)
	if err != nil {
		return err
	}
	files := missionPackageFiles(w.Package.Title, w.Package.Manifest.Description, m, s.inspector.pageLanguage(), s.labelsForSave(), pages)
	files[mod.ManifestName], err = w.Package.manifestBytes(nil)
	if err != nil {
		return err
	}
	w.Project = &editorModDocument{Source: bytes.Clone(files[mod.MissionsFile]), Manifest: bytes.Clone(files[mod.ManifestName]), Original: []mod.Mission{mod.CloneMission(m)}, Missions: []mod.Mission{m}, Files: files}
	return nil
}

func (s *mapInspectionEdits) setModControl(key, value string) (*ui.InspectionDocument, bool, error) {
	if key == "publish" || key == "publish-save" || key == "publish-replace" {
		return nil, true, s.publishEditorMod(key, value)
	}
	if isModAssetKey(key) {
		return s.setModAsset(key, value)
	}
	if key == "output" {
		return nil, true, s.inspector.setPublishFolder(value)
	}
	if key == "save" {
		if s.inspector.mod == nil {
			// A mod with no folder is saved into the proposed working folder,
			// which Save makes; a folder already there is refused, not
			// written over.
			dir := s.inspector.proposedModFolder(s.workspace.Package.ID)
			if dir == "" {
				return nil, true, errors.New(s.inspector.word("editor.mod.err.nofolder"))
			}
			if _, err := os.Lstat(dir); err == nil {
				return nil, true, fmt.Errorf(s.inspector.word("editor.mod.err.exists"), dir)
			}
			d, err := s.openOrCreateWorkspace(dir)
			if err == nil {
				s.saveNote = s.inspector.word("editor.mod.saved") + " " + dir
			}
			return d, true, err
		}
		_, err := s.SaveIntoMod()
		if err == nil {
			s.saveNote = s.inspector.word("editor.mod.saved") + " " + s.inspector.mod.dir
		}
		return nil, true, err
	}
	known := key == "landing" || key == "add" || key == "remove" || key == "up" || key == "down" || strings.HasPrefix(key, "party.")
	if !known {
		return nil, false, nil
	}
	d, err := s.changeWorkspace(func(w *editorWorkspaceState) error {
		return s.editModControl(w, key, value)
	})
	if err == nil && key == "remove" {
		d, err = s.activateWorkspaceMission(s.workspace.Project.Selected+1, false)
	}
	return d, true, err
}

func (s *mapInspectionEdits) editModControl(w *editorWorkspaceState, key, value string) error {
	if err := s.fileMissionFixed(w); err != nil && key != "add" {
		return err
	}
	p := &w.Package
	switch key {
	case "insert":
		return s.insertCampaignMission(w, value)
	case "landing":
		p.MainParty = false
		if value == "2" && (p.Kind == mod.MissionCampaign || p.Kind == mod.MissionReplace) {
			p.MainParty, p.Party, p.Guest, p.Chapter, p.Gold = true, nil, 0, 0, 0
			return nil
		}
		if value == "1" {
			if len(p.Party) == 0 {
				n := p.Guest
				if n == 0 {
					c, e := LoadCampaign(s.inspector.archives.Containers)
					if e != nil {
						return e
					}
					m, e := testPlayGuest(c, s.inspector.archives.Containers)
					if e != nil {
						return e
					}
					n = m.Guest
				}
				p.Party = []mod.GuestMember{{Character: mod.InstallCharacterKey(int32(n))}}
			}
			p.Guest, p.Chapter = 0, 0
		} else if value == "0" {
			p.Party = nil
			c, e := LoadCampaign(s.inspector.archives.Containers)
			if e != nil {
				return e
			}
			m, e := testPlayGuest(c, s.inspector.archives.Containers)
			if e != nil {
				return e
			}
			p.Guest, p.Chapter = m.Guest, m.Chapter
			if town, ok := s.modDestination(w); ok {
				p.Chapter = town
			}
		} else {
			return errors.New(s.inspector.word("editor.mod.err.landing"))
		}
	case "up", "down":
		if err := s.newModDocument(w); err != nil {
			return err
		}
		d := w.Project
		i := d.Selected
		j := i + 1
		if key == "up" {
			j = i - 1
		}
		if i < 0 || j < 0 || j >= len(d.Missions) {
			return nil
		}
		d.Missions[i], d.Missions[j] = d.Missions[j], d.Missions[i]
		d.Selected = j
	case "remove":
		if err := s.newModDocument(w); err != nil {
			return err
		}
		d := w.Project
		i := d.Selected
		if i < 0 || i >= len(d.Missions) {
			return errors.New(s.inspector.word("editor.mod.err.nomission"))
		}
		if !slices.ContainsFunc(append(slices.Clone(d.Missions[:i]), d.Missions[i+1:]...), func(m mod.Mission) bool { return !m.Off }) {
			return errors.New(s.inspector.word("editor.mod.err.lastmission"))
		}
		m := d.Missions[i]
		if !m.Off && m.Kind != mod.MissionReplace {
			for _, next := range d.Missions {
				if next.After == m.Number && next.After != m.After && value != "confirmed" {
					return &ui.WorkspaceConfirmation{Action: "remove", Message: s.inspector.word("editor.mod.reconnect")}
				}
			}
			for n := range d.Missions {
				if d.Missions[n].After == m.Number {
					d.Missions[n].After = m.After
				}
			}
		}
		d.Missions = append(d.Missions[:i], d.Missions[i+1:]...)
		d.Selected = min(i, len(d.Missions)-1)
		for d.Missions[d.Selected].Off {
			d.Selected = (d.Selected + 1) % len(d.Missions)
		}
		w.Package = modPackageSpec(d.Missions[d.Selected], *p)
	case "add":
		if w.Project == nil {
			return s.newModDocument(w)
		}
		d := w.Project
		if d.Selected >= 0 && d.Selected < len(d.Missions) && !d.Missions[d.Selected].Off {
			return nil
		}
		d.Loose = ""
		c, e := LoadCampaign(s.inspector.archives.Containers)
		if e != nil {
			return e
		}
		m, e := testPlayGuest(c, s.inspector.archives.Containers)
		if e != nil {
			return e
		}
		for ; m.Number <= 65535; m.Number++ {
			used := slices.ContainsFunc(d.Missions, func(v mod.Mission) bool { return v.Number == m.Number })
			if !used && checkGuest(c, s.inspector.archives.Containers, mod.MissionData{Missions: []mod.Mission{m}}, m) == nil {
				break
			}
		}
		if m.Number > 65535 {
			return errors.New(s.inspector.word("editor.mod.err.nonumber"))
		}
		m.Mod = p.ID
		m.Key = fmt.Sprintf("map%d", m.Number)
		place := s.place().Free(s.projectFiles(d))
		m.Map = place.Map
		m.Party = []mod.GuestMember{{Character: mod.InstallCharacterKey(int32(m.Guest))}}
		m.Guest, m.Chapter, m.Carry = 0, 0, ""
		pages, err := s.exportPageTree("", nil)
		if err != nil {
			return err
		}
		for name, b := range pages {
			d.Files[place.Pages("")+"/"+name] = bytes.Clone(b)
		}
		if labels := s.labelsForSave(); len(labels) > 0 {
			d.Files[place.Labels()] = labels
		}
		for i := len(d.Missions) - 1; i >= 0; i-- {
			if d.Missions[i].Kind == mod.MissionGuest && !d.Missions[i].Off {
				m.After = d.Missions[i].Number
				break
			}
		}
		d.Missions = append(d.Missions, m)
		d.Selected = len(d.Missions) - 1
		d.Files[m.Map] = s.model.Bytes()
		w.Package = modPackageSpec(m, *p)
	default:
		parts := strings.Split(key, ".")
		if len(parts) != 3 {
			return errors.New(s.inspector.word("editor.mod.err.partyfield"))
		}
		at, e := strconv.Atoi(parts[1])
		if e != nil {
			return e
		}
		if parts[2] == "add" {
			if len(p.Party) >= mod.MaxGuestParty {
				return fmt.Errorf(s.inspector.word("editor.mod.err.partyfull"), mod.MaxGuestParty)
			}
			characters := s.modCharacters()
			for _, c := range characters {
				if !slices.ContainsFunc(p.Party, func(g mod.GuestMember) bool { return g.Character == c.Key }) {
					p.Party = append(p.Party, mod.GuestMember{Character: c.Key})
					return nil
				}
			}
			return errors.New(s.inspector.word("editor.mod.err.nocharacter"))
		}
		if at < 0 || at >= len(p.Party) {
			return errors.New(s.inspector.word("editor.mod.err.nomember"))
		}
		g := &p.Party[at]
		switch parts[2] {
		case "character":
			n, e := strconv.Atoi(value)
			if e == nil {
				cs := s.modCharacters()
				if n < 0 || n >= len(cs) {
					return errors.New(s.inspector.word("editor.mod.err.character"))
				}
				value = cs[n].Key
			}
			for i, c := range p.Party {
				if i != at && c.Character == value {
					return fmt.Errorf(s.inspector.word("editor.mod.err.named"), value)
				}
			}
			g.Character = value
		case "level":
			n, e := strconv.Atoi(value)
			if e != nil || n < 0 || n > mod.MaxGuestLevel {
				return errors.New(s.inspector.word("editor.mod.err.level"))
			}
			g.Level = n
		case "remove":
			if len(p.Party) == 1 {
				return errors.New(s.inspector.word("editor.mod.err.lastmember"))
			}
			p.Party = append(p.Party[:at], p.Party[at+1:]...)
		case "up", "down":
			j := at + 1
			if parts[2] == "up" {
				j = at - 1
			}
			if j >= 0 && j < len(p.Party) {
				p.Party[at], p.Party[j] = p.Party[j], p.Party[at]
			}
		case "armour":
			if value == "0" {
				g.Armour = nil
			} else if value == "1" {
				if g.Armour == nil {
					g.Armour = []string{}
				}
			} else {
				return errors.New(s.inspector.word("editor.mod.err.armour"))
			}
		default:
			kind := parts[2]
			if kind != "weapon" && kind != "shield" && !strings.HasPrefix(kind, "armour") {
				return fmt.Errorf(s.inspector.word("editor.mod.err.partykey"), kind)
			}
			cell := data.HumanCellWeapon
			if kind == "shield" {
				cell = data.HumanCellShield
			}
			if strings.HasPrefix(kind, "armour") {
				n, e := strconv.Atoi(strings.TrimPrefix(kind, "armour"))
				if e != nil || n < 0 || n >= 8 {
					return errors.New(s.inspector.word("editor.mod.err.armourcell"))
				}
				cell = data.HumanCellArmour + n
				kind = "armour"
			}
			if value == "-1" && kind != "armour" {
				if kind == "weapon" {
					g.Weapon = nil
				} else {
					g.Shield = nil
				}
				return nil
			}
			name := value
			if n, e := strconv.ParseInt(value, 10, 64); e == nil {
				if n == 0 {
					name = ""
				} else {
					var ok bool
					name, ok = s.partyItemName(kind, n)
					if !ok {
						return fmt.Errorf(s.inspector.word("editor.mod.err.item"), kind)
					}
				}
			}
			if s.inspector.table != nil {
				if e := mapload.CheckCharacterCells(s.inspector.table, map[int]string{cell: name}); e != nil {
					return e
				}
			}
			if kind == "weapon" {
				g.Weapon = &name
			} else if kind == "shield" {
				g.Shield = &name
			} else {
				for len(g.Armour) <= cell-data.HumanCellArmour {
					g.Armour = append(g.Armour, "")
				}
				g.Armour[cell-data.HumanCellArmour] = name
			}
		}
	}
	return nil
}

func (s *mapInspectionEdits) modCharacters() []mod.Character {
	var out []mod.Character
	if s.inspector.table != nil {
		out = installCharacters(s.inspector.table)
	}
	if m := s.inspector.mod; m != nil && m.catalog != nil {
		out = slices.Clone(m.catalog)
	}
	for _, g := range s.workspace.Package.Party {
		if !slices.ContainsFunc(out, func(c mod.Character) bool { return c.Key == g.Character }) {
			out = append(out, mod.Character{Key: g.Character, Name: g.Character})
		}
	}
	slices.SortFunc(out, func(a, b mod.Character) int { return strings.Compare(a.Key, b.Key) })
	return out
}

func (s *mapInspectionEdits) partyItemName(kind string, code int64) (string, bool) {
	t := s.inspector.table
	if t == nil {
		return "", false
	}
	c := data.ItemCode(code)
	switch kind {
	case "weapon":
		v, e := data.WeaponFromCode(c, t.Shapes, t.Materials, t.Weapons)
		return v.Name, e == nil
	case "shield":
		v, e := data.ShieldFromCode(c, t.Shapes, t.Materials, t.Shields)
		return v.Name, e == nil
	default:
		v, e := data.ArmorFromCode(c, t.Shapes, t.Materials, t.Armors)
		return v.Name, e == nil
	}
}
