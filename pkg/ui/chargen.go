package ui

import (
	"fmt"
	"image"
	"time"

	"againrom/pkg/render/text"
)

// The generation screen's model: a setup handed down by the wiring tier, and
// the pure state machine that turns a sequence of moves into a spread the
// wiring tier can accept.
//
// Every label, bound, cost and option this file shows is an ARGUMENT:
// nothing here names a statistic, a skill or a class, and pointing the
// screen at a different table is a change to the setup builder alone, never
// to this one. The package's import list does not grow to build it -- in
// particular not against pkg/data, which the import-graph test in
// internal/archtest enforces (AC-11) -- so every string the screen shows has
// already been resolved by the caller.
//
// The model is pure: no engine type, no clock, no randomness, no archive.
// Every one of its states is therefore decidable by calling its exported
// methods in a test, which is what makes AC-3 -- "no sequence of steps can
// leave the screen with an illegal spread" -- a claim a test can actually
// prove rather than merely sample.

// ChargenSetup is everything the screen needs to run, resolved by the wiring
// tier from the definition table and the decoded cost curve.
//
// Cost is indexed DIRECTLY by a statistic's value -- Cost[v] is the
// cumulative cost of standing at v, the same table pkg/data's PointCost
// produces one entry of at a time -- not by an offset from a floor. That is
// what lets one table serve every statistic row regardless of where its own
// Floor happens to sit.
type ChargenSetup struct {
	Title   string
	Name    string
	Choices []ChargenChoice
	Stats   []ChargenStat
	Cost    []int
	Budget  int
	Confirm string

	// PreCreate makes this model use the two-stage generator. A nil value keeps
	// the old diagnostic model available to callers that construct a small,
	// headless setup in tests.
	PreCreate *ChargenPreCreate
	// Presets follow the four pre-create portraits. Missing/invalid entries
	// keep the generic starts used by asset-free/custom generator setups.
	Presets [4][]int
	// EncodeName turns a typed rune into the install's stored byte. Nil retains
	// the legacy UTF-8 test seam; real setups always supply it.
	EncodeName func(rune) (byte, bool)

	// Detailed is the source-backed wording used after Forward. It is nil for
	// the legacy diagnostic screen.
	Detailed *ChargenDetailed
	// Preview projects a legal detailed draft without opening a mission. The
	// callback owns the game-side derivation; this model only replaces its
	// complete cached answer after an accepted edit.
	Preview func(ChargenResult) ChargenPreview

	// Derive is what the CURRENT spread would buy, answered by the wiring
	// tier for a result this model has already accepted. It is the same
	// shape Confirm's own begin callback takes at the App seam -- a
	// function this package calls and never a computation it performs --
	// and it is a field of the setup rather than a third argument to
	// OpenChargen so that the one place the wiring tier states what the
	// screen needs stays one place, and so the call site does not move.
	//
	// IT MAY BE NIL, and every setup built before this field existed leaves
	// it so: DerivedText then answers nothing and the screen paints exactly
	// what it painted before. That is not a courtesy to old callers, it is
	// what keeps this file's own promise that a screen with a malformed or
	// half-filled setup goes quiet rather than breaking.
	//
	// IT IS CALLED ONLY WITH Result's OWN OUTPUT, so it never sees an
	// illegal spread: there is nothing to derive from a spread generation
	// could not have produced, and showing the last legal numbers instead
	// would be worse than showing none -- a player would read a stale
	// health as the one he is about to get.
	Derive func(ChargenResult) []ChargenDerived

	// TipSelect are the pre-create popup's three step texts, chrsel1..3
	// (TOWN-518). TipText and TipTextMage are the detailed page's enter
	// texts, chrgen1f and chrgen1m, chosen by the hero's class, and
	// TipTextDetail is chrgen2, its text after the first skill click
	// (TOWN-522). An empty text draws no popup.
	TipSelect            [3]string
	TipText, TipTextMage string
	TipTextDetail        string
	// TipClose and TipToggle are main.txt[127] and main.txt[128], already
	// resolved by the game side. Empty keeps the exact EN fallback for a
	// diagnostic setup assembled without an install.
	TipClose, TipToggle string
	// TipArt is the panel's own shared art (tippanel.go), resolved once by
	// the wiring tier on shopArt's own precedent.
	TipArt *TipPanelArt
	// TipsOn is TipsMode (MENU-135) as the setup was built; the popup
	// checkbox writes it through SetTipsOn.
	TipsOn bool
	// SetTipsOn persists a toggle press across the seam this package may not
	// cross itself: nil in a hand-built test setup leaves the toggle a no-op,
	// on Preview/Derive's own "may be nil" precedent.
	SetTipsOn func(bool)
}

