package game

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// townPick and rowPick are the picks of the graph's town and hand-out nodes:
// what selecting the node names and parsePick reads back.
func townPick(chapter int) string { return "town " + strconv.Itoa(chapter) }

func rowPick(k town.OfferKey) string {
	return fmt.Sprintf("row %s %d %d %d %d", k.Table, k.Town, k.Chapter, k.NPC, k.Topic)
}

// parsePick reads a pick: a town's chapter, or a row's key and true.
func parsePick(p string) (int, town.OfferKey, bool, error) {
	var chapter int
	if _, err := fmt.Sscanf(p, "town %d", &chapter); err == nil {
		return chapter, town.OfferKey{}, false, nil
	}
	var table string
	var k town.OfferKey
	if _, err := fmt.Sscanf(p, "row %s %d %d %d %d", &table, &k.Town, &k.Chapter, &k.NPC, &k.Topic); err != nil {
		return 0, k, false, fmt.Errorf("no town or hand-out %q", p)
	}
	k.Table = town.OfferTable(table)
	return k.Chapter, k, true, nil
}

// cloneOffers is a copy of records that shares no list with them.
func cloneOffers(records []mod.OfferRecord) []mod.OfferRecord {
	if records == nil {
		return nil
	}
	out := make([]mod.OfferRecord, len(records))
	for i, r := range records {
		r.Fields = slices.Clone(r.Fields)
		for _, l := range []*[]int{&r.Row.Stages, &r.Row.Zero, &r.Row.Nonzero, &r.Row.One, &r.Row.Any, &r.Row.Won} {
			*l = slices.Clone(*l)
		}
		out[i] = r
	}
	return out
}

// handoutBuildings are the offer tables of the building choice, in its
// order: the tavern's talk, the shop, the school.
var handoutBuildings = []town.OfferTable{town.TavernTable, town.ShopTable, town.SchoolTable}

// draftOf is the campaign of the draft w: the one W shows when w is the
// workspace and the graph built it, else admitted now.
func (s *mapInspectionEdits) draftOf(w *editorWorkspaceState) (*campaignFlow, error) {
	if w == s.workspace && s.draftFlow != nil && bytes.Equal(s.flowImage, s.workspaceImage) {
		return s.draftFlow, nil
	}
	return s.missionDraft(w)
}

// handoutRows are the offer rows of the draft w's campaign, merged over
// layer 0, and layer 0, the install's.
func (s *mapInspectionEdits) handoutRows(w *editorWorkspaceState) (rows, base []town.OfferRow, err error) {
	stock := s.inspector.installFlow()
	if stock == nil {
		return nil, nil, errors.New(s.inspector.word("editor.graph.unbuilt"))
	}
	base = stock.campaign.Towns.baseOffers(stock.campaign)
	flow, err := s.draftOf(w)
	if err != nil {
		return nil, nil, err
	}
	if merged, ok := flow.campaign.Towns.tavernRows(); ok {
		return merged, base, nil
	}
	return base, base, nil
}

func keyed(rows []town.OfferRow, k town.OfferKey) (town.OfferRow, bool) {
	i := slices.IndexFunc(rows, func(r town.OfferRow) bool { return town.TavernTable.Key(r) == k })
	if i < 0 {
		return town.OfferRow{}, false
	}
	return rows[i], true
}

