package game

import (
	"image"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"againrom/internal/firstgame"
	"againrom/pkg/audio"
	"againrom/pkg/ui"
)

// installedSoundRecorder keeps every request with the length of the
// installed sample it carries, and answers a voice the witness ends.
type installedSoundRecorder struct {
	requests []audio.Request
	lengths  []int
	voices   []*exteriorVoice
}

func (r *installedSoundRecorder) Play(audio.Sample, audio.Placement) {}
func (r *installedSoundRecorder) RequestSample(s audio.Sample, q audio.Request) audio.Voice {
	v := &exteriorVoice{playing: true}
	r.requests = append(r.requests, q)
	r.lengths = append(r.lengths, len(s.PCM))
	r.voices = append(r.voices, v)
	return v
}

func (r *installedSoundRecorder) endAll() {
	for _, v := range r.voices {
		v.playing = false
	}
}

// since names the requests from index from on as selector, with
// "@speech", "@loop" and "@<priority>" marks for the parameters the census
// gives, and fails on a request that carries no installed sample.
func (r *installedSoundRecorder) since(t *testing.T, from int) []string {
	t.Helper()
	out := []string{}
	for i, q := range r.requests[from:] {
		if r.lengths[from+i] == 0 {
			t.Fatalf("request %s carries no installed sample", q.Selector)
		}
		k := q.Selector
		if q.Group == audio.SpeechChannel {
			k += "@speech"
		}
		if q.Repeat {
			k += "@loop"
		}
		if q.Priority != 128 {
			k += "@" + strconv.Itoa(int(q.Priority))
		}
		out = append(out, k)
	}
	return out
}

func wantSounds(t *testing.T, rec *installedSoundRecorder, from int, what string, want ...string) {
	t.Helper()
	if want == nil {
		want = []string{}
	}
	if got := rec.since(t, from); !reflect.DeepEqual(got, want) {
		t.Fatalf("%s requested %q, want %q", what, got, want)
	}
	rec.endAll()
}

// wantThenSpeech is wantSounds where a dialogue's own speech lines may
// follow the census rows.
func wantThenSpeech(t *testing.T, rec *installedSoundRecorder, from int, what string, want ...string) {
	t.Helper()
	got := rec.since(t, from)
	if len(got) < len(want) || !reflect.DeepEqual(got[:len(want)], want) {
		t.Fatalf("%s requested %q, want %q first", what, got, want)
	}
	for _, k := range got[len(want):] {
		if !strings.HasPrefix(k, "speech/") {
			t.Fatalf("%s requested %q after the census rows", what, k)
		}
	}
	rec.endAll()
}

// releasePress sends press and release at p through the App.
func releasePress(t *testing.T, a *ui.App, p image.Point, edges ...string) {
	t.Helper()
	for _, e := range edges {
		if err := a.HeadlessPointer(e, p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
}

// surfacePoint is a point the production hit test resolves to control c on
// the open tavern or school surface.
func surfacePoint(t *testing.T, s *townScreen, c ui.TownSurfaceControl) image.Point {
	t.Helper()
	v := s.TownSurface()
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			if got, ok := ui.TownSurfaceControlAt(v, image.Pt(x, y)); ok && got == c {
				return image.Pt(x, y)
			}
		}
	}
	t.Fatalf("no point answers %+v", c)
	return image.Point{}
}

func tipClosePoint(tip ui.TipPanelView) image.Point {
	r := ui.TipPanelCloseRect(tip.Rect)
	return r.Min.Add(r.Max).Div(2)
}

// squareMaskPoint is a square pixel whose installed mask byte is b.
func squareMaskPoint(t *testing.T, f *FrontEnd, b uint8) image.Point {
	t.Helper()
	mask := f.TownSquareArt.Value().Mask
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			if mask.ColorIndexAt(x, y) == b {
				return image.Pt(x, y)
			}
		}
	}
	t.Fatalf("the installed mask has no byte %d", b)
	return image.Point{}
}

