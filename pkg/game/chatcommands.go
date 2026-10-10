package game

import (
	"strings"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// chatEffect names one operation a command row runs. An operation both games
// reach is one value; what differs between the games is a field of the row.
type chatEffect uint8

const (
	// chatNoFacility: the command exists in the table but reaches a facility
	// the engine does not have, so it changes nothing.
	chatNoFacility chatEffect = iota
	chatCreate
	chatModify
	chatSummon
	chatKillHostile
	chatKillCheaters
	chatKillNamed
	chatPickupAll
	chatShowMap
	chatHideMap
	chatVictory
	// chatEventDialogue opens the event's dialogue directly; chatEventMessage
	// raises the event as the mission script would, through the campaign
	// service's message route.
	chatEventDialogue
	chatEventMessage
	chatUnlock
)

// chatGate is what a matched row requires before its effect runs.
type chatGate uint8

const (
	// chatAdmitted: every line the adapter admitted to the table.
	chatAdmitted chatGate = iota
	// chatUnlocked: an unlocked Player; otherwise the row's refusal reply.
	chatUnlocked
	// chatHost: a line the adapter admitted to the host commands only.
	chatHost
)

// chatCommandRow is one command of a game's table. The parser tests rows in
// table order and runs the first whose text is a prefix of the line.
type chatCommandRow struct {
	text   string
	effect chatEffect
	gate   chatGate
	// refuse is the reply a refused gate sends; reply is the reply the effect
	// sends on success. Zero sends none.
	refuse, reply int
	// line is a console line the effect posts beside its reply, formatted with
	// the Player's name where it holds %s.
	line string
	// foldGold compares the create argument with the gold word ignoring case.
	foldGold bool
	// countFirst reads a summon count before the hero token.
	countFirst bool
	// replyTarget sends the reply for the Player the command named, and none
	// when it named no Player.
	replyTarget bool
	// modifiers are the #modify arms, tested in order after the target word.
	modifiers []chatModifierRow
}

// chatModifier names one #modify arm operation.
type chatModifier uint8

const (
	chatModifyNothing chatModifier = iota
	chatModifyGod
	chatModifySpell
	chatModifyAllSpells
	chatModifyKnowledge
)

type chatModifierRow struct {
	text   string
	op     chatModifier
	reply  int
	spells uint16
	// book: the spell arms act only on a hero holding a spellbook.
	book bool
}

// chatAdmission is the adapter's answer for one command line.
type chatAdmission uint8

const (
	chatEnds chatAdmission = iota
	chatCommands
	chatHostCommands
)

// chatCommand is the one chat command parser. A line not starting with # is
// the adapter's ordinary chat. A command line goes through the adapter's
// admission, then to the first table row whose text it starts with.
func (mw *mapWorld) chatCommand(line string) {
	if mw == nil || mw.world == nil || mw.view == nil || mw.mission == nil {
		return
	}
	g := mw.chatGame()
	if !strings.HasPrefix(line, "#") {
		if g.echo {
			mw.cheatMessage(mw.cheatDisplayPlayerName(sim.SelfSlot) + ": " + EncodeInstallText(line, mw.view.TextSelector()))
		}
		return
	}
	admission := g.admit(mw, line)
	if admission == chatEnds {
		return
	}
	for _, row := range g.commands {
		if (row.gate == chatHost) != (admission == chatHostCommands) || !strings.HasPrefix(line, row.text) {
			continue
		}
		mw.cheats.failure = nil
		if row.gate == chatUnlocked && mw.cheats.privilege[sim.SelfSlot] <= 50 {
			mw.cheatNotice(row.refuse)
			return
		}
		if mw.runChatCommand(row, line[len(row.text):]) {
			mw.refreshPack()
			mw.push()
		}
		return
	}
}

// runChatCommand runs row's effect on the argument after its text and reports
// whether the mission view is refreshed afterwards.
func (mw *mapWorld) runChatCommand(row chatCommandRow, arg string) bool {
	hero, hasHero := mw.cheatHero()
	switch row.effect {
	case chatCreate:
		if !hasHero || hero.Decay != 0 {
			mw.cheatNotice(6)
			return false
		}
		count, name := cheatCount(arg)
		ok := false
		if name == "Gold" || row.foldGold && strings.EqualFold(name, "Gold") {
			ok = mw.world.CheatAddGold(hero.Owner, count)
		} else if item, found := mapload.CheatItem(EncodeInstallText(name, mw.view.TextSelector()), mw.mission.table); found {
			ok = mw.world.CheatAddItem(hero.ID, sim.StackItem(item, count))
		}
		if !ok {
			mw.cheatNotice(6)
			return false
		}
		mw.cheatNotice(row.reply)
	case chatModify:
		mw.chatModify(row, arg, hero, hasHero)
	case chatSummon:
		return hasHero && mw.chatSummon(row, arg, hero)
	case chatKillHostile:
		relations := mw.world.Relations()
		for owner := uint32(0); owner < 50; owner++ {
			if relations.Hostile(owner, sim.SelfSlot) {
				mw.world.CheatKillPlayer(owner)
			}
		}
		mw.cheatNotice(row.reply)
	case chatKillCheaters:
		for owner := uint32(0); owner < 50; owner++ {
			if owner != sim.SelfSlot && mw.cheats.privilege[owner] > 50 {
				mw.cheats.privilege[owner] = 0
				mw.world.CheatKillPlayer(owner)
			}
		}
		mw.cheatNotice(row.reply)
	case chatKillNamed:
		target, found := uint32(0), false
		for owner := uint32(0); owner < 50; owner++ {
			if mw.cheatDisplayPlayerName(owner) == EncodeInstallText(arg, mw.view.TextSelector()) {
				mw.world.CheatKillPlayer(owner)
				target, found = owner, true
				break
			}
		}
		switch {
		case !row.replyTarget:
			mw.cheatNotice(row.reply)
		case found:
			mw.cheatReply(row.reply, target)
		}
	case chatPickupAll:
		if !hasHero {
			return false
		}
		if err := mw.world.CheatPickupAll(hero.ID); err != nil {
			return false
		}
		mw.cheatNotice(row.reply)
		if row.line != "" {
			mw.cheatMessage(row.line)
		}
	case chatShowMap:
		mw.cheats.showMap = true
		for i := range mw.fog.explored {
			mw.fog.explored[i] = 1
		}
		mw.view.SetCheatMapReveal(true)
		mw.cheatNotice(row.reply)
	case chatHideMap:
		mw.cheats.showMap = false
		mw.view.SetCheatMapReveal(false)
		mw.cheatNotice(row.reply)
	case chatVictory:
		mw.mission.announced = true
		mw.mission.outcome = sim.OutcomeWon
		mw.showOutcome()
	case chatEventDialogue:
		mw.openDialogue(int(cheatInteger(arg)))
	case chatEventMessage:
		mw.observeScriptMessages([]int32{cheatInteger(arg)})
	case chatUnlock:
		mw.chatUnlock(row.line, row.reply)
	default:
		return false
	}
	return true
}

// chatUnlock sets the Player's cheat state, the one state every unlocked
// command and debug letter tests, posts line when the game has one and sends
// reply.
func (mw *mapWorld) chatUnlock(line string, reply int) {
	mw.cheats.privilege[sim.SelfSlot] = 255
	if line != "" {
		mw.cheatMessage(strings.Replace(line, "%s", mw.cheatDisplayPlayerName(sim.SelfSlot), 1))
	}
	mw.cheatNotice(reply)
}

func (mw *mapWorld) chatSummon(row chatCommandRow, arg string, hero sim.Entity) bool {
	asHero := false
	count := uint32(1)
	if row.countFirst {
		count, arg = cheatCount(arg)
		if strings.HasPrefix(arg, "hero ") {
			asHero, arg = true, arg[len("hero "):]
		}
	} else if strings.HasPrefix(arg, "hero ") {
		asHero, arg = true, arg[len("hero "):]
	} else {
		count, arg = cheatCount(arg)
	}
	if count > 4096 {
		return false
	}
	difficulty := mw.cheats.difficulty
	if difficulty == 0 {
		difficulty = mapload.DifficultyNormal
	}
	template, pack, worn, err := mapload.CheatActor(EncodeInstallText(arg, mw.view.TextSelector()), asHero, mw.mission.table, difficulty)
	if err != nil {
		mw.cheats.failure = err
		return false
	}
	template.Owner = hero.Owner
	template.ActorLoad.Source.HasOwner = true
	for range count {
		id, err := mw.world.CheatSummon(template, pack, worn, hero.X, hero.Y)
		if err != nil {
			mw.cheats.failure = err
			break
		}
		mw.rememberCheatActor(id, arg)
	}
	mw.cheatNotice(row.reply)
	return true
}

func (mw *mapWorld) chatModify(row chatCommandRow, arg string, hero sim.Entity, hasHero bool) {
	army := strings.HasPrefix(arg, "army")
	if !army && !strings.HasPrefix(arg, "self") {
		return
	}
	arg = strings.TrimLeft(arg[4:], " ")
	for _, m := range row.modifiers {
		if !strings.HasPrefix(arg, m.text) {
			continue
		}
		switch m.op {
		case chatModifyGod:
			if army {
				for _, e := range mw.world.Entities() {
					if e.Owner == sim.SelfSlot {
						mw.world.CheatGod(e.ID)
					}
				}
			} else if hasHero {
				mw.world.CheatGod(hero.ID)
			}
		case chatModifySpell:
			id := cheatInteger(arg[len(m.text):])
			if m.book && !mw.cheatHeroBook(hero) {
				break
			}
			if !army && hasHero && id > 0 && mw.mission.table != nil && mw.mission.table.Spells != nil && int(id) < mw.mission.table.Spells.Len() {
				mw.world.CheatSpell(hero.ID, uint16(id))
			}
		case chatModifyAllSpells:
			if !army && hasHero && (!m.book || mw.cheatHeroBook(hero)) {
				for id := uint16(1); id <= m.spells; id++ {
					mw.world.CheatSpell(hero.ID, id)
				}
			}
		case chatModifyKnowledge:
			mw.cheats.knowledge = mw.cheats.privilege[sim.SelfSlot] > 10
		}
		mw.cheatNotice(m.reply)
		return
	}
}

// cheatHeroBook reports whether the hero holds a spellbook: a book the
// engine's book model carries, or the starting hero built as a mage, whose
// book the engine's second-game hero does not model (DIV-2777).
func (mw *mapWorld) cheatHeroBook(hero sim.Entity) bool {
	if hero.Book.WirePresent(hero.KnownSpells) {
		return true
	}
	return len(mw.mission.party) != 0 && mw.mission.ids[0] == hero.ID && mw.mission.party[0].Mage
}

// launchUnlock is the starter checkbox and -chicken: it submits the
// adapter's unlock line at a fresh mission start, and at a LOAD into a mission
// when the adapter says so.
func (mw *mapWorld) launchUnlock(fresh bool) {
	g := mw.chatGame()
	if fresh || g.unlockOnLoad {
		mw.chatCommand(g.launchLine)
	}
}

// debugLetter is Alt plus a letter: the adapter's key table, for an unlocked
// Player only.
func (mw *mapWorld) debugLetter(letter byte) {
	if mw == nil || mw.world == nil || mw.mission == nil || mw.cheats.privilege[sim.SelfSlot] <= 50 {
		return
	}
	g := mw.chatGame()
	key, ok := g.keys[letter]
	if !ok {
		return
	}
	toggle := func(name string, on bool) {
		if !g.echo {
			return
		}
		state := "off"
		if on {
			state = "on"
		}
		mw.cheatMessage(name + " turned " + state + ".")
	}
	switch key {
	case chatKeyTurnTrace:
		mw.cheats.turnTrace = !mw.cheats.turnTrace
		toggle("Turn tracing", mw.cheats.turnTrace)
	case chatKeyScriptTrace:
		mw.cheats.scriptTrace = !mw.cheats.scriptTrace
		toggle("Script tracing", mw.cheats.scriptTrace)
	case chatKeySafeMode:
		mw.cheats.safe = !mw.cheats.safe
		mw.world.SetSafeMode(mw.cheats.safe)
		toggle("Safe mode", mw.cheats.safe)
	case chatKeyHelp:
		for _, line := range g.help {
			mw.cheatMessage(line)
		}
	case chatKeyLastTurn:
		mw.cheatTurnStatistics()
	case chatKeyUnits:
		mw.cheatMessage("Mission units stats:")
		mw.cheatUnitStatistics()
	}
}
