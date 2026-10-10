package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"
	"time"
)

// The character generator is one builder over a description. A description is
// data: it names the art, the masks, the controls, the texts, the sounds, the
// timings, the keys and the rules of one game's generator. The generator's
// pages, input and tips read every game fact from it; their code holds no
// resource key, coordinate, mask byte, text slot or timing value. A game's
// campaign adapter supplies what the description cannot state as a value:
// the hero a result makes, the preview of that hero and the commit.

// GeneratorDescription is one character generator as data. Every object in
// the encoded form carries a cite list naming the claim, divergence row or
// owner ruling behind its values; the builder ignores it.
type GeneratorDescription struct {
	Generator string             `json:"generator"`
	Cite      []string           `json:"cite"`
	Page      GeneratorPage      `json:"page"`
	Words     GeneratorWords     `json:"words"`
	PreCreate GeneratorPreCreate `json:"pre-create"`
	Detail    GeneratorDetail    `json:"detail"`
	Tips      GeneratorTips      `json:"tips"`
	// Campaign is the part only the game's campaign adapter reads: the hero
	// a result makes and what the commit writes. The builder never decodes it.
	Campaign json.RawMessage `json:"campaign"`
}

// GeneratorPoint is x, y.
type GeneratorPoint [2]int

// Pt is the image point.
func (p GeneratorPoint) Pt() image.Point { return image.Pt(p[0], p[1]) }

// GeneratorRect is min x, min y, max x, max y.
type GeneratorRect [4]int

// Rectangle is the image rectangle.
func (r GeneratorRect) Rectangle() image.Rectangle { return image.Rect(r[0], r[1], r[2], r[3]) }

// GeneratorInk is red, green, blue.
type GeneratorInk [3]uint8

// RGBA is the opaque colour.
func (k GeneratorInk) RGBA() color.RGBA { return color.RGBA{k[0], k[1], k[2], 0xff} }

// GeneratorPage is the frame both pages are composed in and the fill under
// the art.
type GeneratorPage struct {
	Size     GeneratorPoint `json:"size"`
	Backdrop GeneratorInk   `json:"backdrop"`
	Cite     []string       `json:"cite"`
}

// GeneratorWords names the text tables and fonts the generator reads.
type GeneratorWords struct {
	Table    string        `json:"table"`
	Names    string        `json:"names"`
	TextFont GeneratorFont `json:"text-font"`
	NameFont GeneratorFont `json:"name-font"`
	// TipFont is "text" when the install's tip panels draw in the
	// generator's text font, "install" when they keep the install's font.
	TipFont string `json:"tip-font"`
	// CodePage is "windows-1251" when a converting install's tables hold
	// Windows Cyrillic, which the generator converts to the fonts' code page;
	// empty when the tables are in the fonts' code page.
	CodePage string   `json:"code-page"`
	Cite     []string `json:"cite"`
}

// GeneratorFont names a font and its atlas format, "16" or "16a".
type GeneratorFont struct {
	Name  string   `json:"name"`
	Atlas string   `json:"atlas"`
	Cite  []string `json:"cite"`
}

// GeneratorArt is one picture: its archive key, its required size, where it
// is drawn, and whether pure black is a hole.
type GeneratorArt struct {
	Key   string          `json:"key"`
	Size  *GeneratorPoint `json:"size"`
	At    GeneratorPoint  `json:"at"`
	Keyed bool            `json:"keyed"`
	Cite  []string        `json:"cite"`
}

// GeneratorMask is a mask picture and the bytes it must hold.
type GeneratorMask struct {
	Key      string         `json:"key"`
	Size     GeneratorPoint `json:"size"`
	Required []int          `json:"required"`
	Cite     []string       `json:"cite"`
}

// GeneratorWhen is a state condition over one control: each named field must
// equal the control's state; an absent field matches either.
type GeneratorWhen struct {
	Selected          *bool `json:"selected"`
	Hovered           *bool `json:"hovered"`
	NeighbourHovered  *bool `json:"neighbour-hovered"`
	NeighbourSelected *bool `json:"neighbour-selected"`
}

// GeneratorStateArt is one picture a control draws while its condition holds.
type GeneratorStateArt struct {
	When GeneratorWhen   `json:"when"`
	Key  string          `json:"key"`
	At   GeneratorPoint  `json:"at"`
	Size *GeneratorPoint `json:"size"`
	Cite []string        `json:"cite"`
}

