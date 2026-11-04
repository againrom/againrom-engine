package game

import (
	"fmt"
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/audio"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type shopSpeechRecorder struct {
	tavernInteriorRecorder
	requests []audio.Request
}

func (r *shopSpeechRecorder) RequestSample(s audio.Sample, request audio.Request) audio.Voice {
	r.requests = append(r.requests, request)
	return r.tavernInteriorRecorder.RequestSample(s, request)
}

func TestSecondShopTableRequestsItemSpeech(t *testing.T) {
	for townID, dir := range []string{"shop", "shop_kaarg", "shop_druid"} {
		for _, tc := range []struct {
			name    string
			effects []sim.ItemEffect
			raw     []int
			want    string
		}{
			{"first effect tag", []sim.ItemEffect{{Kind: 1}, {Kind: 3, Operand: 99}, {Kind: 5}}, []int{0}, "effects/03.wav"},
			{"last book value", []sim.ItemEffect{{Kind: 42, Operand: 7}, {Kind: 42, Operand: 12}}, []int{0}, "books/12.wav"},
			{"zero last book", []sim.ItemEffect{{Kind: 42, Operand: 7}, {Kind: 42}}, []int{0, 21845}, "s06i05p3.wav"},
			{"masked code", nil, []int{0, 10923}, "s06i05p2.wav"},
			{"endpoint wraps", nil, []int{32767, 32767}, "s06i05p1.wav"},
			{"gate at thirty", nil, []int{9831}, ""},
			{"gate below thirty", nil, []int{9830, 0}, "s06i05p1.wav"},
			{"missing effect stays silent", []sim.ItemEffect{{Kind: 4}, {Kind: 3}}, []int{0}, ""},
			{"missing book stays silent", []sim.ItemEffect{{Kind: 42, Operand: 13}}, []int{0}, ""},
			{"excluded effect tag", []sim.ItemEffect{{Kind: 26}}, []int{0, 0}, "s06i05p1.wav"},
		} {
			t.Run(fmt.Sprintf("town%d/%s", townID+1, tc.name), func(t *testing.T) {
				c := secondShopSample()
				c.current, c.room = partyPlace{placeTown, townID + 1}, secondTownShop
				s := secondShopPage(c, 10000)
				s.defaults = secondTownDescription(townID + 1)
				item := c.shop.shelves[0][0]
				item.Code, item.Effects = 0xa625, tc.effects
				c.shop.shelves[0] = []ShopItem{item}
				var files []synth.File
				keys := []string{"effects/03.wav", "effects/05.wav", "books/07.wav", "books/12.wav", "s06i05p1.wav", "s06i05p2.wav", "s06i05p3.wav"}
				for i, key := range keys {
					files = append(files, synth.File{Path: dir + "/" + key, Data: speechWAV(int16(1000 + i))})
				}
				s.in.SpeechBank = responseBank(t, files)
				recorder := &shopSpeechRecorder{}
				rt := &RuntimeServices{}
				rt.SoundPlayer = &tavernInteriorRecorder{}
				rt.speech.player = recorder
				var draws []int
				rt.ShopRandom = func(n int) int {
					draws = append(draws, n)
					if len(draws) > len(tc.raw) {
						t.Fatalf("extra draw %v", draws)
					}
					return tc.raw[len(draws)-1]
				}
				s.sound, s.draws = runtimeTownAudio{rt}, runtimeTownDraws{rt}
				s.ShopClick(ui.ShopControl{Kind: ui.ShopControlShelfCell})
				s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlShelfCell}, ui.ShopControl{Kind: ui.ShopControlPackCell})
				if len(draws) != 0 || len(recorder.samples) != 0 {
					t.Fatal("a stock click or a release over the bag requested table speech")
				}
				c.shop.table = nil
				c.shop.shelves[0] = []ShopItem{item}
				s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlShelfCell}, ui.ShopControl{Kind: ui.ShopControlTableCell})
				if len(c.shop.Table()) != 1 {
					t.Fatal("fixture did not release a shelf item onto the table")
				}
				if s.speechTableRelease {
					t.Fatal("release context survived the gesture")
				}
				wantDraws := make([]int, len(tc.raw))
				for i := range wantDraws {
					wantDraws[i] = 32768
				}
				if !slices.Equal(draws, wantDraws) {
					t.Fatalf("draws %v, want %v", draws, wantDraws)
				}
				if tc.want == "" {
					if len(recorder.samples) != 0 {
						t.Fatal("silent request played a substitute")
					}
					return
				}
				at := slices.Index(keys, tc.want)
				if len(recorder.samples) != 1 || recorder.samples[0].PCM[0] != int16(1000+at) {
					t.Fatalf("played %d lines, want %s/%s", len(recorder.samples), dir, tc.want)
				}
				if request := recorder.requests[0]; request.Selector != "speech/"+dir+"/"+tc.want || request.Source != "town-response" {
					t.Fatalf("request %+v", request)
				}
				s.TownDialogueActive(true)
				if len(recorder.samples) != 1 || !recorder.voices[0].Playing() {
					t.Fatal("the ordinary room update lost or repeated the response")
				}
				s.campaignTown().setRoom("square")
				s.TownDialogueActive(true)
				if recorder.voices[0].stops != 1 {
					t.Fatal("leaving the campaign shop kept its response playing")
				}
			})
		}
	}
}
