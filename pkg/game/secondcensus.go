package game

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// SecondCensusTicks is how long the census runs each started map.
const SecondCensusTicks = 600

// SecondCause says why a script reference did not bind. CauseFixture is an
// artifact of the census party; every other cause holds for any party.
type SecondCause uint8

const (
	CauseFixture   SecondCause = iota // a hero-band subject the census party does not supply
	CauseDynamic                      // the 8000..9999 band, which this build never binds
	CauseWithdrawn                    // a literal unit whose placement the border ring withdrew
	CauseAbsent                       // a literal unit no placement of the map carries
	CauseStructure                    // a structure word no placed structure carries
	secondCauses
)

func (c SecondCause) String() string {
	return [...]string{"fixture", "dynamic", "withdrawn", "absent", "structure", "?"}[min(int(c), int(secondCauses))]
}

// SecondOmission counts what one cause took out of a map's compiled script:
// the nodes it omitted, the triggers omitted because a condition was, and the
// built triggers that lost an action.
type SecondOmission struct {
	Nodes, Triggers, ActionTriggers int
}

// SecondOpGap is one opcode the compiled script names and this build does not
// run. Triggers counts the triggers it holds: inert for a check, firing
// without that action for an instant.
type SecondOpGap struct {
	Check    bool
	Op, Sub  int32
	Nodes    int
	Triggers int
}

// SecondDeparture is a map's ordinary departure: the cases the published
// claims give and what the engine's campaign controller does with them.
type SecondDeparture struct {
	Case        bool         // R2-ENGINE-145: the departure has a case body for this ID
	Exits       []SecondExit // R2-ENGINE-146 adds and R2-ENGINE-148 movie outputs
	Producer    string       // which published route makes the map available, "" for Unknown
	EngineEntry bool         // the engine's controller can make the map available
	EngineWin   bool         // the engine continues the campaign after its victory
	EngineAdds  []string     // what the engine's continuation adds, every gate open
	EngineSave  bool         // the engine writes a mission SAV on this map
}

// SecondGate is one bank-slot test of a departure exit: the slot is nonzero,
// or it is zero.
type SecondGate struct {
	Slot    int
	Nonzero bool
}

// SecondExit is one result a departure case can produce: a location it adds
// ("mission 21", "town 3", "unresolved record") or a movie output ("movie
// 5"), behind every gate it names. Engine says the controller produces it
// exactly when those gates hold.
type SecondExit struct {
	Target string
	Gates  []SecondGate
	Engine bool
}

// Movie reports whether the exit is a movie output.
func (e SecondExit) Movie() bool { return strings.HasPrefix(e.Target, "movie ") }

// SecondRun is the headless run of a started map: how far it ran, and the
// tick and failure reason of an outcome it reached.
type SecondRun struct {
	Err       string
	Ticks     int
	Outcome   sim.Outcome
	DecidedAt int
	Reason    uint32
}

// SecondMapCensus is one ROM2 campaign map against this build.
type SecondMapCensus struct {
	Mission int
	Err     string // the map did not start

	Placements, Withdrawn int
	NoUnitRow, NoHumanRow []int32 // distinct server ids with no definition row

	// AuthoredHP counts placements whose record carries a current health
	// under the engine-derived field reading (DIV-2357); BornFallen counts
	// those that carry zero or less and so start fallen.
	AuthoredHP, BornFallen         int
	NoRowPlacements                int
	Structures, NoStructureRowKeys int
	NoStructureRow                 []uint16 // distinct kinds with no buildings row
	LootItems, LootDropped         int
	NoItemRow                      []uint16 // distinct loot and script item codes with no row
	Spells, NoSpellRule            []uint16 // distinct spell ids placements or the script name

	Checks, Instants, Triggers int
	Gaps                       []SecondOpGap
	Inert                      []int // map trigger positions an unsupported check holds
	Omitted                    [secondCauses]SecondOmission
	BankSlots                  []int32 // scenario bank slots instant 35 writes

	// SharedNodes counts compiled nodes whose opcode runs on the first game's
	// handler with no published ROM2 contract (DIV-2355); SharedChecks and
	// SharedInstants name those opcodes.
	SharedNodes                  int
	SharedChecks, SharedInstants []int32

	// Events are the ordinary dialogue events the script raises and
	// NoEventText those the mission text has no section for; Reasons and
	// NoReasonText are the same for the failure reasons instant 5 stores.
	Events, NoEventText   []int32
	Reasons, NoReasonText []int32

	Departure SecondDeparture
	Run       SecondRun
}

