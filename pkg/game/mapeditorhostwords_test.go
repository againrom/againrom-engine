package game

import (
	"fmt"
	"image/color"
	"os"
	"regexp"
	"strings"
	"testing"
)

var cyrillic = regexp.MustCompile(`[а-яА-Я]`)

// Every refusal the speaker host returns reads in the editor's language.
func TestSpeakerHostRefusalsAreInTheEditorsLanguage(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			s, _, book := modAssetProject(t, lang)
			small, _ := assetFile(t, "small.png", assetPNG(t, 4, 4, color.RGBA{A: 255}))
			for _, c := range []struct {
				name string
				unit int
				pic  string
				want string
			}{
				{"no name", speakerLady, small, book.Text("editor.speaker.add.noname")},
				{"no unit", 9999, small, fmt.Sprintf(book.Text("editor.speaker.err.nounit"), 9999)},
				{"small face", speakerLady, small, fmt.Sprintf(book.Text("editor.speaker.err.size"), 4, 4, speakerPictureSize().X, speakerPictureSize().Y)},
			} {
				name := "Lady"
				if c.name == "no name" {
					name = ""
				}
				_, err := s.AddSpeaker(c.unit, name, c.pic)
				if err == nil || err.Error() != c.want {
					t.Errorf("%s: the add answered %v, want %q", c.name, err, c.want)
				}
				if lang == "ru" && err != nil && !cyrillic.MatchString(err.Error()) {
					t.Errorf("%s: the Russian refusal reads %q", c.name, err)
				}
			}
		})
	}
}

// The editor host files that return a refusal to an editor window hold no
// English literal in an error; their refusals are word keys.
func TestEditorSpeakerHostErrorsHoldNoLiteral(t *testing.T) {
	b, err := os.ReadFile("modspeakers.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	for i, line := range strings.Split(src, "\n") {
		if (strings.Contains(line, `errors.New("`) || strings.Contains(line, `fmt.Errorf("`)) && !strings.Contains(line, "install has no speaker records") {
			t.Errorf("modspeakers.go:%d holds an English literal error: %s", i+1, strings.TrimSpace(line))
		}
	}
}
