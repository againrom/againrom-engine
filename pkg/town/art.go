package town

import (
	"image"

	"againrom/pkg/screenart"
)

// Loader reads installed art for the composer: the one screen art loader's
// source.
type Loader = screenart.Source

// Art is one scene's resolved art: every entry's frames by name, the mask,
// and the problems of optional entries that did not load.
type Art struct {
	Frames   map[string][]image.Image
	Mask     *image.Paletted
	Problems []string
}

// Pictures answers an entry's frames, nil when it did not load.
func (a *Art) Pictures(name string) []image.Image {
	if a == nil {
		return nil
	}
	return a.Frames[name]
}

// LoadArt resolves every art entry of scene s in order, the square's and a
// room page's alike, through the one screen art loader. A required entry
// that fails fails the whole load with its key in the error; an optional
// entry that fails is left absent and named in Problems, and never removes
// another entry. The mask entry is always required: its size must be the
// view's and every required byte must occur in it. lib reads each key once
// for every screen of one install.
func LoadArt(s *SceneSpec, lib *screenart.Library) (*Art, error) {
	entries := make([]ArtSpec, len(s.Art))
	copy(entries, s.Art)
	for i := range entries {
		if entries[i].Format == "mask" {
			size := screenart.Point(s.View.Size)
			entries[i].Required, entries[i].Size, entries[i].RequiredBytes = true, &size, s.Mask.RequiredBytes
		}
	}
	set, err := lib.Load("town scene", entries)
	if err != nil {
		return nil, err
	}
	a := &Art{Frames: set.Frames, Problems: set.Problems}
	for _, e := range entries {
		if e.Format == "mask" {
			a.Mask = set.Mask(e.Name)
		}
	}
	return a, nil
}