// GeneratorHero is one pre-create hero: the sex and class it stands for, the
// mask byte that hits it, the name line a press writes, the hero beside it
// whose hover and selection it shares art with, and its state art.
type GeneratorHero struct {
	Sex       int                 `json:"sex"`
	Class     int                 `json:"class"`
	Template  string              `json:"template"`
	Mask      int                 `json:"mask"`
	NameLine  int                 `json:"name-line"`
	Neighbour *int                `json:"neighbour"`
	Tooltip   *int                `json:"tooltip"`
	Art       []GeneratorStateArt `json:"art"`
	Cite      []string            `json:"cite"`
}

// GeneratorLevel is one difficulty control.
type GeneratorLevel struct {
	Mask    int                 `json:"mask"`
	Sound   string              `json:"sound"`
	Tooltip *int                `json:"tooltip"`
	Art     []GeneratorStateArt `json:"art"`
	Cite    []string            `json:"cite"`
}

// GeneratorButton is a pre-create button: the mask byte that hits it, the
// region a mask-less hit test clips its art to, the art it draws while
// active, its sound and its tooltip.
type GeneratorButton struct {
	Mask    int            `json:"mask"`
	Region  *GeneratorRect `json:"region"`
	Art     GeneratorArt   `json:"art"`
	Sound   string         `json:"sound"`
	Tooltip *int           `json:"tooltip"`
	Cite    []string       `json:"cite"`
}

// GeneratorPrompt is the name field's label: a main table slot drawn at At,
// left-aligned there or right-aligned ending there.
type GeneratorPrompt struct {
	Slot  int            `json:"slot"`
	At    GeneratorPoint `json:"at"`
	Align string         `json:"align"`
	Ink   GeneratorInk   `json:"ink"`
	Cite  []string       `json:"cite"`
}

// GeneratorNameField is the pre-create name field and its default-name rule.
// EnterLine is the names-table line the page's enter writes over a default
// text; the defaults are every hero's name line and Unnamed. LastPress keeps
// a hero press from writing when it repeats the last pressed hero.
type GeneratorNameField struct {
	Rect      GeneratorRect   `json:"rect"`
	TextAt    GeneratorPoint  `json:"text-at"`
	Ink       GeneratorInk    `json:"ink"`
	Cap       int             `json:"cap"`
	CaretMS   int             `json:"caret-ms"`
	Tooltip   *int            `json:"tooltip"`
	Prompt    GeneratorPrompt `json:"prompt"`
	EnterLine int             `json:"enter-line"`
	Unnamed   string          `json:"unnamed"`
	LastPress bool            `json:"last-press"`
	Cite      []string        `json:"cite"`
}

// GeneratorSparkle is the pre-create sparkle sprite. It steps one frame per
// StepMS of shown time, frame 0 hidden, then waits GapSteps steps or GapMS
// plus a draw divided by GapDrawDivisor milliseconds. Each run is placed at
// the next Tour point, or at a random point of a random control named in
// Within.
type GeneratorSparkle struct {
	Key            string           `json:"key"`
	StepMS         int              `json:"step-ms"`
	ClampMS        int              `json:"clamp-ms"`
	GapSteps       int              `json:"gap-steps"`
	GapMS          int              `json:"gap-ms"`
	GapDrawDivisor int              `json:"gap-draw-divisor"`
	Tour           []GeneratorPoint `json:"tour"`
	Within         []string         `json:"within"`
	Cite           []string         `json:"cite"`
}

// GeneratorLoop is a looping numbered picture series: Count members printed
// from First into Key, drawn at At, one member per paint more than PeriodMS
// after the last, starting Phase members in.
type GeneratorLoop struct {
	Key      string         `json:"key"`
	First    int            `json:"first"`
	Count    int            `json:"count"`
	Size     GeneratorPoint `json:"size"`
	At       GeneratorPoint `json:"at"`
	Phase    int            `json:"phase"`
	PeriodMS int            `json:"period-ms"`
	Cite     []string       `json:"cite"`
}