// FixtureOnly reports whether every script omission on the map is a census
// party artifact.
func (c SecondMapCensus) FixtureOnly() bool {
	for cause := CauseDynamic; cause < secondCauses; cause++ {
		if c.Omitted[cause] != (SecondOmission{}) {
			return false
		}
	}
	return true
}

// The blocker classes Blockers names, in the order it names them.
const (
	BlockStart        = "start"         // the map does not start
	BlockRun          = "run"           // the headless run stops with an error
	BlockHeadlessLoss = "headless-loss" // the unattended run is lost
	BlockOpcode       = "opcode"        // the script names an opcode this build does not run
	BlockReference    = "reference"     // a reference no party could bind
	BlockParty        = "party"         // a hero-band subject the census party lacks
	BlockDefinition   = "definition"    // a placement, structure or item with no row
	BlockSpell        = "spell"         // a named spell with no applicable rule
	BlockEntry        = "entry"         // the controller never makes the map available
	BlockContinuation = "continuation"  // the controller does not continue its victory as published
	BlockMovie        = "movie"         // a departure movie output the engine does not play
	BlockSave         = "save"          // the engine writes no mission SAV on the map
)

// Blockers names every gap class the map shows, in a fixed order. A map
// with none is one this census cannot tell from a supported map. Missing
// event or failure text is reported but blocks nothing: the mission screen
// shows the generic failure notice for it (DIV-2356).
func (c SecondMapCensus) Blockers() []string {
	if c.Err != "" {
		return []string{BlockStart}
	}
	var out []string
	add := func(on bool, name string) {
		if on {
			out = append(out, name)
		}
	}
	d := c.Departure
	add(c.Run.Err != "" || c.Run.Ticks != SecondCensusTicks, BlockRun)
	add(c.Run.Outcome == sim.OutcomeLost, BlockHeadlessLoss)
	add(len(c.Gaps) > 0, BlockOpcode)
	add(!c.FixtureOnly(), BlockReference)
	add(c.Omitted[CauseFixture] != (SecondOmission{}), BlockParty)
	add(c.NoRowPlacements+c.NoStructureRowKeys+len(c.NoItemRow) > 0, BlockDefinition)
	add(len(c.NoSpellRule) > 0, BlockSpell)
	add(!d.EngineEntry, BlockEntry)
	add(!d.EngineWin || d.unsupported(false) || d.extraAdds(), BlockContinuation)
	add(d.unsupported(true), BlockMovie)
	add(!d.EngineSave, BlockSave)
	return out
}

// unsupported reports an exit of the kind the engine does not produce.
func (d SecondDeparture) unsupported(movie bool) bool {
	for _, e := range d.Exits {
		if e.Movie() == movie && !e.Engine {
			return true
		}
	}
	return false
}

// extraAdds reports a location the engine adds that no published exit names.
func (d SecondDeparture) extraAdds() bool {
	for _, a := range d.EngineAdds {
		if !slices.ContainsFunc(d.Exits, func(e SecondExit) bool { return e.Target == a }) {
			return true
		}
	}
	return false
}

// SecondGameCensus walks every campaign map the archives carry, ascending, and
// reports each against this build. It writes nothing.
func SecondGameCensus(archives *Archives) ([]SecondMapCensus, error) {
	if archives == nil || archives.Game() != base.GameROM2 {
		return nil, fmt.Errorf("the census needs a second-game root")
	}
	defs, err := LoadDefinitionsFor(archives.Containers, archives.Game())
	if err != nil {
		return nil, err
	}
	reach := secondEngineReach()
	var out []SecondMapCensus
	for n := 1; n <= 300; n++ {
		address, _ := MissionMap(n)
		raw, err := archives.Containers.ReadFile(address)
		if err != nil {
			continue
		}
		m, err := alm.OpenROM2(raw)
		if err != nil {
			out = append(out, SecondMapCensus{Mission: n, Err: err.Error()})
			continue
		}
		party := MissionPartyAs(false, defs.StartWeapon, defs.Bodies, defs.Table)
		c := secondMapCensus(m, n, defs.Table, party, archives.Containers)
		c.Departure = secondDeparture(n, reach)
		out = append(out, c)
	}
	return out, nil
}

func secondMapCensus(m *alm.Map, n int, t *mapload.Table, party []mapload.PartyMember, src entrySource) SecondMapCensus {
	c := SecondMapCensus{Mission: n, Placements: len(m.Units)}
	authored := slices.Clone(m.Units)
	c.Withdrawn = mapload.WithdrawBorderPlacements(m)
	secondPlacementRows(&c, m, t)
	secondStructureRows(&c, m, t)
	secondLootRows(&c, m, t)
	if err := secondScript(&c, m, authored, t, party); err != nil {
		c.Err = err.Error()
		return c
	}
	secondText(&c, src)
	ms, err := StartMissionFrom(m, "", n, t, mapload.DifficultyNormal, party)
	if err != nil {
		c.Err = err.Error()
		return c
	}
	secondSpellRules(&c, ms.World, len(m.Units))
	c.Run = secondRun(ms.World, SecondCensusTicks)
	return c
}

