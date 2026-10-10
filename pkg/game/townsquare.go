package game

import (
	_ "embed"
	"image"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/base"
	"againrom/pkg/random"
	"againrom/pkg/render/terrain"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// rom1TownJSON is the ROM1 town square as data for the town composer. Every
// value in it carries the claim, divergence row or owner ruling behind it.
//
//go:embed towns/rom1.json
var rom1TownJSON []byte

// townDescriptions are the town descriptions a profile's edition can name, by
// that name. A description that does not decode, or names a hook or condition
// the campaign does not answer, is a build defect, so it stops the process at
// start. They are decoded in init because the hooks they are checked against
// reach the descriptions themselves.
var townDescriptions map[string]*town.Description

func init() {
	townDescriptions = map[string]*town.Description{"rom1": mustDecodeTown(rom1TownJSON)}
	TownTipPath = ROM1TownDescription().Tip.Text
}

// TownDescription is the town description the profile's edition names, or
// nil when its game's town is not a composed square. Callers must not change
// it.
func TownDescription(p base.Profile) *town.Description { return townDescriptions[p.Edition().Town] }

func mustDecodeTown(data []byte) *town.Description {
	d, err := town.Decode(data, townVocabulary())
	if err != nil {
		panic(err)
	}
	return d
}

// townVocabulary is the hook, condition, value and event names the ROM1
// campaign answers.
func townVocabulary() town.Vocabulary {
	v := town.Vocabulary{Events: roomEvents}
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

// TownTipPath is the ROM1 square's tip text, as its description names it.
var TownTipPath string

// LoadTownSquareArt resolves the square's art from the install as the ROM1
// description names it. A required entry that fails carries its address in
// the error and the square falls back to the row-button layout; an optional
// family that fails is named in Problems and leaves the others loaded.
func LoadTownSquareArt(src terrain.EntrySource) (*town.Art, error) {
	return LoadTownSquareArtFor(ROM1TownDescription(), src)
}

// LoadTownSquareArtFor is LoadTownSquareArt over description d. A game with
// no composed square loads no art.
func LoadTownSquareArtFor(d *town.Description, src terrain.EntrySource) (*town.Art, error) {
	if d == nil {
		return nil, nil
	}
	return town.LoadArt(d, townArtLoader{src})
}

// townDescription is the town description of the install's profile; a screen
// bound to no install is the first game's.
func (t *townScreen) townDescription() *town.Description {
	if t == nil {
		return ROM1TownDescription()
	}
	return TownDescription(t.in.profile())
}

// townArtLoader reads the composer's art keys from the install's containers.
type townArtLoader struct{ src terrain.EntrySource }

func (l townArtLoader) Picture(key string) (image.Image, error) {
	pic, err := readChargenBMP(l.src, key)
	if err != nil {
		return nil, err
	}
	return pic, nil
}

func (l townArtLoader) Mask(key string) (*image.Paletted, error) { return readChargenMask(l.src, key) }

func (l townArtLoader) Sprites(key string) []image.Image {
	if l.src == nil {
		return nil
	}
	frames := effectFrames(l.src, key)
	out := make([]image.Image, 0, len(frames))
	for _, f := range frames {
		if f.Width <= 0 || f.Height <= 0 {
			return nil
		}
		out = append(out, f.RGBA())
	}
	return out
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
	"gate-open":     func(t *townScreen) bool { return t.sess.Town.gateMission() != -1 },
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

// squareView answers the square's composer view, building it on first use
// over the presentation's process-scoped town state.
func (t *townScreen) squareView() *town.View {
	if t.square == nil {
		t.square = town.NewView(t.townDescription(), townSquareHost{t}, t.townProcess)
		// In original mode the wildlife draws are raw draws of the shared
		// stream instead of the description's own generator (TOWN-505).
		if st := t.draws.stream(random.TownWildlife); st != nil && st.Shared() {
			t.square.SetRawDraw("wildlife", st.Raw)
		}
	}
	return t.square
}

// townSquareHost is the ROM1 adapter the composer runs over: the install's
// art, the screen's clock and draws, its sound devices and campaign hooks.
type townSquareHost struct{ t *townScreen }

func (h townSquareHost) Art() *town.Art {
	t := h.t
	if t == nil || t.sess == nil {
		return nil
	}
	return t.in.TownSquareArt.Value()
}

func (h townSquareHost) Now() time.Time { return h.t.townAnimationNow() }

func (h townSquareHost) Draw(source string, n int) int {
	t := h.t
	switch source {
	case "animation":
		if draw := t.draws.animationDraw(); draw != nil {
			return boundedPresentationRoll(draw, nil, n)
		}
		return boundedPresentationRoll(nil, t.draws.stream(random.TownAnimation), n)
	case "ambient":
		return boundedPresentationRoll(t.draws.ambientDraw(), t.draws.stream(random.TownAmbient), n)
	}
	return 0
}

func (h townSquareHost) Seed() int64 { return h.t.sound.wildlifeSeed() }

func (h townSquareHost) Condition(name string) bool {
	if c := townConditions[name]; c != nil {
		return c(h.t)
	}
	return false
}

func (h townSquareHost) PlaySound(source, key string) town.Voice {
	t := h.t
	sample, ok := t.in.SoundBank.namedSample(key)
	if !ok {
		return nil
	}
	v := audio.Dispatch(t.roomSoundPlayer(audio.EffectsChannel), sample,
		audio.FixedRequest(source, key, audio.EffectsChannel, 128, false,
			audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}))
	if v == nil {
		return nil
	}
	return v
}

func (h townSquareHost) StopSound(v town.Voice) {
	if av, ok := v.(audio.Voice); ok {
		audio.StopReset(av)
	}
}

func (h townSquareHost) StartLoop(key string) bool {
	t := h.t
	if t.sound.ambientDevice() == nil {
		return false
	}
	sample, ok := t.in.SoundBank.namedSample(key)
	if !ok {
		return false
	}
	request := audio.FixedRequest("town-crowd", key, audio.EffectsChannel, 128, true,
		audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit})
	if ui.DeliveryOwner(t.sound.soundDevice()) != nil {
		t.squareLoop = audio.Dispatch(t.roomSoundPlayer(audio.EffectsChannel), sample, request)
	} else {
		t.squareLoop = ui.RequestAmbient(t.sound.ambientDevice(), ui.AmbientTownCrowd, sample, request)
	}
	return true
}

func (h townSquareHost) StopLoop(key string) {
	t := h.t
	if t.sound.ambientDevice() != nil {
		if ui.DeliveryOwner(t.sound.soundDevice()) != nil {
			audio.StopReset(t.squareLoop)
		} else {
			t.sound.ambientDevice().StopLoop(ui.AmbientTownCrowd)
		}
	}
	t.squareLoop = nil
}

func (h townSquareHost) LeaveSquare() {
	if h.t.audioRoom == roomSquare {
		h.t.destroyRoomAudio()
	}
}

func (h townSquareHost) Hook(name, room string) {
	if hook := townHooks[name]; hook != nil {
		r, ok := townRoomNames[room]
		if !ok {
			r = h.t.room
		}
		hook(h.t, r)
	}
}

// resetTownExterior returns the square's view to the state of a view just
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

func (s townSquareScene) Paint(dst *image.RGBA) { s.t.squareView().Paint(dst) }

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
	return h.Tip, true
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
