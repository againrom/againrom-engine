package game

import (
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/random"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// townAudio is every sound device the town screen plays through, asked for at
// the moment it plays. A nil device is a lawful silent state.
type townAudio interface {
	soundDevice() audio.Player
	speechDevice() audio.Player
	ambientDevice() ui.AmbientDevice
	wildlifeSeed() int64
}

// townDraws is the town screen's presentation clock and bounded draw sources.
// A nil member means the screen's own fallback.
type townDraws interface {
	animationClock() func() time.Time
	animationDraw() func(n int) int
	ambientDraw() func(n int) int
	tavernDraw() func(n int) int
	shopDraw() func(n int) int
	schoolDraw() func(n int) int
	// stream is the session's named stream a source without a seam draws on.
	stream(name random.Name) *random.Stream
}

// townArt is the first-use art the town screen draws with.
type townArt interface {
	menuFrame() *ui.MenuPanelArt
	characterPanes() TownCharacterPaneArt
	characterCorners() *ui.CharacterPaneCornerArt
	shopScreen() *ui.ShopScreenArt
	tipPanel() *ui.TipPanelArt
	documentFont() *text.Font
	worldMap() *worldMapAssets
}

// runtimeTownAudio is the production townAudio over the process's players.
type runtimeTownAudio struct{ rt *RuntimeServices }

func (a runtimeTownAudio) soundDevice() audio.Player       { return a.rt.SoundPlayer }
func (a runtimeTownAudio) speechDevice() audio.Player      { return a.rt.SpeechPlayer }
func (a runtimeTownAudio) ambientDevice() ui.AmbientDevice { return a.rt.AmbientPlayer }
func (a runtimeTownAudio) wildlifeSeed() int64 {
	return a.rt.randomService().StreamSeed(random.TownWildlife)
}

// runtimeTownDraws is the production townDraws over the process's services.
type runtimeTownDraws struct{ rt *RuntimeServices }

func (d runtimeTownDraws) animationClock() func() time.Time { return d.rt.TownAnimationNow }
func (d runtimeTownDraws) animationDraw() func(n int) int   { return d.rt.TownAnimationRandom }
func (d runtimeTownDraws) ambientDraw() func(n int) int     { return d.rt.TownAmbientRandom }
func (d runtimeTownDraws) tavernDraw() func(n int) int      { return d.rt.TavernRandom }
func (d runtimeTownDraws) shopDraw() func(n int) int        { return d.rt.ShopRandom }
func (d runtimeTownDraws) schoolDraw() func(n int) int      { return d.rt.SchoolRandom }
func (d runtimeTownDraws) stream(name random.Name) *random.Stream {
	return d.rt.randomService().Stream(name)
}

// installTownArt is the production townArt over the install and the
// presentation's first-use caches.
type installTownArt struct {
	in *InstallResources
	pr *Presentation
}

func (a installTownArt) menuFrame() *ui.MenuPanelArt { return a.pr.gameMenuArt(a.in) }
func (a installTownArt) characterPanes() TownCharacterPaneArt {
	return a.pr.characterPanes(a.in)
}
func (a installTownArt) characterCorners() *ui.CharacterPaneCornerArt {
	return a.pr.characterPaneCorners(a.in)
}
func (a installTownArt) shopScreen() *ui.ShopScreenArt { return a.pr.shopArt(a.in) }
func (a installTownArt) tipPanel() *ui.TipPanelArt     { return a.pr.tipArt(a.in) }
func (a installTownArt) documentFont() *text.Font      { return a.pr.documentFont(a.in) }
func (a installTownArt) worldMap() *worldMapAssets     { return a.pr.worldMapAssets(a.in) }

// bindServices gives the screen its production services and builds its
// square view, so no later reader changes the screen by building it.
func (t *townScreen) bindServices(rt *RuntimeServices, in *InstallResources, pr *Presentation) {
	t.sound, t.draws = runtimeTownAudio{rt}, runtimeTownDraws{rt}
	t.art, t.townProcess = installTownArt{in, pr}, &pr.townProcess
	if d := t.townDescription(); d != nil {
		t.square = newSquareScene(t, d)
	}
}
