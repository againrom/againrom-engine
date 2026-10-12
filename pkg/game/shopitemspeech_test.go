package game

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/sim"
)

// shopItemSpeech names one recording: an effect description first, then a
// spell description, then one drawn class/row part; a recording the install
// lacks is passed over and an item with none is silent (SHOP-109, DIV-2115).
func TestShopItemSpeechNamesOneRecording(t *testing.T) {
	have := func(names ...string) func(string) bool {
		return func(name string) bool {
			for _, n := range names {
				if n == name {
					return true
				}
			}
			return false
		}
	}
	draw := func(v int) func(int) int { return func(int) int { return v } }
	parts := []string{"speech/shop/s01i02p1.wav", "speech/shop/s01i02p2.wav", "speech/shop/s01i02p3.wav"}
	plain := ShopItem{Code: 0x0102, Count: 1}
	for v, want := range parts {
		if got, ok := shopItemSpeech(plain, have(parts...), draw(v)); !ok || got != want {
			t.Fatalf("draw %d named %q, %v; want %q", v, got, ok, want)
		}
	}
	if got, _ := shopItemSpeech(plain, have(parts[0], parts[2]), draw(1)); got != parts[2] {
		t.Fatalf("a missing middle part: draw 1 named %q, want %q", got, parts[2])
	}
	if got, ok := shopItemSpeech(plain, have(), draw(0)); ok {
		t.Fatalf("an item without recordings named %q", got)
	}
	magic := ShopItem{Code: 0x0102, Count: 1, Effects: []sim.ItemEffect{{Kind: 3, Operand: 1}}}
	effect := "speech/shop/effects/03.wav"
	if got, _ := shopItemSpeech(magic, have(append([]string{effect}, parts...)...), draw(0)); got != effect {
		t.Fatalf("an item with a voiced effect named %q, want %q", got, effect)
	}
	if got, _ := shopItemSpeech(magic, have(parts...), draw(1)); got != parts[1] {
		t.Fatalf("an item whose effect has no recording named %q, want %q", got, parts[1])
	}
}

// A merchant stack with three recorded parts, moved onto the table through
// App, speaks one part and nothing after it completes. Loss control: the same
// screen given the three parts as one queued request, the request each
// selection made before, plays all three.
func TestShopMerchantSpeaksOneLinePerSelection(t *testing.T) {
	f, app, s, r := shopCommentFixture(t)
	f.SpeechBank = responseBank(t, []synth.File{
		{Path: "shop/s01i02p1.wav", Data: speechWAV(1001)},
		{Path: "shop/s01i02p2.wav", Data: speechWAV(1002)},
		{Path: "shop/s01i02p3.wav", Data: speechWAV(1003)},
	})
	f.ShopRandom = func(int) int { return 1 }
	point := shopPointer(t, app)
	drain := func() {
		t.Helper()
		for n := 0; n < 8; n++ {
			for _, v := range r.voices {
				v.playing = false
			}
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
	}

	point("shelf", 0, "press", "release")
	drain()
	if len(r.samples) != 1 || r.samples[0].PCM[0] != 1002 {
		t.Fatalf("one selection played %d recording(s); want one, the drawn second part", len(r.samples))
	}

	before := len(r.samples)
	s.startTownResponse([]string{"speech/shop/s01i02p1.wav", "speech/shop/s01i02p2.wav", "speech/shop/s01i02p3.wav"})
	drain()
	if got := len(r.samples) - before; got != 3 {
		t.Fatalf("loss control: the three-part request played %d recording(s), want 3", got)
	}
}
