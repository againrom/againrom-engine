package game

import (
	"fmt"
	"image"
	"strings"
	"time"

	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// The square's composer state, read by name for tests.

func (t *townScreen) sqEpisode(name string) *town.Episode {
	return t.squareView().Actor(name).(*town.Episode)
}

func (t *townScreen) sqSchool() *town.Pendulum {
	return t.squareView().Actor("school").(*town.Pendulum)
}

func (t *townScreen) sqDoor() *town.Stepper { return t.squareView().Actor("door").(*town.Stepper) }

func (t *townScreen) sqGuard() *town.Driven { return t.squareView().Actor("guard").(*town.Driven) }

func (t *townScreen) sqBirds() *town.Flock { return t.squareView().Actor("birds").(*town.Flock) }

func (t *townScreen) sqWildlife() *town.Families {
	return t.squareView().Actor("wildlife").(*town.Families)
}

// sqHover is the hovered hotspot's name, "" for none.
func (t *townScreen) sqHover() string {
	if h := t.squareView().Hover(); h != nil {
		return h.Name
	}
	return ""
}

// sqPaintLast is the paint clock's last admitted step.
func (t *townScreen) sqPaintLast() time.Time { return t.squareView().ClockLast() }

// sqVoice is a sound slot's retained voice.
func (t *townScreen) sqVoice(slot string) town.Voice { return t.squareView().Voice(slot) }

// squareControlAt is the production square scene's hit test over f.
func squareControlAt(f *FrontEnd, p image.Point) (ui.TownSquareControl, bool) {
	return townSquareScene{f.TownScreen().(*townScreen)}.ControlAt(p)
}

// seededTownProcess is a process state with a stamped paint clock.
func seededTownProcess() town.Process {
	p := &town.Process{}
	town.NewView(ROM1TownDescription(), nil, p).SetClockLast(time.Unix(1777, 0))
	return *p
}

// squareArt builds composer art from named frames and an optional mask.
func squareArt(mask *image.Paletted, frames map[string][]image.Image) *town.Art {
	a := &town.Art{Frames: map[string][]image.Image{}, Mask: mask}
	for name, f := range frames {
		a.Frames[name] = f
	}
	return a
}

// squareFrames is n solid frames of one size.
func squareFrames(n, w, h int) []image.Image {
	out := make([]image.Image, n)
	for i := range out {
		out[i] = image.NewRGBA(image.Rect(0, 0, w, h))
	}
	return out
}

// sqFrameSet is every single-sequence actor's shown frame.
type sqFrameSet struct {
	Shop, Tavern, Fighter, Mage, Guard int
	Door, Sign, Fluger, Star           int
}

// sqFrames snapshots the square's shown frames.
func (t *townScreen) sqFrames() sqFrameSet {
	v, school := t.squareView(), t.sqSchool()
	return sqFrameSet{
		Shop: t.sqEpisode("shop").Frame, Tavern: t.sqEpisode("tavern").Frame,
		Fighter: school.Member("fighter").Frame, Mage: school.Member("mage").Frame,
		Guard: t.sqGuard().Shown(v), Door: t.sqDoor().Shown(v),
		Sign: t.sqEpisode("sign").Frame, Fluger: t.sqEpisode("fluger").Frame,
		Star: t.sqEpisode("star").Frame,
	}
}

// sqSelector is the hovered hotspot as its mask selector: shop 1, tavern 2,
// school 4, gate 8, statue 16, none -1.
func (t *townScreen) sqSelector() int {
	switch t.sqHover() {
	case "shop":
		return 1
	case "tavern":
		return 2
	case "school":
		return 4
	case "gate":
		return 8
	case "statue":
		return 16
	}
	return -1
}

// stillHost is a host with art and nothing else: no clock, draws, sound or
// campaign.
type stillHost struct{ art *town.Art }

func (h stillHost) Art() *town.Art                      { return h.art }
func (h stillHost) Now() time.Time                      { return time.Time{} }
func (h stillHost) Draw(string, int) int                { return 0 }
func (h stillHost) Seed() int64                         { return 0 }
func (h stillHost) Condition(string) bool               { return false }
func (h stillHost) PlaySound(string, string) town.Voice { return nil }
func (h stillHost) StopSound(town.Voice)                {}
func (h stillHost) StartLoop(string) bool               { return false }
func (h stillHost) StopLoop(string)                     {}
func (h stillHost) LeaveSquare()                        {}
func (h stillHost) Hook(string, string)                 {}

// staticSquareArt keeps only the square's still pictures and mask.
func staticSquareArt(full *town.Art) *town.Art {
	a := &town.Art{Frames: map[string][]image.Image{}, Mask: full.Mask}
	for _, name := range []string{"base", "overlay", "label-shop", "label-tavern", "label-school"} {
		a.Frames[name] = full.Pictures(name)
	}
	return a
}

// paintStill paints a square view of art with the bird overlay shown.
func paintStill(art *town.Art) *image.RGBA {
	v := town.NewView(ROM1TownDescription(), stillHost{art}, nil)
	v.Actor("birds").(*town.Flock).Active = true
	dst := image.NewRGBA(image.Rectangle{Max: v.Size()})
	v.Paint(dst)
	return dst
}

// spriteFrame, familyFrame and exteriorFrame are the square's shown state in
// the shape the oracle tests compare.
type spriteFrame struct {
	Family, Frame int
	Visible       bool
}

type familyFrame struct {
	Visible                bool
	Position, Sheet, Frame int
}

type exteriorFrame struct {
	Shop, Tavern, Fighter, Mage, Guard int
	Door, Sign, Fluger                 int
	Birds                              [3]spriteFrame
	BirdOverlayVisible                 bool
	Star                               spriteFrame
	Horse, Baba, Dervish               familyFrame
}

// sqExteriorFrame reads the square's shown state from the composer.
func (t *townScreen) sqExteriorFrame() exteriorFrame {
	v := t.squareView()
	fs := t.sqFrames()
	f := exteriorFrame{Shop: fs.Shop, Tavern: fs.Tavern, Fighter: fs.Fighter, Mage: fs.Mage, Guard: fs.Guard,
		Door: fs.Door, Sign: fs.Sign, Fluger: fs.Fluger}
	star := t.sqEpisode("star")
	f.Star = spriteFrame{Frame: star.Frame, Visible: star.Shown(v)}
	if !v.Ready() {
		return f
	}
	if birds := t.sqBirds(); birds.Active {
		f.BirdOverlayVisible = true
		for i, b := range birds.Shown(v) {
			if i < len(f.Birds) {
				f.Birds[i] = spriteFrame{Family: b.Family, Frame: b.Frame, Visible: b.Visible}
			}
		}
	}
	if w := t.sqWildlife(); w.Entered && v.Present() {
		show := func(name string) familyFrame {
			m := w.Member(name)
			return familyFrame{Visible: true, Position: m.Position, Sheet: m.Sheet, Frame: max(m.Current, 0)}
		}
		f.Horse, f.Baba, f.Dervish = show("horse"), show("baba"), show("dervish")
	}
	return f
}

// withoutAmbience copies art without the bird and star families.
func withoutAmbience(full *town.Art) *town.Art {
	a := &town.Art{Frames: map[string][]image.Image{}, Mask: full.Mask, Problems: full.Problems}
	for name, f := range full.Frames {
		if name != "stars" && !strings.HasPrefix(name, "birds/") {
			a.Frames[name] = f
		}
	}
	return a
}

// sqActorFrames is an actor's declared frame count in the description.
func sqActorFrames(name string) int {
	for _, a := range ROM1TownDescription().Actors {
		if a.Name == name {
			return a.Frames
		}
	}
	panic("no actor " + name)
}

func townBirdFrameCount() int { return sqActorFrames("birds") }
func townStarFrameCount() int { return sqActorFrames("star") }

// sqProgress is the flock's progress words as text.
func (t *townScreen) sqProgress() string { return fmt.Sprint(t.sqBirds().Progress) }
