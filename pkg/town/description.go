// Package town is the town composer: one builder that turns a town
// description into a town view. A description is data. It names the art, the
// mask, the hotspots, the actor programs, the clock, the sounds, the tips, the
// music and the rooms of one town, and the hooks a campaign answers. The
// package holds no fact of any game: no resource key, coordinate, mask byte,
// sound name or timing value appears in its code, and it imports no campaign
// package. A game supplies its descriptions and a Host.
package town

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
)

// Description is one town as data. Every object in the encoded form carries a
// cite list naming the claim, divergence row or owner ruling behind its values;
// the composer ignores it.
type Description struct {
	Town     string        `json:"town"`
	Cite     []string      `json:"cite"`
	View     ViewSpec      `json:"view"`
	Clock    ClockSpec     `json:"clock"`
	Random   []RandomSpec  `json:"random"`
	Art      []ArtSpec     `json:"art"`
	Mask     MaskSpec      `json:"mask"`
	Hotspots []HotspotSpec `json:"hotspots"`
	Pointer  PointerSpec   `json:"pointer"`
	Sounds   SoundSpec     `json:"sounds"`
	Actors   []ActorSpec   `json:"actors"`
	Step     StepSpec      `json:"step"`
	Layers   []LayerSpec   `json:"layers"`
	Tip      TipSpec       `json:"tip"`
	Music    MusicSpec     `json:"music"`
	Square   SquareSpec    `json:"square"`
	Rooms    []RoomSpec    `json:"rooms"`
	Save     SaveSpec      `json:"save"`
}

// ViewSpec is the view's size.
type ViewSpec struct {
	Size Point    `json:"size"`
	Cite []string `json:"cite"`
}

// ClockSpec is the square's paint clock. A step is admitted when the time
// since the last admitted step exceeds the period ("greater") or reaches it
// ("at-least"). The first paint after the view is built only stamps the clock.
type ClockSpec struct {
	PeriodMS int      `json:"period-ms"`
	Compare  string   `json:"compare"`
	Cite     []string `json:"cite"`
}

// RandomSpec names a draw source. "host" draws are bounded draws the Host
// supplies. "lcg" is a linear congruential generator the composer keeps for
// the life of the process, seeded once from the Host's seed.
type RandomSpec struct {
	Name       string   `json:"name"`
	Source     string   `json:"source"`
	Multiplier uint32   `json:"multiplier"`
	Increment  uint32   `json:"increment"`
	Shift      uint     `json:"shift"`
	Mask       uint32   `json:"mask"`
	Cite       []string `json:"cite"`
}

// ArtSpec is one named art entry. Format "picture" is one bitmap; "series" is
// numbered bitmaps loaded as one atomic family; "sprites" is one sprite sheet.
// Index expands the key over one or two zero-based indices, each printed with
// its base added; each expansion is its own entry named name/i or name/i/j.
type ArtSpec struct {
	Name        string      `json:"name"`
	Format      string      `json:"format"`
	Key         string      `json:"key"`
	Index       []IndexSpec `json:"index"`
	Count       int         `json:"count"`
	Counts      []int       `json:"counts"`
	Size        *Point      `json:"size"`
	MaxSize     *Point      `json:"max-size"`
	Required    bool        `json:"required"`
	Transparent string      `json:"transparent"`
	Cite        []string    `json:"cite"`
}

// IndexSpec is one dimension of an indexed key.
type IndexSpec struct {
	Count int      `json:"count"`
	Base  int      `json:"base"`
	Cite  []string `json:"cite"`
}

// MaskSpec names the mask art and the byte of each hotspot. RequiredBytes
// must all be present for the art to load.
type MaskSpec struct {
	Art           string     `json:"art"`
	RequiredBytes []int      `json:"required-bytes"`
	Bytes         []MaskByte `json:"bytes"`
	Cite          []string   `json:"cite"`
}

// MaskByte maps one mask byte to a hotspot. Unmapped bytes are no hotspot.
type MaskByte struct {
	Byte    int      `json:"byte"`
	Hotspot string   `json:"hotspot"`
	Cite    []string `json:"cite"`
}

// HotspotSpec is one region of the view: its row in the list fallback, its
// tooltip text slot, its click program and its hover reactions.
type HotspotSpec struct {
	Name  string   `json:"name"`
	Row   *int     `json:"row"`
	Tip   int      `json:"tip"`
	Click []Op     `json:"click"`
	Hover []Op     `json:"hover"`
	Cite  []string `json:"cite"`
}