// ChargenStage identifies the visible half of the generator.
type ChargenStage uint8

const (
	PreCreateStage ChargenStage = iota
	DetailedStage
)

// ChargenPreCreate is the resolved, install-backed portion of the setup. The
// drawing data itself lives in ChargenPresentation so the state model has no
// image or archive dependency.
type ChargenPreCreate struct {
	Prompt string
	Back   string
	Art    *ChargenPresentation
	// HeroNames are the name field's per-picture texts in preChoiceParts'
	// picture order (TEXT-074), and Unnamed is the text the field is seeded
	// with when no name was given (TEXT-073). Together they are the default
	// texts a hero press and the page's enter rewrite; an empty one is none.
	HeroNames [4]string
	Unnamed   string
}

// ChargenDetailed is the source wording and hover text for the detailed page.
type ChargenDetailed struct {
	Back, Reset, Play       string
	EmptyName, ReservedName string
	SkillHover              [2][5]string
}

// ChargenPreview is one atomically replaced detailed-page projection. Doll is
// nil when its required base image cannot be read.
type ChargenPreview struct {
	Subject PanelSubject
	Doll    image.Image
}

// ChargenDerived is one consequence of the current spread: a label and a
// value, BOTH ALREADY RENDERED by the wiring tier.
type ChargenDerived struct {
	Name  string
	Value string
}

// ChargenChoice is one cycling row: a sex, a class, a skill.
//
// A row is FLAT when Parent is -1: Options is its whole option list, fixed
// for the row's life. A row is DEPENDENT when Parent is a valid index into
// the setup's own Choices: its live option list is OptionsFor, chosen by the
// parent row's current option index. A later screen making a different row
// dependent declares it the same way and changes nothing here.
type ChargenChoice struct {
	Name       string
	Options    []string
	OptionsFor [][]string
	Parent     int

	// Start is the row's OWN OPENING index into its current option list —
	// where it shows on construction, not a floor it is held to afterwards. It
	// exists for exactly one caller in this story: a command-line flag (-skill)
	// already sets the field the skill row would otherwise choose, and the
	// screen must not become a second writer of it, so the wiring tier SEEDS
	// the row at the option matching the flag's own current value instead. The
	// player's own first Adjust still moves it from there exactly as if Start
	// had been 0 -- this field only ever affects the model's very first read.
	//
	// It is CLAMPED, by NewChargen at construction, exactly as a dependent
	// row's held index already is when its own parent moves (see
	// reclampDependents): an index at or past the option count lands on the
	// last option, never resets to 0 as though the list had emptied under
	// it. A NEGATIVE Start clamps to the first option instead -- the one
	// case reclampDependents never needs, because a cycled index can never
	// go negative, but a caller-supplied Start carries no such guarantee.
	Start int
}

// ChargenStat is one stepping row: a floor, a ceiling and a starting value,
// all supplied rather than assumed, so the screen holds no bound of its own.
type ChargenStat struct {
	Name    string
	Floor   int
	Ceiling int
	Start   int
}

// ChargenResult is a generated character before the wiring tier turns it into
// a party: one option index per choice row, one value per statistic row, both
// in the same order the setup declared them.
type ChargenResult struct {
	// 1=Easy, 2=Normal, 3=Hard. Zero is accepted by legacy game callers as
	// Normal. UI emits the selected level, never the legacy sentinel.
	Difficulty int
	Name       string
	Choices    []int
	Stats      []int
}

const chargenMissingCost = 1_000_000

// Chargen is the generation screen's model: which row has the focus, which
// option each choice row currently shows, and what each statistic currently
// stands at.
//
// A malformed ChargenSetup -- a Cost table shorter than a Ceiling, a Parent
// naming a row that does not exist, a Floor above its own Ceiling, an option
// list that is empty -- must not panic and must not index out of range: a
// screen that stops responding is a bug report, a screen that crashes is an
// incident. Every method below states, in its own doc, what it does instead.
// The general shape of the answer is always the same: a row nothing can make
// sense of is a row that never moves, and every read stays inside the bounds
// of the slice it reads.
type Chargen struct {
	sparkAt      time.Time
	sparkElapsed time.Duration
	setup        ChargenSetup
	name         string
	lastPressed  int // the picture the last hero press chose (TEXT-074)
	caret        nameCaret
	choiceIndex  []int // current option index, one per Choices row
	statValue    []int // current value, one per Stats row
	focus        int   // row index into the combined list, choices then stats
	stage        ChargenStage
	preChoice    int
	preLevel     int // three-picture difficulty index; starts at Normal (1)
	preview      ChargenPreview
	// The two pages' tip state: whether each page's popup exists and its
	// step (TOWN-518, TOWN-522), and the guided-cycle highlight the last
	// step answered, -1 for none (TOWN-519).
	preTip, detailTip   bool
	preStep, detailStep int
	cycleDraw           int
	// statHeld is whether the left button is held this frame (MENU-138).
	statHeld bool

	// message is the detailed page's own transient hover/refusal copy (1022
	// round-2), set by SetDetailMessage and drawn by
	// composeChargenDetailedPage's own drawChargenMessage call. It replaces
	// spec B4's dropped `chargenDetailMessageBox`: B4 removed the black
	// rectangle the box painted, not the copy itself, and this field is the
	// channel that copy needed and did not have (DIV-192).
	message string
}

