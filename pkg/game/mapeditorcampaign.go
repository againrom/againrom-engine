package game

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

func campaignFieldKey(key string) bool {
	return key == "AutoGetMission" || key == "landing" || key == "guest" || key == "chapter" ||
		key == "gold" || key == "class" || key == "sex" || strings.HasPrefix(key, "party.")
}

func (s *mapInspectionEdits) campaignRow(w *editorWorkspaceState, n int) (int, error) {
	for i, m := range w.Project.Missions {
		if m.Number == n {
			if m.Off {
				return -1, fmt.Errorf(s.inspector.word("editor.graph.err.campaign"), n)
			}
			if m.ByFileName() {
				w.Project.Missions[i] = mod.Mission{Mod: m.Mod, File: mod.MissionsFile, Key: s.campaignKey(w, n),
					Kind: mod.MissionCampaign, Number: n, Map: m.Map, MapBytes: m.MapBytes}
			}
			return i, nil
		}
	}
	flow := s.inspector.installFlow()
	if flow == nil || !flow.campaign.holds(n) {
		return -1, fmt.Errorf(s.inspector.word("editor.graph.err.campaign"), n)
	}
	m := mod.Mission{Mod: w.Package.ID, File: mod.MissionsFile, Kind: mod.MissionCampaign, Number: n, Key: s.campaignKey(w, n)}
	w.Project.Missions = append(w.Project.Missions, m)
	return len(w.Project.Missions) - 1, nil
}

func (s *mapInspectionEdits) campaignKey(w *editorWorkspaceState, n int) string {
	key := "mission" + strconv.Itoa(n)
	for suffix := 2; slices.ContainsFunc(w.Project.Missions, func(m mod.Mission) bool { return m.Key == key }); suffix++ {
		key = fmt.Sprintf("mission%d_%d", n, suffix)
	}
	return key
}

func (s *mapInspectionEdits) campaignFields(scope string) []ui.InspectionField {
	if s.prepareWorkspace() != nil || s.workspace.Project == nil || s.inspector.mod == nil {
		return nil
	}
	n, _ := strconv.Atoi(strings.TrimPrefix(scope, "campaign."))
	w := cloneWorkspace(s.workspace)
	i, err := s.campaignRow(w, n)
	if err != nil {
		return nil
	}
	w.Project.Selected = i
	w.Package = modPackageSpec(w.Project.Missions[i], w.Package)
	trial := *s
	trial.workspace = w
	var out []ui.InspectionField
	for _, f := range trial.WorkspaceFields("package") {
		if campaignFieldKey(f.Key) {
			f.Column = 0
			out = append(out, f)
		}
	}
	if w.Project.Missions[i].Campaign() {
		add := workspaceText("insert", s.inspector.word("editor.graph.add"), "")
		add.Meaning = s.inspector.word("editor.graph.add.hint")
		out = append(out, add)
	}
	return out
}

func (s *mapInspectionEdits) campaignLocked() error {
	x := s.inspector
	if x.mod == nil || s.workspace == nil || s.workspace.Project == nil {
		return fmt.Errorf("%s", x.word("editor.graph.readonly"))
	}
	if err := x.modReadOnly(); err != nil {
		return err
	}
	if x.installFlow() == nil || x.graphUnread() {
		return fmt.Errorf("%s", x.word("editor.graph.unread"))
	}
	return nil
}

func (s *mapInspectionEdits) setCampaignField(scope, key, value string) (*ui.InspectionDocument, error) {
	x := s.inspector
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	if err := s.campaignLocked(); err != nil {
		return nil, err
	}
	n, err := strconv.Atoi(strings.TrimPrefix(scope, "campaign."))
	if err != nil {
		return nil, err
	}
	return s.changeWorkspace(func(w *editorWorkspaceState) error {
		selected, pack := w.Project.Selected, w.Package
		i, err := s.campaignRow(w, n)
		if err != nil {
			return err
		}
		w.Project.Selected = i
		w.Package = modPackageSpec(w.Project.Missions[i], pack)
		trial := *s
		trial.workspace = w
		switch {
		case key != "insert" && !campaignFieldKey(key):
			err = fmt.Errorf(x.word("editor.mod.err.packagefield"), key)
		case key == "insert" || key == "landing" || strings.HasPrefix(key, "party."):
			err = trial.editModControl(w, key, value)
		default:
			err = trial.editPackageField(w, key, value)
		}
		if err != nil {
			return err
		}
		trial.syncModDocument(w, w.Pages, w.Labels, false)
		edited := w.Package
		w.Project.Selected, w.Package = selected, pack
		if selected == i {
			w.Package = edited
		}
		flow, err := s.missionDraft(w)
		if err != nil {
			return err
		}
		for i, row := range w.Project.Missions {
			for _, admitted := range flow.missions.Missions {
				if row.Mod == admitted.Mod && row.Number == admitted.Number && row.RowID() == admitted.RowID() {
					w.Project.Missions[i] = mod.CloneMission(admitted)
					break
				}
			}
		}
		if selected >= 0 {
			w.Package = modPackageSpec(w.Project.Missions[selected], w.Package)
		}
		return nil
	})
}

func (s *mapInspectionEdits) insertCampaignMission(w *editorWorkspaceState, value string) error {
	number, file, _ := strings.Cut(strings.TrimSpace(value), " ")
	n, err := strconv.Atoi(number)
	if err != nil {
		return fmt.Errorf("%s", s.inspector.word("editor.graph.add.hint"))
	}
	current, err := s.missionDraft(w)
	if err != nil {
		return err
	}
	s.insertCampaignRow(w, current.campaign, n, strings.TrimSpace(file))
	return nil
}

func (s *mapInspectionEdits) insertCampaignRow(w *editorWorkspaceState, c Campaign, n int, file string) {
	row := mod.Mission{Mod: w.Package.ID, File: mod.MissionsFile, Kind: mod.MissionCampaign,
		Number: n, Key: s.campaignKey(w, n), Map: file, MainParty: true}
	if next := c.after(w.Package.Number, w.Package.Number, false).successor; next != 0 {
		row.AutoGetMission = &next
	}
	w.Project.Missions = append(w.Project.Missions, row)
	w.Package.AutoGetMission = &n
}

// missionDraft admits parsed project rows without publishing files or context.
func (s *mapInspectionEdits) missionDraft(w *editorWorkspaceState) (*campaignFlow, error) {
	x, d := s.inspector, w.Project
	src, err := mod.WriteMissions(d.Source, d.Original, d.Missions)
	if err != nil {
		return nil, err
	}
	look, err := mod.TextLookup(x.mod.dir, x.pageLanguage())
	if err != nil {
		return nil, err
	}
	parsed, err := mod.ParseMissions(w.Package.ID, mod.MissionsFile, src, look, x.pageLanguage())
	if err != nil {
		return nil, err
	}
	for _, m := range d.Missions {
		if m.ByFileName() {
			m = mod.CloneMission(m)
			m.MapBytes = nil
			parsed.Missions = append(parsed.Missions, m)
		}
	}
	if err := x.missionFiles(parsed.Missions, d.Files); err != nil {
		return nil, err
	}
	if x.mod.loaded == nil {
		return nil, fmt.Errorf("%s", x.word("editor.graph.unbuilt"))
	}
	result := *x.mod.loaded
	result.Missions = parsed
	f, _, err := x.admittedFork(result)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, fmt.Errorf("%s", x.word("editor.graph.unbuilt"))
	}
	return &campaignFlow{campaign: f.Campaign.Value(), missions: tableMissions(f.Table)}, nil
}