// GeneratorKeys is one page's keyboard and click rules. Enter is "focused"
// (activates the focused control), "forward" or "play"; Escape is "leave"
// (unwinds to the screen that armed the generator), "back" or "none"; Typing
// is "focused" (the name takes typing while focused) or "always".
type GeneratorKeys struct {
	Enter              string   `json:"enter"`
	Escape             string   `json:"escape"`
	EscapeSound        string   `json:"escape-sound"`
	Typing             string   `json:"typing"`
	FocusKeys          bool     `json:"focus-keys"`
	DoubleClickForward bool     `json:"double-click-forward"`
	DoubleClickMS      int      `json:"double-click-ms"`
	Cite               []string `json:"cite"`
}

// GeneratorPreCreate is the first page.
type GeneratorPreCreate struct {
	Cursor     string             `json:"cursor"`
	Background GeneratorArt       `json:"background"`
	Mask       GeneratorMask      `json:"mask"`
	Name       GeneratorNameField `json:"name"`
	Heroes     []GeneratorHero    `json:"heroes"`
	HeroOrder  []int              `json:"hero-order"`
	HeroSound  string             `json:"hero-sound"`
	Levels     []GeneratorLevel   `json:"levels"`
	Back       GeneratorButton    `json:"back"`
	Forward    GeneratorButton    `json:"forward"`
	Sparkle    *GeneratorSparkle  `json:"sparkle"`
	Loops      []GeneratorLoop    `json:"loops"`
	Focus      []string           `json:"focus"`
	Keys       GeneratorKeys      `json:"keys"`
	Cite       []string           `json:"cite"`
}

// GeneratorPane is one detail-page picture drawn into a rectangle.
type GeneratorPane struct {
	Key   string         `json:"key"`
	Size  GeneratorPoint `json:"size"`
	Rect  GeneratorRect  `json:"rect"`
	Keyed bool           `json:"keyed"`
	Cite  []string       `json:"cite"`
}

// GeneratorSkill is one skill cell of a class column.
type GeneratorSkill struct {
	Dir   string         `json:"dir"`
	Mask  int            `json:"mask"`
	At    GeneratorPoint `json:"at"`
	Sound string         `json:"sound"`
	Cite  []string       `json:"cite"`
}

// GeneratorClass is one class's skill column. Selectable is how many of its
// skills, from the first, are drawn, hit and cycled.
type GeneratorClass struct {
	Column     GeneratorArt     `json:"column"`
	Mask       GeneratorArt     `json:"mask"`
	Skills     []GeneratorSkill `json:"skills"`
	Selectable int              `json:"selectable"`
	Tooltip    int              `json:"tooltip"`
	Cite       []string         `json:"cite"`
}

// GeneratorSkillStates are the file names of a skill cell's three pictures.
type GeneratorSkillStates struct {
	SelectedAtRest string   `json:"selected-at-rest"`
	Hover          string   `json:"hover"`
	Selected       string   `json:"selected"`
	Cite           []string `json:"cite"`
}

// GeneratorStatButtons are one direction's button pictures.
type GeneratorStatButtons struct {
	Rest     string   `json:"rest"`
	Hover    string   `json:"hover"`
	Down     string   `json:"down"`
	Unused   string   `json:"unused"`
	Disabled string   `json:"disabled"`
	Cite     []string `json:"cite"`
}

// GeneratorCost is the cumulative cost of a value v:
// trunc(Factor * Base^(v-1) + Round).
type GeneratorCost struct {
	Factor float64  `json:"factor"`
	Base   float64  `json:"base"`
	Round  float64  `json:"round"`
	Cite   []string `json:"cite"`
}

// GeneratorStats is the attribute block: boxes, buttons, inks, bounds,
// budget, cost and tooltips.
type GeneratorStats struct {
	Value        []GeneratorRect      `json:"value"`
	Plus         []GeneratorRect      `json:"plus"`
	Minus        []GeneratorRect      `json:"minus"`
	LabelLeft    int                  `json:"label-left"`
	Pool         GeneratorRect        `json:"pool"`
	ValueInk     GeneratorInk         `json:"value-ink"`
	PoolInk      GeneratorInk         `json:"pool-ink"`
	ButtonSize   GeneratorPoint       `json:"button-size"`
	MinusArt     GeneratorStatButtons `json:"minus-art"`
	PlusArt      GeneratorStatButtons `json:"plus-art"`
	ButtonDir    string               `json:"button-dir"`
	Sound        string               `json:"sound"`
	LabelTooltip int                  `json:"label-tooltip"`
	PoolTooltip  int                  `json:"pool-tooltip"`
	ValueLabel   int                  `json:"value-label"`
	Floor        int                  `json:"floor"`
	Ceiling      int                  `json:"ceiling"`
	Start        int                  `json:"start"`
	Budget       int                  `json:"budget"`
	Cost         GeneratorCost        `json:"cost"`
	Cite         []string             `json:"cite"`
}

