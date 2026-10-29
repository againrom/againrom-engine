package game

import (
	"bytes"
	"errors"
	"fmt"

	"againrom/pkg/mod"
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
		if mission.Key == m.selected {
			d.Selected = i
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

func (d *editorModDocument) selected(p editorPackageSpec) (mod.Mission, error) {
	if d.Selected < 0 || d.Selected >= len(d.Missions) {
		return mod.Mission{}, fmt.Errorf("select a mission")
	}
	m := mod.CloneMission(d.Missions[d.Selected])
	if m.Off {
		return m, nil
	}
	m.Kind, m.Number = p.Kind, p.Number
	m.After, m.Guest, m.Chapter, m.Gold = p.After, p.Guest, p.Chapter, p.Gold
	m.Party = mod.CloneMission(mod.Mission{Party: p.Party}).Party
	if m.Kind == mod.MissionReplace {
		m.After, m.Guest, m.Chapter, m.Gold, m.Carry, m.Party = 0, 0, 0, 0, "", nil
	}
	if len(m.Party) != 0 {
		m.Guest, m.Chapter, m.Carry = 0, 0, ""
	}
	return m, nil
}

func (s *mapInspectionEdits) retainedPackageFiles(b []byte) (map[string][]byte, []mod.Mission, mod.Mission, error) {
	d := s.workspace.Project.clone()
	m, err := d.selected(s.workspace.Package)
	if err != nil {
		return nil, nil, m, err
	}
	if m.Off {
		return nil, nil, m, fmt.Errorf("disabled declaration has no open map")
	}
	d.Missions[d.Selected] = m
	m.MapBytes = bytes.Clone(b)
	m.TextFiles = s.workspace.Pages
	d.Files[m.Map] = bytes.Clone(b)
	d.Missions[d.Selected] = m
	data, err := mod.WriteMissions(d.Source, d.Original, d.Missions)
	if err != nil {
		return nil, nil, m, err
	}
	d.Files[mod.MissionsFile] = data
	d.Files[mod.ManifestName], err = mod.WriteManifestTitle(d.Manifest, s.workspace.Package.Title)
	if err != nil {
		return nil, nil, m, err
	}
	return d.Files, d.Missions, m, nil
}

func (x *MapInspector) checkMissionSet(missions []mod.Mission) error {
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
			return errors.New("mod files changed outside the editor; reopen the mod before saving")
		}
	}
	return nil
}
