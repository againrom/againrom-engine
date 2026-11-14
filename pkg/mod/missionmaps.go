package mod

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	"againrom/pkg/namefold"
)

// missionMapName is a file name the file form reads: a mission number with
// no leading zero, then .alm or .almx, matched without case.
var missionMapName = regexp.MustCompile(`(?i)^([1-9][0-9]{0,4})\.almx?$`)

// missionMap is one replacing mission of the file form. at is the number of
// declared missions when its mod's files were read: its place in load order.
// err is the refusal its map or pages gave, raised only when the install
// holds the campaign map the file names.
type missionMap struct {
	Mission
	at  int
	err error
}

// missionMapNumber is the mission a file name of the file form names.
func missionMapNumber(name string) (int, bool) {
	sub := missionMapName.FindStringSubmatch(name)
	if sub == nil {
		return 0, false
	}
	n, err := strconv.Atoi(sub[1])
	return n, err == nil && n <= maxMissionNumber
}

// ByFileName reports a replacing mission the file form declares: its map
// file is the declaration, and no row of the missions file holds it.
func (m Mission) ByFileName() bool { return m.byName }

// WithMissions is d holding rows as its missions; the file form's missions
// that wait for Listed stay as read; file-form rows are left out.
func (d MissionData) WithMissions(rows []Mission) MissionData {
	d.Missions = missionRows(rows)
	return d
}

// RowID names m among its mod's declarations: the key of a row, or the map
// file of a file-form mission, which has no key. A key holds no dot, so the
// two never meet.
func (m Mission) RowID() string {
	if m.byName {
		return m.Map
	}
	return m.Key
}

// missionRows are missions without the file form's.
func missionRows(missions []Mission) []Mission {
	out := make([]Mission, 0, len(missions))
	for _, m := range missions {
		if !m.byName {
			out = append(out, m)
		}
	}
	return out
}

// readMissionMaps reads the files missionMapFiles finds, waiting for Listed;
// it reports whether the mod holds one.
func readMissionMaps(r DataReader, d *Data, t Tree) (bool, error) {
	found, err := missionMapFiles(t, r.Mod, r.Lang, d.Missions.Missions)
	for _, m := range found {
		err := m.ReadFiles(r.Dir)
		d.Missions.maps = append(d.Missions.maps, missionMap{Mission: m, at: len(d.Missions.Missions), err: err})
	}
	return len(found) > 0, err
}

// rowMaps are the maps mod id's rows name.
func rowMaps(rows []Mission, id string) map[string]bool {
	named := map[string]bool{}
	for _, m := range rows {
		if m.Mod == id && !m.Off && !m.byName {
			named[namefold.ModTree(m.Map)] = true
		}
	}
	return named
}

// missionMapFiles are the file form's missions of mod id in tree t: each
// number-named file of the top or maps folder no row names. It opens none.
func missionMapFiles(t Tree, id, lang string, rows []Mission) ([]Mission, error) {
	named := rowMaps(rows, id)
	var out []Mission
	for _, folder := range []string{".", MapsFolder} {
		if st, err := t.Stat(folder); err != nil || !st.IsDir() {
			continue
		}
		entries, err := t.ReadDir(folder)
		if err != nil {
			return out, err
		}
		for _, e := range entries {
			n, ok := missionMapNumber(e.Name())
			if !ok || e.IsDir() {
				continue
			}
			rel := path.Join(folder, e.Name())
			if named[namefold.ModTree(rel)] {
				continue
			}
			out = append(out, Mission{Mod: id, File: rel, Kind: MissionReplace, Number: n, Map: rel, language: lang, byName: true})
		}
	}
	return out, nil
}

