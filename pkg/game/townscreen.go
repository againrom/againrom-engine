package game

import (
	"fmt"
	"image"
	"math/rand"
	"strings"

	"againrom/pkg/audio"
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/bmp"
	"againrom/pkg/mapload"
	"againrom/pkg/render/text"
	"againrom/pkg/sim"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// missionDoor opens mission n over the party the campaign holds. Opening a
// mission spans the whole game (install, audio, profile, campaign), so the
// screen holds one function and names none of that. The production door is
// (*FrontEnd).MissionOpener.
type missionDoor func(n int) ui.MapOpener

// townRoom is which part of the town is open. It lives HERE and not in
// pkg/ui: the screen tier holds one Screen value for the whole town and
// draws whatever rows it is handed, so it cannot tell the tavern from the
// gates and cannot grow a rule about either.
type townRoom int

const (
	// roomSquare is the town itself — the four doors.
	roomSquare townRoom = iota
	roomGates
	roomShop
	roomSchool
	roomTavern
	// roomTalk is one tavern conversation, which is a room rather than a
	// mode because leaving it is the same press that leaves any other and
	// the flow's own Escape arm should not have to know the difference.
	roomTalk
)

// The four doors of the square, in the order they are listed. THE ORDER IS
// OURS and the naming of the fourth is the owner's: a mission is started by
// walking out of the GATES, and the world map is what stands behind them in
// a later story (owner). What stands there today is the list of missions the
// three buildings have handed out.
//
// The three buildings come first because they are the producers and the
// gates are the consumer: there is one list between them, and reading the
// square top to bottom is reading that mechanism in the order it runs.
var townDoors = [...]struct {
	room townRoom
	name string
}{
	{roomTavern, "TAVERN"},
	{roomShop, "SHOP"},
	{roomSchool, "SCHOOL"},
	{roomGates, "GATES"},
}

// townScreen is the town as the front-end shows it: which room is open,
// which conversation is running, and the words for both.
//
// Room labels and status reports in this file are this build's own chrome.
// Conversation text is not: mission NPCs read `inn\NPC\npc%02dm%d`
// (REG-SCN-064), and mercenary candidates read their installed
// `inn\mercenary\npc%02d` leaves through the same event-text parser.
//
// THE CONVERSATION IS STATE HERE AND NOT ON Town. The model holds what
// registering an offer DID; this holds who is being spoken to and how far in.
// Opening a conversation registers its offer and no key does, so the model is
// decidable with no conversation anywhere near it.
type townScreen struct {
	// The screen's ports: campaign state, install, audio and draw sources,
	// art caches, tip preference and the mission opener. All are nil together
	// on a screen bound to no game.
	sess        *CampaignSession
	in          *InstallResources
	sound       townAudio
	draws       townDraws
	art         townArt
	townProcess *town.Process
	pc          *PersistenceContext
	openMission missionDoor

	room      townRoom
	roomAudio *ui.AudioScope
	audioRoom townRoom

	// npc and offer identify the town conversation. dialogueBuilding selects
	// the shipped text family and the room to return to.
	dialogueBuilding TownBuilding

	// npc and offer are the tavern row being spoken to: the NPC's own id
	// for the words, and the offer it carries. They are read only while room
	// is roomTalk.
	npc   int
	offer TownOffer

	// said is the last attempted part, including a failed lookup.
	said                int
	dialogueRevision    uint64
	dialogueButtonState ui.DialogueButtonState
	dialoguePolicy      ui.DialogueBackdrop
	dialogue            townDialogueState

	// innQueue is the missions the tavern's conversations have queued and
	// leaving the tavern commits, in the order they were heard. It is the
	// visit's own state: a load or a new game drops it.
	innQueue []int

	// mercenaryTalk is the selected mercenary type whose own shipped
	// text/inn/mercenary payload is open. talkDiagnostic is non-empty only
	// when that external payload is missing or malformed; no shipped candidate
	// is expected to take that diagnostic path.
	mercenaryTalk  int
	talkDiagnostic string
	speech         townSpeech

	// The shop screen's own state. shopChosen is which of the four rectangles
	// in the merchant's room has been clicked, shopNoShelf while none has;
	// shopShelf is the shelf that pairs with it; the two bases are the first
	// item each scrolling grid shows. All four are PRESENTATION state and live
	// here rather than on Shop for the reason the conversation does: the model
	// is decidable with no screen anywhere near it.
	shopChosen int
	shopShelf  ShopShelf
	shelfBase  int
	packBase   int

	// shopMember is which roster entry the character panel shows and whose
	// container the bottom strip is bound to (SHOP-PICKER-043's view+0x130).
	// It is presentation for the same reason the four above are.
	shopMember int

	// tavernSelection, schoolCell and townStats are the shared shell's
	// presentation state. The tavern stores a producer-stable key rather than
	// a cell index, because an accepted offer or a hired/returned squad can
	// change the finite producer list underneath it.
	tavernSelection tavernCandidateKey
	// tavernSlotRequests counts the tavern's own numbered sound slots the
	// room has requested. The slot's file is not identified, so a request
	// is recorded and no sample plays.
	tavernSlotRequests map[int]int
	schoolCell         int
	// schoolSpent marks the teacher speech latches already used since the room
	// was entered; a zero value is a set latch (TOWN-502).
	schoolSpent    [schoolLatchCount]bool
	schoolDiamond  schoolDiamondAnimation
	schoolColumn   schoolColumnAnimation
	schoolTraining schoolTrainingAnimation
	// schoolTrainingStatic projects original process-static presentation
	// clocks/counters and therefore survives object-local room/new-game resets.
	schoolTrainingStatic schoolTrainingStatic
	// square is the square's composer view, built on first use. squareRandom
	// is its fallback presentation generators, squareLoop its entry loop's
	// voice and squareAction the action a click's hooks left.
	square       *town.View
	squareRandom [2]*rand.Rand
	squareLoop   audio.Voice
	squareAction ui.TownAction
	// pages are the room pages the composer builds, by room name, and
	// pageRandom their fallback presentation generators by draw source.
	pages      map[string]*town.Page
	pageRandom map[string]*rand.Rand
	townStats  bool
	shopBook   bool

	// schoolSounds holds the school's own chrgen skill instances
	// (VIDEO-SFX-059); presentation only.
	schoolSounds ui.SFXVoices

	// The selected mercenary's generated inspection is immutable during one
	// visit. Cache it so the live client does not decode the same figure and
	// item layers every render frame; every selection/entry/reset boundary
	// below clears or replaces the cache by type.
	tavernDetailType   int
	tavernDetail       ui.TownCharacterView
	tavernDetailMask   *ui.SlotMask
	tavernDetailInfo   [12][]string
	tavernDetailPixels *image.RGBA
	tavernDetailText   []text.DrawCall

	// shopFigures is every party member composed in his own equipment, in
	// roster order, filled by composeShopFaces when a room is entered. It is
	// the composition cache SHOP-LIMIT-049 names as the borrowed panel's own
	// carried state, shared here between the character panel and the town
	// dialogue's speaker figures rather than composed twice.
	shopFigures []*image.RGBA

	// shopFigureMasks is composeShopFaces' own SlotMask beside each entry of
	// shopFigures, in the same roster order (1005 round 2, `SHOP-FIGURE-041`,
	// `DIV-085` restated for the shop): the per-pixel slot lookup the doll's
	// hover, tap and drag all read, built beside the figure by the same
	// composeInventorySubject call rather than derived from it afterwards.
	shopFigureMasks []*ui.SlotMask

	// shopSuppressSlot is the 1-based slot the shop's drag machine last lifted
	// off the shown member's doll, or 0. shopSuppressFigure and
	// shopSuppressMask are the figure and mask composed with that slot cleared,
	// and shopSuppressMember is the roster index they were composed for:
	// ShopScreen substitutes them only while the panel still shows that member.
	shopSuppressSlot   int
	shopSuppressMember int
	shopSuppressFigure *image.RGBA
	shopSuppressMask   *ui.SlotMask

	// shopIconCache is the base item pictures the screen has resolved, keyed
	// by item code and held across frames. Animated stars are a cell-paint
	// layer and never mutate or duplicate these pictures.
	shopIconCache map[uint16]*image.RGBA

	// shopSpellAtlasImg/shopSpellAtlasTried/shopSpellIcons are the shop's
	// install-owned cache for the same spellbook strip the mission view uses.
	shopSpellAtlasImg   *bmp.Image
	shopSpellAtlasTried bool
	shopSpellIcons      map[uint16]*image.RGBA

	// shopTip is the tip widget's own text, loaded once when the shop room is
	// entered (SHOP-TIP-045; `DIV-132`). Empty whenever the install ships no
	// `shop1.txt`, which draws no widget at all.
	shopTip string

	// townTip, schoolTip and tavernTip are the same widget's own text for the
	// other three rooms this story adds it to (1018 spec behaviours 2, 5):
	// town.txt, training.txt and inn.txt, loaded on the room's own entry
	// exactly as shopTip is. Empty means nothing shipped, nothing suppressed
	// it, or the front end could not read it — every one of those draws no
	// panel, on ReadShopTip's own rule.
	townTip, schoolTip, tavernTip string

	// tipClosed belongs to each constructed room popup. loadTip renews it.
	tipClosed   [roomTalk + 1]bool
	tipRevision uint64

	// worldMap is the gates' current presentation state. It is rebuilt from
	// Town.Available on each entry; install art remains cached on FrontEnd.
	worldMap *worldMapState

	// worldPosition is the party's own current-position field on the campaign
	// map (`TOWN-120`/`TOWN-121`, High): it changes only when travel
	// completes, and it survives leaving and re-entering the gates within the
	// same loaded game, so the next journey starts from where the last one
	// ended rather than from the town's own point every time. worldPositionSet
	// distinguishes "never travelled this game" (falls back to the first
	// MapObject's own point) from a position genuinely at that point
	// (`DIV-137`, authored: not evidenced by any claim read for this story).
	worldPosition    image.Point
	worldPositionSet bool

	// worldSelectedOnce is which missions have been registered or selected
	// at least once this loaded game (`DIV-128`/`DIV-138`; `TOWN-123`/
	// `TOWN-040`, High): the persisted marker cache's own selection gate. A
	// native Snapshot copies this set in sorted order and installs it after the
	// full per-game reset on restore; new game and original import leave it nil.
	worldSelectedOnce map[int]bool

	resolver speakerResolver
}

// resetWorldPosition drops the party's own remembered map position
// (`DIV-137`): every arrival in town, whether from finishing a mission or
// loading a save made in town, re-derives it from the first MapObject's own
// point on the next gates entry (`FrontEnd.arriveInTown`).
func (t *townScreen) resetWorldPosition() {
	if t == nil {
		return
	}
	t.worldPosition, t.worldPositionSet = image.Point{}, false
}

// resetWorldMarkers drops the session's own selected-mission marker cache
// (`DIV-128`/`DIV-138`): a genuinely different loaded game must not inherit
// which missions the PREVIOUS one had selected on the map.
func (t *townScreen) resetWorldMarkers() {
	if t == nil {
		return
	}
	t.worldSelectedOnce = nil
}

// resetForNewGame drops room/session state while retaining process services.
func (t *townScreen) resetForNewGame() {
	if t == nil {
		return
	}
	t.destroyRoomAudio()
	t.atSquare()
	t.dialogueBuilding, t.mercenaryTalk, t.talkDiagnostic = TownTavern, 0, ""
	t.dialogue = townDialogueState{}
	t.dialogueButtonState, t.dialoguePolicy = ui.DialogueButtonState{}, ui.DialogueBackdrop{}
	t.innQueue = nil
	t.shopChosen, t.shopShelf = shopNoShelf, ShelfArmour
	t.shelfBase, t.packBase = 0, 0
	t.shopMember, t.tavernSelection, t.schoolCell = 0, tavernCandidateKey{}, schoolNoSelection
	t.tavernSlotRequests = nil
	t.schoolDiamond = schoolDiamondAnimation{}
	t.schoolSpent = [schoolLatchCount]bool{}
	t.schoolColumn = schoolColumnAnimation{}
	t.schoolSounds.Stop()
	t.resetSchoolTraining()
	t.tavernPage().Reset()
	t.shopPage().Reset()
	t.townStats, t.shopBook = false, false
	t.clearTavernDetail()
	t.shopFigures, t.shopFigureMasks = nil, nil
	t.shopSuppressSlot, t.shopSuppressMember = 0, 0
	t.shopSuppressFigure, t.shopSuppressMask = nil, nil
	t.resetWorldMarkers()
	t.resetWorldPosition()
	t.worldMap = nil
	t.tipClosed = [roomTalk + 1]bool{}
}

// TownNPCTextPath is the inn dialogue leaf for one NPC and live mission.
func TownNPCTextPath(npc, mission int) (string, bool) {
	if npc < 0 || mission <= 0 {
		return "", false
	}
	return fmt.Sprintf("%stext/inn/npc/npc%02dm%d.txt", mainPrefix, npc, mission), true
}

// TownMercenaryTextPath maps the complete shipped candidate population to
// its own authored inn dialogue leaf. Types 11 and 15 are intentionally not
// candidates in either preserved population; npc35 is the gate line
// (townGateTextPath), a different inn text producer, and does not enter this
// mapping.
func TownMercenaryTextPath(typ int) (string, bool) {
	switch typ {
	case 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 13, 14:
		return fmt.Sprintf("%stext/inn/mercenary/npc%02d.txt", mainPrefix, typ), true
	default:
		return "", false
	}
}

func TownBuildingTextPath(building TownBuilding, mission int) (string, bool) {
	if mission <= 0 {
		return "", false
	}
	switch building {
	case TownShop:
		return fmt.Sprintf("%stext/shop/npc31m%d.txt", mainPrefix, mission), true
	case TownSchool:
		return fmt.Sprintf("%stext/training/npc34m%d.txt", mainPrefix, mission), true
	}
	return "", false
}

func (t *townScreen) townPayload() []byte {
	path, ok := t.townTextPath()
	if !ok || t.in.Archives == nil || t.in.Archives.Containers == nil {
		return nil
	}
	payload, _ := t.in.Archives.Containers.ReadFile(path)
	return payload
}

func (t *townScreen) townTextPath() (string, bool) {
	if t.mercenaryTalk != 0 {
		return TownMercenaryTextPath(t.mercenaryTalk)
	}
	return t.townTextPathFor(t.dialogueBuilding, t.offer, t.npc)
}

// townGateTextPath is the line the gate answers with when nothing is on offer.
// Its speech, speech/inn/mercenary/npc35p1.wav, follows from the leaf name.
const townGateTextPath = mainPrefix + "text/inn/mercenary/npc35.txt"

func (t *townScreen) townTextPathFor(building TownBuilding, offer TownOffer, npc int) (string, bool) {
	if building == TownGate {
		return townGateTextPath, true
	}
	mission := offer.Mission
	if mission <= 0 {
		mission = t.sess.Town.Chapter()
	}
	if building == TownTavern {
		return TownNPCTextPath(npc, mission)
	}
	return TownBuildingTextPath(building, mission)
}

func (t *townScreen) townPayloadFor(building TownBuilding, offer TownOffer, npc int) []byte {
	path, ok := t.townTextPathFor(building, offer, npc)
	if !ok || t.in.Archives == nil || t.in.Archives.Containers == nil {
		return nil
	}
	payload, err := t.in.Archives.Containers.ReadFile(path)
	if err != nil {
		return nil
	}
	return payload
}

// openTownDialogue retains a constructed panel after a failed first lookup.
// Missing resources remain outside the constructed-panel route.
func (t *townScreen) openTownDialogue(building TownBuilding, offer TownOffer, npc int) bool {
	payload := t.townPayloadFor(building, offer, npc)
	if payload == nil {
		return false
	}
	shows := t.TownDialogueShows() + 1
	t.room, t.dialogueBuilding, t.npc, t.offer, t.said = roomTalk, building, npc, offer, 1
	t.dialogueRevision++
	t.resetTownSpeech()
	t.mercenaryTalk, t.talkDiagnostic = 0, ""
	t.dialogue = newTownDialogue(payload)
	t.dialogue.shows = shows
	t.lookupTownDialogue()
	return true
}

func (t *townScreen) tavernOfferCanTalk(offer TownOffer) bool {
	return offer.Index >= 0 || offer.Mission <= 0 || t.sess.Town == nil ||
		t.sess.Town.progress == nil || t.sess.Town.progress.record(offer.Mission) != nil
}

func (t *townScreen) tavernVisibleOffers() []TownOffer {
	town := t.sess.Town
	offers := town.TavernRoster()
	out := make([]TownOffer, 0, len(offers))
	for _, offer := range offers {
		if !t.tavernOfferCanTalk(offer) && town.Done(offer.Mission) && offer.Mission > 0 &&
			!containsMission(town.camp.Main, offer.Mission) &&
			(containsMission(town.camp.Side, offer.Mission) || containsMission(town.camp.Offered, offer.Mission)) {
			continue
		}
		out = append(out, offer)
	}
	return out
}

// openOfferDialogue opens the conversation of one offered mission and
// registers the offer with it, before any key is pressed. The conversation has
// no accept or decline state: Enter, Escape and the button all turn the page
// and close it, so what the player pressed decides nothing.
func (t *townScreen) openOfferDialogue(building TownBuilding, offer TownOffer, npc int) bool {
	if building == TownTavern && !t.tavernOfferCanTalk(offer) {
		return false
	}
	if building == TownTavern && t.sess.joinOnTalk(t.in.townInstall(), offer.NPC) {
		// The speaker is a party member from here on: the cast the tavern
		// resolves its speakers over is rebuilt with her in it.
		t.composeShopFaces()
	}
	if !t.openTownDialogue(building, offer, npc) {
		return false
	}
	if offer.Mission <= 0 || offer.Index < 0 || t.sess.Town == nil {
		return true
	}
	switch {
	case building != TownTavern:
		// The shop and the school hand the mission over as their window opens.
		if mission, ok := t.sess.Town.Take(building, offer.Index); ok {
			t.sess.Town.registerOfferedMission(mission)
			t.markWorldSelected(mission)
		}
	default:
		// Every live hearing queues its identity, including repeats.
		t.innQueue = append(t.innQueue, offer.Mission)
	}
	return true
}

// commitInnQueue registers, in the order they were heard, the missions the
// tavern's conversations queued during the visit that is ending.
func (t *townScreen) commitInnQueue() {
	queue := t.innQueue
	t.innQueue = nil
	if t.sess == nil || t.sess.Town == nil {
		return
	}
	registerInnMissions(queue, func(mission int) {
		if t.sess.Town.registerInnMission(mission) {
			t.markWorldSelected(mission)
		}
	})
}

func registerInnMissions(queue []int, register func(int)) {
	for _, mission := range queue {
		register(mission)
	}
}

// openMercenaryDialogue opens one candidate's own installed EN/RU payload.
// A bad external file remains visible as a concise diagnostic instead of
// silently substituting prose that no shipped candidate authored.
func (t *townScreen) openMercenaryDialogue(typ int) {
	shows := t.TownDialogueShows() + 1
	t.resetTownSpeech()
	t.room, t.dialogueBuilding, t.npc, t.offer, t.said = roomTalk, TownTavern, 0, TownOffer{}, 1
	t.dialogueRevision++
	t.mercenaryTalk, t.talkDiagnostic = typ, ""
	t.composeShopFaces()
	payload := t.townPayload()
	t.dialogue = newTownDialogue(payload)
	t.dialogue.shows = shows
	if !t.lookupTownDialogue() {
		t.talkDiagnostic = fmt.Sprintf("Mercenary dialogue unavailable: npc%02d", typ)
	}
}

// townLines reads the game's own inn dialogue and pages its numbered parts.
// A missionless NPC uses the live main mission in the leaf name, as the zero
// InnMission arm does. A missing leaf or a payload with no parts is silent.
func (t *townScreen) townLines() []string {
	if t.talkDiagnostic != "" {
		return []string{t.talkDiagnostic}
	}
	payload := t.townPayload()
	dialogue := newTownDialogue(payload)
	var lines []string
	for part := 1; ; part++ {
		if !dialogue.lookup(part, HeroAudience(t.sess.Carried)) {
			break
		}
		lines = append(lines, dialogue.text)
	}
	return lines
}

// CanSave answers the description's save admission.
func (t *townScreen) CanSave() bool {
	return t != nil && t.sess != nil && t.squareView().SaveAdmitted()
}

// saveAdmitted is the campaign's save admission: an open town and no journey
// home under way.
func (t *townScreen) saveAdmitted() bool {
	return t.sess.Town != nil && t.sess.Town.Open() &&
		(t.worldMap == nil || t.worldMap.returnMission == 0)
}

// TownScreen is the ui.TownScreen this front end shows between missions, built
// fresh so that the room it opens on is the square.
//
// IT IS BUILT ONCE, BY App, AND HELD BY THE FLOW FOR THE LIFE OF THE PROCESS.
// The town is what missions happen between, so it outlives every one of them —
// see pkg/ui's own flow.town field comment for why that is the opposite of how
// the map screen's seams are held.
//
// IT IS REMEMBERED ON THE FRONT END (0143), which the flow holding it does not
// make redundant: loading a game replaces the town MODEL behind this screen, and
// the ROOM this screen has open is session state that belongs to the game being
// left. Restore puts it back at the square through this pointer, because pkg/ui
// installs the seam once and has no way to say "the same screen, at the square".
func (f *FrontEnd) TownScreen() ui.TownScreen {
	if f.Base().Profile.GameOf() == base.GameROM2 {
		return f.secondCampaignScreen()
	}
	if f.townUI == nil {
		// shopChosen starts at shopNoShelf, which is the state the original's
		// shop view is constructed in (SHOP-SCREEN-034). schoolCell starts at
		// schoolNoSelection for the same reason: 0 is a real skill, and a
		// freshly built town must not open the school with one already
		// selected and its price quoted.
		f.townUI = f.bindTown(&townScreen{shopChosen: shopNoShelf, schoolCell: schoolNoSelection})
		// The screen is constructed already at roomSquare (townRoom's own
		// zero value; atSquare's doc), so this is that room's own entry load
		// (1018 spec behaviour 2), not a special case of construction.
		f.townUI.loadTip(roomSquare)
	}
	return f.townUI
}

// atSquare closes whatever room is open. A loaded game is entered at the square:
// the room the player was standing in belongs to the game he just left, and
// showing him the loaded game's tavern because the old one's was open would be
// the one transition nobody could account for.
// AtTownSquare reports whether the four-door graphical layout is active. The
// gate line opens over the town view without leaving it, so the square stays
// the layout behind that line (TOWN-475).
func (t *townScreen) AtTownSquare() bool {
	return t != nil && (t.room == roomSquare || t.room == roomTalk && t.dialogueBuilding == TownGate)
}

func (t *townScreen) TownSquareEntryRevision() uint64 { return t.tipRevision }

// dialogueRoom is the room the open conversation stands over and returns to.
func (t *townScreen) dialogueRoom() townRoom {
	switch t.dialogueBuilding {
	case TownShop:
		return roomShop
	case TownSchool:
		return roomSchool
	case TownGate:
		return roomSquare
	}
	return roomTavern
}

// TownMusic exposes the static track owner for the room currently visible.
// A conversation keeps the building behind it, and the school puts the
// currently selected character's class track first in its ordinary list.
func (t *townScreen) TownMusic() (ui.MusicScene, bool) {
	if t == nil {
		return ui.MusicTown, false
	}
	room := t.room
	if room == roomTalk {
		room = t.dialogueRoom()
	}
	switch room {
	case roomGates:
		return ui.MusicCampaign, false
	case roomShop:
		return ui.MusicShop, false
	case roomTavern:
		return ui.MusicTavern, false
	case roomSchool:
		if t.sess == nil {
			return ui.MusicSchool, false
		}
		party := t.nextParty()
		i := townPickerIndex(party, t.shopMember)
		return ui.MusicSchool, len(party) != 0 && party[i].Mage
	default:
		return ui.MusicTown, false
	}
}

// TownSquareView is the square's composed scene, the font its message line
// draws with and its tip panel. A square whose art did not load has no scene,
// and app.go keeps the row-button layout for it.
func (t *townScreen) TownSquareView() ui.TownSquareView {
	if t == nil || t.sess == nil {
		return ui.TownSquareView{}
	}
	tip := rom1Town.Tip.Rect.Rectangle()
	v := ui.TownSquareView{
		Font: t.in.Font.Value(),
		Tip:  t.tipView(roomSquare, t.townTip, ui.TipPanelShrinkRect(tip, t.in.tipFont(), t.townTip)),
	}
	if t.in.TownSquareArt.Value() != nil {
		v.Scene = townSquareScene{t}
	}
	return v
}

// AtTownShop reports whether the shop's five-place table is on screen.
func (t *townScreen) AtTownShop() bool {
	return t != nil && (t.room == roomShop || t.room == roomTalk && t.dialogueBuilding == TownShop)
}

// AtTownSurface reports whether the tavern or school shared-shell surface is
// behind the current modal dialogue.
func (t *townScreen) AtTownSurface() bool {
	if t == nil {
		return false
	}
	if t.room == roomTavern || t.room == roomSchool {
		return true
	}
	return t.room == roomTalk && (t.dialogueBuilding == TownTavern || t.dialogueBuilding == TownSchool)
}

// composeShopFaces builds the portrait resolver a town conversation draws
// through, and the character panel's own figure for every member.
//
// ONE COMPOSITION SERVES BOTH (SHOP-LIMIT-049). The dialogue's speaker figure
// and the shop's character panel are the same picture of the same member in the
// same equipment, so they are composed once, here, and read from
// townScreen.shopFigures afterwards.
func (t *townScreen) composeShopFaces() {
	// A cached talk panel may show a party member; the party just changed.
	if t.tavernDetailType < 0 {
		t.clearTavernDetail()
	}
	party := t.shopParty()
	for i := range party {
		mapload.NormalizePartyShieldLoadout(&party[i], t.in.Table)
	}
	// shopWeaponFallbackCode reads member.Weapon and member.WeaponMaterialized
	// directly (round-2 adversarial review, fifth pass), so no per-room seeding
	// is needed here any more: a party replaced by a completed mission or a
	// restore carries its own members' own WeaponMaterialized bit, never a
	// previous occupant's.
	var src entrySource
	if t.in.Archives != nil {
		src = t.in.Archives.Containers
	}

	var playerFigure *image.RGBA
	playerFigures := make(map[int]*image.RGBA)
	t.shopFigures = make([]*image.RGBA, len(party))
	t.shopFigureMasks = make([]*ui.SlotMask, len(party))
	for i, member := range party {
		// eq IS mapload.EquipmentFromParty, collapsed off the same two-line
		// Carry-over-Worn idiom shopview.go's refreshShopDrag collapsed onto
		// (round-2 adversarial review, twelfth pass, C1 addendum) — see that
		// call's own comment for why.
		eq := mapload.EquipmentFromParty(member)
		// SLOT 1'S FALLBACK GOES THROUGH shopWeaponFallbackCode (round-2
		// adversarial review, third pass, counterexample 2's own duplication
		// finding applied to this compositor too): the unconditional member.Weapon
		// != nil test this replaced fired even when the code already sat in
		// member's own pack — a member who unequipped his starting weapon
		// mid-mission and finished it — drawing a weapon on the doll that
		// shopSlot1Code's own discriminator (shoproom.go) no longer answers for
		// the readers below the picture: the picture and the readers would
		// disagree, DIV-085's own property broken, this time on the compositor's
		// side instead of a reader's.
		if occupied, _ := eq.Occupied(1); !occupied {
			if code, ok := t.shopWeaponFallbackCode(i); ok {
				eq.SetCode(1, data.ItemCode(code))
			}
		}
		// MERC-TYPE-001
		if !member.Hired() && data.ComposesFigure(member.Class) {
			if body, bodyDir, class, matched := data.HeroAppearance(
				t.in.Bodies, eq, member.Mage, false); matched {
				party[i].Body, party[i].BodyDir, party[i].Class = string(body), bodyDir, class
			}
		}
		subject := composeMemberPortraitLayered(src, t.in.Units, uint32(i+1), eq,
			figureLayersFor(mapload.MemberLayers(member, t.in.Table), t.in.Table), member)
		t.shopFigures[i] = subject.Figure
		t.shopFigureMasks[i] = subject.SlotMask
		if i == 0 {
			playerFigure = subject.Figure
		}
		if member.CompanionNPC != 0 {
			playerFigures[member.CompanionNPC] = subject.Figure
		}
	}
	// A RECOMPOSITION INVALIDATES ANY CACHED SUPPRESSED PICTURE (1005 round
	// 2): shopSuppressFigure was composed against the equipment as it stood
	// before this call, and every caller of composeShopFaces just changed
	// that equipment or entered a room the drag machine has not run in yet —
	// refreshShopDrag rebuilds it the next frame a drag is actually live.
	t.shopSuppressSlot, t.shopSuppressMember, t.shopSuppressFigure, t.shopSuppressMask = 0, 0, nil, nil
	// THE TOWN HOLDS NO WORLD, so a speaker no cast answers takes the
	// synthesised arm. That arm is deliberately bare: the school's quest giver
	// is a portrait, not a hero whose starting equipment should be layered onto
	// it. The tavern's client actors are the stock mercenary units, so a tavern
	// resolver answers a Human record with the unit it admits, in that unit's
	// own worn set (TAVERN-TALKPIC-016); the shop and the school have no cast.
	t.resolver = speakerResolver{src: src, units: t.in.Units, npcFaces: t.in.NPCFaces,
		portraits: make(map[string]*image.RGBA), figurePics: make(map[figureCacheKey]*image.RGBA),
		playerFigure: playerFigure, playerFigures: playerFigures}
	if t.room == roomTavern {
		t.resolver.cast = t.tavernSpeakerCast()
	}
	for _, member := range party {
		if primaryPlayerHero(member) {
			t.resolver.cast.playerDir, _ = memberFigure(member)
			t.resolver.cast.hasPlayer = true
			break
		}
	}
}

// mercenaryTalkPicture is the selected candidate's own picture for the town
// dialogue. Human candidates reuse the inspection figure composed from the
// same generated member and worn set as Hire. Siege candidates use their unit
// class's flat tier portrait, which is the other arm of the normal actor
// picture rule.
//
// The event part's NPC tag still owns the portrait window. It does not own the
// picture on this owner-directed tavern path: none of the thirteen installed
// candidate leaves has a live candidate actor in the town cast, so resolving
// the tag through speakerResolver substitutes the party/player figure.
func (t *townScreen) mercenaryTalkPicture(typ int) (*image.RGBA, bool) {
	if t == nil || t.sess == nil {
		return nil, false
	}
	members, ok := t.buildMercenarySquad(typ, 1)
	if !ok || len(members) != 1 {
		return nil, false
	}
	m := members[0]
	if data.ComposesFigure(m.Class) {
		detail, _, _, ok := t.tavernCandidateDetail(typ)
		return detail.Figure, ok && detail.Figure != nil
	}
	if t.in.Units == nil {
		return nil, false
	}
	var src entrySource
	if t.in.Archives != nil {
		src = t.in.Archives.Containers
	}
	if t.resolver.portraits == nil {
		t.resolver.portraits = make(map[string]*image.RGBA)
	}
	pic := classPortrait(src, t.in.Units.Classes[m.Class], m.FigureFace, t.resolver.portraits)
	return pic, pic != nil
}

// A campaign offer may speak through a mercenary platoon record too (npc2
// in chapter 110). Its window belongs to that unit picture, not to the
// player's hero. Prefer a carried member's current loadout, then the same
// installed candidate source used by the tavern's Talk action.
func (t *townScreen) platoonTalkPicture(speaker int) (*image.RGBA, bool) {
	if t.in.Table == nil {
		return nil, false
	}
	if _, ok := t.in.Table.NPC.Mercenary(int32(speaker)); !ok {
		return nil, false
	}
	for i, member := range t.sess.Carried {
		if int(member.MercenaryType) == speaker && i < len(t.shopFigures) && t.shopFigures[i] != nil {
			return t.shopFigures[i], true
		}
	}
	return t.mercenaryTalkPicture(speaker)
}

func (t *townScreen) speakerFace(speaker int) (*image.RGBA, image.Rectangle, bool) {
	rec := t.in.NPCFaces[int32(speaker)]
	if !rec.Tokens.Has(data.NPCTokenPlatoon) {
		return t.resolver.SpeakerFace(speaker)
	}
	var win image.Rectangle
	if rec.HasWindow {
		win = ui.NoticeFaceWindow(rec.WindowX, rec.WindowY)
	}
	pic, ok := t.platoonTalkPicture(speaker)
	return pic, win, ok
}

// TownDialogue renders the current shipped dialogue part through the same
// portrait notice composer used on mission maps.
func (t *townScreen) townDialogueLayout() (ui.NoticeLayout, []byte, bool) {
	if t == nil || t.room != roomTalk {
		return ui.NoticeLayout{}, nil, false
	}
	payload := t.dialogue.payload
	// THE BUTTON WORD IS THE INSTALL'S. This screen composes its own layout
	// rather than taking the viewer's, so the word set is applied here
	// explicitly; every other property of the layout is the authored one.
	l := t.in.Words.OnLayout(ui.AuthoredDialogueLayout())
	l.Frame = t.art.menuFrame()
	return l.WithPortrait(t.mercenaryTalk != 0 || t.dialogue.hasPortrait).WithDialogueButtonState(t.dialogueButtonState).WithDialogueBackdrop(t.dialoguePolicy), payload, true
}

func (t *townScreen) TownDialogueButtonState() ui.DialogueButtonState {
	if t == nil {
		return ui.DialogueButtonState{}
	}
	return t.dialogueButtonState
}

func (t *townScreen) TownDialogueVisual(state ui.DialogueButtonState, policy ui.DialogueBackdrop) {
	if t != nil {
		t.dialogueButtonState, t.dialoguePolicy = state, policy
	}
}

func (t *townScreen) TownDialogue() (*image.RGBA, bool) {
	layout, _, ok := t.townDialogueLayout()
	if !ok || t.in.Font.Value() == nil {
		return nil, false
	}
	line := t.dialogue.text
	if t.talkDiagnostic != "" {
		line = t.talkDiagnostic
	}
	layout = layout.WithFaceWindow(t.dialogue.faceWindow)
	return ui.RenderNotice(layout, t.in.Font.Value(), line, t.dialogue.face), true
}

func (t *townScreen) TownDialogueFrame() (*ui.DialogFrame, image.Rectangle) {
	layout, _, ok := t.townDialogueLayout()
	if !ok || t.in.Font.Value() == nil {
		return nil, image.Rectangle{}
	}
	return layout.Frame, image.Rectangle{Max: layout.Box.Size().Sub(image.Pt(8, 8))}
}

// AdvanceTownDialogue is the one answer to Enter, Escape and a click on the
// button: it turns the page, and on the last page closes the conversation. The
// press that closes it posts no line and registers nothing, whether the offer
// came from the shop, the school or the tavern.
func (t *townScreen) AdvanceTownDialogue() ui.TownAction {
	if t == nil || t.room != roomTalk {
		return ui.TownAction{}
	}
	t.dialogueRevision++
	t.said++
	if t.talkDiagnostic == "" && t.lookupTownDialogue() {
		return ui.TownAction{}
	}
	t.dialogue.closeTips = t.dialogue.tips
	t.room = t.dialogueRoom()
	t.mercenaryTalk, t.talkDiagnostic = 0, ""
	// Closing a conversation preserves the room popup's construction lifetime.
	t.rereadTip(t.room)
	return ui.TownAction{}
}

func (t *townScreen) TownDialogueShows() uint64 {
	if t == nil || t.room != roomTalk {
		return 0
	}
	return t.dialogue.shows
}

func (t *townScreen) TownDialogueRevision() uint64 {
	if t == nil {
		return 0
	}
	return t.dialogueRevision
}

func (t *townScreen) TownDialogueButton() (image.Rectangle, bool) {
	layout, _, ok := t.townDialogueLayout()
	if !ok {
		return image.Rectangle{}, false
	}
	return layout.Button, !layout.Button.Empty()
}

func townItemStacks(items []uint16) []sim.ItemStack {
	var stacks []sim.ItemStack
	positions := make(map[uint16]int)
	for _, code := range items {
		if i, ok := positions[code]; ok {
			stacks[i].Count++
			continue
		}
		positions[code] = len(stacks)
		stacks = append(stacks, sim.ItemStack{Code: code, Count: 1})
	}
	return stacks
}

func (t *townScreen) atSquare() {
	if t == nil {
		return
	}
	t.squareView().EnterSquare()
}

// Header is the line above the list: which room, and which chapter the town
// stands in.
func (t *townScreen) Header() string {
	ch := t.sess.Town.Chapter()
	where := "the town square"
	switch t.room {
	case roomGates:
		where = "the gates"
	case roomShop:
		where = "the shop"
	case roomSchool:
		where = "the school"
	case roomTavern:
		where = "the tavern"
	case roomTalk:
		where = t.dialogueBuilding.String()
		if t.dialogueBuilding == TownTavern {
			where = fmt.Sprintf("the tavern - NPC %d", t.npc)
		}
	}
	if ch == 0 {
		return fmt.Sprintf("%s - the campaign offers nothing further", where)
	}
	return fmt.Sprintf("%s - chapter %d    [Esc: back]", where, ch)
}

// Footer is what the player has, stated wherever in the town he is standing.
func (t *townScreen) Footer() []string {
	party := t.nextParty()
	items, worn := 0, 0
	for _, m := range party {
		items += len(mapload.MemberCarriedItems(m, t.in.Table))
		for _, item := range mapload.MemberItemEquipment(m, t.in.Table) {
			if !item.Empty() {
				worn++
			}
		}
	}
	out := []string{
		fmt.Sprintf("gold %d    party %d    carried %d items, %d worn",
			t.sess.Town.Gold(), len(party), items, worn),
		fmt.Sprintf("available %v    done %d", t.sess.Town.Available(), t.doneCount()),
	}
	return out
}

// doneCount is how many missions the town has recorded finished.
func (t *townScreen) doneCount() int {
	n := 0
	for _, m := range append(append([]int{}, t.in.Campaign.Value().Main...), t.in.Campaign.Value().Side...) {
		if t.sess.Town.Done(m) {
			n++
		}
	}
	return n
}

// Rows is what the open room holds.
//
// A ROOM WITH NOTHING IN IT SAYS SO. Every arm below has an unchoosable row
// for its empty case rather than returning nil: a list of no rows is a
// screen that cannot say what it is, which is exactly the empty rectangle
// the spec refuses.
func (t *townScreen) Rows() []ui.TownRow {
	switch t.room {
	case roomGates:
		return nil
	case roomShop:
		// THE SHOP ROOM HAS NO ROWS. It is drawn as the original's own screen and
		// every control on it is a rectangle, so a list here would be a second,
		// invisible way to press the same things.
		return nil
	case roomSchool:
		return t.shelfRows(TownSchool)
	case roomTavern:
		return t.tavernRows()
	case roomTalk:
		return nil
	}
	return t.squareRows()
}

func (t *townScreen) squareRows() []ui.TownRow {
	rows := make([]ui.TownRow, 0, len(townDoors))
	for _, d := range townDoors {
		hint := ""
		switch d.room {
		case roomTavern:
			hint = fmt.Sprintf("%d with something to say", countOffering(t.sess.Town.Offers(TownTavern)))
		case roomShop:
			hint = fmt.Sprintf("%d on the shelf", len(t.sess.Town.Offers(TownShop)))
		case roomSchool:
			hint = fmt.Sprintf("%d on the board", len(t.sess.Town.Offers(TownSchool)))
		case roomGates:
			hint = fmt.Sprintf("%d mission(s) available", len(t.sess.Town.Available()))
		}
		rows = append(rows, ui.TownRow{
			Text:      fmt.Sprintf("%-8s  %s", d.name, hint),
			Choosable: true,
		})
	}
	return rows
}

// countOffering is how many of the tavern's rows actually hold a mission —
// REG-SCN-064's zero sentinel is an NPC who is there and has nothing to give,
// so he is listed and is not counted.
func countOffering(offers []TownOffer) int {
	n := 0
	for _, o := range offers {
		if o.Mission > 0 {
			n++
		}
	}
	return n
}

// shelfRows is the shop's or the school's list.
//
// ONLY THE FIRST ROW IS CHOOSABLE, which is REG-SCN-064's "take element 0 on
// entry and remove it" with the taking made a press rather than the entry: a
// room that consumed an offer just for being walked into could not be looked
// at.
func (t *townScreen) shelfRows(b TownBuilding) []ui.TownRow {
	ch := t.sess.Town.ChapterData()
	var rows []ui.TownRow
	if b == TownShop {
		rows = append(rows, ui.TownRow{
			Text: fmt.Sprintf("prices here run %d to %d    (trade is not yet available)", ch.ShopMin, ch.ShopMax),
		})
	} else {
		rows = append(rows, ui.TownRow{
			Text: "training is not yet available - what the school has today is work",
		})
	}
	offers := t.sess.Town.Offers(b)
	if len(offers) == 0 {
		rows = append(rows, ui.TownRow{Text: "nothing on offer in this chapter"})
		return rows
	}
	for k, o := range offers {
		rows = append(rows, ui.TownRow{
			Text:      fmt.Sprintf("mission %d%s", o.Mission, headMark(k)),
			Choosable: k == 0,
		})
	}
	return rows
}

func headMark(k int) string {
	if k == 0 {
		return "   <- take this"
	}
	return "   (behind the one above)"
}

// tavernRows lists the chapter's NPCs. An NPC with nothing to give is LISTED
// AND CHOOSABLE — he says so when spoken to, which is REG-SCN-064's zero
// arm, and hiding him would hide half the mechanism.
func (t *townScreen) tavernRows() []ui.TownRow {
	offers := t.tavernVisibleOffers()
	if len(offers) == 0 {
		return []ui.TownRow{{Text: "the tavern is quiet - nobody here has work in this chapter"}}
	}
	rows := make([]ui.TownRow, 0, len(offers))
	for _, o := range offers {
		what := "nothing to say"
		if o.Mission > 0 {
			what = "wants to talk"
		}
		rows = append(rows, ui.TownRow{
			Text:      fmt.Sprintf("NPC %-3d  %s", o.NPC, what),
			Choosable: true,
		})
	}
	return rows
}

// Choose acts on row i of what Rows last reported.
//
// THE ROW INDEX IS INTO THE ROWS THE SCREEN WAS SHOWING, which is why each
// arm re-derives the same list Rows built rather than caching one: pkg/ui
// rebuilds after every Choose and every Back and at no other time, so what
// the player pressed is what Rows last answered, and re-deriving it here is
// the one way to be sure the two agree.
func (t *townScreen) Choose(i int) ui.TownAction {
	if t.room == roomSchool && t.schoolTrainingBusy() {
		return ui.TownAction{}
	}
	switch t.room {
	case roomSquare:
		return t.chooseSquare(i)
	case roomGates:
		return ui.TownAction{}
	case roomShop:
		return ui.TownAction{}
	case roomSchool:
		return t.openBuildingDialogue(TownSchool, i)
	case roomTavern:
		offers := t.tavernVisibleOffers()
		if i < 0 || i >= len(offers) {
			return ui.TownAction{}
		}
		if !t.tavernOfferCanTalk(offers[i]) {
			return ui.TownAction{}
		}
		t.composeShopFaces()
		t.openOfferDialogue(TownTavern, offers[i], offers[i].NPC)
		return ui.TownAction{}
	case roomTalk:
		return t.AdvanceTownDialogue()
	}
	return ui.TownAction{}
}

// openBuildingDialogue opens the shipped shop or school conversation for the
// head offer, which opening registers.
func (t *townScreen) openBuildingDialogue(b TownBuilding, i int) ui.TownAction {
	offers := t.sess.Town.Offers(b)
	if i != 1 || len(offers) == 0 {
		return ui.TownAction{}
	}
	t.composeShopFaces()
	t.openOfferDialogue(b, offers[0], 0)
	return ui.TownAction{}
}

// Back leaves the open room for the square, and reports whether it left one.
//
// FALSE AT THE SQUARE is what lets the flow's own Escape arm leave the town
// without this file knowing where the town sits in the front end.
func (t *townScreen) Back() bool {
	if t.room == roomSquare {
		return false
	}
	if t.room == roomSchool && t.schoolTrainingBusy() {
		return true
	}
	if t.room == roomGates && t.worldMap != nil && t.worldMap.returnMission != 0 {
		// Escape skips homeward presentation through the same arrival writer;
		// merely changing rooms would strand the remembered position away.
		t.arriveWorldMap()
		return true
	}
	if t.room == roomTalk {
		// Escape is the button's answer, whatever page the conversation is on.
		t.AdvanceTownDialogue()
		return true
	}
	// The room's exit steps run, then the square's entry: the tavern commits
	// its queue and the shop clears its table (SHOP-LIFE-013), so nothing the
	// player put down is destroyed by the next restock.
	t.squareView().LeaveRoom(townRoomName(t.room))
	return true
}

// townLine is the headless report of a win that ends in the town: the
// chapter it stands in, what each of the three buildings holds there, and
// what the player walks in with.
//
// IT READS THE REAL TOWN AND GUESSES NOTHING. Every number below comes off the
// same Town the windowed front end shows, after the same FinishMission call a
// genuine win makes — so a headless run and a windowed one agree about what the
// buildings are holding, which is the one thing a run with no window can
// witness about a screen.
//
// IT NAMES NO SCREEN AND NO ROOM. What a door is called is the town screen's
// own business; this states the campaign data behind it, so the line stays true
// if the rooms are ever rearranged.
func (f *FrontEnd) townLine(n int) string {
	ch := f.Town.Chapter()
	line := fmt.Sprintf("againrom: winning mission %d leads to the town, chapter %d", n, ch)
	for _, b := range []TownBuilding{TownTavern, TownShop, TownSchool} {
		var held []string
		for _, o := range f.Town.Offers(b) {
			if b == TownTavern {
				held = append(held, fmt.Sprintf("npc %d -> %d", o.NPC, o.Mission))
				continue
			}
			held = append(held, fmt.Sprintf("%d", o.Mission))
		}
		if len(held) == 0 {
			continue
		}
		line += fmt.Sprintf("; %s holds %s", b, strings.Join(held, ", "))
	}
	cd := f.Town.ChapterData()
	line += fmt.Sprintf("; shop prices %d-%d; gold %d; party %d",
		cd.ShopMin, cd.ShopMax, f.Town.Gold(), len(f.NextParty()))
	// The merchant's shelves, which are the one thing about 0157's screen a
	// run with no window can witness: the ceiling that was applied and how
	// many items each shelf drew under it.
	if f.Shop != nil {
		line += fmt.Sprintf("; merchant ceiling %d, shelves", f.Shop.Ceiling())
		for shelf := ShopShelf(0); shelf < numShopShelves; shelf++ {
			line += fmt.Sprintf(" %s %d", shelf, len(f.Shop.Shelf(shelf)))
		}
	}
	return line
}

// bindTown hands the front end's components to a town screen.
func (f *FrontEnd) bindTown(t *townScreen) *townScreen {
	t.sess, t.in, t.pc = &f.CampaignSession, &f.InstallResources, &f.PersistenceContext
	t.bindServices(&f.RuntimeServices, &f.InstallResources, &f.Presentation)
	t.openMission = f.MissionOpener
	return t
}

func (t *townScreen) nextParty() []mapload.PartyMember {
	return t.sess.nextParty(func() []mapload.PartyMember {
		return MissionParty(t.in.StartWeapon.Value(), t.in.Bodies, t.in.Table)
	})
}
