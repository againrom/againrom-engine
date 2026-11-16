package ui

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/video"
)

func TestCursorConfinementFocusGeometryAndExit(t *testing.T) {
	for _, movie := range []bool{false, true} {
		a := newTestApp(t, nil, nil)
		if movie {
			a.cutscene = &video.Player{}
		}
		focused := true
		r := image.Rect(-1280, 40, -160, 800)
		var calls []image.Rectangle
		a.pointerClip.read = func() (image.Rectangle, bool) { return r, focused }
		a.pointerClip.clip = func(r *image.Rectangle) bool {
			if r == nil {
				calls = append(calls, image.Rectangle{})
			} else {
				calls = append(calls, *r)
			}
			return true
		}
		a.applyPointerMode(false)
		if a.CursorInWindow() || len(calls) != 0 {
			t.Fatal("the default option touched cursor clipping")
		}
		a.SetCursorInWindow(true)
		a.applyPointerMode(false)
		want := []image.Rectangle{r}
		r = r.Add(image.Pt(1200, 50))
		a.applyPointerMode(false)
		want = append(want, r)
		r.Max = r.Max.Add(image.Pt(200, 100))
		a.applyPointerMode(false)
		want = append(want, r)
		focused = false
		a.applyPointerMode(false)
		a.applyPointerMode(false)
		want = append(want, image.Rectangle{})
		focused = true
		a.applyPointerMode(false)
		want = append(want, r)
		a.pointerClip.release()
		a.pointerClip.release()
		want = append(want, image.Rectangle{})
		if !reflect.DeepEqual(calls, want) {
			t.Fatalf("movie=%t: clip calls %v, want %v", movie, calls, want)
		}
	}
}

func TestCursorConfinementEmptyClientDisableAndFailedRelease(t *testing.T) {
	c := cursorConfinement{enabled: true}
	r := image.Rect(0, 0, 800, 600)
	c.read = func() (image.Rectangle, bool) { return r, true }
	clips, frees, failFree := 0, 0, false
	c.clip = func(r *image.Rectangle) bool {
		if r == nil {
			frees++
			return !failFree
		}
		clips++
		return true
	}
	c.apply()
	r = image.Rectangle{}
	c.apply()
	if clips != 1 || frees != 1 || c.clipped {
		t.Fatal("an empty/minimized client did not release the cursor")
	}
	r = image.Rect(10, 10, 810, 610)
	c.apply()
	c.enabled, failFree = false, true
	c.apply()
	if !c.clipped {
		t.Fatal("a failed release was forgotten")
	}
	failFree = false
	c.apply()
	c.apply()
	if clips != 2 || frees != 3 || c.clipped {
		t.Fatalf("disabled: clips=%d frees=%d clipped=%t", clips, frees, c.clipped)
	}
}
