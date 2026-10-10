package ui

import (
	"fmt"
	"image"
	"strings"

	"againrom/pkg/render/text"
)

func mainTooltip(w Words, slot int, id string, font *text.Font) tooltipTarget {
	if slot < 0 || slot >= len(w.Hover) {
		return tooltipTarget{}
	}
	return tooltipTarget{tooltipText, id, tooltipLines(w.Hover[slot]), font}
}

// monsterSpellHoverLines is the card helper's monster spell list
// (TEXT-HOVERTEXT-052): the installed main.txt[192] heading joined with every
// known spell's spell.txt name, in ascending spell-ID order.
//
// It reports false for a subject the helper states nothing about — no
// installed heading, or no known spell — rather than a heading with nothing
// after it. known is the spellbook membership bitmask (bit i set for spell
// ID i, Spellbook's own convention); an id with no installed name is skipped
// rather than widening the list with a blank.
func monsterSpellHoverLines(w Words, known uint32) ([]string, bool) {
	if known == 0 || w.Hover[192] == "" {
		return nil, false
	}
	var names []string
	for id := 1; id < len(w.ItemSpellNames); id++ {
		if known&(uint32(1)<<uint(id)) == 0 || w.ItemSpellNames[id] == "" {
			continue
		}
		names = append(names, w.ItemSpellNames[id])
	}
	if len(names) == 0 {
		return nil, false
	}
	return []string{w.Hover[192] + strings.Join(names, ", ")}, true
}

// MonsterSpellHint is monsterSpellHoverLines under its own name, exported
// for an install-gated witness that has no live character panel to hover a
// creature's SPELLCASTER row on. See cmd/tooltipshot.
func MonsterSpellHint(w Words, known uint32) ([]string, bool) {
	return monsterSpellHoverLines(w, known)
}

func characterCornerTooltip(v TownCharacterView, w Words, p image.Point) tooltipTarget {
	corners := CharacterPaneCornerAt(v, p)
	if len(corners) == 0 {
		return tooltipTarget{}
	}
	c := corners[0]
	slot := -1
	switch c {
	case CharacterPaneBook:
		slot = 8
		if v.BookOpen {
			slot = 9
		}
	case CharacterPaneBackpack:
		slot = 10
		if v.PackOpen {
			slot = 11
		}
	case CharacterPaneMode:
		slot = 13
		if v.Statistics {
			slot = 12
		}
	case CharacterPaneMenu:
		slot = 14
	case CharacterPanePrev:
		slot = 52
		if v.ModeFlag {
			slot = 121
		}
	case CharacterPaneNext:
		slot = 53
		if v.ModeFlag {
			slot = 122
		}
	}
	return mainTooltip(w, slot, fmt.Sprintf("corner/%d/%d/%d", c, v.Subject.ID, slot), v.Font)
}

func panelTooltipSlot(field PanelField, s PanelSubject) int {
	switch field {
	case PanelFieldBody:
		return 155
	case PanelFieldReaction:
		return 156
	case PanelFieldMind:
		return 157
	case PanelFieldSpirit:
		return 158
	case PanelFieldHealth, PanelFieldHealthHeading:
		return 159
	case PanelFieldMana, PanelFieldManaHeading, PanelFieldManaCardHeading, PanelFieldManaCard:
		return 160
	case PanelFieldDamage:
		return 161
	case PanelFieldToHit:
		return 162
	case PanelFieldAbsorption:
		return 163
	case PanelFieldDefence:
		return 164
	case PanelFieldWeight:
		return 165
	case PanelFieldSight:
		return 166
	case PanelFieldMoveSpeed:
		return 167
	case PanelFieldSkill, PanelFieldSkillGeneral, PanelFieldSkillsHeading:
		return 168
	case PanelFieldResistance:
		if s.Char.Band == CharacterBandCreature {
			return 188
		}
		return 169
	case PanelFieldProtection, PanelFieldResistHeading,
		PanelFieldProtFire, PanelFieldProtWater, PanelFieldProtAir, PanelFieldProtEarth, PanelFieldProtAstral:
		return 169
	case PanelFieldExperience:
		return 170
	}
	if field >= PanelFieldSkillBlade && field <= PanelFieldSkillShooting {
		slot := 171 + int(field-PanelFieldSkillBlade)
		if s.Char.Mage {
			slot += 5
		}
		return slot
	}
	return -1
}

