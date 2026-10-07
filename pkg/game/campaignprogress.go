package game

import (
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
)

// campaignProgress is the mutable game-layer copy of an original save's
// campaign projection. Campaign remains the immutable scenario.reg definition;
// this value is the loaded player's position inside that definition.
type campaignProgress struct {
	campaignRecords

	mercenaryWorking  [16]int
	mercenaryPristine [16]int
	mercenaryHired    [16]bool
	mercenaries       []int
	permanent         []int
	innNPC            []int
	innMission        []int
	tcMission         []int
	shopMission       []int
	documents         []Document
	selected          int
	autoGet           uint32
	last              int
	firstMapPoint     bool
	missionTime       uint32
	scoreEvents       uint32
	scoreEventsKnown  bool
	markers           []campaignProgressMarker
	markerSelected    bool
}

type campaignProgressRecord struct {
	mission         int
	mapObject       int
	payment         int
	shopMin         int
	shopMax         int
	announced       bool
	addHero         []int
	enableMercenary []int
	age             int
}

type campaignProgressMarker struct {
	value   int
	picture string
	field0  uint32
	field1  uint32
}

func intsFromU16(in []uint16) []int {
	if len(in) == 0 {
		return nil
	}
	out := make([]int, len(in))
	for i, v := range in {
		out[i] = int(v)
	}
	return out
}

// innArrayError is a campaign store whose InnNPC and InnMission arrays differ
// in length; the two pair by position.
type innArrayError struct{ npcs, missions int }

func (e innArrayError) Error() string {
	return fmt.Sprintf("original campaign inn arrays are %d NPCs and %d missions", e.npcs, e.missions)
}

func progressRecordFromSAV(in sav.CampaignRecord) campaignProgressRecord {
	return campaignProgressRecord{
		mission: int(in.Mission), mapObject: int(in.MapObject), payment: int(in.Payment),
		shopMin: int(in.ShopMin), shopMax: int(in.ShopMax), announced: in.Announced,
		addHero: intsFromU16(in.AddHero), enableMercenary: intsFromU16(in.EnableMercenary),
		age: int(in.Age),
	}
}

func validateMercenaryTypes(name string, values []int) error {
	for i, typ := range values {
		if typ <= 0 || typ >= 16 {
			return fmt.Errorf("original campaign %s[%d] type %d is outside 1..15", name, i, typ)
		}
	}
	return nil
}

func validateCandidateMissions(c Campaign, main int, records map[int]bool, name string, values []int, allowZero bool) error {
	for i, mission := range values {
		if allowZero && mission == 0 {
			continue
		}
		futureMain := mission > main && mission%mainMissionStride == 0 && containsMission(c.Main, mission)
		if mission <= 0 || !records[mission] && !futureMain {
			return fmt.Errorf("original campaign %s[%d] mission %d has no live main or retained-child record", name, i, mission)
		}
	}
	return nil
}

func campaignTownOpen(c Campaign, main int) bool {
	first, ok := c.TownBegins()
	return ok && main >= first
}

// validateCampaignLocation joins the active save header to the campaign
// projection before either is installed. A mission save must name the selected
// live record; a town-only save cannot exist before the campaign's town
// boundary.
func validateCampaignLocation(c Campaign, p *campaignProgress, mission int, source string) error {
	if p == nil {
		return nil
	}
	if mission == 0 {
		first, ok := c.TownBegins()
		if !ok {
			return fmt.Errorf("%s is between missions but the campaign declares no town boundary", source)
		}
		if p.main.mission < first {
			return fmt.Errorf("%s is between missions before campaign town begins at mission %d", source, first)
		}
		return nil
	}
	if p.record(mission) == nil {
		return fmt.Errorf("%s active mission %d has no live main or retained-child record", source, mission)
	}
	if p.selected != mission {
		return fmt.Errorf("%s active mission %d does not match selected mission %d", source, mission, p.selected)
	}
	return nil
}

