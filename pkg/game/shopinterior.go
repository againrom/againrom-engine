package game

import (
	"math/rand"
	"time"

	"againrom/pkg/ui"
)

const (
	shopInteriorStep      = 100 * time.Millisecond
	shopInteriorIdleBase  = 5 * time.Second
	shopInteriorIdleRange = 5
)

const (
	shopMerchantIdle uint8 = 0x10
	shopMerchantYes  uint8 = 0x20
	shopMerchantNo   uint8 = 0x40
)

// shopInteriorAnimation is presentation-only state. It is independent from
// the town exterior, school and tavern controllers and never enters a save or
// a simulation hash. SHOP-ANIMATION-081..084 own the local progression; the
// lifecycle around physical paints remains an explicit Againrom policy.
type shopInteriorAnimation struct {
	ready, active, started bool

	last, idleLast time.Time
	random         *rand.Rand

	selectedRack int
	rackEnabled  [4]bool
	rackIndex    [4]int

	merchantModes uint8
	merchantIndex int
}

func (t *townScreen) resetShopInterior() {
	if t != nil {
		t.shopInterior = shopInteriorAnimation{}
	}
}

func (t *townScreen) enterShopInterior() {
	if t == nil || t.sess == nil {
		return
	}
	now := t.townAnimationNow()
	t.shopInterior = shopInteriorAnimation{
		ready:        true,
		last:         now.Add(-shopInteriorStep),
		idleLast:     now,
		random:       rand.New(rand.NewSource(now.UnixNano())),
		selectedRack: t.shopChosen,
	}
	if t.shopChosen >= 0 && t.shopChosen < len(t.shopInterior.rackEnabled) {
		t.shopInterior.rackEnabled[t.shopChosen] = true
	}
}

func (t *townScreen) inShopInterior() bool { return t != nil && t.AtTownShop() }

// ShopInteriorActive freezes the private clocks while the shop is not the
// focused live room. The first activation retains the researched now-100
// entry stamp; later resumes rebase both clocks and cannot catch up time spent
// behind a menu, cutscene or focus loss.
func (t *townScreen) ShopInteriorActive(active bool) {
	if t == nil || t.sess == nil {
		return
	}
	active = active && t.inShopInterior()
	if active && !t.shopInterior.ready {
		t.enterShopInterior()
	}
	a := &t.shopInterior
	if !active {
		a.active = false
		return
	}
	if a.active {
		return
	}
	if a.started {
		now := t.townAnimationNow()
		a.last, a.idleLast = now, now
	}
	a.started, a.active = true, true
}

func (t *townScreen) shopInteriorRoll(n int) int {
	if n <= 0 {
		return 0
	}
	if draw := t.draws.shopDraw(); draw != nil {
		v := draw(n) % n
		if v < 0 {
			v += n
		}
		return v
	}
	if t.shopInterior.random == nil {
		t.shopInterior.random = rand.New(rand.NewSource(t.townAnimationNow().UnixNano()))
	}
	return t.shopInterior.random.Intn(n)
}

// selectShopInteriorRack mirrors the animation selector without changing the
// shop's stock/view selection. Entry has already selected visual rack zero;
// therefore the player's first click on that same rack opens its stock but
// does not reload or arm another reaction.
func (t *townScreen) selectShopInteriorRack(rack int, react bool) bool {
	if t == nil || !t.shopInterior.ready || rack < 0 || rack >= len(t.shopInterior.rackEnabled) || rack == t.shopInterior.selectedRack {
		return false
	}
	a := &t.shopInterior
	old := a.selectedRack
	if old >= 0 && old < len(a.rackEnabled) && a.rackEnabled[old] {
		a.rackIndex[old] = 9
	}
	a.selectedRack = rack
	a.rackEnabled[rack], a.rackIndex[rack] = true, 0
	if react {
		a.merchantModes |= shopMerchantYes
	}
	return true
}

func (t *townScreen) requestShopMerchantYes() {
	if t != nil && t.shopInterior.ready {
		t.shopInterior.merchantModes |= shopMerchantYes
	}
}

func (t *townScreen) requestShopMerchantNo() {
	if t != nil && t.shopInterior.ready {
		t.shopInterior.merchantModes |= shopMerchantNo
	}
}

func (t *townScreen) advanceShopRack(i int) {
	a := &t.shopInterior
	if i < 0 || i >= len(a.rackEnabled) || !a.rackEnabled[i] {
		return
	}
	next := a.rackIndex[i] + 1
	switch next {
	case 9:
		a.rackIndex[i] = 3
	case 10:
		a.rackEnabled[i], a.rackIndex[i] = false, 0
	default:
		a.rackIndex[i] = next
	}
}