// handoutFields are the fields of the selected town or hand-out: a town adds
// a talk, a shop or a school hand-out; a hand-out shows its building, NPC,
// town, chapter, floor, mission, won missions and talk text, and switches
// off or drops the mod's row.
func (s *mapInspectionEdits) handoutFields() []ui.InspectionField {
	x := s.inspector
	if s.prepareWorkspace() != nil || s.workspace.Project == nil || x.mod == nil {
		return nil
	}
	chapter, k, isRow, err := parsePick(s.handout)
	if err != nil {
		return nil
	}
	line := func(key, text string) ui.InspectionField {
		return ui.InspectionField{Key: key, ReadOnly: true, Text: true, TextValue: text, Lines: []ui.InspectionLine{{Text: x.inspectionUTF8(text)}}}
	}
	if !isRow {
		out := []ui.InspectionField{line("town", fmt.Sprintf(x.word("editor.handout.town"), chapter))}
		for _, add := range []string{"add.tavern", "add.shop", "add.school"} {
			f := workspaceText(add, x.word("editor.handout."+add), "")
			f.Meaning = x.word("editor.handout." + add + ".hint")
			out = append(out, f)
		}
		return out
	}
	rows, base, err := s.handoutRows(s.workspace)
	if err != nil {
		return []ui.InspectionField{line("error", err.Error())}
	}
	r, ok := keyed(rows, k)
	if !ok {
		return []ui.InspectionField{line("gone", x.word("editor.handout.gone"))}
	}
	building := int64(slices.Index(handoutBuildings, town.TavernTable.Of(r)))
	var choices []ui.InspectionChoice
	for i, t := range handoutBuildings {
		choices = append(choices, ui.InspectionChoice{Value: int64(i), Label: x.inspectionUTF8(x.word("editor.handout.building." + string(t)))})
	}
	floor := int64(0)
	if len(r.Stages) == 0 {
		floor = 1
	}
	out := []ui.InspectionField{{Key: "building", Label: x.word("editor.handout.building"), Value: building, Choices: choices}}
	if r.Place == "" {
		out = append(out, workspaceText("npc", x.word("editor.handout.npc"), strconv.Itoa(r.NPC)))
	}
	out = append(out,
		workspaceText("town", x.word("editor.handout.townid"), strconv.Itoa(r.Town)),
		workspaceText("chapter", x.word("editor.handout.chapter"), strconv.Itoa(r.Chapter())),
		ui.InspectionField{Key: "from", Label: x.word("editor.handout.from"), Value: floor, Choices: []ui.InspectionChoice{
			{Value: 0, Label: x.inspectionUTF8(x.word("editor.handout.from.one"))},
			{Value: 1, Label: x.inspectionUTF8(x.word("editor.handout.from.on"))}}},
		workspaceText("mission", x.word("editor.handout.mission"), strconv.Itoa(r.Topic)),
	)
	var listed []string
	for _, n := range r.Won {
		listed = append(listed, strconv.Itoa(n))
	}
	won := workspaceText("won", x.word("editor.handout.won"), strings.Join(listed, ", "))
	won.Meaning = x.word("editor.handout.won.hint")
	out = append(out, won)
	if r.Place == "" {
		out = append(out, line("talk", s.talkLine(r)))
	}
	if _, held := keyed(base, k); held {
		out = append(out, ui.InspectionField{Key: "off", Label: x.word("editor.handout.off"), Action: true})
	}
	if slices.ContainsFunc(s.workspace.Project.Towns, func(o mod.OfferRecord) bool { return o.Key() == k }) {
		out = append(out, ui.InspectionField{Key: "remove", Label: x.word("editor.handout.remove"), Action: true})
	}
	return out
}

// talkLine says which text file the tavern row r speaks in its chapter and
// whether the mod or the install holds it.
func (s *mapInspectionEdits) talkLine(r town.OfferRow) string {
	x := s.inspector
	path, ok := tavernTalkTextPath(r.NPC, r.Topic, r.Chapter())
	if !ok {
		return x.word("editor.handout.talk.none")
	}
	if x.mod.loaded != nil {
		for _, f := range x.mod.loaded.Files.Files {
			if mod.FoldAddress(f.Address) == mod.FoldAddress(path) {
				return fmt.Sprintf(x.word("editor.handout.talk.mod"), path, f.File)
			}
		}
	}
	if x.archives != nil && x.archives.Containers != nil {
		if installText(x.archives.Containers, townDialogueSource.named(path), TextCode{}).ok {
			return fmt.Sprintf(x.word("editor.handout.talk.install"), path)
		}
	}
	return fmt.Sprintf(x.word("editor.handout.talk.missing"), path)
}

// setHandoutField selects a town or hand-out ("select", no edit), or makes
// one edit of the selected one a draft step: a field of its row, a new row
// of a town, switching an install row off or dropping the mod's row. The
// edited draft passes the game's check or is refused with its words.
func (s *mapInspectionEdits) setHandoutField(key, value string) (*ui.InspectionDocument, error) {
	if key == "select" {
		if _, _, _, err := parsePick(value); err != nil {
			return nil, err
		}
		s.handout = value
		return nil, nil
	}
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	if err := s.campaignLocked(); err != nil {
		return nil, err
	}
	chapter, k, isRow, err := parsePick(s.handout)
	if err != nil {
		return nil, err
	}
	pick := s.handout
	doc, err := s.changeWorkspace(func(w *editorWorkspaceState) error {
		rows, base, err := s.handoutRows(w)
		if err != nil {
			return err
		}
		var next town.OfferRow
		op := key
		if !isRow {
			next, err = s.newHandout(rows, chapter, key, value)
		} else {
			r, ok := keyed(rows, k)
			if !ok {
				return errors.New(s.inspector.word("editor.handout.gone"))
			}
			next, err = s.editHandout(r, key, value)
		}
		if err != nil {
			return err
		}
		_, held := keyed(base, k)
		pick = putHandout(w, k, isRow, held, next, op)
		_, err = s.missionDraft(w)
		return err
	})
	if err == nil {
		s.handout = pick
	}
	return doc, err
}

// number reads a whole number of field key.
func (s *mapInspectionEdits) number(key, value string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		if key == "town" {
			key = "townid"
		}
		return 0, fmt.Errorf(s.inspector.word("editor.handout.err.number"), s.inspector.word("editor.handout."+key))
	}
	return n, nil
}