// GeneratorCommand is one of Accept, Reset and Back.
type GeneratorCommand struct {
	Role string         `json:"role"`
	Slot int            `json:"slot"`
	Rect GeneratorRect  `json:"rect"`
	Off  string         `json:"off"`
	On   string         `json:"on"`
	Size GeneratorPoint `json:"size"`
	Cite []string       `json:"cite"`
}

// GeneratorReset is what Reset restores for one install language: "template"
// values or every attribute's "start"; the skill is kept or set to the
// default.
type GeneratorReset struct {
	Language string   `json:"language"`
	Values   string   `json:"values"`
	Skill    string   `json:"skill"`
	Cite     []string `json:"cite"`
}

// GeneratorRepeat is the held attribute button repeat, in App ticks.
type GeneratorRepeat struct {
	DelayTicks    int      `json:"delay-ticks"`
	IntervalTicks int      `json:"interval-ticks"`
	Cite          []string `json:"cite"`
}

// GeneratorRefusals are the Accept refusals: the empty-name and
// reserved-name texts and the reserved names, compared case-folded.
type GeneratorRefusals struct {
	EmptySlot    *int     `json:"empty-slot"`
	ReservedSlot *int     `json:"reserved-slot"`
	Reserved     []string `json:"reserved"`
	Cite         []string `json:"cite"`
}

// GeneratorPreview places the hero figure and the text drawn when it is
// missing.
type GeneratorPreview struct {
	FigureOffset GeneratorPoint `json:"figure-offset"`
	MissingAt    GeneratorPoint `json:"missing-at"`
	Missing      string         `json:"missing"`
	MissingInk   GeneratorInk   `json:"missing-ink"`
	Cite         []string       `json:"cite"`
}

// GeneratorNavFrame is the frame drawn where the command panel's art is
// missing.
type GeneratorNavFrame struct {
	Fill GeneratorInk `json:"fill"`
	Edge GeneratorInk `json:"edge"`
	Cite []string     `json:"cite"`
}

// GeneratorDetail is the second page.
type GeneratorDetail struct {
	Cursor       string               `json:"cursor"`
	Plate        GeneratorPane        `json:"plate"`
	PlateSeam    *GeneratorPane       `json:"plate-seam"`
	Nav          GeneratorPane        `json:"nav"`
	NavSeam      *GeneratorPane       `json:"nav-seam"`
	NavFrame     GeneratorNavFrame    `json:"nav-frame"`
	Card         GeneratorPane        `json:"card"`
	CardSeam     *GeneratorPane       `json:"card-seam"`
	Doll         GeneratorPane        `json:"doll"`
	DollSeam     *GeneratorPane       `json:"doll-seam"`
	ColumnRect   GeneratorRect        `json:"column-rect"`
	SkillClip    GeneratorRect        `json:"skill-clip"`
	Classes      []GeneratorClass     `json:"classes"`
	SkillStates  GeneratorSkillStates `json:"skill-states"`
	Stats        GeneratorStats       `json:"stats"`
	Commands     []GeneratorCommand   `json:"commands"`
	CommandInk   GeneratorInk         `json:"command-ink"`
	Message      GeneratorRect        `json:"message"`
	Preview      GeneratorPreview     `json:"preview"`
	Focus        []string             `json:"focus"`
	Keys         GeneratorKeys        `json:"keys"`
	Repeat       *GeneratorRepeat     `json:"repeat"`
	Reset        []GeneratorReset     `json:"reset"`
	Refusals     GeneratorRefusals    `json:"refusals"`
	DefaultSkill *int                 `json:"default-skill"`
	Cite         []string             `json:"cite"`
}

// GeneratorTipText is one tip text: a whole file, or one section of a file.
type GeneratorTipText struct {
	File    string   `json:"file"`
	Section string   `json:"section"`
	Cite    []string `json:"cite"`
}

