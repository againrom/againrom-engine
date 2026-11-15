package ui

// armedCast is the one home of the armed Cast state: a book spell armed by
// spellArmed and a scroll armed by itemCast. Both may stand at once, as a
// quick-slot arm leaves them. These methods are the only writers of the two
// fields outside tests.
type armedCast struct {
	spellArmed bool
	itemCast   *itemCastSelection
}

// armSpellCast raises the spell's armed state.
func (a *armedCast) armSpellCast() { a.spellArmed = true }

// armScrollCast arms a scroll and drops the spell's armed state.
func (a *armedCast) armScrollCast(sel *itemCastSelection) {
	a.spellArmed = false
	a.itemCast = sel
}

// clearScrollCast drops the scroll and leaves a spell armed.
func (a *armedCast) clearScrollCast() { a.itemCast = nil }

// clearArmedCast drops the spell and the scroll together.
func (a *armedCast) clearArmedCast() {
	a.spellArmed = false
	a.itemCast = nil
}

// scrollDropped is the one scroll keep test. A scroll the book did not arm is
// dropped at a release only. A scroll the book armed is dropped when the book
// stops listing it, and at a release also when the book is closed.
func (v *Viewer) scrollDropped(release bool) bool {
	c := v.itemCast
	if c == nil {
		return false
	}
	if !c.fromBook {
		return release
	}
	return !v.itemCastListed() || release && !v.hudShown(hudPanelBook)
}
