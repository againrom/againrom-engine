package ui

import (
	"slices"
	"testing"

	"againrom/pkg/words"
)

func commandNames(e *MapEditor) []string {
	var out []string
	for _, c := range []uint32{1, 2, 3, 4, 5, 10, 11, 14, 15, 17, 18} {
		out = append(out, e.commandName(c))
	}
	return out
}

func TestGroupCommandLabelsUseTheGamesCommandPanelWords(t *testing.T) {
	for _, tc := range []struct {
		language string
		want     []string
	}{
		{"", []string{"Guard", "Swarm as a crowd", "Stand Ground", "Move", "Swarm", "Attack", "Defend", "Patrol", "Follow", "Roam", "Dwell (no effect)"}},
		{"russian", []string{"Охранять", "Идти толпой в боевой готовности", "Держать позицию", "Идти", "Идти в боевой готовности", "Атаковать", "Защищать", "Патрулировать", "Следовать", "Бродить", "Ждать (ничего не делает)"}},
	} {
		e := groupOrderEditor(t)
		e.SetWords(words.For(tc.language))
		if got := commandNames(e); !slices.Equal(got, tc.want) {
			t.Fatalf("language %q: labels %q, want %q", tc.language, got, tc.want)
		}
	}
}

func TestGroupCommandLabelsReadTheInstallCommandPanel(t *testing.T) {
	e := groupOrderEditor(t)
	e.SetCommandWords([8]string{"Атаковать <A>", "Идти <M>", "Охранять <G>", "Защищать <D>", "Колдовать <C>", "Идти в боевой готовности <S>", "Держать позицию <T>", "Отступить <R>"})
	want := []string{"Охранять", "Swarm as a crowd", "Держать позицию", "Идти", "Идти в боевой готовности", "Атаковать", "Защищать", "Patrol", "Follow", "Roam", "Dwell (no effect)"}
	if got := commandNames(e); !slices.Equal(got, want) {
		t.Fatalf("labels %q, want %q", got, want)
	}
	e.SetCommandWords([8]string{})
	if got := e.commandName(5); got != "Swarm" {
		t.Fatalf("without install words command 5 reads %q, want the authored Swarm", got)
	}
}
