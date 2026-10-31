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
// them to the second town's own sound rows: the shop's entry, buy and sale
// and the inn's TALK request the first game's room sounds (DIV-4372), no
// request names a sound the description does not, and every priority is
// 128.
func TestReleaseSecondTownRequestsTheSameSounds(t *testing.T) {
	f := secondGameFront(t)
	log := &secondTownSoundLog{}
	f.SoundPlayer, f.speech.player = log, log
	app := f.App("second town sounds")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.HeadlessActivate("new game"); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("ordinary NEW GAME first town", err, app.Screen())
	}
	// The menu's own requests before the town are not the town's.
	start := len(log.lines)
	steps := []string{}
	at := map[string]int{}
	mark := func(name string) {
		at[name] = len(log.lines) - start
		steps = append(steps, fmt.Sprintf("%s@%d", name, at[name]))
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
	secondShopActivate(t, f, app, "SHOP")
	mark("shop")
	f.Town.gold = 5000
	secondShopActivate(t, f, app, "ITEM 1", "BUY")
	mark("buy")
	secondShopActivate(t, f, app, "MINE 1", "SELL")
	mark("sell")
	secondShopActivate(t, f, app, "SHELF 3", "ITEM 1", "SHELF 1")
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
	unnamed := []string{"town/school/enter.wav", "town/school/teach.wav", "registry:2"}
	town := log.lines[start:]
	for _, line := range town {
		for _, k := range unnamed {
			if strings.Contains(line, k+" ") {
				t.Errorf("the second town requested a sound its description does not name: %s", line)
			}
		}
		if !strings.Contains(line, " priority=128 ") {
			t.Errorf("the second town requested a priority other than 128: %s", line)
		}
	}
	for _, w := range []struct{ from, to, key string }{
		{"square", "shop", "town/shop/enter.wav"}, {"shop", "buy", "town/buy.wav"},
		{"buy", "sell", "town/sell.wav"}, {"tavern", "talk", ":talk.wav"},
	} {
		found := false
		for _, line := range town[at[w.from]:at[w.to]] {
			found = found || strings.Contains(line, w.key+" ")
		}
		if !found {
			t.Errorf("between %s and %s the second town requested no %s", w.from, w.to, w.key)
		}
	}
	t.Logf("steps %s", strings.Join(steps, " "))
	for i, line := range town {
		t.Logf("request %d %s", i, line)
	}
}

// TestReleaseSecondTownRoomSoundKeysResolve reads every key the three second
// towns' shop and inn slots and triggers name from the root's own sound
// archive: each decodes, Kin1 excepted, which the archive lacks (DIV-4271).
func TestReleaseSecondTownRoomSoundKeysResolve(t *testing.T) {
	archives := secondGameRoot(t)
	bank := &SoundBank{src: archives.Containers.Fork()}
	absent := map[string]bool{"town_kaarg/inn/kin1.wav": true}
	for _, name := range []string{"rom2-town1", "rom2-town2", "rom2-town3"} {
		for _, room := range []string{"shop", "inn"} {
			sounds := townDescriptions[name].Scene(room).Sounds
			keys := []string{}
			for _, sl := range sounds.Slots {
				keys = append(keys, sl.Key)
			}
			for _, tr := range sounds.Triggers {
				if tr.Key != "" {
					keys = append(keys, tr.Key)
				}
			}
			for _, k := range keys {
				if _, ok := bank.namedSample(k); ok == absent[k] {
					t.Errorf("%s %s key %s: decoded %v, want %v", name, room, k, ok, !absent[k])
				}
			}
		}
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
