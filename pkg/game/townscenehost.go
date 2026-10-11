package game

import (
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/random"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// sceneCampaign is what a game's town screen answers the one scene adapter
// for a room of its description: the room's art, the presentation clock and
// draws, the room's sounds, and the campaign's conditions, values and hooks.
type sceneCampaign interface {
	sceneArt(room string) *town.Art
	sceneNow() time.Time
	sceneDraw(source string, n int) int
	sceneSeed() int64
	sceneCondition(name string) bool
	sceneValue(name string) int
	scenePlay(room, source, key string, loop bool) town.Voice
	sceneStartLoop(key string) bool
	sceneStopLoop()
	sceneLeave(room string)
	sceneHook(name, room string)
	sceneDialogue(key string)
}

// townSceneHost is the one adapter the square and each room page of every
// game's town run over, bound to its screen and its room by name.
type townSceneHost struct {
	t    sceneCampaign
	room string
}

func newSquareScene(t *townScreen, d *town.Description) *town.Scene {
	return town.NewScene(d, d.Square.Name, townSceneHost{t: t, room: d.Square.Name}, t.townProcess)
}

func (h townSceneHost) Art() *town.Art                { return h.t.sceneArt(h.room) }
func (h townSceneHost) Now() time.Time                { return h.t.sceneNow() }
func (h townSceneHost) Draw(source string, n int) int { return h.t.sceneDraw(source, n) }
func (h townSceneHost) Seed() int64                   { return h.t.sceneSeed() }
func (h townSceneHost) Condition(name string) bool    { return h.t.sceneCondition(name) }
func (h townSceneHost) Value(name string) int         { return h.t.sceneValue(name) }
func (h townSceneHost) StartLoop(key string) bool     { return h.t.sceneStartLoop(key) }
func (h townSceneHost) StopLoop(string)               { h.t.sceneStopLoop() }
func (h townSceneHost) Leave(room string)             { h.t.sceneLeave(room) }
func (h townSceneHost) Hook(name, room string)        { h.t.sceneHook(name, room) }
func (h townSceneHost) Dialogue(key string)           { h.t.sceneDialogue(key) }
func (h townSceneHost) PlaySound(source, key string, loop bool) town.Voice {
	return h.t.scenePlay(h.room, source, key, loop)
}

func (h townSceneHost) StopSound(v town.Voice) {
	if av, ok := v.(audio.Voice); ok {
		audio.StopReset(av)
	}
}

func (t *townScreen) sceneArt(name string) *town.Art {
	if t == nil || t.sess == nil {
		return nil
	}
	room, ok := townRoomNames[name]
	if !ok {
		return nil
	}
	switch room {
	case roomSquare:
		return t.in.TownSquareArt.Value()
	case roomTavern:
		if a := t.in.TownTavernArt.Value(); a != nil {
			return &town.Art{Frames: a.Scene}
		}
	case roomShop:
		if a := t.art.shopScreen(); a != nil {
			return &town.Art{Frames: a.Scene}
		}
	case roomSchool:
		if a := t.in.TownSchoolArt.Value(); a != nil {
			return &town.Art{Frames: a.Scene}
		}
	}
	return nil
}

func (t *townScreen) sceneNow() time.Time { return t.townAnimationNow() }

// sceneDraw is a draw source's seam and its stream while the seam is unset.
type sceneDraw struct {
	seam   func(townDraws) func(int) int
	stream random.Name
}

var sceneDraws = map[string]sceneDraw{
	"animation": {townDraws.animationDraw, random.TownAnimation},
	"ambient":   {townDraws.ambientDraw, random.TownAmbient},
	"tender":    {townDraws.tavernDraw, random.Tavern},
	"idle":      {townDraws.shopDraw, random.ShopInterior},
	"training":  {townDraws.schoolDraw, random.School},
}

func (t *townScreen) sceneDraw(source string, n int) int { return sceneDrawFrom(t.draws, source, n) }

// sceneDrawFrom is one bounded draw of a scene's host source: its seam when
// set, else its stream, else the session stream of that name.
func sceneDrawFrom(draws townDraws, source string, n int) int {
	bind, ok := sceneDraws[source]
	if !ok {
		return boundedPresentationRoll(nil, draws.stream(random.Name(source)), n)
	}
	if draw := bind.seam(draws); draw != nil {
		return boundedPresentationRoll(draw, nil, n)
	}
	return boundedPresentationRoll(nil, draws.stream(bind.stream), n)
}

func (t *townScreen) sceneSeed() int64 { return t.sound.wildlifeSeed() }

func (t *townScreen) sceneCondition(name string) bool {
	if c := townConditions[name]; c != nil {
		return c(t)
	}
	return false
}

func (t *townScreen) sceneValue(name string) int {
	if v := roomValues[name]; v != nil {
		return v(t)
	}
	return -1
}

// PlaySound plays through the room's audio scope; without a sound device a
// room page asks nothing and the square still opens its scope.
func (t *townScreen) scenePlay(room, source, key string, loop bool) town.Voice {
	if t.sess == nil {
		return nil
	}
	sample, ok := t.in.SoundBank.namedSample(key)
	if !ok {
		return nil
	}
	if townRoomNames[room] != roomSquare && t.sound.soundDevice() == nil {
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

func (t *townScreen) sceneStartLoop(key string) bool {
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

func (t *townScreen) sceneStopLoop() {
	if t.sound.ambientDevice() != nil {
		if ui.DeliveryOwner(t.sound.soundDevice()) != nil {
			audio.StopReset(t.squareLoop)
		} else {
			t.sound.ambientDevice().StopLoop(ui.AmbientTownCrowd)
		}
	}
	t.squareLoop = nil
}

func (t *townScreen) sceneLeave(room string) {
	if r, ok := townRoomNames[room]; ok && t.audioRoom == r {
		t.destroyRoomAudio()
	}
}

func (t *townScreen) sceneHook(name, room string) {
	if hook := townHooks[name]; hook != nil {
		r, ok := townRoomNames[room]
		if !ok {
			r = t.room
		}
		hook(t, r)
	}
}

// sceneDialogue opens nothing: no first-game description opens a dialogue in
// place.
func (t *townScreen) sceneDialogue(string) {}
