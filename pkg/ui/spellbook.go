package ui

import (
	"image"
	"image/color"

	"againrom/pkg/render/text"
)

// The mission book shares MAGIC-ICON-024's fixed row-major catalog with its
// click and hover surfaces. Installed art supplies the atlas, unknown masks
// and gold wings. Compact/custom books retain an icon-or-label fallback.
// Spell knowledge and command identities arrive from pkg/game.

// SpellEntry is one entry of a unit's book, as pkg/game hands it to the
// front end: the spell's own id and its name, verbatim, off the installed
// Spells collection. Nothing about it is a claim of decoded layout — the
// id and the name are the whole of what an entry needs to be clicked and
// read.
type SpellEntry struct {
	ID   uint32
	Name string
	// Unavailable keeps a catalog cell hoverable/bindable without authorizing
	// a cast. Zero preserves the existing custom/compact SetSpellbook seam.
	Unavailable bool
	// PointTarget keeps ground spells aimed at the clicked cell even when a
	// creature covers that pixel. False preserves custom unit-target entries.
	PointTarget bool
	// SelfOnly marks a spell cast on its own caster: the cast cursor shows
	// only over the caster. False keeps the unit-target reading.
	SelfOnly bool

	// Icon is this spell's own picture, or nil for one that has none (0141;
	// `MAGIC-ICON-024`). It arrives already CUT — the tier above takes it out
	// of the one strip the archive carries, because there is no per-spell file
	// to load and this package may not open an archive to find out.
	//
	// NIL IS ORDINARY AND IS NOT A FAILURE. Four of the twenty-eight shipped
	// spells sit in no slot of that strip and have no picture at all; an
	// install whose strip will not read leaves every spell without one. Both
	// draw the abbreviation this bar drew before icons existed, which is the
	// placeholder that was always the fallback and is now only the fallback.
	//
	// It is BORROWED, not copied: the composition reads it while painting and
	// keeps no reference, so a producer reusing one buffer must hand it over
	// again after changing it (ui.Dialogue.Face's own rule for the same shape).
	Icon *image.RGBA

	// Info is the popup's lines for this spell, already composed on the far
	// side of the seam — the same shape InventorySubject.SlotInfo already
	// crosses for an item. An entry carrying none draws no popup, which is what
	// a spell the world's table does not hold arrives as.
	Info []string

	// Autocast is true when this spell is the book owner's own autocast spell.
	// It is the ONE thing that puts the rotating dashed border on this cell,
	// and this package derives it from nothing: which spell an entity casts
	// unbidden is simulation state, read on the far side and carried here as a
	// flag.
	Autocast bool
}

// The bar's own authored palette. None of it is decoded — nothing published
// draws this bar at all — so every colour here is a choice, and every one of
// them is a colour the inventory's own furniture already uses (inventory.go),
// so the three boxes along the bottom read as one set rather than as three
// separate inventions.
var (
	spellbookFill     = invFill
	spellbookBorder   = invBorder
	spellbookText     = color.RGBA{R: 0xf2, G: 0xe6, B: 0xc4, A: 0xff}
	spellbookSelected = color.RGBA{R: 0xff, G: 0xd7, B: 0x3c, A: 0xff}
)

// spellAbbrev is the label one cell carries for a spell: the first letter of
// each word of its name, up to three, and for a single-word name its first
// three letters.
//
// IT WALKS BYTES AND NOT RUNES, which is correct here and would be a defect
// almost anywhere else: a name arrives in the GAME's own byte arrangement and
// the font indexes by that byte (pkg/render/text's walk — "it reads s as a
// byte string in the game's own arrangement, never as UTF-8, never
// transcoded"), so one byte is exactly one drawn glyph and slicing by byte
// can no more split a glyph than slicing by rune could in UTF-8.
//
// FOR THE SAME REASON IT UPPERCASES ASCII ONLY. A byte above 0x7f is a letter
// in the game's own arrangement, whose case mapping is not this package's to
// know, and shifting one by an ASCII rule would draw a different glyph
// entirely. Those bytes are passed through exactly as they arrived.
func spellAbbrev(name string) string {
	up := func(b byte) byte {
		if b >= 'a' && b <= 'z' {
			return b - 'a' + 'A'
		}
		return b
	}

	var initials []byte
	word := true
	for i := 0; i < len(name) && len(initials) < 3; i++ {
		c := name[i]
		if c == ' ' || c == '\t' {
			word = true
			continue
		}
		if word {
			initials = append(initials, up(c))
			word = false
		}
	}
	if len(initials) > 1 {
		return string(initials)
	}

	// One word (or none): its own first three bytes, which says more than one
	// initial does when there is no second word to pair it with.
	var out []byte
	for i := 0; i < len(name) && len(out) < 3; i++ {
		if name[i] == ' ' || name[i] == '\t' {
			continue
		}
		out = append(out, up(name[i]))
	}
	return string(out)
}

