// Package town is the town composer: one builder that turns a town
// description into scenes, the square and each room page, all run by one
// runtime, Scene. A description is data. It names the art, the
// mask, the hotspots, the actor programs, the clock, the sounds, the music
// and the rooms of one town, and the hooks a campaign answers. The
// package holds no fact of any game: no resource key, coordinate, mask byte,
// sound name or timing value appears in its code, and it imports no campaign
// package. A game supplies its descriptions and a Host.
package town

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"

	"againrom/pkg/screenart"
)

// Description is one town as data. Every object in the encoded form carries a
// cite list naming the claim, divergence row or owner ruling behind its values;
// the composer ignores it. The top level from View to Layers is the square's
// scene in its short form, with one paint clock and three step phases;
// SquareScene answers it as a SceneSpec.
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
	Music    MusicSpec     `json:"music"`
	Square   SquareSpec    `json:"square"`
	Rooms    []RoomSpec    `json:"rooms"`
	Save     SaveSpec      `json:"save"`
	// RoomButtons is the rooms' button caption style; nil keeps the
	// engine's own.
	RoomButtons *RoomButtonsSpec `json:"room-buttons"`
	Host        *HostSpec        `json:"host"`
}

// HostSpec selects the town host's presentation and lifecycle rules.
type HostSpec struct {
	CurrentHookRoom bool     `json:"current-hook-room"`
	Shop            string   `json:"shop"`
	TrackSource     string   `json:"track-source"`
	RequireArt      bool     `json:"require-art"`
	Rows            string   `json:"rows"`
	Header          string   `json:"header"`
	HeaderPrefix    string   `json:"header-prefix"`
	Footer          string   `json:"footer"`
	Talk            string   `json:"talk"`
	Tips            string   `json:"tip-lifecycle"`
	Pages           string   `json:"pages"`
	Art             string   `json:"art"`
	AudioRoom       string   `json:"audio-room"`
	Leave           string   `json:"leave"`
	Speech          bool     `json:"speech"`
	SilentSquare    bool     `json:"silent-square"`
	ClickSounds     bool     `json:"click-sounds"`
	DialogueSpeech  bool     `json:"dialogue-speech"`
	CacheScreen     bool     `json:"cache-screen"`
	EagerSquare     bool     `json:"eager-square"`
	StartsInTown    bool     `json:"starts-in-town"`
	Cite            []string `json:"cite"`
}

// ViewSpec is the view's size.
type ViewSpec struct {
	Size Point    `json:"size"`
	Cite []string `json:"cite"`
}

// ClockSpec is the square's paint clock in the short form: a process clock
// named by the square. A step is admitted when the time since the last
// admitted step exceeds the period ("greater") or reaches it ("at-least").
// While it holds no stamp a paint only stamps it.
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

// ArtSpec is one named art entry of the one screen art loader's manifest.
type ArtSpec = screenart.Entry

// IndexSpec is one dimension of an indexed key.
type IndexSpec = screenart.Index

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
// tooltip text slot, or no tip, its click program and its hover reactions.
type HotspotSpec struct {
	Name    string   `json:"name"`
	Row     *int     `json:"row"`
	RowText string   `json:"row-text"`
	Hint    string   `json:"hint"`
	Tip     int      `json:"tip"`
	NoTip   bool     `json:"no-tip"`
	Click   []Op     `json:"click"`
	Hover   []Op     `json:"hover"`
	Cite    []string `json:"cite"`
}

// TipSlot answers the hotspot's tooltip text slot, and false for a hotspot
// that shows no tip.
func (h *HotspotSpec) TipSlot() (int, bool) { return h.Tip, !h.NoTip }

// PointerSpec holds the reactions to every pointer update and to an update
// over no hotspot.
type PointerSpec struct {
	Every []Op     `json:"every"`
	Off   []Op     `json:"off"`
	Cite  []string `json:"cite"`
}