func secondPlacementRows(c *SecondMapCensus, m *alm.Map, t *mapload.Table) {
	for _, u := range m.Units {
		if u.HasCurrentHP {
			c.AuthoredHP++
			if u.CurrentHP <= 0 {
				c.BornFallen++
			}
		}
		r := mapload.Resolve(u, t)
		if r.Found() {
			continue
		}
		c.NoRowPlacements++
		if r.Arm == mapload.ArmServerID {
			c.NoHumanRow = insertSorted(c.NoHumanRow, int32(u.ServerID))
		} else {
			c.NoUnitRow = insertSorted(c.NoUnitRow, int32(u.ServerID))
		}
	}
}

func secondStructureRows(c *SecondMapCensus, m *alm.Map, t *mapload.Table) {
	c.Structures = len(m.Objects)
	_, counts := mapload.Footprints(m, t)
	c.NoStructureRowKeys = counts.Unresolved
	for _, o := range m.Objects {
		k := int(uint8(o.Kind))
		if t == nil || t.Buildings == nil || k < 1 || k >= t.Buildings.Len() {
			c.NoStructureRow = insertSorted(c.NoStructureRow, uint16(o.Kind))
		}
	}
}

func secondLootRows(c *SecondMapCensus, m *alm.Map, t *mapload.Table) {
	loot, err := m.Loot()
	if err != nil {
		return
	}
	owners := make(map[uint32]int, len(m.Units))
	for _, u := range m.Units {
		owners[uint32(u.UnitID)]++
	}
	for _, r := range loot.Records {
		if r.Ground() {
			x, y := r.CellX(), r.CellY()
			if x < 0 || y < 0 || int(x) >= m.Width || int(y) >= m.Height {
				c.LootDropped++
				continue
			}
		} else if owners[r.Owner] != 1 {
			c.LootDropped++
			continue
		}
		for _, e := range r.Elements {
			c.LootItems++
			if !SecondItemRow(e.ItemCode(), t) {
				c.NoItemRow = insertSorted(c.NoItemRow, e.ItemCode())
			}
		}
	}
}

// SecondItemRow reports whether an item code names a row of the collection
// its class selects.
func SecondItemRow(code uint16, t *mapload.Table) bool {
	if t == nil {
		return false
	}
	c := data.ItemCode(code)
	var col data.Collection
	row := c.D()
	switch {
	case c.B() == 1:
		col = t.Weapons
	case c.B() == 2:
		col = t.Shields
	case c.B() >= 3 && c.B() <= 12:
		col = t.Armors
	case c.B() == data.ItemClassCarried:
		col, row = t.MagicItems, int(uint8(code))
	}
	return col != nil && row > 0 && row < col.Len()
}

// SecondRefCause classifies one reference the ROM2 compile could not bind.
// authored is the map's placements before the border ring was withdrawn.
func SecondRefCause(u mapload.ScriptUnresolved, authored []alm.Unit) SecondCause {
	switch {
	case u.Kind == mapload.ScriptRefStructure:
		return CauseStructure
	case u.Value >= 8000 && u.Value <= 9999:
		return CauseDynamic
	case u.Value >= 10001 && u.Value <= 11000:
		return CauseFixture
	}
	for _, a := range authored {
		if uint32(a.UnitID) == u.Value {
			return CauseWithdrawn
		}
	}
	return CauseAbsent
}

// secondSaturatedRefs binds every hero-band value to the hero, so a compile
// over it omits only what no party could bind.
func secondSaturatedRefs(refs mapload.ScriptRefs, hero sim.EntityID) mapload.ScriptRefs {
	refs.Hero, refs.HasHero = hero, true
	refs.Companion, refs.HasCompanion = hero, true
	refs.Roles = make(map[uint32]sim.EntityID, 999)
	for v := uint32(10003); v <= 11000; v++ {
		refs.Roles[v] = hero
	}
	return refs
}

