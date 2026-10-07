package game

// The mission stage of the headless scenario language (0155). A scenario on this
// stage names a campaign mission, orders the units on it, waits for what it
// ordered to happen, and states what must be true — with no window, no
// controller and no save file anywhere in the way.
//
// EVERYTHING HERE IS DEFINED OVER *sim.World AND A REFERENCE TABLE, and
// nothing here opens a file. That is what makes the same scenario language
// runnable three ways: against a lawful install through
// StartScenarioMission, against either preserved root by changing only the
// asset root, and against a world built in test code with no install present
// at all (golden rule 2).

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/vfs"
)

// PlayWorld is the mission a mission-stage scenario drives: the world, and the
// two tables that turn a written reference into an entity.
//
// IT IS A PLAIN STRUCT WITH EXPORTED FIELDS rather than something only a
// mission start can produce. A test builds one over a hand-made sim.World in
// four lines, which is the whole reason the scenario language is testable
// without a game install.
type PlayWorld struct {
	// World is the simulation the steps advance and read.
	World *sim.World

	// Script maps the identifier the map's own script uses for a unit to the
	// entity it became. It is what a "uNN" reference resolves through, and it
	// is empty for a world with no map behind it.
	Script map[uint16]sim.EntityID

	// Party is the entity each party slot became, in start order, which is
	// what a "pN" reference resolves through.
	Party []sim.EntityID

	// reached is what the drive actually walked into. See playCensus.
	reached playCensus

	// primary is the starting hero's entity, when the mission marks one.
	primary   sim.EntityID
	primaryOK bool

	// The announcement half. ann derives raises from the world's own latch
	// array; mission, src and audience are what turns a raised number into the
	// text a player would have read; announced is every one so far, in the
	// order raised.
	//
	// ALL FOUR ARE ABSENT ON A SYNTHETIC WORLD, which has no map script and so
	// raises nothing. A nil announcer samples to nothing, which is the right
	// answer rather than a guard.
	ann           *Announcer
	mission       int
	src           entrySource
	audience      EventAudience
	eventAudience func(int) EventAudience
	announced     []HeadlessAnnouncement
}

// The two asset sources a scenario can declare.
//
// The split is on where the BYTES come from, not on what the scenario says. A
// scenario states what happens; the provider states what it happens to. The two
// prove different things and neither substitutes for the other:
//
//   - AssetsSynthetic proves MECHANISM. The rule fired, the state changed, the
//     order is right. It cannot prove this build reads the original's bytes
//     correctly, because the bytes were written here from the same belief the
//     decode holds. A green synthetic suite over a wrong decode is green.
//   - AssetsInstall proves FIDELITY OF INGESTION. The real archive opens, the
//     real map loads, the counts come out. It cannot run where no install is
//     present, so it never runs under `go test`.
//
// Neither states what the original game does. Only a research claim does.
const (
	// AssetsInstall reads a lawful game root. It is the default.
	AssetsInstall = "install"
	// AssetsSynthetic runs on the world the scenario itself authors.
	AssetsSynthetic = "synthetic"
)

// AssetSource is where this scenario's bytes come from, with the default applied.
func (s HeadlessScenario) AssetSource() string {
	name := strings.ToLower(strings.TrimSpace(s.Assets))
	if name == "" {
		return AssetsInstall
	}
	return name
}

// playCensus is what a driven run REACHED, as opposed to what the compiled
// script contains.
//
// pipeline/check-milestone.sh counts script arms this build cannot run by
// compiling each map and walking the compiled arrays. That is a static count and
// it includes arms on branches no party ever walks. This counts the arms a
// PLAYED mission actually arrived at: an unsupported instant in a trigger that
// fired, and an unsupported check the pass evaluated. The reached count is the
// one that blocks a playthrough, and it is strictly smaller than and derivable
// from neither the static count nor the outcome.
//
// unresolved is counted beside them because it is the dangerous silence: a check
// whose unit reference did not resolve writes no register, inertness is derived
// from unimplemented checks alone, so a live trigger then compares whatever the
// register last held against an authored value.
type playCensus struct {
	instants   map[int32]int
	checks     map[int32]int
	unresolved map[int32]int
}

func (c *playCensus) observe(tr *sim.ScriptTrace) {
	for _, f := range tr.Firings {
		for _, in := range f.Instants {
			if in.Supported {
				continue
			}
			if c.instants == nil {
				c.instants = map[int32]int{}
			}
			c.instants[in.Op]++
		}
	}
	for _, s := range tr.Silent {
		switch s.Why {
		case sim.ScriptSilenceUnsupported:
			if c.checks == nil {
				c.checks = map[int32]int{}
			}
			c.checks[s.Op]++
		default:
			// EVERY OTHER REASON IS AN UNRESOLVED REFERENCE, and the arm is a
			// default rather than a list of the reasons that exist today. A
			// list has to be edited whenever pkg/sim names a new reason, and a
			// reason missing from it is counted NOWHERE — the silence becomes
			// invisible to the census instead of being reported under the
			// wrong heading. ScriptSilenceNoStructure was added and missed
			// exactly that way. Unsupported is the only reason that is not an
			// unresolved reference, and it is the case above.
			if c.unresolved == nil {
				c.unresolved = map[int32]int{}
			}
			c.unresolved[s.Op]++
		}
	}
}

// total is how many unrunnable arms the drive walked into, counting every
// arrival rather than every distinct opcode.
func (c *playCensus) total() int {
	n := 0
	for _, times := range c.instants {
		n += times
	}
	for _, times := range c.checks {
		n += times
	}
	return n
}

// rows is the census as sorted records, so the report is stable between runs.
func (c *playCensus) rows() []HeadlessGapCount {
	var out []HeadlessGapCount
	add := func(kind string, m map[int32]int) {
		ops := make([]int, 0, len(m))
		for op := range m {
			ops = append(ops, int(op))
		}
		sort.Ints(ops)
		for _, op := range ops {
			out = append(out, HeadlessGapCount{Kind: kind, Op: int32(op), Times: m[int32(op)]})
		}
	}
	add("instant", c.instants)
	add("check", c.checks)
	add("unresolved", c.unresolved)
	return out
}

// HeadlessGapCount is one opcode the drive reached and could not run, with how
// many times it arrived there.
type HeadlessGapCount struct {
	Kind  string `json:"kind"`
	Op    int32  `json:"op"`
	Times int    `json:"times"`
}

// step advances the world one tick and records what the script did.
//
// EVERY MISSION-STAGE TICK IS TRACED. A trace is a return value that nothing
// stores on the world and nothing enters in the digest (pkg/sim/scripttrace.go),
// so a traced run and an untraced one are the same run — which is what lets the
// census be unconditional rather than a mode a caller has to remember.
func (p *PlayWorld) step(cmds []sim.Command) {
	p.StepTraced(cmds)
}

