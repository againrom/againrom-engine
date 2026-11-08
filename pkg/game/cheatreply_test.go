package game

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/encoding/charmap"

	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestCheatReplyUsesSessionTextAndWordSpaces(t *testing.T) {
	for _, tc := range []struct {
		name     string
		game     base.Game
		selector int
		fileCode *charmap.Charmap
		prefix   string
		suffix   string
		life     time.Duration
	}{
		{"ROM1_EN", base.GameROM1, 0, nil, "Before", "after", 3 * time.Second},
		{"ROM1_RU", base.GameROM1, 1, charmap.CodePage866, "Начало", "конец", 3 * time.Second},
		{"ROM2_EN", base.GameROM2, 0, nil, "Before", "after", 5 * time.Second},
		{"ROM2_RU", base.GameROM2, 1, charmap.Windows1251, "Начало", "конец", 5 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, edges := range []struct {
				name        string
				left, right string
			}{
				{"neither", "", ""},
				{"prefix", " ", ""},
				{"suffix", "", " "},
				{"both", " ", " "},
			} {
				t.Run(edges.name, func(t *testing.T) {
					rows := map[int]string{}
					for code := 5; code <= 7; code++ {
						index := 221 + 2*(code-5)
						rows[index] = fmt.Sprintf("%s%d%s", tc.prefix, code, edges.left)
						rows[index+1] = fmt.Sprintf("%s%s%d", edges.right, tc.suffix, code)
					}
					payload := textFile(227, rows)
					if tc.fileCode != nil {
						var err error
						payload, err = tc.fileCode.NewEncoder().Bytes(payload)
						if err != nil {
							t.Fatal(err)
						}
					}
					src := installFixture{MainTextPath: payload, LanguagePath: []byte(fmt.Sprint(tc.selector))}
					edition := tc.game.Edition()
					table := LoadTextTable(src, MainTextPath, InstallTextCode(src, edition))
					groups := make([]alm.Group, sim.SelfSlot)
					groups[sim.SelfSlot-1].Name = "Self"
					view := &ui.Viewer{}
					font := missionFont()
					font.Selector = tc.selector
					view.SetFont(font)
					view.Layout(1024, 768)
					mw := &mapWorld{view: view, mission: &missionNotices{
						src: src, table: &mapload.Table{Edition: &edition},
						state: &Mission{Map: &alm.Map{Groups: groups}},
						party: []mapload.PartyMember{{Name: "Hero"}},
					}}
					for code := 5; code <= 7; code++ {
						t.Run(fmt.Sprint(code), func(t *testing.T) {
							view.ClearMessages()
							mw.cheatReply(code, sim.SelfSlot)
							index := 221 + 2*(code-5)
							prefix, _ := table.At(index)
							suffix, _ := table.At(index + 1)
							want := EncodeInstallText(fmt.Sprintf("%s%d Self %s%d", tc.prefix, code, tc.suffix, code), tc.selector)
							if session := strings.TrimRight(prefix, " ") + " Self " + strings.TrimLeft(suffix, " "); session != want {
								t.Fatalf("session text = %q, want %q", session, want)
							}
							got := view.MessageLines()
							if len(got) != 1 || got[0].Text != want || got[0].Ink != ui.MessageWhite || got[0].Life != tc.life {
								t.Fatalf("reply = %+v, want one white %q for %s", got, want, tc.life)
							}
						})
					}
				})
			}
		})
	}
}
