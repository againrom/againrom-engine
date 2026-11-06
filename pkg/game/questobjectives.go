package game

import (
	"fmt"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
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

// objectivesProfile is a game's row of the objectives panel and outcome text:
// subobjectives reads the mission text's sub-objective labels and shows each
// opened one below the briefing, lossEntries shows the failure entry of a
// lost mission's latched reason in place of the generic body, and npcKeys
// reads the placed speakers' keys. The empty row is the briefing and the
// body as given.
type objectivesProfile struct {
	subobjectives bool
	lossEntries   bool
	npcKeys       bool
}

// loadMissionText reads the per-mission text the game's objectives row names.
func loadMissionText(mw *mapWorld, src entrySource, ms *Mission, t *mapload.Table) {
	code := InstallTextCode(src, tableEdition(t))
	mw.authoredNames = missionUnitNames(src, code, ms.Number)
	row := mw.mission.files().objectives
	if row == (objectivesProfile{}) {
		return
	}
	if row.subobjectives {
		mw.mission.objectiveLabels = secondGameObjectiveLabels(src, code, ms.Number)
	}
	if row.npcKeys {
		mw.mission.npcKeys = secondGameNPCKeys(ms, t)
	}
	if row.lossEntries {
		mw.mission.failureText = secondGameFailureText(src, code, ms.Number)
	}
}

// outcomeText is the outcome notice's text: the body as given, or for a lost
// mission the failure entry of the latched loss reason - 2 (R2-ENGINE-629);
// a reason outside the entries keeps the generic body (DIV-4223).
func outcomeText(mw *mapWorld, body string) string {
	m := mw.mission
	if !m.files().objectives.lossEntries {
		return body
	}
	if r := int64(m.lossReason) - 2; m.outcome == sim.OutcomeLost && r >= 0 && r < int64(len(m.failureText)) && m.failureText[r] != "" {
		body = m.failureText[r]
	}
	return body
}

// missionObjectives is the one producer of the objectives window's rows:
// for each cached text i, in cache order, whose bank slot 752+i is nonzero,
// the text and the mark its state selects (R2-ENGINE-888). A game whose row
// reads no sub-objectives shows none.
func (mw *mapWorld) missionObjectives() []ui.QuestObjective {
	if !mw.mission.files().objectives.subobjectives {
		return nil
	}
	var rows []ui.QuestObjective
	for i, label := range mw.mission.objectiveLabels {
		value, ok := mw.world.ROM2ScenarioValue(int32(sim.ObjectiveSlotBase + i))
		if !ok || value == 0 {
			continue
		}
		rows = append(rows, ui.QuestObjective{Text: label, Mark: objectiveMark(value)})
	}
	return rows
}

// objectiveMark is the subobj.256 frame an objective state draws: bit 2
// frame 1, else bit 4 frame 2, else bit 1 frame 0, else none (R2-ENGINE-888,
// R2-ENGINE-889).
func objectiveMark(state int32) int {
	switch {
	case state&2 != 0:
		return 1
	case state&4 != 0:
		return 2
	case state&1 != 0:
		return 0
	}
	return ui.NoObjectiveMark
}