func secondScript(c *SecondMapCensus, m *alm.Map, authored []alm.Unit, t *mapload.Table, party []mapload.PartyMember) error {
	src, err := m.Script()
	if err != nil {
		return err
	}
	c.Checks, c.Instants, c.Triggers = len(src.Conditions), len(src.Actions), len(src.Triggers)
	refs := campaignScriptRefs(m, t, party)
	_, fixture, err := mapload.CompileROM2ScriptFrom(src, refs)
	if err != nil {
		return err
	}
	full, saturated, err := mapload.CompileROM2ScriptFrom(src, secondSaturatedRefs(refs, mapload.PartyEntity(m, 0)))
	if err != nil {
		return err
	}
	for _, r := range saturated.Raises {
		if !secondSpecialEvent(r.Event) {
			c.Events = insertSorted(c.Events, r.Event)
		}
	}
	secondOmissions(c, src, fixture, authored)
	secondGaps(c, full)
	for _, s := range full.Instants() {
		if !secondClaimedInstant(s.Op) {
			c.SharedNodes++
			c.SharedInstants = insertSorted(c.SharedInstants, s.Op)
		}
		if s.Op == sim.ScriptInstantLose {
			c.Reasons = insertSorted(c.Reasons, s.Args[0])
		}
		if s.Op == sim.ScriptInstantSetScenario {
			c.BankSlots = insertSorted(c.BankSlots, s.Args[0])
		}
		if s.HasItem && !SecondItemRow(s.Item, t) {
			c.NoItemRow = insertSorted(c.NoItemRow, s.Item)
		}
		if spell, ok := scriptCastSpell(s); ok {
			c.Spells = insertSorted(c.Spells, spell)
		}
	}
	for _, k := range full.Checks() {
		if !secondClaimedCheck(k.Op) {
			c.SharedNodes++
			c.SharedChecks = insertSorted(c.SharedChecks, k.Op)
		}
		if k.HasItem && !SecondItemRow(k.Item, t) {
			c.NoItemRow = insertSorted(c.NoItemRow, k.Item)
		}
	}
	return nil
}

// secondSpecialEvent names the event IDs UI 433 routes away from ordinary
// dialogue (R2-ENGINE-050).
func secondSpecialEvent(e int32) bool { return e == 250 || e >= 253 && e <= 255 }

// secondText checks every ordinary event and failure reason against the
// mission's own text, the failure reason with the installed fallback the
// mission screen uses.
func secondText(c *SecondMapCensus, src entrySource) {
	for _, e := range c.Events {
		if _, ok := ReadEventTextFor(src, base.GameROM2, c.Mission, int(e)); !ok {
			c.NoEventText = append(c.NoEventText, e)
		}
	}
	labels := secondGameFailureText(src, c.Mission)
	for _, r := range c.Reasons {
		if r != 0 && (r < 2 || int(r-2) >= len(labels) || labels[r-2] == "") {
			c.NoReasonText = append(c.NoReasonText, r)
		}
	}
}

// secondClaimedCheck and secondClaimedInstant name the opcodes a published
// ROM2 claim describes: the new arms (R2-ENGINE-045..058), the preset
// (R2-ENGINE-043), the message (R2-ENGINE-047) and the result and counter
// actions (R2-ENGINE-060).
func secondClaimedCheck(op int32) bool {
	return op == sim.ScriptCheckConstant || op >= sim.ScriptCheckScenario && op <= sim.ScriptCheckCentered
}

func secondClaimedInstant(op int32) bool {
	switch op {
	case sim.ScriptInstantMessage, sim.ScriptInstantWin, sim.ScriptInstantLose, sim.ScriptInstantIncVariable:
		return true
	}
	return op >= sim.ScriptInstantSetScenario && op <= sim.ScriptInstantClearGroupActivity
}

func scriptCastSpell(s sim.ScriptInstant) (uint16, bool) {
	switch s.Op {
	case sim.ScriptInstantCastAtCell:
		return uint16(uint8(s.Args[4])), true
	case sim.ScriptInstantCastAtUnit:
		return uint16(uint8(s.Args[2])), true
	}
	return 0, false
}