// drawSpellLabel draws one cell's abbreviation, centred, dropping letters
// until what is left fits the cell — so a font wider than this bar expected
// loses the tail of a label rather than spilling it over the neighbouring
// cell.
//
// A nil font, a font that measures the label to nothing, and a cell too small
// for even one letter all draw NOTHING and are not failures: the bar composes
// whether or not a font has ever reached the viewer, exactly as the pack's own
// count does (drawInventoryPackCount, inventory.go).
func drawSpellLabel(dst *image.RGBA, f *text.Font, box image.Rectangle, label string, c color.RGBA) {
	if f == nil || label == "" {
		return
	}
	in := box.Inset(2)
	for s := label; s != ""; s = s[:len(s)-1] {
		w, h := f.Measure(s)
		if w <= 0 || h <= 0 {
			continue
		}
		if w > in.Dx() || h > in.Dy() {
			continue
		}
		f.Draw(dst, s, in.Min.X+(in.Dx()-w)/2, in.Min.Y+(in.Dy()-h)/2, c)
		return
	}
}

// spellbookBar is where the bar stands in WINDOW PIXELS, how many columns it
// draws, and whether one stands anywhere at all — inventoryWindowRect's own
// shape and reason, applied to a box with no fixed size: the origin is
// computed once here and read by the hit test, the swallow and the draw alike,
// so none of the three can drift from where the bar actually paints.
//
// IT ASKS ITS OWN SWITCH FIRST (0140: hudPanelBook, the S button on the control
// panel — hudtoggles.go). This was the ONE box of the four whose gate was left
// out when the switches landed, and the owner found it: the button flipped a
// flag nothing read, so the book could not be hidden. Every other box asks
// hudShown inside its own rect function (inventory.go's packBar, wornBox,
// dollBox) and this is that same statement, in the same place, for the fourth.
//
// A VIEWER HOLDING NO FONT SHOWS NOTHING (panelPresent's own font gate;
// inventoryCaptures' own "a closed window captures nothing"). A switched-on
// book with no unit selected stands empty and says so (DIV-2264):
// pkg/game clears the book on every frame with no unit selected (spell.go's
// pushSpellbook), which leaves v.spellbookHeld false.
//
// DIV-326
func (v *Viewer) spellbookBar() (image.Rectangle, int, bool) {
	if !v.hudShown(hudPanelBook) || v.font == nil {
		return image.Rectangle{}, 0, false
	}
	bar, cols, ok := bookBarRect(image.Pt(v.frameW, v.frameH))
	if !ok {
		return image.Rectangle{}, 0, false
	}
	// With no inventory bar on screen, the book owns the bottom slot itself.
	// Keeping the old stacked Y would leave a dead strip below it that was
	// neither game surface nor interface.
	if _, _, packShown := v.packBarShown(); !packShown {
		bar = bar.Add(image.Pt(0, v.frameH-bar.Max.Y))
	}
	return bar, cols, true
}

// spellbookEntryAt reports which entry, if any, stands under the window pixel
// (x, y) — inventoryPackCellAt's own shape and reason: read spellbookBar's own
// rectangle and bookCellRects' own cells and nothing else, never a fresh
// computation from the camera or the view size.
//
// A CELL PAST THE END OF THE BOOK ANSWERS NO ENTRY. The bar is drawn full
// width whatever is known, so most of it is usually empty cells, and an empty
// cell names no spell to select.
func (v *Viewer) spellbookEntryAt(x, y int) (int, bool) {
	bar, cols, ok := v.spellbookBar()
	if !ok {
		return 0, false
	}
	p := image.Pt(x, y)
	for i, r := range bookCellRects(bar, cols) {
		if !p.In(r) {
			continue
		}
		if i >= len(v.spellbook) || v.spellbook[i].ID == 0 {
			return 0, false
		}
		return i, true
	}
	return 0, false
}

// spellbookCaptures reports that the bar stands under the window pixel
// (x, y) — inventoryCaptures' own shape: an empty book captures nothing, so
// every press on every screen the bar never grew on is answered by the same
// statement rather than by a caller remembering to ask whether one exists
// first.
func (v *Viewer) spellbookCaptures(x, y int) bool {
	bar, _, ok := v.spellbookBar()
	return ok && v.spellbookHeld && image.Pt(x, y).In(bar)
}

