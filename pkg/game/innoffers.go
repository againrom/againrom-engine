package game

import (
	"slices"

	"againrom/pkg/herokind"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/town"
)

// innOffer is one option of an inn's or tavern's offer list: kind, topic and
// NPC key, and the high bit the hire state sets (R2-ENGINE-216,
// R2-ENGINE-221). Kinds 0 and 3 are talks, 1 and 2 hires. A squad hire
// also carries its size, capacity and price and whether the purse affords it.
type innOffer struct {
	kind, topic, npc       int
	place                  string
	high                   bool
	count, capacity, price int
	affordable             bool
}

func (o innOffer) speaker() bool { return o.kind == town.OfferTalk || o.kind == town.OfferMission }

// innState is the campaign state an offer list is admitted over: the town
// the party stands in (every admits the rows of every town), the stage, the
// bank the rows' gates read, nil reading zero, the starting hero a row's
// class and sex are chosen by, and the hire the hire options come from,
// absent when hire is nil, read over the bank, the town's pools in home and
// the actor table. won answers whether a mission is won; nil admits every
// row whatever is won, which is the census of a stage's rows.
type innState struct {
	town  int
	every bool
	stage int
	won   func(mission int) bool
	bank  *[innBankSlots]int32
	hero  herokind.Kind
	hire  *hireProfile
	home  *Town
	table *mapload.Table
}

func (s innState) slot(i int) int32 {
	if s.bank == nil {
		return 0
	}
	return s.bank[i]
}

// admits is the rows' predicates over s: the town, the stage, the stage
// floors, the won missions, the starting hero and every bank gate. A row
// that stands from a stage on is offered until its own mission is won.
func (s innState) admits(r town.OfferRow) bool {
	switch {
	case r.Town != 0 && !s.every && r.Town != s.town,
		len(r.Stages) > 0 && !slices.Contains(r.Stages, s.stage),
		r.After > 0 && s.stage <= r.After,
		r.From > 0 && s.stage < r.From,
		!herokind.Named(r.Class, r.Sex).Holds(s.hero):
		return false
	}
	if s.won != nil {
		if r.From > 0 && r.Topic > 0 && s.won(r.Topic) || slices.ContainsFunc(r.Won, func(m int) bool { return !s.won(m) }) {
			return false
		}
	}
	for _, i := range r.Zero {
		if s.slot(i) != 0 {
			return false
		}
	}
	for _, i := range r.Nonzero {
		if s.slot(i) == 0 {
			return false
		}
	}
	for _, i := range r.One {
		if s.slot(i) != 1 {
			return false
		}
	}
	return len(r.Any) == 0 || slices.ContainsFunc(r.Any, func(i int) bool { return s.slot(i) != 0 })
}

// innOffers is the one builder of an offer list, the inn's, the tavern's and
// the talks': the rows s admits, in row order, then the hire options of s's
// hire. It changes no state.
func innOffers(rows []town.OfferRow, s innState) []innOffer {
	var out []innOffer
	for _, r := range rows {
		if s.admits(r) {
			out = append(out, innOffer{kind: r.Kind, topic: r.Topic, npc: r.NPC, place: r.Place})
		}
	}
	if s.hire != nil {
		out = append(out, hireOffers(s.hire, hireState{town: s.home, bank: s.bank, here: s.town, table: s.table})...)
	}
	return out
}

// innBankSlots is the size of the campaign bank the rows' gates read.
const innBankSlots = 1024

// replaceHeroGate clears the part of row's class and sex that rec's lists
// replace. The hero slots hold a fighter and a male hero at zero and a mage
// and a female hero at nonzero, so that part is a gate of the list it reads
// as, and a record that sets zero or nonzero replaces it with the list's
// other gates. The part in a list rec does not set stays as class or sex.
func replaceHeroGate(row *town.OfferRow, rec mod.OfferRecord) {
	sets := func(list string) bool {
		return slices.ContainsFunc(rec.Fields, func(f mod.PatchField) bool { return f.Name == list })
	}
	zero, nonzero := sets("zero"), sets("nonzero")
	for _, word := range []*string{&row.Class, &row.Sex} {
		switch *word {
		case herokind.Fighter, herokind.Male:
			if zero {
				*word = ""
			}
		case herokind.Mage, herokind.Female:
			if nonzero {
				*word = ""
			}
		}
	}
}

// checkOfferRows refuses a row of table whose gates read a slot outside the
// bank or whose class and sex name no starting hero.
func checkOfferRows(rows []town.OfferRow, table town.OfferTable) error {
	for _, r := range rows {
		if _, err := herokind.Parse(r.Class, r.Sex); err != nil {
			k := table.Key(r)
			return town.OfferErrorf(k, "%s row %v: %v", table, k, err)
		}
		for _, slots := range r.Gates() {
			for _, s := range slots {
				if k := table.Key(r); s >= innBankSlots {
					return town.OfferErrorf(k, "%s row %v: bank slot %d is outside the bank, 0 to %d", table, k, s, innBankSlots-1)
				}
			}
		}
	}
	return nil
}

// offerRows are the book's offer rows; nil for no book.
func (b *townBook) offerRows() []town.OfferRow {
	if b == nil || b.catalog == nil {
		return nil
	}
	return b.catalog.Offers
}