// SetDetailMessage records the detailed page's own transient hover/refusal
// line for the next composition. It is the App's own responsibility to call
// this every frame with whatever a.chargenDetailMessage() currently answers
// (pkg/ui/app.go's composeChargenScreen) — Chargen itself tracks no hover or
// refusal state of its own, exactly as before this field existed.
func (c *Chargen) SetDetailMessage(msg string) {
	if c == nil {
		return
	}
	c.message = msg
}

// NewChargen builds the model at its setup's own starting point: every
// choice row opens at its own Start, CLAMPED into its current option list
// (startIndex below); every statistic row at its own Start, UNCLAMPED; the
// focus on row 0.
//
// A STATISTIC ROW'S START IS NOT VALIDATED, the same choice NewPicker makes
// over its own rows: an out-of-range Start or an inverted Floor/Ceiling is
// not rejected here, it plays out wherever it first matters (Legal, Adjust),
// and every place it can matter is documented at that place.
//
// CHOICE ROWS ARE RESOLVED IN THEIR OWN SLICE ORDER, index 0 upward, so a
// dependent row's Start is clamped against the option list its PARENT's own
// (already-resolved) Start selects -- not against every list the parent
// could ever show. A setup that declares a dependent row before the row it
// depends on is reading its parent's zero value early, the same as any other
// malformed setup this package only promises not to panic over.
func NewChargen(setup ChargenSetup) *Chargen {
	c := &Chargen{
		preLevel:    1,
		cycleDraw:   -1,
		setup:       setup,
		name:        setup.Name,
		choiceIndex: make([]int, len(setup.Choices)),
		statValue:   make([]int, len(setup.Stats)),
	}
	if setup.PreCreate != nil {
		c.stage = PreCreateStage
		c.enterPreCreate()
	} else {
		c.stage = DetailedStage
		c.enterDetailed()
	}
	for i := range setup.Choices {
		c.choiceIndex[i] = c.startIndex(i)
	}
	for i, s := range setup.Stats {
		c.statValue[i] = s.Start
	}
	return c
}

// Stage reports which half of the generator is active.
func (c *Chargen) Stage() ChargenStage {
	if c == nil {
		return DetailedStage
	}
	return c.stage
}

// PreChoice reports the selected combined class-and-sex picture. It is always
// in the four-picture range, even for a legacy one-stage setup.
func (c *Chargen) PreChoice() int {
	if c == nil {
		return 0
	}
	return c.preChoice
}

// SelectPreChoice chooses one of male fighter, male mage, female fighter and
// female mage, in that order. It has effect only on the pre-create stage.
//
// It is a hero press (TEXT-074): when the picture differs from the last
// pressed one and the field holds a default text, the field takes the
// picture's own name. A typed name survives every press, and the press is
// remembered whether or not it wrote. The page presses on the left
// button-down and on keyboard Enter (DIV-1495).
func (c *Chargen) SelectPreChoice(choice int) {
	if c == nil || c.stage != PreCreateStage || choice < 0 || choice > 3 {
		return
	}
	c.preChoice = choice
	if pre := c.setup.PreCreate; pre != nil && choice != c.lastPressed && c.defaultName() && pre.HeroNames[choice] != "" {
		c.name = pre.HeroNames[choice]
	}
	c.lastPressed = choice
}

// enterPreCreate is the page's enter (TEXT-073), run when the generator opens
// and when the detailed page's Back returns: the first picture is lit and
// counts as the last pressed, and a default text becomes the first picture's
// name. A typed name is kept. The difficulty is not touched.
func (c *Chargen) enterPreCreate() {
	c.preChoice, c.lastPressed = 0, 0
	c.preTip, c.preStep, c.cycleDraw = c.setup.TipsOn, 0, -1
	if pre := c.setup.PreCreate; pre != nil && c.defaultName() && pre.HeroNames[0] != "" {
		c.name = pre.HeroNames[0]
	}
}

