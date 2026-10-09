package town

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"
	"time"
)

// The tests build synthetic descriptions over generated art: every picture is
// a solid colour whose red channel names its entry and green its frame, so a
// painted pixel says which frame of which entry reached it.

type fakeLoader struct {
	sizes map[string]image.Point // key -> size; missing keys fail
	mask  *image.Paletted
}

func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return img
}

func keyColour(key string) color.RGBA {
	var r, g uint8
	for _, c := range key {
		r = r*31 + uint8(c)
	}
	if i := strings.LastIndexAny(key, "0123456789"); i >= 0 {
		g = key[i] - '0' + 1
	}
	return color.RGBA{R: r | 1, G: g, B: 7, A: 255}
}

func (l fakeLoader) Picture(key string) (image.Image, error) {
	size, ok := l.sizes[key]
	if !ok {
		return nil, fmt.Errorf("%s: missing", key)
	}
	return solid(size.X, size.Y, keyColour(key)), nil
}

func (l fakeLoader) Mask(key string) (*image.Paletted, error) {
	if l.mask == nil {
		return nil, fmt.Errorf("%s: no mask", key)
	}
	return l.mask, nil
}

func (l fakeLoader) Sprites(key string) []image.Image {
	size, ok := l.sizes[key]
	if !ok {
		return nil
	}
	n := size.Y / 100
	var out []image.Image
	for i := 0; i < n; i++ {
		out = append(out, solid(size.X, 4, color.RGBA{R: 9, G: uint8(i + 1), B: 7, A: 255}))
	}
	return out
}

type fakeVoice struct {
	h       *fakeHost
	key     string
	playing bool
}

func (v *fakeVoice) Playing() bool { return v.playing }
func (v *fakeVoice) Stop()         { v.playing = false }

type fakeHost struct {
	art        *Art
	now        time.Time
	draws      []int
	conditions map[string]bool
	log        []string
	voices     []*fakeVoice
}

func (h *fakeHost) Art() *Art      { return h.art }
func (h *fakeHost) Now() time.Time { return h.now }
func (h *fakeHost) Draw(source string, n int) int {
	if len(h.draws) == 0 {
		return 0
	}
	v := h.draws[0]
	h.draws = h.draws[1:]
	return v % n
}
func (h *fakeHost) Seed() int64                { return 1 }
func (h *fakeHost) Condition(name string) bool { return h.conditions[name] }
func (h *fakeHost) PlaySound(source, key string) Voice {
	h.log = append(h.log, "play "+source+":"+key)
	v := &fakeVoice{h: h, key: key, playing: true}
	h.voices = append(h.voices, v)
	return v
}
func (h *fakeHost) StopSound(v Voice) {
	h.log = append(h.log, "stop "+v.(*fakeVoice).key)
	v.Stop()
}
func (h *fakeHost) StartLoop(key string) bool { h.log = append(h.log, "loop "+key); return true }
func (h *fakeHost) StopLoop(key string)       { h.log = append(h.log, "stoploop "+key) }
func (h *fakeHost) LeaveSquare()              { h.log = append(h.log, "leave") }
func (h *fakeHost) Hook(name, room string)    { h.log = append(h.log, "hook "+name+"@"+room) }

func (h *fakeHost) take() string {
	out := strings.Join(h.log, ",")
	h.log = nil
	return out
}

// finish makes every voice the host started stop playing.
func (h *fakeHost) finish() {
	for _, v := range h.voices {
		v.playing = false
	}
}

func (h *fakeHost) tick(d time.Duration) { h.now = h.now.Add(d) }

var testVocabulary = Vocabulary{
	Hooks:      []string{"enter-hook", "leave-hook", "closed"},
	Conditions: []string{"open", "admitted"},
}

// decodeJSON decodes a description written as a Go value.
func decodeJSON(t *testing.T, v any) (*Description, error) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return Decode(data, testVocabulary)
}

func mustDecode(t *testing.T, v any) *Description {
	t.Helper()
	d, err := decodeJSON(t, v)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

type obj = map[string]any
type list = []any

// baseTown is a small valid square: a 40x30 view, a mask with two hotspots,
// one room and one episode actor.
func baseTown() obj {
	return obj{
		"town":   "test",
		"cite":   list{"owner"},
		"view":   obj{"size": list{40, 30}},
		"clock":  obj{"period-ms": 50, "compare": "greater"},
		"random": list{obj{"name": "host", "source": "host"}},
		"art": list{
			obj{"name": "base", "format": "picture", "key": "base.bmp", "size": list{40, 30}, "required": true},
			obj{"name": "mask", "format": "mask", "key": "mask.bmp", "required": true},
			obj{"name": "sprite", "format": "sprites", "key": "sprite.16a", "count": 3},
		},
		"mask": obj{"art": "mask", "required-bytes": list{1, 2}, "bytes": list{
			obj{"byte": 1, "hotspot": "door"},
			obj{"byte": 2, "hotspot": "gate"},
		}},
		"hotspots": list{
			obj{"name": "door", "row": 0, "tip": 3, "click": list{obj{"room": "inn"}}, "hover": list{obj{"arm": "spin"}}},
			obj{"name": "gate", "row": 1, "tip": 4, "click": list{obj{"if": "open", "then": list{obj{"menu": "m"}}, "else": list{obj{"hook": "closed"}}}}},
		},
		"sounds": obj{"slots": list{obj{"name": "a", "source": "s"}}},
		"actors": list{obj{"name": "spin", "program": "episode", "art": "sprite",
			"start-sound": obj{"slot": "a", "key": "spin.wav"}}},
		"step":   obj{"admitted": list{"advance spin"}},
		"layers": list{obj{"art": "base", "mode": "copy", "at": list{0, 0}}, obj{"actor": "spin", "mode": "over", "at": list{2, 3}}},
		"tip":    obj{"text": "tips.txt", "rect": list{0, 0, 10, 10}},
		"music":  obj{"track": "t.wav"},
		"square": obj{"name": "square", "enter": list{obj{"composer": "reset"}, obj{"hook": "leave-hook"}}},
		"rooms": list{obj{"name": "inn", "page": "inn",
			"enter": list{obj{"composer": "reset"}, obj{"hook": "enter-hook"}},
			"exit":  list{obj{"hook": "leave-hook"}}}},
		"save": obj{"admitted-when": "admitted"},
	}
}

// testMask is a 40x30 mask: byte 1 in the left half, byte 2 in the right.
func testMask() *image.Paletted {
	pal := make(color.Palette, 256)
	for i := range pal {
		pal[i] = color.Gray{Y: uint8(i)}
	}
	m := image.NewPaletted(image.Rect(0, 0, 40, 30), pal)
	for y := 0; y < 30; y++ {
		for x := 0; x < 40; x++ {
			if x < 20 {
				m.SetColorIndex(x, y, 1)
			} else {
				m.SetColorIndex(x, y, 2)
			}
		}
	}
	return m
}

func testLoader() fakeLoader {
	return fakeLoader{
		sizes: map[string]image.Point{
			"base.bmp":   {40, 30},
			"sprite.16a": {5, 300},
		},
		mask: testMask(),
	}
}
