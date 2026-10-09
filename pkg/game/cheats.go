package game

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type runtimeSwitches struct {
	deterministicFrames bool
	chicken             bool
}

type cheatConsole struct {
	privilege   [50]byte
	knowledge   bool
	showMap     bool
	turnTrace   bool
	scriptTrace bool
	safe        bool
	difficulty  mapload.Difficulty
	failure     error
}

func (f *FrontEnd) SetChickenAtMissionStart(on bool) { f.runtime.chicken = on }
func (f *FrontEnd) ChickenAtMissionStart() bool      { return f != nil && f.runtime.chicken }

func (f *FrontEnd) chatCommand(line string) {
	if f != nil && f.live != nil {
		f.live.chatCommand(line)
	}
}

func (f *FrontEnd) debugLetter(letter byte) {
	if f != nil && f.live != nil {
		f.live.debugLetter(letter)
	}
}

func (mw *mapWorld) cheatHero() (sim.Entity, bool) {
	if mw == nil || mw.mission == nil || len(mw.mission.ids) == 0 {
		return sim.Entity{}, false
	}
	return mw.world.Entity(mw.mission.ids[0])
}

func (mw *mapWorld) cheatPlayerName(owner uint32) string {
	if mw.mission != nil && mw.mission.state != nil && mw.mission.state.Map != nil {
		groups := mw.mission.state.Map.Groups
		if owner > 0 && int(owner) <= len(groups) && groups[owner-1].Name != "" {
			return groups[owner-1].Name
		}
	}
	if owner == sim.SelfSlot && mw.mission != nil && len(mw.mission.party) != 0 {
		return mw.mission.party[0].Name
	}
	return strconv.FormatUint(uint64(owner), 10)
}

func (mw *mapWorld) cheatMessage(line string) {
	mw.view.PostMessage(line, ui.MessageWhite, 3*time.Second)
}

func (mw *mapWorld) cheatDisplayPlayerName(owner uint32) string {
	name := mw.cheatPlayerName(owner)
	if !utf8.ValidString(name) {
		return name
	}
	return EncodeInstallText(name, mw.view.TextSelector())
}

