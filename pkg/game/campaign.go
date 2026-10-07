package game

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"againrom/pkg/formats/reg"
	"againrom/pkg/vfs"
)

// ScenarioRegistry is the ADDRESS the campaign's own spine is loaded from,
// composed exactly as NPCRegistry beside it is (mission.go): the campaign
// container's identity segment and the entry inside it, so a renamed container
// moves the maps, the NPC table and this together.
var ScenarioRegistry = scenarioPrefix + "scenario.reg"

// The three keys a town building's offer list is written under, and the one
// registry section every mission record is a child of. Each key name is the
// registry's own, spelled once here; nothing else in this package spells them.
const (
	innMissionKey  = "InnMission"
	shopMissionKey = "ShopMission"
	tcMissionKey   = "TCMission"

	// autoGetMissionKey is the key a [Mission<n>] section declares its own
	// successor under. It is spelled once here, same as the three offer keys
	// above.
	autoGetMissionKey = "AutoGetMission"

	missionSection    = "Mission"
	generalSection    = "General"
	totalMissionsKey  = "TotalMissions"
	mainMissionStride = 10

	// The four keys a chapter carries beside its three offer lists, each
	// spelled once here like the four above it.
	//
	// innNPCKey IS THE INN'S OTHER HALF and it is read for the first time in
	// this story. REG-SCN-064 closed REG-SCN-059's unpaired InnNPC:
	// InnNPC[i] pairs POSITIONALLY with InnMission[i], so a tavern that
	// listed missions without the NPCs holding them would be listing half a
	// mechanism.
	innNPCKey          = "InnNPC"
	shopMinKey         = "ShopMinPrice"
	shopMaxKey         = "ShopMaxPrice"
	mapObjectKey       = "MapObject"
	paymentKey         = "Payment"
	addHeroKey         = "AddHero"
	mercenaryCountKey  = "MercenaryCount"
	mercenariesKey     = "Mercenaries"
	enableMercenaryKey = "EnableMercenary"

	// The two keys a chapter grants campaign documents under (REG-SCN-097).
	//
	// THE SECOND ONE IS SPELLED AS THE FILE STORES IT AND NOT AS IT READS.
	// A .reg node's name field is sixteen bytes and the format cuts a longer
	// name at fifteen, recording the cut in the node's own kind word
	// (pkg/formats/reg's flagTruncated). `AddTextDocument` is exactly fifteen
	// characters and survives; `AddPictureDocument` does not, and every
	// shipped scenario stores it as `AddPictureDocum`. A reader spelling the
	// untruncated name finds no key at all, on either root, and the picture
	// element mission 60 grants would silently never arrive.
	addTextDocumentKey    = "AddTextDocument"
	addPictureDocumentKey = "AddPictureDocum"
)