// secondOmissions attributes each omitted node, omitted trigger and built
// trigger that lost an action to one cause. A node or trigger with several
// causes takes the greatest, so a party artifact never hides a real gap.
func secondOmissions(c *SecondMapCensus, src alm.Script, rep mapload.ScriptReport, authored []alm.Unit) {
	type key struct {
		cond bool
		id   uint32
	}
	cause := make(map[key]SecondCause)
	for _, u := range rep.Unresolved {
		k := key{u.Condition, u.NodeID}
		got := SecondRefCause(u, authored)
		if was, seen := cause[k]; !seen || got > was {
			cause[k] = got
		}
	}
	for _, id := range rep.OmittedChecks {
		c.Omitted[cause[key{true, id}]].Nodes++
	}
	omittedAction := make(map[uint32]bool, len(rep.OmittedActions))
	for _, id := range rep.OmittedActions {
		omittedAction[id] = true
		c.Omitted[cause[key{false, id}]].Nodes++
	}
	omittedTrigger := make(map[int]bool, len(rep.OmittedTriggers))
	for _, i := range rep.OmittedTriggers {
		omittedTrigger[i] = true
		worst, any := SecondCause(0), false
		t := src.Triggers[i]
		for k := range t.Left {
			for _, id := range [2]uint32{t.Left[k], t.Right[k]} {
				if got, ok := cause[key{true, id}]; ok && (!any || got > worst) {
					worst, any = got, true
				}
			}
		}
		c.Omitted[worst].Triggers++
	}
	for i, t := range src.Triggers {
		if omittedTrigger[i] || t.Left[0] == 0 {
			continue
		}
		worst, any := SecondCause(0), false
		for _, id := range t.Acts {
			if !omittedAction[id] {
				continue
			}
			if got := cause[key{false, id}]; !any || got > worst {
				worst, any = got, true
			}
		}
		if any {
			c.Omitted[worst].ActionTriggers++
		}
	}
}

func secondGaps(c *SecondMapCensus, s *sim.Script) {
	if s == nil {
		return
	}
	triggers := s.Triggers()
	for _, i := range s.InertTriggers() {
		c.Inert = append(c.Inert, int(triggers[i].Latch))
	}
	slices.Sort(c.Inert)
	checks := s.Checks()
	for _, g := range s.Unsupported() {
		gap := SecondOpGap{Check: g.Kind == sim.ScriptGapCheck, Op: g.Op, Sub: g.Sub}
		held := 0
		for _, t := range triggers {
			if gap.Check {
				reg := checks[g.Index].Register
				for _, p := range t.Pairs {
					if p.Used && (p.Left == reg || p.Right == reg) {
						held++
						break
					}
				}
				continue
			}
			if slices.Contains(t.Instants[:], g.Index) {
				held++
			}
		}
		j := slices.IndexFunc(c.Gaps, func(o SecondOpGap) bool { return o.Check == gap.Check && o.Op == gap.Op && o.Sub == gap.Sub })
		if j < 0 {
			c.Gaps = append(c.Gaps, gap)
			j = len(c.Gaps) - 1
		}
		c.Gaps[j].Nodes++
		c.Gaps[j].Triggers += held
	}
}

func secondSpellRules(c *SecondMapCensus, w *sim.World, placed int) {
	for i, e := range w.Entities() {
		if i >= placed {
			break
		}
		for id := uint16(1); id < 32; id++ {
			if e.KnownSpells&(1<<id) != 0 {
				c.Spells = insertSorted(c.Spells, id)
			}
		}
		for _, s := range e.CreatureSpells {
			if s.ID != 0 {
				c.Spells = insertSorted(c.Spells, uint16(s.ID))
			}
		}
		if e.WeaponSpell != 0 {
			c.Spells = insertSorted(c.Spells, e.WeaponSpell)
		}
	}
	rules := w.Spells()
	for _, id := range c.Spells {
		j := slices.IndexFunc(rules, func(r sim.SpellRule) bool { return r.ID == id })
		if j < 0 || !sim.SpellRuleLands(rules[j]) {
			c.NoSpellRule = insertSorted(c.NoSpellRule, id)
		}
	}
}

func secondRun(w *sim.World, ticks int) (run SecondRun) {
	defer func() {
		if r := recover(); r != nil {
			run.Err = fmt.Sprint("panic: ", r)
		}
		run.Ticks, run.Outcome = int(w.Tick()), w.Outcome()
	}()
	for i := 0; i < ticks; i++ {
		sim.Step(w, nil)
		if run.DecidedAt == 0 && w.Outcome() != sim.OutcomeUndecided {
			run.DecidedAt = int(w.Tick())
			_, run.Reason = w.ScriptCounters()
		}
	}
	return run
}

// secondDepartureCases holds every exit of each published ordinary-departure
// case with its gates, in case-body order (R2-ENGINE-145, R2-ENGINE-146,
// R2-ENGINE-148). A case with no exit stores no output and adds nothing.
var secondDepartureCases = map[int][]SecondExit{
	10: {{Target: "mission 20"}, {Target: "movie 1"}},
	20: {{Target: "town 2"}, {Target: "mission 21", Gates: []SecondGate{{772, true}}}},
	30: {{Target: "movie 2"}},
	31: {{Target: "mission 32"}},
	40: {{Target: "mission 50"}, {Target: "mission 60"}},
	50: {{Target: "town 3"}, {Target: "unresolved record", Gates: []SecondGate{{780, true}}}},
	60: {{Target: "mission 80"}},
	70: {{Target: "movie 3", Gates: []SecondGate{{777, false}, {778, true}}}},
	80: {{Target: "movie 3", Gates: []SecondGate{{778, false}, {777, true}}}},
	90: nil, 100: nil,
	110: {{Target: "movie 5", Gates: []SecondGate{{779, true}}}, {Target: "movie 4", Gates: []SecondGate{{779, false}}}},
}

