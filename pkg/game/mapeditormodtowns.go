package game

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"againrom/pkg/mod"
	"againrom/pkg/namefold"
	"againrom/pkg/town"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// townsBook is the install's town book when its catalog takes a mod's
// towns layer, the first game's; the Towns page has no fields otherwise.
func (s *mapInspectionEdits) townsBook() *townBook {
	x := s.inspector
	if x.archives == nil {
		return nil
	}
	book := townBookOf(x.archives.Base.Profile)
	if book == nil || book.catalog == nil || !townCatalogTakesLayer(book.catalog) {
		return nil
	}
	return book
}

// townShown is the name the Towns page lists a town by.
func (x *MapInspector) townShown(e town.Entry) string {
	if e.Name != "" {
		return e.Name
	}
	return fmt.Sprintf(x.word("editor.towns.town"), e.ID)
}

// townValues is town id as the game has it, as the draft d makes it, and
// the draft's record of it, nil for none.
func townValues(book *townBook, d *editorModDocument, id int) (base, merged town.Entry, rec *mod.TownRecord, inGame bool) {
	base, inGame = book.catalog.Town(id)
	if !inGame {
		base = town.Entry{ID: id}
	}
	merged = base
	if i := slices.IndexFunc(d.TownRecords, func(r mod.TownRecord) bool { return r.Entry.ID == id }); i >= 0 {
		rec = &d.TownRecords[i]
		rec.Apply(&merged)
	}
	return base, merged, rec, inGame
}

// townIDs are the game's towns and the mod's own, in order.
func townIDs(book *townBook, d *editorModDocument) []int {
	ids := book.catalog.IDs()
	for _, r := range d.TownRecords {
		if !slices.Contains(ids, r.Entry.ID) {
			ids = append(ids, r.Entry.ID)
		}
	}
	return ids
}

// campaignMissions are the missions a tavern talk may offer: the game's
// and the mod's, ascending.
func (s *mapInspectionEdits) campaignMissions(d *editorModDocument) (main, all []int) {
	if flow := s.inspector.installFlow(); flow != nil {
		main = flow.campaign.Main
		all = append(append(all, flow.campaign.Main...), flow.campaign.Side...)
	}
	for _, m := range d.Missions {
		if !m.Off && m.Number > 0 && !slices.Contains(all, m.Number) {
			all = append(all, m.Number)
		}
	}
	slices.Sort(all)
	return main, all
}

func (s *mapInspectionEdits) missionLabel(d *editorModDocument, n int) string {
	x := s.inspector
	if n == 0 {
		return x.word("editor.towns.offer.nothing")
	}
	for _, m := range d.Missions {
		if m.Number == n && !m.Off && m.Title != "" {
			return fmt.Sprintf("%s (%d)", m.Title, n)
		}
	}
	if title, _ := x.graphMission(n); title != "" {
		return fmt.Sprintf("%s (%d)", title, n)
	}
	return fmt.Sprintf(x.word("editor.towns.mission"), n)
}

func (s *mapInspectionEdits) characterLabel(npc int) string {
	x := s.inspector
	if t := x.table; t != nil && t.NPCCatalog != nil {
		if e, ok := t.NPCCatalog.Entry(int32(npc)); ok {
			return fmt.Sprintf("%s (%d)", x.npcName(e, false), npc)
		}
	}
	return fmt.Sprintf(x.word("editor.towns.character"), npc)
}

// characterChoices are the characters a tavern talk may be said by, and
// cur when it is not one of them.
func (s *mapInspectionEdits) characterChoices(cur int) []ui.InspectionChoice {
	out := s.inspector.npcChoices(math.MaxInt32, false)
	if cur > 0 && !slices.ContainsFunc(out, func(c ui.InspectionChoice) bool { return c.Value == int64(cur) }) {
		out = append(out, ui.InspectionChoice{Value: int64(cur), Label: s.inspector.inspectionUTF8(s.characterLabel(cur))})
	}
	return out
}

// talkPath is the install path of the text row r speaks.
func talkPath(r town.OfferRow) (string, bool) {
	return tavernTalkTextPath(r.NPC, r.Topic, r.Chapter())
}

