package ui

import "time"

const shopUseWindow = 500 * time.Millisecond

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
	at                  time.Time
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

func (a *App) expireShopUseTap(now time.Time) {
	a.validateShopUseTap()
	p := a.shopUseTap
	if p == nil || now.Sub(p.at) <= shopUseWindow {
		return
	}
	a.shopUseTap = nil
}

func (a *App) useShopTap(v ShopScreenView, c ShopControl, now time.Time) bool {
	if p := a.shopUseTap; p != nil && p.control == c && p.matches(v) && now.Sub(p.at) <= shopUseWindow {
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
	p := &shopUseTap{control: c, epoch: v.InputEpoch, member: v.Member, base: v.PackOffset, shelf: v.Chosen, statistics: v.Character.Statistics, key: after.Pack[c.Index].UseItemKey, count: after.Pack[c.Index].Count, table: -1, at: now}
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