// StepTraced advances the production mission-stage driver by one ordinary
// command tick and returns the script observation from that same tick.
//
// The driver still performs its two normal observer duties: it updates the
// unsupported-operation census and samples announcements. The returned trace
// is not stored on the world or the driver. Developer tools can therefore join
// exact authored nodes to ordinary headless play without bypassing the driver
// or adding observer state to the simulation.
func (p *PlayWorld) StepTraced(cmds []sim.Command) sim.ScriptTrace {
	tr := sim.StepTraced(p.World, cmds)
	p.reached.observe(&tr)
	p.sampleAnnouncements()
	return tr
}

// sampleAnnouncements records the announcements this tick raised.
//
// IT IS SAMPLED ONCE PER TICK, at the granularity the world advances at, for
// the reason the windowed driver samples there: a repeating trigger that fires
// and then does not inside one coarser interval has its latch cleared again
// before a coarser sampler could look, and that raise would be lost rather than
// merged.
//
// NOTHING HERE REACHES THE WORLD. The announcer reads the latch array and writes
// to its own memory, and the text read below opens an archive entry that no
// simulation state depends on, so a drive that records and one that does not
// produce the same ticks and the same digests.
func (p *PlayWorld) sampleAnnouncements() {
	for _, e := range p.ann.Sample(p.World) {
		p.announced = append(p.announced, p.announcementOf(e))
	}
}

// announcementOf resolves one raised number the way the windowed driver does:
// the reserved number is not a text, an unshipped file is silence, and the part
// is the one this drive's own hero receives.
func (p *PlayWorld) announcementOf(e int32) HeadlessAnnouncement {
	out := HeadlessAnnouncement{Message: e, Tick: p.World.Tick()}
	if e == ReservedMessageNumber {
		out.Reserved = true
		return out
	}
	payload, ok := ReadEventText(p.src, p.mission, int(e))
	if !ok {
		return out
	}
	out.Shipped = true
	audience := p.audience
	if p.eventAudience != nil {
		audience = p.eventAudience(int(e))
	}
	for n := 1; ; n++ {
		body, ok := EventPart(payload, n, audience)
		if !ok {
			break
		}
		out.Parts = append(out.Parts, decodeInstallText(body))
	}
	out.Shown = len(out.Parts) > 0
	return out
}

// watchAnnouncements gives a mission-stage world what it needs to turn a
// raised number into the text a player would have read.
//
// It is separate from NewPlayWorld because the three inputs are the CALLER's:
// the archive the mission was opened from, and the speaker table, are things
// StartScenarioMission holds and a started mission does not carry.
func (p *PlayWorld) watchAnnouncements(ms *Mission, src entrySource, faces map[int32]data.NPCFace, tables ...*mapload.Table) {
	if p == nil || ms == nil {
		return
	}
	p.ann = NewAnnouncer(ms.World, ms.Raises)
	p.mission, p.src = ms.Number, src
	p.audience = speakerAudience(HeroAudience(ms.Party), faces)
	var table *mapload.Table
	if len(tables) != 0 {
		table = tables[0]
	}
	p.eventAudience = func(event int) EventAudience {
		return scenarioEventAudience(p.audience, ms, event, table, faces)
	}
}

// decodeInstallText is the install's own code page turned into UTF-8, for a
// report that is JSON and must therefore be valid UTF-8.
//
// NOTHING ON THE DRAW PATH CALLS THIS. The panel receives the raw bytes, which
// is what its byte-to-glyph rule expects; converting for it would corrupt every
// Cyrillic byte twice over.
//
// The code page itself lives at the archive tier, which is where this package
// reaches every other fact about a container entry (0027 SC-8).
func decodeInstallText(s string) string {
	return vfs.DecodeText([]byte(s))
}

// HeadlessAnnouncement is one announcement a drive raised.
//
// Parts is what the drive's own hero would be shown, in page order, after the
// eight conditional markup arms have been applied — so a file whose every tag
// for part 1 refuses this hero reports Shipped with no parts, which is the
// original's silence and not a failure to read.
//
// THE TEXT IS DECODED HERE AND NOWHERE ELSE. This report is JSON, so the
// install's CP866 bytes are converted to UTF-8 for it. The panel keeps receiving
// the raw bytes, because the font path converts them itself.
type HeadlessAnnouncement struct {
	Message  int32    `json:"message"`
	Tick     uint64   `json:"tick"`
	Reserved bool     `json:"reserved,omitempty"`
	Shipped  bool     `json:"shipped"`
	Shown    bool     `json:"shown"`
	Parts    []string `json:"parts,omitempty"`
}

// HeadlessWorldSpec is an authored world: its size, its units, and which owner
// slots are hostile to which.
//
// IT IS COMPOSED, NEVER SAMPLED. Cutting a subset out of a real archive would
// put game data in the repository at every commit including history (golden rule
// 1), so a synthetic scenario states its world in its own terms and this builds
// a sim.World from that.
type HeadlessWorldSpec struct {
	Width   int32              `json:"width"`
	Height  int32              `json:"height"`
	Seed    uint64             `json:"seed,omitempty"`
	Units   []HeadlessUnitSpec `json:"units"`
	Hostile [][2]uint32        `json:"hostile,omitempty"`

	// Spells is the world's own spell table (0154). An authored world states
	// its rows outright, so a scenario proves the mechanism the rules describe
	// rather than the installed table's contents — scenarios/README.md's own
	// split between what `synthetic` and what `install` can show.
	Spells []HeadlessSpellSpec `json:"spells,omitempty"`
}

// HeadlessSpellSpec is one row of an authored spell table. It is the
// SpellRule the simulation holds, in JSON, with the two arms named as one
// word rather than as two flags a file could set together.
type HeadlessSpellSpec struct {
	ID          uint16 `json:"id"`
	ManaCost    int32  `json:"mana_cost,omitempty"`
	School      uint8  `json:"school,omitempty"`
	MaxRange    uint8  `json:"max_range,omitempty"`
	DamageMin   int32  `json:"damage_min,omitempty"`
	DamageMax   int32  `json:"damage_max,omitempty"`
	TargetsUnit bool   `json:"targets_unit,omitempty"`

	// Area and AreaDuration are the two shape columns: a row Area is true of
	// lands an AREA EFFECT on a cell, timed by `(AreaDuration << 4) + (power <<
	// 4)/10` ticks. A headless scenario authoring neither describes a point
	// row, which is the shipped table's own majority.
	Area         bool  `json:"area,omitempty"`
	AreaDuration int32 `json:"area_duration,omitempty"`

	// Arm is "damage" or "heal". Empty is a row this build has no arm for,
	// which every cast at it is refused by — the state the shipped table's own
	// sixteen non-damage rows are in.
	Arm string `json:"arm,omitempty"`
}

