package game

import (
	"fmt"
	"math"
	"strings"

	"againrom/pkg/audio"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/render/text"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func (t *townScreen) townCharacterView() ui.TownCharacterView {
	panes := t.art.characterPanes()
	// TWO FONTS, NOT ONE (1027 B1). Font is the shell chrome's — the chevrons
	// and the mode box, both sized for font1's 16x15 cell. CardFont is the
	// statistics card's, and it is tipFont's already-resolved font2 rather than
	// a second resolution path: the card is the same 160x242 CompactPanelLayout
	// the character generator composes at font2, and at font1 it truncates
	// every right-column value away and pushes its last two rows off the bottom
	// of the card. THE NINE CORNER BITMAPS REACH THE TOWN PANE TOO. `TOWN-353`
	// establishes that the mission frame and the shop view carry ONE instance
	// of this widget, so the shipped corner art is a property of the widget and
	// not of the screen it is re-parented into. With `sess` at
	// CharacterPaneShop, `TOWN-354`'s own art gates draw rect B's spellbook,
	// rect C's mode bitmap and the party picker's ar1/ar2, and leave rects A
	// and F unpainted. Without this push the town pane fell back to the
	// authored chevrons and drew rect C nothing at all, because this project's
	// stand-in chrome is not painted over the figure crop
	// (drawCharacterPaneCorners' own rule) -- so the mode control the story
	// moves to rect C would have been invisible on every town screen.
	//
	// The same inspection book can be toggled in every city room.
	v := ui.TownCharacterView{Statistics: t.townStats, Font: t.in.Font.Value(), CardFont: t.in.tipFont(),
		FigurePane: panes.Figure, StatsPane: panes.Stats,
		CornerArt: t.art.characterCorners(), Session: ui.CharacterPaneShop, BookOpen: t.shopBook}
	party := t.shopParty()
	t.clampTownMember()
	shown := t.shopMemberIndex()
	v.Member, v.MemberCount = townPickerPosition(party, shown)
	if len(party) == 0 {
		return v
	}
	m := party[shown]
	v.HasSubject = true
	v.Subject = partyPanelSubject(m, t.in.Table, t.in.Words)
	v.Subject.Unplaced = true
	v.Subject.Selected = 1
	v.Figure = t.shopFigure(shown)
	return v
}

func (t *townScreen) stepTownMember(step int) ui.TownAction {
	return t.shopStepMember(step)
}

type tavernCandidateKind uint8

const (
	tavernCandidateNone tavernCandidateKind = iota
	tavernCandidateOffer
	tavernCandidateMercenary
)

// tavernCandidateKey survives list compaction: mission offers use their
// chapter-local offer index (a kept speaker its negative one) and mercenaries
// use their researched type id.
type tavernCandidateKey struct {
	kind tavernCandidateKind
	id   int
}

type tavernCandidate struct {
	key   tavernCandidateKey
	cell  ui.TownSurfaceCell
	offer TownOffer
	merc  TavernOffer
}

// tavernCandidates puts mercenary cells before visible talk-only cells.
// Talkers keep their registry order, including kept speakers (DIV-1529, DIV-2170).
func (t *townScreen) tavernCandidates() []tavernCandidate {
	offers := t.tavernVisibleOffers()
	mercenaries := t.tavernMercenaries()
	out := make([]tavernCandidate, 0, len(offers)+len(mercenaries))
	art := t.in.TownTavernArt.Value()
	for _, o := range mercenaries {
		cell := ui.TownSurfaceCell{Detail: fmt.Sprintf("%d/%d", o.Count, o.Capacity),
			Price: ui.GroupDigits(int64(o.Price)), Semantic: fmt.Sprintf("Mercenary %d", o.Type),
			Enabled: o.Affordable, Hired: o.Hired, Portrait: true}
		if art != nil && o.Type > 0 && o.Type < len(art.Units) {
			cell.Picture = art.Units[o.Type]
			cell.Frames = art.UnitFrames[o.Type]
		}
		out = append(out, tavernCandidate{key: tavernCandidateKey{kind: tavernCandidateMercenary, id: o.Type}, cell: cell, merc: o})
	}
	for _, o := range offers {
		cell := ui.TownSurfaceCell{Label: fmt.Sprintf("NPC %d", o.NPC),
			Semantic: fmt.Sprintf("NPC %d", o.NPC),
			Enabled:  true, Portrait: true, TalkOnly: true}
		if frames := tavernTalkFrames(art, t.tavernTalkObject(o.NPC)); len(frames) > 0 {
			cell.Picture, cell.Frames = frames[0], frames
		}
		out = append(out, tavernCandidate{
			key: tavernCandidateKey{kind: tavernCandidateOffer, id: o.Index}, offer: o,
			cell: cell,
		})
	}
	return out
}

