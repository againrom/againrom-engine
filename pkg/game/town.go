package game

import (
	"slices"
	"sort"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// initialPlayerPurse is written when the campaign participant is created. It
// belongs to the player-level campaign state represented by Town, not to a
// hero, inventory subject, or UI label. PARTY-MONEY-024 establishes the
// shipped participant factory's value and that a later companion join adds
// zero to it.
const initialPlayerPurse = 100

// TownBuilding names which of a chapter's three offer lists a row came from
// (REG-SCN-064: the tavern's InnMission, the shop's ShopMission and the
// school's TCMission, one reader each).
type TownBuilding int

const (
	// TownTavern is the inn — the per-NPC list, and the only one of the
	// three that pairs with a second array.
	TownTavern TownBuilding = iota
	// TownShop is the shop, whose section also carries the two price
	// bounds.
	TownShop
	// TownSchool is `TC` — REG-SCN-064 established that the view reading
	// TCMission is the school's.
	TownSchool
	// TownGate is no building and names no offer list. It is the source of the
	// line the gate answers with when nothing is on offer (TOWN-475).
	TownGate
)

func (b TownBuilding) String() string {
	switch b {
	case TownTavern:
		return "tavern"
	case TownShop:
		return "shop"
	case TownSchool:
		return "school"
	case TownGate:
		return "gate"
	}
	return "unknown"
}

// offerRef addresses one element of one building's list in one chapter: the
// three coordinates that make an offer unique across a whole campaign. It is
// the key of the consumed set, which is what "the shop removed element 0"
// becomes here — a mark rather than a mutation, so the Campaign a Town was
// built over stays exactly what the registry said.
type offerRef struct {
	chapter  int
	building TownBuilding
	index    int
}

// TownOffer is one row of a building's list as the town shows it: which
// element it is, what mission it holds, and — for the tavern alone — the NPC
// holding it.
//
// Mission IS ZERO FOR THE INN'S SENTINEL and for nothing else. REG-SCN-064:
// InnMission[i] == 0 means the NPC at InnNPC[i] has nothing to give, and the
// original's zero arm speaks a line and queues nothing. So a zero here is a
// row that is shown, can be spoken to, and yields no mission — not a row that
// is missing.
//
// Index is negative only for a kept speaker (TavernRoster), which is not an
// offer and is never taken.
type TownOffer struct {
	Index   int
	Mission int
	NPC     int
}

// Town owns current campaign progress, consumed offers, visit history and
// the resources that survive a mission. The current main also exists before
// any town building has opened; entering a side mission does not replace it.
type Town struct {
	second      *secondCampaign
	camp        Campaign
	cityObjects *cityObjectTopology
	// progress holds ordinary current campaign records when present.
	progress *campaignProgress
	records  campaignRecords
	// main owns native progress until an ordinary campaign record supplies it.
	main     int
	selected int

	// open latches the first time the campaign reaches a town offer and is
	// never cleared.
	open bool

	gold int

	// knowledge is the local Player's Diary the last won mission ended with. It
	// is replaced whole, never edited, and a new campaign starts without one.
	knowledge []sim.SavedDiary

	// A mission is marked once and paid on that same transition.
	won map[int]bool

	available map[int]bool

	// taken records consumed stable offer indices. Native authored lists use
	// it as their removal set; ordinary arrays already contain the removals.
	taken       map[offerRef]bool
	offerLabels *currentOfferLabels

	// Consumed native campaign grants, separate from the immutable registry.
	// Original campaigns keep their current AddHero array in progress.
	heroGrants map[int]bool

	// mercPool, mercEnabled and mercHired are the tavern economy's three
	// independent per-type records. Index zero is unused because the shipped
	// type ids are one-based.
	mercPool     [16]int
	mercCapacity [16]int
	mercEnabled  [16]bool
	mercHired    [16]bool

	// cityGroups is the Player's group membership, replaced and never edited
	// in place so a copied Town shares it.
	cityGroups []cityLiveGroup

	// documents is the campaign's document collection: one append-only list
	// of (value, kind) pairs, deduplicated on the pair, in the order the
	// campaign granted them (REG-SCN-097).
	//
	// IT IS TOWN STATE FOR THE REASON EVERY OTHER FIELD HERE IS: it survives
	// a mission, it belongs to the campaign rather than to a world, and it is
	// what a save carries. The original owns it on the campaign record at
	// `app+0x548` and this build has no such record; Town is where this
	// build already keeps what survives a mission.
	documents []Document

	// docMission is the highest mission whose grants have been collected, and
	// it is the whole of REG-SCN-097's monotonic guard: the original's loader
	// runs only for a mission STRICTLY HIGHER than the current one and
	// returns 0 for a lower one, so replaying an earlier mission grants
	// nothing a second time.
	//
	// THAT CLAUSE IS MEDIUM (REG-SCN-097's own confidence line: one compare
	// read in a decompiled caller, with the campaign's mission sequence never
	// driven), so this field reproduces a guard research states rather than a
	// guard research proved. The append is deduplicated on the pair as well,
	// which is High, so the guard's failure mode here is a missing grant
	// rather than a duplicated one.
	docMission int

	// docPayload is the Againrom-only payload read from the carrier pairs of
	// the loaded SAV's document list, held from LOAD to the next SAVE and
	// written back as one block. docPayloadError says why carriers that were
	// present did not decode; they are then dropped. Neither is a document:
	// no panel, grant or counter sees them.
	docPayload      *sav.DocPayload
	docPayloadError string

	// lastMap is the map name of the last mission played, without its
	// directory, as a town SAV's head names it: every pure original town in
	// the lawful corpus names the map of the mission played before it. A
	// loaded SAV supplies its own head value; a mission's end replaces it.
	lastMap string
}

// Document is one element of the campaign document collection: the registry
// value the campaign granted, and which of the two kinds it is
// (REG-SCN-097 — 1 text, 0 picture, discriminated by each append routine's
// own reject polarity agreeing with its own kind literal).
//
// THE VALUE IS NOT AN INDEX. It formats straight into a resource path,
// `graphics\interface\Docs\%d.bmp` for a picture and `main\text\Docs\%d.txt`
// for a text (MISSION-DOC-021), so two elements of different kinds may carry
// the same value and name two different files — which mission 10's text 1
// and mission 60's picture 1 do on a stock scenario.
type Document struct {
	Value int
	Kind  int
}

// The two document kinds, as the appending routines assign them
// (REG-SCN-097).
const (
	DocumentPicture = 0
	DocumentText    = 1
)

// NewTown builds the town over a campaign. The zero campaign is legal and
// yields a town with no chapter, which is what an install declaring no campaign
// has always meant one level up.
func NewTown(c Campaign) *Town {
	t := &Town{
		camp:      c,
		gold:      initialPlayerPurse,
		won:       make(map[int]bool),
		available: make(map[int]bool),
		taken:     make(map[offerRef]bool),
	}
	if len(c.Main) != 0 {
		t.main = c.Main[0]
	}
	t.records = newCampaignRecords(c, t.main)
	for i, n := range c.MercenaryCount {
		if i+1 >= len(t.mercPool) {
			break
		}
		if n > 0 {
			t.mercPool[i+1] = n
			t.mercCapacity[i+1] = n
		}
	}
	return t
}

func newTownFromCampaignProgress(c Campaign, p *campaignProgress) *Town {
	t := NewTown(c)
	if p == nil {
		return t
	}
	t.progress = p
	// A campaign projection says where the player is, not that every consumer
	// behind the town boundary is already lawful. The first building-offered
	// mission is the existing campaign boundary; pre-town main records stay
	// closed until ordinary mission completion reaches it.
	t.open = campaignTownOpen(c, p.main.mission)
	t.mercPool = p.mercenaryWorking
	t.mercCapacity = p.mercenaryPristine
	t.mercHired = p.mercenaryHired
	for _, typ := range p.permanent {
		if typ > 0 && typ < len(t.mercEnabled) {
			t.mercEnabled[typ] = true
		}
	}
	for _, r := range append([]campaignProgressRecord{p.main}, p.children...) {
		if r.announced {
			t.available[r.mission] = true
		}
	}
	for _, d := range p.documents {
		t.addDocument(d)
	}
	t.docMission = p.main.mission
	return t
}

// CollectDocuments appends whatever mission n's own `[Mission<n>]` section
// grants to the campaign document collection (REG-SCN-097).
//
// IT IS CALLED WHEN A MISSION IS ENTERED and it is idempotent for a mission
// already collected, twice over: the monotonic guard refuses a mission at or
// below the highest one collected, and the append itself is deduplicated on
// the (value, kind) pair. Re-entering mission 10 from a save therefore grants
// nothing, and so does replaying it.
//
// TEXT IS APPENDED BEFORE PICTURE within one section. No shipped section
// carries both keys, so the order is unobservable on stock content and is
// written down rather than left to whichever loop came first.
//
// A mission whose section carries neither key still MOVES THE GUARD, which is
// what makes the guard a record of how far the campaign has come rather than
// of the last section that happened to grant something.
func (t *Town) CollectDocuments(n int) {
	if t == nil {
		return
	}
	if t.progress != nil {
		// The original persists only the pair-deduplicated collection, not
		// againrom's authored monotone guard. A restored side mission can be
		// numerically below the main record and still grant a new pair.
		ch := t.camp.Chapters[n]
		for _, v := range ch.TextDocuments {
			t.addDocument(Document{Value: v, Kind: DocumentText})
		}
		for _, v := range ch.PictureDocuments {
			t.addDocument(Document{Value: v, Kind: DocumentPicture})
		}
		t.progress.documents = append(t.progress.documents[:0], t.documents...)
		return
	}
	if n <= t.docMission {
		return
	}
	t.docMission = n
	t.appendGrants(n)
}

func (t *Town) appendGrants(n int) {
	ch := t.camp.Chapters[n]
	for _, v := range ch.TextDocuments {
		t.addDocument(Document{Value: v, Kind: DocumentText})
	}
	for _, v := range ch.PictureDocuments {
		t.addDocument(Document{Value: v, Kind: DocumentPicture})
	}
}

func (t *Town) collectWonDocuments(n int) {
	t.appendGrants(n)
	if t.progress != nil {
		t.progress.documents = append(t.progress.documents[:0], t.documents...)
	}
	if n > t.docMission {
		t.docMission = n
	}
}

// addDocument is the append REG-SCN-097 describes: one growable list, no
// clear, and the pair deduplicated.
func (t *Town) addDocument(d Document) {
	for _, have := range t.documents {
		if have == d {
			return
		}
	}
	t.documents = append(t.documents, d)
}

// Documents is the campaign document collection as it stands, in the order it
// was granted. The slice is a fresh copy: a caller that held the field could
// otherwise write into campaign state through a reader.
func (t *Town) Documents() []Document {
	if t == nil || len(t.documents) == 0 {
		return nil
	}
	out := make([]Document, len(t.documents))
	copy(out, t.documents)
	return out
}

// DocumentMission is the highest mission whose grants have been collected —
// the monotonic guard's own value, read by the save and by nothing else.
func (t *Town) DocumentMission() int {
	if t == nil {
		return 0
	}
	return t.docMission
}

// Arrive latches the town open. It is called from the one place that can know
// the campaign has reached a town — FinishMission, on the arm where the offer
// it computed is a town offer — and calling it twice is calling it once.
func (t *Town) Arrive() {
	if t == nil {
		return
	}
	if t.progress == nil {
		if first, ok := t.camp.TownBegins(); ok && t.main < first {
			for _, main := range t.camp.Main {
				if main >= first && t.camp.offers(main) && !t.won[main] {
					t.main = main
					t.records.loadMain(t.camp, main, false)
					t.selected = main
					break
				}
			}
		}
	}
	t.open = true
}

// Open reports whether the campaign has reached the town yet.
func (t *Town) Open() bool { return t != nil && t.open }

// Gold is what the player has.
func (t *Town) Gold() int {
	if t == nil {
		return 0
	}
	return t.gold
}

// Won records that mission n is finished and pays what its own section
// declares (REG-SCN-067).
//
// THE PAYMENT IS ADDED ON THE STATEMENT THAT MARKS THE WIN AND ONLY WHEN THE
// MARK IS NEW, so no path can pay one mission twice — and the mission leaves
// the available list on the same statement, because a mission that is done is
// not one the gates should still be offering.
//
// It reads Payment out of the CHAPTER MAP and not out of a table here: 9 of the
// shipped 24 sections declare one and the rest pay nothing, which falls out of
// a missing key rather than out of a list of exceptions.
func (t *Town) Won(n int) (side bool, accepted bool) {
	if t == nil || n <= 0 || t.won[n] {
		return false, false
	}
	if t.progress != nil {
		r := t.progress.record(n)
		if r == nil {
			return false, false
		}
		payment := r.payment
		unlocks := append([]int(nil), r.enableMercenary...)
		side, accepted = t.progress.complete(t.camp, n)
		if !accepted {
			return false, false
		}
		t.won[n] = true
		t.collectWonDocuments(n)
		delete(t.available, n)
		// The restored record supplies ROM1's scenario payment. Againrom's
		// separately authored transition reward remains a product rule and is
		// applied on the same boundary as it is for a fresh campaign.
		t.gold += payment + t.camp.TransitionRewards[n]
		for _, typ := range unlocks {
			if typ > 0 && typ < len(t.mercEnabled) {
				t.mercEnabled[typ] = true
			}
		}
		// Announced records are the restored availability source. A main
		// advance may age children out and load new unannounced candidates, so
		// rebuild the set from the records that survived the transition.
		t.refreshAvailable()
		return side, true
	}
	t.won[n] = true
	t.collectWonDocuments(n)
	delete(t.available, n)
	t.gold += t.camp.Reward(n)
	t.applyMercenaryUnlocks(n)
	side = t.records.removeChild(n)
	routed := n == t.main
	t.advanceMain(n)
	// REG-SCN-063: a won main mission whose own section names its successor
	// routes the party there and latches that record's announce flag.
	if next, ok := t.camp.AutoAdvance(n); routed && ok && next == t.main {
		t.announceMission(next)
	}
	t.refreshAvailable()
	t.selected = t.main
	return side, true
}

func (t *Town) advanceMain(completed int) {
	if completed < t.main || !containsMission(t.camp.Main, completed) {
		return
	}
	previous := t.main
	t.main = completed
	if next, ok := t.camp.NextMission(completed); ok {
		t.main = next.Mission
	}
	if t.main != previous {
		t.records.loadMain(t.camp, t.main, false)
	}
}

// Mission activation changes a native main only after the new world is ready.
// An ordinary restored campaign record already owns its main independently of
// the selected mission, including a side mission or a replay of an earlier one.
func (t *Town) activateMission(mission int) {
	if t != nil && t.progress == nil {
		if mission > t.main && containsMission(t.camp.Main, mission) {
			t.main = mission
			t.records.loadMain(t.camp, mission, false)
			t.refreshAvailable()
		}
		t.selectMission(mission)
	}
}

func (t *Town) applyMercenaryUnlocks(mission int) {
	if t == nil {
		return
	}
	for _, typ := range t.camp.Chapters[mission].EnableMercenary {
		if typ > 0 && typ < len(t.mercEnabled) {
			t.mercEnabled[typ] = true
		}
	}
}

// MercenaryPool reports the surviving stock for one type.
func (t *Town) MercenaryPool(typ int) int {
	if t == nil || typ <= 0 || typ >= len(t.mercPool) {
		return 0
	}
	return t.mercPool[typ]
}

// MercenaryCapacity is the campaign-authored original size of one squad.
// Unlike the live pool it is not consumed by hiring or casualties, so the
// tavern can show the surviving count as current/original.
func (t *Town) MercenaryCapacity(typ int) int {
	if t == nil || typ <= 0 || typ >= len(t.mercCapacity) {
		return 0
	}
	return t.mercCapacity[typ]
}

// MercenaryEnabled reports whether a completed mission permanently unlocked
// one type.
func (t *Town) MercenaryEnabled(typ int) bool {
	return t != nil && typ > 0 && typ < len(t.mercEnabled) && t.mercEnabled[typ]
}

// MercenaryHired reports the immediate tavern toggle state for one type.
func (t *Town) MercenaryHired(typ int) bool {
	return t != nil && typ > 0 && typ < len(t.mercHired) && t.mercHired[typ]
}

// mercenaryHire sets the hired flag and charges the fee; pool untouched until merge.
func (t *Town) mercenaryHire(typ, cost int) (count int, ok bool) {
	if t == nil || typ <= 0 || typ >= len(t.mercPool) || t.mercHired[typ] ||
		t.mercPool[typ] <= 0 || cost < 0 || t.gold < cost {
		return 0, false
	}
	count = t.mercPool[typ]
	t.mercHired[typ] = true
	t.gold -= cost
	return count, true
}

// mercenaryReturn clears the hired flag and refunds the fee; pool untouched.
func (t *Town) mercenaryReturn(typ, count, refund int) bool {
	if t == nil || typ <= 0 || typ >= len(t.mercPool) || !t.mercHired[typ] || count <= 0 {
		return false
	}
	t.mercHired[typ] = false
	t.gold += refund
	return true
}

// mercenaryBoundary is MERC-DEATH-006's end-of-mission merge: the half of the
// tavern economy that runs when a mission ends rather than when a man is hired.
//
// live is that type's mercenaries still standing when the mission ended, one
// element per type, index zero unused as everywhere else here.
//
// THE TWO ARMS ARE NOT SYMMETRICAL. For a type that was hired the pool becomes
// the live tally outright, so every man who died is gone from the pool for
// good. For a type that was not hired the pool grows by one, and only while it
// is under the campaign-authored original size held in mercCapacity, which is
// the claim's MercenaryCount. A squad wiped on a mission therefore takes as
// many further missions to rebuild as it lost men, and only while it is left at
// home. A type the campaign never authored has capacity 0 and does not grow.
//
// Then all fifteen hire flags are zeroed, so no hire survives a mission.
// That is the same fact DropHired states from the party's side.
func (t *Town) mercenaryBoundary(live [16]int) {
	if t == nil {
		return
	}
	for typ := 1; typ < len(t.mercPool); typ++ {
		if t.mercHired[typ] {
			t.mercPool[typ] = live[typ]
			continue
		}
		if t.mercPool[typ] < t.mercCapacity[typ] {
			t.mercPool[typ]++
		}
	}
	t.mercHired = [16]bool{}
}

func (t *Town) spend(cost int) bool {
	if t == nil || cost < 0 || t.gold < cost {
		return false
	}
	t.gold -= cost
	return true
}

func mercenaryUnitPrice(mission int) int {
	ladder := [...]int{10, 15, 20, 40, 60, 80, 100, 600, 800, 1000, 6000, 8000, 10000}
	i := mission/10 - 3
	if i < 0 || i >= len(ladder) {
		return 0
	}
	return ladder[i]
}

// Done reports whether mission n has been finished.
func (t *Town) Done(n int) bool {
	if t == nil {
		return false
	}
	if t.progress != nil && n > 0 && n%mainMissionStride == 0 && n < t.progress.main.mission {
		return true
	}
	return t.won[n]
}

// finishedCount is how many missions have been finished, whatever they were.
//
// It is one of the three inputs the shop's seed is mixed from, and it is the
// count and not the set because the seed needs a number that moves on every
// win, including a side mission that leaves the chapter where it is.
func (t *Town) finishedCount() int {
	if t == nil {
		return 0
	}
	return len(t.won)
}

func (t *Town) currentMain() int {
	if t == nil {
		return 0
	}
	if t.progress != nil {
		return t.progress.main.mission
	}
	return t.main
}

// Chapter is the current main mission, including the closed pre-town chapter.
// Completing the final main leaves no further chapter.
func (t *Town) Chapter() int {
	main := t.currentMain()
	if t != nil && t.won[main] && (t.progress == nil || t.camp.TerminalMission(main)) {
		return 0
	}
	return main
}

// ChapterData keeps current main identity even when its section has no
// building data. Missing section values remain empty.
func (t *Town) ChapterData() Chapter {
	if t == nil {
		return Chapter{}
	}
	if t.progress != nil {
		return t.progress.chapter()
	}
	main := t.Chapter()
	chapter := t.camp.Chapters[main]
	chapter.Mission = main
	return chapter
}

// Available is the sorted presentation projection of accepted live records.
func (t *Town) Available() []int {
	if t == nil {
		return nil
	}
	out := make([]int, 0, len(t.available))
	for n := range t.available {
		if !t.won[n] {
			out = append(out, n)
		}
	}
	sort.Ints(out)
	return out
}

// Offers is the live chapter's un-taken rows for one building, in the file's
// own order.
//
// THE INN'S PAIRING WALKS THE SHORTER OF THE TWO ARRAYS (plan R-1). They are
// equal in 22/22 shipped sections and a modded scenario need not be; pairing to
// the shorter loses entries rather than reading past one.
//
// THE ZERO SENTINEL IS KEPT AND ONLY THE INN CAN HOLD ONE. A tavern row with
// no mission is an NPC with nothing to give and is shown as one; a shop or
// school element of zero would be a value the registry does not ship and it is
// dropped, because those two arrays have no positional partner for a hole to
// mean anything against.
func (t *Town) Offers(b TownBuilding) []TownOffer {
	if !t.Open() || b < TownTavern || b > TownSchool {
		return nil
	}
	ch := t.ChapterData()
	var labels []TownOffer
	if t.progress != nil {
		labels = t.currentOfferLabels(b, ch)
	}
	var out []TownOffer
	if b == TownTavern {
		n := len(ch.Inn)
		if len(ch.InnNPC) < n {
			n = len(ch.InnNPC)
		}
		for i := 0; i < n; i++ {
			if t.progress == nil && t.taken[offerRef{ch.Mission, b, i}] {
				continue
			}
			out = append(out, TownOffer{Index: currentOfferIndex(labels, i), Mission: ch.Inn[i], NPC: ch.InnNPC[i]})
		}
		return out
	}
	list := ch.Shop
	if b == TownSchool {
		list = ch.School
	}
	for i, m := range list {
		if m <= 0 || t.progress == nil && t.taken[offerRef{ch.Mission, b, i}] {
			continue
		}
		out = append(out, TownOffer{Index: currentOfferIndex(labels, i), Mission: m})
	}
	return out
}

// TavernRoster is the tavern's talk list in the chapter's registry order: every
// live offer, with a kept speaker in the place of each pair the live arrays no
// longer hold.
//
// A pair leaves both arrays when its mission is accepted (SAV-CAMPAIGN-081) and
// a talk cell follows its array position (SAV-1112); the speaker is kept so the
// conversation can be heard again (DIV-1529). A kept speaker's Index is -1 minus
// its registry position, which Take refuses, so hearing it again queues nothing
// (DIV-1530).
//
// NOTHING IS STORED. The registry is immutable and the live arrays are what a
// save holds, so a game the engine played, one it reloaded and one an original
// save carries all list the same speakers, and the chapter's next advance
// replaces the registry rows with the next chapter's.
func (t *Town) TavernRoster() []TownOffer {
	live := t.Offers(TownTavern)
	if !t.Open() {
		return live
	}
	registry := t.camp.Chapters[t.ChapterData().Mission]
	rows := min(len(registry.Inn), len(registry.InnNPC))
	out := make([]TownOffer, 0, max(rows, len(live)))
	next := 0
	for i := 0; i < rows; i++ {
		npc, mission := registry.InnNPC[i], registry.Inn[i]
		found := -1
		for j := next; j < len(live); j++ {
			if live[j].NPC == npc && live[j].Mission == mission {
				found = j
				break
			}
		}
		if found >= 0 {
			out = append(out, live[next:found+1]...)
			next = found + 1
		} else if mission > 0 {
			out = append(out, TownOffer{Index: -1 - i, Mission: mission, NPC: npc})
		}
	}
	return append(out, live[next:]...)
}

// Take accepts one of the live chapter's offers: the mission goes on the
// available list and the entry is consumed.
//
// IT REPORTS WHAT WAS ACCEPTED AND NOT WHETHER SOMETHING HAPPENED. A zero
// mission is the inn's sentinel — the NPC has nothing to give, the original
// speaks a line and queues nothing — so this consumes nothing and answers
// (0, false), and that NPC can be spoken to again. Everything else is
// consumed and answers its mission.
//
// A MISSION ALREADY WON IS STILL CONSUMED and still answers, because the entry
// is the building's and the win is the player's: leaving it on the shelf would
// offer him a mission he has finished for the rest of the chapter. It does not
// reach the available list, which Available's own filter already refuses.
//
// The index is the stable label returned by Offers. Ordinary compacted
// arrays translate that label to the current row before consumption.
func (t *Town) Take(b TownBuilding, index int) (int, bool) {
	if !t.Open() || index < 0 {
		return 0, false
	}
	if t.progress != nil {
		return t.takeCurrentOffer(b, index)
	}
	ch := t.ChapterData()
	if ch.Mission == 0 {
		return 0, false
	}
	ref := offerRef{ch.Mission, b, index}
	if t.taken[ref] {
		return 0, false
	}
	var list []int
	switch b {
	case TownTavern:
		list = ch.Inn
	case TownShop:
		list = ch.Shop
	case TownSchool:
		list = ch.School
	}
	if index >= len(list) {
		return 0, false
	}
	m := list[index]
	if m <= 0 {
		// REG-SCN-064's zero arm: nothing is queued and — this is the part
		// that is easy to get wrong — nothing is REMOVED either. Only an
		// accepted mission is removed from both arrays.
		return 0, false
	}
	if !t.prepareOfferedRecord(m) {
		return 0, false
	}
	t.taken[ref] = true
	t.announceMission(m)
	return m, true
}

// registerInnMission consumes the first remaining pair by identity. Selection
// and announcement still run after the last matching offer has gone.
func (t *Town) registerInnMission(mission int) bool {
	if !t.Open() || mission <= 0 || !t.currentRecords().accepts(t.camp, mission) {
		return false
	}
	for _, offer := range t.Offers(TownTavern) {
		if offer.Mission == mission {
			t.Take(TownTavern, offer.Index)
			break
		}
	}
	return t.registerOfferedMission(mission)
}

func (t *Town) registerOfferedMission(mission int) bool {
	if !t.Open() || mission <= 0 || !t.prepareOfferedRecord(mission) {
		return false
	}
	if t.progress != nil {
		t.progress.selected = mission
	} else {
		t.selectMission(mission)
	}
	t.announceMission(mission)
	return true
}

func (t *Town) campaignPayment(n int) int {
	if t == nil {
		return 0
	}
	if t.progress != nil {
		if r := t.progress.record(n); r != nil {
			return r.payment
		}
	}
	return t.camp.Chapters[n].Payment
}

func (t *Town) canOpenMission(n int) bool {
	return t == nil || t.progress == nil || !t.progress.lowerMainBlocked(n)
}

func (t *Town) restoredCampaign() bool { return t != nil && t.progress != nil }

func (t *Town) selectedMission() int {
	if t == nil {
		return 0
	}
	if t.progress == nil {
		if t.selected != 0 {
			return t.selected
		}
		return t.currentMain()
	}
	return t.progress.selected
}

func (t *Town) selectMission(mission int) {
	if t != nil && t.progress != nil {
		t.progress.selectMission(mission)
	} else if t != nil && (containsMission(t.camp.Main, mission) || containsMission(t.camp.Side, mission)) {
		t.selected = mission
	}
}

func (t *Town) takeAddHeroes(chapter int) []int {
	return t.takeAddHeroesExcept(chapter, nil)
}

// takeAddHeroesExcept consumes the chapter's pending grants except those keep
// names, which stay pending where the save records them.
func (t *Town) takeAddHeroesExcept(chapter int, keep func(npc int) bool) []int {
	if t == nil {
		return nil
	}
	if t.progress != nil {
		return t.progress.takeAddHeroesExcept(chapter, keep)
	}
	pending := t.pendingNativeHeroGrants(chapter)
	var out []int
	kept := false
	for _, npc := range pending {
		if keep != nil && keep(npc) {
			kept = true
			continue
		}
		out = append(out, npc)
	}
	if len(pending) != 0 && !kept {
		if t.heroGrants == nil {
			t.heroGrants = make(map[int]bool)
		}
		t.heroGrants[chapter] = true
	}
	return out
}

// takeAddHero consumes the one pending grant of npc in the chapter and reports
// whether it was pending.
func (t *Town) takeAddHero(chapter, npc int) bool {
	if t == nil {
		return false
	}
	if t.progress != nil {
		return t.progress.takeAddHero(chapter, npc)
	}
	if !slices.Contains(t.pendingNativeHeroGrants(chapter), npc) {
		return false
	}
	if t.heroGrants == nil {
		t.heroGrants = make(map[int]bool)
	}
	t.heroGrants[chapter] = true
	return true
}

func (t *Town) firstMapPoint() (bool, bool) {
	if t == nil || t.progress == nil {
		return false, false
	}
	return t.progress.firstMapPoint, true
}

func (t *Town) selectedMarkerMissions() []int {
	if t == nil || t.progress == nil {
		return nil
	}
	return t.progress.selectedMarkers()
}

// permanentLoaded is the loaded append-ordered unlock list, nil without one.
func (t *Town) permanentLoaded() []int {
	if t == nil || t.progress == nil {
		return nil
	}
	return t.progress.permanent
}
