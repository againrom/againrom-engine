package game

import (
	"strconv"
	"strings"

	"againrom/pkg/ui"
)

func (s OptionsStore) MusicPreferences() (ui.MusicPreferences, error) {
	p := ui.DefaultMusicPreferences()
	values, err := s.readAll()
	if err != nil {
		return p, err
	}
	for key, field := range map[string]*bool{"MusicEnabled": &p.Enabled, "RandomOrder": &p.RandomOrder} {
		if value, err := strconv.Atoi(values[key]); err == nil {
			*field = value != 0
		}
	}
	return p, nil
}

func (s OptionsStore) SetMusicPreferences(p ui.MusicPreferences) error {
	values, err := s.readAll()
	if err != nil {
		return err
	}
	values["MusicEnabled"] = strconv.Itoa(boolOption(p.Enabled))
	values["RandomOrder"] = strconv.Itoa(boolOption(p.RandomOrder))
	return s.writeAll(values)
}

func musicTitles(table *TextTable) map[string]string {
	titles := map[string]string{}
	for i := 0; i < table.Lines(); i++ {
		line, _ := table.At(i)
		key, value, found := strings.Cut(line, "=")
		if found && key != "" && value != "" {
			titles[strings.ToLower(key)] = value
		}
	}
	return titles
}

func (f *FrontEnd) wireMusicPreferences(c *ui.SoundOptionControls) {
	p, _ := f.Options.MusicPreferences()
	c.ReadPlayback = func() ui.MusicPreferences { return p }
	c.WritePlayback = func(next ui.MusicPreferences) error {
		if f.Options.Path != "" {
			if err := f.Options.SetMusicPreferences(next); err != nil {
				return err
			}
		}
		p = next
		return nil
	}
	var table *TextTable
	if f.Archives != nil {
		table = LoadTextTable(f.Archives.Containers, mainPrefix+"text/tunes.txt", f.textCode())
	}
	titles := musicTitles(table)
	c.TrackTitle = func(name string) string {
		if title := titles[strings.ToLower(name)]; title != "" {
			return title
		}
		return name
	}
}