// activateTavernSelection is the inn view's activation store (TOWN-468):
// position 0 when the roster has a mercenary cell, else no selection, which
// leaves every cell unpainted until a click stores one.
func (t *townScreen) activateTavernSelection(candidates []tavernCandidate) {
	t.tavernSelection = tavernCandidateKey{}
	if len(candidates) > 0 && candidates[0].key.kind == tavernCandidateMercenary {
		t.tavernSelection = candidates[0].key
	}
}

func (t *townScreen) clearTavernDetail() {
	t.tavernDetailType, t.tavernDetail = 0, ui.TownCharacterView{}
	t.tavernDetailMask, t.tavernDetailInfo, t.tavernDetailPixels = nil, [12][]string{}, nil
	t.tavernDetailText = nil
}

func (t *townScreen) selectedTavernCandidate() (tavernCandidate, bool) {
	_, selected, ok := t.tavernSnapshot(t.tavernCandidates())
	return selected, ok
}

func (t *townScreen) tavernSnapshot(candidates []tavernCandidate) ([]ui.TownSurfaceCell, tavernCandidate, bool) {
	out := make([]ui.TownSurfaceCell, len(candidates))
	if t.tavernSelection.kind == tavernCandidateNone {
		// The original holds -1 only while the roster has no mercenary cell.
		t.activateTavernSelection(candidates)
	}
	found := t.tavernSelection.kind == tavernCandidateNone
	var selected tavernCandidate
	for i := range candidates {
		out[i] = candidates[i].cell
		out[i].Key = fmt.Sprintf("%d:%d", candidates[i].key.kind, candidates[i].key.id)
		if candidates[i].key == t.tavernSelection {
			out[i].Selected = true
			selected = candidates[i]
			found = true
		}
	}
	if !found {
		t.clearTavernDetail()
		if t.tavernSelection.kind == tavernCandidateOffer {
			return out, selected, false
		}
		t.activateTavernSelection(candidates)
		if len(out) > 0 && t.tavernSelection.kind != tavernCandidateNone {
			out[0].Selected, selected, found = true, candidates[0], true
		}
	}
	return out, selected, found && t.tavernSelection.kind != tavernCandidateNone
}

func (t *townScreen) tavernSurfaceCells() []ui.TownSurfaceCell {
	cells, _, _ := t.tavernSnapshot(t.tavernCandidates())
	return cells
}

func (t *townScreen) selectedMercenary() (TavernOffer, bool) {
	c, ok := t.selectedTavernCandidate()
	if !ok || c.key.kind != tavernCandidateMercenary {
		return TavernOffer{}, false
	}
	return c.merc, true
}

