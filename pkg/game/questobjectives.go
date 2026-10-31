package game

import (
	"fmt"
	"strings"

	"againrom/pkg/base"
)

// MissionObjectives reads the installed briefing for this mission. The owner
// identified m110's briefing on the original objectives panel. MISSION-TEXT-005
// establishes the mission text family; ALM.Description is a map label.
func MissionObjectives(src entrySource, mission int) string {
	return installText(src, briefingSource.at(mission), TextCode{}).text()
}

// MissionObjectivesFor is the objectives text of the mission as game g's files
// lay it out.
func MissionObjectivesFor(src entrySource, game base.Game, mission int) string {
	return filesOf(game).briefing(src, game.Edition(), mission)
}

// secondGameBriefing is the briefing section of the second game's mission text
// file.
func secondGameBriefing(src entrySource, e base.Edition, mission int) string {
	briefing, _ := installText(src, missionTextSource.at(mission), InstallTextCode(src, e)).sectionText("briefing")
	return briefing
}

func secondGameObjectiveLabels(src entrySource, code TextCode, mission int) []string {
	text := installText(src, missionTextSource.at(mission), code)
	if !text.present() {
		return nil
	}
	var labels []string
	for i := 0; i < 1024-752; i++ {
		label, ok := text.sectionText(fmt.Sprintf("subobjective%d", i))
		if !ok {
			break
		}
		labels = append(labels, label)
	}
	return labels
}

// missionObjectives is the objectives panel text: the briefing, extended by
// the campaign service where its game keeps per-objective state.
func (mw *mapWorld) missionObjectives(briefing string) string {
	return mw.mission.campaign().objectives(mw, briefing)
}

// secondGameObjectivePanel adds each opened sub-objective of the second
// game's scenario bank, marked done or failed, below the briefing.
func (mw *mapWorld) secondGameObjectivePanel(briefing string) string {
	var rows []string
	if briefing != "" {
		rows = append(rows, briefing, "")
	}
	for i, label := range mw.mission.objectiveLabels {
		value, ok := mw.world.ROM2ScenarioValue(int32(752 + i))
		if !ok || value == 0 {
			continue
		}
		mark := "[ ]"
		switch {
		case value&2 != 0:
			mark = "[+]"
		case value&4 != 0:
			mark = "[-]"
		}
		rows = append(rows, mark+" "+label)
	}
	return strings.TrimSpace(strings.Join(rows, "\n"))
}
