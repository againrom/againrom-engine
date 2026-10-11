package screenart

import (
	"fmt"
	"image"
	"image/draw"
	"reflect"
	"sync"
)

// Source decodes an install's art. Picture and Mask return an error that
// names the key; Sprites returns nil for a sheet it cannot read or whose
// frames are not all of a positive size.
type Source interface {
	Picture(key string) (image.Image, error)
	Mask(key string) (*image.Paletted, error)
	Sprites(key string) []image.Image
}

// Set is one screen's resolved art: every entry's frames by name, the masks
// by name, and the problems of optional entries that did not load.
type Set struct {
	Frames   map[string][]image.Image
	Masks    map[string]*image.Paletted
	Problems []string
	// numbers maps an indexed entry's name to its members' printed numbers.
	numbers map[string]map[int]string
}

// Pictures answers an entry's frames, nil when it did not load.
func (s *Set) Pictures(name string) []image.Image {
	if s == nil {
		return nil
	}
	return s.Frames[name]
}

// Picture answers an entry's first frame, nil when it did not load.
func (s *Set) Picture(name string) image.Image {
	if f := s.Pictures(name); len(f) > 0 {
		return f[0]
	}
	return nil
}

// Mask answers a mask entry, nil when it did not load.
func (s *Set) Mask(name string) *image.Paletted {
	if s == nil {
		return nil
	}
	return s.Masks[name]
}

// Numbered answers the loaded members of a one-index entry by the number
// each member's key is printed with.
func (s *Set) Numbered(name string) map[int][]image.Image {
	out := map[int][]image.Image{}
	if s == nil {
		return out
	}
	for n, member := range s.numbers[name] {
		if f, ok := s.Frames[member]; ok {
			out[n] = f
		}
	}
	return out
}

// Library reads pictures for every screen of one install. A key is read and
// validated once, by its rule: its format, its size or size envelope and its
// key colour. A later entry naming the same key under another rule is an
// error that names both screens, so one picture is held to one rule
// whichever screens draw it. A Library is safe for concurrent use.
type Library struct {
	src   Source
	mu    sync.Mutex
	memo  map[string]*memo
	reads int
}

type memo struct {
	rule   rule
	screen string
	key    string
	frames []image.Image
	mask   *image.Paletted
	err    error
}

// rule is what one key is validated by.
type rule struct {
	Format      string
	Size        *Point
	MaxSize     *Point
	Transparent string
}

func (r rule) same(o rule) bool { return reflect.DeepEqual(r, o) }

func (r rule) String() string {
	s := r.Format
	if r.Size != nil {
		s += fmt.Sprintf(" size %dx%d", r.Size[0], r.Size[1])
	}
	if r.MaxSize != nil {
		s += fmt.Sprintf(" within %dx%d", r.MaxSize[0], r.MaxSize[1])
	}
	if r.Transparent != "" {
		s += " transparent " + r.Transparent
	}
	return s
}

// NewLibrary answers a library over src.
func NewLibrary(src Source) *Library { return &Library{src: src, memo: map[string]*memo{}} }

// Load reads entries for screen with a library of its own.
func Load(screen string, entries []Entry, src Source) (*Set, error) {
	return NewLibrary(src).Load(screen, entries)
}

// Reads is the number of keys the library has read from its source.
func (l *Library) Reads() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.reads
}

// Reread reads into l every key from has read, under the same rule and for
// the same screen. A library over a fresh read of the install then holds
// what from holds unless a screen wrote a picture from handed out.
func (l *Library) Reread(from *Library) {
	from.mu.Lock()
	held := make([]memo, 0, len(from.memo))
	for _, m := range from.memo {
		held = append(held, *m)
	}
	from.mu.Unlock()
	for _, m := range held {
		spec := Entry{Format: m.rule.Format, Size: m.rule.Size, MaxSize: m.rule.MaxSize, Transparent: m.rule.Transparent}
		switch m.rule.Format {
		case "picture":
			l.picture(m.screen, spec, m.key)
		case "mask":
			l.mask(m.screen, spec, m.key)
		default:
			l.sheet(m.screen, spec, m.key)
		}
	}
}

// Load resolves entries in order for screen. A required entry that fails
// fails the whole load with its key in the error; an optional entry that
// fails is left absent and named in Problems, and never removes another
// entry.
func (l *Library) Load(screen string, entries []Entry) (*Set, error) {
	set := &Set{Frames: map[string][]image.Image{}, Masks: map[string]*image.Paletted{}, numbers: map[string]map[int]string{}}
	for _, spec := range entries {
		if spec.Bound {
			continue
		}
		for _, e := range expand(spec) {
			if e.number != nil {
				if set.numbers[spec.Name] == nil {
					set.numbers[spec.Name] = map[int]string{}
				}
				set.numbers[spec.Name][*e.number] = e.name
			}
			if spec.Format == "mask" {
				mask, err := l.mask(screen, spec, e.key)
				if err != nil {
					if spec.Required {
						return nil, err
					}
					set.Problems = append(set.Problems, err.Error())
					continue
				}
				set.Masks[e.name] = mask
				continue
			}
			frames, err := l.entry(screen, spec, e)
			if err != nil {
				if spec.Required {
					return nil, err
				}
				set.Problems = append(set.Problems, err.Error())
				continue
			}
			set.Frames[e.name] = frames
		}
	}
	return set, nil
}