// Hit the same laid-out, disclosure-filtered rows the card actually paints.
// A hidden field or blank space between its cells does not acquire a hint.
func characterStatsTooltip(v TownCharacterView, w Words, p image.Point) tooltipTarget {
	if !v.HasSubject || !v.Statistics || !p.In(v.paneRect()) && !p.In(v.cardRect()) || v.cardFont() == nil {
		return tooltipTarget{}
	}
	l, font := v.statsLayout(), v.cardFont()
	items := panelItems(l, v.Subject)
	lines := layoutLines(l, font, items)
	p = p.Sub(v.cardRect().Min)
	for i, line := range lines {
		if i >= len(items) || p.Y < line.at.Y || p.Y >= line.at.Y+font.Height() {
			continue
		}
		field, hit := items[i].field, false
		left := line.at.X + line.leftIndent
		width := max(font.Advance(line.label), line.valueX+font.Advance(line.value))
		if p.X >= left && p.X < left+width {
			hit = true
		}
		if line.right {
			right := line.at.X + line.rightX
			rw := max(font.Advance(line.rightLabel), line.rightValueX+font.Advance(line.rightValue))
			if p.X >= right && p.X < right+rw {
				field, hit = items[i].rightField, true
			}
		}
		if hit {
			if field == PanelFieldSpellcaster && v.Subject.Char.Band == CharacterBandCreature {
				if lines, ok := monsterSpellHoverLines(w, v.Subject.KnownSpells); ok {
					return tooltipTarget{tooltipText, fmt.Sprintf("stat-spells/%d", v.Subject.ID), lines, font}
				}
			}
			return mainTooltip(w, panelTooltipSlot(field, v.Subject),
				fmt.Sprintf("stat/%d/%d", v.Subject.ID, field), font)
		}
	}
	return tooltipTarget{}
}

func (v *Viewer) additionalTooltipAt(p image.Point) tooltipTarget {
	if arrow, ok := v.packArrowAt(p.X, p.Y); ok {
		slot := 54
		if arrow > 0 {
			slot = 55
		}
		return mainTooltip(v.words, slot, fmt.Sprintf("pack-arrow/%d", arrow), v.font)
	}
	if bar, ok := v.packBarArea(); ok && p.In(bar) {
		return mainTooltip(v.words, 58, "pack-background", v.font)
	}
	view := v.characterPaneScreenView()
	if hint := characterCornerTooltip(view, v.words, p); hint.key() != "" {
		return hint
	}
	if hint := characterStatsTooltip(view, v.words, p); hint.key() != "" {
		return hint
	}
	if !view.Statistics {
		if r, ok := missionCardBoxRect(image.Pt(v.frameW, v.frameH)); ok && p.In(r) {
			view.Statistics, view.PaneRect, view.CornerArt = true, r, nil
			if hint := characterStatsTooltip(view, v.words, p); hint.key() != "" {
				return hint
			}
		}
	}
	return tooltipTarget{}
}

func (a *App) tooltipTarget() tooltipTarget {
	return a.tooltipTargetWithSurface(nil)
}

func (a *App) tooltipTargetWithSurface(known *TownSurfaceView) tooltipTarget {
	if a.flow == nil || a.cutscene != nil || a.cutsceneDrain {
		return tooltipTarget{}
	}
	if a.flow.screen == ScreenMap && a.flow.viewer != nil {
		return a.flow.viewer.tooltipTarget()
	}
	p, ok := a.windowToNativeFrame(a.tooltipPoint.X, a.tooltipPoint.Y)
	if !ok {
		return tooltipTarget{}
	}
	if a.flow.screen == ScreenPicker {
		return a.pickerTooltip(p)
	}
	if a.flow.screen == ScreenChargen {
		return a.chargenTooltip(p)
	}
	if a.flow.screen != ScreenTown {
		return tooltipTarget{}
	}
	if d, ok := a.flow.town.(TownDialogueScreen); ok {
		if _, showing := d.TownDialogue(); showing {
			return tooltipTarget{}
		}
	}
	if _, dragging := a.shopDragItemPresent(); dragging {
		return tooltipTarget{}
	}
	w := a.flow.words
	if world, ok := townWorldMapScreen(a.flow.town); ok {
		view := world.WorldMapView()
		if view.Returning {
			return tooltipTarget{}
		}
		if i, ok := WorldMapRegionAt(view, p); ok {
			m := view.Missions[i]
			if m.Object >= 0 && m.Object < len(w.SiteHints) {
				return tooltipTarget{tooltipText, fmt.Sprintf("site/%d", m.Object), tooltipLines(w.SiteHints[m.Object]), view.Font}
			}
		}
		return tooltipTarget{}
	}
	if shop, ok := townShopScreen(a.flow.town); ok {
		if shop.TipPanel.Covers(p) {
			return tooltipTarget{}
		}
		return shopTooltip(shop, w, p)
	}
	surface, inSurface := TownSurfaceView{}, false
	if known != nil {
		surface, inSurface = *known, true
	} else {
		surface, inSurface = townSurfaceScreen(a.flow.town)
	}
	if inSurface {
		if surface.Tip.Covers(p) {
			return tooltipTarget{}
		}
		if surface.BookView != nil && p.In(shopTableRegion) {
			return shopTooltip(*surface.BookView, w, p)
		}
		if hint := characterCornerTooltip(surface.Hero, w, p); hint.key() != "" {
			return hint
		}
		if hint := characterStatsTooltip(surface.Hero, w, p); hint.key() != "" {
			return hint
		}
		if lines, ok := TownCandidateHoverLines(surface, p); ok {
			return tooltipTarget{tooltipText, fmt.Sprintf("candidate-item/%d", surface.Candidate.Subject.ID), lines, surface.Font}
		}
		if surface.Kind == TownSurfaceTavern {
			candidate := tavernCandidateStats(surface.Candidate, surface.TavernArt)
			if hint := characterStatsTooltip(candidate, w, p); hint.key() != "" {
				return hint
			}
		}
		if control, ok := schoolControlAt(surface, p); ok && control.Index >= 0 && control.Index < 10 {
			// schoolControlAt already maps the painted mask to semantic skill
			// order. Applying the original visual permutation again swaps skills.
			slot := 171 + control.Index
			return mainTooltip(w, slot, fmt.Sprintf("school/%d", slot), surface.Font)
		}
		return tooltipTarget{}
	}
	if square, ok := townSquareView(a.flow.town); ok && !square.Tip.Covers(p) {
		if slot, ok := square.Scene.TipAt(p); ok {
			return mainTooltip(w, slot, fmt.Sprintf("town/%d", slot), square.Font)
		}
	}
	return tooltipTarget{}
}

