package ui

import "image"

func (f *flow) initSoundTracks() {
	c := &f.soundOptions
	c.list = NewPicker(nil).SetWindow(soundTrackRows)
	if c.music == nil || c.music() == nil {
		return
	}
	var rows []PickerRow
	for _, name := range c.music().Candidates() {
		label := name
		if c.TrackTitle != nil {
			label = c.TrackTitle(name)
		}
		rows = append(rows, PickerRow{Text: label, Choosable: true})
	}
	c.list = NewPicker(rows).SetWindow(soundTrackRows)
}

func (f *flow) soundTrackAt(p image.Point) int {
	if f.soundOptions.list == nil {
		return -1
	}
	row, ok := soundTrackBox(f.menuFont).RowAt(p)
	top, count := f.soundOptions.list.Visible()
	if !ok || row >= count {
		return -1
	}
	return top + row
}

func (f *flow) focusMusicTracks() {
	for i, row := range f.menuRows() {
		if row.Action == gameMenuMusicTracks {
			f.menuList.Select(i)
			return
		}
	}
}

func (f *flow) chooseMusicOption(action gameMenuAction) {
	c := &f.soundOptions
	if c.ReadPlayback == nil || c.WritePlayback == nil {
		return
	}
	if action == gameMenuMusicTracks {
		return
	}
	p := c.ReadPlayback()
	switch action {
	case gameMenuMusicRandom:
		p.RandomOrder = !p.RandomOrder
	case gameMenuMusicPlay:
		p.Enabled = true
	case gameMenuMusicStop:
		p.Enabled = false
	}
	if err := c.WritePlayback(p); err != nil {
		f.msg = c.Words.NotSaved + err.Error()
		return
	}
	f.msg = ""
	if c.music == nil || c.music() == nil {
		return
	}
	m := c.music()
	m.SetPreferences(p)
	if action == gameMenuMusicPlay && c.list != nil && !m.Play(c.list.Selection()) {
		f.msg = c.Words.Unavailable
	}
}

// MusicPlaybackState is a detached observation for installed interaction
// witnesses. It does not expose mutable controller or decoder state.
func (a *App) MusicPlaybackState() (candidates []string, playing string) {
	if a == nil || a.music == nil {
		return nil, ""
	}
	return a.music.Candidates(), a.music.Playing()
}

// MusicRequestLog is a copy of the music request log for witnesses.
func (a *App) MusicRequestLog() []string {
	if a == nil || a.music == nil {
		return nil
	}
	return a.music.RequestLog()
}
