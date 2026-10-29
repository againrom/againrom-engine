package game

import (
	"bytes"
	"fmt"
	"path/filepath"
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
	m := w.Package.mission(s.model.Bytes(), w.Pages)
	pages, err := s.exportPageTree("", nil)
	if err != nil {
		return err
	}
	if len(pages) > 0 {
		m.Texts = testPlayTexts + "/{lang}"
	}
	files := missionPackageFiles(w.Package.Title, "Map editor mission package", m, s.inspector.pageLanguage(), pages)
	w.Project = &editorModDocument{Source: bytes.Clone(files[mod.MissionsFile]), Manifest: bytes.Clone(files[mod.ManifestName]), Original: []mod.Mission{mod.CloneMission(m)}, Missions: []mod.Mission{m}, Files: files}
	return nil
}

func (s *mapInspectionEdits) setModControl(key, value string) (*ui.InspectionDocument, bool, error) {
	if key == "publish" || key == "publish-save" || key == "publish-replace" {
		return nil, true, s.publishEditorMod(key, value)
	}
	if key == "output" {
		dir, err := editorPhysicalDirectory(strings.TrimSpace(value))
		if err == nil {
			s.inspector.modOutput = dir
		}
		return nil, true, err
	}
	if key == "save" {
		if s.inspector.mod == nil {
			return nil, true, fmt.Errorf("choose a working mod folder first")
		}
		file, err := s.SaveIntoMod()
		if err == nil {
			s.saveNote = s.inspector.word("editor.mod.saved") + " " + filepath.Dir(file)
		}
		return nil, true, err
	}
	known := key == "landing" || key == "add" || key == "remove" || key == "up" || key == "down" || strings.HasPrefix(key, "party.")
	if !known {
		return nil, false, nil
	}
	d, err := s.changeWorkspace(func(w *editorWorkspaceState) error {
		p := &w.Package
		switch key {
		case "landing":
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
				return fmt.Errorf("choose Companion or Party")
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
				return fmt.Errorf("select a mission")
			}
			if !slices.ContainsFunc(append(slices.Clone(d.Missions[:i]), d.Missions[i+1:]...), func(m mod.Mission) bool { return !m.Off }) {
				return fmt.Errorf("a mod needs an enabled mission")
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
			w.Package = modPackageSpec(d.Missions[d.Selected], p.ID, p.Title)
		case "add":
			if w.Project == nil {
				return s.newModDocument(w)
			}
			d := w.Project
			if d.Selected >= 0 && d.Selected < len(d.Missions) && !d.Missions[d.Selected].Off {
				return nil
			}
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
				return fmt.Errorf("no unused mission number")
			}
			m.Mod = p.ID
			m.Key = fmt.Sprintf("map%d", m.Number)
			m.Map = "mission/" + m.Key + "/map.alm"
			m.Party = []mod.GuestMember{{Character: mod.InstallCharacterKey(int32(m.Guest))}}
			m.Guest, m.Chapter, m.Carry = 0, 0, ""
			pages, err := s.exportPageTree("", nil)
			if err != nil {
				return err
			}
			if len(pages) > 0 {
				root := "mission/" + m.Key + "/text"
				m.Texts = root + "/{lang}"
				for name, b := range pages {
					d.Files[root+"/"+name] = bytes.Clone(b)
				}
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
			w.Package = modPackageSpec(m, p.ID, p.Title)
		default:
			parts := strings.Split(key, ".")
			if len(parts) != 3 {
				return fmt.Errorf("invalid party field")
			}
			at, e := strconv.Atoi(parts[1])
			if e != nil {
				return e
			}
			if parts[2] == "add" {
				if len(p.Party) >= mod.MaxGuestParty {
					return fmt.Errorf("a party has at most %d members", mod.MaxGuestParty)
				}
				characters := s.modCharacters()
				for _, c := range characters {
					if !slices.ContainsFunc(p.Party, func(g mod.GuestMember) bool { return g.Character == c.Key }) {
						p.Party = append(p.Party, mod.GuestMember{Character: c.Key})
						return nil
					}
				}
				return fmt.Errorf("choose an unused character")
			}
			if at < 0 || at >= len(p.Party) {
				return fmt.Errorf("select a party member")
			}
			g := &p.Party[at]
			switch parts[2] {
			case "character":
				n, e := strconv.Atoi(value)
				if e == nil {
					cs := s.modCharacters()
					if n < 0 || n >= len(cs) {
						return fmt.Errorf("unknown character")
					}
					value = cs[n].Key
				}
				for i, c := range p.Party {
					if i != at && c.Character == value {
						return fmt.Errorf("party already names %s", value)
					}
				}
				g.Character = value
			case "level":
				n, e := strconv.Atoi(value)
				if e != nil || n < 0 || n > mod.MaxGuestLevel {
					return fmt.Errorf("level must be Default or 1..4")
				}
				g.Level = n
			case "remove":
				if len(p.Party) == 1 {
					return fmt.Errorf("a party needs at least one member")
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
					return fmt.Errorf("choose Template or Override all armour")
				}
			default:
				kind := parts[2]
				if kind != "weapon" && kind != "shield" && !strings.HasPrefix(kind, "armour") {
					return fmt.Errorf("unknown party field %q", kind)
				}
				cell := data.HumanCellWeapon
				if kind == "shield" {
					cell = data.HumanCellShield
				}
				if strings.HasPrefix(kind, "armour") {
					n, e := strconv.Atoi(strings.TrimPrefix(kind, "armour"))
					if e != nil || n < 0 || n >= 8 {
						return fmt.Errorf("invalid armour cell")
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
							return fmt.Errorf("unknown %s item", kind)
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
	})
	if err == nil && key == "remove" {
		d, err = s.activateWorkspaceMission(s.workspace.Project.Selected+1, false)
	}
	return d, true, err
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