// Bind loads the bound entry name of entries for screen, its key formatted
// with args, under the entry's rule. It answers the entry's frames, or the
// error that names the formatted key.
func (l *Library) Bind(screen string, entries []Entry, name string, args ...any) ([]image.Image, error) {
	for _, spec := range entries {
		if spec.Name != name || !spec.Bound {
			continue
		}
		spec.Key, spec.Bound, spec.Required = fmt.Sprintf(spec.Key, args...), false, true
		set, err := l.Load(screen, []Entry{spec})
		if err != nil {
			return nil, err
		}
		return set.Pictures(name), nil
	}
	return nil, fmt.Errorf("screen %s names no bound art %q", screen, name)
}

type expanded struct {
	name, key    string
	count, first int
	number       *int
}

// expand lists the entries one spec names: itself, or one entry per index
// combination.
func expand(spec Entry) []expanded {
	switch len(spec.Index) {
	case 0:
		return []expanded{{name: spec.Name, key: spec.Key, count: spec.Count, first: spec.First}}
	case 1:
		var out []expanded
		x := spec.Index[0]
		for i := 0; i < x.size(); i++ {
			n := x.number(i)
			out = append(out, expanded{
				name:   fmt.Sprintf("%s/%d", spec.Name, i),
				key:    fmt.Sprintf(spec.Key, n),
				count:  spec.CountAt(i),
				first:  spec.FirstAt(i),
				number: &n,
			})
		}
		return out
	default:
		var out []expanded
		x, y := spec.Index[0], spec.Index[1]
		for i := 0; i < x.size(); i++ {
			for j := 0; j < y.size(); j++ {
				out = append(out, expanded{
					name:  fmt.Sprintf("%s/%d/%d", spec.Name, i, j),
					key:   fmt.Sprintf(spec.Key, x.number(i), y.number(j)),
					count: spec.CountAt(j),
					first: spec.FirstAt(j),
				})
			}
		}
		return out
	}
}

// Keys lists every install key the entry reads, a series member by member;
// a bound entry reads none until it is bound.
func (s Entry) Keys() []string {
	if s.Bound {
		return nil
	}
	var out []string
	for _, e := range expand(s) {
		if s.Format != "series" {
			out = append(out, e.key)
			continue
		}
		for i := 0; i < e.count; i++ {
			out = append(out, fmt.Sprintf(e.key, i+e.first))
		}
	}
	return out
}

// Rule describes the rule the entry reads its keys under; two entries
// naming one key must describe the same rule.
func (s Entry) Rule() string { return s.rule().String() }

func (s Entry) rule() rule {
	switch s.Format {
	case "picture", "series":
		return rule{Format: "picture", Size: s.Size, MaxSize: s.MaxSize, Transparent: s.Transparent}
	case "mask":
		return rule{Format: "mask", Size: s.Size}
	}
	return rule{Format: "sheet"}
}

// CountAt is the frame count of the entry whose last index is i: Counts by
// that index when given, else Count.
func (s Entry) CountAt(i int) int {
	if i >= 0 && i < len(s.Counts) {
		return s.Counts[i]
	}
	return s.Count
}

// FirstAt is the first series number of the entry whose last index is i:
// Firsts by that index when given, else First.
func (s Entry) FirstAt(i int) int {
	if i >= 0 && i < len(s.Firsts) {
		return s.Firsts[i]
	}
	return s.First
}