func shopTooltip(v ShopScreenView, w Words, p image.Point) tooltipTarget {
	character := shopCharacterView(v)
	if hint := characterCornerTooltip(character, w, p); hint.key() != "" {
		return hint
	}
	if hint := characterStatsTooltip(character, w, p); hint.key() != "" {
		return hint
	}
	if lines, ok := ShopHoverLines(v, p); ok {
		return tooltipTarget{tooltipText, fmt.Sprintf("shop-item/%d/%d/%d/%d", character.Subject.ID, v.Chosen, v.PackOffset, v.ShelfOffset), lines, v.Font}
	}
	c, ok := shopScreenControlAt(v, p)
	if !ok {
		return tooltipTarget{}
	}
	slot := -1
	switch c.Kind {
	case ShopControlArrowUp:
		slot = 56
	case ShopControlArrowDown:
		slot = 57
	case ShopControlPackLeft:
		slot = 54
	case ShopControlPackRight:
		slot = 55
	case ShopControlShelfPick:
		if c.Index >= 0 && c.Index < 4 {
			slot = 62 + c.Index
		}
	case ShopControlMerchant:
		slot = 61
	case ShopControlShelfCell:
		slot = 60
	case ShopControlTableCell:
		if !v.Book {
			slot = 59
		}
	case ShopControlPackCell:
		slot = 58
		if c.Index >= 0 && c.Index < len(v.Pack) && v.Pack[c.Index].Money {
			slot = 74
		}
	}
	return mainTooltip(w, slot, fmt.Sprintf("shop/%d/%d", c.Kind, c.Index), v.Font)
}

