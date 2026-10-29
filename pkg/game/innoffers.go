package game

import (
	"slices"

	"againrom/pkg/town"
)

// innOffer is one option of an inn's or tavern's offer list: kind, topic and
// NPC key, and the high bit the hire state sets (R2-ENGINE-216,
// R2-ENGINE-221). Kinds 0 and 3 are talks, 1 and 2 hires.
type innOffer struct {
	kind, topic, npc int
	high             bool
}

func (o innOffer) speaker() bool { return o.kind == town.OfferTalk || o.kind == town.OfferMission }

// innState is the campaign state an offer list is admitted over: the town
// the party stands in (every admits the rows of every town), the stage, the
// bank the rows' gates read, nil reading zero, and the hire bank the hire
// options come from, absent when hire is false.
type innState struct {
	town  int
	every bool
	stage int
	bank  *[innBankSlots]int32
	hire  bool
}

func (s innState) slot(i int) int32 {
	if s.bank == nil {
		return 0
	}
	return s.bank[i]
}

// admits is the rows' predicates over s: the town, the stage, the stage
// floor and every bank gate.
func (s innState) admits(r town.OfferRow) bool {
	switch {
	case r.Town != 0 && !s.every && r.Town != s.town,
		len(r.Stages) > 0 && !slices.Contains(r.Stages, s.stage),
		r.After > 0 && s.stage <= r.After:
		return false
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

// innOffers is the one builder of an offer list: the rows s admits, in row
// order, then the hire options of s's hire bank. It changes no state.
func innOffers(rows []town.OfferRow, s innState) []innOffer {
	var out []innOffer
	for _, r := range rows {
		if s.admits(r) {
			out = append(out, innOffer{kind: r.Kind, topic: r.Topic, npc: r.NPC})
		}
	}
	if s.hire && s.bank != nil {
		out = append(out, innHires(s.bank, s.town)...)
	}
	return out
}

// innBankSlots is the size of the campaign bank the rows' gates read.
const innBankSlots = 1024

// checkOfferBank refuses a row of table whose gates read a slot outside the
// bank.
func checkOfferBank(rows []town.OfferRow, table town.OfferTable) error {
	for _, r := range rows {
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

// The hire bank: hire h (1..20) has selection slot 511+h, availability slot
// 531+h and town slot 551+h (R2-ENGINE-221, R2-ENGINE-380).
const (
	secondHireSlots        = 20
	secondHireSelection    = 512
	secondHireAvailability = 532
	secondHireTown         = 552
)

// innHires is the one hire producer: every hire of the bank standing in
// town whose availability is 1 or 2, as an option of that kind with the
// hire's number as NPC key and its selection as the high bit.
func innHires(b *[innBankSlots]int32, here int) []innOffer {
	var out []innOffer
	for i := 0; i < secondHireSlots; i++ {
		kind := b[secondHireAvailability+i]
		if int(b[secondHireTown+i]) == here && (kind == 1 || kind == 2) {
			out = append(out, innOffer{kind: int(kind), npc: i + 1, high: b[secondHireSelection+i] > 0})
		}
	}
	return out
}

// offerRows are the book's offer rows; nil for no book.
func (b *townBook) offerRows() []town.OfferRow {
	if b == nil || b.catalog == nil {
		return nil
	}
	return b.catalog.Offers
}
