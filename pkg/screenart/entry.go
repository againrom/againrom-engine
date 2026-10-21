// Package screenart is the one screen art loader. A screen names its art as
// data, a manifest of entries, and Load answers the decoded pictures by entry
// name. Each entry states a picture's install key, its format, the size or
// size rule it must meet and its key colour; a screen states whether the
// entry is required. The package holds no fact of any game: no key, size or
// colour appears in its code. A game supplies the manifests and a Source that
// decodes its install's formats.
package screenart

import (
	"fmt"
	"image"
	"strings"
)

// Point is a width and height, or an x and y.
type Point [2]int

// Rect is a rectangle as x0, y0, x1, y1.
type Rect [4]int

// Rectangle is r as an image.Rectangle.
func (r Rect) Rectangle() image.Rectangle { return image.Rect(r[0], r[1], r[2], r[3]) }

// Entry is one named art entry. Format "picture" is one bitmap; "series" is
// numbered bitmaps loaded as one atomic family; "sprites" is every frame of a
// sprite sheet; "frame" is one frame of a sprite sheet; "mask" is one indexed
// bitmap read as its palette indices. Index expands the key over one or two
// zero-based indices, each printed with its base added, or with its values;
// each expansion is its own entry named name/i or name/i/j.
type Entry struct {
	Name   string  `json:"name"`
	Format string  `json:"format"`
	Key    string  `json:"key"`
	Index  []Index `json:"index"`
	Count  int     `json:"count"`
	Counts []int   `json:"counts"`
	// First is the number the first member of a series is printed with;
	// Firsts gives it by the entry's last index, as Counts gives the count.
	First  int   `json:"first"`
	Firsts []int `json:"firsts"`
	// Frame is the frame a "frame" entry takes from its sheet.
	Frame   int    `json:"frame"`
	Size    *Point `json:"size"`
	MaxSize *Point `json:"max-size"`
	// Required entries fail the whole load; an optional entry that fails is
	// left absent and named in the set's Problems.
	Required bool `json:"required"`
	// Transparent "black" makes every pure-black pixel transparent.
	Transparent string `json:"transparent"`
	// KeepMissing loads a series member by member: a member that does not
	// load is a nil frame, and the series is kept.
	KeepMissing bool `json:"keep-missing"`
	// RequiredBytes are the indices a mask must hold.
	RequiredBytes []int `json:"required-bytes"`
	// At and Inside place a picture: drawn at At it must lie inside Inside.
	At     *Point `json:"at"`
	Inside *Rect  `json:"inside"`
	// Crop takes the entry as this rectangle of the picture, copied.
	Crop *Rect `json:"crop"`
	// Bound entries name a picture the screen learns at run time: Load
	// skips them, and Bind loads one with its key formatted.
	Bound bool     `json:"bound"`
	Cite  []string `json:"cite"`
}

// Index is one dimension of an indexed key: Count numbers from Base, or the
// listed Values.
type Index struct {
	Count  int      `json:"count"`
	Base   int      `json:"base"`
	Values []int    `json:"values"`
	Cite   []string `json:"cite"`
}

// size is the index's member count.
func (x Index) size() int {
	if len(x.Values) > 0 {
		return len(x.Values)
	}
	return x.Count
}

// number is the number member i is printed with.
func (x Index) number(i int) int {
	if len(x.Values) > 0 {
		return x.Values[i]
	}
	return i + x.Base
}

// Formats are the entry formats Load reads.
var Formats = []string{"picture", "series", "sprites", "frame", "mask"}

// Check reports the first structural fault of entries: an unknown format, an
// entry without name or key, more than two indices, or a name used twice.
func Check(entries []Entry) error {
	names := map[string]bool{}
	for _, e := range entries {
		known := false
		for _, f := range Formats {
			known = known || e.Format == f
		}
		if !known {
			return fmt.Errorf("art %q has unknown format %q; accepted: %s", e.Name, e.Format, strings.Join(Formats, ", "))
		}
		if e.Name == "" || e.Key == "" {
			return fmt.Errorf("art entry without name or key")
		}
		if len(e.Index) > 2 {
			return fmt.Errorf("art %q has %d indices", e.Name, len(e.Index))
		}
		if names[e.Name] {
			return fmt.Errorf("art %q is named twice", e.Name)
		}
		names[e.Name] = true
	}
	return nil
}
