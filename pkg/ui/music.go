package ui

import (
	"againrom/pkg/audio"
	"againrom/pkg/random"
)

// MusicScene is one static candidate-list owner. Overlay screens deliberately
// have no scene: App leaves the underlying list running while they are open.
type MusicScene uint8

const (
	MusicSilent MusicScene = iota
	MusicMenu
	MusicChargen
	MusicCampaign
	MusicMission
	MusicTown
	MusicShop
	MusicTavern
	MusicSchool
)

// MusicSource reads and decodes one named track. Implementations may refuse any
// name; refusal is silence.
type MusicSource interface {
	Track(name string) (audio.Track, bool)
}

// MusicDevice owns the one retained playback stream. Start replaces its prior
// stream. Ended becomes true only after a started stream reaches EOF.
type MusicDevice interface {
	Start(audio.Track)
	Stop()
	Ended() bool
	SetSettings(audio.Settings)
}

type musicRequest struct {
	scene     MusicScene
	mageFirst bool
	// track replaces the scene's static list with this one track when set.
	track string
}

var missionMusicTracks = [...]string{
	"B00.wav", "B01.wav", "B02.wav", "B03.wav", "B04.wav", "B05.wav",
	"B06.wav", "B07.wav", "B08.wav", "B09.wav", "B10.wav", "B11.wav",
}

// MusicController owns the current ordinary list and no archive. A source is
// consulted only for the track about to start, and no decoded track is retained
// here after Start returns.
type MusicController struct {
	source MusicSource
	device MusicDevice
	draws  *random.Stream

	request     musicRequest
	set         bool
	order       []string
	position    int
	active      bool
	preferences MusicPreferences
	log         []string
}

func NewMusicController(source MusicSource, device MusicDevice, draws *random.Stream) *MusicController {
	return &MusicController{source: source, device: device, draws: draws, preferences: DefaultMusicPreferences()}
}

// TownMusicScreen exposes only the town room's static music identity and the
// selected school member's class order. It carries no game model type.
type TownMusicScreen interface {
	TownMusic() (scene MusicScene, mageFirst bool)
}

// TownMusicTrack optionally names the one track a town room plays in place
// of its scene's static list; false keeps the static list.
type TownMusicTrack interface {
	TownMusicTrack() (string, bool)
}

// SetMusic installs the optional source and retained device, then starts the
// screen already showing. Missing halves remain silent.
func (a *App) SetMusic(source MusicSource, device MusicDevice, draws *random.Stream) {
	if a == nil {
		return
	}
	if a.music != nil {
		a.music.Stop()
	}
	a.music = NewMusicController(source, device, draws)
	if a.flow != nil && a.flow.soundOptions.ReadPlayback != nil {
		a.music.SetPreferences(a.flow.soundOptions.ReadPlayback())
	}
	a.syncMusic()
}

// StopMusic is the explicit application teardown seam.
func (a *App) StopMusic() {
	if a != nil && a.music != nil {
		a.music.Stop()
	}
}

func (a *App) updateMusic() {
	if a == nil || a.music == nil {
		return
	}
	a.syncMusic()
	a.music.Update()
}

func (a *App) syncMusic() {
	if a == nil || a.music == nil {
		return
	}
	next, replace := a.musicRequest()
	if !replace {
		return
	}
	scene := next.scene
	// A completed mission load always re-requests; any other surface keeps
	// an equal list running.
	if a.flow != nil && a.flow.loadUI.completed != a.musicLoads {
		a.musicLoads = a.flow.loadUI.completed
		if scene == MusicMission {
			a.music.replaceRequest(next)
			return
		}
		if a.music.set && a.music.request == next {
			a.music.record("skip:" + sceneKey(a.music.request))
		}
	}
	a.music.setRequest(next)
}

// musicRequest is the request the shown screen makes: its scene, the
// school's class order and a town room's own track.
func (a *App) musicRequest() (musicRequest, bool) {
	scene, mageFirst, replace := a.musicScene()
	next := musicRequest{scene: scene, mageFirst: mageFirst}
	if replace && a.flow != nil && a.flow.screen == ScreenTown && a.cutscene == nil {
		if town, ok := a.flow.town.(TownMusicTrack); ok {
			if track, ok := town.TownMusicTrack(); ok {
				next.track = track
			}
		}
	}
	return next, replace
}

func (a *App) musicScene() (MusicScene, bool, bool) {
	if a != nil && a.cutscene != nil {
		return MusicSilent, false, true
	}
	if a == nil || a.flow == nil {
		return MusicSilent, false, true
	}
	switch a.flow.screen {
	case ScreenMenu, ScreenCutsceneLibrary:
		return MusicMenu, false, true
	case ScreenCredits:
		return MusicMenu, false, true
	case ScreenPicker:
		return MusicCampaign, false, true
	case ScreenMap:
		return MusicMission, false, true
	case ScreenChargen:
		return MusicChargen, false, true
	case ScreenTown:
		if town, ok := a.flow.town.(TownMusicScreen); ok {
			scene, mageFirst := town.TownMusic()
			return scene, mageFirst, true
		}
		return MusicTown, false, true
	case ScreenGameMenu, ScreenLoad, ScreenDocuments, ScreenSave, ScreenMod:
		// Full-screen overlays retain the list beneath them. Closing one finds
		// the controller already on the restored scene and is idempotent.
		return MusicSilent, false, false
	default:
		return MusicSilent, false, true
	}
}