// secondProducer names the published route that makes a mission available:
// R2-ENGINE-073 for the first inn talk, R2-ENGINE-146 for each departure add.
func secondProducer(n int) string {
	if n == 10 {
		return "town 1 inn talk"
	}
	target := fmt.Sprintf("mission %d", n)
	for _, from := range slices.Sorted(maps.Keys(secondDepartureCases)) {
		if slices.ContainsFunc(secondDepartureCases[from], func(e SecondExit) bool { return e.Target == target }) {
			return fmt.Sprintf("departure of %d", from)
		}
	}
	return ""
}

func secondLocationName(l secondLocation) string {
	if l.kind == 2 {
		return fmt.Sprintf("town %d", l.id)
	}
	return fmt.Sprintf("mission %d", l.id)
}

// secondEngineReach walks the engine's own controller from a new game: the
// first inn talk, the first departure, then every continuation it accepts,
// with every departure gate satisfied.
func secondEngineReach() map[int]bool {
	c := newSecondCampaign()
	c.talk()
	c.leaveTown()
	reach := map[int]bool{}
	for progressed := true; progressed; {
		progressed = false
		for _, l := range slices.Clone(c.available) {
			if l.kind != 1 || reach[l.id] || !c.canEnter(l.id) {
				continue
			}
			reach[l.id], progressed = true, true
			if !secondContinues(l.id) {
				continue
			}
			c.current = l
			bank := c.bank
			bank[772], bank[780] = 1, 1
			c.completeBank(bank)
		}
	}
	return reach
}

func secondDeparture(n int, reach map[int]bool) SecondDeparture {
	d := SecondDeparture{Producer: secondProducer(n), EngineEntry: reach[n], EngineSave: secondSaveMission(n), EngineWin: secondContinues(n)}
	exits, ok := secondDepartureCases[n]
	d.Case = ok
	var open [1024]int32
	for _, e := range exits {
		for _, g := range e.Gates {
			if g.Nonzero {
				open[g.Slot] = 1
			}
		}
	}
	d.EngineAdds = secondEngineAdds(n, open)
	for _, e := range exits {
		e.Gates = slices.Clone(e.Gates)
		e.Engine = secondEngineExit(n, e)
		d.Exits = append(d.Exits, e)
	}
	return d
}

// secondEngineAdds is what the engine's continuation of mission n adds from
// the given bank; nothing when the engine does not continue n.
func secondEngineAdds(n int, bank [1024]int32) []string {
	out, _ := secondEngineDeparture(n, bank)
	return out
}

// secondEngineDeparture runs the engine's continuation of mission n over the
// given bank: the locations it adds and the movie output it stores, "" for
// none.
func secondEngineDeparture(n int, bank [1024]int32) ([]string, string) {
	if !secondContinues(n) {
		return nil, ""
	}
	c := newSecondCampaign()
	c.current = secondLocation{kind: 1, id: n}
	c.available = []secondLocation{c.current}
	output := c.completeBank(bank)
	var out []string
	for _, l := range c.available {
		out = append(out, secondLocationName(l))
	}
	movie := ""
	if output >= 0 {
		movie = fmt.Sprintf("movie %d", output)
	}
	return out, movie
}

// secondEngineExit reports whether the engine produces e exactly when its
// gates hold: with every gate held, and with no gate inverted.
func secondEngineExit(n int, e SecondExit) bool {
	produced := func(flip int) bool {
		var bank [1024]int32
		for i, g := range e.Gates {
			if g.Nonzero != (i == flip) {
				bank[g.Slot] = 1
			}
		}
		adds, movie := secondEngineDeparture(n, bank)
		if e.Movie() {
			return movie == e.Target
		}
		return slices.Contains(adds, e.Target)
	}
	if !produced(-1) {
		return false
	}
	for i := range e.Gates {
		if produced(i) {
			return false
		}
	}
	return true
}