// HeadlessUnitSpec is one authored unit. Ref is the name the scenario's own
// steps address it by, in the same grammar an installed map's units answer to.
type HeadlessUnitSpec struct {
	Ref   string `json:"ref"`
	X     int32  `json:"x"`
	Y     int32  `json:"y"`
	Owner uint32 `json:"owner,omitempty"`
	Group uint32 `json:"group,omitempty"`
	HP    int32  `json:"hp"`

	// MaxHP is the health the unit was BUILT with, defaulting to HP (0154).
	// A unit that starts hurt is a state 0155's own form could not express —
	// hp was both the current and the maximum — and it is the one a heal is
	// aimed at, so a scenario that could not author it could not drive one.
	MaxHP int32 `json:"max_hp,omitempty"`

	Damage     int32 `json:"damage,omitempty"`
	ScanRange  uint8 `json:"scan_range,omitempty"`
	AlwaysHits bool  `json:"always_hits,omitempty"`

	// The caster's own four fields (0154): the pool a cast is paid from, the
	// statistic its power reads, the book it must know a spell from — as a
	// list of ids rather than as the mask the simulation holds, so a file
	// names spells rather than bits — and the spell it casts unbidden.
	Mana        int32    `json:"mana,omitempty"`
	MaxMana     int32    `json:"max_mana,omitempty"`
	Mind        int32    `json:"mind,omitempty"`
	KnownSpells []uint32 `json:"known_spells,omitempty"`
	Autocast    uint16   `json:"autocast,omitempty"`

	// RotationSpeed is the per-tick facing-arc budget the movement and attack
	// gate spends a turn against (1047). It defaults to 0, which
	// pkg/sim/facing.go's own gate reads as the pre-1047 compatibility snap:
	// an authored unit that names no rotation_speed turns instantly, exactly
	// as every scenario file predating this field already assumed. A
	// scenario that wants to drive or witness a real multi-tick turn names
	// this field; nothing else in the authored world changes shape.
	RotationSpeed int32 `json:"rotation_speed,omitempty"`
}

func (w *HeadlessWorldSpec) validate() error {
	if w == nil {
		return fmt.Errorf("assets %q requires a world", AssetsSynthetic)
	}
	if w.Width <= 0 || w.Height <= 0 {
		return errors.New("world.width and world.height must be positive")
	}
	if len(w.Units) == 0 {
		return errors.New("world has no units")
	}
	seen := map[string]bool{}
	for i, u := range w.Units {
		switch {
		case u.Ref == "":
			return fmt.Errorf("world.units[%d] has no ref", i)
		case seen[u.Ref]:
			return fmt.Errorf("world.units[%d]: ref %q is used twice", i, u.Ref)
		case u.HP <= 0:
			return fmt.Errorf("world.units[%d] (%s): hp must be positive", i, u.Ref)
		case u.MaxHP < 0:
			return fmt.Errorf("world.units[%d] (%s): max_hp must not be negative", i, u.Ref)
		case u.MaxHP > 0 && u.HP > u.MaxHP:
			return fmt.Errorf("world.units[%d] (%s): hp %d is above max_hp %d",
				i, u.Ref, u.HP, u.MaxHP)
		case u.X < 0 || u.X >= w.Width || u.Y < 0 || u.Y >= w.Height:
			return fmt.Errorf("world.units[%d] (%s): (%d,%d) is outside the world",
				i, u.Ref, u.X, u.Y)
		}
		if _, _, err := headlessRefParts(u.Ref); err != nil {
			return fmt.Errorf("world.units[%d]: %w", i, err)
		}
		seen[u.Ref] = true
	}
	ids := map[uint16]bool{}
	for i, sp := range w.Spells {
		switch strings.ToLower(strings.TrimSpace(sp.Arm)) {
		case "", "damage", "heal":
		default:
			return fmt.Errorf("world.spells[%d] (id %d): arm %q: want damage, heal or nothing",
				i, sp.ID, sp.Arm)
		}
		if sp.ID == 0 {
			return fmt.Errorf("world.spells[%d]: id 0 names no row", i)
		}
		if ids[sp.ID] {
			return fmt.Errorf("world.spells[%d]: id %d is used twice", i, sp.ID)
		}
		ids[sp.ID] = true
	}
	return nil
}

// Build turns the authored world into a PlayWorld, with both reference tables
// filled from the refs the units declare.
func (w *HeadlessWorldSpec) Build() (*PlayWorld, error) {
	if err := w.validate(); err != nil {
		return nil, err
	}
	ents := make([]sim.Entity, 0, len(w.Units))
	script := map[uint16]sim.EntityID{}
	party := map[int]sim.EntityID{}
	highest := -1
	for i, u := range w.Units {
		id := sim.EntityID(i)
		var known uint32
		for _, spell := range u.KnownSpells {
			if spell < 32 {
				known |= 1 << spell
			}
		}
		ents = append(ents, sim.Entity{
			ID: id, X: u.X, Y: u.Y, Owner: u.Owner, Group: u.Group,
			HP: u.HP, MaxHP: headlessMaxHP(u), DamageBase: u.Damage,
			ScanRange: u.ScanRange, AlwaysHits: u.AlwaysHits,
			Mana: u.Mana, MaxMana: u.MaxMana, Mind: u.Mind,
			KnownSpells: known, AutoSpell: u.Autocast,
			RotationSpeed: u.RotationSpeed,
		})
		kind, n, err := headlessRefParts(u.Ref)
		if err != nil {
			return nil, err
		}
		switch kind {
		case 'u':
			script[uint16(n)] = id
		case 'p':
			party[n] = id
			if n > highest {
				highest = n
			}
		}
	}
	slots := make([]sim.EntityID, highest+1)
	for slot := range slots {
		id, ok := party[slot]
		if !ok {
			return nil, fmt.Errorf("world declares p%d but no p%d", highest, slot)
		}
		slots[slot] = id
	}
	var rel sim.Relations
	for _, pair := range w.Hostile {
		rel.Set(pair[0], pair[1], 1)
	}
	spells := make([]sim.SpellRule, 0, len(w.Spells))
	for _, sp := range w.Spells {
		rule := sim.SpellRule{ID: sp.ID, ManaCost: sp.ManaCost, School: sp.School,
			MaxRange: sp.MaxRange, DamageMin: sp.DamageMin, DamageMax: sp.DamageMax,
			TargetsUnit: sp.TargetsUnit, Area: sp.Area, AreaDuration: sp.AreaDuration}
		switch strings.ToLower(strings.TrimSpace(sp.Arm)) {
		case "damage":
			rule.Damaging = true
		case "heal":
			rule.Restorative = true
		}
		spells = append(spells, rule)
	}
	world, err := sim.NewStockedSpelledWorld(w.Seed, sim.Bounds{Width: w.Width, Height: w.Height},
		sim.ModeCanonical, sim.Terrain{}, ents, nil, rel, nil, nil, spells)
	if err != nil {
		return nil, err
	}
	return &PlayWorld{World: world, Script: script, Party: slots}, nil
}

// headlessMaxHP is the maximum one authored unit was built with: its own
// max_hp, or its hp for a file that states none — which is every 0155 scenario
// and is what those files meant.
func headlessMaxHP(u HeadlessUnitSpec) int32 {
	if u.MaxHP > 0 {
		return u.MaxHP
	}
	return u.HP
}

// headlessRefParts splits a reference into its kind and its number.
func headlessRefParts(ref string) (byte, int, error) {
	s := strings.TrimSpace(ref)
	if len(s) < 2 || (s[0] != 'u' && s[0] != 'p' && s[0] != 'e') {
		return 0, 0, fmt.Errorf("unit %q: want uNN, pN or eNN", ref)
	}
	n, err := strconv.Atoi(s[1:])
	if err != nil || n < 0 {
		return 0, 0, fmt.Errorf("unit %q: %q is not a non-negative number", ref, s[1:])
	}
	if s[0] == 'u' && n > 0xffff {
		return 0, 0, fmt.Errorf("unit %q: script id out of range", ref)
	}
	return s[0], n, nil
}

