package ui

// shopUseTap is a pack tap that a double click's second tap on the same cell
// turns into a use: a carried item goes to the doll's route, as the shop
// backpack's double click runs the character panel's item action (MENU-145,
// ITEM-USE-112). It names the tapped record, so a second tap after the pack or
// table changed under it is a single tap.
type shopUseTap struct {
	control             ShopControl
	epoch               *byte
	member, base, shelf int
	statistics          bool
	key                 string
	count               uint32
	table               int
	stagedKey           string
	stagedCount         uint32
}

func (p *shopUseTap) matches(v ShopScreenView) bool {
	if p == nil || p.epoch != v.InputEpoch || p.member != v.Member || p.base != v.PackOffset || p.shelf != v.Chosen || p.statistics != v.Character.Statistics || p.control.Index < 0 || p.control.Index >= len(v.Pack) {
		return false
	}
	c := v.Pack[p.control.Index]
	if c.UseItemKey != p.key || c.Count != p.count {
		return false
	}
	if p.table < 0 {
		return true
	}
	c = v.Table[p.table]
	return c.Mine && c.UseItemKey == p.stagedKey && c.Count == p.stagedCount
}

func (a *App) validateShopUseTap() {
	if a.shopUseTap == nil {
		return
	}
	if a.flow == nil || a.flow.screen != ScreenTown {
		a.shopUseTap = nil
		return
	}
	if _, dialogue := townDialogue(a.flow.town); dialogue {
		a.shopUseTap = nil
		return
	}
	v, ok := townShopScreen(a.flow.town)
	if !ok || !a.shopUseTap.matches(v) {
		a.shopUseTap = nil
	}
}

// useShopTap judges a tap on c. double is whether its press was the second
// press of a double click.
func (a *App) useShopTap(v ShopScreenView, c ShopControl, double bool) bool {
	if p := a.shopUseTap; p != nil && double && p.control == c && p.matches(v) {
		a.shopUseTap = nil
		from := c
		if p.table >= 0 {
			from = ShopControl{Kind: ShopControlTableCell, Index: p.table}
		}
		a.flow.dragShop(from, ShopControl{Kind: ShopControlDoll})
		return true
	}
	if c.Kind != ShopControlPackCell || c.Shift || c.Index < 0 || c.Index >= len(v.Pack) || v.Pack[c.Index].UseItemKey == "" {
		a.shopUseTap = nil
		return false
	}
	a.shopUseTap = nil
	a.clickShop(c)
	after, ok := townShopScreen(a.flow.town)
	if !ok {
		return true
	}
	p := &shopUseTap{control: c, epoch: v.InputEpoch, member: v.Member, base: v.PackOffset, shelf: v.Chosen, statistics: v.Character.Statistics, key: after.Pack[c.Index].UseItemKey, count: after.Pack[c.Index].Count, table: -1}
	for i, cell := range after.Table {
		before := v.Table[i]
		if cell.Mine && cell.UseItemKey != "" && (cell.UseItemKey != before.UseItemKey || cell.Count > before.Count) {
			if p.table >= 0 {
				return true
			}
			p.table, p.stagedKey, p.stagedCount = i, cell.UseItemKey, cell.Count
		}
	}
	if p.table < 0 && (p.key != v.Pack[c.Index].UseItemKey || p.count != v.Pack[c.Index].Count) {
		return true
	}
	if p.matches(after) {
		a.shopUseTap = p
	}
	return true
}