// defaultName reports whether the field holds one of the default texts
// ChargenPreCreate names, compared byte for byte.
func (c *Chargen) defaultName() bool {
	pre := c.setup.PreCreate
	if pre == nil {
		return false
	}
	if pre.Unnamed != "" && c.name == pre.Unnamed {
		return true
	}
	for _, s := range pre.HeroNames {
		if s != "" && c.name == s {
			return true
		}
	}
	return false
}

// SelectDifficulty changes only the pre-create draft. Forward, Back and the
// detailed stat Reset preserve it; a newly opened generator starts at Normal.
func (c *Chargen) SelectDifficulty(level int) {
	if c != nil && c.stage == PreCreateStage && level >= 0 && level < 3 {
		c.preLevel = level
	}
}

func (c *Chargen) Difficulty() int {
	if c == nil {
		return 2
	}
	return c.preLevel + 1
}

// preChoiceParts is which sex option and which class option one combined
// picture stands for: male fighter, male mage, female fighter, female mage, in
// that order.
//
// IT IS A FUNCTION AND NOT A SWITCH INSIDE Forward because the headless
// generation surface reports the same identity as a label. The label a
// scenario matches against and the identity Forward commits are then one
// statement read twice; two copies of a four-arm switch is exactly the shape
// that drifts when a fifth picture is added.
func preChoiceParts(choice int) (sex, class int) {
	switch choice {
	case 1:
		return 0, 1
	case 2:
		return 1, 0
	case 3:
		return 1, 1
	}
	return 0, 0
}

// Forward enters a fresh detailed draft without validating the name.
func (c *Chargen) Forward() {
	if c == nil || c.stage != PreCreateStage {
		return
	}
	sex, class := preChoiceParts(c.preChoice)
	if len(c.choiceIndex) > 0 {
		c.choiceIndex[0] = sex
	}
	if len(c.choiceIndex) > 1 {
		c.choiceIndex[1] = class
	}
	c.resetStats()
	if len(c.choiceIndex) > 2 {
		c.choiceIndex[2] = c.startIndex(2)
	}
	c.focus = 0
	c.stage = DetailedStage
	c.preTip = false
	c.enterDetailed()
	c.rebuildPreview()
}

// enterDetailed is the detailed page's enter (TOWN-522): the popup exists
// while TipsMode is set, and its step restarts at 0.
func (c *Chargen) enterDetailed() {
	c.detailTip, c.detailStep, c.cycleDraw = c.setup.TipsOn, 0, -1
}

// Back returns true when detailed state was discarded and pre-create is now
// active. A false return means the caller should leave the generator itself.
// The pre-create page is entered again, so the first picture is lit and a
// default name becomes its name (TEXT-073).
func (c *Chargen) Back() bool {
	if c == nil || c.stage != DetailedStage || c.setup.PreCreate == nil {
		return false
	}
	c.stage = PreCreateStage
	c.detailTip = false
	c.enterPreCreate()
	c.focus = 0
	c.preview = ChargenPreview{}
	return true
}

// Reset restores the detailed fields to their setup starts. It is deliberately
// idempotent and does not alter the stored pre-create identity.
func (c *Chargen) Reset() {
	if c == nil || c.stage != DetailedStage {
		return
	}
	c.resetStats()
	c.rebuildPreview()
}

func (c *Chargen) resetStats() {
	for i, s := range c.setup.Stats {
		c.statValue[i] = s.Start
	}
	if c.preChoice < 0 || c.preChoice >= len(c.setup.Presets) {
		return
	}
	preset := c.setup.Presets[c.preChoice]
	if len(preset) != len(c.statValue) {
		return
	}
	for i, value := range preset {
		if value < c.setup.Stats[i].Floor || value > c.setup.Stats[i].Ceiling {
			return
		}
	}
	before := append([]int(nil), c.statValue...)
	copy(c.statValue, preset)
	if !c.Legal() {
		copy(c.statValue, before)
	}
}

// SelectSkill chooses a detailed-stage position directly. It returns whether
// the selected position changed and therefore caused a preview replacement.
func (c *Chargen) SelectSkill(skill int) bool {
	if c == nil || c.stage != DetailedStage || len(c.choiceIndex) <= 2 {
		return false
	}
	opts := c.choiceOptions(2)
	if skill < 0 || skill >= len(opts) || c.choiceIndex[2] == skill {
		return false
	}
	c.choiceIndex[2] = skill
	c.rebuildPreview()
	return true
}

// AdjustStat changes one named spread row directly. It returns true only for
// an accepted point-buy step.
func (c *Chargen) AdjustStat(stat, delta int) bool {
	if c == nil || c.stage != DetailedStage || stat < 0 || stat >= len(c.statValue) {
		return false
	}
	before := c.statValue[stat]
	c.adjustStat(stat, delta)
	if c.statValue[stat] == before {
		return false
	}
	c.rebuildPreview()
	return true
}