// NewPlayWorld reads the three fields off a started mission.
func NewPlayWorld(ms *Mission) *PlayWorld {
	if ms == nil {
		return nil
	}
	p := &PlayWorld{
		World:  ms.World,
		Script: mapload.ScriptUnits(ms.Map, ms.Party),
		Party:  append([]sim.EntityID(nil), ms.Start.IDs...),
	}
	for i, member := range ms.Party {
		if i < len(ms.Start.IDs) && primaryPlayerHero(member) {
			p.primary, p.primaryOK = ms.Start.IDs[i], true
			break
		}
	}
	return p
}

// pickUp is the game's pick-up order over this driver: orderPickup's refusal
// and walk, settlePickup's arrival test after each ordinary Step, and the
// shared underfoot take.
func (p *PlayWorld) pickUp(id sim.EntityID, x, y int32) error {
	sackThere := func() bool {
		for _, s := range p.World.Sacks() {
			if s.X == x && s.Y == y {
				return true
			}
		}
		return false
	}
	if !sackThere() {
		return fmt.Errorf("pick_item: (%d,%d) holds no sack", x, y)
	}
	p.step([]sim.Command{sim.GroupMoveTo(id, sim.CellPoint{X: x, Y: y}, 1)})
	for n := 0; ; n++ {
		e, ok := p.World.Entity(id)
		if !ok || !e.Alive() || e.OffMap {
			return fmt.Errorf("pick_item: entity %d stopped before (%d,%d)", id, x, y)
		}
		if !sackThere() {
			return fmt.Errorf("pick_item: the sack at (%d,%d) went before entity %d arrived", x, y, id)
		}
		if e.X == x && e.Y == y && sackPickupArrived(p.World, e) {
			break
		}
		if n >= headlessDefaultWait {
			return fmt.Errorf("pick_item: entity %d did not reach (%d,%d) in %d ticks", id, x, y, n)
		}
		p.step(nil)
	}
	if _, ok := takeSackUnderfoot(p.World, id, p.primary, p.primaryOK); !ok {
		return fmt.Errorf("pick_item: entity %d did not take the sack at (%d,%d)", id, x, y)
	}
	return nil
}

// Resolve reads one unit reference.
//
// The grammar is cmd/missionrun's, verbatim: uNN is the identifier the map's
// script uses, pN is the party's Nth member. It is taken over rather than
// reinvented so that a scenario file and a missionrun invocation name the same
// unit with the same string, and a drive written as flags can be transcribed
// into a scenario without re-deriving anything.
//
// eNN is the one addition: a raw entity id, for a world that has neither a
// script table nor a party — which is exactly the world a test builds.
func (p *PlayWorld) Resolve(ref string) (sim.EntityID, error) {
	s := strings.TrimSpace(ref)
	if len(s) < 2 {
		return 0, fmt.Errorf("unit %q: want uNN, pN or eNN", ref)
	}
	n, err := strconv.Atoi(s[1:])
	if err != nil || n < 0 {
		return 0, fmt.Errorf("unit %q: %q is not a non-negative number", ref, s[1:])
	}
	switch s[0] {
	case 'u':
		if n > 0xffff {
			return 0, fmt.Errorf("unit %q: script id out of range", ref)
		}
		id, ok := p.Script[uint16(n)]
		if !ok {
			return 0, fmt.Errorf("unit %q: this map's script names no unit %d", ref, n)
		}
		return id, nil
	case 'p':
		if n >= len(p.Party) {
			return 0, fmt.Errorf("unit %q: the party has %d member(s)", ref, len(p.Party))
		}
		return p.Party[n], nil
	case 'e':
		return sim.EntityID(n), nil
	}
	return 0, fmt.Errorf("unit %q: want uNN, pN or eNN", ref)
}

// entity is the live record for an id, and whether the world still holds one.
func (p *PlayWorld) entity(id sim.EntityID) (sim.Entity, bool) {
	for _, e := range p.World.Entities() {
		if e.ID == id {
			return e, true
		}
	}
	return sim.Entity{}, false
}

// HeadlessCell is a point and the Chebyshev radius around it that counts as
// being there. A radius is required rather than defaulted to zero because the
// cell itself is routinely occupied and a mission check that measures a distance
// is satisfied at its own radius (cmd/missionrun's waypoint grammar).
type HeadlessCell struct {
	X      int32 `json:"x"`
	Y      int32 `json:"y"`
	Radius int32 `json:"radius"`
}

func (c HeadlessCell) holds(e sim.Entity) bool {
	dx, dy := e.X-c.X, e.Y-c.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx < dy {
		dx = dy
	}
	return dx <= c.Radius
}

// HeadlessUntil is one condition a wait_until step waits for.
//
// THE TWO STAGES HAVE DISJOINT FORMS and validate takes the stage so that
// each admits only its own. Unit, Within, Dead and Outcome are the mission
// stage's; Screen and Notice are the front-end stage's. One struct with two
// admitted vocabularies keeps the parser's "exactly one form" rule stated
// once, where two structs would state it twice.
type HeadlessUntil struct {
	Unit    string        `json:"unit,omitempty"`
	Within  *HeadlessCell `json:"within,omitempty"`
	Dead    *bool         `json:"dead,omitempty"`
	Outcome string        `json:"outcome,omitempty"`

	// Screen is the screen name the front end must be showing, in flow.go's
	// own vocabulary: menu, picker, map, chargen, town, gamemenu, load.
	Screen string `json:"screen,omitempty"`
	// Notice is whether a notice must be open over the map. It is the signal
	// that a mission has reached a verdict and is waiting to be dismissed.
	Notice *bool `json:"notice,omitempty"`
	// Control is a choosable frontend list row, matched by its label or its
	// whitespace-delimited prefix. Waiting never activates the control.
	Control string `json:"control,omitempty"`
}

func (u *HeadlessUntil) validate(stage string) error {
	if u == nil {
		return errors.New("until is empty")
	}
	if stage == StageFrontEnd {
		return u.validateFrontEnd()
	}
	if u.Screen != "" || u.Notice != nil || u.Control != "" {
		return fmt.Errorf("until.screen, until.notice and until.control belong to stage %q", StageFrontEnd)
	}
	forms := 0
	if u.Within != nil {
		forms++
		if u.Within.Radius < 0 {
			return errors.New("until.within.radius must not be negative")
		}
	}
	if u.Dead != nil {
		forms++
	}
	if u.Outcome != "" {
		forms++
		if _, err := headlessOutcomeWanted(u.Outcome); err != nil {
			return err
		}
	}
	if forms != 1 {
		return errors.New("until names exactly one of within, dead or outcome")
	}
	if (u.Within != nil || u.Dead != nil) == (u.Unit == "") {
		return errors.New("until.within and until.dead require until.unit; until.outcome forbids it")
	}
	return nil
}

