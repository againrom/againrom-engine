package game

import (
	"fmt"
	"slices"
	"strconv"

	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// companionsAddKey is the Companions page's adder.
const companionsAddKey = "newjoin"

// joinCampaign is the install's campaign, nil when the install has none.
func (x *MapInspector) joinCampaign() *Campaign {
	flow := x.installFlow()
	if flow == nil || len(flow.campaign.Chapters) == 0 {
		return nil
	}
	return &flow.campaign
}

// joinGrants are the companions the town of chapter ch grants.
func joinGrants(c *Campaign, ch int) []int {
	var out []int
	if c == nil {
		return out
	}
	for _, id := range c.Chapters[ch].AddHero {
		if c.townGrants(id) && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

// joinChapters are the chapters whose town grants a companion, ascending.
func joinChapters(c *Campaign) []int {
	var out []int
	if c == nil {
		return out
	}
	for ch := range c.Chapters {
		if len(joinGrants(c, ch)) > 0 {
			out = append(out, ch)
		}
	}
	slices.Sort(out)
	return out
}

// joinProblem is why the campaign cannot run condition j, in the page's
// words, or "" when it can or there is no campaign to ask.
func (x *MapInspector) joinProblem(j mod.CompanionJoin) string {
	c := x.joinCampaign()
	if c == nil || j.Off {
		return ""
	}
	if _, ok := c.Chapters[j.Chapter]; !ok {
		return fmt.Sprintf(x.word("editor.companions.err.chapter"), j.Chapter)
	}
	if !slices.Contains(joinGrants(c, j.Chapter), j.Companion) {
		return fmt.Sprintf(x.word("editor.companions.err.grant"), j.Chapter, j.Companion)
	}
	if !slices.Contains(c.chapter(j.Chapter).InnNPC, j.Talk) {
		return fmt.Sprintf(x.word("editor.companions.err.talk"), j.Chapter, j.Talk)
	}
	return ""
}

// companionsRows is the save gate of the companions file: a join condition
// the campaign cannot run is refused, naming the row.
var companionsRows = modRowsPage{save: "editor.companions.err.save", rel: mod.CompanionsFile,
	check: func(s *mapInspectionEdits, held mod.Data) error {
		x := s.inspector
		for _, j := range held.Companions.Joins {
			if p := x.joinProblem(j); p != "" {
				return fmt.Errorf(x.word("editor.companions.err.save"), s.joinName(j), p)
			}
		}
		return nil
	}}

// joinName names a condition by its companion and chapter.
func (s *mapInspectionEdits) joinName(j mod.CompanionJoin) string {
	return fmt.Sprintf(s.inspector.word("editor.companions.name"), s.characterLabel(j.Companion), j.Chapter)
}

// joinChoices are the choices of numbers vals, with cur when it is none of
// them.
func (x *MapInspector) joinChoices(vals []int, cur int, label func(int) string) []ui.InspectionChoice {
	if cur > 0 && !slices.Contains(vals, cur) {
		vals = append(slices.Clone(vals), cur)
	}
	out := make([]ui.InspectionChoice, len(vals))
	for i, v := range vals {
		out[i] = ui.InspectionChoice{Value: int64(v), Label: x.inspectionUTF8(label(v))}
	}
	return out
}

// companionsFields is W's Companions page: the Add button, then each
// condition's chapter, companion, tavern conversation and removal.
func (s *mapInspectionEdits) companionsFields() []ui.InspectionField {
	x := s.inspector
	if !s.modPageHolds() {
		return nil
	}
	project := s.workspace.Project
	if project == nil {
		project = x.mod.document()
	}
	held, err := s.heldRows(project, nil, mod.CompanionsFile)
	if err != nil {
		return []ui.InspectionField{x.pageLine("error", "", err.Error())}
	}
	out := []ui.InspectionField{{Key: companionsAddKey, Label: x.word("editor.companions.add"), Meaning: x.word("editor.companions.add.help"), Action: true, Column: 1}}
	c := x.joinCampaign()
	chapterLabel := func(v int) string { return fmt.Sprintf(x.word("editor.companions.chapter.value"), v) }
	number := func(key, label, hint string, v int, vals []int, name func(int) string) ui.InspectionField {
		var f ui.InspectionField
		if c != nil && len(vals) > 0 {
			f = ui.InspectionField{Key: key, Label: label, Value: int64(v), Choices: x.joinChoices(vals, v, name)}
		} else {
			f = workspaceText(key, label, strconv.Itoa(v))
		}
		f.Meaning = hint
		return f
	}
	for i, j := range held.Companions.Joins {
		key := "join." + strconv.Itoa(i) + "."
		start := len(out)
		if j.Off {
			out = append(out, x.pageLine(key+"state", "", x.word("editor.companions.isoff")))
		} else {
			talks := []int(nil)
			if c != nil {
				talks = c.chapter(j.Chapter).InnNPC
			}
			out = append(out,
				number(key+"chapter", x.word("editor.companions.chapter"), x.word("editor.companions.chapter.help"), j.Chapter, joinChapters(c), chapterLabel),
				number(key+"companion", x.word("editor.companions.companion"), x.word("editor.companions.companion.help"), j.Companion, joinGrants(c, j.Chapter), s.characterLabel),
				x.pageLine(key+"building", "", fmt.Sprintf(x.word("editor.companions.building"), j.Building)),
				number(key+"talk", x.word("editor.companions.talk"), x.word("editor.companions.talk.help"), j.Talk, talks, s.characterLabel))
			if p := x.joinProblem(j); p != "" {
				out = append(out, x.pageLine(key+"state", "", fmt.Sprintf(x.word("editor.companions.state"), p)))
			}
		}
		out = append(out, ui.InspectionField{Key: key + "remove", Label: x.word("editor.companions.remove"), Meaning: x.word("editor.companions.remove.help"), Action: true, Column: 1})
		for k := start; k < len(out); k++ {
			out[k].Group = s.joinName(j)
		}
	}
	return out
}

// editJoins is one draft step of the companions file in the project w.
func (s *mapInspectionEdits) editJoins(w *editorWorkspaceState, edit func(*mod.CompanionData) error) error {
	return kindFiles{s: s, w: w}.editRows(nil, []string{mod.CompanionsFile}, func(d *mod.Data) error { return edit(&d.Companions) })
}

// setCompanionsField makes one edit of the Companions page a draft step.
func (s *mapInspectionEdits) setCompanionsField(key, value string) (*ui.InspectionDocument, error) {
	return s.modPageEdit(key, value, s.companionsEdit)
}

func (s *mapInspectionEdits) companionsEdit(key, value string) (*ui.InspectionDocument, error) {
	x := s.inspector
	gone := x.modGone("editor.companions.err.gone", key)
	if key == companionsAddKey {
		return s.editKind(func(k kindFiles) error {
			return s.editJoins(k.w, func(d *mod.CompanionData) error { return s.addJoin(d) })
		})
	}
	kind, i, field, ok := modRowKey(key)
	if !ok || kind != "join" {
		return nil, gone
	}
	return s.editKind(func(k kindFiles) error {
		return s.editJoins(k.w, func(d *mod.CompanionData) error {
			if i >= len(d.Joins) {
				return gone
			}
			return s.editJoin(d, i, field, value, gone)
		})
	})
}

// joinFree reports whether no condition but index skip names the companion of
// the chapter.
func joinFree(joins []mod.CompanionJoin, skip, chapter, companion int) bool {
	for n, j := range joins {
		if n != skip && j.Chapter == chapter && j.Companion == companion {
			return false
		}
	}
	return true
}

// addJoin appends a condition for the first companion a town grants that has
// none, tavern conversation first of its chapter's; without a campaign, the
// first free numbers.
func (s *mapInspectionEdits) addJoin(d *mod.CompanionData) error {
	x := s.inspector
	c := x.joinCampaign()
	chapter, companion, talk := 1, 1, 1
	if c != nil {
		found := false
		for _, ch := range joinChapters(c) {
			for _, id := range joinGrants(c, ch) {
				if !found && joinFree(d.Joins, -1, ch, id) {
					chapter, companion, found = ch, id, true
				}
			}
		}
		if !found {
			return fmt.Errorf("%s", x.word("editor.companions.err.none"))
		}
		if talks := c.chapter(chapter).InnNPC; len(talks) > 0 {
			talk = talks[0]
		}
	} else {
		for !joinFree(d.Joins, -1, chapter, companion) {
			companion++
		}
	}
	d.Joins = append(d.Joins, mod.CompanionJoin{Key: joinKey(d.Joins, chapter, companion), Chapter: chapter, Companion: companion,
		Building: mod.BuildingTavern, Talk: talk})
	return nil
}

// editJoin makes the edit of field of condition i.
func (s *mapInspectionEdits) editJoin(d *mod.CompanionData, i int, field, value string, gone error) error {
	x := s.inspector
	if field == "remove" {
		d.Joins = slices.Delete(d.Joins, i, i+1)
		return nil
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 || n > mod.MaxJoinNumber {
		return fmt.Errorf(x.word("editor.companions.err.number"), 1, mod.MaxJoinNumber)
	}
	j := &d.Joins[i]
	if j.Off {
		return gone
	}
	set := func(name string) {
		if fields := j.PatchFields(); !slices.ContainsFunc(fields, func(f mod.PatchField) bool { return f.Name == name }) {
			j.Fields = append(slices.Clone(fields), mod.PatchField{Name: name, Line: j.Line})
		}
	}
	switch field {
	case "chapter":
		j.Chapter = n
		if c := x.joinCampaign(); c != nil {
			// A companion or conversation the new chapter lacks gives way to its first.
			if grants := joinGrants(c, n); len(grants) > 0 && !slices.Contains(grants, j.Companion) {
				j.Companion = grants[0]
			}
			if talks := c.chapter(n).InnNPC; len(talks) > 0 && !slices.Contains(talks, j.Talk) {
				j.Talk = talks[0]
				set("talk")
			}
		}
	case "companion":
		j.Companion = n
	case "talk":
		j.Talk = n
		set("talk")
	default:
		return gone
	}
	if !joinFree(d.Joins, i, j.Chapter, j.Companion) {
		return fmt.Errorf(x.word("editor.companions.err.twice"), s.joinName(*j))
	}
	return nil
}

// joinKey is a key no condition holds: join_<chapter>_<companion>, with the
// next free numeric suffix when that is taken.
func joinKey(joins []mod.CompanionJoin, chapter, companion int) string {
	base := fmt.Sprintf("join_%d_%d", chapter, companion)
	key := base
	for n := 2; slices.ContainsFunc(joins, func(j mod.CompanionJoin) bool { return j.Key == key }); n++ {
		key = fmt.Sprintf("%s_%d", base, n)
	}
	return key
}
