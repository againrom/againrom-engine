package game

import (
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/random"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// townSceneHost is the one adapter the square and each room page of every
// game's town run over, bound to its screen and its room by name.
type townSceneHost struct {
	t    *townScreen
	room string
}

func newSquareScene(t *townScreen, d *town.Description) *town.Scene {
	return t.scene(d, d.Square.Name)
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
func (h townSceneHost) PlaySound(r town.SoundRequest) town.Voice {
	return h.t.scenePlay(h.room, r)
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
	if t.hostRules().Art == "manifest" {
		if d := t.townDescription(); d != nil {
			return t.roomArt(d, name).scene
		}
		return nil
	}
	room, ok := townRoomNames[name]
	if !ok {
		return nil
	}
	switch room {
	case roomSquare:
		return t.squareArt()
	case roomTavern:
		if a := t.tavernArt(); a != nil {
			return &town.Art{Frames: a.Scene}
		}
	case roomShop:
		if a := t.shopArt(); a != nil {
			return &town.Art{Frames: a.Scene}
		}
	case roomSchool:
		if a := t.schoolArt(); a != nil {
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

// sceneRawFrom is one raw value 0..RandMax of a scene's host source: its
// seam's draw of RandMax+1 when set, else its stream's raw value, which under
// the original generator is rand() itself.
func sceneRawFrom(draws townDraws, source string) int {
	name := random.Name(source)
	if bind, ok := sceneDraws[source]; ok {
		if draw := bind.seam(draws); draw != nil {
			return boundedPresentationRoll(draw, nil, random.RandMax+1)
		}
		name = bind.stream
	}
	if st := draws.stream(name); st != nil {
		return st.Raw()
	}
	return 0
}

// bindRawDraws gives every host source of a built scene its raw values, so
// each raw pick of the scene reads them.
func (t *townScreen) bindRawDraws(s *town.Scene) {
	for _, source := range s.HostSources() {
		s.SetRawDraw(source, func() int { return sceneRawFrom(t.draws, source) })
	}
}

func (t *townScreen) sceneSeed() int64 { return t.sound.wildlifeSeed() }

func (t *townScreen) sceneCondition(name string) bool {
	if c := townConditions[name]; c != nil {
		return c(t)
	}
	return t.campaignTown().condition(name)
}

func (t *townScreen) sceneValue(name string) int {
	if v := roomValues[name]; v != nil {
		return v(t)
	}
	return t.campaignTown().value(name)
}

// PlaySound plays through the room's audio scope at the slot's priority and
// volume class; without a sound device a room page asks nothing and the
// square still opens its scope.
func (t *townScreen) scenePlay(room string, r town.SoundRequest) town.Voice {
	if t.sess == nil {
		return nil
	}
	sample, ok := t.in.SoundBank.namedSample(r.Key)
	if !ok {
		return nil
	}
	if (room != "square" || !t.hostRules().SilentSquare) && t.sound.soundDevice() == nil {
		return nil
	}
	group, priority := audio.EffectsChannel, townSoundPriority(r)
	if r.Speech && t.hostRules().Speech {
		group = audio.SpeechChannel
	}
	v := audio.Dispatch(t.roomSoundPlayer(group, room), sample,
		audio.FixedRequest(r.Source, r.Key, group, priority, r.Loop,
			audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}))
	if v == nil {
		return nil
	}
	return v
}

// townSoundPriority is a slot request's priority: the slot's own, else 128,
// the priority of every town room request but click01's (TOWN-554,
// VIDEO-SFX-099).
func townSoundPriority(r town.SoundRequest) uint8 {
	if r.Priority > 0 && r.Priority < 256 {
		return uint8(r.Priority)
	}
	return 128
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
		t.squareLoop = audio.Dispatch(t.roomSoundPlayer(audio.EffectsChannel, t.roomName()), sample, request)
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
	if t.audioRoom == room {
		t.destroyRoomAudio()
	}
}

func (t *townScreen) sceneHook(name, room string) {
	if hook := townHooks[name]; hook.run != nil {
		if room == "" && t.hostRules().CurrentHookRoom {
			room = t.roomName()
		}
		hook.run(t, room)
	}
}

func (t *townScreen) sceneDialogue(key string) {
	if t.hostRules().Talk == "section" {
		t.squareAction = t.openTownText(key)
	}
}
