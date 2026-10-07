package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func TestMusic1189PreferencesColdStartAndFailedWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	if err := os.WriteFile(path, []byte("Other=kept\nMusicEnabled=1\nMusicEnabled=0\nRandomOrder=0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	f := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	c := ui.SoundOptionControls{}
	f.wireMusicPreferences(&c)
	if c.ReadPlayback() != (ui.MusicPreferences{}) {
		t.Fatal("startup ignored disabled preferences")
	}
	on := ui.MusicPreferences{Enabled: true, RandomOrder: true}
	if err := c.WritePlayback(on); err != nil {
		t.Fatal(err)
	}
	if cold, err := (OptionsStore{Path: path}).MusicPreferences(); err != nil || cold != on {
		t.Fatal("cold startup", cold, err)
	}
	b, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(b), "Other=kept\n") {
		t.Fatal("unrelated preference lost", err)
	}
	f.Options.Path = t.TempDir()
	if err := c.WritePlayback(ui.MusicPreferences{}); err == nil || c.ReadPlayback() != on {
		t.Fatal("failed write changed live preferences")
	}
	if got, err := (OptionsStore{}).MusicPreferences(); err != nil || got != ui.DefaultMusicPreferences() {
		t.Fatal("missing profile", got, err)
	}
	titles := musicTitles(SplitTextTable([]byte("b03.wav=First\r\nb03.wav=Last\r\nwrong\r\ntown.wav=Town\r\n")))
	if titles["b03.wav"] != "Last" || len(titles) != 2 {
		t.Fatal("tune dictionary", titles)
	}
}