// Campaign is the mission set `scenario.reg` declares, and the ONE thing
// this tree reads out of it: which missions exist, which of them are main
// missions, which of them a town building hands out, and — per section —
// the successor it declares under AutoGetMission.
//
// IT IS NOT A TOWN AND IT DOES NOT PRETEND TO BE. The campaign's real spine is
// three town buildings — a tavern, a shop and a school — each holding an offer
// list the player consumes by visiting it (REG-SCN-064). That is a subsystem
// and this tree does not have it. What this type reads is the DATA those
// buildings are driven from, which is enough to say which missions exist and
// where the town's part of the campaign begins, and not enough to say which
// mission a player would be offered next inside it.
//
// EVERY NUMBER HERE COMES OUT OF THE FILE. There is no mission number, no
// count and no stride written as a value in this package: the sections are
// enumerated, the offer keys are read, and what is main and what is offered
// falls out of both. A modded scenario answers for its own contents, which is
// the standing rule against hardcoding applied to the one place a literal
// ladder would have been easiest.
type Campaign struct {
	// Last retains each installed LastMission flag, including side sections.
	Last map[int]bool
	// Declared is [General] TotalMissions as the file states it, or zero
	// where the key is absent. It is READ AND REPORTED AND NEVER USED AS A
	// BOUND — the sections are what exist, and a count that disagreed with
	// them would be a fact about the file worth seeing rather than a reason
	// to drop a section. Whether the original reads this key at all is
	// itself open (REG-SCN-059's own Unknown).
	Declared int

	// MercenaryCount is the fifteen-entry initial tavern pool from [General].
	// The index is the zero-based form of the game's one-based mercenary type.
	MercenaryCount []int

	// Main is every [Mission<n>] section whose number is a whole multiple of
	// ten, ascending: the campaign's main missions, which REG-SCN-059
	// establishes are numbered by tens. A stock scenario holds fifteen.
	Main []int

	// Side is every other [Mission<n>] section, ascending — nine on a stock
	// scenario, and each of them a mission a building offers beside a main
	// one rather than a step of the campaign.
	Side []int

	// Offered is every mission named by any of the three building keys, over
	// every section, ascending and distinct, with the zero sentinel dropped
	// — InnMission's own "this NPC has nothing to give" (REG-SCN-064).
	//
	// IT IS THE SET THE TOWN HANDS OUT, and its LOWER BOUND is the only
	// thing in this file that says where the town's part of the campaign
	// begins. On a stock scenario it holds twenty-six missions and starts at
	// thirty: missions 10 and 20 are named by no building at all.
	Offered []int

	// Auto is the successor each [Mission<n>] section declares under its own
	// AutoGetMission key, keyed by the mission that declares it.
	//
	// AutoAdvance is how this is read.
	Auto map[int]int

	// MapObjects is each mission record's campaign-map object. It is kept
	// apart from Chapters because a section may carry only this record field
	// and still have no town data.
	MapObjects map[int]int

	// Chapters is every [Mission<n>] section's own building data, keyed by the
	// mission that declares it.
	//
	// IT DOES NOT REPLACE Offered AND Offered IS NOT DERIVED FROM IT. Offered
	// is the merged, deduplicated SET a town hands out, and it is what
	// NextMission, TownBegins and Offer.Town all read; this map keeps the
	// three lists APART, per section, because a town has three buildings and
	// each one hands out its own. Both are read from the same loop over the
	// same sections and neither is computed from the other, so the answer
	// NextMission gave before this story is the answer it gives after it —
	// see this file's own doc above: "IT IS NOT A TOWN AND IT DOES NOT
	// PRETEND TO BE" is now false of the DATA and still true of this type,
	// which reads and does not decide.
	//
	// A section carrying none of the six keys gets no entry, so the map holds
	// the town chapters and not every mission.
	Chapters map[int]Chapter

	// TransitionRewards is an authored reward paid when a completed mission
	// crosses a campaign transition. It is deliberately separate from
	// Chapter.Payment: the latter is shipped scenario.reg data, while this map
	// is a configurable product rule and must never make an absent Payment key
	// look present.
	TransitionRewards map[int]int
}

// firstTownTransitionReward is the owner's authored reward for completing the
// prologue and entering the town. It is not an original-game fact and is not
// read from scenario.reg.
const firstTownTransitionReward = 500

// WithDefaultTransitionRewards installs the product's default authored reward
// policy. The transition mission is derived from the campaign: it is the last
// main mission before the first mission any town building offers. Callers may
// replace TransitionRewards afterwards; Town reads the map and holds no second
// copy of the policy.
func WithDefaultTransitionRewards(c Campaign) Campaign {
	firstTown, ok := c.TownBegins()
	if !ok {
		return c
	}
	transition := 0
	for _, n := range c.Main {
		if n >= firstTown {
			break
		}
		transition = n
	}
	if transition == 0 {
		return c
	}
	c.TransitionRewards = map[int]int{transition: firstTownTransitionReward}
	return c
}