func enterTownRoom(t *testing.T, a *ui.App, s *townScreen, row string, room townRoom) {
	t.Helper()
	if err := a.HeadlessActivate(row); err != nil {
		t.Fatal(err)
	}
	for n := 0; s.room == roomTalk && n < 32; n++ {
		if err := a.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if s.room != room {
		t.Fatalf("%s entered room %d, want %d", row, s.room, room)
	}
}

// TestReleaseTownRoomsRequestTheCensusSounds drives the installed town through
// the App's pointer: the square's tip Close and building hovers, the shop, the
// tavern and the school, each request carrying an installed sample and the
// census's parameters (town sound census rows S3..S21, S28, T1, Q1, Q3, V1,
// V4..V9, V11, C1, C4). A last pass with the shop's trigger rows removed
// requests nothing for the same gestures.
func TestReleaseTownRoomsRequestTheCensusSounds(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	f.SetTipsOff(false)
	campaign := f.Campaign.Value()
	chapter := releaseMercenaryChapter(t, campaign)
	tn := NewTown(campaign)
	for mission := range campaign.Chapters {
		if mission < chapter {
			tn.Won(mission)
		}
	}
	f.Town = tn
	f.Carried = f.NextParty()
	f.arriveInTown()
	f.Town.announceMission(f.Town.currentMain())
	now := time.Unix(400, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.TownAnimationRandom = func(int) int { return 0 }
	f.ShopRandom = func(int) int { return 99 }
	rec := &installedSoundRecorder{}
	f.SoundPlayer, f.speech.player, f.MusicPlayer, f.CutsceneAudioPlayer = rec, rec, nil, nil
	f.AmbientPlayer = &townCrowdRecorder{}
	a := f.App("town room sounds")
	t.Cleanup(a.StopAudio)
	a.Layout(640, 480)
	a.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town.ags", Label: "Town"}} },
		func(string) (ui.MapOpener, bool, error) { return nil, true, nil })
	if err := a.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ui.ScreenTown {
		t.Fatalf("load route: %v", a.Screen())
	}
	s := f.TownScreen().(*townScreen)
	if s.room != roomSquare {
		t.Fatalf("the town opened room %d, want the square", s.room)
	}
	rec.endAll()

	// Square: the tip's Close press clicks; the shop and school hovers latch
	// their sounds (TOWN-547, TOWN-549).
	tip := s.TownSquareView().Tip
	if !tip.Showing() {
		t.Fatal("the square shows no tip")
	}
	base := len(rec.requests)
	releasePress(t, a, tipClosePoint(tip), "press")
	wantSounds(t, rec, base, "the square tip Close press", "registry:2@220")
	releasePress(t, a, tipClosePoint(tip), "release")
	wantSounds(t, rec, base+1, "the square tip Close release")
	base = len(rec.requests)
	releasePress(t, a, squareMaskPoint(t, f, 144), "hover")
	releasePress(t, a, squareMaskPoint(t, f, 192), "hover")
	wantSounds(t, rec, base, "the shop and school hovers", "town/shop/enter.wav", "town/school/point.wav")
	releasePress(t, a, image.Pt(0, 0), "hover")

	// Shop: entry, tip, table drop and pick, the buttons' presses, trades,
	// arrows and shelf change (SHOP-SOUND-135..138).
	base = len(rec.requests)
	enterTownRoom(t, a, s, "SHOP", roomShop)
	// The merchant's greeting dialogue pages its own speech after the
	// entry requests.
	wantThenSpeech(t, rec, base, "the shop entry", "town/shop/depart.wav", "town/shop/enter.wav",
		"town/shop/inshop.wav@loop", "town/shop/start.wav@speech")
	var entry []string
	base = len(rec.requests)
	releasePress(t, a, tipClosePoint(s.ShopScreen().TipPanel), "press", "release")
	wantSounds(t, rec, base, "the shop tip Close", "registry:2@220")
	shop := func(kind string, index int, edges ...string) {
		t.Helper()
		x, y, err := a.HeadlessShopPoint(kind, index)
		if err != nil {
			t.Fatal(err)
		}
		releasePress(t, a, image.Pt(x, y), edges...)
	}
	base = len(rec.requests)
	shop("shelf", 0, "press", "release")
	wantSounds(t, rec, base, "a shelf item to the table", "put_on.wav")
	base = len(rec.requests)
	shop("table", 0, "press", "release")
	wantSounds(t, rec, base, "a table pick", "put_off.wav")
	shop("shelf", 0, "press", "release")
	rec.endAll()
	base = len(rec.requests)
	shop("button", 0, "press")
	wantSounds(t, rec, base, "the Undo press", "undo.wav")
	base = len(rec.requests)
	shop("button", 0, "release")
	wantSounds(t, rec, base, "the table clear", "put_off.wav")
	shop("shelf", 0, "press", "release")
	rec.endAll()
	table := f.Shop.Table()
	if len(table) != 1 {
		t.Fatalf("before the buy the table holds %d place(s), want 1", len(table))
	}
	bought := uint16(table[0].ShopItem.Code)
	gold := f.Town.gold
	f.Town.gold = 0
	base = len(rec.requests)
	shop("button", 1, "press", "release")
	wantSounds(t, rec, base, "a short buy", "town/shop/nofit.wav")
	f.Town.gold = gold + 100000
	base = len(rec.requests)
	shop("button", 1, "press", "release")
	wantSounds(t, rec, base, "a buy", "town/buy.wav")
	base = len(rec.requests)
	shop("button", 1, "press", "release")
	shop("button", 2, "press", "release")
	wantSounds(t, rec, base, "an empty buy and sale")
	shop("pack", currentTownPackCell(t, s, bought), "press", "release")
	rec.endAll()
	if f.Shop.SellTotal() == 0 {
		t.Fatal("the table holds no sale value")
	}
	base = len(rec.requests)
	shop("button", 2, "press", "release")
	wantSounds(t, rec, base, "a sale", "town/sell.wav")
	base = len(rec.requests)
	s.ShopPress(ui.ShopControl{Kind: ui.ShopControlArrowDown})
	s.ShopPress(ui.ShopControl{Kind: ui.ShopControlPackRight})
	shop("shelf_pick", 1, "press", "release")
	wantSounds(t, rec, base, "the arrows and a shelf change", "scroll.wav", "scroll.wav", "town/shop/depart.wav")
	base = len(rec.requests)
	shop("button", 3, "press")
	wantSounds(t, rec, base, "the Out press", "out.wav")
	shop("button", 3, "release")
	if s.room != roomSquare {
		t.Fatalf("Out left room %d, want the square", s.room)
	}
	rec.endAll()

	// Tavern: entry, tip, roster press, hire refused, hire, dismissal, the
	// biography and the leave press (TAVERN-027, TAVERN-028).
	base = len(rec.requests)
	enterTownRoom(t, a, s, "TAVERN", roomTavern)
	entry = rec.since(t, base)
	if len(entry) == 0 || entry[0] != "town/inn/enter.wav" {
		t.Fatalf("the tavern entry requested %q, want Inn/enter first", entry)
	}
	rec.endAll()
	base = len(rec.requests)
	releasePress(t, a, tipClosePoint(s.TownSurface().Tip), "press", "release")
	wantSounds(t, rec, base, "the tavern tip Close", "registry:2@220")
	merc := -1
	for i, c := range s.tavernCandidates() {
		if c.key.kind == tavernCandidateMercenary {
			merc = i
			break
		}
	}
	if merc < 0 {
		t.Fatal("the tavern offers no mercenary")
	}
	cell := surfacePoint(t, s, ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: merc})
	base = len(rec.requests)
	releasePress(t, a, cell, "press", "release")
	wantSounds(t, rec, base, "a roster cell press", "town/inn/helper.wav")
	// Short of money the hire button is disabled; the roster's double click
	// reaches the refused hire (TAVERN-CLICK-019) and its press repeats Helper.
	f.Town.gold = 0
	base = len(rec.requests)
	releasePress(t, a, cell, "press", "release", "press", "release")
	refused := 0
	for _, k := range rec.since(t, base) {
		switch k {
		case "town/shop/nofit.wav":
			refused++
		case "town/inn/helper.wav":
		default:
			t.Fatalf("a double click short of money requested %q", k)
		}
	}
	if refused != 1 {
		t.Fatalf("a double click short of money requested nofit %d time(s), want 1", refused)
	}
	rec.endAll()
	f.Town.gold = gold + 100000
	hire := surfacePoint(t, s, ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonHire})
	base = len(rec.requests)
	releasePress(t, a, hire, "press", "release")
	wantSounds(t, rec, base, "a hire", "add.wav")
	base = len(rec.requests)
	releasePress(t, a, hire, "press", "release")
	wantSounds(t, rec, base, "a dismissal", "noadd.wav")
	talk := surfacePoint(t, s, ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonTalk})
	base = len(rec.requests)
	releasePress(t, a, talk, "press", "release")
	wantThenSpeech(t, rec, base, "the mercenary biography", "talk.wav")
	for n := 0; s.room == roomTalk && n < 32; n++ {
		if err := a.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
	}
	rec.endAll()
	leave := surfacePoint(t, s, ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonExit})
	base = len(rec.requests)
	releasePress(t, a, leave, "press")
	wantSounds(t, rec, base, "the tavern leave press", "out.wav")
	releasePress(t, a, leave, "release")
	if s.room != roomSquare {
		t.Fatalf("the tavern leave left room %d, want the square", s.room)
	}
	rec.endAll()

	// School: entry, tip and the leave press (TOWN-551).
	base = len(rec.requests)
	enterTownRoom(t, a, s, "SCHOOL", roomSchool)
	entry = rec.since(t, base)
	if len(entry) == 0 || entry[0] != "town/school/enter.wav" {
		t.Fatalf("the school entry requested %q, want School/Enter first", entry)
	}
	rec.endAll()
	base = len(rec.requests)
	releasePress(t, a, tipClosePoint(s.TownSurface().Tip), "press", "release")
	wantSounds(t, rec, base, "the school tip Close", "registry:2@220")
	leave = surfacePoint(t, s, ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 1})
	base = len(rec.requests)
	releasePress(t, a, leave, "press")
	wantSounds(t, rec, base, "the school leave press", "out.wav")
	releasePress(t, a, leave, "release")
	if s.room != roomSquare {
		t.Fatalf("the school leave left room %d, want the square", s.room)
	}
	rec.endAll()

	// Control: without the shop's trigger rows the same gestures request
	// nothing from the shop's table.
	enterTownRoom(t, a, s, "SHOP", roomShop)
	rec.endAll()
	withShopTriggers(t, s.townDescription(), nil)
	base = len(rec.requests)
	shop("shelf", 0, "press", "release")
	shop("table", 0, "press", "release")
	shop("button", 3, "press")
	for _, k := range rec.since(t, base) {
		if !strings.HasPrefix(k, "registry:") {
			t.Fatalf("with no trigger rows the shop requested %q", k)
		}
	}
	t.Logf("%d installed requests matched the census rows", len(rec.requests))
}

