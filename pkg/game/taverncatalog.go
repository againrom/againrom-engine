package game

import (
	"errors"
	"fmt"
	"slices"

	"againrom/pkg/mod"
	"againrom/pkg/town"
)

// A first-game tavern row's layer 0 is the scenario's InnNPC and InnMission
// pairs, walking the shorter array, and a shop's and a school's row the
// positive elements of ShopMission and TCMission (REG-SCN-064). An unlayered
// book states no rows and each chapter reads its own lists.

// chapter is chapter n's offer data, its tavern pairs and its shop and
// school lists from the catalog's rows through the one offer builder. The
// lists hold every row of the chapter whatever is won; shows asks that.
func (c Campaign) chapter(n int) Chapter {
	ch := c.Chapters[n]
	rows, ok := c.Towns.tavernRows()
	if !ok {
		return ch
	}
	ch.InnNPC, ch.Inn, ch.Shop, ch.School = nil, nil, nil, nil
	for _, o := range innOffers(rows, innState{every: true, stage: n}) {
		switch o.place {
		case town.PlaceShop:
			ch.Shop = append(ch.Shop, o.topic)
		case town.PlaceSchool:
			ch.School = append(ch.School, o.topic)
		default:
			ch.InnNPC = append(ch.InnNPC, o.npc)
			ch.Inn = append(ch.Inn, o.topic)
		}
	}
	return ch
}

// offerPlaces are the row places of the three buildings.
var offerPlaces = map[TownBuilding]string{TownTavern: "", TownShop: town.PlaceShop, TownSchool: town.PlaceSchool}

// chapterRow is the catalog row behind an element of building b's list in
// chapter n: the tavern's by its NPC record, a shop's or a school's by its
// mission. No layer, or no such row, answers false.
func (c Campaign) chapterRow(n int, b TownBuilding, npc, mission int) (town.OfferRow, bool) {
	rows, _ := c.Towns.tavernRows()
	return chapterRowIn(rows, n, b, npc, mission)
}

// chapterRowIn is chapterRow over the rows given.
func chapterRowIn(rows []town.OfferRow, n int, b TownBuilding, npc, mission int) (town.OfferRow, bool) {
	at := innState{every: true, stage: n}
	for _, r := range rows {
		if r.Place == offerPlaces[b] && at.admits(r) && (b == TownTavern && r.NPC == npc || b != TownTavern && r.Topic == mission) {
			return r, true
		}
	}
	return town.OfferRow{}, false
}

// shows reports whether building b's element of the live chapter is on
// offer now: the row's test over what the campaign has won.
func (t *Town) shows(b TownBuilding, npc, mission int) bool {
	n := t.ChapterData().Mission
	r, ok := t.camp.chapterRow(n, b, npc, mission)
	return !ok || innState{every: true, stage: n, won: t.Done}.admits(r)
}

// chapterForms are chapter n's merged pairs, then the scenario's when a
// layer changes them.
func (c Campaign) chapterForms(n int) []Chapter {
	merged := c.chapter(n)
	base := c.Chapters[n]
	if slices.Equal(merged.InnNPC, base.InnNPC) && slices.Equal(merged.Inn, base.Inn) {
		return []Chapter{merged}
	}
	return []Chapter{merged, base}
}

// tavernRows are the tavern rows a layer stated, and whether one did.
func (b *townBook) tavernRows() ([]town.OfferRow, bool) {
	if b == nil || b.catalog == nil || b.catalog.OfferTable != town.TavernTable || b.catalog.Offers == nil {
		return nil, false
	}
	return b.catalog.Offers, true
}

// scenarioTavern is layer 0 of the tavern rows in chapter and file order,
// each in the first town offering its mission, else town 1; after a
// chapter's talks come its shop's and its school's rows, one for each
// positive mission of the list, a repeated one once.
func scenarioTavern(c Campaign, towns []town.Entry) []town.OfferRow {
	chapters := make([]int, 0, len(c.Chapters))
	for n := range c.Chapters {
		chapters = append(chapters, n)
	}
	slices.Sort(chapters)
	out := []town.OfferRow{}
	for _, n := range chapters {
		ch := c.Chapters[n]
		home := func(mission int) int {
			if j := slices.IndexFunc(towns, func(e town.Entry) bool { return townOffersMission(e, towns, mission) }); j >= 0 {
				return towns[j].ID
			}
			return chapterTown
		}
		for i := range min(len(ch.InnNPC), len(ch.Inn)) {
			out = append(out, town.ChapterOffer(home(ch.Inn[i]), n, ch.InnNPC[i], ch.Inn[i]))
		}
		for _, list := range []struct {
			place    string
			missions []int
		}{{town.PlaceShop, ch.Shop}, {town.PlaceSchool, ch.School}} {
			for i, m := range list.missions {
				if m > 0 && !slices.Contains(list.missions[:i], m) {
					row := town.ChapterOffer(home(m), n, 0, m)
					row.Place = list.place
					out = append(out, row)
				}
			}
		}
	}
	return out
}

