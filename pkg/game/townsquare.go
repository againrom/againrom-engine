package game

import (
	_ "embed"
	"slices"

	"againrom/pkg/base"
	"againrom/pkg/random"
	"againrom/pkg/render/terrain"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// rom1TownCatalogJSON is the first game's town catalog: its one town.
//
//go:embed towncatalogs/rom1.json
var rom1TownCatalogJSON []byte

// rom2TownCatalogJSON is the second game's town catalog: its three towns.
//
//go:embed towncatalogs/rom2.json
var rom2TownCatalogJSON []byte

// rom1TownJSON is the ROM1 town square as data for the town composer. Every
// value in it carries the claim, divergence row or owner ruling behind it.
//
//go:embed towns/rom1.json
var rom1TownJSON []byte

// rom2Town1JSON is the second game's first town as data for the town
// composer: its square and its inn page, every value beside the second-game
// claim or divergence row behind it.
//
//go:embed towns/rom2-town1.json
var rom2Town1JSON []byte

// rom2Town2JSON is the second game's second town, Kaarg, as data for the
// town composer: its square and its inn page, every value beside the
// second-game claim or divergence row behind it.
//
//go:embed towns/rom2-town2.json
var rom2Town2JSON []byte

// rom2Town3JSON is the second game's third town, the druid town, as data for
// the town composer: its square and its inn page, every value beside the
// second-game claim or divergence row behind it.
//
//go:embed towns/rom2-town3.json
var rom2Town3JSON []byte

// TownDescription is the description of the town a first-game party starts
// in: the profile's catalog town 1 under the chapters hook set; nil when the
// profile's catalog has no such town. Its art is the install's town art
// every front end shares; a screen reads the party's town's own
// (townScreen.townDescription). Callers must not change it.
func TownDescription(p base.Profile) *town.Description {
	return townBookOf(p).description(chapterTown, townHooksChapters)
}

// townPages are the room page kinds the town host draws.
var townPages = []string{"tavern", "shop", "school", "world-map", "inn"}

// townVocabulary names the host rules and campaign values descriptions use.
func townVocabulary() town.Vocabulary {
	v := town.Vocabulary{Events: append(slices.Clone(roomEvents), secondShopEvents...), Pages: slices.Clone(townPages)}
	for name := range townHooks {
		v.Hooks = append(v.Hooks, name)
	}
	for name := range townConditions {
		v.Conditions = append(v.Conditions, name)
	}
	for name := range roomValues {
		v.Values = append(v.Values, name)
	}
	for name := range secondTownConditions {
		v.Conditions = append(v.Conditions, name)
	}
	for name := range secondTownValues {
		v.Values = append(v.Values, name)
	}
	for _, words := range []*[]string{&v.Hooks, &v.Conditions, &v.Values, &v.Events, &v.Pages} {
		slices.Sort(*words)
		*words = slices.Compact(*words)
	}
	return v
}

// ROM1TownDescription answers the ROM1 town description the square is built
// from: the description of the zero profile, which is the first game's.
// Callers must not change it.
func ROM1TownDescription() *town.Description { return TownDescription(base.Profile{}) }

// LoadTownSquareArtFor resolves the square's art from the install as
// description d names it. A required entry that fails carries its address in
// the error and the square falls back to the row-button layout; an optional
// family that fails is named in Problems and leaves the others loaded. A game
// with no composed square loads no art.
func LoadTownSquareArtFor(d *town.Description, src terrain.EntrySource) (*town.Art, error) {
	if d == nil {
		return nil, nil
	}
	return town.LoadArt(d.SquareScene(), screenArtOf(src))
}

// townDescription is the description of the party's town, the town it
// stands in or left for its mission, from its campaign's catalog; a
// campaign with none reads the install profile's, and a screen bound to no
// install the zero profile's, the first game's. The screen's rooms and
// their art read it too.
func (t *townScreen) townDescription() *town.Description {
	if t == nil {
		return ROM1TownDescription()
	}
	b := t.campaignTown().book()
	e, ok := b.entries().Town(t.campaignTown().townID())
	if !ok {
		return nil
	}
	return b.description(e.ID, e.Hooks)
}

// townBook is the book of the party's campaign; a campaign with none reads
// the install profile's.
func (t *townScreen) townBook() *townBook { return t.campaignTown().book() }

func (t *townScreen) profile() base.Profile {
	if t == nil {
		return base.Profile{}
	}
	return t.in.profile()
}

// townRoomNames binds the description's room names to the screen's rooms.
var townRoomNames = map[string]townRoom{
	"square": roomSquare,
	"tavern": roomTavern,
	"shop":   roomShop,
	"school": roomSchool,
	"gates":  roomGates,
	"talk":   roomTalk,
}

func townRoomName(r townRoom) string {
	for name, room := range townRoomNames {
		if room == r {
			return name
		}
	}
	return ""
}

// townConditions answers the conditions the description names.
var townConditions = map[string]func(t *townScreen) bool{
	// The gate opens on a mission to play or a town to travel to.
	"gate-open": func(t *townScreen) bool {
		return t.campaignTown().condition("gate-open")
	},
	"save-admitted": func(t *townScreen) bool { return t.saveAdmitted() },
}

// townHooks are the campaign hooks the description names, each run for the
// room its step belongs to. Their bodies are the campaign's room logic; their
// order is the description's.
type townHook struct {
	run    func(*townScreen, string)
	offers string
}

var townHooks = map[string]townHook{
	"shop-open":  {run: secondShopOpen, offers: "scenario-bank"},
	"shop-leave": {run: func(t *townScreen, _ string) { c := t.state(); c.shop.leave(c.heroBag()) }, offers: "scenario-bank"},
	"gate-closed": {run: func(t *townScreen, _ string) {
		if t.hostRules().Talk == "section" {
			t.squareAction = t.openTownText(secondGateText)
			return
		}
		t.composeShopFaces()
		if !t.openTownDialogue(TownGate, TownOffer{}, 0) {
			t.squareAction = ui.TownAction{Msg: "Town dialogue unavailable: " + townGateTextPath}
		}
	}, offers: ""},
	"release-audio": {run: func(t *townScreen, _ string) { t.destroyRoomAudio() }, offers: ""},
	"set-room":      {run: func(t *townScreen, room string) { t.campaignTown().setRoom(room); t.composed.hero = nil }, offers: ""},
	"talk-clear":    {run: func(t *townScreen, _ string) { t.npc, t.offer, t.talkPage.part = 0, TownOffer{}, 0 }, offers: "chapter-keys"},
	"tip":           {run: func(t *townScreen, room string) { t.loadHostTip(room) }, offers: ""},
	"school-reset": {run: func(t *townScreen, _ string) {
		t.schoolSpent = [schoolLatchCount]bool{}
		t.enterSchoolPage()
		t.clearSchoolSelection()
	}, offers: "chapter-keys"},
	"shop-shelf": {run: func(t *townScreen, _ string) {
		t.packBase, t.shopBook = 0, false
		t.openStockedShopShelf()
	}, offers: "chapter-keys"},
	"shop-interior": {run: func(t *townScreen, _ string) { t.shopPage().Enter() }, offers: "chapter-keys"},
	"faces":         {run: func(t *townScreen, _ string) { t.composeShopFaces() }, offers: "chapter-keys"},
	"offer-on-entry": {run: func(t *townScreen, room string) {
		building := TownShop
		if room == "school" {
			building = TownSchool
		}
		if offers := t.sess.Town.Offers(building); len(offers) > 0 {
			t.openOfferDialogue(building, offers[0], 0)
		}
	}, offers: "chapter-keys"},
	"tavern-interior":     {run: func(t *townScreen, _ string) { t.tavernPage().Enter() }, offers: "chapter-keys"},
	"tavern-detail-clear": {run: func(t *townScreen, _ string) { t.clearTavernDetail() }, offers: "chapter-keys"},
	"tavern-selection":    {run: func(t *townScreen, _ string) { t.activateTavernSelection(t.tavernCandidates()) }, offers: "chapter-keys"},
	"navigate": {run: func(t *townScreen, _ string) {
		t.campaignTown().leave()
		if t.hostRules().Leave == "room-map" {
			t.enterWorldMap()
		}
	}, offers: ""},
	"inn-queue-commit": {run: func(t *townScreen, _ string) { t.commitInnQueue() }, offers: "chapter-keys"},
	"trade-cancel":     {run: func(t *townScreen, _ string) { t.shopClear() }, offers: "chapter-keys"},
	"dialogue-reset": {run: func(t *townScreen, _ string) {
		t.talkWindow.Touch()
		t.resetTownSpeech()
	}, offers: "chapter-keys"},
	"tavern-leave": {run: func(t *townScreen, _ string) { t.leaveTavernInterior() }, offers: "chapter-keys"},
	"shop-reset":   {run: func(t *townScreen, _ string) { t.shopPage().Reset() }, offers: "chapter-keys"},
	"school-leave": {run: func(t *townScreen, _ string) { t.leaveSchoolTraining() }, offers: "chapter-keys"},
}

// squareView answers the square's composer scene, building it on first use
// over the presentation's process-scoped town state.
func (t *townScreen) squareView() *town.Scene {
	d := t.townDescription()
	if d == nil {
		return nil
	}
	fresh := t.square == nil
	s := t.scene(d, d.Square.Name)
	if fresh && t.hostRules().Pages == "retained" {
		if st := t.draws.stream(random.TownWildlife); st != nil && st.Shared() {
			s.SetRawDraw("wildlife", st.Raw)
		}
	}
	return s
}

// resetTownExterior returns the square's scene to the state of a scene just
// built; its process-scoped state survives.
func (t *townScreen) resetTownExterior() {
	if t.square != nil {
		t.square.Reset()
	}
}

// TownSquareActive is an explicit client lifecycle policy (DIV-834/836).
// Pausing is not a blank-mask event: retained flags and sound latches remain.

// TownSquarePointer receives each admitted update, not just mask transitions
// (TOWN-399).

// AdvanceTownSquareAnimation is called by production composition. Update and
// hit-test/view readers cannot advance this paint-owned clock.

// chooseSquare runs the click program of the hotspot at list row i and
// answers the action its hooks left.
func (t *townScreen) chooseSquare(i int) ui.TownAction {
	v := t.squareView()
	h := v.HotspotForRow(i)
	if h == nil {
		return ui.TownAction{}
	}
	t.squareAction = ui.TownAction{}
	v.Click(h)
	action := t.squareAction
	t.squareAction = ui.TownAction{}
	return action
}

func boundedPresentationRoll(draw func(int) int, fallback *random.Stream, n int) int {
	if n <= 0 {
		return 0
	}
	value := 0
	if draw != nil {
		value = draw(n)
	} else if fallback != nil {
		value = fallback.Scaled(n)
	}
	value %= n
	if value < 0 {
		value += n
	}
	return value
}
