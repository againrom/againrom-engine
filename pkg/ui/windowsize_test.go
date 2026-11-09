package ui

import (
	"image"
	"testing"
)

func TestParseWindowSize(t *testing.T) {
	def := image.Pt(DefaultWindowW, DefaultWindowH)
	for _, c := range []struct {
		value string
		empty image.Point
		want  image.Point
	}{
		{"640x480", image.Point{}, image.Pt(640, 480)},
		{"1024x768", def, image.Pt(1024, 768)},
		{"8192x8192", image.Point{}, image.Pt(8192, 8192)},
		{"", def, def},
	} {
		if got, err := ParseWindowSize(c.value, c.empty); err != nil || got != c.want {
			t.Errorf("ParseWindowSize(%q) = %v, %v; want %v", c.value, got, err, c.want)
		}
	}
	for _, value := range []string{"", "640", "0x480", "640x-1", "640x480x2", "oops", "99999999999999999999x1", "8193x1", "1x8193", "640x480junk"} {
		if got, err := ParseWindowSize(value, image.Point{}); err == nil {
			t.Errorf("ParseWindowSize(%q) = %v, accepted", value, got)
		}
	}
}