// secondExitList prints exits as "target" or "target if 779!=0 778=0".
func secondExitList(exits []SecondExit, engine bool) string {
	var out []string
	for _, e := range exits {
		if e.Engine != engine {
			continue
		}
		s := e.Target
		for i, g := range e.Gates {
			if i == 0 {
				s += " if"
			}
			op := "="
			if g.Nonzero {
				op = "!="
			}
			s += fmt.Sprintf(" %d%s0", g.Slot, op)
		}
		out = append(out, s)
	}
	if len(out) == 0 {
		return "-"
	}
	return strings.Join(out, ",")
}

func insertSorted[T int32 | uint16](s []T, v T) []T {
	i, found := slices.BinarySearch(s, v)
	if found {
		return s
	}
	return slices.Insert(s, i, v)
}

// SecondCensusTotals sums a census over its maps.
type SecondCensusTotals struct {
	Maps, Started, Ran, Won, Lost           int
	Placements, Withdrawn, NoRowPlacements  int
	AuthoredHP, BornFallen                  int
	NoStructureRow, NoItemRow, NoSpellMaps  int
	Events, NoEventText, NoReasonText       int
	Checks, Instants, Triggers, Gaps, Inert int
	SharedNodes                             int
	Omitted                                 [secondCauses]SecondOmission
	FixtureMaps, EntryMaps, WinMaps, Ready  int
	Exits, GatedExits, EngineExits          int
}

// SecondTotals sums cs.
func SecondTotals(cs []SecondMapCensus) SecondCensusTotals {
	var t SecondCensusTotals
	for _, c := range cs {
		t.Maps++
		if c.Err != "" {
			continue
		}
		t.Started++
		if c.Run.Err == "" && c.Run.Ticks == SecondCensusTicks {
			t.Ran++
		}
		switch c.Run.Outcome {
		case sim.OutcomeWon:
			t.Won++
		case sim.OutcomeLost:
			t.Lost++
		}
		t.Placements += c.Placements
		t.Withdrawn += c.Withdrawn
		t.NoRowPlacements += c.NoRowPlacements
		t.AuthoredHP += c.AuthoredHP
		t.BornFallen += c.BornFallen
		t.NoStructureRow += c.NoStructureRowKeys
		t.NoItemRow += len(c.NoItemRow)
		if len(c.NoSpellRule) > 0 {
			t.NoSpellMaps++
		}
		t.Events += len(c.Events)
		t.NoEventText += len(c.NoEventText)
		t.NoReasonText += len(c.NoReasonText)
		t.Checks += c.Checks
		t.Instants += c.Instants
		t.Triggers += c.Triggers
		for _, g := range c.Gaps {
			t.Gaps += g.Nodes
		}
		t.Inert += len(c.Inert)
		t.SharedNodes += c.SharedNodes
		for i := range c.Omitted {
			t.Omitted[i].Nodes += c.Omitted[i].Nodes
			t.Omitted[i].Triggers += c.Omitted[i].Triggers
			t.Omitted[i].ActionTriggers += c.Omitted[i].ActionTriggers
		}
		if c.Omitted[CauseFixture] != (SecondOmission{}) {
			t.FixtureMaps++
		}
		if c.Departure.EngineEntry {
			t.EntryMaps++
		}
		if c.Departure.EngineWin {
			t.WinMaps++
		}
		for _, e := range c.Departure.Exits {
			t.Exits++
			if len(e.Gates) > 0 {
				t.GatedExits++
			}
			if e.Engine {
				t.EngineExits++
			}
		}
		if len(c.Blockers()) == 0 {
			t.Ready++
		}
	}
	return t
}

