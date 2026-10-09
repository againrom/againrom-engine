package ui

import (
	"image"

	"againrom/pkg/audio"
)

// SoundSliderPoint is a frame point on a channel's slider whose position
// selects the given percentage. Installed witnesses press it through the
// ordinary pointer path.
func SoundSliderPoint(channel audio.Channel, percent int) image.Point {
	s := hSlider{Rect: soundSliderRect(channel), Max: soundSliderRange}
	y := s.Rect.Min.Y + s.Rect.Dy()/2
	for x := s.left().Max.X; x < s.right().Min.X; x++ {
		if soundSliderPercent(s.posAt(x)) == percent {
			return image.Pt(x, y)
		}
	}
	return image.Pt(s.left().Max.X, y)
}

// SoundTrackPoint is the middle of one visible row of the track list.
func SoundTrackPoint(row int) image.Point {
	r := soundTrackBox(nil).Row(row)
	return r.Min.Add(r.Size().Div(2))
}

// SoundButtonPoint is the middle of the Play, Stop or OK control.
func SoundButtonPoint(name string) image.Point {
	action := map[string]gameMenuAction{"play": gameMenuMusicPlay, "stop": gameMenuMusicStop, "ok": gameMenuPageReturn}[name]
	r := soundOptionRect(action)
	return r.Min.Add(r.Size().Div(2))
}