// composeSpellBar is the asset-free and compact-shop fallback. The mission
// catalog uses composeOriginalSpellBar when the installed family is present.
func composeSpellBar(f *text.Font, entries []SpellEntry, selected uint32, cols int, bar image.Rectangle, phase int) *image.RGBA {
	size := bar.Size()
	img := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	drawFrame(img, panelFrame(image.Rectangle{Max: size}, spellbookFill, spellbookBorder))

	for i, box := range bookCellRects(bar, cols) {
		box = box.Sub(bar.Min)
		if i >= len(entries) || entries[i].ID == 0 {
			drawInventoryCell(img, box, nil, invCellBorder)
			continue
		}
		e := entries[i]
		border, ink := invCellBorder, spellbookText
		if e.Unavailable {
			ink = invCellBorder
		}
		if e.ID == selected {
			border, ink = spellbookSelected, spellbookSelected
		}
		// Icons keep their native size; missing art falls back to a label.
		drawInventoryCell(img, box, nil, border)
		drawInventoryPicture(img, box, e.Icon)
		if e.Unavailable && e.Icon != nil {
			// Darken only the icon interior. Current and binding marks remain
			// legible and separate from availability.
			for y := box.Min.Y + 1; y < box.Max.Y-1; y++ {
				for x := box.Min.X + 1; x < box.Max.X-1; x++ {
					c := img.RGBAAt(x, y)
					c.R, c.G, c.B = c.R/3, c.G/3, c.B/3
					img.SetRGBA(x, y, c)
				}
			}
		}
		if e.ID == selected {
			drawFrame(img, frameSpec{Kind: frameOutline, Rect: box, Border: border})
		}
		if e.Icon == nil {
			drawSpellLabel(img, f, box, spellAbbrev(e.Name), ink)
		}
		// AND THE AUTOCAST MARK LAST, OVER BOTH: the rotating dashed border is
		// drawn after the icon and after the label, so it reads over the picture
		// rather than under it. It is the only mark that says a spell is on
		// autocast, and it is drawn on the cell and nowhere else.
		if e.Autocast {
			drawAutocastBorder(img, box, phase, autocastDashColor)
		}
	}
	return img
}

// spellbookPresent is the picture to draw this frame and where its top-left
// corner goes, or false for a frame that draws no bar at all — the panel's own
// shape: it decides everything and Draw decides nothing.
func (v *Viewer) spellbookPresent() (*image.RGBA, image.Point, bool) {
	bar, cols, ok := v.spellbookBar()
	if !ok {
		return nil, image.Point{}, false
	}
	var img *image.RGBA
	original := v.bottomHUDArt != nil && v.spellbookFixed
	if !v.spellbookHeld {
		original = v.bottomHUDArt != nil
		if original {
			img = composeOriginalSpellBar(v.bottomHUDArt, nil, 0, bar, int(v.anim.Count()))
		} else {
			img = composeSpellBar(v.font, nil, 0, cols, bar, int(v.anim.Count()))
		}
		drawNoHeroText(img, v.words.NoHeroSelected, v.cardFont(), v.font, original)
		return img, bar.Min, true
	}
	if original {
		img = composeOriginalSpellBar(v.bottomHUDArt, v.spellbook, v.selectedSpell, bar, int(v.anim.Count()))
	} else {
		img = composeSpellBar(v.font, v.spellbook, v.selectedSpell, cols, bar, int(v.anim.Count()))
	}
	if v.quickSpells != nil {
		for i, cell := range bookCellRects(bar, cols) {
			if i >= len(v.spellbook) || v.spellbook[i].ID == 0 {
				continue
			}
			for slot, id := range v.quickSpells {
				if id == v.spellbook[i].ID {
					if original {
						if !v.spellbook[i].Unavailable {
							drawOriginalQuickSpellMark(img, v.cardFont(), cell.Sub(bar.Min), slot, id == v.selectedSpell)
						}
					} else {
						drawQuickSpellMark(img, v.font, cell.Sub(bar.Min), slot)
					}
				}
			}
		}
	}
	return img, bar.Min, true
}

// drawNoHeroText centres the install's own "no hero selected" line on an
// empty book strip. An empty line (an install without the string) draws
// nothing.
func drawNoHeroText(img *image.RGBA, line string, cardFont, font *text.Font, original bool) {
	f := font
	if original && cardFont != nil {
		f = cardFont
	}
	if line == "" || f == nil {
		return
	}
	w, h := f.Measure(line)
	b := img.Bounds()
	x0, dx := b.Min.X, b.Dx()
	if original {
		x0, dx = bookAtlasX(b), bookBarW
	}
	f.Draw(img, line, x0+(dx-w)/2, b.Min.Y+(b.Dy()-h)/2, spellbookText)
}