// tavernCandidateDetail builds one real generated candidate through the same
// subject, equipment, doll compositor and item-popup producers used by the
// character pane and inventory. It has no mutation handle back to the member.
func (t *townScreen) tavernCandidateDetail(typ int) (ui.TownCharacterView, *ui.SlotMask, [12][]string, bool) {
	if t.tavernDetailType == typ && t.tavernDetail.HasSubject {
		return t.tavernDetail, t.tavernDetailMask, t.tavernDetailInfo, true
	}
	var info [12][]string
	members, ok := t.buildMercenarySquad(typ, 1)
	if !ok || len(members) != 1 {
		return ui.TownCharacterView{}, nil, info, false
	}
	m := members[0]
	subject := partyPanelSubject(m, t.in.Table, t.in.Words)
	// Generated template ids such as NPC06_1 are source keys, not the
	// installed display name. A human template's own TypeID, rather than the
	// equipment-derived body Class stored on PartyMember, indexes UnitNames.
	// Let the shared panel projection resolve that localized name and then
	// make it explicit so PanelFieldName's actual-name-first rule is obeyed.
	nameIndex := int(m.Class)
	if typ > 2 {
		template := fmt.Sprintf("NPC%02d_%d", typ, mercenaryLevel(t.sess.Town.Chapter()))
		if i := data.FindHumanByName(t.in.Table.Humans, template); i != data.NotFound {
			if def, err := data.NewHumanDef(template, t.in.Table.Humans.EntryParams(i)); err == nil {
				nameIndex = int(def.TypeID)
			}
		}
	}
	subject.UnitNameIndex, subject.Char.UnitNameIndex = nameIndex, nameIndex
	subject.Name = ""
	subject.Char.Name = ""
	if name, named := ui.PanelSubjectName(subject); named {
		subject.Name = name
		subject.Char.Name = name
	}
	if name, ok := t.in.Table.Mods.CharacterName(m.Name); ok {
		subject.Name, subject.Char.Name = name, name
	}
	subject.Role = t.in.Words.PanelCaptions[mainSlotMercenary]
	// A candidate is a live actor whose sheet prints its whole sight word
	// (HERO-104); the recompute keeps only its high byte, the whole cells.
	if c := subject.Char; c.Mind+c.Reaction > 0 {
		raw := data.SightSubCells(int32(c.Mind), int32(c.Reaction))
		raw += int32(c.Sight)<<8 - raw&^0xff
		if raw > 0 && raw <= 0xffff {
			subject.Char.Sight256 = uint16(raw)
		}
	}
	subject.Unplaced, subject.Selected = true, 1

	var src entrySource
	if t.in.Archives != nil {
		src = t.in.Archives.Containers
	}
	eq := mapload.EquipmentFromParty(m)
	composed := composeMemberPortrait(src, t.in.Units, uint32(typ), eq, m)
	items := mapload.MemberItemEquipment(m, t.in.Table)
	for i, item := range items {
		if item.Code != 0 {
			info[i] = itemInstanceInfoLines(item, t.in.Table, t.in.Words)
		}
	}
	panes := t.art.characterPanes()
	v := ui.TownCharacterView{Subject: subject, HasSubject: true, Figure: composed.Figure,
		Font: t.in.Font.Value(), CardFont: t.in.tipFont(), FigurePane: panes.Figure, StatsPane: panes.Stats}
	hoverMask := composed.HoverSlotMask
	if hoverMask == nil {
		hoverMask = composed.SlotMask
	}
	t.tavernDetailType, t.tavernDetail, t.tavernDetailMask, t.tavernDetailInfo = typ, v, hoverMask, info
	t.tavernDetailText = text.Record(func() {
		t.tavernDetailPixels = ui.RenderTownCandidateInspection(v, t.in.TownTavernArt.Value())
	})
	return v, hoverMask, info, true
}