// Reward is the total gold a completion pays: the scenario's own Payment and
// the separately disclosed authored transition reward.
func (c Campaign) Reward(n int) int {
	return c.Chapters[n].Payment + c.TransitionRewards[n]
}

// Chapter is one [Mission<n>] section's town data: the three offer lists
// REG-SCN-064 establishes, the NPC list the inn's own pairs with, the shop's
// two price bounds and the mission's declared payment.
//
// THE ARRAYS ARE THE FILE'S OWN ORDER AND NOTHING IS DROPPED FROM THEM — not
// the zero sentinel, not a duplicate, not a value another section already
// named. Offered drops all three because it is a set; this is a chapter, and
// InnMission[i] == 0 is a POSITION in it (the NPC at that index has nothing to
// give) rather than an absence. A reader that compacted the arrays would break
// the positional pairing REG-SCN-064 closed.
type Chapter struct {
	// Mission is the section this chapter was read from.
	Mission int

	// InnNPC and Inn are the tavern's two parallel arrays: the NPC at
	// InnNPC[i] holds the mission at Inn[i], and Inn[i] == 0 is the
	// sentinel for an NPC with nothing to give (REG-SCN-064).
	//
	// THEY MAY BE OF DIFFERENT LENGTHS HERE and are equal in 22/22 shipped
	// sections. Nothing pads or truncates them at the read: a modded
	// scenario's own file is what it is, and the pairing walks the shorter
	// of the two (plan R-1) at the one place that pairs them.
	InnNPC []int
	Inn    []int

	// Shop and School are the other two buildings' lists, read from
	// ShopMission and TCMission. `TC` is the school (REG-SCN-064).
	Shop   []int
	School []int

	// ShopMin and ShopMax are the chapter's own price bounds, and they are
	// carried for the one reason the shop screen needs them: a room that could
	// say nothing true about itself would be an empty rectangle. Nothing prices
	// anything with them.
	ShopMin int
	ShopMax int

	// Payment is what winning this mission pays (REG-SCN-067, Medium on the
	// reward reading). Zero is a section that declares none, which is 15 of
	// the shipped 24.
	Payment int
	AddHero []int

	// Mercenaries is the roster named by this chapter. EnableMercenary is the
	// permanent unlock reward applied when this section's mission is won.
	Mercenaries     []int
	EnableMercenary []int

	// TextDocuments and PictureDocuments are the campaign document values
	// this chapter grants, in the file's own order (REG-SCN-097). A value is
	// a NUMBER THAT FORMATS INTO A RESOURCE PATH and not an index into
	// anything here (MISSION-DOC-021).
	//
	// THEY ARE KEPT APART BECAUSE THE KIND IS THE KEY'S, not the value's:
	// the same value 1 is granted by both keys on a stock scenario —
	// mission 10 grants text 1 and mission 60 grants picture 1 — and they
	// resolve to two different files. Merging them into one list would
	// need a kind field per element anyway, which is what Document (town.go)
	// is; the split here keeps this type a transcription of the section and
	// leaves the pairing to the one place that appends.
	//
	// Nothing is dropped: a zero, a duplicate and a value another section
	// already named are all kept, on Chapter's own rule above.
	TextDocuments    []int
	PictureDocuments []int
}

// AutoAdvance is the successor mission n's own [Mission<n>] section declares
// — read off Auto, never derived from Main or computed by NextMission's
// ascending order.
func (c Campaign) AutoAdvance(n int) (int, bool) {
	v, ok := c.Auto[n]
	return v, ok
}

// autoGetMissionValue is the SAV Campaign+0x110 (AutoGetMission) value a
// mission-context or city-context document carries for mission n: the
// scenario's own declared successor (REG-SCN-063: 10's own [Mission10]
// declares 20, and no other stock section declares one at all, mission 20
// and every mission from 30 on returning to the town instead), or the
// original's own no-successor sentinel -1 (0xFFFFFFFF) when the scenario
// declares none. Every writer of this field goes through this one function
// so a section that DID declare a successor for 20 or a >=30 mission is
// honoured here rather than silently overridden by a literal.
func autoGetMissionValue(c Campaign, n int) uint32 {
	if next, ok := c.AutoAdvance(n); ok {
		return uint32(next)
	}
	return 0xFFFFFFFF
}