func drawQuickSpellMark(dst *image.RGBA, font *text.Font, cell image.Rectangle, slot int) {
	label := string([]byte{'F', byte('5' + slot)})
	w, h := font.Measure(label)
	if w <= 0 || h <= 0 || w+4 > cell.Dx() || h+4 > cell.Dy() {
		return
	}
	box := image.Rect(cell.Max.X-w-3, cell.Max.Y-h-3, cell.Max.X-1, cell.Max.Y-1)
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			dst.SetRGBA(x, y, spellbookFill)
		}
	}
	font.Draw(dst, label, box.Min.X+1, box.Min.Y+1, spellbookText)
}

// MAGIC-ICON-024 paints the atlas once and covers every unknown cell with
// SpellBack. Known spells retain the installed pixels, including their frame.
func composeOriginalSpellBar(art *BottomHUDArt, entries []SpellEntry, selected uint32, bar image.Rectangle, phase int) *image.RGBA {
	img := image.NewRGBA(image.Rectangle{Max: bar.Size()})
	drawBookGround(img, art)
	for i, box := range bookCellRects(bar, bookColumns) {
		box = box.Sub(bar.Min)
		if i >= len(entries) || entries[i].ID == 0 || entries[i].Unavailable {
			blit(img, art.UnknownSpell, box.Min.X, box.Min.Y)
			continue
		}
		if entries[i].ID == selected {
			liftPressedSlot(img, box)
		}
		if entries[i].Autocast {
			drawAutocastBorder(img, box, phase, autocastDashColor)
		}
	}
	return img
}

const pressedSlotLift = 45

func liftPressedSlot(img *image.RGBA, box image.Rectangle) {
	box = box.Intersect(img.Bounds())
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			c := img.RGBAAt(x, y)
			img.SetRGBA(x, y, color.RGBA{liftChannel(c.R), liftChannel(c.G), liftChannel(c.B), c.A})
		}
	}
}

func liftChannel(v uint8) uint8 {
	return v + uint8((255-int(v))*pressedSlotLift/100)
}

// quickSpellMarkInk is the digit ink of the mission spell bar's quick-spell
// marks. It is its own value: the pack readout's ink changed in SHOP-108 and
// this one did not.
var quickSpellMarkInk = color.RGBA{R: 200, G: 174, B: 84, A: 0xff}

func drawOriginalQuickSpellMark(dst *image.RGBA, font *text.Font, cell image.Rectangle, slot int, selected bool) {
	if font == nil {
		return
	}
	pad := 0
	if selected {
		pad = 2
	}
	label := string(rune('5' + slot))
	font.Draw(dst, label, cell.Min.X+pad+1, cell.Min.Y+pad+1, DamageNumeralShadowColor)
	font.Draw(dst, label, cell.Min.X+pad, cell.Min.Y+pad, quickSpellMarkInk)
}

// SetSpellbook hands the viewer the SELECTED unit's own book, replacing
// whatever it held before — pkg/game's own per-tick push, SetEntities' own
// frequency, and SetInventorySubject's own shape one door over: a Viewer
// method reached directly, not a ninth seam through MapOpener (plan T6's own
// "one new read-only seam").
//
// OWNER IS WHICH ENTITY THIS BOOK BELONGS TO. Entity id 0 is a real unit, so
// "no owner" is ClearSpellbook and never a zero owner. It is the WHOLE of
// "changing the selected unit clears it": a call naming a DIFFERENT owner than
// the one last recorded, or the first call after ClearSpellbook, clears
// whatever spell was selected, and a call repeating the same owner — the
// ordinary case, once a frame, for as long as the same unit stays selected —
// leaves the selection exactly where the player put it.
func (v *Viewer) SetSpellbook(owner uint32, entries []SpellEntry) {
	if !v.spellbookHeld || owner != v.spellbookOwner {
		v.selectedSpell = 0
		v.spellArmed = false
		v.itemCast = nil
	}
	v.spellbookHeld = true
	v.spellbookOwner, v.spellbook = owner, entries
	v.spellbookFixed = false
}

// ClearSpellbook is the frame with no unit selected: no bar stands and the
// selected spell is dropped.
func (v *Viewer) ClearSpellbook() {
	if v.spellbookHeld {
		v.selectedSpell = 0
		v.spellArmed = false
		v.itemCast = nil
	}
	v.spellbookHeld, v.spellbookOwner, v.spellbook = false, 0, nil
	v.spellbookFixed = false
}