// offerRules judge a row a mod adds or changes; a nil answer skips its
// check. missions names where mission looks; talk answers where the row's
// talk text stands and whether it is held; stages are the stages a row
// that stands from n on is offered at.
type offerRules struct {
	stage    func(n int) bool
	stages   func(n int) []int
	mission  func(n int) bool
	missions string
	offers   func(id, n int) bool
	npc      func(n int) bool
	talk     func(r town.OfferRow) (string, bool)
}

// offerRulesFor are table's rules: a tavern row's by the scenario and its
// inn talk file, an inn row's by the world map and the town text.
func offerRulesFor(table town.OfferTable, c Campaign, towns []town.Entry, in modTownInstall) offerRules {
	r := offerRules{npc: in.NPC}
	switch table {
	case town.TavernTable:
		r.stage = func(n int) bool { return containsMission(c.Main, n) }
		r.stages = func(n int) []int {
			return slices.DeleteFunc(slices.Sorted(slices.Values(c.Main)), func(m int) bool { return m < n })
		}
		r.mission = func(n int) bool { return containsMission(c.Main, n) || containsMission(c.Side, n) }
		r.missions = "a mission section of the campaign"
		r.offers = func(id, n int) bool {
			e := slices.IndexFunc(towns, func(e town.Entry) bool { return e.ID == id })
			return e >= 0 && townOffersMission(towns[e], towns, n)
		}
		if in.Text != nil {
			r.talk = func(o town.OfferRow) (string, bool) {
				if o.Place != "" {
					return "", true
				}
				at := []int{o.Chapter()}
				if o.Topic == 0 && len(o.Stages) == 0 {
					at = r.stages(o.From)
				}
				for _, n := range at {
					if path, _ := tavernTalkTextPath(o.NPC, o.Topic, n); !in.Text(path) {
						return path, false
					}
				}
				return "", true
			}
		}
	case town.InnTable:
		r.mission, r.missions = in.Mission, "a mission record of the world map"
		if in.Talk != nil {
			r.talk = func(o town.OfferRow) (string, bool) {
				key := secondTalkKey(o.NPC, o.Topic)
				return "section " + key + " of " + secondTownText, in.Talk(key)
			}
		}
	}
	return r
}

// check judges row r of table.
func (rules offerRules) check(r town.OfferRow, table town.OfferTable) error {
	k := table.Key(r)
	bad := func(format string, args ...any) error {
		return town.OfferErrorf(k, "%s row %v: "+format, append([]any{k.Table, k}, args...)...)
	}
	if rules.stage != nil && !rules.stage(r.Chapter()) {
		return bad("chapter %d is not a main mission of the campaign", r.Chapter())
	}
	for _, m := range r.Won {
		if rules.mission != nil && !rules.mission(m) {
			return bad("won names mission %d, which is not %s", m, rules.missions)
		}
	}
	if r.Kind == town.OfferMission {
		if rules.mission != nil && !rules.mission(r.Topic) {
			return bad("mission %d is not %s", r.Topic, rules.missions)
		}
		if rules.offers != nil && !rules.offers(r.Town, r.Topic) {
			return bad("town %d does not offer mission %d", r.Town, r.Topic)
		}
	}
	if rules.npc != nil && !rules.npc(r.NPC) {
		return bad("npc %d is not a record of the install's NPC table", r.NPC)
	}
	if rules.talk != nil {
		if where, ok := rules.talk(r); !ok {
			return bad("its talk text %s is neither in the install nor served by a loaded mod", where)
		}
	}
	return nil
}

