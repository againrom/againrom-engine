package game

import (
	"fmt"
	"strings"

	"againrom/pkg/base"
	"againrom/pkg/vfs"
)

// MissionObjectives reads the installed briefing for this mission. The owner
// identified m110's briefing on the original objectives panel. MISSION-TEXT-005
// establishes the mission text family; ALM.Description is a map label.
func MissionObjectives(src entrySource, mission int) string {
	if src == nil || mission <= 0 {
		return ""
	}
	raw, err := src.ReadFile(fmt.Sprintf("%stext/battle/m%d/briefing.txt", mainPrefix, mission))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(vfs.DecodeText(raw))
}

func MissionObjectivesFor(src entrySource, game base.Game, mission int) string {
	if !game.Edition().SecondMissionText {
		return MissionObjectives(src, mission)
	}
	payload, ok := readSecondGameMissionText(src, InstallTextCode(src, game.Edition()), mission)
	if !ok {
		return ""
	}
	return strings.TrimSpace(vfs.DecodeText([]byte(secondGameTextSection(payload, "briefing"))))
}

func secondGameObjectiveLabels(src entrySource, code TextCode, mission int) []string {
	payload, ok := readSecondGameMissionText(src, code, mission)
	if !ok {
		return nil
	}
	var labels []string
	for i := 0; i < 1024-752; i++ {
		label := secondGameTextSection(payload, fmt.Sprintf("subobjective%d", i))
		if label == "" {
			break
		}
		labels = append(labels, strings.TrimSpace(vfs.DecodeText([]byte(label))))
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