// StatStepText is the generator's `%+d` text for one statistic's step button
// (TOWN-469): the raise cost negated, so a raise reads negative, or the
// lowering refund unchanged, so a lowering reads positive, grouped.
func (c *Chargen) StatStepText(stat int, raise bool) (string, bool) {
	if c == nil || stat < 0 || stat >= len(c.statValue) {
		return "", false
	}
	v := c.statValue[stat]
	if raise {
		return GroupSigned(-int64(c.costAt(v+1) - c.costAt(v))), true
	}
	return GroupSigned(int64(c.costAt(v) - c.costAt(v-1))), true
}

// Preview reports the current complete detailed projection.
func (c *Chargen) Preview() ChargenPreview {
	if c == nil {
		return ChargenPreview{}
	}
	return c.preview
}

func (c *Chargen) rebuildPreview() {
	if c == nil || c.setup.Preview == nil || c.stage != DetailedStage {
		return
	}
	res, ok := c.Result()
	if !ok {
		return
	}
	c.preview = c.setup.Preview(res)
}

// startIndex is choice row i's own Start, clamped into the row's CURRENT
// option list -- the same clamp reclampDependents applies when a parent's
// move leaves a held index out of range for its new list: an empty option
// list answers 0, an index at or past the option count clamps to the last
// option. It additionally clamps a NEGATIVE Start to the first option, which
// reclampDependents never has to: a cycled index can never go negative, but
// Start arrives from the setup rather than from a cycle.
func (c *Chargen) startIndex(i int) int {
	opts := c.choiceOptions(i)
	n := len(opts)
	if n == 0 {
		return 0
	}
	start := c.setup.Choices[i].Start
	switch {
	case start < 0:
		return 0
	case start >= n:
		return n - 1
	default:
		return start
	}
}

// Rows reports how many rows the screen has: every choice row, then every
// statistic row.
func (c *Chargen) Rows() int { return len(c.setup.Choices) + len(c.setup.Stats) }

// nameCap is the name field's byte cap for typing (TEXT-075).
const nameCap = 10

// EditName applies typed characters and one backspace edge (TEXT-075). The
// field only appends: under the cap a character's byte goes to the end unless
// it is below 0x20, and backspace removes the last byte. Nothing selects or
// replaces, so the first keystroke extends a default name. A character under
// the cap also restarts the caret; one at the cap and backspace do not.
func (c *Chargen) EditName(typed string, backspace bool) {
	if c == nil {
		return
	}
	if backspace && c.name != "" {
		c.name = c.name[:len(c.name)-1]
	}
	for _, r := range typed {
		if len(c.name) >= nameCap {
			continue
		}
		c.caret.restart = true
		if b, ok := c.encodeName(r); ok && b >= 0x20 {
			c.name += string([]byte{b})
		}
	}
}

// encodeName is a typed rune's stored byte: the setup's install encoder, or
// the single-byte UTF-8 test seam when a setup names none.
func (c *Chargen) encodeName(r rune) (byte, bool) {
	if c.setup.EncodeName != nil {
		return c.setup.EncodeName(r)
	}
	if r >= 0x20 && r < 0x7f {
		return byte(r), true
	}
	return 0, false
}

func (c *Chargen) NameText() string {
	if c == nil {
		return ""
	}
	return c.name
}

// Focus reports the index of the focused row, into the same combined list
// Rows counts.
func (c *Chargen) Focus() int { return c.focus }

// Move shifts the focus by d rows and wraps at both ends, over the WHOLE
// combined list -- choice rows first, then statistic rows -- so moving past
// the last statistic reaches the first choice and moving back from there
// reaches the last statistic. An empty screen (no rows at all) leaves the
// focus untouched rather than wrapping over nothing.
func (c *Chargen) Move(d int) {
	n := c.Rows()
	if c.stage == PreCreateStage && c.setup.PreCreate != nil {
		n = 10 // name, four pictures, Back, Forward, three difficulty pictures
	} else if c.stage == DetailedStage && c.setup.PreCreate != nil {
		n = detailedFocusCount
	}
	if n == 0 {
		return
	}
	c.focus = ((c.focus+d)%n + n) % n
}