// Listed is the one mission set of d for the starter, the editor and the
// game: its rows and the file form's missions at their load-order place.
// held answers nil for a number whose campaign map the install holds; any
// other number's file is a warning. A nil held keeps every file. A file a
// row names is that row's map; a second declaration of a kept file's
// mission is refused naming both.
func (d MissionData) Listed(held func(n int) error) (MissionData, []string, error) {
	if len(d.maps) == 0 {
		return d, nil, nil
	}
	var warnings []string
	var kept []missionMap
	named := map[string]map[string]bool{}
	for _, m := range d.maps {
		if named[m.Mod] == nil {
			named[m.Mod] = rowMaps(d.Missions, m.Mod)
		}
		if named[m.Mod][namefold.ModTree(m.Map)] {
			continue
		}
		if held != nil {
			if err := held(m.Number); err != nil {
				warnings = append(warnings, fmt.Sprintf("mod %q: %s: %v; the file replaces nothing", m.Mod, m.File, err))
				continue
			}
		}
		if m.err != nil {
			return MissionData{}, warnings, fmt.Errorf("mod %q: %w", m.Mod, m.err)
		}
		for _, other := range kept {
			if other.Mod == m.Mod && other.Number == m.Number {
				return MissionData{}, warnings, Source{Mod: m.Mod, File: m.File}.Errorf(
					"%s is a map of mission %d too; keep one of the two files", other.File, m.Number)
			}
		}
		for _, row := range d.Missions {
			if row.Mod != m.Mod || row.Number != m.Number {
				continue
			}
			at := Source{Mod: row.Mod, File: row.File, Line: row.Line}
			if row.Kind == MissionReplace && !row.Off {
				return MissionData{}, warnings, at.Errorf("mission %d is replaced with %s here, and %s replaces it by its name; name %s as the map or remove one of them",
					m.Number, row.Map, m.File, m.File)
			}
			return MissionData{}, warnings, at.Errorf("mission %d is declared here, and %s replaces it by its name; remove one of them", m.Number, m.File)
		}
		kept = append(kept, m)
	}
	out := MissionData{Order: d.Order, Missions: make([]Mission, 0, len(d.Missions)+len(kept))}
	next := 0
	for i, row := range d.Missions {
		for ; next < len(kept) && kept[next].at <= i; next++ {
			out.Missions = append(out.Missions, kept[next].Mission)
		}
		out.Missions = append(out.Missions, row)
	}
	for ; next < len(kept); next++ {
		out.Missions = append(out.Missions, kept[next].Mission)
	}
	return out, warnings, nil
}

// Replacements reports campaign rows and replaced maps with their source.
func (d MissionData) Replacements() []string {
	var out []string
	for _, m := range d.Missions {
		if m.Off {
			continue
		}
		if m.Kind == MissionCampaign {
			var fields []string
			line := m.Line
			if m.AutoGetMission != nil {
				fields = append(fields, fmt.Sprintf("AutoGetMission = %d", *m.AutoGetMission))
				if m.AutoLine > 0 {
					line = m.AutoLine
				}
			}
			if m.Payment != nil {
				fields = append(fields, fmt.Sprintf("Payment = %d", *m.Payment))
			}
			if m.Map != "" {
				fields = append(fields, "map "+m.Map)
			}
			switch {
			case m.Guest != 0:
				fields = append(fields, fmt.Sprintf("guest = %d, chapter = %d", m.Guest, m.Chapter))
			case len(m.Party) != 0:
				var members []string
				for _, member := range m.Party {
					members = append(members, strconv.Quote(member.Character))
				}
				fields = append(fields, "party = ["+strings.Join(members, ", ")+"]")
			default:
				fields = append(fields, "party = main")
			}
			out = append(out, fmt.Sprintf("mission %d %s by mod %q, declared at %s:%d", m.Number, strings.Join(fields, ", "), m.Mod, m.File, line))
			continue
		}
		if m.AutoGetMission != nil {
			out = append(out, fmt.Sprintf("mission %d AutoGetMission = %d by mod %q, declared at %s:%d", m.Number, *m.AutoGetMission, m.Mod, m.File, m.AutoLine))
		}
		if m.Kind != MissionReplace {
			continue
		}
		by := "named by its file name"
		if !m.byName {
			by = fmt.Sprintf("declared at %s:%d", m.File, m.Line)
		}
		out = append(out, fmt.Sprintf("mission %d replaced by mod %q: map %s, %s", m.Number, m.Mod, m.Map, by))
	}
	return out
}
