package ui

// GameMenuRows are a game's menu root rows per place; nil is the first game's.
type GameMenuRows struct {
	Mission, Town []GameMenuRowSpec
}

// GameMenuRowSpec is one root row: label word, key, action, and when it is
// listed and enabled.
type GameMenuRowSpec struct {
	word     menuWord
	fallback menuKey
	action   gameMenuAction
	shown    menuRowCondition
	enabled  menuRowCondition
}

// menuKey is a row's key letter; with menuKeyFixed the label's `~` does not
// replace it.
type menuKey uint16

const menuKeyFixed menuKey = 0x100

type menuWord int

const (
	menuWordSave menuWord = iota
	menuWordLoad
	menuWordDiplomacy
	menuWordGameOptions
	menuWordSoundOptions
	menuWordQuestObjectives
	menuWordEndQuest
	menuWordAbort
	menuWordReturn
)

func (w Words) menuWord(k menuWord) string {
	switch k {
	case menuWordSave:
		return w.MenuSave
	case menuWordLoad:
		return w.MenuLoad
	case menuWordDiplomacy:
		return w.MenuDiplomacy
	case menuWordGameOptions:
		return w.MenuGameOptions
	case menuWordSoundOptions:
		return w.MenuSoundOptions
	case menuWordQuestObjectives:
		return w.MenuQuestObjectives
	case menuWordEndQuest:
		return w.MenuEndQuest
	case menuWordAbort:
		return w.MenuAbort
	}
	return w.MenuReturn
}

type menuRowCondition int

const (
	menuRowAlways menuRowCondition = iota
	menuRowCampaign
	menuRowStandalone
	menuRowCampaignSave
	menuRowCanLoad
	menuRowCanSound
	menuRowTownSave
)

type gameMenuRowState struct {
	campaign, canSave, canLoad, canSound bool
	// townSaveClosed is the town Save row's save point closed.
	townSaveClosed bool
}

func (s gameMenuRowState) holds(c menuRowCondition) bool {
	switch c {
	case menuRowCampaign:
		return s.campaign
	case menuRowStandalone:
		return !s.campaign
	case menuRowCampaignSave:
		return s.campaign && s.canSave
	case menuRowCanLoad:
		return s.canLoad
	case menuRowCanSound:
		return s.canSound
	case menuRowTownSave:
		return !s.townSaveClosed
	}
	return true
}

// firstGameMenuRows: the mission (MENU-ITEM-011) shows Load or Diplomacy by
// the campaign boolean; quest objectives keeps the inert immediate M behind
// its `~Q`, and the town's Abort Game the immediate E (MENU-KEY-013). The
// town (MENU-ITEM-012) puts Save before Load.
var firstGameMenuRows = GameMenuRows{
	Mission: []GameMenuRowSpec{
		{menuWordSave, 'S', gameMenuSave, menuRowAlways, menuRowCampaignSave},
		{menuWordLoad, 'L', gameMenuLoad, menuRowCampaign, menuRowCanLoad},
		{menuWordDiplomacy, 'D', gameMenuDiplomacy, menuRowStandalone, menuRowAlways},
		{menuWordGameOptions, 'O', gameMenuGameOptions, menuRowAlways, menuRowAlways},
		{menuWordSoundOptions, 'N', gameMenuSoundOptions, menuRowAlways, menuRowCanSound},
		{menuWordQuestObjectives, 'M', gameMenuQuestObjectives, menuRowAlways, menuRowCampaign},
		{menuWordEndQuest, 'E', gameMenuEndQuest, menuRowAlways, menuRowAlways},
		{menuWordReturn, 'R', gameMenuReturn, menuRowAlways, menuRowAlways},
	},
	Town: []GameMenuRowSpec{
		{menuWordSave, 'S', gameMenuSave, menuRowAlways, menuRowTownSave},
		{menuWordLoad, 'L', gameMenuLoad, menuRowAlways, menuRowAlways},
		{menuWordGameOptions, 'O', gameMenuGameOptions, menuRowAlways, menuRowAlways},
		{menuWordSoundOptions, 'N', gameMenuSoundOptions, menuRowAlways, menuRowAlways},
		{menuWordAbort, 'E', gameMenuAbortGame, menuRowAlways, menuRowAlways},
		{menuWordReturn, 'R', gameMenuReturn, menuRowAlways, menuRowAlways},
	},
}

// SecondGameMenuRows: the town statue's popup is the five buttons of
// R2-ENGINE-441 and R2-ENGINE-569, dialogs.txt rows 34, 35, 37, 77 and 40
// with the keys S, L, N, E and R read from a data block, not from the row
// text; the mission keeps the first game's rows.
var SecondGameMenuRows = GameMenuRows{
	Town: []GameMenuRowSpec{
		{menuWordSave, menuKeyFixed | 'S', gameMenuSave, menuRowAlways, menuRowTownSave},
		{menuWordLoad, menuKeyFixed | 'L', gameMenuLoad, menuRowAlways, menuRowAlways},
		{menuWordSoundOptions, menuKeyFixed | 'N', gameMenuSoundOptions, menuRowAlways, menuRowAlways},
		{menuWordAbort, menuKeyFixed | 'E', gameMenuAbortGame, menuRowAlways, menuRowAlways},
		{menuWordReturn, menuKeyFixed | 'R', gameMenuReturn, menuRowAlways, menuRowAlways},
	},
}

func (p GameMenuProfile) rowsAt(s gameMenuSurface) []GameMenuRowSpec {
	rows, first := p.Rows.Mission, firstGameMenuRows.Mission
	if s == gameMenuTown {
		rows, first = p.Rows.Town, firstGameMenuRows.Town
	}
	if rows == nil {
		return first
	}
	return rows
}

// gameMenuRootRows builds one place's root rows from its data.
func gameMenuRootRows(specs []GameMenuRowSpec, w Words, s gameMenuRowState) []gameMenuRow {
	rows := make([]gameMenuRow, 0, len(specs))
	for _, r := range specs {
		if !s.holds(r.shown) {
			continue
		}
		rows = append(rows, gameMenuRow{Label: w.menuWord(r.word), Fallback: byte(r.fallback), FixedKey: r.fallback&menuKeyFixed != 0,
			Action: r.action, Enabled: s.holds(r.enabled)})
	}
	return rows
}