// validateFrontEnd, headlessScreenNamed and frontEndHolds are in headless.go.
// This file is defined over *sim.World alone and imports no controller, which
// is what its own header promises; the front-end forms of this struct are the
// front-end stage's business and live beside it.

// holds reports whether the condition is satisfied right now.
func (u *HeadlessUntil) holds(p *PlayWorld) (bool, error) {
	if u.Outcome != "" {
		want, err := headlessOutcomeWanted(u.Outcome)
		if err != nil {
			return false, err
		}
		return want(p.World.Outcome()), nil
	}
	id, err := p.Resolve(u.Unit)
	if err != nil {
		return false, err
	}
	e, ok := p.entity(id)
	if u.Dead != nil {
		// A world that no longer holds the entity satisfies "dead" and
		// refutes "alive", so a unit removed rather than felled is not a
		// condition that can never be met.
		return (!ok || e.HP <= 0) == *u.Dead, nil
	}
	if !ok {
		return false, nil
	}
	return u.Within.holds(e), nil
}

func (u *HeadlessUntil) String() string {
	switch {
	case u.Screen != "":
		return "screen " + u.Screen
	case u.Control != "":
		return "control " + u.Control
	case u.Notice != nil:
		return fmt.Sprintf("notice=%v", *u.Notice)
	case u.Outcome != "":
		return "outcome " + u.Outcome
	case u.Dead != nil:
		return fmt.Sprintf("%s dead=%v", u.Unit, *u.Dead)
	default:
		return fmt.Sprintf("%s within %d of (%d,%d)", u.Unit, u.Within.Radius, u.Within.X, u.Within.Y)
	}
}

// HeadlessUnitAssertion is what one unit must be.
type HeadlessUnitAssertion struct {
	Alive     *bool         `json:"alive,omitempty"`
	X         *int32        `json:"x,omitempty"`
	Y         *int32        `json:"y,omitempty"`
	Within    *HeadlessCell `json:"within,omitempty"`
	HPAtLeast *int32        `json:"hp_at_least,omitempty"`
	HPAtMost  *int32        `json:"hp_at_most,omitempty"`

	// The caster's own two (0154): the pool a cast is paid from, and the
	// spell the unit casts unbidden — 0 asserting that it casts none.
	ManaAtLeast *int32  `json:"mana_at_least,omitempty"`
	ManaAtMost  *int32  `json:"mana_at_most,omitempty"`
	Autocast    *uint16 `json:"autocast,omitempty"`
	Owner       *uint32 `json:"owner,omitempty"`
	Group       *uint32 `json:"group,omitempty"`

	// The unit's own container (0156).
	Carries []HeadlessCarry `json:"carries,omitempty"`
}

// HeadlessCarry is one clause about a unit's own container.
//
// Code is the PACKED ITEM CODE, not a name: the container this reaches holds
// codes and this build's simulation tier carries no item-name table. Mission
// 30's quest item is 3614 (0x0e1e), which is what its own script node compiles
// to.
//
// Exactly one of Count and Absent may be stated, and stating neither means "at
// least one unit of this code". They are separate fields rather than a count of
// zero because an element with a count of zero is a state no container may
// hold: asking for one would be asking whether the world is malformed, where
// Absent asks whether the code is there at all.
type HeadlessCarry struct {
	Code   uint16  `json:"code"`
	Count  *uint32 `json:"count,omitempty"`
	Absent bool    `json:"absent,omitempty"`
}

func (a *HeadlessUnitAssertion) validate() error {
	if a == nil {
		return errors.New("expect is empty")
	}
	if a.Alive == nil && a.X == nil && a.Y == nil && a.Within == nil &&
		a.HPAtLeast == nil && a.HPAtMost == nil && a.Owner == nil && a.Group == nil &&
		a.ManaAtLeast == nil && a.ManaAtMost == nil && a.Autocast == nil &&
		len(a.Carries) == 0 {
		return errors.New("expect states nothing")
	}
	if a.Within != nil && a.Within.Radius < 0 {
		return errors.New("expect.within.radius must not be negative")
	}
	for i, c := range a.Carries {
		if c.Code == 0 {
			return fmt.Errorf("expect.carries[%d] names item code 0, which is not an item", i)
		}
		if c.Absent && c.Count != nil {
			return fmt.Errorf("expect.carries[%d] states both absent and a count", i)
		}
	}
	return nil
}

func (a *HeadlessUnitAssertion) check(p *PlayWorld, ref string) error {
	id, err := p.Resolve(ref)
	if err != nil {
		return err
	}
	e, ok := p.entity(id)
	if a.Alive != nil {
		alive := ok && e.HP > 0
		if alive != *a.Alive {
			return fmt.Errorf("unit %s alive = %v, want %v", ref, alive, *a.Alive)
		}
	}
	if !ok {
		// Every remaining clause reads a field of an entity. Reporting the
		// absence once is exact; reporting "x = 0" for a unit the world does
		// not hold would be a number that never existed.
		if a.X != nil || a.Y != nil || a.Within != nil || a.HPAtLeast != nil ||
			a.HPAtMost != nil || a.Owner != nil || a.Group != nil ||
			a.ManaAtLeast != nil || a.ManaAtMost != nil || a.Autocast != nil ||
			len(a.Carries) > 0 {
			return fmt.Errorf("unit %s: the world holds no entity %d", ref, uint32(id))
		}
		return nil
	}
	if a.X != nil && e.X != *a.X {
		return fmt.Errorf("unit %s x = %d, want %d", ref, e.X, *a.X)
	}
	if a.Y != nil && e.Y != *a.Y {
		return fmt.Errorf("unit %s y = %d, want %d", ref, e.Y, *a.Y)
	}
	if a.Within != nil && !a.Within.holds(e) {
		return fmt.Errorf("unit %s at (%d,%d) is not within %d of (%d,%d)",
			ref, e.X, e.Y, a.Within.Radius, a.Within.X, a.Within.Y)
	}
	if a.HPAtLeast != nil && e.HP < *a.HPAtLeast {
		return fmt.Errorf("unit %s hp = %d, want at least %d", ref, e.HP, *a.HPAtLeast)
	}
	if a.HPAtMost != nil && e.HP > *a.HPAtMost {
		return fmt.Errorf("unit %s hp = %d, want at most %d", ref, e.HP, *a.HPAtMost)
	}
	if a.Owner != nil && e.Owner != *a.Owner {
		return fmt.Errorf("unit %s owner = %d, want %d", ref, e.Owner, *a.Owner)
	}
	if a.Group != nil && e.Group != *a.Group {
		return fmt.Errorf("unit %s group = %d, want %d", ref, e.Group, *a.Group)
	}
	if a.ManaAtLeast != nil && e.Mana < *a.ManaAtLeast {
		return fmt.Errorf("unit %s mana = %d, want at least %d", ref, e.Mana, *a.ManaAtLeast)
	}
	if a.ManaAtMost != nil && e.Mana > *a.ManaAtMost {
		return fmt.Errorf("unit %s mana = %d, want at most %d", ref, e.Mana, *a.ManaAtMost)
	}
	if a.Autocast != nil && e.AutoSpell != *a.Autocast {
		return fmt.Errorf("unit %s autocast = %d, want %d", ref, e.AutoSpell, *a.Autocast)
	}
	if len(a.Carries) > 0 {
		// The ELEMENTS and not the flat expansion, so a clause can say what
		// the count of one code is. A flat list cannot tell one element at
		// count 2 from two elements at count 1.
		held, ok := p.World.CarriedStacks(id)
		if !ok {
			return fmt.Errorf("unit %s: the world holds no container for entity %d", ref, uint32(id))
		}
		for _, c := range a.Carries {
			var count uint32
			for _, st := range held {
				if st.Code == c.Code {
					count += st.Count
				}
			}
			switch {
			case c.Absent && count != 0:
				return fmt.Errorf("unit %s carries %d of item code %d (%#04x), want none",
					ref, count, c.Code, c.Code)
			case c.Absent:
			case c.Count != nil && count != *c.Count:
				return fmt.Errorf("unit %s carries %d of item code %d (%#04x), want %d",
					ref, count, c.Code, c.Code, *c.Count)
			case c.Count == nil && count == 0:
				return fmt.Errorf("unit %s carries no item code %d (%#04x)", ref, c.Code, c.Code)
			}
		}
	}
	return nil
}