// campaignProgressFromSAV validates the complete projection before returning
// any state. Callers install only the returned value, so a malformed field set
// cannot partially replace a running campaign. AutoGetMission is the main
// record's own declaration, as the record loader writes it (REG-SCN-063); a
// file's stale value is not carried (DIV-1497).
func campaignProgressFromSAV(c Campaign, in sav.CampaignProjection) (*campaignProgress, error) {
	if in.Main.Mission == 0 || !containsMission(c.Main, int(in.Main.Mission)) {
		return nil, fmt.Errorf("original campaign main mission %d is not a declared main mission", in.Main.Mission)
	}
	if len(in.MercenaryWorking) != 15 || len(in.MercenaryPristine) != 15 || len(in.MercenaryHired) != 15 {
		return nil, fmt.Errorf("original campaign mercenary arrays are %d/%d/%d, want 15/15/15",
			len(in.MercenaryWorking), len(in.MercenaryPristine), len(in.MercenaryHired))
	}
	if len(in.InnNPC) != len(in.InnMission) {
		return nil, innArrayError{npcs: len(in.InnNPC), missions: len(in.InnMission)}
	}

	p := &campaignProgress{
		campaignRecords:  campaignRecords{main: progressRecordFromSAV(in.Main)},
		mercenaries:      intsFromU16(in.Mercenaries),
		permanent:        intsFromU16(in.PermanentMercenaries),
		innNPC:           intsFromU16(in.InnNPC),
		innMission:       intsFromU16(in.InnMission),
		tcMission:        intsFromU16(in.TCMission),
		shopMission:      intsFromU16(in.ShopMission),
		selected:         int(in.SelectedMission),
		autoGet:          autoGetMissionValue(c, int(in.Main.Mission)),
		last:             int(in.LastMission),
		firstMapPoint:    in.FirstMapPoint,
		missionTime:      in.MissionTime,
		scoreEvents:      in.ScoreEvents,
		scoreEventsKnown: in.ScoreEventsKnown,
		markerSelected:   len(in.Markers) > 0,
	}
	for i := 1; i < 16; i++ {
		p.mercenaryWorking[i] = int(in.MercenaryWorking[i-1])
		p.mercenaryPristine[i] = int(in.MercenaryPristine[i-1])
		p.mercenaryHired[i] = in.MercenaryHired[i-1]
	}
	if err := validateMercenaryTypes("Mercenaries", p.mercenaries); err != nil {
		return nil, err
	}
	if err := validateMercenaryTypes("permanent unlocks", p.permanent); err != nil {
		return nil, err
	}
	if err := validateMercenaryTypes("main EnableMercenary", p.main.enableMercenary); err != nil {
		return nil, err
	}
	seen := map[int]bool{p.main.mission: true}
	selected := p.selected == p.main.mission
	p.children = make([]campaignProgressRecord, len(in.Children))
	for i, child := range in.Children {
		r := progressRecordFromSAV(child)
		if r.mission <= 0 || !(containsMission(c.Side, r.mission) || containsMission(c.Offered, r.mission)) {
			return nil, fmt.Errorf("original campaign child %d mission %d is not a declared side mission", i, r.mission)
		}
		if r.age < 0 || r.age >= 2 {
			return nil, fmt.Errorf("original campaign child %d age %d is not retained", i, r.age)
		}
		if seen[r.mission] {
			return nil, fmt.Errorf("original campaign mission %d occurs more than once", r.mission)
		}
		if err := validateMercenaryTypes(fmt.Sprintf("child %d EnableMercenary", i), r.enableMercenary); err != nil {
			return nil, err
		}
		seen[r.mission] = true
		selected = selected || p.selected == r.mission
		p.children[i] = r
	}
	if p.selected <= 0 || !selected {
		return nil, fmt.Errorf("original campaign selected mission %d is neither main nor retained child", p.selected)
	}
	if err := validateCandidateMissions(c, p.main.mission, seen, "InnMission", p.innMission, true); err != nil {
		return nil, err
	}
	if err := validateCandidateMissions(c, p.main.mission, seen, "TCMission", p.tcMission, false); err != nil {
		return nil, err
	}
	if err := validateCandidateMissions(c, p.main.mission, seen, "ShopMission", p.shopMission, false); err != nil {
		return nil, err
	}
	for i, d := range in.Documents {
		if d.Kind > DocumentText {
			return nil, fmt.Errorf("original campaign document %d kind %d is outside 0..1", i, d.Kind)
		}
		p.documents = append(p.documents, Document{Value: int(d.Value), Kind: int(d.Kind)})
	}
	for _, m := range in.Markers {
		p.markers = append(p.markers, campaignProgressMarker{
			value: int(m.Value), picture: m.Picture, field0: m.Field0, field1: m.Field1,
		})
	}
	return p, nil
}

func (p *campaignRecords) record(mission int) *campaignProgressRecord {
	if p == nil {
		return nil
	}
	if p.main.mission == mission {
		return &p.main
	}
	for i := range p.children {
		if p.children[i].mission == mission {
			return &p.children[i]
		}
	}
	return nil
}

func (p *campaignProgress) chapter() Chapter {
	if p == nil {
		return Chapter{}
	}
	return Chapter{
		Mission: p.main.mission, InnNPC: append([]int(nil), p.innNPC...),
		Inn: append([]int(nil), p.innMission...), Shop: append([]int(nil), p.shopMission...),
		School: append([]int(nil), p.tcMission...), ShopMin: p.main.shopMin, ShopMax: p.main.shopMax,
		Payment: p.main.payment, AddHero: append([]int(nil), p.main.addHero...),
		Mercenaries:     append([]int(nil), p.mercenaries...),
		EnableMercenary: append([]int(nil), p.main.enableMercenary...),
	}
}