// Offer is the answer to "what follows this mission": the mission number, and
// whether naming it is this tree's own placeholder or the campaign's own.
type Offer struct {
	// Mission is the next main mission.
	Mission int

	// Town reports that the town is what would have decided this, and
	// therefore that Mission is a PLACEHOLDER STANDING IN FOR THE GATE
	// CHOICE rather than an order anything has established.
	//
	// It is true exactly when Mission is one of the missions a building
	// offers — read off Campaign.Offered, never off a mission number
	// compared against a literal here.
	Town bool
}

// NextMission answers the mission this tree offers once mission n is won: the
// next main mission the registry declares, ascending.
//
// THE ORDER IS AUTHORED AND THE FLAG SAYS HOW MUCH OF IT IS. Nobody has read
// the original's mission-advance routine — AI-CENSUS-048's own confidence line
// says the ordering by number is not a read of it — so ascending order is
// OURS. What the file does establish is where the town takes over: the missions
// no building offers are the ones the town cannot have handed out, and on a
// stock scenario those are exactly the two the owner reports are played before
// the town is ever reached. Below that boundary the successor is the campaign's
// own and this tree agrees with it; at or above it, Offer.Town is set and the
// number returned is scaffolding.
//
// A SIDE MISSION'S SUCCESSOR IS THE NEXT MAIN ONE, because a side mission is
// not a step of the campaign — it is something a building offered beside one —
// so finishing it advances nothing and the ladder continues from where it was.
//
// The second result is false when n is at or past the last main mission the
// registry declares, and for a campaign that declares none at all.
func (c Campaign) NextMission(n int) (Offer, bool) {
	for _, m := range c.Main {
		if m <= n {
			continue
		}
		return Offer{Mission: m, Town: c.offers(m)}, true
	}
	return Offer{}, false
}

// offers reports whether any building's list names mission n.
func (c Campaign) offers(n int) bool {
	i := sort.SearchInts(c.Offered, n)
	return i < len(c.Offered) && c.Offered[i] == n
}

// TownBegins is the lowest mission any building offers, and whether there is
// one: the boundary between the prologue the campaign plays before a town and
// the part a town drives.
//
// IT IS A READ AND NOT A CONSTANT. Missions 10 and 20 are the answer on a
// stock scenario, and they are the answer because no building names them —
// not because this function knows their numbers.
func (c Campaign) TownBegins() (int, bool) {
	if len(c.Offered) == 0 {
		return 0, false
	}
	return c.Offered[0], true
}

// LoadCampaign reads the scenario registry out of the container filesystem.
//
// It stands beside LoadNPCDefs (table.go) and for its reason: the file lives
// in the campaign container, and a tool pointed at one archive must be able to
// read this without being handed a whole install.
func LoadCampaign(fsys *vfs.FS) (Campaign, error) {
	b, err := fsys.ReadFile(ScenarioRegistry)
	if err != nil {
		return Campaign{}, fmt.Errorf("read %s: %w", ScenarioRegistry, err)
	}
	r, err := reg.Parse(b)
	if err != nil {
		return Campaign{}, fmt.Errorf("%s: %w", ScenarioRegistry, err)
	}
	campaign := ReadCampaign(r)
	if raw, err := fsys.ReadFile(globalMapRegistry); err == nil {
		if parsed, err := reg.Parse(raw); err == nil {
			if data, err := ReadGlobalMap(parsed); err == nil {
				for mission, object := range data.Missions {
					if object >= 0 && object < len(data.Objects) {
						if campaign.MapObjects == nil {
							campaign.MapObjects = make(map[int]int)
						}
						campaign.MapObjects[mission] = object
					}
				}
			}
		}
	}
	return campaign, nil
}