// Adjust drives the focused row: a choice row cycles its option index within
// its CURRENT option set, a statistic row steps by d.
//
// A choice row has no bound to test in the same sense -- cycling wraps, so
// there is no "too far" -- its only refusal is an empty current option list,
// which cannot be cycled onto anything and so is left exactly where it stood.
func (c *Chargen) Adjust(d int) {
	if c.stage == DetailedStage && c.setup.PreCreate != nil {
		id := detailedFocusControl(c.focus)
		switch {
		case id >= chargenStatMinus0 && id <= chargenStatMinus3:
			c.AdjustStat(int(id-chargenStatMinus0), d)
		case id >= chargenStatPlus0 && id <= chargenStatPlus3:
			c.AdjustStat(int(id-chargenStatPlus0), d)
		}
		return
	}
	n := c.Rows()
	if n == 0 || c.focus < 0 || c.focus >= n {
		return
	}
	if c.focus < len(c.setup.Choices) {
		c.adjustChoice(c.focus, d)
		return
	}
	c.adjustStat(c.focus-len(c.setup.Choices), d)
}

// adjustChoice cycles choice row i's index by d within its current option
// list, wrapping, and re-clamps whatever depends on it (AC-5).
func (c *Chargen) adjustChoice(i, d int) {
	opts := c.choiceOptions(i)
	n := len(opts)
	if n == 0 {
		return
	}
	next := ((c.choiceIndex[i]+d)%n + n) % n
	if next == c.choiceIndex[i] {
		return
	}
	c.choiceIndex[i] = next
	c.reclampDependents(i)
}

func (c *Chargen) adjustStat(j, d int) {
	if j < 0 || j >= len(c.setup.Stats) {
		return
	}
	stat := c.setup.Stats[j]
	candidate := c.statValue[j] + d
	if candidate < stat.Floor || candidate > stat.Ceiling {
		return
	}
	remaining := c.setup.Budget
	for k, v := range c.statValue {
		if k == j {
			v = candidate
		}
		remaining -= c.costAt(v)
	}
	if remaining < 0 {
		return
	}
	c.statValue[j] = candidate
	if c.stage == DetailedStage && c.setup.PreCreate == nil {
		c.rebuildPreview()
	}
}

// choiceOptions reports choice row i's CURRENT option list.
//
// A flat row (Parent == -1) always answers Options. A dependent row answers
// OptionsFor at its parent's current index -- but a malformed setup can name
// a Parent that does not resolve to a real row, or an OptionsFor shorter than
// the parent's own index reaches, and this is the one place both of those
// have to be survived rather than assumed away. Parent outside [0,
// len(Choices)) is treated exactly like -1 -- a dependency on a row that
// cannot exist is not a dependency at all, flat is the only thing left for it
// to mean -- and a parent index that falls outside OptionsFor answers no
// options at all, which every caller of this method already treats as "this
// row offers nothing right now" rather than as a special case of its own.
func (c *Chargen) choiceOptions(i int) []string {
	ch := c.setup.Choices[i]
	if ch.Parent < 0 || ch.Parent >= len(c.setup.Choices) {
		return ch.Options
	}
	pi := c.choiceIndex[ch.Parent]
	if pi < 0 || pi >= len(ch.OptionsFor) {
		return nil
	}
	return ch.OptionsFor[pi]
}

// reclampDependents fixes up every row whose option list can have changed
// because row changed's index just did, walking the Parent relation
// breadth-first from there (AC-5).
//
// A dependent row's index survives a parent change when it still names a
// real option in the new list; when the new list is SHORTER than the held
// index, it is clamped to the new last option rather than reset to 0.
//
// The walk is breadth-first with a visited guard rather than a direct
// recursion on Parent, because Parent is caller-supplied and nothing stops a
// malformed setup from building a cycle (row 0 depends on row 1 which depends
// on row 0). The guard is what keeps that a screen nobody would author rather
// than an infinite loop.
func (c *Chargen) reclampDependents(changed int) {
	visited := make([]bool, len(c.setup.Choices))
	queue := []int{changed}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if p < 0 || p >= len(visited) || visited[p] {
			continue
		}
		visited[p] = true
		for i, ch := range c.setup.Choices {
			if ch.Parent != p {
				continue
			}
			opts := c.choiceOptions(i)
			switch {
			case len(opts) == 0:
				if c.choiceIndex[i] != 0 {
					c.choiceIndex[i] = 0
					queue = append(queue, i)
				}
			case c.choiceIndex[i] >= len(opts):
				c.choiceIndex[i] = len(opts) - 1
				queue = append(queue, i)
			}
		}
	}
}

// costAt is Cost[v], total over every int v: out of the table's bounds it
// answers chargenMissingCost rather than indexing out of range.
func (c *Chargen) costAt(v int) int {
	if v < 0 || v >= len(c.setup.Cost) {
		return chargenMissingCost
	}
	return c.setup.Cost[v]
}

