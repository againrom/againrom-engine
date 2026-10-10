package game

import (
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/random"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// townSceneHost is the one adapter the square and each room page run over,
// bound to its room by name.
type townSceneHost struct {
	t    *townScreen
	room string
}

func newSquareScene(t *townScreen, d *town.Description) *town.Scene {
	return town.NewScene(d, d.Square.Name, townSceneHost{t: t, room: d.Square.Name}, t.townProcess)
}

func (h townSceneHost) Art() *town.Art {
	t := h.t
	if t == nil || t.sess == nil {
		return nil
	}
	room, ok := townRoomNames[h.room]
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

func (h townSceneHost) Now() time.Time { return h.t.townAnimationNow() }

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

func (h townSceneHost) Draw(source string, n int) int {
	t := h.t
	bind, ok := sceneDraws[source]
	if !ok {
		return boundedPresentationRoll(nil, t.draws.stream(random.Name(source)), n)
	}
	if draw := bind.seam(t.draws); draw != nil {
		return boundedPresentationRoll(draw, nil, n)
	}
	return boundedPresentationRoll(nil, t.draws.stream(bind.stream), n)
}

func (h townSceneHost) Seed() int64 { return h.t.sound.wildlifeSeed() }

func (h townSceneHost) Condition(name string) bool {
	if c := townConditions[name]; c != nil {
		return c(h.t)
	}
	return false
}

func (h townSceneHost) Value(name string) int {
	if v := roomValues[name]; v != nil {
		return v(h.t)
	}
	return -1
}

// PlaySound plays through the room's audio scope; without a sound device a
// room page asks nothing and the square still opens its scope.
func (h townSceneHost) PlaySound(source, key string, loop bool) town.Voice {
	t := h.t
	if t.sess == nil {
		return nil
	}
	sample, ok := t.in.SoundBank.namedSample(key)
	if !ok {
		return nil
	}
	if townRoomNames[h.room] != roomSquare && t.sound.soundDevice() == nil {
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

func (h townSceneHost) StopSound(v town.Voice) {
	if av, ok := v.(audio.Voice); ok {
		audio.StopReset(av)
	}
}

func (h townSceneHost) StartLoop(key string) bool {
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

func (h townSceneHost) StopLoop(key string) {
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

func (h townSceneHost) Leave(room string) {
	if r, ok := townRoomNames[room]; ok && h.t.audioRoom == r {
		h.t.destroyRoomAudio()
	}
}

func (h townSceneHost) Hook(name, room string) {
	if hook := townHooks[name]; hook != nil {
		r, ok := townRoomNames[room]
		if !ok {
			r = h.t.room
		}
		hook(h.t, r)
	}
}