// HeadlessWorldAssertion is what the mission as a whole must be.
//
// UnsupportedAtMost is the script-gap census this project's milestone gate
// measures: how many arms of this mission's own compiled script this build
// cannot run. A scenario that names it fails the moment a decode regresses,
// which is the one number a headless run can carry that no party snapshot can.
type HeadlessWorldAssertion struct {
	Outcome           string  `json:"outcome,omitempty"`
	TickAtLeast       *uint64 `json:"tick_at_least,omitempty"`
	TickAtMost        *uint64 `json:"tick_at_most,omitempty"`
	UnsupportedAtMost *int    `json:"unsupported_at_most,omitempty"`
	ReachedAtMost     *int    `json:"reached_unsupported_at_most,omitempty"`
	AliveAtLeast      *int    `json:"alive_at_least,omitempty"`
	FallenAtMost      *int    `json:"fallen_at_most,omitempty"`
	Latched           []int32 `json:"latched,omitempty"`
	NotLatched        []int32 `json:"not_latched,omitempty"`

	// Announced is the message numbers the drive must have raised, in order and
	// complete. It states the whole list rather than a subset, because "these
	// were raised" and "these and nothing else" are different claims and only
	// the second can catch a raise that stopped happening.
	Announced []int32 `json:"announced,omitempty"`
}

func (a *HeadlessWorldAssertion) validate() error {
	if a == nil {
		return errors.New("world is empty")
	}
	if a.Outcome == "" && a.TickAtLeast == nil && a.TickAtMost == nil &&
		a.UnsupportedAtMost == nil && a.ReachedAtMost == nil && a.AliveAtLeast == nil &&
		a.FallenAtMost == nil && len(a.Latched) == 0 && len(a.NotLatched) == 0 &&
		len(a.Announced) == 0 {
		return errors.New("world states nothing")
	}
	if a.Outcome != "" {
		if _, err := headlessOutcomeWanted(a.Outcome); err != nil {
			return err
		}
	}
	return nil
}

func (a *HeadlessWorldAssertion) check(p *PlayWorld, start map[sim.EntityID]int32) error {
	w := p.World
	if a.Outcome != "" {
		want, err := headlessOutcomeWanted(a.Outcome)
		if err != nil {
			return err
		}
		if !want(w.Outcome()) {
			return fmt.Errorf("outcome = %s, want %s", headlessOutcomeName(w.Outcome()), a.Outcome)
		}
	}
	if a.TickAtLeast != nil && w.Tick() < *a.TickAtLeast {
		return fmt.Errorf("tick = %d, want at least %d", w.Tick(), *a.TickAtLeast)
	}
	if a.TickAtMost != nil && w.Tick() > *a.TickAtMost {
		return fmt.Errorf("tick = %d, want at most %d", w.Tick(), *a.TickAtMost)
	}
	if a.UnsupportedAtMost != nil {
		got := len(w.Script().Unsupported())
		if got > *a.UnsupportedAtMost {
			return fmt.Errorf("script has %d arm(s) this build cannot run, want at most %d",
				got, *a.UnsupportedAtMost)
		}
	}
	if a.ReachedAtMost != nil {
		got := p.reached.total()
		if got > *a.ReachedAtMost {
			return fmt.Errorf("the drive reached %d unrunnable script arm(s), want at most %d",
				got, *a.ReachedAtMost)
		}
	}
	alive, fallen := headlessCounts(p, start)
	if a.AliveAtLeast != nil && alive < *a.AliveAtLeast {
		return fmt.Errorf("alive = %d, want at least %d", alive, *a.AliveAtLeast)
	}
	if a.FallenAtMost != nil && fallen > *a.FallenAtMost {
		return fmt.Errorf("fallen = %d, want at most %d", fallen, *a.FallenAtMost)
	}
	for _, i := range a.Latched {
		if !w.ScriptLatched(i) {
			return fmt.Errorf("latch %d is not set", i)
		}
	}
	for _, i := range a.NotLatched {
		if w.ScriptLatched(i) {
			return fmt.Errorf("latch %d is set", i)
		}
	}
	if len(a.Announced) > 0 {
		got := make([]int32, 0, len(p.announced))
		for _, an := range p.announced {
			got = append(got, an.Message)
		}
		if !reflect.DeepEqual(got, a.Announced) {
			return fmt.Errorf("the drive raised announcements %v, want %v", got, a.Announced)
		}
	}
	return nil
}

// headlessCounts is how many entities are still standing, and how many fell
// since the run began. start is each entity's health at the first step.
func headlessCounts(p *PlayWorld, start map[sim.EntityID]int32) (alive, fallen int) {
	for _, e := range p.World.Entities() {
		if e.HP > 0 {
			alive++
		} else if start[e.ID] > 0 {
			fallen++
		}
	}
	return alive, fallen
}

func headlessOutcomeName(o sim.Outcome) string {
	switch o {
	case sim.OutcomeWon:
		return "won"
	case sim.OutcomeLost:
		return "lost"
	}
	return "undecided"
}

// headlessOutcomeWanted turns a written outcome into the test it names.
// "decided" is either verdict, which is what a drive with a tick ceiling waits
// for when it does not assert which way the mission goes.
func headlessOutcomeWanted(name string) (func(sim.Outcome) bool, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "won":
		return func(o sim.Outcome) bool { return o == sim.OutcomeWon }, nil
	case "lost":
		return func(o sim.Outcome) bool { return o == sim.OutcomeLost }, nil
	case "undecided":
		return func(o sim.Outcome) bool { return o == sim.OutcomeUndecided }, nil
	case "decided":
		return func(o sim.Outcome) bool { return o != sim.OutcomeUndecided }, nil
	}
	return nil, fmt.Errorf("unknown outcome %q: want won, lost, undecided or decided", name)
}

