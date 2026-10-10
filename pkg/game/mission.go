package game

import (
	"fmt"
	"strconv"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// NPCRegistry is the ADDRESS the scenario NPC table is loaded from: the campaign
// container's identity segment and the entry inside it.
//
// It is composed from the prefix this package already derives for its map
// addresses, so a renamed container moves the maps and the registry together. A
// second spelling is how half a container's addresses come to name an identity
// nothing in the set answers.
var NPCRegistry = scenarioPrefix + "npc.reg"

// MissionMap is the address campaign mission n's map is read at, and whether n
// names a mission at all.
//
// The address is the campaign container's identity segment, the number in
// DECIMAL, and the map extension — the original composes the same name from the
// same number, and the leading identity segment is what dispatches the address to
// that container rather than to the loose tree.
//
// A number that is not positive names no mission and is refused HERE, before
// anything is opened: 0 is the number a save taken outside a mission carries and
// a negative one is a caller's arithmetic, and neither should reach an archive as
// a lookup that merely happens to miss.
//
// An upper bound is deliberately not applied. The campaign declares a mission
// count and the original rejects a request above ten times it; this asks the
// container instead, so a modded scenario archive answers for its own contents
// and an absent map fails at the read with its address in the message.
func MissionMap(n int) (string, bool) {
	if n <= 0 {
		return "", false
	}
	return scenarioPrefix + strconv.Itoa(n) + ".alm", true
}

// Mission is a started campaign mission: the map it was built from, the world
// that was built, and what the start decided.
//
// The DECODED MAP comes back beside the world because everything this story does
// not own needs it — the script, the mission's text, the tiles a viewer draws —
// and re-decoding to get at it would be both wasteful and a second chance to
// disagree about what the file says.
type Mission struct {
	// Number is the mission the world was built for.
	Number int

	// Address is where its map was read, so a report can name the file without
	// recomposing the address and getting it subtly different.
	Address string

	// Map is the decoded map, exactly as the archive holds it.
	Map *alm.Map

	// World is the built world: the map's own placements, then the party,
	// RUNNING the mission's compiled script. A map whose script will not decode
	// runs none — see RaiseErr.
	World *sim.World

	// Start is where the start put the party, and how it decided.
	Start mapload.Start

	// Party is the members this mission was started with, in the order the
	// start took them — so Party[i] is the member standing on Start.Cells[i]
	// and carrying entity Start.IDs[i].
	//
	// IT IS KEPT BECAUSE THE TWO HALVES OF A CHARACTER MEET NOWHERE ELSE. The
	// world keeps what the fold produced and drops what produced it — a hero's
	// statistics are a loader input and reach no entity field, no byte form and
	// no digest — so a reader that wants to say "this entity's numbers came from
	// these statistics" needs the member beside the id, and this is the last
	// place both exist.
	Party []mapload.PartyMember

	ActorManifest      *SnapshotActorManifest
	actorRegistry      *originalActorRegistry
	originalSaveReport *OriginalSaveResume
	savedDocument      *SnapshotSAVDocument
	pendingPickups     []sim.ItemStack

	// DeadArt resolves a virtual corpse's TypeID. Original LOAD and the admitted
	// document on native LOAD both reconstruct this presentation-only map.
	DeadArt map[sim.EntityID]uint16

	// Raises is every announcement this mission's script can raise: a latch and
	// an event number, in trigger order and then in action-slot order.
	//
	// IT IS THE COMPILE'S ANSWER. The builder is the one place holding both an
	// authored action id and the compiled instant it became, while build-time
	// actions become no instant and leave no runtime trigger slot.
	Raises []mapload.ScriptRaise

	// RaiseErr is why Raises is empty, for a mission whose script would not
	// decode. It is carried rather than returned for the reason FontErr is: the
	// mission still starts, and an install whose map holds a script this build
	// cannot read is a fact about that install that should not vanish.
	RaiseErr error
}

// StartMission reads campaign mission n out of the opened archives and builds its
// world with the party standing where the map's script says a mission begins.
//
// IT OPENS NOTHING. Both files it reads — the map and the definition table
// behind the table argument — come through the filesystem the caller already
// holds, so no path outside the given archives can be reached and no asset root
// is spelled here (golden rule 3).
//
// The table is an ARGUMENT and not something loaded here, for the reason the
// world builder takes one: what a placement resolves to is a property of the
// installed files, and a loader that reached for them itself would give one
// mission two worlds depending on when it was called. LoadTable is what builds
// one, once, at front-end construction.
//
// Every failure names the address it happened at. A number naming no entry, an
// entry that will not decode and a world the builder refuses are three different
// messages, because they are three different things wrong with an install.
func StartMission(fsys entrySource, n int, t *mapload.Table, diff mapload.Difficulty,
	party []mapload.PartyMember) (*Mission, error) {
	addr, ok := MissionMap(n)
	if !ok {
		return nil, fmt.Errorf("mission %d: not a campaign mission number", n)
	}
	if fsys == nil {
		return nil, fmt.Errorf("read %s: no archive", addr)
	}
	b, err := fsys.ReadFile(addr)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", addr, err)
	}
	open := alm.Open
	if tableGame(t).Edition().SecondMaps {
		open = alm.OpenROM2
	}
	m, err := open(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", addr, err)
	}
	mapload.WithdrawBorderPlacements(m)
	return StartMissionFrom(m, addr, n, t, diff, party)
}