// talkFile is the project's file of the talk text at path in language lang,
// under the spelling the project holds it by; rel is its path when it holds
// none.
func talkFile(d *editorModDocument, lang, path string) (rel string, held bool) {
	rel = mod.FilesDir + "/" + lang + "/" + path
	for name, b := range d.Files {
		if b != nil && namefold.ModTree(name) == namefold.ModTree(rel) {
			return name, true
		}
	}
	return rel, false
}

func townsOfferName(k town.OfferKey) string {
	return fmt.Sprintf("offer.%d.%d.%d.", k.Town, k.Chapter, k.NPC)
}

// townsFields is W's Towns page: the towns of the game and the mod, then the
// tavern offers of each town and chapter; a change shows its mark and the
// game's value beside it.
func (s *mapInspectionEdits) townsFields() []ui.InspectionField {
	x := s.inspector
	if !s.modPageHolds() || s.workspace.Project == nil {
		return nil
	}
	book := s.townsBook()
	if book == nil {
		return nil
	}
	d := s.workspace.Project
	var out []ui.InspectionField
	_, all := s.campaignMissions(d)
	opens := []ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.towns.opens.start"))}}
	for _, n := range all {
		opens = append(opens, ui.InspectionChoice{Value: int64(n), Label: x.inspectionUTF8(s.missionLabel(d, n))})
	}
	for _, id := range townIDs(book, d) {
		base, now, rec, _ := townValues(book, d, id)
		name := x.townShown(now)
		start := len(out)
		prefix := "town." + strconv.Itoa(id) + "."
		switch {
		case rec != nil && rec.Off:
			name = fmt.Sprintf(x.word("editor.towns.isoff"), name)
			out = append(out, ui.InspectionField{Key: prefix + "on", Label: x.word("editor.towns.switchon"), Meaning: x.word("editor.towns.switchon.hint"), Action: true, Column: 1})
		default:
			if rec != nil {
				name = fmt.Sprintf(x.word("editor.towns.changed"), name)
				out = append(out, x.pageLine(prefix+"game", "", fmt.Sprintf(x.word("editor.towns.game"), x.townShown(base), s.missionLabel(d, base.OpensAfter))))
			}
			f := workspaceText(prefix+"name", x.word("editor.towns.name"), now.Name)
			f.Meaning = x.word("editor.towns.name.hint")
			out = append(out, f)
			o := ui.InspectionField{Key: prefix + "opens", Label: x.word("editor.towns.opens"), Meaning: x.word("editor.towns.opens.hint"), Value: int64(now.OpensAfter), Choices: slices.Clone(opens)}
			if !slices.ContainsFunc(o.Choices, func(c ui.InspectionChoice) bool { return c.Value == o.Value }) {
				o.Choices = append(o.Choices, ui.InspectionChoice{Value: o.Value, Label: x.inspectionUTF8(s.missionLabel(d, now.OpensAfter))})
			}
			out = append(out, o)
			px, py := "", ""
			if p, _, ok := now.MapPoint.Stated(); ok {
				px, py = strconv.Itoa(p.X), strconv.Itoa(p.Y)
			}
			fx, fy := workspaceText(prefix+"x", x.word("editor.towns.mapx"), px), workspaceText(prefix+"y", x.word("editor.towns.mapy"), py)
			fx.Meaning, fy.Meaning = x.word("editor.towns.map.hint"), x.word("editor.towns.map.hint")
			out = append(out, fx, fy)
			out = append(out, ui.InspectionField{Key: prefix + "off", Label: x.word("editor.towns.switchoff"), Meaning: x.word("editor.towns.switchoff.hint"), Action: true, Column: 1})
		}
		if rec != nil {
			out = append(out, ui.InspectionField{Key: prefix + "reset", Label: x.word("editor.towns.reset"), Meaning: x.word("editor.towns.reset.hint"), Action: true, Column: 1})
		}
		for j := start; j < len(out); j++ {
			out[j].Group = fmt.Sprintf("%s (%d)", name, id)
		}
	}
	rows, base, err := s.handoutRows(s.workspace)
	if err != nil {
		return append(out, x.pageLine("error", "", err.Error()))
	}
	main, _ := s.campaignMissions(d)
	cur := func(n int) []ui.InspectionChoice { return s.characterChoices(n) }
	add := ui.InspectionField{Key: "add", Label: x.word("editor.towns.offer.add"), Meaning: x.word("editor.towns.offer.add.hint"),
		Choices: append([]ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.towns.offer.choose"))}}, cur(0)...)}
	out = append(out, add)
	listed := map[town.OfferKey]bool{}
	var shown []town.OfferRow
	for _, r := range rows {
		if r.Place == "" {
			shown = append(shown, r)
			listed[town.TavernTable.Key(r)] = true
		}
	}
	for _, r := range base {
		if r.Place == "" && !listed[town.TavernTable.Key(r)] {
			shown = append(shown, r)
			// A game offer the mod switches off stays in the list, to switch on.
			out = append(out, s.offerOffFields(r)...)
		}
	}
	chapters := make([]ui.InspectionChoice, 0, len(main))
	_, allMissions := s.campaignMissions(d)
	missions := []ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.towns.offer.nothing"))}}
	for _, n := range allMissions {
		missions = append(missions, ui.InspectionChoice{Value: int64(n), Label: x.inspectionUTF8(s.missionLabel(d, n))})
	}
	for _, n := range main {
		chapters = append(chapters, ui.InspectionChoice{Value: int64(n), Label: x.inspectionUTF8(fmt.Sprintf(x.word("editor.towns.chapter"), n))})
	}
	for _, r := range shown {
		k := town.TavernTable.Key(r)
		prefix := townsOfferName(k)
		_, tn, _, _ := townValues(book, d, r.Town)
		group := fmt.Sprintf(x.word("editor.towns.offer.group"), x.townShown(tn), r.Chapter(), s.characterLabel(r.NPC), s.missionLabel(d, r.Topic))
		_, game := keyed(base, k)
		start := len(out)
		who := ui.InspectionField{Key: prefix + "npc", Label: x.word("editor.towns.offer.who"), Meaning: x.word("editor.towns.offer.who.hint"), Value: int64(r.NPC), Choices: cur(r.NPC)}
		mission := ui.InspectionField{Key: prefix + "mission", Label: x.word("editor.towns.offer.mission"), Meaning: x.word("editor.towns.offer.mission.hint"), Value: int64(r.Topic), Choices: slices.Clone(missions)}
		if !slices.ContainsFunc(mission.Choices, func(c ui.InspectionChoice) bool { return c.Value == mission.Value }) {
			mission.Choices = append(mission.Choices, ui.InspectionChoice{Value: mission.Value, Label: x.inspectionUTF8(s.missionLabel(d, r.Topic))})
		}
		chapter := ui.InspectionField{Key: prefix + "chapter", Label: x.word("editor.towns.offer.chapter"), Meaning: x.word("editor.towns.offer.chapter.hint"), Value: int64(r.Chapter()), Choices: slices.Clone(chapters)}
		if !slices.ContainsFunc(chapter.Choices, func(c ui.InspectionChoice) bool { return c.Value == chapter.Value }) {
			chapter.Choices = append(chapter.Choices, ui.InspectionChoice{Value: chapter.Value, Label: x.inspectionUTF8(fmt.Sprintf(x.word("editor.towns.chapter"), r.Chapter()))})
		}
		out = append(out, who, mission, chapter)
		if path, ok := talkPath(r); ok {
			for _, l := range textLanguages() {
				value := ""
				if rel, held := talkFile(d, l, path); held {
					value = mod.TalkBody(d.Files[rel])
				}
				f := workspaceText(prefix+"talk."+l, fmt.Sprintf(x.word("editor.towns.talk"), languageLabel(l)), value)
				f.Meaning = x.word("editor.towns.talk.hint")
				out = append(out, f)
			}
		}
		label, hint := x.word("editor.towns.offer.remove"), x.word("editor.towns.offer.remove.hint")
		if game {
			label, hint = x.word("editor.towns.offer.off"), x.word("editor.towns.offer.off.hint")
		}
		out = append(out, ui.InspectionField{Key: prefix + "drop", Label: label, Meaning: hint, Action: true, Column: 1})
		for j := start; j < len(out); j++ {
			out[j].Group = group
		}
	}
	return out
}

// offerOffFields is the group of a game offer the mod has switched off.
func (s *mapInspectionEdits) offerOffFields(r town.OfferRow) []ui.InspectionField {
	x := s.inspector
	group := fmt.Sprintf(x.word("editor.towns.offer.isoff"), fmt.Sprintf(x.word("editor.towns.offer.group"), fmt.Sprintf(x.word("editor.towns.town"), r.Town), r.Chapter(), s.characterLabel(r.NPC), s.missionLabel(s.workspace.Project, r.Topic)))
	return []ui.InspectionField{{Key: townsOfferName(town.TavernTable.Key(r)) + "drop", Label: x.word("editor.towns.offer.on"), Meaning: x.word("editor.towns.offer.on.hint"), Action: true, Column: 1, Group: group}}
}

// setTownsField makes one edit of the Towns page one draft step, the value
// untrimmed (a talk text keeps its spaces); the draft
// passes the game's check or is refused with its words.
func (s *mapInspectionEdits) setTownsField(key, value string) (*ui.InspectionDocument, error) {
	return s.modPageEdit(key, value, func(k, _ string) (*ui.InspectionDocument, error) { return s.townsEdit(k, value) })
}

func (s *mapInspectionEdits) townsEdit(key, value string) (*ui.InspectionDocument, error) {
	x := s.inspector
	book := s.townsBook()
	if book == nil || s.workspace.Project == nil {
		return nil, x.modGone("editor.towns.err.gone")
	}
	gone := x.modGone("editor.towns.err.gone")
	switch {
	case key == "add":
		if value == "0" || value == "" {
			return nil, nil
		}
		npc, err := strconv.Atoi(value)
		if err != nil || npc <= 0 {
			return nil, gone
		}
		return s.changeWorkspace(func(w *editorWorkspaceState) error { return s.addOffer(w, npc) })
	case strings.HasPrefix(key, "town."):
		parts := strings.SplitN(key, ".", 3)
		id, err := strconv.Atoi(parts[1])
		if len(parts) != 3 || err != nil {
			return nil, gone
		}
		return s.changeWorkspace(func(w *editorWorkspaceState) error {
			if err := s.editTown(w, book, id, parts[2], value); err != nil {
				return err
			}
			_, err := s.missionDraft(w)
			return err
		})
	case strings.HasPrefix(key, "offer."):
		var k town.OfferKey
		k.Table = town.TavernTable
		parts := strings.SplitN(key, ".", 5)
		if len(parts) != 5 {
			return nil, gone
		}
		for i, p := range []*int{&k.Town, &k.Chapter, &k.NPC} {
			n, err := strconv.Atoi(parts[i+1])
			if err != nil {
				return nil, gone
			}
			*p = n
		}
		return s.changeWorkspace(func(w *editorWorkspaceState) error {
			if err := s.editOffer(w, k, parts[4], value, gone); err != nil {
				return err
			}
			_, err := s.missionDraft(w)
			return err
		})
	}
	return nil, gone
}

// townNeededBy is the mission of the mod that the town id's offers need: a
// mission the town lists, or one a tavern talk of the town offers.
func (s *mapInspectionEdits) townNeededBy(w *editorWorkspaceState, book *townBook, id int) (mod.Mission, bool) {
	d := w.Project
	_, now, _, _ := townValues(book, d, id)
	rows, _, _ := s.handoutRows(w)
	for _, m := range d.Missions {
		if m.Off || m.Number <= 0 {
			continue
		}
		if slices.Contains(now.Missions, m.Number) ||
			slices.ContainsFunc(rows, func(r town.OfferRow) bool { return r.Town == id && r.Topic == m.Number && r.Place == "" }) {
			return m, true
		}
	}
	return mod.Mission{}, false
}

func (s *mapInspectionEdits) neededError(format string, m mod.Mission) error {
	name := m.Title
	if name == "" {
		name = m.Key
	}
	return fmt.Errorf(format, name, m.Number)
}

// editTown applies the page's edit of field to town id in w's records.
func (s *mapInspectionEdits) editTown(w *editorWorkspaceState, book *townBook, id int, field, value string) error {
	x := s.inspector
	d := w.Project
	base, now, found, inGame := townValues(book, d, id)
	if !inGame && found == nil {
		return x.modGone("editor.towns.err.gone")
	}
	i := slices.IndexFunc(d.TownRecords, func(r mod.TownRecord) bool { return r.Entry.ID == id })
	rec := mod.TownRecord{Source: mod.Source{Mod: w.Package.ID, File: mod.TownsFile}, Entry: town.Entry{ID: id}}
	if i >= 0 {
		rec = d.TownRecords[i]
	}
	set := func(name string, differs bool) {
		at := slices.IndexFunc(rec.Fields, func(f mod.PatchField) bool { return f.Name == name })
		switch {
		case differs && at < 0:
			rec.Fields = append(rec.Fields, mod.PatchField{Name: name})
		case !differs && at >= 0:
			rec.Fields = slices.Delete(rec.Fields, at, at+1)
		}
	}
	switch field {
	case "reset":
		if i >= 0 {
			d.TownRecords = slices.Delete(d.TownRecords, i, i+1)
		}
		return nil
	case "on":
		if i >= 0 && rec.Off {
			d.TownRecords = slices.Delete(d.TownRecords, i, i+1)
		}
		return nil
	case "off":
		if m, ok := s.townNeededBy(w, book, id); ok {
			return s.neededError(s.inspector.word("editor.towns.err.townneeded"), m)
		}
		rec.Off, rec.Entry, rec.Fields = true, town.Entry{ID: id}, []mod.PatchField{{Name: mod.OffField}}
	case "name":
		v := strings.TrimSpace(value)
		if n := utf8.RuneCountInString(v); n > 40 || !utf8.ValidString(v) {
			return errors.New(x.word("editor.towns.err.name"))
		}
		rec.Entry.Name = v
		set("name", v != "" && v != base.Name)
	case "opens":
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || n < 0 {
			return x.modGone("editor.towns.err.gone")
		}
		rec.Entry.OpensAfter = n
		set("opens-after", n != base.OpensAfter)
	case "x", "y":
		v := strings.TrimSpace(value)
		if v == "" {
			rec.Entry.MapPoint = base.MapPoint
			set("map-point", false)
			break
		}
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return errors.New(x.word("editor.towns.err.point"))
		}
		p, r, ok := now.MapPoint.Stated()
		if !ok {
			// A point the game reads from the install has no size: the
			// click rectangle around a typed point is 40 by 30.
			r = image.Rect(0, 0, 40, 30)
		}
		if field == "x" {
			p.X = n
		} else {
			p.Y = n
		}
		wd, ht := r.Dx(), r.Dy()
		rec.Entry.MapPoint = town.MapPoint{Point: []int{p.X, p.Y}, Rect: []int{max(0, p.X-wd/2), max(0, p.Y-ht/2), wd, ht}}
		set("map-point", true)
	default:
		return x.modGone("editor.towns.err.gone")
	}
	if !rec.Off {
		rec.Fields = slices.DeleteFunc(rec.Fields, func(f mod.PatchField) bool { return f.Name == mod.OffField })
	}
	rec.Entry.ID = id
	switch {
	case len(rec.Fields) == 0 && i >= 0:
		d.TownRecords = slices.Delete(d.TownRecords, i, i+1)
	case len(rec.Fields) == 0:
	case i >= 0:
		d.TownRecords[i] = rec
	default:
		d.TownRecords = append(d.TownRecords, rec)
	}
	return nil
}

// addOffer adds a tavern offer of character npc in the first chapter the
// character has none, with a talk text in each language.
func (s *mapInspectionEdits) addOffer(w *editorWorkspaceState, npc int) error {
	rows, base, err := s.handoutRows(w)
	if err != nil {
		return err
	}
	main, _ := s.campaignMissions(w.Project)
	for _, n := range main {
		if slices.ContainsFunc(rows, func(r town.OfferRow) bool { return r.Place == "" && r.NPC == npc && r.Chapter() == n }) {
			continue
		}
		next, err := s.newHandout(rows, n, "add.tavern", fmt.Sprintf("%d 0", npc))
		if err != nil {
			return err
		}
		_, held := keyed(base, town.TavernTable.Key(next))
		putHandout(w, town.OfferKey{}, false, held, next, "add")
		if err := s.ensureTalk(w, next); err != nil {
			return err
		}
		_, err = s.missionDraft(w)
		return err
	}
	return errors.New(s.inspector.word("editor.towns.err.nochapter"))
}

// offerNeeded refuses to take away the offer r while it is the one that
// gives a mission of the mod.
func (s *mapInspectionEdits) offerNeeded(rows []town.OfferRow, r town.OfferRow, w *editorWorkspaceState) error {
	for _, m := range w.Project.Missions {
		if m.Off || m.Number <= 0 || r.Topic != m.Number {
			continue
		}
		others := slices.ContainsFunc(rows, func(o town.OfferRow) bool {
			return o.Topic == m.Number && town.TavernTable.Key(o) != town.TavernTable.Key(r)
		})
		if !others {
			return s.neededError(s.inspector.word("editor.towns.err.offerneeded"), m)
		}
	}
	return nil
}

// editOffer applies the page's edit of field to the tavern offer k.
func (s *mapInspectionEdits) editOffer(w *editorWorkspaceState, k town.OfferKey, field, value string, gone error) error {
	rows, base, err := s.handoutRows(w)
	if err != nil {
		return err
	}
	r, present := keyed(rows, k)
	_, game := keyed(base, k)
	if !present {
		if !game {
			return gone
		}
		if field == "drop" {
			putHandout(w, k, true, game, town.OfferRow{}, "remove")
			return nil
		}
		return gone
	}
	if lang, ok := strings.CutPrefix(field, "talk."); ok {
		path, has := talkPath(r)
		if !has || !slices.Contains(textLanguages(), lang) || strings.TrimSpace(value) == "" || utf8.RuneCountInString(value) > 1000 {
			return errors.New(s.inspector.word("editor.towns.err.talk"))
		}
		rel, _ := talkFile(w.Project, lang, path)
		w.Project.Files[rel] = mod.TalkText(r.NPC, value)
		return nil
	}
	if field == "drop" {
		if err := s.offerNeeded(rows, r, w); err != nil {
			return err
		}
		op := "remove"
		if game {
			op = "off"
		}
		putHandout(w, k, true, game, r, op)
		return s.dropTalk(w, r)
	}
	if !slices.Contains([]string{"npc", "mission", "chapter"}, field) {
		return gone
	}
	next, err := s.editHandout(r, field, value)
	if err != nil {
		return err
	}
	if field == "mission" && next.Topic != r.Topic {
		if err := s.offerNeeded(rows, r, w); err != nil {
			return err
		}
	}
	putHandout(w, k, true, game, next, field)
	if err := s.ensureTalk(w, next); err != nil {
		return err
	}
	return s.dropTalk(w, r)
}

// dropTalk removes the talk texts the page made for the offer r from the
// project when no remaining offer speaks the same text path. A file that
// differs from the page's own text format was not made by it and stays.
func (s *mapInspectionEdits) dropTalk(w *editorWorkspaceState, r town.OfferRow) error {
	path, ok := talkPath(r)
	if !ok {
		return nil
	}
	rows, _, err := s.handoutRows(w)
	if err != nil {
		return err
	}
	for _, o := range rows {
		if p, has := talkPath(o); has && p == path {
			return nil
		}
	}
	for _, l := range textLanguages() {
		rel, held := talkFile(w.Project, l, path)
		if held && bytes.Equal(w.Project.Files[rel], mod.TalkText(r.NPC, mod.TalkBody(w.Project.Files[rel]))) {
			delete(w.Project.Files, rel)
		}
	}
	return nil
}

// ensureTalk gives row r a talk text in each language where neither the
// project nor the install language's own text holds one.
func (s *mapInspectionEdits) ensureTalk(w *editorWorkspaceState, r town.OfferRow) error {
	x := s.inspector
	path, ok := talkPath(r)
	if !ok {
		return nil
	}
	installed := x.archives != nil && x.archives.Containers != nil && installText(x.archives.Containers, townDialogueSource.named(path), TextCode{}).ok
	for _, l := range textLanguages() {
		if _, held := talkFile(w.Project, l, path); held || installed && l == s.installTextLanguage() {
			continue
		}
		rel, _ := talkFile(w.Project, l, path)
		w.Project.Files[rel] = mod.TalkText(r.NPC, words.For(l).Text("editor.towns.talk.default"))
	}
	return nil
}
