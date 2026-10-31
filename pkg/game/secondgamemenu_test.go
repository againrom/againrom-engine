package game

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/ui"
)

// The second game's town popup reads its five words from dialogs.txt rows 34,
// 35, 37, 77 and 40 (R2-ENGINE-441, R2-ASSET-098), written here as literals
// with distinct neighbours so a moved slot cannot stay green.
// The RU rows are the install's CP866 bytes of "~Сохранить игру",
// "~Восстановить игру", "~Звуковые опции", "~Выход из игры" and "Вер~нуться
// к игре".
func TestSecondGameMenuWordsAreTheDialogsRows(t *testing.T) {
	for _, want := range [][]string{
		{"~Save Game", "~Load Game", "Sou~nd Options", "Abort Game", "~Return to Game"},
		{"~\x91\xae\xe5\xe0\xa0\xad\xa8\xe2\xec \xa8\xa3\xe0\xe3", "~\x82\xae\xe1\xe1\xe2\xa0\xad\xae\xa2\xa8\xe2\xec \xa8\xa3\xe0\xe3",
			"~\x87\xa2\xe3\xaa\xae\xa2\xeb\xa5 \xae\xaf\xe6\xa8\xa8", "~\x82\xeb\xe5\xae\xa4 \xa8\xa7 \xa8\xa3\xe0\xeb",
			"\x82\xa5\xe0~\xad\xe3\xe2\xec\xe1\xef \xaa \xa8\xa3\xe0\xa5"},
	} {
		src := installFixture{DialogsTextPath: textFile(176, map[int]string{
			33: "before", 34: want[0], 35: want[1], 36: "Game ~Options", 37: want[2], 38: "after",
			39: "~End Quest", 40: want[4], 41: "after", 76: "before", 77: want[3], 78: "after",
		})}
		w := LoadInstallWords(src, TextCode{}).Words()
		got := []string{w.MenuSave, w.MenuLoad, w.MenuSoundOptions, w.MenuAbort, w.MenuReturn}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("popup words %q, want %q", got, want)
		}
	}
}

func TestSecondGameMenuProfileCarriesThePopup(t *testing.T) {
	second := filesOf(base.GameROM2).gameMenu
	if !reflect.DeepEqual(second.Rows, ui.SecondGameMenuRows) || second.TownRootPanel != image.Rect(100, 100, 540, 440) {
		t.Fatalf("second game menu rows %v panel %v", second.Rows, second.TownRootPanel)
	}
	if first := filesOf(base.GameROM1).gameMenu; !reflect.DeepEqual(first, ui.GameMenuProfile{}) {
		t.Fatalf("first game menu profile %+v, want the zero value", first)
	}
}