// StartMissionFrom is StartMission over a map ALREADY DECODED, and it is what
// StartMission is defined in terms of.
//
// It exists for the front end's own door. Opening a mission needs both a
// world and a viewer, and the viewer's load path decodes the map on the way
// to building the terrain — so a door calling StartMission would decode
// the same bytes a second time, and "the world and the terrain on screen
// describe the same map" would become a consistency check somebody has to
// keep rather than a property of there being one decoded map.
//
// addr is where m came from and is used only to name a failure, so a caller that
// decoded the bytes itself still reports the entry rather than a bare error.
func StartMissionFrom(m *alm.Map, addr string, n int, t *mapload.Table, diff mapload.Difficulty,
	party []mapload.PartyMember) (*Mission, error) {
	return startMissionFromWith(m, addr, n, t, diff, party, nil)
}

// startMissionFromWith places the party with draws; nil is the placement
// sequence at its fixed seed.
func startMissionFromWith(m *alm.Map, addr string, n int, t *mapload.Table, diff mapload.Difficulty,
	party []mapload.PartyMember, draws *sim.Draws) (*Mission, error) {
	for _, p := range party {
		if err := mapload.ValidatePartyLoad(p); err != nil {
			return nil, err
		}
		h, ok := p.OriginalHumanState()
		if p.OriginalHuman != nil && !ok {
			// Native fallback cannot reconstruct either observed source pair.
			if err := p.OriginalHuman.State.FighterProjectionError(); err != nil {
				return nil, fmt.Errorf("%s: %s: %w", addr, p.ID, err)
			}
		}
		if ok {
			if err := h.ProjectionError(); err != nil {
				return nil, fmt.Errorf("%s: %s: %w", addr, p.ID, err)
			}
		}
	}
	party = mapload.NormalizePartyShieldLoadouts(party, t)
	// THE COMPILE COMES FIRST, because the world is built to RUN its program:
	// the mission's world is the scripted one, so its triggers are evaluated as
	// it is stepped, its latches are set, and its outcome follows from them.
	// Until T7 the program was compiled here and thrown away, and a mission
	// therefore ran no script at all — no trigger fired, no latch was ever
	// set, and no outcome was ever reached on a real install.
	//
	// The unit table is the map's own; a reference the binder cannot resolve is
	// reported by the compile and is not this call's business, exactly as it is
	// not almtool's. The report is carried for its RAISE LIST, which is the
	// compile's own answer and not the compiled program's — see
	// Mission.Raises.
	//
	// A SCRIPT THAT WILL NOT DECODE IS NOT FATAL, and its error is CARRIED
	// rather than returned. That is the mission-start path's own behaviour, not
	// a softening of it: DropCells already answers a body it cannot tile with no
	// cells and no complaint, so a start over such a map already succeeds and
	// refusing here would make one story's tolerance another's error. It is kept
	// rather than discarded, because a mission whose script will not read is a
	// fact about that install and dropping it is how such a thing goes unnoticed
	// — the same reason the front end keeps FontErr. Such a mission starts with
	// a nil program, which is the world it has always been given.
	//
	// The hero band is resolved here from the party, before the world is built,
	// because the world is built to run the compiled program. A party of none
	// binds nobody.
	edition := tableGame(t).Edition()
	compile := mapload.CompileScript
	if edition.SecondScripts {
		compile = mapload.CompileROM2Script
	}
	s, rep, raiseErr := compile(m, campaignScriptRefs(m, t, party))
	// Mission 40 is owner-authored to protect the persistent companion added by
	// scenario NPC 22. The shipped map has no VIP node for her, so this cannot be
	// recovered from the ALM script; the stable party identity is the campaign
	// reference that survives both an ordinary carry and an original-save load.
	// Compile the rule into the simulation script rather than deciding loss in
	// the client. That gives save-loaded and fresh worlds the same hashed rule,
	// and leaves every other party member outside the objective.
	if companion := edition.CompanionObjectiveMission; companion != 0 && n == companion {
		for i := range party {
			if party[i].CompanionNPC != 22 && party[i].ID != "npc:22" {
				continue
			}
			critical := mapload.PartyEntity(m, i)
			var err error
			s, err = s.WithVIP(critical)
			if err != nil {
				return nil, fmt.Errorf("%s: add mission-40 companion objective: %w", addr, err)
			}
			break
		}
	}
	w, st, err := mapload.StartCampaignMissionScriptedWith(m, t, diff, party, n, s, draws)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", addr, err)
	}
	return &Mission{Number: n, Address: addr, Map: m, World: w, Start: st, Party: mapload.OwnParty(party),
		Raises: rep.Raises, RaiseErr: raiseErr}, nil
}