// WriteSecondCensus prints one tab-separated row per map, then the totals.
func WriteSecondCensus(w io.Writer, cs []SecondMapCensus) error {
	head := "mission\tstart\trun\toutcome\tplaced\twithdrawn\tauthored_hp\tborn_fallen\tno_row\tstructures\tno_structure_row\tloot\tloot_dropped\tno_item_row\tspells\tno_spell_rule\t" +
		"events\tno_event_text\treasons\tno_reason_text\tchecks\tinstants\ttriggers\tshared_nodes\tshared_checks\tshared_instants\tgaps\tinert\t" +
		"fixture\tdynamic\twithdrawn_ref\tabsent\tstructure_ref\tbank_slots\t" +
		"case\texits\tproducer\tengine_entry\tengine_win\tengine_adds\tengine_exits\tengine_save\tblockers\n"
	if _, err := io.WriteString(w, head); err != nil {
		return err
	}
	for _, c := range cs {
		start, run := "ok", "ok"
		if c.Err != "" {
			start, run = c.Err, "-"
		} else if c.Run.Err != "" {
			run = c.Run.Err
		}
		d := c.Departure
		row := []string{strconv.Itoa(c.Mission), start, run, outcomeName(c.Run),
			strconv.Itoa(c.Placements), strconv.Itoa(c.Withdrawn), strconv.Itoa(c.AuthoredHP), strconv.Itoa(c.BornFallen), strconv.Itoa(c.NoRowPlacements),
			strconv.Itoa(c.Structures), strconv.Itoa(c.NoStructureRowKeys), strconv.Itoa(c.LootItems), strconv.Itoa(c.LootDropped),
			secondList(c.NoItemRow), secondList(c.Spells), secondList(c.NoSpellRule),
			secondList(c.Events), secondList(c.NoEventText), secondList(c.Reasons), secondList(c.NoReasonText),
			strconv.Itoa(c.Checks), strconv.Itoa(c.Instants), strconv.Itoa(c.Triggers),
			strconv.Itoa(c.SharedNodes), secondList(c.SharedChecks), secondList(c.SharedInstants), secondGapList(c.Gaps), secondList(c.Inert)}
		for _, o := range c.Omitted {
			row = append(row, fmt.Sprintf("%d/%d/%d", o.Nodes, o.Triggers, o.ActionTriggers))
		}
		row = append(row, secondList(c.BankSlots), secondYes(d.Case), secondExitList(d.Exits, false), d.Producer,
			secondYes(d.EngineEntry), secondYes(d.EngineWin), strings.Join(d.EngineAdds, ","), secondExitList(d.Exits, true), secondYes(d.EngineSave), strings.Join(c.Blockers(), ","))
		if _, err := io.WriteString(w, strings.Join(row, "\t")+"\n"); err != nil {
			return err
		}
	}
	t := SecondTotals(cs)
	_, err := fmt.Fprintf(w, "totals maps=%d started=%d ran%d=%d won=%d lost=%d placed=%d withdrawn=%d authored_hp=%d born_fallen=%d no_row=%d no_structure_row=%d no_item_row=%d no_spell_maps=%d events=%d no_event_text=%d no_reason_text=%d "+
		"checks=%d instants=%d triggers=%d shared_nodes=%d gaps=%d inert=%d fixture=%s dynamic=%s withdrawn_ref=%s absent=%s structure_ref=%s fixture_maps=%d engine_entry=%d engine_win=%d exits=%d gated_exits=%d engine_exits=%d ready=%d\n",
		t.Maps, t.Started, SecondCensusTicks, t.Ran, t.Won, t.Lost, t.Placements, t.Withdrawn, t.AuthoredHP, t.BornFallen, t.NoRowPlacements, t.NoStructureRow, t.NoItemRow, t.NoSpellMaps, t.Events, t.NoEventText, t.NoReasonText,
		t.Checks, t.Instants, t.Triggers, t.SharedNodes, t.Gaps, t.Inert, secondOmit(t.Omitted[CauseFixture]), secondOmit(t.Omitted[CauseDynamic]),
		secondOmit(t.Omitted[CauseWithdrawn]), secondOmit(t.Omitted[CauseAbsent]), secondOmit(t.Omitted[CauseStructure]),
		t.FixtureMaps, t.EntryMaps, t.WinMaps, t.Exits, t.GatedExits, t.EngineExits, t.Ready)
	return err
}

func secondOmit(o SecondOmission) string {
	return fmt.Sprintf("%d/%d/%d", o.Nodes, o.Triggers, o.ActionTriggers)
}

func secondYes(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func outcomeName(r SecondRun) string {
	switch r.Outcome {
	case sim.OutcomeWon:
		return fmt.Sprintf("won@%d", r.DecidedAt)
	case sim.OutcomeLost:
		return fmt.Sprintf("lost@%d:reason%d", r.DecidedAt, r.Reason)
	}
	return "undecided"
}

func secondList[T int | int32 | uint16](s []T) string {
	if len(s) == 0 {
		return "-"
	}
	out := make([]string, len(s))
	for i, v := range s {
		out[i] = strconv.Itoa(int(v))
	}
	return strings.Join(out, ",")
}

func secondGapList(gs []SecondOpGap) string {
	if len(gs) == 0 {
		return "-"
	}
	out := make([]string, len(gs))
	for i, g := range gs {
		kind := "instant"
		if g.Check {
			kind = "check"
		}
		out[i] = fmt.Sprintf("%s%d", kind, g.Op)
		if g.Sub != 0 {
			out[i] += fmt.Sprintf(".%d", g.Sub)
		}
		out[i] += fmt.Sprintf("x%d/%dtrig", g.Nodes, g.Triggers)
	}
	return strings.Join(out, ",")
}
