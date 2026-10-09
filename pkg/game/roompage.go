package game

import (
	"errors"
	"image"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/random"
	"againrom/pkg/render/terrain"
	"againrom/pkg/town"
)

// roomPage answers the composer page of a room the ROM1 description gives a
// scene, building it on first use.
func (t *townScreen) roomPage(room string) *town.Page {
	if p := t.pages[room]; p != nil {
		return p
	}
	if t.pages == nil {
		t.pages = map[string]*town.Page{}
	}
	p := town.NewPage(rom1Town, room, roomPageHost{t: t, room: room}, t.townProcess)
	t.pages[room] = p
	return p
}

func (t *townScreen) tavernPage() *town.Page { return t.roomPage("tavern") }

func (t *townScreen) shopPage() *town.Page { return t.roomPage("shop") }

// loadRoomSceneArt resolves a room scene's art as the ROM1 description names
// it: its pictures by entry name, and the problems of the entries that did
// not load joined into one error. A required entry that fails answers no
// pictures.
func loadRoomSceneArt(room string, src terrain.EntrySource) (map[string][]image.Image, error) {
	for i := range rom1Town.Rooms {
		r := &rom1Town.Rooms[i]
		if r.Name != room || r.Scene == nil {
			continue
		}
		art, err := town.LoadSceneArt(r.Scene, townArtLoader{src})
		if err != nil {
			return nil, err
		}
		var problems []error
		for _, p := range art.Problems {
			problems = append(problems, errors.New(p))
		}
		return art.Frames, errors.Join(problems...)
	}
	return nil, nil
}

// roomPageHost is the ROM1 adapter a room page runs over: the room's install
// art, the screen's clock and draws, its room sound and the values the
// description names.
type roomPageHost struct {
	t    *townScreen
	room string
}

func (h roomPageHost) Art() *town.Art {
	t := h.t
	if t == nil || t.sess == nil {
		return nil
	}
	switch h.room {
	case "tavern":
		if a := t.in.TownTavernArt.Value(); a != nil {
			return &town.Art{Frames: a.Scene}
		}
	case "shop":
		if a := t.art.shopScreen(); a != nil {
			return &town.Art{Frames: a.Scene}
		}
	case "school":
		if a := t.in.TownSchoolArt.Value(); a != nil {
			return &town.Art{Frames: a.Scene}
		}
	}
	return nil
}

func (h roomPageHost) Now() time.Time { return h.t.townAnimationNow() }

// roomDraws binds each scene draw source to the runtime's draw seam.
var roomDraws = map[string]func(townDraws) func(int) int{
	"tender":   townDraws.tavernDraw,
	"idle":     townDraws.shopDraw,
	"training": townDraws.schoolDraw,
}

// roomStreams names the session stream each scene draw source draws on when
// its seam is unset.
var roomStreams = map[string]random.Name{
	"tender":   random.Tavern,
	"idle":     random.ShopInterior,
	"training": random.School,
}

func (h roomPageHost) Draw(source string, n int) int {
	t := h.t
	var draw func(int) int
	if bind := roomDraws[source]; bind != nil {
		draw = bind(t.draws)
	}
	if draw != nil {
		return boundedPresentationRoll(draw, nil, n)
	}
	name, ok := roomStreams[source]
	if !ok {
		name = random.Name(source)
	}
	return boundedPresentationRoll(nil, t.draws.stream(name), n)
}

// Reseed keeps the source's session stream running: the original reseeds
// nothing on a room's entry (TOWN-505), and a stream derived from the
// session seed replays without a restart.
func (h roomPageHost) Reseed(string) {}

func (h roomPageHost) PlaySound(source, key string, loop bool) town.Voice {
	t := h.t
	if t.sess == nil {
		return nil
	}
	sample, ok := t.in.SoundBank.namedSample(key)
	if !ok || t.sound.soundDevice() == nil {
		return nil
	}
	v := audio.Dispatch(t.roomSoundPlayer(audio.EffectsChannel), sample,
		audio.FixedRequest(source, key, audio.EffectsChannel, 128, loop,
			audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}))
	if v == nil {
		return nil
	}
	return v
}

func (h roomPageHost) StopSound(v town.Voice) {
	if av, ok := v.(audio.Voice); ok {
		audio.StopReset(av)
	}
}

// roomEvents are the events the ROM1 campaign raises on its room pages: a
// shop rack picked by the player, the merchant's reactions to a trade, a
// school lesson and a change of the shown member's class.
var roomEvents = []string{"rack", "merchant-yes", "merchant-no", "train", "class-change"}

// roomValues answers the values the description names.
var roomValues = map[string]func(t *townScreen) int{
	"shop-chosen":  func(t *townScreen) int { return t.shopChosen },
	"member-class": (*townScreen).schoolMemberClass,
}

func (h roomPageHost) Value(name string) int {
	if v := roomValues[name]; v != nil {
		return v(h.t)
	}
	return -1
}

// roomScene is a page's composed centre as ui paints it.
type roomScene struct{ p *town.Page }

func (s roomScene) Paint(dst *image.RGBA, group string) { s.p.Paint(dst, group) }

func (t *townScreen) inTavernInterior() bool {
	return t != nil && (t.room == roomTavern || t.room == roomTalk && t.dialogueBuilding == TownTavern)
}

// leaveTavernInterior pauses the tavern page and releases the tavern's room
// audio once the screen is elsewhere.
func (t *townScreen) leaveTavernInterior() {
	if t == nil {
		return
	}
	t.tavernPage().SetActive(false)
	if !t.inTavernInterior() && t.audioRoom == roomTavern {
		t.destroyRoomAudio()
	}
}

// TavernInteriorActive is an explicit Againrom pause policy for lifecycle
// arms the accepted research leaves open. A resume rebases all page clocks,
// so time spent unfocused, in a menu or outside the room cannot catch up.
func (t *townScreen) TavernInteriorActive(active bool) {
	if t == nil || t.sess == nil {
		return
	}
	t.tavernPage().SetActive(active && t.inTavernInterior())
}

func (t *townScreen) inShopInterior() bool { return t != nil && t.AtTownShop() }

// ShopInteriorActive freezes the shop page's clocks while the shop is not the
// focused live room. The first activation keeps the entry stamps; later
// resumes rebase both clocks and cannot catch up time spent behind a menu,
// cutscene or focus loss.
func (t *townScreen) ShopInteriorActive(active bool) {
	if t == nil || t.sess == nil {
		return
	}
	t.shopPage().SetActive(active && t.inShopInterior())
}

// AdvanceShopInteriorAnimation is one paint of the shop page.
func (t *townScreen) AdvanceShopInteriorAnimation() {
	if t == nil || !t.inShopInterior() {
		return
	}
	t.shopPage().Advance()
}