// campaignScriptRefs resolves a map script's hero-band references against the party.
func campaignScriptRefs(m *alm.Map, table *mapload.Table, party []mapload.PartyMember) mapload.ScriptRefs {
	refs := campaignScriptPartyRefs(m, table, party, func(i int) sim.EntityID { return mapload.PartyEntity(m, i) })
	refs.Units, refs.Structures = mapload.ScriptUnits(m, party), mapload.ScriptStructures(m)
	return refs
}

// maxHeroOrdinal: ordinal k reads section 21+k of a 256-section table (TRIG-HEROFAIL-078).
const maxHeroOrdinal = 234

// heroTraits is what a hero ordinal reads of a member; a hired unit is never a woman.
func heroTraits(p mapload.PartyMember) data.HeroTraits {
	return data.HeroTraits{
		Female: !p.Hired() && data.FigureDir(p.FigureDir).Female(),
		Mage:   p.Mage,
		Face:   uint8(p.FigureFace) & 0x3f,
	}
}

// campaignScriptPartyRefs resolves hero ordinal k (value 10001+k) to the first party
// member in list order that passes its role test (TRIG-HEROORD-075, TRIG-HEROTPL-076).
// Members count as named; a map with player capacity above one resolves nothing.
func campaignScriptPartyRefs(m *alm.Map, table *mapload.Table, party []mapload.PartyMember, entity func(int) sim.EntityID) mapload.ScriptRefs {
	var refs mapload.ScriptRefs
	if len(party) == 0 || m != nil && m.Meta.Word70 > 1 {
		return refs
	}
	primary := 0
	for i, p := range party {
		if p.StartingHero {
			primary = i
			break
		}
	}
	var npc *data.NPCDefs
	if table != nil {
		npc = table.NPC
	}
	roster := make([]data.HeroTraits, len(party))
	for i, p := range party {
		roster[i] = heroTraits(p)
	}
	for k := 0; k <= maxHeroOrdinal; k++ {
		i, ok := npc.ResolveHero(k, roster[primary], roster)
		if !ok {
			continue
		}
		switch k {
		case 0:
			refs.Hero, refs.HasHero = entity(i), true
		case 1:
			refs.Companion, refs.HasCompanion = entity(i), true
		default:
			if refs.Roles == nil {
				refs.Roles = make(map[uint32]sim.EntityID)
			}
			refs.Roles[uint32(10001+k)] = entity(i)
		}
	}
	return refs
}

// entrySource is the one thing StartMission needs of a filesystem: the bytes
// behind an address. It is an interface so the entry point is drivable from a
// stub holding two entries, which is what lets the address derivation and every
// failure message be tested with no archive on disk.
type entrySource interface {
	ReadFile(name string) ([]byte, error)
}