func (t *townScreen) finishShopMerchantMode(now time.Time, mode uint8) {
	a := &t.shopInterior
	a.merchantModes &^= mode
	a.merchantIndex = 0
	a.idleLast = now
}

// activeShopMerchantMode selects priority from state alone. Art availability
// is deliberately not part of this decision: an unavailable winning family
// falls back to the base merchant and cannot expose a lower-priority mode at
// the winning mode's shared index.
func activeShopMerchantMode(modes uint8) uint8 {
	switch {
	case modes&shopMerchantIdle != 0:
		return shopMerchantIdle
	case modes&shopMerchantYes != 0:
		return shopMerchantYes
	case modes&shopMerchantNo != 0:
		return shopMerchantNo
	default:
		return 0
	}
}

func (t *townScreen) advanceShopMerchant(now time.Time) {
	a := &t.shopInterior
	switch activeShopMerchantMode(a.merchantModes) {
	case shopMerchantIdle:
		next := a.merchantIndex + 1
		if next >= 28 {
			t.finishShopMerchantMode(now, shopMerchantIdle)
			return
		}
		a.merchantIndex = next
	case shopMerchantYes:
		next := a.merchantIndex + 1
		if next >= 12 {
			t.finishShopMerchantMode(now, shopMerchantYes)
			return
		}
		a.merchantIndex = next
	case shopMerchantNo:
		next := a.merchantIndex + 1
		if next >= 12 {
			t.finishShopMerchantMode(now, shopMerchantNo)
			return
		}
		a.merchantIndex = next
	}
}

// AdvanceShopInteriorAnimation performs at most one eligible paint step. A
// long gap never catches up. The idle candidate is redrawn on every eligible
// call, and merchant update/draw priority is Idle, Yes, No.
func (t *townScreen) AdvanceShopInteriorAnimation() {
	if t == nil || !t.inShopInterior() || !t.shopInterior.ready || !t.shopInterior.active {
		return
	}
	a := &t.shopInterior
	now := t.townAnimationNow()
	if now.Before(a.last) {
		a.last = now
		return
	}
	if now.Sub(a.last) < shopInteriorStep {
		return
	}
	a.last = now
	idleAfter := shopInteriorIdleBase + time.Duration(t.shopInteriorRoll(shopInteriorIdleRange))*time.Second
	if a.merchantModes == 0 && now.Sub(a.idleLast) >= idleAfter {
		a.merchantModes |= shopMerchantIdle
	}
	for i := range a.rackEnabled {
		t.advanceShopRack(i)
	}
	t.advanceShopMerchant(now)
}

func (t *townScreen) shopInteriorFrame(art *ui.ShopScreenArt) ui.ShopInteriorFrame {
	var frame ui.ShopInteriorFrame
	if t == nil || t.sess == nil || !t.shopInterior.ready || art == nil {
		return frame
	}
	frame.Animated = true
	a := &t.shopInterior
	for i := range frame.Rack {
		series := art.RackAnimation[i]
		if len(series) != shopRackFrameCount {
			frame.Rack[i] = art.ShelfAnim[i]
			frame.RackVisible[i] = art.ShelfAnim[i] != nil
			continue
		}
		if a.rackEnabled[i] && a.rackIndex[i] >= 0 && a.rackIndex[i] < len(series) {
			frame.Rack[i], frame.RackVisible[i] = series[a.rackIndex[i]], true
		}
	}
	// Do not put a nil *image.RGBA into image.Image: that interface would be
	// non-nil and the compositor would call Bounds on a nil concrete pointer.
	// A missing base stays an actual nil interface while complete animations
	// remain independently drawable.
	if art.Merchant != nil {
		frame.Merchant = art.Merchant
	}
	index := a.merchantIndex
	switch activeShopMerchantMode(a.merchantModes) {
	case shopMerchantIdle:
		if index > 0 && index < 28 && len(art.MerchantIdle) == shopMerchantIdleCount && art.MerchantIdle[index-1] != nil {
			frame.Merchant = art.MerchantIdle[index-1]
		}
	case shopMerchantYes:
		if index > 0 && index < 12 && len(art.MerchantYes) == shopMerchantReactCount && art.MerchantYes[index-1] != nil {
			frame.Merchant = art.MerchantYes[index-1]
		}
	case shopMerchantNo:
		if index > 0 && index < 12 && len(art.MerchantNo) == shopMerchantReactCount && art.MerchantNo[index-1] != nil {
			frame.Merchant = art.MerchantNo[index-1]
		}
	}
	return frame
}