// Remaining is the budget less the sum of every statistic row's cumulative
// cost at its current value.
//
// It is NEVER CLAMPED. A clamp at 0 would erase the one distinction the
// screen exists to show: "spent it all" reads the same as "cannot be
// generated" unless a spread over budget is allowed to say so by going
// negative, which is exactly what a spread over budget is (spec Terms).
func (c *Chargen) Remaining() int {
	r := c.setup.Budget
	for _, v := range c.statValue {
		r -= c.costAt(v)
	}
	return r
}

// Legal reports whether the current spread is one generation could have
// produced: every statistic inside its own [Floor, Ceiling], and Remaining
// not negative. It says nothing about the choice rows -- the spec's own
// "spread" (Terms) names the four statistics alone, and widening Legal to
// also fail on an exhausted choice row (an empty option list, itself only
// reachable from a malformed setup) would be this package inventing a rule
// the spec does not state.
func (c *Chargen) Legal() bool {
	for i, stat := range c.setup.Stats {
		v := c.statValue[i]
		if v < stat.Floor || v > stat.Ceiling {
			return false
		}
	}
	return c.Remaining() >= 0
}

// Result reports the current choice indices and statistic values, and
// whether they are fit to hand to the wiring tier.
func (c *Chargen) Result() (ChargenResult, bool) {
	if !c.Legal() {
		return ChargenResult{}, false
	}
	choices := append([]int(nil), c.choiceIndex...)
	stats := append([]int(nil), c.statValue...)
	return ChargenResult{Name: c.name, Choices: choices, Stats: stats, Difficulty: c.Difficulty()}, true
}

// TipPanel is the showing page's tip popup: pre-create at PreCreateTipRect
// with its step's chrsel text (TOWN-518), the detailed page at ChargenTipRect
// with chrgen1f or chrgen1m, then chrgen2 after the first skill click
// (TOWN-522, MENU-137).
func (c *Chargen) TipPanel() TipPanelView {
	if c == nil || c.setup.TipArt == nil {
		return TipPanelView{}
	}
	var font *text.Font
	if c.setup.PreCreate != nil && c.setup.PreCreate.Art != nil {
		font = c.setup.PreCreate.Art.Font
	}
	rect, tipText := ChargenTipRect, ""
	switch {
	case c.stage == PreCreateStage && c.preTip && c.preStep >= 0 && c.preStep < len(c.setup.TipSelect):
		rect, tipText = PreCreateTipRect, c.setup.TipSelect[c.preStep]
	case c.stage == DetailedStage && c.detailTip && c.detailStep != 0:
		tipText = c.setup.TipTextDetail
	case c.stage == DetailedStage && c.detailTip:
		tipText = c.setup.TipText
		if len(c.choiceIndex) > 1 && c.choiceIndex[1] != 0 && c.setup.TipTextMage != "" {
			tipText = c.setup.TipTextMage
		}
	}
	if tipText == "" {
		return TipPanelView{}
	}
	return TipPanelView{
		Rect:        rect,
		Text:        tipText,
		ToggleOn:    c.setup.TipsOn,
		CloseLabel:  c.setup.TipClose,
		ToggleLabel: c.setup.TipToggle,
		Art:         c.setup.TipArt,
		Font:        font,
	}
}

// CloseTip deletes the showing page's popup (MENU-137). The page's next
// enter builds it again while TipsMode is set.
func (c *Chargen) CloseTip() {
	if c == nil {
		return
	}
	if c.stage == PreCreateStage {
		c.preTip = false
	} else {
		c.detailTip = false
	}
	c.cycleDraw = -1
}

// TipStep is the showing page's tip step: 0..2 on pre-create, 0..1 on the
// detailed page.
func (c *Chargen) TipStep() int {
	if c == nil {
		return 0
	}
	if c.stage == PreCreateStage {
		return c.preStep
	}
	return c.detailStep
}

// tipPortraitClicked and tipLevelClicked are the pre-create step routine
// (TOWN-518): a pointer click on a portrait at step 0 or on a level at step 1
// advances the step while TipsMode is set and the popup exists.
func (c *Chargen) tipPortraitClicked() { c.advancePreTip(0) }
func (c *Chargen) tipLevelClicked()    { c.advancePreTip(1) }

func (c *Chargen) advancePreTip(from int) {
	if c != nil && c.stage == PreCreateStage && c.setup.TipsOn && c.preTip && c.preStep == from {
		c.preStep = from + 1
	}
}

// tipSkillClicked is the detailed step routine (TOWN-522): the first skill
// click retexts chrgen2 while TipsMode is set and the popup exists.
func (c *Chargen) tipSkillClicked() {
	if c != nil && c.stage == DetailedStage && c.setup.TipsOn && c.detailTip && c.detailStep == 0 {
		c.detailStep = 1
	}
}