func (p *campaignProgress) lowerMainBlocked(mission int) bool {
	return p != nil && mission > 0 && mission%mainMissionStride == 0 && mission < p.main.mission
}

// selectedMarkers reads the mission identities TAVERN-024 guards in the
// persisted marker collection TOWN-123 identifies. Presentation coalesces
// imported duplicates in list order (DIV-904); projection retains the payloads.
func (p *campaignProgress) selectedMarkers() []int {
	if p == nil || len(p.markers) == 0 {
		return nil
	}
	seen := make(map[int]bool, len(p.markers))
	var out []int
	for _, m := range p.markers {
		if m.value <= 0 || seen[m.value] {
			continue
		}
		seen[m.value] = true
		out = append(out, m.value)
	}
	return out
}

func u16sFromInts(in []int) []uint16 {
	if len(in) == 0 {
		return nil
	}
	out := make([]uint16, len(in))
	for i, v := range in {
		out[i] = uint16(v)
	}
	return out
}

func (r campaignProgressRecord) savRecord(child bool) sav.CampaignRecord {
	out := sav.CampaignRecord{
		Mission: uint32(r.mission), MapObject: uint32(r.mapObject), Payment: uint32(r.payment),
		ShopMin: uint32(r.shopMin), ShopMax: uint32(r.shopMax), Announced: r.announced,
		AddHero: u16sFromInts(r.addHero), EnableMercenary: u16sFromInts(r.enableMercenary),
	}
	if child {
		out.Age = uint32(r.age)
	}
	return out
}

// projection returns a detached value for againrom's additive save envelope.
// It preserves the original typed campaign state without copying raw SAV bytes.
func (p *campaignProgress) projection() sav.CampaignProjection {
	if p == nil {
		return sav.CampaignProjection{}
	}
	out := sav.CampaignProjection{
		Main: p.main.savRecord(false), Mercenaries: u16sFromInts(p.mercenaries),
		PermanentMercenaries: u16sFromInts(p.permanent), InnNPC: u16sFromInts(p.innNPC),
		InnMission: u16sFromInts(p.innMission), TCMission: u16sFromInts(p.tcMission),
		ShopMission: u16sFromInts(p.shopMission), SelectedMission: uint32(p.selected),
		AutoGetMission: p.autoGet, LastMission: uint32(p.last),
		FirstMapPoint: p.firstMapPoint, MissionTime: p.missionTime,
		ScoreEvents: p.scoreEvents, ScoreEventsKnown: p.scoreEventsKnown,
	}
	for i := 1; i < 16; i++ {
		out.MercenaryWorking = append(out.MercenaryWorking, uint16(p.mercenaryWorking[i]))
		out.MercenaryPristine = append(out.MercenaryPristine, uint16(p.mercenaryPristine[i]))
		out.MercenaryHired = append(out.MercenaryHired, p.mercenaryHired[i])
	}
	for _, child := range p.children {
		out.Children = append(out.Children, child.savRecord(true))
	}
	for _, d := range p.documents {
		out.Documents = append(out.Documents, sav.CampaignDocument{Value: uint32(d.Value), Kind: uint32(d.Kind)})
	}
	for _, m := range p.markers {
		out.Markers = append(out.Markers, sav.CampaignMarker{
			Value: uint32(m.value), Picture: m.picture, Field0: m.field0, Field1: m.field1,
		})
	}
	return out
}

func progressRecordFromCampaign(c Campaign, mission int) campaignProgressRecord {
	ch := c.Chapters[mission]
	return campaignProgressRecord{
		mission: mission, mapObject: c.MapObjects[mission], payment: ch.Payment,
		shopMin: ch.ShopMin, shopMax: ch.ShopMax,
		addHero:         append([]int(nil), ch.AddHero...),
		enableMercenary: append([]int(nil), ch.EnableMercenary...),
	}
}

func candidateSides(ch Chapter) []int {
	seen := make(map[int]bool)
	var out []int
	for _, list := range [][]int{ch.Inn, ch.School, ch.Shop} {
		for _, mission := range list {
			if mission <= 0 || mission%mainMissionStride == 0 || seen[mission] {
				continue
			}
			seen[mission] = true
			out = append(out, mission)
		}
	}
	return out
}

func (p *campaignProgress) advanceMain(c Campaign, mission int) {
	next := mission + mainMissionStride
	if !containsMission(c.Main, next) {
		p.selected = p.main.mission
		return
	}
	// REG-SCN-063: a won record whose AutoGetMission names the next mission
	// loads it and latches its announce flag; loading writes +0x110 anew.
	routed := p.autoGet == uint32(next)
	p.loadMain(c, next, routed)
}

