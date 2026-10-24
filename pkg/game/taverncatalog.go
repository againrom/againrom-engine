package game

import (
	"errors"
	"slices"

	"againrom/pkg/mod"
	"againrom/pkg/town"
)

// A first-game town's tavern talks and quests are rows of the town catalog
// (town.TavernRow). Layer 0 is the scenario's: each chapter's InnNPC and
// InnMission pairs, walking the shorter array (REG-SCN-064). A book no mod
// layers a row over states no rows, and each chapter then reads its pairs
// from the campaign's one scenario reader on every read; a layered book
// holds the merged rows, which every tavern reader takes through
// Campaign.chapter.

// chapter is chapter n's offer data with its tavern pairs read from the
// catalog's rows, in row order.
func (c Campaign) chapter(n int) Chapter {
	ch := c.Chapters[n]
	rows, ok := c.Towns.tavernRows()
	if !ok {
		return ch
	}
	ch.InnNPC, ch.Inn = nil, nil
	for _, r := range rows {
		if r.Chapter == n {
			ch.InnNPC = append(ch.InnNPC, r.NPC)
			ch.Inn = append(ch.Inn, r.Mission)
		}
	}
	return ch
}

// chapterForms are the forms chapter n's tavern pairs can have been written
// in: the merged rows, then layer 0, the scenario's pairs, when a tavern
// layer changes the chapter.
func (c Campaign) chapterForms(n int) []Chapter {
	merged := c.chapter(n)
	base := c.Chapters[n]
	if slices.Equal(merged.InnNPC, base.InnNPC) && slices.Equal(merged.Inn, base.Inn) {
		return []Chapter{merged}
	}
	return []Chapter{merged, base}
}

// tavernRows are the rows a layer stated, and whether one did.
func (b *townBook) tavernRows() ([]town.TavernRow, bool) {
	if b == nil || b.catalog == nil || b.catalog.Tavern == nil {
		return nil, false
	}
	return b.catalog.Tavern, true
}

// addedTavernRow is the row of chapter that NPC record npc speaks when a
// mod added it, rather than layer 0.
func (b *townBook) addedTavernRow(chapter, npc int) (town.TavernRow, bool) {
	rows, _ := b.tavernRows()
	for _, r := range rows {
		if r.Chapter == chapter && r.NPC == npc {
			return r, b.addedTavern[r.Key()]
		}
	}
	return town.TavernRow{}, false
}

// scenarioTavern is layer 0 of the tavern rows: every chapter's pairs in
// chapter order and the file's order, each in the first of towns that
// offers its mission, town 1 when none does.
func scenarioTavern(c Campaign, towns []town.Entry) []town.TavernRow {
	chapters := make([]int, 0, len(c.Chapters))
	for n := range c.Chapters {
		chapters = append(chapters, n)
	}
	slices.Sort(chapters)
	out := []town.TavernRow{}
	for _, n := range chapters {
		ch := c.Chapters[n]
		for i := range min(len(ch.InnNPC), len(ch.Inn)) {
			row := town.TavernRow{Town: chapterTown, Chapter: n, NPC: ch.InnNPC[i], Mission: ch.Inn[i]}
			if i := slices.IndexFunc(towns, func(e town.Entry) bool { return townOffersMission(e, towns, row.Mission) }); i >= 0 {
				row.Town = towns[i].ID
			}
			out = append(out, row)
		}
	}
	return out
}

