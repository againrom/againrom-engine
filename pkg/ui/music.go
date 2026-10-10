package ui

import (
	"math"
	"slices"

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
	MusicCredits

	// musicUnlisted is a screen no scene owns. The description's Unlisted
	// rule makes it silent or leaves the held list alone.
	musicUnlisted MusicScene = 0xff
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

// MusicPauser is a device that can hold its stream at its position. Pause
// answers false when nothing is held; Resume answers false when no paused
// stream remains, and the controller then starts the entry again.
type MusicPauser interface {
	Pause() bool
	Resume() bool
}

// MusicArea is one music-area record of a map: centre and radius in tiles and
// four themes, each an index into the mission list or -1 for none. A record
// at (0, 0) is the map's default.
type MusicArea struct {
	X, Y, Radius int32
	Themes       [4]int32
}

// MusicHero answers the hero's position in 1/256 tile, the mission tick and
// whether the mission has a hero.
type MusicHero func() (x, y int32, tick uint64, ok bool)

type musicRequest struct {
	scene     MusicScene
	mageFirst bool
	// track replaces the scene's static list with this one track when set.
	track string
}

// musicAreaSource is the mission's areas and hero, as the map screen holds them.
type musicAreaSource struct {
	areas []MusicArea
	hero  MusicHero
}

// MusicController owns the current ordinary list and no archive. A source is
// consulted only for the track about to start, and no decoded track is retained
// here after Start returns. Every list, keep rule, order rule and area rule is
// the description's.
type MusicController struct {
	desc   *MusicDescription
	source MusicSource
	device MusicDevice
	draws  *random.Stream

	// last is the request the shown screen made at the last sync.
	last    musicRequest
	lastSet bool
	// request owns the list the player holds.
	request  musicRequest
	set      bool
	order    []string
	position int
	active   bool
	// paused: the device holds the entry at its position. resumeStarts: the
	// entry was stopped without a hold and a resume starts it again.
	paused       bool
	resumeStarts bool
	pick         int
	areas        musicAreaSource
	areaTick     uint64
	preferences  MusicPreferences
	log          []string
}

func NewMusicController(desc *MusicDescription, source MusicSource, device MusicDevice, draws *random.Stream) *MusicController {
	return &MusicController{desc: desc, source: source, device: device, draws: draws, pick: -1, preferences: DefaultMusicPreferences()}
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

// SetMusic installs the game's description, the optional source and the
// retained device, then starts the screen already showing. Missing halves
// remain silent.
func (a *App) SetMusic(desc *MusicDescription, source MusicSource, device MusicDevice, draws *random.Stream) {
	if a == nil {
		return
	}
	if a.music != nil {
		a.music.Stop()
	}
	a.music = NewMusicController(desc, source, device, draws)
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
	a.music.observeAreas()
	a.music.Update()
}

func (a *App) syncMusic() {
	if a == nil || a.music == nil {
		return
	}
	a.music.areas = a.missionMusicAreas()
	next, replace := a.musicRequest()
	if next.scene == musicUnlisted {
		if a.music.desc != nil && a.music.desc.Unlisted == MusicUnlistedKeep {
			replace = false
		}
		next.scene = MusicSilent
	}
	if !replace {
		return
	}
	// A completed load always makes a fresh request; the scene's keep rule
	// then decides between keeping the held list and setting its own.
	fresh := false
	if a.flow != nil && a.flow.loadUI.completed != a.musicLoads {
		a.musicLoads = a.flow.loadUI.completed
		fresh = true
	}
	a.music.requestFrom(next, fresh)
}

// missionMusicAreas is the shown map's areas and hero; other screens have none.
func (a *App) missionMusicAreas() musicAreaSource {
	if a == nil || a.flow == nil || a.flow.screen != ScreenMap || a.flow.viewer == nil {
		return musicAreaSource{}
	}
	return a.flow.viewer.musicAreas
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
		return MusicCredits, false, true
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
		return musicUnlisted, false, true
	}
}

// SetScene makes the request a screen showing scene makes. Repeating the
// same request, including the same school class order, is an exact no-op.
func (m *MusicController) SetScene(scene MusicScene, mageFirst bool) {
	if m == nil {
		return
	}
	m.requestFrom(musicRequest{scene: scene, mageFirst: mageFirst}, false)
}

// requestFrom applies the shown screen's request. An unchanged request does
// nothing unless fresh. A silent request clears or pauses as the description
// says; a scene that keeps, and silence, hold an equal list, resuming it when
// paused; any other request sets its list.
func (m *MusicController) requestFrom(next musicRequest, fresh bool) {
	if !fresh && m.lastSet && m.last == next {
		return
	}
	m.last, m.lastSet = next, true
	tracks := m.requestTracks(next)
	if len(tracks) == 0 && m.desc != nil && m.desc.Silence == MusicSilencePause {
		m.pause()
		return
	}
	if m.set && (len(tracks) == 0 || m.desc.sceneSpec(next.scene).Keep) && slices.Equal(tracks, m.requestTracks(m.request)) {
		if fresh {
			m.record("skip:" + m.sceneKey(next))
		}
		m.request = next
		m.resume()
		return
	}
	m.replaceRequest(next)
}

// RequestScene replaces the ordinary candidate list whether or not the scene
// changed, as a fresh request does: the current stream stops, the list is
// reordered and its first entry starts. A one-file list restarts its file.
func (m *MusicController) RequestScene(scene MusicScene, mageFirst bool) {
	if m == nil {
		return
	}
	next := musicRequest{scene: scene, mageFirst: mageFirst}
	m.last, m.lastSet = next, true
	m.replaceRequest(next)
}

func (m *MusicController) replaceRequest(next musicRequest) {
	if m.set && m.device != nil {
		m.device.Stop()
		m.record("stop")
	}
	m.record("request:" + m.sceneKey(next))
	m.request, m.set, m.active, m.paused, m.resumeStarts, m.pick = next, true, false, false, false, -1
	tracks := m.requestTracks(next)
	switch {
	case m.desc != nil && m.desc.List == MusicListOrdered:
		m.order, m.position = tracks, 0
		if m.draws != nil && len(tracks) != 0 {
			m.position = m.draws.Raw() % len(tracks)
		}
	case m.draws != nil && m.draws.Shared():
		m.order, m.position = m.originalOrder(tracks)
	default:
		m.order = m.shuffle(tracks)
		m.position = 0
	}
	if next.scene == MusicMission {
		m.enterAreas()
	}
	if m.preferences.Enabled {
		m.startCurrent()
	}
}

// pause stops the player and keeps its list and position. A device that
// cannot hold its stream is stopped, and the next resume starts the held
// entry again.
func (m *MusicController) pause() {
	if !m.active {
		return
	}
	held := false
	if p, ok := m.device.(MusicPauser); ok {
		held = p.Pause()
	} else if m.device != nil {
		m.device.Stop()
	}
	m.record("pause")
	m.active, m.paused, m.resumeStarts = false, held, !held
}

// resume continues a paused entry, or starts the held entry again when the
// device kept no paused stream.
func (m *MusicController) resume() {
	if m.active || !m.preferences.Enabled {
		return
	}
	if m.paused {
		m.paused = false
		if p, ok := m.device.(MusicPauser); ok && p.Resume() {
			m.active = true
			m.record("resume:" + m.Playing())
			return
		}
	}
	if !m.resumeStarts {
		return
	}
	m.resumeStarts = false
	m.startCurrent()
	m.record("resume:" + m.Playing())
}

// Update advances the ordinary list after EOF: to the area pick when one is
// set, else to the next entry. A one-entry list therefore reloads the same
// file; a shuffled list repeats only after every entry.
func (m *MusicController) Update() {
	if m == nil || !m.active || m.device == nil || !m.device.Ended() {
		return
	}
	m.active = false
	if len(m.order) != 0 {
		if m.pick >= 0 && m.pick < len(m.order) {
			m.position = m.pick
		} else {
			m.position = (m.position + 1) % len(m.order)
		}
		m.startCurrent()
	}
}

// enterAreas is the mission list's own area select: with a hero the area
// holding it picks a theme, and a pick replaces the drawn start entry.
func (m *MusicController) enterAreas() {
	if m.desc == nil || m.desc.Areas == nil || m.areas.hero == nil {
		return
	}
	x, y, tick, ok := m.areas.hero()
	m.areaTick = tick
	if !ok {
		return
	}
	m.selectArea(x, y)
	if m.pick >= 0 && m.pick < len(m.order) {
		m.position = m.pick
	}
}

// observeAreas runs the area select once when the mission tick has passed a
// multiple of the description's period since the last observation.
func (m *MusicController) observeAreas() {
	if m == nil || m.desc == nil || m.desc.Areas == nil || m.areas.hero == nil || !m.set || m.request.scene != MusicMission {
		return
	}
	x, y, tick, ok := m.areas.hero()
	period := uint64(m.desc.Areas.Period)
	crossed := tick > m.areaTick && tick/period != m.areaTick/period
	m.areaTick = tick
	if crossed && ok {
		m.selectArea(x, y)
	}
}

// selectArea walks the records in order. The default record is taken while
// no closer record has been; another record with no theme is skipped; any
// other is taken when the position lies inside its radius and nearer than
// the last taken. Each taken record draws its pick from its themes.
func (m *MusicController) selectArea(x, y int32) {
	best := 1e20
	for _, area := range m.areas.areas {
		if area.X == 0 && area.Y == 0 {
			if best > 1e15 {
				best = 1e10
				m.drawTheme(area)
			}
			continue
		}
		if area.Themes == [4]int32{-1, -1, -1, -1} {
			continue
		}
		dx, dy := float64(x)-float64(area.X)*256, float64(y)-float64(area.Y)*256
		d := math.Sqrt(dx*dx + dy*dy)
		if d < float64(area.Radius)*256 && d < best {
			best = d
			m.drawTheme(area)
		}
	}
}

// drawTheme draws one of the record's four themes, drawing again while it
// names none.
func (m *MusicController) drawTheme(area MusicArea) {
	if m.draws == nil || area.Themes == [4]int32{-1, -1, -1, -1} {
		return
	}
	for {
		theme := area.Themes[(m.draws.Raw()*4)>>15]
		if theme >= 0 {
			m.pick = int(theme)
			return
		}
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
func (m *MusicController) requestTracks(r musicRequest) []string {
	if r.track != "" {
		return []string{r.track}
	}
	return m.desc.sceneTracks(r.scene, r.mageFirst)
}

func (m *MusicController) sceneKey(r musicRequest) string {
	tracks := m.requestTracks(r)
	if len(tracks) == 0 {
		return "silent"
	}
	if len(tracks) == 1 {
		return tracks[0]
	}
	return tracks[0] + ".." + tracks[len(tracks)-1]
}

// RequestLog is the bounded witness log: stop, request:<list>, skip:<list>,
// pause and resume:<track>.
func (m *MusicController) RequestLog() []string {
	if m == nil {
		return nil
	}
	return append([]string(nil), m.log...)
}

// AreaPick is the mission list index the next track end opens, or -1.
func (m *MusicController) AreaPick() int {
	if m == nil {
		return -1
	}
	return m.pick
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
	m.order, m.active, m.set, m.paused, m.resumeStarts, m.lastSet, m.pick = nil, false, false, false, false, false, -1
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