func headlessDifficulty(name string) (mapload.Difficulty, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "normal":
		return mapload.DifficultyNormal, nil
	case "easy":
		return mapload.DifficultyEasy, nil
	case "hard":
		return mapload.DifficultyHard, nil
	}
	return 0, fmt.Errorf("unknown difficulty %q: want easy, normal or hard", name)
}

// validateOrder checks one order step's parameters against the verb it names.
func (s HeadlessStep) validateOrder() error {
	verb := strings.ToLower(strings.TrimSpace(s.Order))
	aimed, victim := false, false
	switch verb {
	case "move", "patrol", "march":
		aimed = true
	case "attack":
		victim = true
	case "cast":
		victim = true
		if s.Spell == nil || *s.Spell == 0 {
			return errors.New(`order "cast" takes a spell, and 0 names no row`)
		}
	case "autocast":
		// And the toggle names a spell alone — 0 being the id that CLEARS
		// it, which is why Spell is a pointer and its absence is the error.
		if s.Spell == nil {
			return errors.New(`order "autocast" takes a spell, 0 clearing it`)
		}
	case "guard", "stand":
	default:
		return fmt.Errorf("unknown order %q: want move, attack, cast, autocast, patrol, march, guard or stand", s.Order)
	}
	if aimed != (s.X != nil && s.Y != nil) {
		return fmt.Errorf("order %q takes x and y", verb)
	}
	if (s.X == nil) != (s.Y == nil) {
		return errors.New("x and y are given together")
	}
	if victim != (s.Target != "") {
		return fmt.Errorf("order %q takes target", verb)
	}
	return nil
}

// commands is the production simulation commands one order step issues.
func (s HeadlessStep) commands(p *PlayWorld) ([]sim.Command, error) {
	id, err := p.Resolve(s.Unit)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(s.Order)) {
	case "move":
		return []sim.Command{sim.MoveTo(id, sim.CellPoint{X: *s.X, Y: *s.Y})}, nil
	case "attack":
		victim, err := p.Resolve(s.Target)
		if err != nil {
			return nil, err
		}
		return []sim.Command{sim.Attack(id, victim)}, nil
	case "cast":
		victim, err := p.Resolve(s.Target)
		if err != nil {
			return nil, err
		}
		return []sim.Command{sim.Cast(id, victim, sim.SpellID(*s.Spell))}, nil
	case "autocast":
		return []sim.Command{sim.Autocast(id, sim.SpellID(*s.Spell))}, nil
	case "patrol":
		return []sim.Command{sim.GroupPatrolTo(id, sim.CellPoint{X: *s.X, Y: *s.Y}, 0)}, nil
	case "march":
		return []sim.Command{sim.GroupSwarmTo(id, sim.CellPoint{X: *s.X, Y: *s.Y}, 0)}, nil
	case "guard":
		return []sim.Command{sim.GroupStance(id, sim.OrderGuard, 0)}, nil
	case "stand":
		return []sim.Command{sim.GroupStance(id, sim.OrderStandGround, 0)}, nil
	}
	return nil, fmt.Errorf("unknown order %q", s.Order)
}

// HeadlessWorldState is one deterministic observation of a mission.
//
// Unsupported is the COMPILED count: arms of this map's script this build
// cannot run, whether or not anything walked into them. Reached is how many
// the drive actually arrived at. The two answer different questions and a
// scenario that records only the first is recording a property of the file
// rather than of the play.
//
// Hash is the simulation's own digest. pkg/sim advances on a fixed integer tick
// with no clock, no math/rand and no floats, so two runs of the same scenario
// over the same world produce the same value; two runs over different assets do
// not, which is why it is recorded rather than asserted here.
type HeadlessWorldState struct {
	Tick        uint64              `json:"tick"`
	Outcome     string              `json:"outcome"`
	Alive       int                 `json:"alive"`
	Fallen      int                 `json:"fallen"`
	Unsupported int                 `json:"unsupported"`
	Reached     int                 `json:"reached_unsupported"`
	Hash        uint64              `json:"hash"`
	Census      []HeadlessGapCount  `json:"census,omitempty"`
	Units       []HeadlessUnitState `json:"units,omitempty"`

	// Announcements is every announcement the drive has raised so far, in the
	// order raised. It is carried on the steps that already carry per-entity
	// rows, for their reason: a four-thousand-tick drive would otherwise write
	// the same list four thousand times.
	Announcements []HeadlessAnnouncement `json:"announcements,omitempty"`
}

// HeadlessUnitState is one entity as a report step states it.
type HeadlessUnitState struct {
	Ref    string `json:"ref"`
	Entity uint32 `json:"entity"`
	X      int32  `json:"x"`
	Y      int32  `json:"y"`
	HP     int32  `json:"hp"`
	MaxHP  int32  `json:"max_hp"`
	Owner  uint32 `json:"owner"`
	Group  uint32 `json:"group"`
}

// worldState is the summary every mission-stage step emits. units is whether
// the per-entity rows are included, which only a report step asks for: a run of
// four thousand ticks past a hundred units would otherwise write four hundred
// thousand rows nobody reads.
func (p *PlayWorld) worldState(start map[sim.EntityID]int32, units bool) HeadlessWorldState {
	alive, fallen := headlessCounts(p, start)
	state := HeadlessWorldState{
		Tick:        p.World.Tick(),
		Outcome:     headlessOutcomeName(p.World.Outcome()),
		Alive:       alive,
		Fallen:      fallen,
		Unsupported: len(p.World.Script().Unsupported()),
		Reached:     p.reached.total(),
		Hash:        p.World.Hash(),
	}
	if !units {
		return state
	}
	state.Census = p.reached.rows()
	state.Announcements = p.announced
	names := p.refNames()
	for _, e := range p.World.Entities() {
		state.Units = append(state.Units, HeadlessUnitState{
			Ref: names[e.ID], Entity: uint32(e.ID), X: e.X, Y: e.Y,
			HP: e.HP, MaxHP: e.MaxHP, Owner: e.Owner, Group: e.Group,
		})
	}
	return state
}

// refNames is the reference each entity answers to, preferring the party naming
// because a party member has no script identifier at all.
func (p *PlayWorld) refNames() map[sim.EntityID]string {
	names := map[sim.EntityID]string{}
	ids := make([]int, 0, len(p.Script))
	for mapID := range p.Script {
		ids = append(ids, int(mapID))
	}
	sort.Ints(ids)
	for _, mapID := range ids {
		names[p.Script[uint16(mapID)]] = fmt.Sprintf("u%d", mapID)
	}
	for i, id := range p.Party {
		names[id] = fmt.Sprintf("p%d", i)
	}
	return names
}