// GeneratorTips are both pages' tip panels and highlight cycles. The
// pre-create cycle's targets are mask bytes per tip step; the detail cycle
// walks the selectable skills.
type GeneratorTips struct {
	PreCreateRect  GeneratorRect      `json:"pre-create-rect"`
	DetailRect     GeneratorRect      `json:"detail-rect"`
	Select         []GeneratorTipText `json:"select"`
	Fighter        GeneratorTipText   `json:"fighter"`
	Mage           GeneratorTipText   `json:"mage"`
	After          GeneratorTipText   `json:"after"`
	PreCreateCycle [][]int            `json:"pre-create-cycle"`
	HoverWaitMS    int                `json:"hover-wait-ms"`
	StepGapMS      int                `json:"step-gap-ms"`
	Cite           []string           `json:"cite"`
}

// The builder's control slots. A description fills each one; the counts are
// the builder's limits, not a game's.
const (
	generatorHeroes   = int(chargenChoice3-chargenChoice0) + 1
	generatorLevels   = int(chargenLevel2-chargenLevel0) + 1
	generatorSkills   = int(chargenSkill4-chargenSkill0) + 1
	generatorStats    = int(chargenStatPlus3-chargenStatPlus0) + 1
	generatorClasses  = 2
	generatorCommands = 3
)

// The command roles, in the order a description lists them.
var generatorCommandRoles = [generatorCommands]string{"play", "reset", "back"}

// DecodeGenerator reads one description strictly: an unknown field, a
// trailing value or a failed check is an error naming the description.
func DecodeGenerator(data []byte) (*GeneratorDescription, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var d GeneratorDescription
	if err := dec.Decode(&d); err != nil {
		return nil, fmt.Errorf("generator description: %w", err)
	}
	if dec.More() {
		return nil, fmt.Errorf("generator description %s: trailing data", d.Generator)
	}
	if err := d.validate(); err != nil {
		return nil, fmt.Errorf("generator description %s: %w", d.Generator, err)
	}
	return &d, nil
}

func (d *GeneratorDescription) validate() error {
	p, t := &d.PreCreate, &d.Detail
	switch {
	case d.Generator == "":
		return fmt.Errorf("no name")
	case d.Page.Size[0] <= 0 || d.Page.Size[1] <= 0:
		return fmt.Errorf("page size %v", d.Page.Size)
	case len(p.Heroes) != generatorHeroes:
		return fmt.Errorf("%d heroes, want %d", len(p.Heroes), generatorHeroes)
	case len(p.Levels) != generatorLevels:
		return fmt.Errorf("%d levels, want %d", len(p.Levels), generatorLevels)
	case len(t.Classes) != generatorClasses:
		return fmt.Errorf("%d classes, want %d", len(t.Classes), generatorClasses)
	case len(t.Stats.Value) != generatorStats || len(t.Stats.Plus) != generatorStats || len(t.Stats.Minus) != generatorStats:
		return fmt.Errorf("attribute boxes are not %d each", generatorStats)
	case len(t.Commands) != generatorCommands:
		return fmt.Errorf("%d commands, want %d", len(t.Commands), generatorCommands)
	case len(d.Tips.Select) != len(d.Tips.PreCreateCycle):
		return fmt.Errorf("%d pre-create tip texts for %d cycle steps", len(d.Tips.Select), len(d.Tips.PreCreateCycle))
	case p.Name.Cap <= 0 || p.Name.CaretMS <= 0:
		return fmt.Errorf("name cap %d, caret %d ms", p.Name.Cap, p.Name.CaretMS)
	}
	if err := checkOrder(p.HeroOrder, generatorHeroes); err != nil {
		return fmt.Errorf("hero order: %w", err)
	}
	for i, h := range p.Heroes {
		if h.Neighbour != nil && (*h.Neighbour < 0 || *h.Neighbour >= generatorHeroes || *h.Neighbour == i) {
			return fmt.Errorf("hero %d: neighbour %d", i, *h.Neighbour)
		}
		if len(h.Art) == 0 {
			return fmt.Errorf("hero %d: no art", i)
		}
	}
	for i, l := range p.Levels {
		if len(l.Art) == 0 {
			return fmt.Errorf("level %d: no art", i)
		}
	}
	for i, c := range t.Classes {
		if len(c.Skills) != generatorSkills || c.Selectable < 1 || c.Selectable > generatorSkills {
			return fmt.Errorf("class %d: %d skills, %d selectable", i, len(c.Skills), c.Selectable)
		}
	}
	for i, c := range t.Commands {
		if c.Role != generatorCommandRoles[i] {
			return fmt.Errorf("command %d is %q, want %q", i, c.Role, generatorCommandRoles[i])
		}
	}
	for _, k := range []GeneratorKeys{p.Keys, t.Keys} {
		if !oneOf(k.Enter, "focused", "forward", "play") || !oneOf(k.Escape, "leave", "back", "none") || !oneOf(k.Typing, "focused", "always") {
			return fmt.Errorf("keys %q/%q/%q", k.Enter, k.Escape, k.Typing)
		}
	}
	if !oneOf(p.Name.Prompt.Align, "left", "right") {
		return fmt.Errorf("prompt alignment %q", p.Name.Prompt.Align)
	}
	if !oneOf(d.Words.TipFont, "text", "install") {
		return fmt.Errorf("tip font %q", d.Words.TipFont)
	}
	if !oneOf(d.Words.CodePage, "", "windows-1251") {
		return fmt.Errorf("code page %q", d.Words.CodePage)
	}
	if len(t.Reset) == 0 {
		return fmt.Errorf("no reset rule")
	}
	for _, r := range t.Reset {
		if !oneOf(r.Values, "template", "start") || !oneOf(r.Skill, "keep", "default") {
			return fmt.Errorf("reset %q/%q", r.Values, r.Skill)
		}
	}
	for _, f := range [][]string{p.Focus, t.Focus} {
		for _, name := range f {
			if _, ok := generatorControlNamed(name); !ok {
				return fmt.Errorf("focus names %q", name)
			}
		}
	}
	if s := p.Sparkle; s != nil {
		if s.StepMS <= 0 || len(s.Tour) == 0 && len(s.Within) == 0 {
			return fmt.Errorf("sparkle step %d ms, %d tour points, %d controls", s.StepMS, len(s.Tour), len(s.Within))
		}
		for _, name := range s.Within {
			if _, ok := generatorControlNamed(name); !ok {
				return fmt.Errorf("sparkle names %q", name)
			}
		}
	}
	for _, l := range p.Loops {
		if l.Count <= 0 || l.PeriodMS <= 0 {
			return fmt.Errorf("loop %s: %d members, %d ms", l.Key, l.Count, l.PeriodMS)
		}
	}
	return nil
}