func (t *townScreen) schoolSurfaceCells() []ui.TownSurfaceCell {
	fighter, mage := data.SkillNames(false), data.SkillNames(true)
	out := make([]ui.TownSurfaceCell, 0, 10)
	party := t.shopParty()
	i := t.shopMemberIndex()
	var member *mapload.PartyMember
	if i >= 0 && i < len(party) {
		member = &party[i]
	}
	for class := 0; class < 2; class++ {
		names := fighter
		if class == 1 {
			names = mage
		}
		for slot, name := range names {
			level := int32(0)
			enabled := member != nil && member.Mage == (class == 1) && t.schoolPanelClass() == class
			if member != nil {
				level = member.Hero.Skill[slot+1]
			}
			price := heroSkillPrice(level)
			if member != nil {
				price = memberSchoolPrice(*member, slot+1)
			}
			out = append(out, ui.TownSurfaceCell{Label: name,
				Detail: fmt.Sprintf("level %d / %d", level, price), Enabled: enabled})
		}
	}
	if t.schoolCell < schoolNoSelection || t.schoolCell >= len(out) {
		t.schoolCell = schoolNoSelection
	}
	if t.schoolCell >= 0 {
		out[t.schoolCell].Selected = true
	}
	return out
}

// schoolNoSelection is townScreen.schoolCell's value for "no skill is pending".
// It is the -1 TOWN-138 reads the original writing into the school view's own
// selected-slot field (TOWN-GENERAL-107) when the party picker steps.
const schoolNoSelection = -1

// clearSchoolSelection discards the pending skill and, with it, the price the
// Train button quotes: selectedSchoolSlot answers false for a cleared cell, so
// the button reports 0 and is disabled until a skill is clicked again.
//
// IT IS GATED ON THE SCHOOL ROOM because shopStepMember is one routine serving
// three rooms here and three separate routines in the original. TOWN-138 reads
// the school's own step routine; the shop's and the tavern's pending state is
// not its subject and is not touched. The gate is the same t.room == roomSchool
// test TownSurfaceClick uses to assign schoolCell, so the two halves of the
// selection agree about which room owns it.
//
// TOWN-138 also names five-dword arrays in the same routine. This build has no
// counterpart for them; the selected-slot and price fields are the two that
// map. docs/DIVERGENCES.md DIV-143 carries the residual.
func (t *townScreen) clearSchoolSelection() {
	if t.room == roomSchool {
		t.schoolCell = schoolNoSelection
		t.resetTownSpeech()
	}
}

func (t *townScreen) selectedSchoolSlot() (int, int, bool) {
	party := t.shopParty()
	i := t.shopMemberIndex()
	if i < 0 || i >= len(party) || t.schoolCell < 0 || t.schoolCell >= 10 {
		return 0, 0, false
	}
	wantMage := t.schoolCell >= 5
	if party[i].Mage != wantMage || t.schoolPanelClass() != t.schoolCell/5 {
		return 0, 0, false
	}
	slot := t.schoolCell%5 + 1
	price := memberSchoolPrice(party[i], slot)
	return slot, price, price > 0
}

const (
	tavernButtonSleep = iota
	tavernButtonHire
	tavernButtonTalk
	tavernButtonExit
)

