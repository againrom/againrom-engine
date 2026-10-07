package ui

import (
	"image"
	"math/rand"
	"strconv"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/render/terrain"
)

// AmbientLoop names retained repeating presentation sounds. The first two are
// map sounds recovered by VIDEO-SFX-017; AmbientTownCrowd is TOWN-420's town
// entry request. Separate selectors give them independent cleanup channels.
type AmbientLoop uint8

const (
	AmbientRiver AmbientLoop = iota
	AmbientWallOfFire
	AmbientTownCrowd
	ambientLoopCount
)

const (
	ambientRiverSlot    = 50
	ambientBirdSlot     = 60
	ambientCrowSlot     = 70
	ambientWallFireSlot = 90
)

// AmbientDevice owns the retained loop streams. Moving a loop changes its
// placement without resetting its audible phase; missing devices are silence.
type AmbientDevice interface {
	StartLoop(AmbientLoop, audio.Sample, audio.Placement)
	MoveLoop(AmbientLoop, audio.Placement)
	StopLoop(AmbientLoop)
	Stop()
	SetSettings(audio.Settings)
}

type ambientStaticCell struct {
	cell       image.Point
	fireObject int32
}

type ambientSnapshot struct {
	geometry audio.ViewGeometry
	listener image.Point
	hasView  bool
	river    []image.Point
	wallFire []image.Point
	birds    []image.Point
	crows    []image.Point
}

type ambientLoopState struct {
	active    bool
	placement audio.Placement
	voice     audio.Voice
	request   audio.Request
}

// ambientController is presentation state only. Its RNG and wall-clock
// deadline never enter the world, its byte form, or its digest.
type ambientController struct {
	bank        SoundBank
	ones        audio.Player
	device      AmbientDevice
	rng         *rand.Rand
	next        time.Time
	loop        [ambientLoopCount]ambientLoopState
	origin      image.Point
	originReady bool
}

func newAmbientController(bank SoundBank, ones audio.Player, device AmbientDevice, seed int64) *ambientController {
	return &ambientController{bank: bank, ones: ones, device: device, rng: rand.New(rand.NewSource(seed))}
}

func (c *ambientController) update(now time.Time, s ambientSnapshot) {
	if c == nil {
		return
	}

	// The first visit establishes the original deadline instead of making a
	// startup bird call. A deadline with no eligible objects remains due, so an
	// object entering the camera can satisfy it on the next presentation frame.
	if c.next.IsZero() {
		c.next = now.Add(c.nextDelay())
	}
	moved := !c.originReady || c.origin != s.geometry.Origin
	due := now.After(c.next)
	if !s.hasView {
		c.stopLoop(AmbientRiver, &c.loop[AmbientRiver])
		c.stopLoop(AmbientWallOfFire, &c.loop[AmbientWallOfFire])
		return
	}
	if !moved && !due {
		return
	}
	if moved {
		c.origin, c.originReady = s.geometry.Origin, true
	}
	c.updateLoop(AmbientRiver, ambientRiverSlot, s.geometry, s.hasView, s.river)
	c.updateLoop(AmbientWallOfFire, ambientWallFireSlot, s.geometry, s.hasView, s.wallFire)
	if !due {
		return
	}
	total := len(s.birds) + len(s.crows)
	if total == 0 {
		return
	}

	slot := ambientCrowSlot
	if ambientBirdWins(c.rng, total, len(s.birds)) {
		// The executable divides a 0..32767 draw by 16383. Slot 62 is therefore
		// possible but rare, and its absent archive entry lawfully yields silence.
		slot = ambientBirdSlot + c.rng.Intn(0x8000)/0x3fff
	}
	all := make([]image.Point, 0, total)
	all = append(all, s.birds...)
	all = append(all, s.crows...)
	if c.ones != nil && c.bank != nil && s.hasView {
		if sample, ok := c.bank.Sample(slot); ok {
			if request, ok := audio.AmbientRequest("mission-bird", "registry:"+strconv.Itoa(slot), false, all, s.geometry, true); ok {
				if slot == ambientCrowSlot {
					request.Source, request.Recipe = "mission-crow", "mission-crow"
				}
				audio.Dispatch(c.ones, sample, request)
			}
		}
	}
	c.next = now.Add(c.nextDelay())
}

