package game

import (
	"strings"
	"time"

	"againrom/pkg/sim"
)

// chatKey names one debug-letter operation.
type chatKey uint8

const (
	chatKeyTurnTrace chatKey = iota + 1
	chatKeyScriptTrace
	chatKeySafeMode
	chatKeyHelp
	chatKeyLastTurn
	chatKeyUnits
)

// chatAdapter is one game's chat command adapter: its command table, the rule
// that admits a command line before the table is searched, its debug letters
// and the rules the parser defers to. The parser holds no rule of either game.
type chatAdapter struct {
	commands []chatCommandRow
	admit    func(mw *mapWorld, line string) chatAdmission
	keys     map[byte]chatKey
	help     []string
	// echo shows an ordinary line as chat and prints the debug letters'
	// state lines.
	echo      bool
	replyTime time.Duration
	// launchLine is what the launch flag submits; unlockOnLoad also submits it
	// at a LOAD into a mission.
	launchLine   string
	unlockOnLoad bool
}

// chatGame is the adapter of the mission's campaign service.
func (mw *mapWorld) chatGame() *chatAdapter { return mw.mission.campaign().chat() }

// The first game's commands: MENU-099 through MENU-114, MENU-062 for the debug
// letters.
var firstChat = chatAdapter{
	commands: []chatCommandRow{
		{text: "#create ", effect: chatCreate, gate: chatUnlocked, refuse: 6, reply: 7},
		{text: "#modify ", effect: chatModify, modifiers: []chatModifierRow{
			{text: "+god", op: chatModifyGod, reply: 7},
			{text: "+spell ", op: chatModifySpell, reply: 7},
			{text: "+spells", op: chatModifyAllSpells, reply: 7, spells: 28},
			{text: "+knowledge", op: chatModifyKnowledge, reply: 7},
		}},
		{text: "#summon ", effect: chatSummon, gate: chatUnlocked, refuse: 6},
		{text: "#killall", effect: chatKillHostile, gate: chatUnlocked, refuse: 6, reply: 7},
		{text: "#kill all", effect: chatKillHostile, gate: chatUnlocked, refuse: 6, reply: 7},
		{text: "#kill cheaters", effect: chatKillCheaters, gate: chatUnlocked, refuse: 6},
		{text: "#kill ", effect: chatKillNamed, gate: chatUnlocked, refuse: 6, reply: 7},
		{text: "#pickup all", effect: chatPickupAll, gate: chatUnlocked, refuse: 6, reply: 7, line: "All sacks picked up"},
		{text: "#show map", effect: chatShowMap, gate: chatUnlocked, refuse: 6, reply: 7},
		{text: "#hide map", effect: chatHideMap, gate: chatUnlocked, refuse: 6, reply: 7},
		{text: "#victory", effect: chatVictory, gate: chatUnlocked, refuse: 6},
		{text: "#event ", effect: chatEventDialogue},
		{text: "#Chicken", effect: chatUnlock, reply: 5, line: "Player %s enable cheating."},
	},
	keys: map[byte]chatKey{'D': chatKeyTurnTrace, 'T': chatKeyScriptTrace, 'Q': chatKeySafeMode, 'H': chatKeyHelp, 'I': chatKeyLastTurn, 'U': chatKeyUnits},
	help: []string{"<Alt-h> Help", "<Alt-q> Safe mode", "<Alt-t> Script tracing", "<Alt-i> Turn statistics", "<Alt-d> Turn tracing", "<Alt-u> Mission units stats"},
	echo: true, replyTime: 3 * time.Second,
	launchLine: "#Chicken",
}

// secondUnlock is the prefix R2-ENGINE-297's check accepts.
const secondUnlock = "##Cowar"

// The second game's commands: R2-ENGINE-295 (texts and order), R2-ENGINE-296
// (campaign only; host and latency commands outside it), R2-ENGINE-297 (the
// unlock gate), R2-ENGINE-298 (replies), R2-ENGINE-299 to R2-ENGINE-303 (the
// commands) and R2-ENGINE-304 (debug letters). DIV-2774 to DIV-2777.
var secondChat = chatAdapter{
	commands: []chatCommandRow{
		{text: "#kick ", gate: chatHost},
		{text: "#locate ", gate: chatHost},
		{text: "#set latency ", gate: chatHost},
		{text: "#show latency", gate: chatHost},
		{text: "#create ", effect: chatCreate, reply: 7, foldGold: true},
		{text: "#modify ", effect: chatModify, modifiers: []chatModifierRow{
			{text: "+god", op: chatModifyGod, reply: 7},
			{text: "+spell ", op: chatModifySpell, reply: 7, book: true},
			{text: "+spells", op: chatModifyAllSpells, reply: 7, spells: 29, book: true},
			{text: "+knowledge", op: chatModifyNothing},
		}},
		{text: "#summon ", effect: chatSummon, countFirst: true},
		{text: "#killall", effect: chatKillHostile, reply: 7},
		{text: "#kill all", effect: chatKillHostile, reply: 7},
		{text: "#kill cheaters", effect: chatKillCheaters},
		{text: "#kill ", effect: chatKillNamed, reply: 7, replyTarget: true},
		{text: "#pickup all", effect: chatPickupAll, reply: 7},
		{text: "#show map", effect: chatShowMap, reply: 7},
		{text: "#hide map", effect: chatHideMap, reply: 7},
		{text: "#victory", effect: chatVictory},
		{text: "#event ", effect: chatEventMessage},
	},
	keys:       map[byte]chatKey{'D': chatKeyTurnTrace, 'T': chatKeyScriptTrace, 'Q': chatKeySafeMode},
	replyTime:  5 * time.Second,
	launchLine: secondUnlock, unlockOnLoad: true,
}

// The admission rules reach the adapters again through the replies they send,
// so they are attached once the tables exist.
func init() {
	firstChat.admit = firstChatAdmit
	secondChat.admit = secondChatAdmit
}

// firstChatAdmit: a map declaring more than one participant refuses every
// command (MENU-100).
func firstChatAdmit(mw *mapWorld, _ string) chatAdmission {
	if mw.mission.state != nil && mw.mission.state.Map != nil && int32(mw.mission.state.Map.Meta.Word70) > 1 {
		return chatEnds
	}
	return chatCommands
}

// secondChatAdmit: outside the campaign only the host and latency commands
// are tested (R2-ENGINE-296); a locked Player's line can only unlock, and the
// unlock runs no command (R2-ENGINE-297).
func secondChatAdmit(mw *mapWorld, line string) chatAdmission {
	if !mw.cheats.campaign {
		return chatHostCommands
	}
	if mw.cheats.privilege[sim.SelfSlot] > 50 {
		return chatCommands
	}
	if strings.HasPrefix(line, secondUnlock) {
		mw.chatUnlock("", 5)
	}
	return chatEnds
}