// newHandout is the row an add field of the town of chapter makes: in the
// town of the chapter's first row, else the chapter town, in that chapter.
func (s *mapInspectionEdits) newHandout(rows []town.OfferRow, chapter int, key, value string) (town.OfferRow, error) {
	place, ok := strings.CutPrefix(key, "add.")
	if !ok {
		return town.OfferRow{}, fmt.Errorf(s.inspector.word("editor.mod.err.packagefield"), key)
	}
	fields := strings.Fields(value)
	want := 1
	if place == "tavern" {
		want = 2
	}
	var n []int
	for _, f := range fields {
		v, err := strconv.Atoi(f)
		if err != nil {
			break
		}
		n = append(n, v)
	}
	if len(fields) != want || len(n) != want {
		return town.OfferRow{}, errors.New(s.inspector.word("editor.handout." + key + ".hint"))
	}
	id := chapterTown
	at := innState{every: true, stage: chapter}
	if i := slices.IndexFunc(rows, func(r town.OfferRow) bool { return r.Town > 0 && at.admits(r) }); i >= 0 {
		id = rows[i].Town
	}
	if place == "tavern" {
		return town.ChapterOffer(id, chapter, n[0], n[1]), nil
	}
	r := town.ChapterOffer(id, chapter, 0, n[0])
	r.Place = place
	return r, nil
}

// editHandout is row r with field key set to value.
func (s *mapInspectionEdits) editHandout(r town.OfferRow, key, value string) (town.OfferRow, error) {
	r.Stages, r.Won = slices.Clone(r.Stages), slices.Clone(r.Won)
	switch key {
	case "off", "remove":
		return r, nil
	case "building":
		i, err := s.number(key, value)
		if err == nil && (i < 0 || i >= len(handoutBuildings)) {
			_, err = s.number(key, "")
		}
		if err != nil {
			return r, err
		}
		r.Place = handoutBuildings[i].Place()
		if r.Place != "" {
			r.NPC = 0
		}
	case "from":
		on := strings.TrimSpace(value) == "1"
		n := r.Chapter()
		r.Stages, r.From = []int{n}, 0
		if on {
			r.Stages, r.From = nil, n
		}
	case "won":
		r.Won = nil
		for _, f := range strings.FieldsFunc(value, func(c rune) bool { return c == ',' || c == ' ' }) {
			n, err := s.number(key, f)
			if err != nil {
				return r, err
			}
			r.Won = append(r.Won, n)
		}
	case "npc", "town", "chapter", "mission":
		n, err := s.number(key, value)
		if err != nil {
			return r, err
		}
		switch key {
		case "npc":
			r.NPC = n
		case "town":
			r.Town = n
		case "chapter":
			if len(r.Stages) == 0 {
				r.From = n
			} else {
				r.Stages = []int{n}
			}
		default:
			r.Topic, r.Kind = n, town.MissionKind(n)
		}
	default:
		return r, fmt.Errorf(s.inspector.word("editor.mod.err.packagefield"), key)
	}
	return r, nil
}

// putHandout writes the edit op of the row of key old (none for a new row;
// held when the install holds it) as next into w's offer records and
// answers the pick of the row it leaves selected. A row whose key changes
// leaves an install row switched off and a mod row replaced in its place.
func putHandout(w *editorWorkspaceState, old town.OfferKey, isRow, held bool, next town.OfferRow, op string) string {
	d := w.Project
	at := func(k town.OfferKey) int {
		return slices.IndexFunc(d.Towns, func(r mod.OfferRecord) bool { return r.Key() == k })
	}
	put := func(i int, r mod.OfferRecord) {
		r.Mod, r.File = w.Package.ID, mod.TownsFile
		if i >= 0 {
			r.Source = d.Towns[i].Source
			d.Towns[i] = r
			return
		}
		d.Towns = append(d.Towns, r)
	}
	table := town.TavernTable.Of(next)
	key := table.Key(next)
	switch {
	case op == "remove":
		if i := at(old); i >= 0 {
			d.Towns = slices.Delete(d.Towns, i, i+1)
		}
		return rowPick(old)
	case op == "off":
		put(at(old), mod.OfferRecordOf(old.Table, next, true))
		return rowPick(old)
	case !isRow || key == old:
		put(at(key), mod.OfferRecordOf(table, next, false))
		return rowPick(key)
	}
	i := at(old)
	switch {
	case held:
		put(i, mod.OfferRecordOf(old.Table, town.ChapterOffer(old.Town, old.Chapter, old.NPC, old.Topic), true))
		i = at(key)
	case i >= 0 && old.Table != key.Table:
		// The file's table names the place: the old row leaves and the
		// row of the new table is added.
		d.Towns = slices.Delete(d.Towns, i, i+1)
		i = at(key)
	}
	put(i, mod.OfferRecordOf(table, next, false))
	return rowPick(key)
}