func (a *App) chargenTooltip(p image.Point) tooltipTarget {
	c := a.flow.chargen
	l := c.layout()
	if c == nil || l == nil || c.setup.PreCreate == nil || c.setup.PreCreate.Art == nil || c.TipPanel().Covers(p) {
		return tooltipTarget{}
	}
	font, w := c.setup.PreCreate.Art.Font, a.flow.words
	slot, id := -1, chargenNone
	pick := func(n *int) {
		if n != nil {
			slot = *n
		}
	}
	pre := &l.PreCreate
	if c.stage == PreCreateStage {
		id = preControlAt(c, p)
		switch {
		case id == chargenName:
			pick(pre.Name.Tooltip)
		case id >= chargenLevel0 && id <= chargenLevel2:
			pick(pre.Levels[id-chargenLevel0].Tooltip)
		case id >= chargenChoice0 && id <= chargenChoice3:
			pick(pre.Heroes[id-chargenChoice0].Tooltip)
		case id == chargenForward:
			pick(pre.Forward.Tooltip)
		case id == chargenBack:
			pick(pre.Back.Tooltip)
		}
	} else if c.stage == DetailedStage {
		d := &l.Detail
		id = detailedControlAt(c, p)
		switch {
		case id >= chargenSkill0 && id <= chargenSkill4:
			_, class := c.heroParts(c.preChoice)
			slot = d.Classes[class].Tooltip + int(id-chargenSkill0)
		case id >= chargenStatMinus0 && id <= chargenStatMinus3:
			stat := int(id - chargenStatMinus0)
			return chargenStatTooltip(c, stat, false, id, font)
		case id >= chargenStatPlus0 && id <= chargenStatPlus3:
			stat := int(id - chargenStatPlus0)
			return chargenStatTooltip(c, stat, true, id, font)
		case id == chargenName:
			pick(pre.Name.Tooltip)
		}
		for i, g := range d.Stats.Value {
			r := g.Rectangle()
			if p.In(r) {
				if lines, ok := chargenAttributeLines(w, c, i); ok {
					return tooltipTarget{tooltipText, fmt.Sprintf("chargen/%d/value/%d/%d", c.stage, i, c.statValue[i]), lines, font}
				}
			}
			if p.In(image.Rect(d.Stats.LabelLeft, r.Min.Y, r.Min.X, r.Max.Y)) {
				slot = d.Stats.LabelTooltip + i
			}
		}
		if p.In(d.Stats.Pool.Rectangle()) {
			slot = d.Stats.PoolTooltip
		}
		if card := d.Card.Rect.Rectangle(); slot == -1 && p.In(card) {
			return characterStatsTooltip(c.CardView(), w, p)
		}
	}
	return mainTooltip(w, slot, fmt.Sprintf("chargen/%d/%d/%d", c.stage, id, slot), font)
}

// chargenAttributeLines is "label = value" (TEXT-082).
func chargenAttributeLines(w Words, c *Chargen, stat int) ([]string, bool) {
	l := c.layout()
	if l == nil {
		return nil, false
	}
	slot := l.Detail.Stats.ValueLabel + stat
	if slot >= len(w.Hover) || w.Hover[slot] == "" || stat < 0 || stat >= len(c.statValue) {
		return nil, false
	}
	return tooltipLines(fmt.Sprintf("%s = %d", w.Hover[slot], c.statValue[stat])), true
}

// chargenStatTooltip is the raise cost and lower refund text (TEXT-082).
func chargenStatTooltip(c *Chargen, stat int, raise bool, id chargenControl, font *text.Font) tooltipTarget {
	s, ok := c.StatStepText(stat, raise)
	if !ok {
		return tooltipTarget{}
	}
	return tooltipTarget{tooltipText, fmt.Sprintf("chargen/%d/step/%d/%d", c.stage, id, c.statValue[stat]), []string{s}, font}
}

// pickerTooltip is the map list's hover (TEXT-083): the cursor x alone picks
// the row's description or the installed caption of a metadata column.
func (a *App) pickerTooltip(p image.Point) tooltipTarget {
	if a.flow.picker == nil || a.tooltip.baseFont == nil || !a.flow.picker.InList(p) {
		return tooltipTarget{}
	}
	column := MapListColumnAt(p.X)
	if column > 0 {
		lines, ok := mapListColumnLines(a.flow.words, column)
		if !ok {
			return tooltipTarget{}
		}
		return tooltipTarget{tooltipText, fmt.Sprintf("maplist-column/%d", column), lines, a.tooltip.baseFont}
	}
	row, ok := a.flow.picker.RowAt(p)
	if !ok {
		return tooltipTarget{}
	}
	rows := a.flow.picker.Rows()
	if row < 0 || row >= len(rows) {
		return tooltipTarget{}
	}
	lines, ok := mapListHoverLines(a.flow.words, rows[row])
	if !ok {
		return tooltipTarget{}
	}
	return tooltipTarget{tooltipText, fmt.Sprintf("maplist/%d", row), lines, a.tooltip.baseFont}
}

// mapListHoverLines is the description column's hint: the row's own decoded
// description, null for a row whose metadata did not decode.
func mapListHoverLines(w Words, row PickerRow) ([]string, bool) {
	if row.Description == "" {
		return nil, false
	}
	return tooltipLines(row.Description), true
}

// mapListColumnLines is a metadata column's hint: dialogs.txt[134], [135] or
// [136] for the size column and the two record-word columns.
func mapListColumnLines(w Words, column int) ([]string, bool) {
	caption := ""
	switch column {
	case 1:
		caption = w.MapListSize
	case 2, 3:
		caption = w.MapListColumns[column-2]
	}
	if caption == "" {
		return nil, false
	}
	return tooltipLines(caption), true
}

// MapListHint is one column's hint, exported for cmd/tooltipshot; column 0 is
// row's description.
func MapListHint(w Words, row PickerRow, column int) ([]string, bool) {
	if column == 0 {
		return mapListHoverLines(w, row)
	}
	return mapListColumnLines(w, column)
}