// Op is one instruction of a click or hover program. Exactly one verb field is
// set: arm, latch, drive, clear-latch, room, hook, menu, dialogue or if. A
// latch with no slot requests no sound. A dialogue opens the conversation
// whose key is its template formatted with the named value, or the template
// itself when no value is named. A hover operation marked Changed runs only on
// the pointer update that changed the hovered hotspot.
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
	Dialogue   string      `json:"dialogue"`
	Value      string      `json:"value"`
	If         string      `json:"if"`
	Then       []Op        `json:"then"`
	Else       []Op        `json:"else"`
	Changed    bool        `json:"changed"`
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

// PickSpec is base plus draw(n) in the named arithmetic form. With Raw set,
// the draw is a raw value r in [0,Raw) from a host source and the value is
// base plus r/Divide, or base plus (r*n/(Raw-1))%n in the "scaled" form. With
// Times set, the drawn value is multiplied by it.
type PickSpec struct {
	Draw   string   `json:"draw"`
	Form   string   `json:"form"`
	N      int      `json:"n"`
	Base   int      `json:"base"`
	Raw    int      `json:"raw"`
	Divide int      `json:"divide"`
	Times  int      `json:"times"`
	Cite   []string `json:"cite"`
}

// SoundSpec is the view's sound slots, in release order, and its entry loop.
type SoundSpec struct {
	Slots []SlotSpec `json:"slots"`
	Loop  string     `json:"loop"`
	// Triggers bind the game's named room events to slot requests: one
	// table per room says which event requests which sample.
	Triggers []TriggerSpec `json:"triggers"`
	Cite     []string      `json:"cite"`
}

// TriggerSpec is one room event's request: the slot, an optional key in
// place of the slot's own, and the slots stopped first.
type TriggerSpec struct {
	On   string   `json:"on"`
	Slot string   `json:"slot"`
	Key  string   `json:"key"`
	Stop []string `json:"stop"`
	Cite []string `json:"cite"`
}

// SoundRequest is one slot request as the host receives it.
type SoundRequest struct {
	Source string
	Key    string
	Loop   bool
	// Priority is the request's channel priority; 0 when the slot names none,
	// and the host then uses its default.
	Priority int
	// Speech plays at the speech volume instead of the effects volume.
	Speech bool
}

// SlotSpec is one retained voice: a slot holds at most one sound, and a
// request while its sound plays is no request.
type SlotSpec struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	// Key is the sound a "sound <slot>" step requests.
	Key string `json:"key"`
	// Loop asks the host for a looping voice.
	Loop bool `json:"loop"`
	// Untracked plays every request and keeps no voice: a later request does
	// not wait for it and a silence does not stop it.
	Untracked bool `json:"untracked"`
	// Restart stops and rewinds a playing sound and requests it again.
	Restart bool `json:"restart"`
	// Priority is the request priority; 0 leaves the host's default.
	Priority int `json:"priority"`
	// Channel "speech" plays at the speech volume; "" at the effects volume.
	Channel string   `json:"channel"`
	Cite    []string `json:"cite"`
}

func (sl SlotSpec) request(key string) SoundRequest {
	return SoundRequest{Source: sl.Source, Key: key, Loop: sl.Loop, Priority: sl.Priority, Speech: sl.Channel == "speech"}
}

// CueSpec is one sound request on a slot, after stopping other slots.
type CueSpec struct {
	Slot string   `json:"slot"`
	Key  string   `json:"key"`
	Stop []string `json:"stop"`
	Cite []string `json:"cite"`
}