// baseOffers is layer 0 of b's offer rows: the catalog's own, else the
// scenario's of campaign c; nil with no book.
func (b *townBook) baseOffers(c Campaign) []town.OfferRow {
	switch {
	case b == nil || b.catalog == nil:
		return nil
	case b.catalog.Offers != nil:
		return b.catalog.Offers
	}
	return scenarioTavern(c, b.chapterTowns())
}

// layerOffers merges the mods' offer records over book's rows, or the
// scenario's when it states none, judging each mod's result: the catalog's
// row bounds, the bank and offerRules for each added or changed row.
func layerOffers(book *townBook, d mod.TownData, c Campaign, in modTownInstall) ([]mod.Override, error) {
	table := book.catalog.OfferTable
	for _, r := range d.Offers {
		if r.Table.Catalog() != table {
			return nil, r.Errorf("catalog %s takes [[%s]] offer rows, not [[%s]] ones", book.catalog.Catalog, table, r.Table)
		}
	}
	towns := book.chapterTowns()
	base := book.baseOffers(c)
	rules := offerRulesFor(table, c, towns, in)
	key := table.Key
	patches := make([]mod.Patch[town.OfferKey, town.OfferRow], 0, len(d.Offers))
	for _, r := range d.Offers {
		apply := func(row *town.OfferRow) {
			replaceHeroGate(row, r)
			r.Apply(row)
		}
		patches = append(patches, mod.Patch[town.OfferKey, town.OfferRow]{Source: r.Source, Key: r.Key(), Fields: r.Fields, Off: r.Off, Apply: apply})
	}
	layer0 := map[town.OfferKey]town.OfferRow{}
	for _, r := range base {
		layer0[key(r)] = r
	}
	check := func(modID string, l *mod.Layered[town.OfferKey, town.OfferRow]) error {
		cat := *book.catalog
		cat.Offers = l.Records
		err := cat.Validate()
		if err == nil {
			err = checkOfferRows(cat.Offers, table)
		}
		for i := 0; err == nil && i < len(l.Records); i++ {
			r := l.Records[i]
			if prior, ok := layer0[key(r)]; ok && prior.Equal(r) {
				continue
			}
			err = rules.check(r, table)
		}
		for i := 0; err == nil && i < len(in.Joins.Joins); i++ {
			err = checkJoinTalk(in.Joins.Joins[i], d, modID, table, l.Records)
		}
		if err == nil {
			return nil
		}
		src := d.First()
		if i := slices.IndexFunc(d.Offers, func(r mod.OfferRecord) bool { return r.Mod == modID }); i >= 0 {
			src = d.Offers[i].Source
		}
		var oe *town.OfferError
		if errors.As(err, &oe) {
			if named, ok := l.Named[oe.Row]; ok {
				src = named
			}
		}
		return src.Errorf("%v", err)
	}
	l, err := mod.Merge(fmt.Sprintf("%s row", table), base, key, patches, d.Order, check)
	if err != nil {
		return nil, err
	}
	cat := *book.catalog
	cat.Offers = append([]town.OfferRow{}, l.Records...)
	book.catalog = &cat
	book.addedOffers = map[town.OfferKey]bool{}
	for _, r := range l.Records {
		if _, ok := layer0[key(r)]; !ok {
			book.addedOffers[key(r)] = true
		}
	}
	return l.Overrides, nil
}

// checkJoinTalk refuses mod modID's records when they drop the talk that
// companion join j waits for, naming the last that names it.
func checkJoinTalk(j mod.CompanionJoin, d mod.TownData, modID string, table town.OfferTable, rows []town.OfferRow) error {
	talks := func(k town.OfferKey) bool { return k.Table == table && k.Chapter == j.Chapter && k.NPC == j.Talk }
	at := innState{every: true, stage: j.Chapter}
	if j.Building != mod.BuildingTavern || slices.ContainsFunc(rows, func(r town.OfferRow) bool {
		return r.Place == "" && r.NPC == j.Talk && at.admits(r)
	}) {
		return nil
	}
	for _, r := range slices.Backward(d.Offers) {
		if k := r.Key(); r.Mod == modID && talks(k) {
			return town.OfferErrorf(k, "%s row %v is switched off, but mod %q (%s line %d) makes companion %d join on this talk in chapter %d",
				r.Table, k, j.Mod, j.File, j.Line, j.Companion, j.Chapter)
		}
	}
	return nil
}
