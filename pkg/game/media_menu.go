package game

import (
	"fmt"
	"image"
	"strings"

	"againrom/pkg/ui"
)

const seenMoviesKey = "EncounteredCutscenes"

// The two installed tables are positional peers (TEXT-NAMETAB-026). A group
// is intro, newgame, start or a canonical mission directory, never a file path.
func validMovieDirectory(s string) bool {
	if s == "intro" || s == "newgame" || s == "start" {
		return true
	}
	if len(s) < 2 || len(s) > 5 || s[0] != 'm' {
		return false
	}
	for _, c := range s[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s[1] != '0'
}

func (s OptionsStore) EncounteredCutscenes() ([]string, error) {
	m, err := s.readAll()
	if err != nil {
		return nil, err
	}
	var out []string
	seen := map[string]bool{}
	for _, key := range strings.Split(m[seenMoviesKey], ",") {
		if validMovieDirectory(key) && !seen[key] {
			out = append(out, key)
			seen[key] = true
		}
	}
	return out, nil
}

func (s OptionsStore) EncounterCutscene(key string) error {
	if !validMovieDirectory(key) {
		return fmt.Errorf("invalid movie group %q", key)
	}
	m, err := s.readAll()
	if err != nil {
		return err
	}
	for _, existing := range strings.Split(m[seenMoviesKey], ",") {
		if existing == key {
			return nil
		}
	}
	if m[seenMoviesKey] != "" {
		m[seenMoviesKey] += ","
	}
	m[seenMoviesKey] += key
	return s.writeAll(m)
}

func (f *FrontEnd) movieCatalog() []ui.CutsceneEntry {
	if f.Archives == nil || f.Archives.Containers == nil {
		return nil
	}
	paths := LoadTextTable(f.Archives.Containers, mainPrefix+"text/cutpaths.txt")
	titles := LoadTextTable(f.Archives.Containers, mainPrefix+"text/cutscene.txt")
	var out []ui.CutsceneEntry
	for i := 0; i < 99; i++ {
		path, ok := paths.At(i)
		if !ok {
			break
		}
		title, ok := titles.At(i)
		path = strings.ToLower(strings.TrimSpace(path))
		if ok && title != "" && validMovieDirectory(path) {
			out = append(out, ui.CutsceneEntry{Directory: path, Title: title})
		}
	}
	return out
}

func (f *FrontEnd) wireMediaMenu(a *ui.App) {
	words := ui.CutsceneLibraryWords{Title: "View Cutscenes", OK: "OK", Cancel: "Cancel"}
	if f.Archives != nil && f.Archives.Containers != nil {
		t := LoadTextTable(f.Archives.Containers, DialogsTextPath)
		for i, p := range map[int]*string{153: &words.Title, 0: &words.OK, 1: &words.Cancel} {
			if s, ok := t.At(i); ok {
				*p = s
			}
		}
	}
	seen, _ := f.Options.EncounteredCutscenes()
	a.SetCutsceneLibrary(f.movieCatalog(), seen, f.Options.EncounterCutscene, words)
	if f.Archives != nil && f.Archives.Containers != nil {
		const path = graphicsPrefix + "interface/scrlbars.256"
		if raw, err := f.Archives.Containers.ReadFile(path); err == nil {
			if frames, err := decodeCursor256Frames(path, raw); err == nil {
				a.SetCutsceneScrollArt(frames)
			}
		}
	}
	a.SetCredits(f.creditsView)
}

func (f *FrontEnd) creditsView() ui.CreditsView {
	v := ui.CreditsView{Logos: map[string]*image.RGBA{}}
	if f.Archives == nil || f.Archives.Containers == nil {
		return v
	}
	// The preserved credits nodes use LF internally, unlike the positional
	// interface tables. Keep empty lines: they reserve space for logos.
	if raw, err := f.Archives.Containers.ReadFile(mainPrefix + "text/credits.txt"); err == nil && len(raw) <= 256*1024 {
		v.Lines = strings.Split(strings.TrimSuffix(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n"), "\n")
		if len(v.Lines) > 4096 {
			v.Lines = v.Lines[:4096]
		}
	}
	for _, name := range []string{"nival", "buka", "monolith", "1c"} {
		if pic, err := readChargenBMP(f.Archives.Containers, graphicsPrefix+"interface/logo/"+name+".bmp"); err == nil {
			v.Logos[name] = pic
		}
	}
	return v
}