func (c *ambientController) updateLoop(kind AmbientLoop, slot int, geometry audio.ViewGeometry, hasView bool, cells []image.Point) {
	state := &c.loop[kind]
	if !hasView || len(cells) == 0 || c.device == nil || c.bank == nil {
		c.stopLoop(kind, state)
		return
	}
	family := "mission-river"
	if kind == AmbientWallOfFire {
		family = "mission-fire"
	}
	request, ok := audio.AmbientRequest(family, "registry:"+strconv.Itoa(slot), true, cells, geometry, false)
	if !ok {
		c.stopLoop(kind, state)
		return
	}
	if state.active && (state.voice == nil || state.voice.Playing()) {
		if state.request.Attenuation != request.Attenuation || state.request.Pan != request.Pan {
			if typed, ok := c.device.(interface {
				MoveLoopRequest(AmbientLoop, audio.Request)
			}); ok {
				typed.MoveLoopRequest(kind, request)
			} else {
				c.device.MoveLoop(kind, request.Placement)
			}
			state.placement, state.request = request.Placement, request
		}
		return
	}
	sample, ok := c.bank.Sample(slot)
	if !ok {
		return
	}
	state.voice = RequestAmbient(c.device, kind, sample, request)
	state.active, state.placement, state.request = state.voice != nil, request.Placement, request
}

func (c *ambientController) stopLoop(kind AmbientLoop, state *ambientLoopState) {
	if !state.active {
		return
	}
	if c.device != nil {
		c.device.StopLoop(kind)
	}
	*state = ambientLoopState{}
}

func (c *ambientController) stop() {
	if c == nil {
		return
	}
	if c.device != nil {
		c.device.Stop()
	}
	c.loop = [ambientLoopCount]ambientLoopState{}
}

func (c *ambientController) nextDelay() time.Duration {
	return time.Duration(10000+c.rng.Intn(0x8000)/2) * time.Millisecond
}

// ambientBirdWins mirrors the executable's count-ratio draw. FireObject is a
// two-way class flag here, never a weight or an object-class subscript.
func ambientBirdWins(rng *rand.Rand, total, birds int) bool {
	if rng == nil || total <= 0 {
		return false
	}
	divisor := 0x7fff / total
	if divisor < 1 {
		divisor = 1
	}
	return rng.Intn(0x8000)/divisor < birds
}

func ambientCentroid(cells []image.Point) image.Point {
	if len(cells) == 0 {
		return image.Point{}
	}
	x, y := 0, 0
	for _, cell := range cells {
		x += cell.X
		y += cell.Y
	}
	return image.Pt(x/len(cells), y/len(cells))
}

func ambientStaticCells(g terrain.Grid, set *terrain.StaticSet) []ambientStaticCell {
	if set == nil || g.Width <= 0 || g.Height <= 0 || len(g.Overlay) != g.Width*g.Height {
		return nil
	}
	var out []ambientStaticCell
	for i, code := range g.Overlay {
		if code == 0 || set.Classes[code] == nil {
			continue
		}
		class := set.Classes[code]
		if len(g.Tiles) == g.Width*g.Height && g.Tiles[i]&0x2000 != 0 && class.Dead != nil {
			class = class.Dead
		}
		fire := class.FireObject
		if fire < 0 && fire != -2 {
			continue
		}
		out = append(out, ambientStaticCell{cell: image.Pt(i%g.Width, i/g.Width), fireObject: fire})
	}
	return out
}

