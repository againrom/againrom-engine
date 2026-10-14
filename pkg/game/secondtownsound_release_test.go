package game

import (
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/ui"
)

// secondTownSoundLog keeps every effects or speech request the second town
// sends, as source:selector with its group, priority and loop.
type secondTownSoundLog struct{ lines []string }

func (r *secondTownSoundLog) Play(audio.Sample, audio.Placement) {}
func (r *secondTownSoundLog) RequestSample(s audio.Sample, q audio.Request) audio.Voice {
	r.lines = append(r.lines, fmt.Sprintf("%s:%s group=%d priority=%d loop=%v pcm=%v",
		q.Source, q.Selector, q.Group, q.Priority, q.Repeat, len(s.PCM) > 0))
	return nil
}

// TestReleaseSecondTownRequestsTheSameSounds drives the second game's first
// town through the App: the square, the shop with a buy, a sale and a shelf
// change, and the tavern with its first talk. It logs every request and holds
// them to the second town's own sound rows: no first-game room sample and no
// priority but 128 (the first game's room sound table is not the second
// game's).
func TestReleaseSecondTownRequestsTheSameSounds(t *testing.T) {
	f := secondGameFront(t)
	log := &secondTownSoundLog{}
	f.SoundPlayer, f.SpeechPlayer = log, log
	app := f.App("second town sounds")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.HeadlessActivate("new game"); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("ordinary NEW GAME first town", err, app.Screen())
	}
	// The menu's own requests before the town are not the town's.
	start := len(log.lines)
	steps := []string{}
	mark := func(name string) {
		steps = append(steps, fmt.Sprintf("%s@%d", name, len(log.lines)-start))
	}
	// Hover the whole square on a 16-pixel grid, then let it run.
	for y := 8; y < 768; y += 16 {
		for x := 8; x < 1024; x += 16 {
			if err := app.HeadlessPointer("hover", x, y); err != nil {
				t.Fatal(err)
			}
		}
	}
	secondTownSteps(t, app, 200)
	mark("square")
	secondShopActivate(t, app, "SHOP")
	mark("shop")
	f.Town.gold = 5000
	secondShopActivate(t, app, "ITEM 1", "BUY")
	mark("buy")
	secondShopActivate(t, app, "MINE 1", "SELL")
	mark("sell")
	secondShopActivate(t, app, "SHELF 3", "ITEM 1", "SHELF 1")
	mark("shelf")
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	mark("leave shop")
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	secondTownSteps(t, app, 600)
	mark("tavern")
	secondTownTalk(t, app)
	mark("talk")
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	mark("leave tavern")
	firstGame := []string{"put_on.wav", "put_off.wav", "scroll.wav", "town/shop/depart.wav", "town/shop/inshop.wav",
		"town/shop/start.wav", "town/shop/nofit.wav", "town/buy.wav", "town/sell.wav", "undo.wav", ":out.wav",
		":add.wav", "noadd.wav", ":talk.wav", "town/school/enter.wav", "registry:2"}
	town := log.lines[start:]
	for _, line := range town {
		for _, k := range firstGame {
			if strings.Contains(line, k+" ") {
				t.Errorf("the second town requested a first-game room sound: %s", line)
			}
		}
		if !strings.Contains(line, " priority=128 ") {
			t.Errorf("the second town requested a priority other than 128: %s", line)
		}
	}
	t.Logf("steps %s", strings.Join(steps, " "))
	for i, line := range town {
		t.Logf("request %d %s", i, line)
	}
}

func secondTownSteps(t *testing.T, app *ui.App, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
}