// RunPlayScenario executes a validated mission-stage scenario against a world.
//
// JSON events go to machine, one per step, and a compact trace goes to human.
// The two writers are the same contract the front-end stage already publishes,
// so a caller consuming one route's output consumes the other's unchanged.
func RunPlayScenario(p *PlayWorld, scenario HeadlessScenario, machine, human io.Writer) error {
	if p == nil || p.World == nil {
		return errors.New("play run: no world")
	}
	if err := scenario.Validate(); err != nil {
		return err
	}
	if scenario.StageName() != StageMission {
		return fmt.Errorf("play run: scenario is on stage %q", scenario.StageName())
	}
	start := map[sim.EntityID]int32{}
	for _, e := range p.World.Entities() {
		start[e.ID] = e.HP
	}
	enc := json.NewEncoder(machine)
	for i, step := range scenario.Steps {
		command := strings.ToLower(strings.TrimSpace(step.Command))
		note, err := runPlayStep(p, step, start)
		if err != nil {
			return fmt.Errorf("step %d (%s): %w", i+1, step.Command, err)
		}
		state := p.worldState(start, command == "report")
		event := HeadlessEvent{Step: i + 1, Command: command, World: &state}
		if command == "report" {
			event.Name = step.Name
		}
		if err := enc.Encode(event); err != nil {
			return fmt.Errorf("step %d output: %w", i+1, err)
		}
		fmt.Fprintf(human, "[%d] %-12s tick=%d outcome=%s alive=%d fallen=%d unsupported=%d reached=%d hash=%016x%s\n",
			i+1, command, state.Tick, state.Outcome, state.Alive, state.Fallen,
			state.Unsupported, state.Reached, state.Hash, note)
		for _, row := range state.Census {
			fmt.Fprintf(human, "    reached %-10s op %-3d %d time(s)\n", row.Kind, row.Op, row.Times)
		}
		for _, u := range state.Units {
			fmt.Fprintf(human, "    %-6s entity=%d (%d,%d) hp=%d/%d slot=%d group=%d\n",
				u.Ref, u.Entity, u.X, u.Y, u.HP, u.MaxHP, u.Owner, u.Group)
		}
	}
	return nil
}

// runPlayStep performs one step and returns what the human trace should add to
// the summary line for it.
func runPlayStep(p *PlayWorld, step HeadlessStep, start map[sim.EntityID]int32) (string, error) {
	switch strings.ToLower(strings.TrimSpace(step.Command)) {
	case "kill":
		id, err := p.Resolve(step.Unit)
		if err != nil {
			return "", err
		}
		if err := p.World.HeadlessKill(id); err != nil {
			return "", err
		}
		p.step(nil)
		return fmt.Sprintf("  %s", step.Unit), nil
	case "kill_player":
		killed, err := p.World.HeadlessKillPlayer(*step.Player)
		if err != nil {
			return "", err
		}
		p.step(nil)
		return fmt.Sprintf("  player=%d killed=%d", *step.Player, killed), nil
	case "place":
		id, err := p.Resolve(step.Unit)
		if err != nil {
			return "", err
		}
		if err := p.World.HeadlessPlace(id, *step.X, *step.Y); err != nil {
			return "", err
		}
		p.step(nil)
		e, _ := p.World.Entity(id)
		return fmt.Sprintf("  %s near (%d,%d) at (%d,%d)", step.Unit, *step.X, *step.Y, e.X, e.Y), nil
	case "pick_item":
		id, err := p.Resolve(step.Unit)
		if err != nil {
			return "", err
		}
		if err := p.pickUp(id, *step.X, *step.Y); err != nil {
			return "", err
		}
		p.step(nil)
		return fmt.Sprintf("  %s <- (%d,%d)", step.Unit, *step.X, *step.Y), nil
	case "heal":
		id, err := p.Resolve(step.Unit)
		if err != nil {
			return "", err
		}
		if err := p.World.HeadlessHeal(id); err != nil {
			return "", err
		}
		p.step(nil)
		return fmt.Sprintf("  %s", step.Unit), nil
	case "order":
		cmds, err := step.commands(p)
		if err != nil {
			return "", err
		}
		p.step(cmds)
		return fmt.Sprintf("  %s %s", step.Unit, step.Order), nil
	case "wait_ticks":
		for n := 0; n < step.Ticks; n++ {
			if p.World.Outcome() != sim.OutcomeUndecided {
				break
			}
			p.step(nil)
		}
		return "", nil
	case "wait_until":
		return waitUntil(p, step)
	case "assert_unit":
		return "", step.Expect.check(p, step.Unit)
	case "assert_world":
		return "", step.World.check(p, start)
	case "report":
		return "  " + step.Name, nil
	}
	return "", fmt.Errorf("unknown command %q", step.Command)
}

// headlessDefaultWait is the tick ceiling a wait_until with no ticks of its own
// gets. It is cmd/missionrun's own -ticks default, so a drive transcribed from
// that tool's flags waits exactly as long as it used to.
const headlessDefaultWait = 40000

// waitUntil steps the world until the condition holds, the mission is decided,
// or the ceiling is spent.
//
// A CEILING SPENT WITH THE CONDITION UNMET IS A FAILURE, not a step that
// quietly did nothing. A scenario that waits for an arrival and then asserts on
// it would otherwise report the assertion's failure and say nothing about the
// walk that never finished.
func waitUntil(p *PlayWorld, step HeadlessStep) (string, error) {
	ceiling := step.Ticks
	if ceiling == 0 {
		ceiling = headlessDefaultWait
	}
	for n := 0; ; n++ {
		held, err := step.Until.holds(p)
		if err != nil {
			return "", err
		}
		if held {
			return fmt.Sprintf("  %s after %d tick(s)", step.Until, n), nil
		}
		if n >= ceiling {
			return "", fmt.Errorf("%s did not hold within %d tick(s)", step.Until, ceiling)
		}
		// A decided mission stops accepting work, so waiting past the verdict
		// can only spend the whole ceiling. The outcome conditions are the
		// exception: they are tested above, before this returns.
		if p.World.Outcome() != sim.OutcomeUndecided {
			return "", fmt.Errorf("%s did not hold; the mission was decided %s at tick %d",
				step.Until, headlessOutcomeName(p.World.Outcome()), p.World.Tick())
		}
		p.step(nil)
	}
}

// StartScenarioMission builds the mission a mission-stage scenario names, out of
// archives the caller already opened.
//
// IT OPENS NOTHING AND SPELLS NO PATH (golden rule 3). The archives come from
// the asset root the command line or the environment resolved, exactly as
// cmd/missionrun's do, and the party is built through the same MissionPartyAs
// the front end uses — so a scenario, a missionrun invocation and the game start
// the identical party or none of them does.
func StartScenarioMission(a *Archives, s HeadlessScenario) (*PlayWorld, *Mission, error) {
	if a == nil {
		return nil, nil, errors.New("mission stage: no archives")
	}
	if s.StageName() != StageMission {
		return nil, nil, fmt.Errorf("mission stage: scenario is on stage %q", s.StageName())
	}
	diff, err := headlessDifficulty(s.Difficulty)
	if err != nil {
		return nil, nil, err
	}
	defs, err := LoadDefinitionsFor(a.Containers, a.Game())
	if err != nil {
		return nil, nil, err
	}
	party := MissionPartyAs(s.Mage, defs.StartWeapon, defs.Bodies, defs.Table)
	ms, err := StartMission(a.Containers, s.Mission, defs.Table, diff, party)
	if err != nil {
		return nil, nil, err
	}
	p := NewPlayWorld(ms)
	p.watchAnnouncements(ms, a.Containers, LoadNPCFaces(a.Containers), defs.Table)
	return p, ms, nil
}
