package ui

import (
	"image"

	"againrom/pkg/audio"
)

// SoundSliderPoint is a frame point on a channel's slider whose position
// selects the given percentage. Installed witnesses press it through the
// ordinary pointer path.
func SoundSliderPoint(channel audio.Channel, percent int) image.Point {
	track := soundSliderRect(channel)
	y := track.Min.Y + track.Dy()/2
	for x := track.Min.X; x < track.Max.X; x++ {
		if soundSliderPercent(soundSliderValue(channel, x)) == percent {
			return image.Pt(x, y)
		}
	}
	return image.Pt(track.Min.X, y)
}

// SoundTrackPoint is the middle of one visible row of the track list.
func SoundTrackPoint(row int) image.Point {
	r := soundOptionRect(gameMenuMusicTracks)
	return image.Pt(r.Min.X+r.Dx()/2, r.Min.Y+row*soundTrackRowH+soundTrackRowH/2)
}

// SoundButtonPoint is the middle of the Play, Stop or OK control.
func SoundButtonPoint(name string) image.Point {
	action := map[string]gameMenuAction{"play": gameMenuMusicPlay, "stop": gameMenuMusicStop, "ok": gameMenuPageReturn}[name]
	r := soundOptionRect(action)
	return r.Min.Add(r.Size().Div(2))
}