// PointerSpec holds the reactions to every pointer update and to an update
// over no hotspot.
type PointerSpec struct {
	Every []Op     `json:"every"`
	Off   []Op     `json:"off"`
	Cite  []string `json:"cite"`
}

// Op is one instruction of a click or hover program. Exactly one verb field is
// set: arm, latch, drive, clear-latch, room, hook, menu or if.
type Op struct {
	Arm        string      `json:"arm"`
	Chance     *ChanceSpec `json:"chance"`
	Latch      string      `json:"latch"`
	Slot       string      `json:"slot"`
	Sound      string      `json:"sound"`
	Stop       []string    `json:"stop"`
	Clear      []string    `json:"clear"`
	Drive      string      `json:"drive"`
	Dir        int         `json:"dir"`
	Unless     string      `json:"unless"`
	ClearLatch []string    `json:"clear-latch"`
	Room       string      `json:"room"`
	Hook       string      `json:"hook"`
	Menu       string      `json:"menu"`
	If         string      `json:"if"`
	Then       []Op        `json:"then"`
	Else       []Op        `json:"else"`
	Cite       []string    `json:"cite"`
}

// ChanceSpec is a draw from a named source compared against a threshold:
// the chance holds when draw(n) > above.
type ChanceSpec struct {
	Draw  string   `json:"draw"`
	N     int      `json:"n"`
	Above int      `json:"above"`
	Cite  []string `json:"cite"`
}

// WaitSpec is a wait of base plus draw(span) milliseconds in the named
// arithmetic form. The wait has elapsed when the time since its start exceeds
// it ("greater") or reaches it ("at-least").
type WaitSpec struct {
	Draw    string   `json:"draw"`
	BaseMS  int      `json:"base-ms"`
	SpanMS  int      `json:"span-ms"`
	Form    string   `json:"form"`
	Compare string   `json:"compare"`
	Cite    []string `json:"cite"`
}

// PickSpec is base plus draw(n) in the named arithmetic form.
type PickSpec struct {
	Draw string   `json:"draw"`
	Form string   `json:"form"`
	N    int      `json:"n"`
	Base int      `json:"base"`
	Cite []string `json:"cite"`
}

// SoundSpec is the view's sound slots, in release order, and its entry loop.
type SoundSpec struct {
	Slots []SlotSpec `json:"slots"`
	Loop  string     `json:"loop"`
	Cite  []string   `json:"cite"`
}

// SlotSpec is one retained voice: a slot holds at most one sound, and a
// request while its sound plays is no request.
type SlotSpec struct {
	Name   string   `json:"name"`
	Source string   `json:"source"`
	Cite   []string `json:"cite"`
}

// CueSpec is one sound request on a slot, after stopping other slots.
type CueSpec struct {
	Slot string   `json:"slot"`
	Key  string   `json:"key"`
	Stop []string `json:"stop"`
	Cite []string `json:"cite"`
}

// ActorSpec is one animated element and its program. Program is one of
// episode, pendulum, stepper, driven, flock or families; the fields each
// program reads are named on its runtime type.
type ActorSpec struct {
	Name    string `json:"name"`
	Program string `json:"program"`
	Art     string `json:"art"`

	// episode
	Trigger      *ChanceSpec `json:"trigger"`
	StartSound   *CueSpec    `json:"start-sound"`
	Frames       int         `json:"frames"`
	End          string      `json:"end"`
	RewindEvery  int         `json:"rewind-every"`
	ResetOnEntry bool        `json:"reset-on-entry"`

	// pendulum
	Members []MemberSpec `json:"members"`
	Chance  *ChanceSpec  `json:"chance"`

	// stepper and driven
	Rest               string `json:"rest"`
	Hold               string `json:"hold"`
	HoldUnless         string `json:"hold-unless"`
	TowardFirstOnHover string `json:"toward-first-on-hover"`
	FirstSound         string `json:"first-sound"`
	LastSound          string `json:"last-sound"`
	Slot               string `json:"slot"`

	// flock
	Groups    int        `json:"groups"`
	GroupSize int        `json:"group-size"`
	Wait      *WaitSpec  `json:"wait"`
	Group     *PickSpec  `json:"group"`
	Count     *PickSpec  `json:"count"`
	CountCues []CountCue `json:"count-cues"`

	// families
	Entry      []EntryStep `json:"entry"`
	StepOrder  []string    `json:"step-order"`
	PaintOrder []string    `json:"paint-order"`

	Cite []string `json:"cite"`
}