func checkOrder(order []int, n int) error {
	if len(order) != n {
		return fmt.Errorf("%d entries, want %d", len(order), n)
	}
	seen := make([]bool, n)
	for _, i := range order {
		if i < 0 || i >= n || seen[i] {
			return fmt.Errorf("entry %d", i)
		}
		seen[i] = true
	}
	return nil
}

func oneOf(s string, options ...string) bool {
	for _, o := range options {
		if s == o {
			return true
		}
	}
	return false
}

// generatorControlNamed is the control a description names: "name", "back",
// "forward", "hero/i", "level/i", "skill/i", "minus/i", "plus/i", "reset" or
// "play".
func generatorControlNamed(name string) (chargenControl, bool) {
	switch name {
	case "name":
		return chargenName, true
	case "back":
		return chargenBack, true
	case "forward":
		return chargenForward, true
	case "reset":
		return chargenReset, true
	case "play":
		return chargenPlay, true
	}
	kind, number, found := strings.Cut(name, "/")
	i, err := strconv.Atoi(number)
	if !found || err != nil {
		return chargenNone, false
	}
	for _, k := range []struct {
		kind  string
		first chargenControl
		count int
	}{{"hero", chargenChoice0, generatorHeroes}, {"level", chargenLevel0, generatorLevels}, {"skill", chargenSkill0, generatorSkills},
		{"minus", chargenStatMinus0, generatorStats}, {"plus", chargenStatPlus0, generatorStats}} {
		if k.kind == kind && i >= 0 && i < k.count {
			return k.first + chargenControl(i), true
		}
	}
	return chargenNone, false
}

// ms is a description's millisecond count as a duration.
func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

// ResetFor is the reset rule for an install language: the rule naming it, else
// the rule naming no language.
func (d *GeneratorDetail) ResetFor(language string) GeneratorReset {
	var fallback GeneratorReset
	for _, r := range d.Reset {
		if r.Language == language {
			return r
		}
		if r.Language == "" {
			fallback = r
		}
	}
	return fallback
}
