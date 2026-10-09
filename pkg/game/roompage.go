package game

import (
	"errors"
	"image"
	"math/rand"
	"time"

	"againrom/pkg/audio"
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
	}
	return nil
}

func (h roomPageHost) Now() time.Time { return h.t.townAnimationNow() }

// roomDraws binds each scene draw source to the runtime's draw service.
var roomDraws = map[string]func(townDraws) func(int) int{
	"tender": townDraws.tavernDraw,
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
	if t.pageRandom[source] == nil {
		h.Reseed(source)
	}
	return boundedPresentationRoll(nil, t.pageRandom[source], n)
}

func (h roomPageHost) Reseed(source string) {
	t := h.t
	if t.pageRandom == nil {
		t.pageRandom = map[string]*rand.Rand{}
	}
	t.pageRandom[source] = rand.New(rand.NewSource(t.townAnimationNow().UnixNano()))
}

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

// roomValues answers the values the description names.
var roomValues = map[string]func(t *townScreen) int{}

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
