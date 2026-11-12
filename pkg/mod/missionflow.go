package mod

import "slices"

// ValidateCampaign checks the merged declarations against the install's sections.
func (d MissionData) ValidateCampaign(main, side, offered []int, auto map[int]int, retained ...map[int]int) error {
	known := append(slices.Clone(main), side...)
	known = append(known, offered...)
	installed := slices.Clone(known)
	links := make(map[int]int, len(auto))
	for n, next := range auto {
		links[n] = next
	}
	for _, m := range d.Missions {
		if !m.Campaign() {
			continue
		}
		at := Source{Mod: m.Mod, File: m.File, Line: m.Line}
		if !slices.Contains(known, m.Number) {
			if m.Kind == MissionReplace {
				return at.Errorf("number %d is no mission of this install's campaign", m.Number)
			}
			if m.Kind != MissionCampaign || m.Map == "" {
				return at.Errorf("unknown mission %d; an added campaign mission needs a map", m.Number)
			}
			if m.Number%10 == 0 {
				return at.Errorf("added mission %d must be side-numbered (not a multiple of 10)", m.Number)
			}
			known = append(known, m.Number)
		}
	}
	for _, m := range d.Missions {
		if m.AutoGetMission == nil {
			continue
		}
		at := Source{Mod: m.Mod, File: m.File, Line: m.AutoLine}
		if at.Line == 0 {
			at.Line = m.Line
		}
		next := *m.AutoGetMission
		if next == -1 {
			delete(links, m.Number)
			continue
		}
		if !slices.Contains(known, next) {
			return at.Errorf("AutoGetMission names unknown mission %d", next)
		}
		if slices.Contains(installed, next) && !slices.Contains(main, next) {
			return at.Errorf("AutoGetMission names installed side mission %d; use a later main mission or an added mission", next)
		}
		links[m.Number] = next
	}
	for _, m := range d.Missions {
		if m.AutoGetMission == nil {
			continue
		}
		at := Source{Mod: m.Mod, File: m.File, Line: m.AutoLine}
		if at.Line == 0 {
			at.Line = m.Line
		}
		seen := map[int]bool{m.Number: true}
		for n := m.Number; ; {
			next, ok := links[n]
			if !ok {
				break
			}
			if seen[next] {
				return at.Errorf("AutoGetMission cycle from mission %d through mission %d", m.Number, next)
			}
			seen[next], n = true, next
		}
	}
	reached := map[int]bool{}
	for _, root := range installed {
		current := root / 10 * 10
		if len(retained) != 0 {
			if main, ok := retained[0][root]; ok {
				current = main
			}
		}
		for n, steps := root, 0; steps <= len(known); steps++ {
			reached[n] = true
			next, ok := links[n]
			if !ok {
				break
			}
			if next <= current {
				if m, ok := d.CampaignMission(n); ok && m.AutoGetMission != nil {
					line := m.AutoLine
					if line == 0 {
						line = m.Line
					}
					return Source{Mod: m.Mod, File: m.File, Line: line}.Errorf("AutoGetMission %d is not above current main mission %d", next, current)
				}
			}
			if slices.Contains(main, next) {
				current = next
			}
			n = next
		}
	}
	for _, m := range d.Missions {
		if m.Campaign() && !slices.Contains(installed, m.Number) && !reached[m.Number] {
			return Source{Mod: m.Mod, File: m.File, Line: m.Line}.Errorf("added mission %d is not reached by AutoGetMission", m.Number)
		}
	}
	return nil
}