// SetScene replaces the ordinary candidate list once. Repeating the same
// scene, including the same school class order, is an exact no-op.
func (m *MusicController) SetScene(scene MusicScene, mageFirst bool) {
	if m == nil {
		return
	}
	m.setRequest(musicRequest{scene: scene, mageFirst: mageFirst})
}

func (m *MusicController) setRequest(next musicRequest) {
	if m.set && m.request == next {
		return
	}
	m.replaceRequest(next)
}

// RequestScene replaces the ordinary candidate list whether or not the scene
// changed, as a fresh request does: the current stream stops, the list is
// reshuffled and its first entry starts. A one-file list restarts its file.
func (m *MusicController) RequestScene(scene MusicScene, mageFirst bool) {
	if m == nil {
		return
	}
	m.replaceRequest(musicRequest{scene: scene, mageFirst: mageFirst})
}

func (m *MusicController) replaceRequest(next musicRequest) {
	if m.set && m.device != nil {
		m.device.Stop()
		m.record("stop")
	}
	m.record("request:" + sceneKey(next))
	m.request, m.set, m.active = next, true, false
	if m.draws != nil && m.draws.Shared() {
		m.order, m.position = m.originalOrder(requestTracks(next))
	} else {
		m.order = m.shuffle(requestTracks(next))
		m.position = 0
	}
	if m.preferences.Enabled {
		m.startCurrent()
	}
}

// Update advances the ordinary list after EOF. A one-entry list therefore
// reloads the same file; longer lists repeat only after every shuffled entry.
func (m *MusicController) Update() {
	if m == nil || !m.active || m.device == nil || !m.device.Ended() {
		return
	}
	m.active = false
	if len(m.order) != 0 {
		m.position = (m.position + 1) % len(m.order)
		m.startCurrent()
	}
}

const maxMusicLog = 64

func (m *MusicController) record(event string) {
	if len(m.log) == maxMusicLog {
		m.log = append(m.log[:0], m.log[1:]...)
	}
	m.log = append(m.log, event)
}

// requestTracks is a request's list: its own track, else its scene's.
func requestTracks(r musicRequest) []string {
	if r.track != "" {
		return []string{r.track}
	}
	return staticMusicTracks(r.scene, r.mageFirst)
}

func sceneKey(r musicRequest) string {
	tracks := requestTracks(r)
	if len(tracks) == 0 {
		return "silent"
	}
	if len(tracks) == 1 {
		return tracks[0]
	}
	return tracks[0] + ".." + tracks[len(tracks)-1]
}

// RequestLog is the bounded witness log: stop, request:<list>, skip:<list>.
func (m *MusicController) RequestLog() []string {
	if m == nil {
		return nil
	}
	return append([]string(nil), m.log...)
}

// Stop tears down the current stream and clears the list.
func (m *MusicController) Stop() {
	if m == nil {
		return
	}
	if m.device != nil {
		m.device.Stop()
		m.record("stop")
	}
	m.order, m.active, m.set = nil, false, false
}

func (m *MusicController) shuffle(tracks []string) []string {
	out := append([]string(nil), tracks...)
	if !m.preferences.RandomOrder || m.draws == nil {
		return out
	}
	for range out {
		x, y := m.draws.Raw()%len(out), m.draws.Raw()%len(out)
		out[x], out[y] = out[y], out[x]
	}
	return out
}

// originalOrder is the original's list replace (MAGIC-286): one start draw
// rand()%n, then the order build, which swaps n times only in Random Order,
// and the position of the start track in that order.
func (m *MusicController) originalOrder(tracks []string) ([]string, int) {
	if len(tracks) == 0 {
		return nil, 0
	}
	start := m.draws.Raw() % len(tracks)
	index := make([]int, len(tracks))
	for i := range index {
		index[i] = i
	}
	if m.preferences.RandomOrder {
		for range index {
			x, y := m.draws.Raw()%len(index), m.draws.Raw()%len(index)
			index[x], index[y] = index[y], index[x]
		}
	}
	out, position := make([]string, len(index)), 0
	for i, at := range index {
		out[i] = tracks[at]
		if at == start {
			position = i
		}
	}
	return out, position
}

func (m *MusicController) startCurrent() {
	if m.device == nil || m.source == nil || len(m.order) == 0 {
		return
	}
	for tried := 0; tried < len(m.order); tried++ {
		name := m.order[m.position]
		if m.startName(name) {
			return
		}
		m.position = (m.position + 1) % len(m.order)
	}
}

func (m *MusicController) residentTracks() int {
	if m != nil && m.active {
		return 1
	}
	return 0
}

func staticMusicTracks(scene MusicScene, mageFirst bool) []string {
	switch scene {
	case MusicMenu:
		return []string{"menu.wav"}
	case MusicChargen:
		return []string{"chrgen.wav"}
	case MusicCampaign:
		return []string{"map.wav"}
	case MusicMission:
		return append([]string(nil), missionMusicTracks[:]...)
	case MusicTown:
		return []string{"town.wav"}
	case MusicShop:
		return []string{"shop.wav"}
	case MusicTavern:
		return []string{"inn.wav"}
	case MusicSchool:
		if mageFirst {
			return []string{"schoolm.wav", "schoolw.wav"}
		}
		return []string{"schoolw.wav", "schoolm.wav"}
	default:
		return nil
	}
}