// SetSpellbookCatalog keeps all 24 installed cells visible in the two-row bar.
// The ordinary setter remains a table-order/custom-book seam.
func (v *Viewer) SetSpellbookCatalog(owner uint32, entries []SpellEntry) {
	v.SetSpellbook(owner, entries)
	v.spellbookFixed = true
}

// SelectedUnits is the accepted display population in selection order. It is
// rebuilt, not retained by the session; callers receive a detached ID slice.
func (v *Viewer) SelectedUnits() []uint32 {
	present := presentSelected(v.sel, v.entities)
	ids := make([]uint32, len(present))
	for i, e := range present {
		ids[i] = e.ID
	}
	return ids
}

func (v *Viewer) bookTargetsPoint(id uint32) bool {
	for _, entry := range v.spellbook {
		if entry.ID == id {
			return entry.PointTarget
		}
	}
	return false
}

// SelectedUnit is the entity id the panel already describes: the first of
// the present selection, presentSelected's own order (panelSubject's own
// reading in panel.go), exported here for the one caller across the seam
// that needs it — pkg/game's per-tick push, which has no selection of its
// own to read and must ask this package for the one it holds.
//
// A viewer with no present entity answers false, mirroring
// inventoryEligible's own empty-selection reading.
func (v *Viewer) SelectedUnit() (uint32, bool) {
	present := presentSelected(v.sel, v.entities)
	if len(present) == 0 {
		return 0, false
	}
	return present[0].ID, true
}

// MapAutocast is the seam the autocast toggle leaves this package through:
// the entity whose setting is being changed, and the spell it is being set
// to — 0 for "clear it", which is the same "no spell" every id on this
// seam already reads as.
//
// IT IS SET ON THE VIEWER DIRECTLY rather than threaded through MapLoader's
// tuple, on SetSpellbook's own precedent immediately below ("a Viewer method
// reached directly, not a ninth seam"). The reason is the same one: this is not
// a gesture on the map and it produces no order — decide (command.go) never
// sees it, and no press reaches it — so the tuple every map order travels
// through would carry a value it never means on every other seam in it.
//
// A uint32 PAIR IS THE WHOLE PAYLOAD, MapStance's own reason: this package may
// import the render tier and no other, so it must not be able to name a
// simulation type. Both ids were minted on the far side and arrived in a
// MapEntity and a SpellEntry, so this package still cannot CONSTRUCT one.
type MapAutocast func(entity, spell uint32)

// SetAutocastSink hands the viewer the seam above, replacing whatever it held.
// A viewer never given one toggles nothing, which is every viewer cmd/mapview
// can build — the standalone viewer gains no autocast, by the same mechanism
// armAttack's own unexported-ness gives it no attack mode.
func (v *Viewer) SetAutocastSink(fn MapAutocast) { v.autocastSink = fn }

// toggleAutocast is the key's whole effect: the spell SELECTED in the book
// becomes the SELECTED unit's autocast spell, and pressing it again on the
// same spell clears it.
//
// FOUR THINGS MAKE IT A NO-OP, and each is a state the player can be in rather
// than an error: no sink, no unit selected, no spell selected in the book, and
// a unit the local player does not own. The ownership test is canArmAttack's
// own gate, borrowed rather than restated — putting an enemy's spell on repeat
// is exactly the class of order that gate exists to refuse.
//
// WHAT IT SENDS IS THE NEW SETTING AND NOT A TOGGLE, so the far side stores a
// value rather than interpreting one: the id when it is being turned on, 0 when
// it is being turned off. Which of the two this press is comes from the book
// entry's own Autocast flag, which the far side set on the previous push — so
// the state the player sees is the state the toggle reads, and the two cannot
// come apart across a frame.
func (v *Viewer) toggleAutocast() {
	if v.autocastSink == nil || v.selectedSpell == 0 {
		return
	}
	for i := range v.spellbook {
		if v.spellbook[i].ID == v.selectedSpell {
			v.toggleAutocastAt(i)
			return
		}
	}
}

// toggleAutocastAt is the spell-cell gesture's simulation boundary. The cell
// supplies the spell directly: right-clicking autocast does not also select a
// one-shot manual cast, and it never mutates the displayed book in advance of
// the next simulation push.
func (v *Viewer) toggleAutocastAt(index int) {
	if v.autocastSink == nil || index < 0 || index >= len(v.spellbook) {
		return
	}
	owner, ok := v.SelectedUnit()
	if !ok || !canArmAttack(v.sel, v.entities, v.localOwner) {
		return
	}
	e := v.spellbook[index]
	if e.ID == 0 || e.Unavailable {
		return
	}
	next := e.ID
	if e.Autocast {
		next = 0
	}
	v.autocastSink(owner, next)
}