// CountCue is the sound a flock requests when it starts with Count members;
// a zero Count matches any count.
type CountCue struct {
	Count int      `json:"count"`
	Key   string   `json:"key"`
	Cite  []string `json:"cite"`
}

// MemberSpec is one member of a pendulum group or a family group.
type MemberSpec struct {
	Name      string     `json:"name"`
	Art       string     `json:"art"`
	Mode      string     `json:"mode"`
	Sheets    int        `json:"sheets"`
	Positions []Point    `json:"positions"`
	Later     *WaitSpec  `json:"later"`
	SheetPick *PickSpec  `json:"sheet-pick"`
	FrameCues []FrameCue `json:"frame-cues"`
	Cite      []string   `json:"cite"`
}

// FrameCue requests a sound while a member shows Frame of Sheet; a negative
// Sheet matches any sheet. The first matching cue wins.
type FrameCue struct {
	Sheet int      `json:"sheet"`
	Frame int      `json:"frame"`
	Slot  string   `json:"slot"`
	Key   string   `json:"key"`
	Cite  []string `json:"cite"`
}

// EntryStep is one draw a family group makes when the view is entered: a
// member's position (optionally redrawn until unequal to another member's), or
// a member's first wait.
type EntryStep struct {
	Position string    `json:"position"`
	Wait     string    `json:"wait"`
	Pick     *PickSpec `json:"pick"`
	After    *WaitSpec `json:"after"`
	Unequal  string    `json:"unequal"`
	Cite     []string  `json:"cite"`
}

// StepSpec orders the clock's work. Before runs on every paint, then Admitted
// when the clock admits a step, then After on every paint. Each entry is
// "<phase> <actor>".
type StepSpec struct {
	Before   []string `json:"before"`
	Admitted []string `json:"admitted"`
	After    []string `json:"after"`
	Cite     []string `json:"cite"`
}

// LayerSpec is one paint in order: an art picture or an actor's frame, copied
// or composited over, at a view-relative point, when its condition holds.
type LayerSpec struct {
	Art   string    `json:"art"`
	Actor string    `json:"actor"`
	Mode  string    `json:"mode"`
	At    Point     `json:"at"`
	When  *WhenSpec `json:"when"`
	Cite  []string  `json:"cite"`
}

// WhenSpec is a layer condition: the hovered hotspot, or an active actor.
type WhenSpec struct {
	Hover  string   `json:"hover"`
	Active string   `json:"active"`
	Cite   []string `json:"cite"`
}

// TipSpec is the view's tip popup: its text key and rectangle.
type TipSpec struct {
	Text string   `json:"text"`
	Rect Rect     `json:"rect"`
	Cite []string `json:"cite"`
}

// MusicSpec is the square's music track.
type MusicSpec struct {
	Track string   `json:"track"`
	Cite  []string `json:"cite"`
}

// SquareSpec names the square as a room and orders the steps that return to
// it.
type SquareSpec struct {
	Name  string   `json:"name"`
	Enter []Step   `json:"enter"`
	Cite  []string `json:"cite"`
}

// RoomSpec is one room reached from the square: the page that draws it and
// the ordered steps of its entry and its exit.
type RoomSpec struct {
	Name  string   `json:"name"`
	Page  string   `json:"page"`
	Enter []Step   `json:"enter"`
	Exit  []Step   `json:"exit"`
	Cite  []string `json:"cite"`
}

// Step is one entry or exit step: a composer step ("reset" the square view)
// or a named campaign hook.
type Step struct {
	Composer string   `json:"composer"`
	Hook     string   `json:"hook"`
	Cite     []string `json:"cite"`
}

// SaveSpec names the condition that admits a save from the town.
type SaveSpec struct {
	AdmittedWhen string   `json:"admitted-when"`
	Cite         []string `json:"cite"`
}

// Point is an x, y pair.
type Point [2]int

// Pt is the image point.
func (p Point) Pt() image.Point { return image.Pt(p[0], p[1]) }

// Rect is min x, min y, max x, max y.
type Rect [4]int

// Rectangle is the image rectangle.
func (r Rect) Rectangle() image.Rectangle { return image.Rect(r[0], r[1], r[2], r[3]) }

// Decode reads a description strictly: an unknown field, a trailing value or a
// description that fails Validate is a named error.
func Decode(data []byte) (*Description, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var d Description
	if err := dec.Decode(&d); err != nil {
		return nil, fmt.Errorf("town description: %w", err)
	}
	if dec.More() {
		return nil, fmt.Errorf("town description: trailing data")
	}
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return &d, nil
}