// TestReleaseMissionTipClosePressRequestsClick01: in mission 10 the tip's
// Close press requests click01 at priority 220 and its release nothing
// (TOWN-547; census row T2).
func TestReleaseMissionTipClosePressRequestsClick01(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := missionTipFront(t)
	rec := &installedSoundRecorder{}
	f.SoundPlayer = rec
	app, live := openMissionTipMap(t, f, 10)
	pageMissionDialogue(t, app, live, "start dialogue")
	wantMissionTip(t, f, live, 10, 2, "start trigger")
	clicks := func() int {
		n := 0
		for i, q := range rec.requests {
			if q.Selector == "registry:2" {
				if rec.lengths[i] == 0 || q.Priority != 220 || q.Group != audio.EffectsChannel || q.Repeat {
					t.Fatalf("click01 request %+v", q)
				}
				n++
			}
		}
		return n
	}
	before := clicks()
	closeAt := ui.TipPanelCloseRect(missionTipRect).Min.Add(image.Pt(4, 4))
	releasePress(t, app, closeAt, "press")
	if got := clicks() - before; got != 1 {
		t.Fatalf("the mission tip Close press requested click01 %d time(s), want 1", got)
	}
	releasePress(t, app, closeAt, "release")
	if got := clicks() - before; got != 1 || live.view.MissionTip().Showing() {
		t.Fatalf("after the release click01 %d time(s), tip showing %v", got, live.view.MissionTip().Showing())
	}
}
