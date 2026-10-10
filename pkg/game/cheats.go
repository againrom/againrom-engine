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
	// campaign: the town the mission opened from holds a destinations
	// campaign; the second game's adapter admits cheat commands only there.
	campaign bool
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
	mw.cheatReply(code, sim.SelfSlot)
}

// cheatReply posts reply code for the Player in owner's slot: that Player's
// name between the two main.txt lines the code selects (5 the unlock, 6 a
// refusal, 7 a success), shown for the adapter's reply time.
func (mw *mapWorld) cheatReply(code int, owner uint32) {
	if code == 0 || mw.mission == nil || mw.mission.src == nil {
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
	mw.view.PostMessage(lines[index]+mw.cheatDisplayPlayerName(owner)+lines[index+1], ui.MessageWhite, mw.chatGame().replyTime)
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

func (mw *mapWorld) cardKnowledge(e sim.Entity) int {
	if mw.cheats.knowledge {
		return 15
	}
	return mw.world.KnowledgeLevel(e)
}

func (mw *mapWorld) cheatTurnStatistics() {
	mw.cheatMessage("Last Turn Statistics:")
	mw.cheatMessage(fmt.Sprintf("Turn: %d; active: %d", mw.world.Tick(), len(mw.world.Entities())))
	mw.cheatMessage("Average Turn Statistics:")
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