func (mw *mapWorld) cheatNotice(code int) {
	if mw.mission == nil || mw.mission.src == nil {
		return
	}
	raw, err := mw.mission.src.ReadFile(MainTextPath)
	if err != nil {
		return
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	index := 221 + 2*(code-5)
	if index < 0 || index+1 >= len(lines) {
		return
	}
	mw.cheatMessage(lines[index] + mw.cheatDisplayPlayerName(sim.SelfSlot) + lines[index+1])
}

func (mw *mapWorld) cheatAllowed() bool {
	if mw.cheats.privilege[sim.SelfSlot] > 50 {
		return true
	}
	mw.cheatNotice(6)
	return false
}

func cheatInteger(s string) int32 {
	s = strings.TrimLeft(s, " \t\r\n")
	end := 0
	if len(s) > 0 && (s[0] == '-' || s[0] == '+') {
		end++
	}
	start := end
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == start {
		return 0
	}
	v, err := strconv.ParseInt(s[:end], 10, 32)
	if err != nil {
		return 0
	}
	return int32(v)
}

func cheatCount(s string) (uint32, string) {
	if at := strings.IndexByte(s, ' '); at >= 0 {
		if n := cheatInteger(s[:at]); n > 0 {
			return uint32(n), s[at+1:]
		}
	}
	return 1, s
}

func (mw *mapWorld) chatCommand(line string) {
	if mw == nil || mw.world == nil || mw.view == nil || mw.mission == nil {
		return
	}
	if mw.mission.secondGame() {
		return
	}
	if !strings.HasPrefix(line, "#") {
		mw.cheatMessage(mw.cheatDisplayPlayerName(sim.SelfSlot) + ": " + EncodeInstallText(line, mw.view.TextSelector()))
		return
	}
	if mw.mission.state != nil && mw.mission.state.Map != nil && int32(mw.mission.state.Map.Meta.Word70) > 1 {
		return
	}
	hero, hasHero := mw.cheatHero()
	switch {
	case strings.HasPrefix(line, "#create "):
		if !mw.cheatAllowed() {
			return
		}
		if !hasHero || hero.Decay != 0 {
			mw.cheatNotice(6)
			return
		}
		count, name := cheatCount(line[len("#create "):])
		ok := false
		if name == "Gold" {
			ok = mw.world.CheatAddGold(hero.Owner, count)
		} else if item, found := mapload.CheatItem(EncodeInstallText(name, mw.view.TextSelector()), mw.mission.table); found {
			ok = mw.world.CheatAddItem(hero.ID, sim.StackItem(item, count))
		}
		if !ok {
			mw.cheatNotice(6)
			return
		}
		mw.cheatNotice(7)
	case strings.HasPrefix(line, "#modify "):
		mw.cheatModify(line[len("#modify "):], hero, hasHero)
	case strings.HasPrefix(line, "#summon "):
		mw.cheats.failure = nil
		if !mw.cheatAllowed() || !hasHero {
			return
		}
		arg := line[len("#summon "):]
		asHero := strings.HasPrefix(arg, "hero ")
		count := uint32(1)
		if asHero {
			arg = arg[len("hero "):]
		} else {
			count, arg = cheatCount(arg)
		}
		if count > 4096 {
			return
		}
		difficulty := mw.cheats.difficulty
		if difficulty == 0 {
			difficulty = mapload.DifficultyNormal
		}
		template, pack, worn, err := mapload.CheatActor(EncodeInstallText(arg, mw.view.TextSelector()), asHero, mw.mission.table, difficulty)
		if err != nil {
			mw.cheats.failure = err
			return
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
	case strings.HasPrefix(line, "#killall") || strings.HasPrefix(line, "#kill all"):
		if !mw.cheatAllowed() {
			return
		}
		relations := mw.world.Relations()
		for owner := uint32(0); owner < 50; owner++ {
			if relations.Hostile(owner, sim.SelfSlot) {
				mw.world.CheatKillPlayer(owner)
			}
		}
		mw.cheatNotice(7)
	case strings.HasPrefix(line, "#kill cheaters"):
		if !mw.cheatAllowed() {
			return
		}
		for owner := uint32(0); owner < 50; owner++ {
			if owner != sim.SelfSlot && mw.cheats.privilege[owner] > 50 {
				mw.cheats.privilege[owner] = 0
				mw.world.CheatKillPlayer(owner)
			}
		}
	case strings.HasPrefix(line, "#kill "):
		if !mw.cheatAllowed() {
			return
		}
		name := line[len("#kill "):]
		for owner := uint32(0); owner < 50; owner++ {
			if mw.cheatDisplayPlayerName(owner) == EncodeInstallText(name, mw.view.TextSelector()) {
				mw.world.CheatKillPlayer(owner)
				break
			}
		}
		mw.cheatNotice(7)
	case strings.HasPrefix(line, "#pickup all"):
		if !mw.cheatAllowed() || !hasHero {
			return
		}
		if err := mw.world.CheatPickupAll(hero.ID); err != nil {
			return
		}
		mw.cheatNotice(7)
		mw.cheatMessage("All sacks picked up")
	case strings.HasPrefix(line, "#show map"):
		if !mw.cheatAllowed() {
			return
		}
		mw.cheats.showMap = true
		for i := range mw.fog.explored {
			mw.fog.explored[i] = 1
		}
		mw.view.SetCheatMapReveal(true)
		mw.cheatNotice(7)
	case strings.HasPrefix(line, "#hide map"):
		if !mw.cheatAllowed() {
			return
		}
		mw.cheats.showMap = false
		mw.view.SetCheatMapReveal(false)
		mw.cheatNotice(7)
	case strings.HasPrefix(line, "#victory"):
		if !mw.cheatAllowed() {
			return
		}
		mw.mission.announced = true
		mw.mission.outcome = sim.OutcomeWon
		mw.showOutcome()
	case strings.HasPrefix(line, "#event "):
		mw.openDialogue(int(cheatInteger(line[len("#event "):])))
	case strings.HasPrefix(line, "#Chicken"):
		mw.cheats.privilege[sim.SelfSlot] = 255
		mw.cheatMessage("Player " + mw.cheatDisplayPlayerName(sim.SelfSlot) + " enable cheating.")
		mw.cheatNotice(5)
	default:
		return
	}
	mw.refreshPack()
	mw.push()
}

func (mw *mapWorld) rememberCheatActor(id sim.EntityID, name string) {
	manifest := cloneActorManifest(mw.mission.state.ActorManifest)
	if manifest == nil {
		manifest = &SnapshotActorManifest{Version: actorManifestVersion}
	}
	manifest.NativeNamesOnly = false
	manifest.Actors = append(manifest.Actors, SnapshotActor{ID: id, Name: name, Constructed: true})
	mw.mission.state.ActorManifest = manifest
	mw.installActorManifest(manifest)
	mw.rememberCheatCharacter(id, name)
}

func (mw *mapWorld) rememberCheatCharacter(id sim.EntityID, name string) {
	e, ok := mw.world.Entity(id)
	if !ok || e.ActorLoad.Source.Class == 0 {
		return
	}
	if mw.chars == nil {
		mw.chars = map[sim.EntityID]ui.UnitCharacter{}
	}
	ch := ui.UnitCharacter{Known: true, Name: name, UnitNameIndex: int(e.TypeID), Band: ui.CharacterBandCreature}
	if e.Humanoid {
		ch.Band = ui.CharacterBandPerson
		if p, err := mapload.SourceActorPerson(e, name, mw.mission.table); err == nil {
			ch.Mage = p.Mage
			if mw.figures == nil {
				mw.figures = make(map[sim.EntityID]figureID)
			}
			mw.figures[id] = rosterFigureID(p, mw.world, id)
		}
	}
	mw.chars[id] = ch
	mw.refreshSourceCharacter(e)
	if !e.Humanoid {
		ch = mw.chars[id]
		for i, r := range e.Resistance {
			ch.Skills[i+1] = int(r)
		}
		mw.chars[id] = ch
	}
}

func (mw *mapWorld) cheatModify(arg string, hero sim.Entity, hasHero bool) {
	army := strings.HasPrefix(arg, "army")
	if !army && !strings.HasPrefix(arg, "self") {
		return
	}
	arg = strings.TrimLeft(arg[4:], " ")
	switch {
	case strings.HasPrefix(arg, "+god"):
		if army {
			for _, e := range mw.world.Entities() {
				if e.Owner == sim.SelfSlot {
					mw.world.CheatGod(e.ID)
				}
			}
		} else if hasHero {
			mw.world.CheatGod(hero.ID)
		}
	case strings.HasPrefix(arg, "+spell "):
		id := cheatInteger(arg[len("+spell "):])
		if !army && hasHero && id > 0 && mw.mission.table != nil && mw.mission.table.Spells != nil && int(id) < mw.mission.table.Spells.Len() {
			mw.world.CheatSpell(hero.ID, uint16(id))
		}
	case strings.HasPrefix(arg, "+spells"):
		if !army && hasHero {
			for id := uint16(1); id <= 28; id++ {
				mw.world.CheatSpell(hero.ID, id)
			}
		}
	case strings.HasPrefix(arg, "+knowledge"):
		mw.cheats.knowledge = mw.cheats.privilege[sim.SelfSlot] > 10
	default:
		return
	}
	mw.cheatNotice(7)
}

func (mw *mapWorld) cardKnowledge(e sim.Entity) int {
	if mw.cheats.knowledge {
		return 15
	}
	return mw.world.KnowledgeLevel(e)
}

func (mw *mapWorld) debugLetter(letter byte) {
	if mw == nil || mw.world == nil || mw.mission == nil || mw.mission.secondGame() || mw.cheats.privilege[sim.SelfSlot] <= 50 {
		return
	}
	toggle := func(name string, on bool) {
		state := "off"
		if on {
			state = "on"
		}
		mw.cheatMessage(name + " turned " + state + ".")
	}
	switch letter {
	case 'D':
		mw.cheats.turnTrace = !mw.cheats.turnTrace
		toggle("Turn tracing", mw.cheats.turnTrace)
	case 'T':
		mw.cheats.scriptTrace = !mw.cheats.scriptTrace
		toggle("Script tracing", mw.cheats.scriptTrace)
	case 'Q':
		mw.cheats.safe = !mw.cheats.safe
		mw.world.SetSafeMode(mw.cheats.safe)
		toggle("Safe mode", mw.cheats.safe)
	case 'H':
		for _, line := range []string{"<Alt-h> Help", "<Alt-q> Safe mode", "<Alt-t> Script tracing", "<Alt-i> Turn statistics", "<Alt-d> Turn tracing", "<Alt-u> Mission units stats"} {
			mw.cheatMessage(line)
		}
	case 'I':
		mw.cheatMessage("Last Turn Statistics:")
		mw.cheatMessage(fmt.Sprintf("Turn: %d; active: %d", mw.world.Tick(), len(mw.world.Entities())))
		mw.cheatMessage("Average Turn Statistics:")
	case 'U':
		mw.cheatMessage("Mission units stats:")
		mw.cheatUnitStatistics()
	}
}

func (mw *mapWorld) cheatUnitStatistics() {
	type groupStat struct {
		owner, group uint32
		count        int
		experience   uint64
	}
	var rows []groupStat
	for _, e := range mw.world.Entities() {
		index := slices.IndexFunc(rows, func(r groupStat) bool { return r.owner == e.Owner && r.group == e.Group })
		if index < 0 {
			index = len(rows)
			rows = append(rows, groupStat{owner: e.Owner, group: e.Group})
		}
		rows[index].count++
		if e.ActorLoad.Source.Class != 0 {
			rows[index].experience += uint64(e.SourceNow().Experience)
		}
	}
	slices.SortFunc(rows, func(a, b groupStat) int {
		if a.owner < b.owner {
			return -1
		}
		if a.owner > b.owner {
			return 1
		}
		if a.group < b.group {
			return -1
		}
		if a.group > b.group {
			return 1
		}
		return 0
	})
	for _, r := range rows {
		mw.cheatMessage(fmt.Sprintf("%s / group %d: %d; experience: %d", mw.cheatDisplayPlayerName(r.owner), r.group, r.count, r.experience))
	}
}