// TownSurface resolves one full tavern or school frame.
func (t *townScreen) TownSurface() ui.TownSurfaceView {
	v := ui.TownSurfaceView{Font: t.in.Font.Value(), CardFont: t.in.tipFont(), HiredLabel: t.in.Words.TavernHired, Hero: t.townCharacterView(), HoverCell: -1, SchoolClass: -1}
	// The school and the tavern have no spellbook: no corner, key or open page.
	v.Hero.NoBook, v.Hero.BookOpen = true, false
	if t.room == roomSchool || t.room == roomTalk && t.dialogueBuilding == TownSchool {
		v.Kind, v.Title, v.Cells = ui.TownSurfaceSchool, "", t.schoolSurfaceCells()
		v.SchoolArt = t.in.TownSchoolArt.Value()
		v.Scene = roomScene{t.schoolPage()}
		v.SchoolClass = t.schoolPanelClass()
		v.SchoolColumnFrame, v.SchoolColumnSet = t.schoolColumnShown()
		if shine := t.schoolShine(); shine.Stamped {
			v.SchoolIdleShine, v.SchoolIdleSlot = true, shine.Slot(v.SchoolClass)
		}
		_, price, ok := t.selectedSchoolSlot()
		// Two buttons, not three: the school's shipped area picture bakes two
		// wells and this build no longer draws a Talk box over them (1017;
		// owner ruling quoted in docs/1017-button-frames/spec.md — a mission
		// offer fires once on entering the room instead).
		v.Buttons = []ui.TownSurfaceButton{
			{Label: t.in.Words.SchoolTrain, Value: ui.GroupDigits(int64(price)), Enabled: ok && price <= t.sess.Town.Gold()},
			{Label: t.in.Words.SchoolExit, Value: ui.GroupDigits(int64(t.sess.Town.Gold())), Enabled: true},
		}
		v.Tip = t.roomTipView(roomSchool, t.schoolTip)
		return v
	}
	candidates := t.tavernCandidates()
	v.Kind, v.Title = ui.TownSurfaceTavern, "TAVERN"
	cells, selected, hasSelection := t.tavernSnapshot(candidates)
	v.Cells = cells
	v.TavernArt = t.in.TownTavernArt.Value()
	v.Scene = roomScene{t.tavernPage()}
	o, merc := selected.merc, hasSelection && selected.key.kind == tavernCandidateMercenary
	value := ""
	if merc {
		// TOWN-469: the tavern button panel numbers are grouped.
		value = ui.GroupDigits(int64(o.Price))
		if candidate, mask, info, ok := t.tavernCandidateDetail(o.Type); ok {
			v.Candidate, v.CandidateSlotMask, v.CandidateSlotInfo = candidate, mask, info
			v.CandidatePixels, v.CandidateText = t.tavernDetailPixels, t.tavernDetailText
		}
	} else if hasSelection && selected.key.kind == tavernCandidateOffer {
		t.renderTavernTalkDetail(&v, selected.offer.NPC)
	}
	v.RosterUnpainted = !hasSelection && (len(candidates) == 0 || candidates[0].key.kind != tavernCandidateMercenary)
	// TOWN-391/392: no selected squad has no hire caption or price; an
	// active squad selects the install's hire/fire word, not the constructor
	// caption. The owner-added Sleep control is intentionally separate.
	hireLabel := ""
	if merc {
		hireLabel = t.in.Words.TavernHire
		if o.Hired {
			hireLabel = t.in.Words.TavernFire
		}
	}
	talk := hasSelection && (selected.key.kind != tavernCandidateOffer || t.tavernOfferCanTalk(selected.offer))
	v.Buttons = []ui.TownSurfaceButton{
		{Label: t.in.Words.TavernSleep, Enabled: t.sess.Shop != nil && t.in.Table != nil},
		{Label: hireLabel, Value: value, Enabled: merc && (o.Hired || o.Affordable)},
		{Label: t.in.Words.TavernTalk, Enabled: talk},
		{Label: t.in.Words.TavernExit, Value: ui.GroupDigits(int64(t.sess.Town.Gold())), Enabled: true},
	}
	v.Tip = t.roomTipView(roomTavern, t.tavernTip)
	return v
}

// TownSurfacePress requests the school's skill member on a left press in an
// enabled skill column, unless that member's instance plays. The member
// follows the column's skill slot and class, fighter cells 0..4 and mage
// cells 5..9 (VIDEO-SFX-059). The press changes no state.
func (t *townScreen) TownSurfacePress(c ui.TownSurfaceControl) {
	if t.room != roomSchool || c.Kind != ui.TownSurfaceControlCell || c.Index < 0 || c.Index >= 10 || t.schoolTrainingBusy() {
		return
	}
	if cells := t.TownSurface().Cells; c.Index >= len(cells) || !cells[c.Index].Enabled {
		return
	}
	t.schoolSounds.RequestFor("school-skill", t.roomSoundPlayer(audio.EffectsChannel), t.in.SoundBank, ui.ChargenSkillSounds[c.Index/5][c.Index%5])
}