// ActorSpec names an actor program and its parameters. Each runtime type
// defines the fields its program reads.
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

	// stepper and driven; HoldUnless also holds a cycle's idle
	Rest               string `json:"rest"`
	Hold               string `json:"hold"`
	HoldUnless         string `json:"hold-unless"`
	TowardFirstOnHover string `json:"toward-first-on-hover"`
	TowardLastOnHover  string `json:"toward-last-on-hover"`
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

	// route: its first wait, then Wait, and the route each start picks
	First *WaitSpec `json:"first"`
	Route *PickSpec `json:"route"`

	// room programs: loop, alternating, selector, priority, bounce, cycle,
	// target and training
	Loop      int           `json:"loop"`
	Clock     string        `json:"clock"`
	Modulus   int           `json:"modulus"`
	Delay     *PickSpec     `json:"delay"`
	DelayMS   int           `json:"delay-ms"`
	States    []StateSpec   `json:"states"`
	On        []EventSpec   `json:"on"`
	LoopAt    int           `json:"loop-at"`
	LoopTo    int           `json:"loop-to"`
	StopAt    int           `json:"stop-at"`
	Release   int           `json:"release"`
	Selected  string        `json:"selected"`
	Raise     string        `json:"raise"`
	Endpoints []int         `json:"endpoints"`
	Order     [][]int       `json:"order"`
	Variants  []VariantSpec `json:"variants"`
	HoldPick  *PickSpec     `json:"hold-pick"`
	Column    string        `json:"column"`
	Value     string        `json:"value"`

	// priority: the state an elapsed wait raises, and an index that
	// outlives every entry
	IdlePick  *PickSpec `json:"idle-pick"`
	KeepIndex bool      `json:"keep-index"`

	// transit: the cells, the cells shown while Toward is chosen, and the
	// two loops the cursor rests in
	Segments         []string `json:"segments"`
	SelectedSegments []string `json:"selected-segments"`
	Loops            [][2]int `json:"loops"`
	Toward           *int     `json:"toward"`

	Cite []string `json:"cite"`
}

// CountCue is the sound a flock requests when it starts with Count members;
// a zero Count matches any count.
type CountCue struct {
	Count int      `json:"count"`
	Key   string   `json:"key"`
	Cite  []string `json:"cite"`
}

// MemberSpec is one member of a pendulum group or a family group. A family
// member's art is indexed by position then sheet; art with one index and
// sheets is indexed by sheet alone. Stamp "start" keeps the wait's clock from
// the episode's start, where by default every advance restamps it. IdleSheet
// shows that sheet's first frame while no episode runs. FramePositions move
// a running member while its frame lies in a range. HoverSheet is the sheet
// a member starts on and idles on while its hotspot is hovered, and a hover
// arm starts it there at once. SheetPositions place a member by the sheet it
// draws, in place of Positions. StartWhen "idle" starts a member only when no
// episode runs: an elapsed wait still restamps its clock and draws the next.
type MemberSpec struct {
	Name           string          `json:"name"`
	Art            string          `json:"art"`
	Mode           string          `json:"mode"`
	Sheets         int             `json:"sheets"`
	Positions      []Point         `json:"positions"`
	Later          *WaitSpec       `json:"later"`
	SheetPick      *PickSpec       `json:"sheet-pick"`
	Stamp          string          `json:"stamp"`
	IdleSheet      *int            `json:"idle-sheet"`
	FramePositions []FramePosition `json:"frame-positions"`
	FrameCues      []FrameCue      `json:"frame-cues"`
	HoverSheet     *HoverSheet     `json:"hover-sheet"`
	SheetPositions []Point         `json:"sheet-positions"`
	StartWhen      string          `json:"start-when"`
	Cite           []string        `json:"cite"`
}

// HoverSheet names the hotspot whose hover chooses a member's sheet.
type HoverSheet struct {
	Hotspot string   `json:"hotspot"`
	Sheet   int      `json:"sheet"`
	Cite    []string `json:"cite"`
}