// ReadCampaign is LoadCampaign over a registry ALREADY PARSED, and it is what
// LoadCampaign is defined in terms of — so the whole of what this package makes
// of a scenario registry is decidable against bytes a test builds, with no
// archive and no install anywhere near it.
//
// A nil registry, one holding no mission section, or one whose sections carry
// none of the three keys, each yields the zero Campaign rather than an error.
// None of those is a malformed file: it is a scenario this tree can say less
// about, and the front end's own answer to saying less is to name no next
// mission rather than to refuse to start.
func ReadCampaign(r *reg.Reg) Campaign {
	var c Campaign
	if r == nil || r.Root == nil {
		return c
	}
	seen := make(map[int]bool)
	for _, section := range r.Root.Children {
		if !section.Dir {
			continue
		}
		if strings.EqualFold(section.Name, generalSection) {
			c.Declared = int(sectionInt(section, totalMissionsKey))
			c.MercenaryCount = sectionIntSlice(section, mercenaryCountKey)
			continue
		}
		n, ok := sectionMission(section.Name)
		if !ok {
			continue
		}
		if sectionInt(section, "LastMission") != 0 {
			if c.Last == nil {
				c.Last = make(map[int]bool)
			}
			c.Last[n] = true
		}
		if n%mainMissionStride == 0 {
			c.Main = append(c.Main, n)
		} else {
			c.Side = append(c.Side, n)
		}
		if object := int(sectionInt(section, mapObjectKey)); object != 0 {
			if c.MapObjects == nil {
				c.MapObjects = make(map[int]int)
			}
			c.MapObjects[n] = object
		}
		if v, ok := sectionIntOK(section, autoGetMissionKey); ok && v != -1 {
			if c.Auto == nil {
				c.Auto = make(map[int]int)
			}
			c.Auto[n] = int(v)
		}
		// THE THREE KEYS ARE READ TOGETHER AND MERGED, because what is
		// wanted here is the SET a town hands out and not which building
		// hands out which — a distinction this tree has no building to act
		// on. REG-SCN-064's own census says no mission is named by two of
		// them anyway, so the merge loses nothing it could have kept.
		for _, key := range [...]string{innMissionKey, shopMissionKey, tcMissionKey} {
			for _, v := range sectionInts(section, key) {
				// ZERO IS THE SENTINEL AND NOT A MISSION: InnMission[i]
				// is zero for an NPC with nothing to give (REG-SCN-064),
				// and three sections of a stock scenario carry one.
				if v <= 0 || seen[int(v)] {
					continue
				}
				seen[int(v)] = true
				c.Offered = append(c.Offered, int(v))
			}
		}
		// THE CHAPTER IS READ FROM THE SAME SECTION IN THE SAME PASS and is not
		// derived from anything above it. The loop over the three keys just above
		// builds a SET and drops the order, the sentinel and the duplicates; this
		// keeps all three, per building, because that is what a town screen has to
		// show.
		if ch, ok := readChapter(section, n); ok {
			if c.Chapters == nil {
				c.Chapters = make(map[int]Chapter)
			}
			c.Chapters[n] = ch
		}
	}
	sort.Ints(c.Main)
	sort.Ints(c.Side)
	sort.Ints(c.Offered)
	return c
}