// TownSurfaceClick is the shared shell's one mutation door.
func (t *townScreen) TownSurfaceClick(c ui.TownSurfaceControl, double bool) ui.TownAction {
	if t.room == roomSchool && t.schoolTrainingBusy() {
		return ui.TownAction{}
	}
	switch c.Kind {
	case ui.TownSurfaceControlBook:
		return ui.TownAction{}
	case ui.TownSurfaceControlBookPage:
		return ui.TownAction{}
	case ui.TownSurfaceControlPrevious:
		return t.stepTownMember(-1)
	case ui.TownSurfaceControlNext:
		return t.stepTownMember(+1)
	case ui.TownSurfaceControlMode:
		if len(t.shopParty()) == 0 {
			return ui.TownAction{}
		}
		t.townStats = !t.townStats
		return ui.TownAction{}
	case ui.TownSurfaceControlCell:
		if t.room == roomSchool {
			cells := t.TownSurface().Cells
			if c.Index < 0 || c.Index >= len(cells) || !cells[c.Index].Enabled {
				return ui.TownAction{}
			}
			t.schoolCell = c.Index
		} else {
			candidates := t.tavernCandidates()
			if c.Index < 0 || c.Index >= len(candidates) {
				return ui.TownAction{}
			}
			if t.tavernSelection != candidates[c.Index].key {
				t.clearTavernDetail()
			}
			t.tavernSelection = candidates[c.Index].key
		}
		if double && t.room == roomTavern {
			if o, ok := t.selectedMercenary(); ok {
				t.pressMercenary(o.Type)
				return ui.TownAction{}
			}
			return t.townSurfaceButton(tavernButtonTalk)
		}
		return ui.TownAction{}
	case ui.TownSurfaceControlButton:
		return t.townSurfaceButton(c.Index)
	}
	return ui.TownAction{}
}

// townSurfaceButton dispatches a button press by its own room's index. The
// index a button sits at is no longer shared between rooms (1017): the school
// has two buttons (Train, Exit) and the tavern four (Sleep, Hire/Fire, Talk,
// Exit), so EXIT is index 1 for the school and index 3 for the tavern. The
// school's Talk case is gone with the box it fired; openBuildingDialogue
// itself stays for the legacy row-list Choose() path.
func (t *townScreen) townSurfaceButton(i int) ui.TownAction {
	if t.room == roomSchool {
		if t.schoolTrainingBusy() {
			return ui.TownAction{}
		}
		switch i {
		case 0:
			slot, _, ok := t.selectedSchoolSlot()
			if !ok {
				return ui.TownAction{}
			}
			gold := t.sess.Town.Gold()
			msg := t.trainHeroSkill(slot)
			if t.sess.Town.Gold() < gold {
				t.speakSchoolTraining(slot)
			}
			return ui.TownAction{Msg: trainingPostedLine(msg), Info: true}
		case 1:
			t.Back()
		}
		return ui.TownAction{}
	}
	if t.room == roomTavern {
		switch i {
		case tavernButtonSleep:
			t.restockShop()
			return ui.TownAction{}
		case tavernButtonHire:
			if o, ok := t.selectedMercenary(); ok {
				t.pressMercenary(o.Type)
				return ui.TownAction{}
			}
			return ui.TownAction{}
		case tavernButtonTalk:
			candidate, ok := t.selectedTavernCandidate()
			if !ok {
				return ui.TownAction{}
			}
			if candidate.key.kind == tavernCandidateOffer {
				if !t.tavernOfferCanTalk(candidate.offer) {
					return ui.TownAction{}
				}
				t.composeShopFaces()
				t.openOfferDialogue(TownTavern, candidate.offer, candidate.offer.NPC)
				return ui.TownAction{}
			}
			if candidate.key.kind == tavernCandidateMercenary {
				t.openMercenaryDialogue(candidate.merc.Type)
				return ui.TownAction{}
			}
		case tavernButtonExit:
			t.Back()
		}
	}
	return ui.TownAction{}
}