// ToggleTips flips the permanent suppression store through the callback the
// wiring tier supplied (spec behaviour 4): this package has no store of its
// own to write, only the setup's cached reading and a door to change it. A
// setup with no SetTipsOn (a hand-built test setup) leaves the press a
// no-op, matching Preview/Derive's own nil-callback shape.
func (c *Chargen) ToggleTips() {
	if c == nil || c.setup.SetTipsOn == nil {
		return
	}
	c.setup.TipsOn = !c.setup.TipsOn
	c.setup.SetTipsOn(c.setup.TipsOn)
}

// chargenHelp is the screen's own navigation chrome: how to drive it.
const chargenHelp = "up/down: focus   left/right: adjust   enter: confirm   esc: back"

// HeaderText is the screen's one-line header: its title, from the setup, and
// how to drive it.
func (c *Chargen) HeaderText() string {
	return c.setup.Title + "   " + chargenHelp
}

// FooterText is the screen's one-line footer: what remains of the budget, and
// the confirm label the setup supplied. The two are printed together because
// they answer the same question a player has at the moment they reach for
// enter -- can I confirm, and what does confirm do -- not because either one
// is derived from the other.
func (c *Chargen) FooterText() string {
	return fmt.Sprintf("Remaining: %d   %s", c.Remaining(), c.setup.Confirm)
}

// chargenDerivedTitle and chargenDerivedNone are the consequence block's own
// chrome: generic screen words on chargenHelp's own precedent, never a game
// label. The second one is what stands where the numbers would, and it says
// the spread is not payable rather than showing a number that is not the
// player's -- Legal's own two clauses, in one sentence, because the screen
// has no room to say which of them failed and the answer is the same either
// way: move something back.
const (
	chargenDerivedTitle = "Derived"
	chargenDerivedNone  = "  (nothing to derive: this spread is not one generation could produce)"
)

// DerivedText is the consequence block the draw path paints under the
// footer: a title line, then one indented line per consequence the wiring
// tier reports for the CURRENT spread.
//
// It answers NOTHING AT ALL -- a nil slice, not an empty title -- when the
// setup names no Derive, and when a Derive answers no consequences. A lone
// heading over nothing is a screen element that says only that something is
// missing, which is worse than the blank line it replaces.
//
// WHILE THE SPREAD IS ILLEGAL it answers the title and one fixed line, and
// never calls Derive at all. Result is the gate, so this method inherits its
// refusal rather than restating it: there is exactly one place in this file
// that decides whether the current state is fit to be read as a character,
// and this is a second reader of that decision, not a second decision.
//
// The two-space indent is RowText's own marker column, so a consequence line
// and an unfocused row start at the same column.
func (c *Chargen) DerivedText() []string {
	if c.setup.Derive == nil {
		return nil
	}
	res, ok := c.Result()
	if !ok {
		return []string{chargenDerivedTitle, chargenDerivedNone}
	}
	derived := c.setup.Derive(res)
	if len(derived) == 0 {
		return nil
	}
	out := make([]string, 0, len(derived)+1)
	out = append(out, chargenDerivedTitle)
	for _, d := range derived {
		out = append(out, fmt.Sprintf("  %s: %s", d.Name, d.Value))
	}
	return out
}

// RowText renders row i as the draw path shows it: a focus marker, then the
// row's own text -- "Name: option" for a choice row, "Name: value" for a
// statistic row. An index outside [0, Rows()) answers the empty string, the
// same total-and-silent shape as an out-of-range read anywhere else in this
// file.
//
// The marker matches picker.go's own RowText exactly -- two columns whether
// or not the row is focused, so the text of every row starts at the same
// column and the list does not jitter as the focus moves.
func (c *Chargen) RowText(i int) string {
	if i < 0 || i >= c.Rows() {
		return ""
	}
	marker := "  "
	if i == c.focus {
		marker = "> "
	}
	if i < len(c.setup.Choices) {
		return marker + c.choiceRowText(i)
	}
	return marker + c.statRowText(i-len(c.setup.Choices))
}

// choiceRowText renders choice row i's own content, without the marker: its
// name and its current option, or nothing after the colon when the row's
// current option list is empty (a malformed setup's doing -- see
// choiceOptions).
func (c *Chargen) choiceRowText(i int) string {
	ch := c.setup.Choices[i]
	opts := c.choiceOptions(i)
	idx := c.choiceIndex[i]
	val := ""
	if idx >= 0 && idx < len(opts) {
		val = opts[idx]
	}
	return fmt.Sprintf("%s: %s", ch.Name, val)
}

// statRowText renders statistic row j's own content, without the marker: its
// name and its current value.
func (c *Chargen) statRowText(j int) string {
	stat := c.setup.Stats[j]
	return fmt.Sprintf("%s: %d", stat.Name, c.statValue[j])
}