// readChapter is one section's town data, and whether the section carries
// any.
//
// A SECTION WITH NONE OF THE SIX KEYS YIELDS NOTHING, so Chapters holds the
// town chapters rather than every mission — 13 of the shipped 24 sections carry
// a building key at all. The test is on the KEYS and not on the mission number:
// a chapter is a section that says something about a building, which is a fact
// about the file, where "a multiple of ten" would be this tree deciding.
func readChapter(section *reg.Node, n int) (Chapter, bool) {
	ch := Chapter{
		Mission:         n,
		InnNPC:          sectionIntSlice(section, innNPCKey),
		Inn:             sectionIntSlice(section, innMissionKey),
		Shop:            sectionIntSlice(section, shopMissionKey),
		School:          sectionIntSlice(section, tcMissionKey),
		ShopMin:         int(sectionInt(section, shopMinKey)),
		ShopMax:         int(sectionInt(section, shopMaxKey)),
		Payment:         int(sectionInt(section, paymentKey)),
		AddHero:         sectionIntSlice(section, addHeroKey),
		Mercenaries:     sectionIntSlice(section, mercenariesKey),
		EnableMercenary: sectionIntSlice(section, enableMercenaryKey),

		TextDocuments:    sectionIntSlice(section, addTextDocumentKey),
		PictureDocuments: sectionIntSlice(section, addPictureDocumentKey),
	}
	empty := len(ch.InnNPC) == 0 && len(ch.Inn) == 0 && len(ch.Shop) == 0 &&
		len(ch.School) == 0 && len(ch.AddHero) == 0 && len(ch.Mercenaries) == 0 &&
		len(ch.EnableMercenary) == 0 && ch.ShopMin == 0 && ch.ShopMax == 0 && ch.Payment == 0 &&
		len(ch.TextDocuments) == 0 && len(ch.PictureDocuments) == 0
	return ch, !empty
}

// sectionIntSlice is sectionInts widened to int, keeping every element the file
// holds — the zero sentinel included, since Chapter's own doc says why.
func sectionIntSlice(section *reg.Node, key string) []int {
	src := sectionInts(section, key)
	if len(src) == 0 {
		return nil
	}
	out := make([]int, len(src))
	for i, v := range src {
		out[i] = int(v)
	}
	return out
}

// sectionMission is the mission number a section name yields, and whether it
// yields one at all: the literal "Mission" folded case-insensitively, then a
// positive decimal integer and nothing else.
//
// The fold is the registry's own — reg's lookup compares names that way — and
// the parse is strconv.Atoi, exactly as WithMissions parses a map's stem
// (maplist.go), so neither place invents a second idea of what a mission
// number looks like.
func sectionMission(name string) (int, bool) {
	if len(name) <= len(missionSection) {
		return 0, false
	}
	if !strings.EqualFold(name[:len(missionSection)], missionSection) {
		return 0, false
	}
	n, err := strconv.Atoi(name[len(missionSection):])
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// sectionInts is every integer a section's key holds, whichever of the two
// shapes it is written in.
//
// BOTH SHAPES ARE THE FILE'S OWN AND NEITHER IS THE ODD ONE OUT. A stock
// scenario writes InnMission as a bare integer in six sections and as an array
// in the other seven, so a reader that handled one shape would silently read
// half the campaign's offers as nothing at all — which is a hole that draws no
// error and shows up only as a boundary in the wrong place.
func sectionInts(section *reg.Node, key string) []int32 {
	n := childNamed(section, key)
	if n == nil {
		return nil
	}
	switch n.Type {
	case reg.TypeInt:
		return []int32{n.Int}
	case reg.TypeIntArray:
		return n.Ints
	}
	return nil
}

// sectionIntOK is a section's key as a single integer, and whether the key
// is present as one at all.
func sectionIntOK(section *reg.Node, key string) (int32, bool) {
	n := childNamed(section, key)
	if n == nil || n.Type != reg.TypeInt || n.Dir {
		return 0, false
	}
	return n.Int, true
}

// sectionInt is a section's key as a single integer, and zero where it is
// absent or is not one.
func sectionInt(section *reg.Node, key string) int32 {
	v, _ := sectionIntOK(section, key)
	return v
}

// childNamed is the value node a section holds under name, folded
// case-insensitively like every other name comparison against this format, and
// nil for a name the section does not hold or holds as a directory.
func childNamed(section *reg.Node, name string) *reg.Node {
	for _, ch := range section.Children {
		if ch.Dir {
			continue
		}
		if strings.EqualFold(ch.Name, name) {
			return ch
		}
	}
	return nil
}