// FramePosition draws a running member at At while its frame lies in
// [From, To].
type FramePosition struct {
	From int      `json:"from"`
	To   int      `json:"to"`
	At   Point    `json:"at"`
	Cite []string `json:"cite"`
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

// StepSpec orders the square's clock work in the short form. Before runs on
// every paint, then Admitted when the clock admits a step, then After on
// every paint. Each entry is "<phase> <actor>" or "sound <slot>".
type StepSpec struct {
	Before   []string `json:"before"`
	Admitted []string `json:"admitted"`
	After    []string `json:"after"`
	Cite     []string `json:"cite"`
}

// LayerSpec is one paint in order: an art picture or an actor's frame, copied
// or composited over, at a view-relative point, when its condition holds. A
// room scene's layers belong to named groups the page draws between its own
// widgets, and may be clipped to a rectangle.
type LayerSpec struct {
	Art   string    `json:"art"`
	Actor string    `json:"actor"`
	Mode  string    `json:"mode"`
	At    Point     `json:"at"`
	When  *WhenSpec `json:"when"`
	Group string    `json:"group"`
	Clip  *Rect     `json:"clip"`
	Cite  []string  `json:"cite"`
}

// WhenSpec is a layer condition: the hovered hotspot, an active actor, a
// family member running an episode on a sheet, a campaign condition, or any
// one of a list of conditions.
type WhenSpec struct {
	If      string     `json:"if"`
	Hover   string     `json:"hover"`
	Active  string     `json:"active"`
	Running string     `json:"running"`
	Sheet   *int       `json:"sheet"`
	Any     []WhenSpec `json:"any"`
	Cite    []string   `json:"cite"`
}

// MusicSpec is a view's music: one track, or one track per value of a
// variant the host answers, the shown variant's track first.
type MusicSpec struct {
	Track  string   `json:"track"`
	Tracks []string `json:"tracks"`
	By     string   `json:"by"`
	Cite   []string `json:"cite"`
}

// RoomMusicSpec selects a room's music scene and its preferred class.
type RoomMusicSpec struct {
	Scene string   `json:"scene"`
	By    string   `json:"by"`
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
	Name  string     `json:"name"`
	Page  string     `json:"page"`
	Enter []Step     `json:"enter"`
	Exit  []Step     `json:"exit"`
	Scene *SceneSpec `json:"scene"`
	// Art is the page's own art outside its scene: the panels, controls and
	// sheets its page kind draws, read by the one screen art loader.
	Art       []ArtSpec      `json:"art"`
	Music     *MusicSpec     `json:"music"`
	MusicRule *RoomMusicSpec `json:"music-rule"`
	// Seats and Buttons are a page's own controls as data: the seat grid its
	// page draws the campaign's roster in, and its buttons in index order.
	Seats   *SeatSpec    `json:"seats"`
	Buttons []ButtonSpec `json:"buttons"`
	// Pane is the page's character pane; nil shows none.
	Pane *PaneSpec `json:"pane"`
	// Shop is a shop page's own drawing rules; nil keeps the first game's.
	Shop *ShopSpec `json:"shop"`
	Cite []string  `json:"cite"`
}

// ShopSpec is a shop page's drawing rules as data. MoneyLast shows the gold
// after the bag's items; TableShelfBack draws the merchant's items on the
// table on the shelf's backs; LabelOffset moves each command plaque's label
// line, in plaque order; ShelfGems draws a shelf scroll gem dark while the
// shelf cannot turn that way.
type ShopSpec struct {
	MoneyLast      bool     `json:"money-last"`
	ShelfGems      bool     `json:"shelf-gems"`
	TableShelfBack bool     `json:"table-shelf-back"`
	LabelOffset    [4]Point `json:"label-offset"`
	Cite           []string `json:"cite"`
}

// PaneSpec is a page's character pane: the campaign's shown party member in
// Rect as figure and worn items or as statistics card, with the picker and
// mode corners, and the spellbook corner when Book. BookOffset moves the open
// book from the table strip's corner. Font is the chrome's font:
// PaneFontInstall the install's, empty the generator's, the card's font.
type PaneSpec struct {
	Rect       Rect     `json:"rect"`
	Book       bool     `json:"book"`
	BookOffset Point    `json:"book_offset"`
	Font       string   `json:"font"`
	Cite       []string `json:"cite"`
}

// PaneFontInstall draws a pane's chrome in the install's font.
const PaneFontInstall = "install"

// Step is one entry or exit step: a composer step ("reset" the square view)
// or a named campaign hook.
type Step struct {
	Composer string   `json:"composer"`
	Hook     string   `json:"hook"`
	Cite     []string `json:"cite"`
}

// RoomButtonsSpec is how the rooms' button captions draw: in the game's
// generator font Font names ("name" or "text"), in Ink at rest and HoverInk
// under the pointer; a button that cannot act keeps Ink, or with Lit draws
// as one that can and still refuses the press. BuyDebit shows the shop's
// pending buy as the gold it takes, below zero. OneLine names the shop
// buttons, by panel index, whose caption and number are one line.
// LabelShift and ValueShift move the label and the value line of a
// two-line caption.
type RoomButtonsSpec struct {
	Font     string   `json:"font"`
	Ink      [3]uint8 `json:"ink"`
	HoverInk [3]uint8 `json:"hover-ink"`
	BuyDebit bool     `json:"buy-debit"`
	Lit      bool     `json:"lit"`
	OneLine  []int    `json:"one-line"`
	// The shifts are in pixels, [x, y].
	LabelShift Point    `json:"label-shift"`
	ValueShift Point    `json:"value-shift"`
	Cite       []string `json:"cite"`
}

// SaveSpec names the condition that admits a save from the town.
type SaveSpec struct {
	AdmittedWhen string   `json:"admitted-when"`
	Cite         []string `json:"cite"`
}

// SquareScene answers the square's scene: the description's top level, its
// clock a process clock named by the square, its steps three groups of which
// the second is bound to that clock, entered each time it is shown, and
// advanced only while active. The answer shares the description's lists.
func (d *Description) SquareScene() *SceneSpec {
	period := d.Clock.PeriodMS
	clock := d.Square.Name
	return &SceneSpec{
		View: d.View, Random: d.Random, Art: d.Art, Mask: d.Mask, Hotspots: d.Hotspots,
		Pointer: d.Pointer, Sounds: d.Sounds, Actors: d.Actors, Layers: d.Layers,
		Clocks: []SceneClock{{Name: clock, PeriodMS: &period, Compare: d.Clock.Compare,
			Process: true, Cite: d.Clock.Cite}},
		Lifecycle:   LifecycleShown,
		AdvanceWhen: "active",
		Steps: []StepGroup{
			{Run: d.Step.Before, Cite: d.Step.Cite},
			{Clock: clock, Run: d.Step.Admitted, Cite: d.Step.Cite},
			{Run: d.Step.After, Cite: d.Step.Cite},
		},
		Cite: d.Cite,
	}
}

// Scene answers the named room's scene: the square's name answers
// SquareScene, a room its own scene, and a room without a scene or an
// unknown name nil.
func (d *Description) Scene(room string) *SceneSpec {
	if room == d.Square.Name {
		return d.SquareScene()
	}
	for i := range d.Rooms {
		if d.Rooms[i].Name == room {
			return d.Rooms[i].Scene
		}
	}
	return nil
}

// Room answers the named room, nil for the square or an unknown name.
func (d *Description) Room(name string) *RoomSpec {
	for i := range d.Rooms {
		if d.Rooms[i].Name == name {
			return &d.Rooms[i]
		}
	}
	return nil
}

// Point is an x, y pair.
type Point [2]int

// Pt is the image point.
func (p Point) Pt() image.Point { return image.Pt(p[0], p[1]) }

// Rect is min x, min y, max x, max y.
type Rect [4]int

// Rectangle is the image rectangle.
func (r Rect) Rectangle() image.Rectangle { return image.Rect(r[0], r[1], r[2], r[3]) }

// Vocabulary is the hook, condition, value, event and page kind names a game
// answers. A description that names any other is refused when it is read.
type Vocabulary struct {
	Hooks      []string
	Conditions []string
	Values     []string
	Events     []string
	Pages      []string
}

func (v Vocabulary) has(list []string, name string) bool {
	for _, n := range list {
		if n == name {
			return true
		}
	}
	return false
}

// Decode reads a description strictly: an unknown field, any data after the
// description, a hook or condition outside vocab, or a description that fails
// Validate is a named error.
func Decode(data []byte, vocab Vocabulary) (*Description, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var d Description
	if err := dec.Decode(&d); err != nil {
		return nil, fmt.Errorf("town description: %w", err)
	}
	if rest := bytes.TrimSpace(data[dec.InputOffset():]); len(rest) > 0 {
		return nil, fmt.Errorf("town description: trailing data %q", firstBytes(rest))
	}
	if err := d.Validate(); err != nil {
		return nil, err
	}
	if err := d.checkVocabulary(vocab); err != nil {
		return nil, err
	}
	return &d, nil
}

func firstBytes(b []byte) []byte {
	if len(b) > 16 {
		return b[:16]
	}
	return b
}