// trainingPostedLine keeps training and SAV fault lines; player refusals post none (TAVERN-LINES-022).
func trainingPostedLine(msg string) string {
	for _, prefix := range []string{"trained ", "SAV training", "training requires", "training produced", "training price"} {
		if strings.HasPrefix(msg, prefix) {
			return msg
		}
	}
	return ""
}

func (t *townScreen) restockShop() bool {
	if t == nil || t.sess == nil || t.sess.Town == nil || t.sess.Shop == nil {
		return false
	}
	seed := shopSeed(t.sess.Town.Chapter(), t.sess.Town.finishedCount(), t.sess.Shop.Ceiling())
	return t.sess.Shop.RestockWith(t.in.Table, seed, t.draws.stream(random.ShopStock))
}

func heroSkillPrice(level int32) int {
	return int(math.Pow(1.1, float64(level)) * 200)
}

func (t *townScreen) trainHeroSkill(slot int) string {
	return t.withCityBookMutation(func(n *townScreen) ui.TownAction {
		return ui.TownAction{Msg: n.trainHeroSkillValues(slot)}
	}).Msg
}

func (t *townScreen) trainHeroSkillValues(slot int) string {
	if slot < 0 || slot >= data.SkillSlots {
		return "that skill slot is invalid"
	}
	party := t.shopParty()
	i := t.shopMemberIndex()
	if i < 0 || i >= len(party) {
		return "there is no hero to train"
	}
	m := &party[i]
	if handled, message := t.trainOriginalCityFighter(slot, m); handled {
		return message
	}
	if mapload.HasSourceActor(*m) {
		next, price, err := mapload.TrainSourceParty(*m, slot)
		if err != nil {
			return "cannot train that skill"
		}
		if !t.sess.Town.spend(int(price)) {
			return fmt.Sprintf("training costs %d", price)
		}
		*m = next
		refreshDerivedPartyBook(m, t.in.Table)
		t.composeShopFaces()
		return fmt.Sprintf("trained %s to %d for %d", schoolSkillName(m.Mage, slot), m.Hero.Skill[slot], price)
	}
	price := heroSkillPrice(m.Hero.Skill[slot])
	if t.room == roomSchool && price > 0 && price <= t.sess.Town.Gold() {
		// TOWN-379: local admission arms before the purchase, not on a
		// successful reply. A refusal leaves any running animation alone.
		t.schoolPage().Event("train")
	}
	if !t.sess.Town.spend(price) {
		return fmt.Sprintf("training costs %d", price)
	}
	if m.Carry == nil {
		carried := mapload.MemberCarriedItems(*m, t.in.Table)
		worn := mapload.MemberItemEquipment(*m, t.in.Table)
		var carriedCodes []uint16
		if len(carried) != 0 {
			carriedCodes = make([]uint16, len(carried))
		}
		for i := range carried {
			carriedCodes[i] = carried[i].Code
		}
		if len(carried) == 0 {
			carried = nil
		}
		var wornCodes [sim.EquipSlots]uint16
		for i := range worn {
			wornCodes[i] = worn[i].Code
		}
		m.Carry = &mapload.Carry{SkillXP: m.Hero.Reward().SkillXP,
			Items: carriedCodes, ItemInstances: carried,
			Equipped: wornCodes, EquippedItems: worn}
	}
	m.Hero.Skill[slot]++
	m.Carry.SkillXP[slot] = data.SkillXPFor(m.Hero.Skill[slot]) + 1
	m.RetireOriginalHuman()
	mapload.UpdatePartyLoad(mapload.CloneParty([]mapload.PartyMember{*m})[0], m, t.in.Table, true, true)
	refreshDerivedPartyBook(m, t.in.Table)
	t.composeShopFaces()
	return fmt.Sprintf("trained %s to %d for %d", schoolSkillName(m.Mage, slot), m.Hero.Skill[slot], price)
}

func schoolSkillName(mage bool, slot int) string {
	names := data.SkillNames(mage)
	if slot <= 0 || slot > len(names) {
		return "skill"
	}
	return names[slot-1]
}
