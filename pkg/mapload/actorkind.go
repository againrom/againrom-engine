package mapload

import (
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/rules"
)

// RowActorKind is the kind word and main skill school the second game's
// constructor stores for an actor of a Units row (human false) or a Humans
// row. A Units row stores the low word of its parameter 55 unchanged; a
// Humans row stores PersonNPCWord of its parameter 24, none when it is -1
// (R2-ENGINE-525). A `_Hero` or `Start_` Humans row's main skill is its
// largest school, the first of equal ones (R2-ENGINE-319); any other row's
// is 0. The first game and a row outside its table hold 0 for both.
func RowActorKind(human bool, row int, t *Table) (kind uint16, main int32) {
	if t == nil || !t.ActorKindWords {
		return 0, 0
	}
	rows := t.units()
	if human {
		rows = t.humans()
	}
	if rows == nil || row < 0 || row >= rows.Len() {
		return 0, 0
	}
	params := rows.EntryParams(row)
	if !human {
		id, _ := data.UnitServerID(params)
		return uint16(id), 0
	}
	if id, ok := data.HumanServerID(params); ok && id != -1 {
		kind = PersonNPCWord(id)
	}
	name := rows.EntryName(row)
	if !strings.Contains(name, "_Hero") && !strings.Contains(name, "Start_") {
		return kind, 0
	}
	if d, err := data.NewHumanDef(name, params); err == nil {
		main = rules.MainSkill(d.Skill)
	}
	return kind, main
}

// partyActorKind is the kind word and main skill school of a second-game
// party member's actor: the identity a hero, a campaign hire or a companion
// holds (R2-ENGINE-319, R2-ENGINE-525), else the row's own word in b; and
// PartyMainSkill.
func partyActorKind(p PartyMember, b spawnBlock, t *Table) (uint16, int32) {
	if t == nil || !t.ActorKindWords {
		return 0, 0
	}
	kind := b.kindWord
	if id, ok := PartyIdentity(p, t); ok {
		kind = uint16(id)
	}
	return kind, PartyMainSkill(p)
}

// PartyMainSkill is a second-game party member's main skill school. A member
// an older SAVE carried without one keeps the rule that SAVE wrote: the
// hero's and a companion's largest school.
func PartyMainSkill(p PartyMember) int32 {
	if p.MainSkill == 0 && (p.StartingHero || p.Membership().Is(CompanionMember)) {
		return rules.MainSkill(p.Hero.Skill)
	}
	return p.MainSkill
}
