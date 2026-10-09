package town

import (
	"fmt"
	"image"
	"image/draw"
)

// Loader reads installed art for the composer. Picture and Mask return an
// error that names the key; Sprites returns nil for a sheet it cannot read.
type Loader interface {
	Picture(key string) (image.Image, error)
	Mask(key string) (*image.Paletted, error)
	Sprites(key string) []image.Image
}

// Art is one description's resolved art: every entry's frames by name, the
// mask, and the problems of optional entries that did not load.
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

// LoadArt resolves every art entry of d in order. A required entry that fails
// fails the whole load with its key in the error; an optional entry that fails
// is left absent and named in Problems, and never removes another entry. The
// mask entry is always required.
func LoadArt(d *Description, src Loader) (*Art, error) {
	a := &Art{Frames: map[string][]image.Image{}}
	for _, spec := range d.Art {
		if spec.Format == "mask" {
			mask, err := loadMask(spec, d, src)
			if err != nil {
				return nil, err
			}
			a.Mask = mask
			continue
		}
		for _, entry := range expandArt(spec) {
			frames, err := loadEntry(spec, entry, src)
			if err != nil {
				if spec.Required {
					return nil, err
				}
				a.Problems = append(a.Problems, err.Error())
				continue
			}
			a.Frames[entry.name] = frames
		}
	}
	return a, nil
}

type artEntry struct {
	name, key string
	count     int
}

// loadMask reads the mask entry: its size must be the view's and every
// required byte must occur in it.
func loadMask(spec ArtSpec, d *Description, src Loader) (*image.Paletted, error) {
	mask, err := src.Mask(spec.Key)
	if err != nil {
		return nil, err
	}
	if err = checkSize(mask, d.View.Size, spec.Key); err != nil {
		return nil, err
	}
	seen := map[int]bool{}
	for _, index := range mask.Pix {
		seen[int(index)] = true
	}
	for _, b := range d.Mask.RequiredBytes {
		if !seen[b] {
			return nil, fmt.Errorf("%s: missing required mask index %d", spec.Key, b)
		}
	}
	return mask, nil
}

// expandArt lists the entries one spec names: itself, or one entry per index
// combination.
func expandArt(spec ArtSpec) []artEntry {
	switch len(spec.Index) {
	case 0:
		return []artEntry{{name: spec.Name, key: spec.Key, count: spec.Count}}
	case 1:
		var out []artEntry
		for i := 0; i < spec.Index[0].Count; i++ {
			out = append(out, artEntry{
				name:  fmt.Sprintf("%s/%d", spec.Name, i),
				key:   fmt.Sprintf(spec.Key, i+spec.Index[0].Base),
				count: spec.countAt(i),
			})
		}
		return out
	default:
		var out []artEntry
		for i := 0; i < spec.Index[0].Count; i++ {
			for j := 0; j < spec.Index[1].Count; j++ {
				out = append(out, artEntry{
					name:  fmt.Sprintf("%s/%d/%d", spec.Name, i, j),
					key:   fmt.Sprintf(spec.Key, i+spec.Index[0].Base, j+spec.Index[1].Base),
					count: spec.countAt(j),
				})
			}
		}
		return out
	}
}

// countAt is the frame count of the entry whose last index is i: Counts by
// that index when given, else Count.
func (s ArtSpec) countAt(i int) int {
	if i >= 0 && i < len(s.Counts) {
		return s.Counts[i]
	}
	return s.Count
}

func loadEntry(spec ArtSpec, e artEntry, src Loader) ([]image.Image, error) {
	switch spec.Format {
	case "picture":
		pic, err := src.Picture(e.key)
		if err != nil {
			return nil, err
		}
		if spec.Size != nil {
			if err = checkSize(pic, *spec.Size, e.key); err != nil {
				return nil, err
			}
		}
		if spec.Transparent == "black" {
			pic = keyBlack(pic)
		}
		return []image.Image{pic}, nil
	case "series":
		var frames []image.Image
		for i := 0; i < e.count; i++ {
			key := fmt.Sprintf(e.key, i+spec.First)
			pic, err := src.Picture(key)
			if err == nil && spec.Size != nil {
				err = checkSize(pic, *spec.Size, key)
			}
			if err == nil && spec.MaxSize != nil {
				err = checkMaxSize(pic, *spec.MaxSize, key)
			}
			if err != nil && spec.KeepMissing {
				frames = append(frames, nil)
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("%s: %v", key, err)
			}
			if spec.Transparent == "black" {
				pic = keyBlack(pic)
			}
			frames = append(frames, pic)
		}
		return frames, nil
	case "sprites":
		frames := src.Sprites(e.key)
		valid := len(frames) == e.count
		for _, f := range frames {
			b := f.Bounds()
			valid = valid && b.Dx() > 0 && b.Dy() > 0
			if spec.MaxSize != nil {
				valid = valid && b.Dx() <= spec.MaxSize[0] && b.Dy() <= spec.MaxSize[1]
			}
		}
		if !valid {
			return nil, fmt.Errorf("%s: expected %d readable frames", e.key, e.count)
		}
		return frames, nil
	}
	return nil, fmt.Errorf("%s: unknown art format %q", e.key, spec.Format)
}

func checkSize(pic image.Image, want Point, key string) error {
	if pic == nil {
		return fmt.Errorf("%s: missing image", key)
	}
	if pic.Bounds().Dx() != want[0] || pic.Bounds().Dy() != want[1] {
		return fmt.Errorf("%s: size %dx%d, want %dx%d", key, pic.Bounds().Dx(), pic.Bounds().Dy(), want[0], want[1])
	}
	return nil
}

// checkMaxSize admits a picture with a positive size inside the envelope.
func checkMaxSize(pic image.Image, envelope Point, key string) error {
	b := pic.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 || b.Dx() > envelope[0] || b.Dy() > envelope[1] {
		return fmt.Errorf("%s: invalid bounds %v", key, b)
	}
	return nil
}

// keyBlack makes every pure-black pixel transparent.
func keyBlack(pic image.Image) image.Image {
	rgba, ok := pic.(*image.RGBA)
	if !ok {
		rgba = image.NewRGBA(pic.Bounds())
		draw.Draw(rgba, rgba.Bounds(), pic, pic.Bounds().Min, draw.Src)
	}
	for i := 0; i+3 < len(rgba.Pix); i += 4 {
		if rgba.Pix[i] == 0 && rgba.Pix[i+1] == 0 && rgba.Pix[i+2] == 0 {
			rgba.Pix[i+3] = 0
		}
	}
	return rgba
}
