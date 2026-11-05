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

// restockInput is what a shop state's restock reads: the install's item
// table, the session's town and the draws whose shop-stock stream it takes.
type restockInput struct {
	table *mapload.Table
	town  *Town
	draws townDraws
}

// townRestocker is a town shop as Sleep asks it to restock; the first game's
// Shop and the second game's campaign, whose shop stocks the party's town,
// answer it.
type townRestocker interface {
	restock(restockInput) bool
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
	return shop.restock(restockInput{table: t.in.Table, town: t.sess.Town, draws: t.draws})
}

// restock clear-refills the first game's shop from the town's arrival seed
// folded with the next restock count. A trade table holding items refuses it.
func (s *Shop) restock(in restockInput) bool {
	if s == nil || in.town == nil {
		return false
	}
	seed := shopSeed(in.town.Chapter(), in.town.finishedCount(), s.Ceiling())
	return s.RestockWith(in.table, seed, in.draws.stream(random.ShopStock))
}

// restock draws the second game's stock of the party's town anew from its
// record with the next shop-stock draws, as a LOAD of the town does.
func (c *secondCampaign) restock(in restockInput) bool {
	tables, ok := secondShopTables(in.table)
	if !ok || in.draws == nil {
		return false
	}
	c.shop.drawStock(tables, c.current.id, in.draws.stream(random.ShopStock).Raw)
	return true
}
