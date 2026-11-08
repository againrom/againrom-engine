package game

import (
	"fmt"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

type innProfile struct {
	keepSpeakers, uniqueSpeakers, hideCompleted bool
	tokenActors, identityActors, keyedActors    bool
	disablePoorHire, squadHire                  bool
	sectionText, joinOnOpen, applyTalk          bool
	queueOffers                                 bool
	leaveScene, soundBeforeTalk                 bool
	hire                                        *hireProfile
	hireTalkKey                                 string
}

var chapterInnProfile = innProfile{keepSpeakers: true, hideCompleted: true,
	tokenActors: true, disablePoorHire: true, squadHire: true, joinOnOpen: true, queueOffers: true, hire: &tavernHires}

var bankInnProfile = innProfile{uniqueSpeakers: true, identityActors: true, keyedActors: true,
	sectionText: true, applyTalk: true, leaveScene: true, soundBeforeTalk: true, hire: &bankHires, hireTalkKey: "npc%dabout"}

type innSeat struct {
	word  innOffer
	index int
}

func (s innSeat) offer() TownOffer {
	return TownOffer{Index: s.index, Mission: s.word.topic, NPC: s.word.npc}
}

func (s innSeat) hire() bool { return !s.word.speaker() }

func innTalkRows(rows []TownOffer) []innSeat {
	var out []innSeat
	for _, o := range rows {
		out = append(out, innSeat{word: innOffer{npc: o.NPC, topic: o.Mission}, index: o.Index})
	}
	return out
}

// innRoster projects talk options once; registry retention and key uniqueness
// are independent policies. Neither policy changes the saved offer arrays.
func innRoster(p innProfile, home *Town, options []innOffer) []innSeat {
	var live []innSeat
	if p.keepSpeakers {
		live = innTalkRows(home.Offers(TownTavern))
	} else {
		for _, o := range options {
			if o.speaker() {
				live = append(live, innSeat{word: o})
			}
		}
	}
	if p.keepSpeakers && home.Open() {
		forms := home.camp.chapterForms(home.ChapterData().Mission)
		if home.progress == nil {
			forms = forms[:1]
		}
		var first, whole []innSeat
		for i, registry := range forms {
			out, ok := innRegistry(home, registry, live)
			if len(forms) == 1 || ok && innKeptAccepted(home, out) {
				return out
			}
			if i == 0 {
				first = out
			}
			if ok && whole == nil {
				whole = out
			}
		}
		if whole != nil {
			return whole
		}
		return first
	}
	if !p.uniqueSpeakers {
		return live
	}
	var out []innSeat
	seen := map[int]bool{}
	for _, s := range live {
		if !seen[s.word.npc] {
			seen[s.word.npc] = true
			out = append(out, s)
		}
	}
	return out
}

func innRegistry(home *Town, registry Chapter, live []innSeat) ([]innSeat, bool) {
	rows := min(len(registry.Inn), len(registry.InnNPC))
	out := make([]innSeat, 0, max(rows, len(live)))
	next, sub := 0, true
	for i := 0; i < rows; i++ {
		npc, mission := registry.InnNPC[i], registry.Inn[i]
		if !home.tavernHere(npc, mission) {
			continue
		}
		found := -1
		for j := next; j < len(live); j++ {
			if live[j].word.npc == npc && live[j].word.topic == mission {
				found = j
				break
			}
		}
		if found >= 0 {
			sub = sub && found == next
			out = append(out, live[next:found+1]...)
			next = found + 1
		} else if mission > 0 {
			out = append(out, innSeat{word: innOffer{topic: mission, npc: npc}, index: -1 - i})
		}
	}
	return append(out, live[next:]...), sub && next == len(live)
}

func innKeptAccepted(home *Town, seats []innSeat) bool {
	for _, s := range seats {
		o := s.offer()
		if o.Index < 0 && home.progress.record(o.Mission) == nil && !home.available[o.Mission] && !home.Done(o.Mission) {
			return false
		}
	}
	return true
}

func innCanTalk(home *Town, offer TownOffer) bool {
	return offer.Index >= 0 || offer.Mission <= 0 || home == nil || home.progress == nil || home.progress.record(offer.Mission) != nil
}

// innSeats partitions the displayed seats into hires then talkers. The profile
// controls completed retained offers; the input order survives both partitions.
func innSeats(p innProfile, home *Town, options, hires []innOffer) ([]innSeat, int) {
	var out []innSeat
	for _, o := range hires {
		out = append(out, innSeat{word: o})
	}
	n := len(out)
	for _, s := range innRoster(p, home, options) {
		o := s.offer()
		if p.hideCompleted && !innCanTalk(home, o) && home.Done(o.Mission) && o.Mission > 0 &&
			!containsMission(home.camp.Main, o.Mission) && (containsMission(home.camp.Side, o.Mission) || containsMission(home.camp.Offered, o.Mission)) {
			continue
		}
		out = append(out, s)
	}
	return out, n
}

type innActorInput struct {
	party    []mapload.PartyMember
	stock    []tavernStockActor
	record   data.NPCFace
	known    bool
	campaign *secondCampaign
	install  *InstallResources
}

type innPerson struct {
	member mapload.PartyMember
	known  bool
	object tavernTalkObject
}

// innActor resolves membership, hire templates, token matches and fallback
// records in that order, using only the strategies enabled by the profile.
func innActor(p innProfile, npc int, in innActorInput) innPerson {
	if p.identityActors {
		for _, m := range in.party {
			if m.Membership().Matches(mapload.CompanionMember, npc) || m.Membership().Matches(mapload.CampaignMember, npc) {
				return innPerson{member: m, known: true}
			}
		}
	}
	if p.keyedActors {
		for _, o := range in.campaign.hires() {
			if o.npc == npc {
				s := in.campaign.hireState()
				s.table = in.install.Table
				members, ok := hireMembers(p.hire, s, npc, 1)
				if !ok {
					return innPerson{}
				}
				return innPerson{member: members[0], known: true}
			}
		}
	}
	if p.tokenActors {
		rec, known, party, stock := in.record, in.known, in.party, in.stock
		if known && rec.Tokens.Has(data.NPCTokenPlatoon) && mapload.TableEdition(nil).TavernType(npc) {
			return innPerson{object: platoonTalkObject(npc, party)}
		}
		cast := tavernTalkCast(party, stock)
		if known && !rec.Tokens.Has(data.NPCTokenPlatoon) {
			if a, ok := cast.resolveNPC(int32(npc), rec); ok {
				o := tavernTalkObject{live: int(a.id), hero: a.hero, mage: a.fig.Dir.Mage(), human: !a.hero}
				if i := int(a.id); i < len(party) {
					o.sheet = int(party[i].TavernType())
				} else {
					o.stock = stock[i-len(party)].typ
					o.sheet = o.stock
				}
				return innPerson{object: o}
			}
		}
		o := tavernTalkObject{live: -1, sheet: npc}
		if known {
			t := rec.Tokens
			o.hero, o.human, o.mage = t.Has(data.NPCTokenHero), t.Has(data.NPCTokenHuman), t.Has(data.NPCTokenMage)
			if (t.Has(data.NPCTokenMyClass) || t.Has(data.NPCTokenNotMyClass)) && cast.hasPlayer {
				o.mage = cast.playerDir.Mage() == t.Has(data.NPCTokenMyClass)
			}
		}
		return innPerson{object: o}
	}
	if p.keyedActors {
		if _, ok := tableEdition(in.install.Table).PartyIdentity(npc); ok {
			m, ok := in.install.townInstall().character(mapload.CharacterSpec{NPC: int32(npc)}, in.install.localizedNPCName(npc-1, ""), in.party)
			return innPerson{member: m, known: ok}
		}
		m, ok := mapload.CampaignActor(in.install.Table, mapload.CampaignActorSpec{Profile: mapload.CampaignActorProfile{ServerBase: int32(npc), PlayerCharacter: true, UnitPerson: true}}, nil)
		return innPerson{member: m, known: ok}
	}
	return innPerson{}
}

type innVerb uint8

const (
	innSleep innVerb = iota
	innHire
	innTalk
	innExit
	innActivate
)

func innButtons(p innProfile, home *Town, seat *innSeat, sleep bool) [4]bool {
	buttons := [4]bool{sleep, false, false, true}
	if seat != nil {
		buttons[innHire] = seat.hire() && (!p.disablePoorHire || seat.word.high || seat.word.affordable)
		buttons[innTalk] = seat.hire() || !p.keepSpeakers || innCanTalk(home, seat.offer())
	}
	return buttons
}

// innAction is the seat mutation door for both hosts. Geometry and list-row
// adapters supply a verb; they never choose a different talk or hire rule.
func (t *townScreen) innAction(p innProfile, seat *innSeat, verb innVerb) ui.TownAction {
	if verb == innActivate {
		verb = innTalk
		if seat != nil && seat.hire() {
			verb = innHire
		}
	}
	switch verb {
	case innSleep:
		t.sleep()
	case innExit:
		if p.leaveScene {
			t.raiseShownRoomEvent(roomEventLeave)
			t.squareView().LeaveRoom(t.roomName())
		} else {
			t.Back()
		}
	case innHire:
		if seat == nil || !seat.hire() {
			return ui.TownAction{}
		}
		if p.squadHire {
			t.pressMercenary(seat.word.npc)
			return ui.TownAction{}
		}
		c, npc := t.state(), seat.word.npc
		fired := c.hired(npc)
		key := c.seatHire(npc)
		switch {
		case fired:
			t.raiseShownRoomEvent(roomEventDismiss)
		case c.hired(npc):
			t.raiseShownRoomEvent(roomEventHire)
		case key != "":
			t.raiseShownRoomEvent(roomEventHireShort)
		}
		if key != "" {
			return t.openTownText(key)
		}
	case innTalk:
		if seat == nil || !innButtons(p, t.sess.Town, seat, false)[innTalk] {
			return ui.TownAction{}
		}
		if p.soundBeforeTalk && !seat.hire() {
			t.raiseShownRoomEvent(roomEventTalk)
		}
		req := innTalkRequest{building: TownTavern, offer: seat.offer(), npc: seat.word.npc}
		if seat.hire() {
			if p.sectionText {
				req.key = fmt.Sprintf(p.hireTalkKey, seat.word.npc)
			} else {
				req.candidate = seat.word.npc
			}
		} else {
			if p.sectionText {
				req.key, req.word = secondTalkKey(seat.word.npc, seat.word.topic), &seat.word
			}
			if !p.sectionText {
				t.composeShopFaces()
			}
		}
		action, _ := t.startInnTalk(p, req)
		if !p.soundBeforeTalk {
			t.raiseRoomEvent(roomTavern, roomEventTalk)
		}
		return action
	}
	return ui.TownAction{}
}

var innButtonVerbs = map[string]innVerb{"hire": innHire, "talk": innTalk, "exit": innExit}

// innButtonVerb maps a control name to its inn verb; the Sleep control is
// recognised by the one test sleep.go owns.
func innButtonVerb(name string) (innVerb, bool) {
	if sleepControlName(name) {
		return innSleep, true
	}
	verb, ok := innButtonVerbs[name]
	return verb, ok
}

func innControl(p innProfile, home *Town, seats []innOffer, selected int, name string, button int) (ui.TownSquareControl, bool) {
	var seat *innSeat
	if selected >= 0 && selected < len(seats) {
		seat = &innSeat{word: seats[selected]}
	}
	verb, found := innButtonVerb(name)
	if !found || !innButtons(p, home, seat, true)[verb] {
		return ui.TownSquareControl{}, false
	}
	row := selected + 1
	if verb == innSleep {
		row = 0
	}
	if verb == innExit {
		row = len(seats) + 1
	}
	if verb == innTalk && seat.hire() {
		return ui.TownSquareControl{Kind: ui.TownSquareControlButton, Door: button}, true
	}
	return ui.TownSquareControl{Kind: ui.TownSquareControlDoor, Door: row}, true
}

func (t *townScreen) TownSquareButton(i int) ui.TownAction {
	d := t.townDescription()
	r := d.Room(t.roomName())
	if r == nil || i < 0 || i >= len(r.Buttons) {
		return ui.TownAction{}
	}
	seats, _ := t.state().innSeats()
	selected := t.composed.seat
	if selected < 0 || selected >= len(seats) {
		return ui.TownAction{}
	}
	verb, ok := innButtonVerb(r.Buttons[i].Name)
	if !ok {
		return ui.TownAction{}
	}
	return t.innAction(bankInnProfile, &innSeat{word: seats[selected]}, verb)
}

type innTalkRequest struct {
	building       TownBuilding
	offer          TownOffer
	npc, candidate int
	key            string
	word           *innOffer
	plain, loaded  bool
	payload        []byte
}

// startInnTalk opens one cursor and applies the profile's quest timing.
// File and section sources retain their distinct missing-text policies.
func (t *townScreen) startInnTalk(p innProfile, req innTalkRequest) (ui.TownAction, bool) {
	if !req.plain && p.keepSpeakers && req.building == TownTavern && req.candidate == 0 && !innCanTalk(t.sess.Town, req.offer) {
		return ui.TownAction{}, false
	}
	if !req.plain && p.joinOnOpen && req.building == TownTavern && req.candidate == 0 && t.sess.joinAtTalk(t.in.townInstall(), req.offer.NPC) {
		t.composeShopFaces()
	}
	wasOpen := t.room == roomTalk
	if req.candidate != 0 {
		t.resetTownSpeech()
		t.room, t.dialogueBuilding, t.npc, t.offer = roomTalk, TownTavern, 0, TownOffer{}
		t.mercenaryTalk, t.talkDiagnostic = req.candidate, ""
		t.composeShopFaces()
	}
	payload := req.payload
	if !req.loaded {
		if p.sectionText {
			var err error
			payload, err = readSecondTownTalk(t.in, req.key)
			if err != nil {
				return ui.TownAction{Msg: err.Error()}, false
			}
		} else if req.candidate != 0 {
			payload = t.townPayload()
		} else {
			payload = t.townPayloadFor(req.building, req.offer, req.npc)
		}
	}
	if p.applyTalk && req.word != nil {
		t.state().talkTo(*req.word)
	}
	if payload == nil && req.candidate == 0 && !req.loaded {
		if p.applyTalk && req.word != nil {
			t.endInnTalk(*req.word)
		}
		return ui.TownAction{}, false
	}
	if p.sectionText {
		if t.adapter == nil {
			t.adapter = destinationTownCampaign{t}
			t.defaults = townDescriptions["rom2-town1"]
		}
		if req.word != nil {
			t.state().talking = req.word
		}
		wasOpen = false
	} else if req.candidate == 0 {
		t.room, t.dialogueBuilding, t.npc, t.offer = roomTalk, req.building, req.npc, req.offer
		t.mercenaryTalk, t.talkDiagnostic = 0, ""
	}
	t.talkPage().openSection(req.key, payload)
	if req.candidate == 0 {
		t.resetTownSpeech()
	}
	if p.sectionText {
		t.resolver = t.townSpeakers()
	}
	read := t.openTalk(t, payload, wasOpen)
	if req.candidate != 0 {
		t.dialogue.hasPortrait = true
		if !read {
			t.talkDiagnostic = fmt.Sprintf("Mercenary dialogue unavailable: npc%02d", req.candidate)
		}
	}
	if !req.plain && p.queueOffers && req.candidate == 0 && req.offer.Mission > 0 && req.offer.Index >= 0 && t.sess.Town != nil {
		if req.building == TownTavern {
			t.innQueue = append(t.innQueue, req.offer.Mission)
		} else if mission, ok := t.sess.Town.Take(req.building, req.offer.Index); ok {
			t.sess.Town.registerOfferedMission(mission)
			t.markWorldSelected(mission)
		}
	}
	return ui.TownAction{}, true
}

// platoonTalkObject is the object a Platoon record of mercenary type typ
// stands for: the first carried member of that type, else the stock unit of
// that type, which counts as live after the party.
func platoonTalkObject(typ int, party []mapload.PartyMember) tavernTalkObject {
	o := tavernTalkObject{live: len(party), sheet: typ, stock: typ, human: typ > 0 && !mapload.TableEdition(nil).TavernUnit(typ)}
	for i, p := range party {
		if p.Membership().Matches(mapload.TavernMember, typ) {
			o.live, o.stock = i, 0
			break
		}
	}
	return o
}