func (p *campaignProgress) loadMain(c Campaign, main int, announced bool) {
	ch := c.Chapters[main]
	p.campaignRecords.loadMain(c, main, announced)
	p.autoGet = autoGetMissionValue(c, main)
	p.mercenaries = append(p.mercenaries[:0], ch.Mercenaries...)
	p.innNPC = append(p.innNPC[:0], ch.InnNPC...)
	p.innMission = append(p.innMission[:0], ch.Inn...)
	p.tcMission = append(p.tcMission[:0], ch.School...)
	p.shopMission = append(p.shopMission[:0], ch.Shop...)
	p.selected = main
}

// complete advances the current main record or removes a retained side
// record. The original record has no completed-side set
// (SAV-CAMPAIGN-077/SAV-CAMPAIGN-082).
func (p *campaignProgress) complete(c Campaign, mission int) (side bool, ok bool) {
	if p == nil || p.lowerMainBlocked(mission) {
		return false, false
	}
	for i := range p.children {
		if p.children[i].mission != mission {
			continue
		}
		for _, typ := range p.children[i].enableMercenary {
			if !containsMission(p.permanent, typ) {
				p.permanent = append(p.permanent, typ)
			}
		}
		copy(p.children[i:], p.children[i+1:])
		p.children = p.children[:len(p.children)-1]
		p.selected = p.main.mission
		return true, true
	}
	if mission == p.main.mission {
		for _, typ := range p.main.enableMercenary {
			if !containsMission(p.permanent, typ) {
				p.permanent = append(p.permanent, typ)
			}
		}
		p.main.enableMercenary = nil
		p.advanceMain(c, mission)
		return false, true
	}
	return false, false
}

func (p *campaignProgress) takeAddHeroes(chapter int) []int {
	return p.takeAddHeroesExcept(chapter, nil)
}

// takeAddHeroesExcept hands out the pending grants except those keep names;
// those stay in the main record's array, which the save writes.
func (p *campaignProgress) takeAddHeroesExcept(chapter int, keep func(npc int) bool) []int {
	if p == nil || p.main.mission != chapter || len(p.main.addHero) == 0 {
		return nil
	}
	var out, left []int
	for _, npc := range p.main.addHero {
		if keep != nil && keep(npc) {
			left = append(left, npc)
			continue
		}
		out = append(out, npc)
	}
	p.main.addHero = left
	return out
}

// takeAddHero removes the one pending grant of npc from the main record.
func (p *campaignProgress) takeAddHero(chapter, npc int) bool {
	if p == nil || p.main.mission != chapter {
		return false
	}
	i := slices.Index(p.main.addHero, npc)
	if i < 0 {
		return false
	}
	p.main.addHero = slices.Delete(slices.Clone(p.main.addHero), i, i+1)
	if len(p.main.addHero) == 0 {
		p.main.addHero = nil
	}
	return true
}

func (p *campaignProgress) selectMission(mission int) bool {
	if p == nil || p.record(mission) == nil {
		return false
	}
	p.selected = mission
	p.firstMapPoint = false
	p.markerSelected = true
	return true
}

func (p *campaignProgress) announce(mission int) {
	if r := p.record(mission); r != nil {
		r.announced = true
	}
}

// take consumes the building candidate before latching the selected record.
// Invalid missions leave the candidate array unchanged.
func (p *campaignProgress) take(c Campaign, b TownBuilding, index int) (int, bool) {
	if p == nil || index < 0 {
		return 0, false
	}
	var mission int
	switch b {
	case TownTavern:
		if index >= len(p.innMission) || index >= len(p.innNPC) || p.innMission[index] <= 0 {
			return 0, false
		}
		mission = p.innMission[index]
		if !p.accepts(c, mission) {
			return 0, false
		}
		p.innMission = append(p.innMission[:index:index], p.innMission[index+1:]...)
		p.innNPC = append(p.innNPC[:index:index], p.innNPC[index+1:]...)
	case TownShop:
		if index >= len(p.shopMission) || p.shopMission[index] <= 0 {
			return 0, false
		}
		mission = p.shopMission[index]
		if !p.accepts(c, mission) {
			return 0, false
		}
		p.shopMission = append(p.shopMission[:index:index], p.shopMission[index+1:]...)
	case TownSchool:
		if index >= len(p.tcMission) || p.tcMission[index] <= 0 {
			return 0, false
		}
		mission = p.tcMission[index]
		if !p.accepts(c, mission) {
			return 0, false
		}
		p.tcMission = append(p.tcMission[:index:index], p.tcMission[index+1:]...)
	default:
		return 0, false
	}
	if p.record(mission) == nil {
		p.loadMain(c, mission, false)
	}
	p.announce(mission)
	return mission, true
}
