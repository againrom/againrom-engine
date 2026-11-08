package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/town"
)

// sleepControl names the Sleep control row of a town description's room,
// the button whose word is tavern.sleep: the first game's tavern and the
// second game's inns carry it.
const sleepControl = "sleep"

// sleepButton reports whether button i of a room's buttons is its Sleep
// control.
func sleepButton(buttons []town.ButtonSpec, i int) bool {
	return i >= 0 && i < len(buttons) && buttons[i].Name == sleepControl
}

// sleepControlName reports whether a control name is the Sleep control, for
// callers that hold a name and no button index.
func sleepControlName(name string) bool {
	return name == sleepControl
}

// restockInput is what a shop state's restock reads: the install's item
// table, the session's town, the party's town the stock is drawn for and the
// draws whose shop-stock stream it takes.
type restockInput struct {
	table    *mapload.Table
	town     *Town
	location int
	draws    townDraws
}

// sleep is the town's one Sleep: the campaign's town shop restocks once and
// nothing else changes. The town is timeless, so no time passes and nobody
// heals or drinks (DIV-483, DIV-4440).
func (t *townScreen) sleep() bool {
	if t == nil || t.sess == nil {
		return false
	}
	shop := t.campaignTown().townShop()
	if shop == nil {
		return false
	}
	return shop.restock(restockInput{table: t.in.Table, town: t.sess.Town, location: t.campaignTown().townID(), draws: t.draws})
}

// restock draws the stock anew for the party's town when the shop holds
// that town's record.
func (s *Shop) restock(in restockInput) bool {
	if s == nil || in.town == nil || in.draws == nil {
		return false
	}
	if s.records[in.location] != nil {
		s.location = in.location
	}
	seed := shopSeed(in.town.Chapter(), in.town.finishedCount(), s.Ceiling())
	return s.RestockWith(in.table, seed, in.draws.stream(random.ShopStock))
}