func (v *Viewer) ambientSnapshot() ambientSnapshot {
	geometry, ok := v.soundGeometry()
	listener := geometry.Origin.Add(geometry.Span.Div(2))
	s := ambientSnapshot{geometry: geometry, listener: listener, hasView: ok}
	if !ok {
		return s
	}
	col0, row0 := max(geometry.Origin.X-geometry.Span.X, 8), max(geometry.Origin.Y-geometry.Span.Y, 8)
	col1, row1 := min(geometry.Origin.X+2*geometry.Span.X, v.grid.Width-8), min(geometry.Origin.Y+2*geometry.Span.Y, v.grid.Height-8)
	visible := func(cell image.Point) bool {
		return cell.X >= col0 && cell.X <= col1 && cell.Y >= row0 && cell.Y <= row1
	}
	if len(v.grid.Tiles) == v.grid.Width*v.grid.Height {
		for row := row0; row <= row1; row++ {
			for col := col0; col <= col1; col++ {
				if terrain.Resolve(v.grid.Tiles[row*v.grid.Width+col]).Water {
					s.river = append(s.river, image.Pt(col, row))
				}
			}
		}
	}
	for _, cell := range v.ambientWallFire {
		if visible(cell) {
			s.wallFire = append(s.wallFire, cell)
		}
	}
	for _, source := range v.ambientStatics {
		if !visible(source.cell) {
			continue
		}
		if source.fireObject >= 0 {
			s.birds = append(s.birds, source.cell)
		} else if source.fireObject == -2 {
			s.crows = append(s.crows, source.cell)
		}
	}
	return s
}

// SetAmbientAudio installs mission ambience. The ordinary sound player owns
// deadline one-shots; the retained device owns river and Wall of Fire loops.
func (v *Viewer) SetAmbientAudio(ones audio.Player, bank SoundBank, device AmbientDevice, seed int64) {
	if v == nil {
		return
	}
	v.StopAmbient()
	v.ambient = newAmbientController(bank, ones, device, seed)
}

// SetAmbientWallFire replaces the live Wall of Fire source cells. The copy and
// de-duplication keep overlapping canonical effect records one audible source.
func (v *Viewer) SetAmbientWallFire(cells []image.Point) {
	if v == nil || len(cells) == 0 {
		if v != nil {
			v.ambientWallFire = nil
		}
		return
	}
	seen := make(map[image.Point]struct{}, len(cells))
	out := make([]image.Point, 0, len(cells))
	for _, cell := range cells {
		if _, exists := seen[cell]; exists {
			continue
		}
		seen[cell] = struct{}{}
		out = append(out, cell)
	}
	v.ambientWallFire = out
}

func (v *Viewer) stepAmbient(now time.Time) {
	if v != nil && v.ambient != nil {
		v.ambient.update(now, v.ambientSnapshot())
	}
}

// StopAmbient is the map teardown seam. A retained Viewer cannot restart a
// departed map's loops because its controller is dropped here.
func (v *Viewer) StopAmbient() {
	if v == nil {
		return
	}
	if v.ambient != nil {
		v.ambient.stop()
	}
	v.ambient = nil
}

// SetAmbientDevice gives App ownership of process teardown without teaching
// App how a map chooses sources.
func (a *App) SetAmbientDevice(device AmbientDevice) {
	if a == nil {
		return
	}
	if a.ambientDevice != nil {
		a.ambientDevice.Stop()
	}
	a.ambientDevice = device
}

// StopAudio closes process-owned streams and retained delivery buffers.
func (a *App) StopAudio() {
	if a == nil {
		return
	}
	a.StopCutscene()
	a.StopMusic()
	if a.flow != nil {
		if town, ok := a.flow.town.(TownDialogueLifecycle); ok {
			town.TownDialogueActive(false)
		}
	}
	if a.ambientDevice != nil {
		a.ambientDevice.Stop()
	}
	if a.flow != nil && a.flow.viewer != nil {
		a.flow.viewer.DestroyAudio()
	}
	if owner := DeliveryOwner(a.soundPlayer); owner != nil {
		owner.Service.Close()
	}
}