// layerTavern applies the mods' tavern records over layer 0 of book's
// towns through the one layer step, judging the rows after each mod: the
// catalog's row rules, a chapter that is a main mission of the campaign, a
// quest whose mission is a section of the campaign and whose town offers
// it, and, for a row the mod adds or changes, an NPC record of the install
// and a talk text the install or a loaded mod holds.
func layerTavern(book *townBook, d mod.TownData, c Campaign, in modTownInstall) ([]mod.Override, error) {
	towns := book.chapterTowns()
	base := scenarioTavern(c, towns)
	patches := make([]mod.Patch[town.TavernKey, town.TavernRow], 0, len(d.Tavern))
	for _, r := range d.Tavern {
		patches = append(patches, mod.Patch[town.TavernKey, town.TavernRow]{Source: r.Source, Key: r.Row.Key(), Fields: r.Fields, Off: r.Off, Apply: r.Apply})
	}
	layer0 := map[town.TavernKey]town.TavernRow{}
	for _, r := range base {
		layer0[r.Key()] = r
	}
	check := func(modID string, l *mod.Layered[town.TavernKey, town.TavernRow]) error {
		cat := *book.catalog
		cat.Tavern = l.Records
		err := cat.Validate()
		for i := 0; err == nil && i < len(l.Records); i++ {
			r := l.Records[i]
			if prior, ok := layer0[r.Key()]; ok && prior == r {
				continue
			}
			err = checkTavernRow(r, c, towns, in)
		}
		for i := 0; err == nil && i < len(in.Joins.Joins); i++ {
			err = checkJoinTalk(in.Joins.Joins[i], d, modID, l.Records)
		}
		if err == nil {
			return nil
		}
		src := firstTavernSource(d, modID)
		var te *town.TavernError
		if errors.As(err, &te) {
			if named, ok := l.Named[te.Row]; ok {
				src = named
			}
		}
		return src.Errorf("%v", err)
	}
	l, err := mod.Merge("tavern row", base, town.TavernRow.Key, patches, d.Order, check)
	if err != nil {
		return nil, err
	}
	cat := *book.catalog
	cat.Tavern = append([]town.TavernRow{}, l.Records...)
	book.catalog = &cat
	book.addedTavern = map[town.TavernKey]bool{}
	for _, r := range l.Records {
		if _, ok := layer0[r.Key()]; !ok {
			book.addedTavern[r.Key()] = true
		}
	}
	return l.Overrides, nil
}

// firstTavernSource is mod's first tavern record.
func firstTavernSource(d mod.TownData, modID string) mod.Source {
	for _, r := range d.Tavern {
		if r.Mod == modID {
			return r.Source
		}
	}
	return d.First()
}

// checkJoinTalk refuses mod modID's tavern records when they leave chapter
// j.Chapter without the talk companion join j waits for, naming the last of
// them that names the talk.
func checkJoinTalk(j mod.CompanionJoin, d mod.TownData, modID string, rows []town.TavernRow) error {
	if j.Building != mod.BuildingTavern || slices.ContainsFunc(rows, func(r town.TavernRow) bool { return r.Chapter == j.Chapter && r.NPC == j.Talk }) {
		return nil
	}
	for _, r := range slices.Backward(d.Tavern) {
		if r.Mod == modID && r.Row.Chapter == j.Chapter && r.Row.NPC == j.Talk {
			k := r.Row.Key()
			return town.TavernErrorf(k, "tavern row %v is switched off, but mod %q (%s line %d) makes companion %d join on this talk in chapter %d",
				k, j.Mod, j.File, j.Line, j.Companion, j.Chapter)
		}
	}
	return nil
}

// checkTavernRow is the campaign's and the install's rules for one row a
// mod adds or changes.
func checkTavernRow(r town.TavernRow, c Campaign, towns []town.Entry, in modTownInstall) error {
	k := r.Key()
	if !containsMission(c.Main, r.Chapter) {
		return town.TavernErrorf(k, "tavern row %v: chapter %d is not a main mission of the campaign", k, r.Chapter)
	}
	if r.Mission != 0 {
		if !containsMission(c.Main, r.Mission) && !containsMission(c.Side, r.Mission) {
			return town.TavernErrorf(k, "tavern row %v: mission %d is not a mission section of the campaign", k, r.Mission)
		}
		e := slices.IndexFunc(towns, func(e town.Entry) bool { return e.ID == r.Town })
		if e < 0 || !townOffersMission(towns[e], towns, r.Mission) {
			return town.TavernErrorf(k, "tavern row %v: town %d does not offer mission %d", k, r.Town, r.Mission)
		}
	}
	if in.NPC != nil && !in.NPC(r.NPC) {
		return town.TavernErrorf(k, "tavern row %v: npc %d is not a record of the install's NPC table", k, r.NPC)
	}
	path, _ := tavernTalkTextPath(r.NPC, r.Mission, r.Chapter)
	if in.Text != nil && !in.Text(path) {
		return town.TavernErrorf(k, "tavern row %v: its talk text %s is neither in the install nor served by a loaded mod", k, path)
	}
	return nil
}
