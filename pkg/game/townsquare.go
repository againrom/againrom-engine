package game

import (
	_ "embed"
	"image"

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

// townPages are the page kinds the ROM1 town screen draws its rooms with.
var townPages = []string{"tavern", "shop", "school", "world-map"}

// townVocabulary is the hook, condition, value, event and page names the
// ROM1 campaign answers.
func townVocabulary() town.Vocabulary {
	v := town.Vocabulary{Events: roomEvents, Pages: townPages}
	for name := range townHooks {
		v.Hooks = append(v.Hooks, name)
	}
	for name := range townConditions {
		v.Conditions = append(v.Conditions, name)
	}
	for name := range roomValues {
		v.Values = append(v.Values, name)
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
	id := chapterTown
	var book *townBook
	if t != nil && t.sess != nil && t.sess.Town != nil {
		id, book = t.sess.Town.townID(), t.sess.Town.camp.Towns
	}
	if book == nil {
		book = townBookOf(t.profile())
	}
	return book.description(id, townHooksChapters)
}

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
		return t.sess.Town.gateMission() != -1 || len(t.sess.Town.travelTowns()) > 0
	},
	"save-admitted": func(t *townScreen) bool { return t.saveAdmitted() },
}

// townHooks are the campaign hooks the description names, each run for the
// room its step belongs to. Their bodies are the campaign's room logic; their
// order is the description's.
var townHooks = map[string]func(t *townScreen, room townRoom){
	"gate-closed": func(t *townScreen, _ townRoom) {
		t.composeShopFaces()
		if !t.openTownDialogue(TownGate, TownOffer{}, 0) {
			t.squareAction = ui.TownAction{Msg: "Town dialogue unavailable: " + townGateTextPath}
		}
	},
	"release-audio": func(t *townScreen, _ townRoom) { t.destroyRoomAudio() },
	"set-room":      func(t *townScreen, room townRoom) { t.room = room },
	"talk-clear":    func(t *townScreen, _ townRoom) { t.npc, t.offer, t.said = 0, TownOffer{}, 0 },
	"tip":           func(t *townScreen, room townRoom) { t.loadTip(room) },
	"school-reset": func(t *townScreen, _ townRoom) {
		t.schoolSpent = [schoolLatchCount]bool{}
		t.enterSchoolPage()
		t.clearSchoolSelection()
	},
	"shop-shelf": func(t *townScreen, _ townRoom) {
		t.packBase, t.shopBook = 0, false
		t.openStockedShopShelf()
	},
	"shop-interior": func(t *townScreen, _ townRoom) { t.shopPage().Enter() },
	"faces":         func(t *townScreen, _ townRoom) { t.composeShopFaces() },
	"offer-on-entry": func(t *townScreen, room townRoom) {
		building := TownShop
		if room == roomSchool {
			building = TownSchool
		}
		if offers := t.sess.Town.Offers(building); len(offers) > 0 {
			t.openOfferDialogue(building, offers[0], 0)
		}
	},
	"tavern-interior":     func(t *townScreen, _ townRoom) { t.tavernPage().Enter() },
	"tavern-detail-clear": func(t *townScreen, _ townRoom) { t.clearTavernDetail() },
	"tavern-selection":    func(t *townScreen, _ townRoom) { t.activateTavernSelection(t.tavernCandidates()) },
	"navigate":            func(t *townScreen, _ townRoom) { t.enterWorldMap() },
	"inn-queue-commit":    func(t *townScreen, _ townRoom) { t.commitInnQueue() },
	"trade-cancel":        func(t *townScreen, _ townRoom) { t.shopClear() },
	"dialogue-reset": func(t *townScreen, _ townRoom) {
		t.dialogueRevision++
		t.resetTownSpeech()
	},
	"tavern-leave": func(t *townScreen, _ townRoom) { t.leaveTavernInterior() },
	"shop-reset":   func(t *townScreen, _ townRoom) { t.shopPage().Reset() },
	"school-leave": func(t *townScreen, _ townRoom) { t.leaveSchoolTraining() },
}

// squareView answers the square's composer scene, building it on first use
// over the presentation's process-scoped town state.
func (t *townScreen) squareView() *town.Scene {
	if t.square == nil {
		t.square = newSquareScene(t, t.townDescription())
		// In original mode the wildlife draws are raw draws of the shared
		// stream instead of the description's own generator (TOWN-505).
		if st := t.draws.stream(random.TownWildlife); st != nil && st.Shared() {
			t.square.SetRawDraw("wildlife", st.Raw)
		}
	}
	return t.square
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
func (t *townScreen) TownSquareActive(active bool) {
	if t == nil || t.sess == nil {
		return
	}
	t.squareView().SetActive(active, t.AtTownSquare())
}

// TownSquarePointer receives each admitted update, not just mask transitions
// (TOWN-399).
func (t *townScreen) TownSquarePointer(p image.Point) {
	if t == nil || t.sess == nil || !t.AtTownSquare() {
		return
	}
	t.TownSquareActive(true)
	t.squareView().Pointer(p)
}

// AdvanceTownSquareAnimation is called by production composition. Update and
// hit-test/view readers cannot advance this paint-owned clock.
func (t *townScreen) AdvanceTownSquareAnimation() {
	if t == nil || t.sess == nil || !t.AtTownSquare() {
		return
	}
	t.squareView().Advance()
}

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

// townSquareScene is the composed square ui draws and hit-tests.
type townSquareScene struct{ t *townScreen }

func (s townSquareScene) Size() image.Point { return s.t.squareView().Size() }

func (s townSquareScene) Paint(dst *image.RGBA) { s.t.squareView().Paint(dst, "") }

func (s townSquareScene) ControlAt(p image.Point) (ui.TownSquareControl, bool) {
	h := s.t.squareView().HotspotAt(p)
	switch {
	case h == nil:
		return ui.TownSquareControl{}, false
	case h.Row != nil:
		return ui.TownSquareControl{Kind: ui.TownSquareControlDoor, Door: *h.Row}, true
	case town.Menu(h.Click) != "":
		return ui.TownSquareControl{Kind: ui.TownSquareControlMenu}, true
	}
	return ui.TownSquareControl{}, false
}

func (s townSquareScene) TipAt(p image.Point) (int, bool) {
	h := s.t.squareView().HotspotAt(p)
	if h == nil {
		return 0, false
	}
	return h.TipSlot()
}

// TownMusicTrack answers the square's own track from the description.
func (t *townScreen) TownMusicTrack() (string, bool) {
	scene, _ := t.TownMusic()
	if scene != ui.MusicTown {
		return "", false
	}
	return t.townDescription().Music.Track, true
}

// boundedPresentationRoll is one bounded presentation draw: the runtime's
// draw when supplied, else the fallback generator, else zero.
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