func (l *Library) entry(screen string, spec Entry, e expanded) ([]image.Image, error) {
	switch spec.Format {
	case "picture":
		pic, err := l.picture(screen, spec, e.key)
		if err != nil {
			return nil, err
		}
		if pic, err = place(pic, spec, e.key); err != nil {
			return nil, err
		}
		return []image.Image{pic}, nil
	case "series":
		var frames []image.Image
		for i := 0; i < e.count; i++ {
			key := fmt.Sprintf(e.key, i+e.first)
			pic, err := l.picture(screen, spec, key)
			if err == nil {
				pic, err = place(pic, spec, key)
			}
			if err != nil && spec.KeepMissing {
				frames = append(frames, nil)
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("%s: %v", key, err)
			}
			frames = append(frames, pic)
		}
		return frames, nil
	case "sprites":
		frames, err := l.sheet(screen, spec, e.key)
		if err != nil {
			return nil, err
		}
		valid := len(frames) > 0 && (e.count == 0 || len(frames) == e.count)
		for _, f := range frames {
			b := f.Bounds()
			if spec.MaxSize != nil {
				valid = valid && b.Dx() <= spec.MaxSize[0] && b.Dy() <= spec.MaxSize[1]
			}
		}
		if !valid {
			if e.count == 0 {
				return nil, fmt.Errorf("%s: no readable frames", e.key)
			}
			return nil, fmt.Errorf("%s: expected %d readable frames", e.key, e.count)
		}
		return frames, nil
	case "frame":
		frames, err := l.sheet(screen, spec, e.key)
		if err != nil {
			return nil, err
		}
		if spec.Frame < 0 || spec.Frame >= len(frames) {
			return nil, fmt.Errorf("%s: frame %d, sheet carries %d", e.key, spec.Frame, len(frames))
		}
		pic := frames[spec.Frame]
		if spec.Size != nil {
			if err = checkSize(pic, *spec.Size, e.key); err != nil {
				return nil, err
			}
		}
		return []image.Image{pic}, nil
	}
	return nil, fmt.Errorf("%s: unknown art format %q", e.key, spec.Format)
}

// cached answers the memo of key under r, reading it with read on first
// use. A memo held under another rule is an error naming both screens.
func (l *Library) cached(screen, key string, r rule, read func() *memo) *memo {
	l.mu.Lock()
	defer l.mu.Unlock()
	id := r.Format + "|" + key
	if m, ok := l.memo[id]; ok {
		if !m.rule.same(r) {
			return &memo{err: fmt.Errorf("%s: screen %s reads it as %s, screen %s as %s", key, m.screen, m.rule, screen, r)}
		}
		return m
	}
	l.reads++
	m := read()
	m.rule, m.screen, m.key = r, screen, key
	l.memo[id] = m
	return m
}

// picture reads one bitmap under its rule: its size or envelope, then its
// key colour.
func (l *Library) picture(screen string, spec Entry, key string) (image.Image, error) {
	m := l.cached(screen, key, spec.rule(), func() *memo {
		pic, err := l.src.Picture(key)
		if err == nil && spec.Size != nil {
			err = checkSize(pic, *spec.Size, key)
		}
		if err == nil && spec.MaxSize != nil {
			err = checkMaxSize(pic, *spec.MaxSize, key)
		}
		if err != nil {
			return &memo{err: err}
		}
		if spec.Transparent == "black" {
			pic = keyBlack(pic)
		}
		return &memo{frames: []image.Image{pic}}
	})
	if m.err != nil {
		return nil, m.err
	}
	return m.frames[0], nil
}

// sheet reads every frame of one sprite sheet.
func (l *Library) sheet(screen string, spec Entry, key string) ([]image.Image, error) {
	m := l.cached(screen, key, rule{Format: "sheet"}, func() *memo {
		frames := l.src.Sprites(key)
		if len(frames) == 0 {
			return &memo{err: fmt.Errorf("%s: no readable frames", key)}
		}
		return &memo{frames: frames}
	})
	if m.err != nil && spec.Format == "sprites" && spec.Count > 0 {
		return nil, fmt.Errorf("%s: expected %d readable frames", key, spec.Count)
	}
	return m.frames, m.err
}

// mask reads one indexed bitmap: its size when stated, and every required
// index must occur in it.
func (l *Library) mask(screen string, spec Entry, key string) (*image.Paletted, error) {
	m := l.cached(screen, key, spec.rule(), func() *memo {
		mask, err := l.src.Mask(key)
		if err == nil && spec.Size != nil {
			err = checkSize(mask, *spec.Size, key)
		}
		if err != nil {
			return &memo{err: err}
		}
		return &memo{mask: mask}
	})
	if m.err != nil {
		return nil, m.err
	}
	pix := m.mask.Pix
	if len(spec.RequiredBytes) != 0 {
		var seen [1 << 8]bool
		for _, index := range pix {
			seen[index] = true
		}
		for _, b := range spec.RequiredBytes {
			if b < 0 || b >= len(seen) || !seen[b] {
				return nil, fmt.Errorf("%s: missing required mask index %d", key, b)
			}
		}
	}
	return m.mask, nil
}

// place applies an entry's own use of a picture: that it lies inside its
// rectangle when drawn at its origin, then its crop.
func place(pic image.Image, spec Entry, key string) (image.Image, error) {
	if spec.Inside != nil {
		var at image.Point
		if spec.At != nil {
			at = image.Pt(spec.At[0], spec.At[1])
		}
		if bounds := spec.Inside.Rectangle(); !pic.Bounds().Add(at).In(bounds) {
			return nil, fmt.Errorf("%s: patch %v at %v lies outside %v", key, pic.Bounds(), at, bounds)
		}
	}
	if spec.Crop != nil {
		r := spec.Crop.Rectangle()
		out := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
		draw.Draw(out, out.Bounds(), pic, r.Min, draw.Src)
		return out, nil
	}
	return pic, nil
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
