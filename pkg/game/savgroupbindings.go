package game

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Nil is the explicit pre-binding native format, not permission to reconstruct
// current identities from a retained SAV. Unavailable is an export coverage gap;
// the complete current native World remains saveable independently of it.
type SnapshotSAVGroupBindings struct {
	Version     uint32
	Groups      []SnapshotSAVGroupBinding
	Members     []SnapshotSAVGroupMemberBinding
	Unavailable string
	// Absent in all predecessors. A slot is a semantic lookup value, never
	// the identity of an enclosing Player or a reason to merge alias objects.
	PlayersPresent     bool
	FormationsPresent  bool
	PlayersConstructed bool
	Players            []SnapshotSAVGroupPlayerBinding
}

type SnapshotSAVGroupPlayerBinding struct {
	ID          uint32
	ObjectIndex uint16
	Constructed bool `json:",omitempty"`
}

func nativeSavedPlayerBindings(bindings *SnapshotSAVGroupBindings) []SnapshotSAVGroupPlayerBinding {
	var players []SnapshotSAVGroupPlayerBinding
	for _, player := range bindings.Players {
		if !player.Constructed {
			players = append(players, player)
		}
	}
	return players
}

// Groups are inline Player records, not archive objects. ID is the native
// registry identity; PlayerObject and InlineIndex identify the exact container.
type SnapshotSAVGroupBinding struct {
	ID               uint32
	PlayerObject     uint16
	InlineIndex      uint32
	Reference, Owner SnapshotSAVGroupReferenceBinding
	ContainerID      uint32
	Authored         bool
	RootOnly         bool // ordinary reachability for an actor without a native Group
}

// ObjectIndex records the resolved object, never a cast of Key or owner slot.
// The source key/class/owner tuple detects changed native reference authority.
type SnapshotSAVGroupReferenceBinding struct {
	ObjectIndex uint16
	Key         uint32
	Class       uint8
	Owner       uint32
}

// Bound entries use EntityID (including zero). UnresolvedHandle only joins an
// already persisted, unmaterialized SavedGroupMember to its detached object.
// It is NOT an output archive tag or object identity. Output always uses the
// independently assigned ObjectIndex; transient import-origin rows are dropped.
type SnapshotSAVGroupMemberBinding struct {
	EntityID         sim.EntityID
	UnresolvedHandle uint16
	Bound            bool
	ObjectIndex      uint16
}

func savedGroupMemberCompare(a, b SnapshotSAVGroupMemberBinding) int {
	if a.Bound != b.Bound {
		if a.Bound {
			return 1
		}
		return -1
	}
	if a.Bound {
		return cmp.Compare(a.EntityID, b.EntityID)
	}
	return cmp.Compare(a.UnresolvedHandle, b.UnresolvedHandle)
}

func cloneSavedGroupBindings(src *SnapshotSAVGroupBindings, doc *sav.DocumentData, actors []SnapshotSAVActor) (*SnapshotSAVGroupBindings, error) {
	if src == nil {
		return nil, nil
	}
	if src.FormationsPresent && !src.PlayersPresent {
		return nil, fmt.Errorf("saved SAV formations lack exact Player bindings")
	}
	if src.PlayersConstructed && (!src.PlayersPresent || src.FormationsPresent) {
		return nil, fmt.Errorf("constructed Player roots conflict with native carrier presence")
	}
	if doc == nil || src.Version != 1 || len(src.Unavailable) > 4096 || strings.ContainsRune(src.Unavailable, '\x00') || len(src.Groups) > savedGroupFieldListLimit || len(src.Members) > len(doc.Objects) {
		return nil, fmt.Errorf("saved SAV Group bindings have invalid version or bounds")
	}
	roots := make(map[uint16]bool)
	groupCount := 0
	for _, index := range doc.Players {
		if index == 0 || roots[index] {
			continue
		}
		if int(index) > len(doc.Objects) || doc.Objects[index-1].Class != "Player" {
			return nil, fmt.Errorf("saved SAV Group binding has invalid Player root")
		}
		roots[index] = true
		groupCount += len(doc.Objects[index-1].Groups)
	}
	if len(src.Groups) != groupCount {
		return nil, fmt.Errorf("saved SAV Group bindings do not cover inline Groups")
	}
	players, err := savedGroupPlayerObjects(src, doc, roots)
	if err != nil {
		return nil, err
	}
	type site struct {
		player uint16
		inline uint32
	}
	seenSites := make(map[site]bool, len(src.Groups))
	for i, g := range src.Groups {
		location := site{g.PlayerObject, g.InlineIndex}
		if g.ID == 0 || i > 0 && src.Groups[i-1].ID >= g.ID || !roots[g.PlayerObject] || uint64(g.InlineIndex) >= uint64(len(doc.Objects[g.PlayerObject-1].Groups)) || seenSites[location] {
			return nil, fmt.Errorf("saved SAV Group bindings have ambiguous identity or container")
		}
		seenSites[location] = true
		if src.PlayersPresent && !src.PlayersConstructed {
			if g.ContainerID == 0 || players[g.ContainerID] != g.PlayerObject {
				return nil, fmt.Errorf("saved SAV Group binding names a different Player container")
			}
		} else if !src.PlayersConstructed && (g.ContainerID != 0 || g.Authored) {
			return nil, fmt.Errorf("saved SAV Group binding has container state without Players")
		}
		// An export gap may retain an older valid document, never malformed
		// construction authority. Generated references are null; their typed
		// owner must name this exact Player, not another equal-slot object.
		if (g.Authored || g.RootOnly) && (g.Reference != (SnapshotSAVGroupReferenceBinding{}) ||
			g.Owner.Class != 1 || g.Owner.ObjectIndex != g.PlayerObject) {
			return nil, fmt.Errorf("saved SAV authored Group has invalid reference or Player authority")
		}
		if g.RootOnly {
			if err := validateActorRootGroup(doc, g); err != nil {
				return nil, err
			}
		}
		for _, ref := range []SnapshotSAVGroupReferenceBinding{g.Reference, g.Owner} {
			if ref.Class > 2 || int(ref.ObjectIndex) > len(doc.Objects) || (ref.Class == 0) != (ref.ObjectIndex == 0) || ref.Class != 1 && ref.Owner != 0 {
				return nil, fmt.Errorf("saved SAV Group binding has invalid resolved reference")
			}
			if ref.ObjectIndex != 0 && ((doc.Objects[ref.ObjectIndex-1].Class == "Player") != (ref.Class == 1) || ref.Key == 0) {
				return nil, fmt.Errorf("saved SAV Group binding reference class/key mismatch")
			}
			if ref.ObjectIndex != 0 {
				target := &doc.Objects[ref.ObjectIndex-1]
				field := "Identity"
				if ref.Class == 1 {
					field = "This"
				}
				key, err := savedStructureValue(target, field)
				if err != nil || key != ref.Key {
					return nil, fmt.Errorf("saved SAV Group reference binding names a different source key")
				}
				if ref.Class == 1 {
					slot, err := savedStructureValue(target, "Slot")
					// ActorGraph's resolved PlayerSlot is a WORD; retain the
					// full source scalar in the document independently.
					if err != nil || uint32(uint16(slot)) != ref.Owner {
						return nil, fmt.Errorf("saved SAV Group reference binding names a different Player slot")
					}
				}
			}
		}
	}
	actorObjects := make(map[sim.EntityID]uint16, len(actors))
	for _, a := range actors {
		actorObjects[a.EntityID] = a.ObjectIndex
	}
	seenObjects := make(map[uint16]bool, len(src.Members))
	for i, m := range src.Members {
		if m.ObjectIndex == 0 || int(m.ObjectIndex) > len(doc.Objects) || seenObjects[m.ObjectIndex] || i > 0 && savedGroupMemberCompare(src.Members[i-1], m) >= 0 || m.Bound && m.UnresolvedHandle != 0 || !m.Bound && (m.UnresolvedHandle == 0 || m.EntityID != 0) {
			return nil, fmt.Errorf("saved SAV Group bindings have ambiguous member")
		}
		seenObjects[m.ObjectIndex] = true
		switch doc.Objects[m.ObjectIndex-1].Class {
		case "Unit", "Human", "Humanoid":
		default:
			return nil, fmt.Errorf("saved SAV Group member binds a non-actor object")
		}
		if m.Bound && actorObjects[m.EntityID] != 0 && actorObjects[m.EntityID] != m.ObjectIndex {
			return nil, fmt.Errorf("saved SAV Group member disagrees with actor binding")
		}
	}
	out := *src
	out.Groups, out.Members = slices.Clone(src.Groups), slices.Clone(src.Members)
	out.Players = slices.Clone(src.Players)
	return &out, nil
}

func savedGroupPlayerObjects(src *SnapshotSAVGroupBindings, doc *sav.DocumentData, roots map[uint16]bool) (map[uint32]uint16, error) {
	players := make(map[uint32]uint16)
	if !src.PlayersPresent {
		if len(src.Players) != 0 {
			return nil, fmt.Errorf("saved SAV absent Player registry has bindings")
		}
		return players, nil
	}
	if len(src.Players) != len(roots) {
		return nil, fmt.Errorf("saved SAV Player bindings do not cover distinct roots")
	}
	seen := make(map[uint16]bool, len(src.Players))
	var orderedRoots []uint16
	for _, index := range doc.Players {
		if index != 0 && !seen[index] {
			orderedRoots = append(orderedRoots, index)
			seen[index] = true
		}
	}
	clear(seen)
	for i, player := range src.Players {
		if player.Constructed && src.PlayersConstructed {
			return nil, fmt.Errorf("saved SAV constructed Player has conflicting carrier presence")
		}
		if player.ID == 0 || i > 0 && src.Players[i-1].ID >= player.ID || !roots[player.ObjectIndex] || seen[player.ObjectIndex] || player.ObjectIndex != orderedRoots[i] {
			return nil, fmt.Errorf("saved SAV Player binding has ambiguous identity or object")
		}
		seen[player.ObjectIndex], players[player.ID] = true, player.ObjectIndex
	}
	return players, nil
}

// Validate the persisted native/document pairing without deriving identities
// from either side. Changed native constructors may be explicitly uncovered;
// an allegedly covered snapshot must still carry its exact source registry.
func validateSavedGroupBindingWorld(state *SnapshotSAVDocument, world *sim.World) error {
	if state == nil || state.GroupBindings == nil {
		return nil
	}
	groups, _, present := world.SavedGroups()
	if !present {
		if state.GroupBindings.PlayersConstructed || currentNativePolicy(state, func(p *sim.CurrentWorldPolicy) bool { return !p.GroupCarrier }) {
			return nil
		}
		return fmt.Errorf("saved SAV Group bindings lack their native registry")
	}
	players, havePlayers := world.SavedGroupPlayers()
	if havePlayers != (state.GroupBindings.PlayersPresent && !state.GroupBindings.PlayersConstructed) {
		return fmt.Errorf("saved SAV Player binding presence differs from native registry")
	}
	if havePlayers {
		native := nativeSavedPlayerBindings(state.GroupBindings)
		if len(players) != len(native) {
			return fmt.Errorf("saved SAV Player binding count differs from native registry")
		}
		for i, player := range players {
			binding := native[i]
			slot, err := savedStructureValue(&state.Document.Objects[binding.ObjectIndex-1], "Slot")
			if err != nil || player.ID != binding.ID || player.Slot != uint32(uint16(slot)) {
				return fmt.Errorf("saved SAV Player binding differs from native registry")
			}
		}
	}
	if state.GroupBindings.Unavailable != "" {
		return nil
	}
	count := 0
	for _, b := range state.GroupBindings.Groups {
		if !b.RootOnly {
			count++
		}
	}
	if len(groups) != count {
		return fmt.Errorf("saved SAV Group bindings do not cover the native registry")
	}
	byID := make(map[uint32]sim.SavedGroup, len(groups))
	for _, g := range groups {
		byID[g.ID] = g
	}
	for _, binding := range state.GroupBindings.Groups {
		if binding.RootOnly {
			continue
		}
		g, exists := byID[binding.ID]
		if !exists || g.Authored != binding.Authored || g.ContainerID != binding.ContainerID {
			return fmt.Errorf("saved SAV Group %d lacks its source native registry entry", binding.ID)
		}
		if state.GroupBindings.FormationsPresent && g.OwnerID != savedFormationOwnerID(state.GroupBindings, binding.Owner) {
			return fmt.Errorf("saved SAV Group %d formation owner differs from exact binding", binding.ID)
		}
		if binding.Authored {
			if err := validateAuthoredSavedGroupAuthority(state, binding, g); err != nil {
				return err
			}
			continue
		}
		for _, pair := range []struct {
			bound   SnapshotSAVGroupReferenceBinding
			current sim.SavedGroupReference
		}{{binding.Reference, g.Reference}, {binding.Owner, g.Owner}} {
			if pair.bound.Key != pair.current.Key || pair.bound.Class != pair.current.Class || pair.bound.Owner != pair.current.Owner {
				return fmt.Errorf("saved SAV Group %d reference differs from its native registry", binding.ID)
			}
		}
	}
	return nil
}

func importSavedGroupBindings(doc *sav.DocumentData, world *sim.World, byArchive map[uint16]uint16) (*SnapshotSAVGroupBindings, error) {
	groups, _, present := world.SavedGroups()
	if !present {
		return nil, nil
	}
	out := &SnapshotSAVGroupBindings{Version: 1, PlayersPresent: true}
	var players []sim.SavedGroupPlayer
	var containers []sim.SavedGroupContainer
	byID := make(map[uint32]sim.SavedGroup, len(groups))
	for _, g := range groups {
		byID[g.ID] = g
	}
	seenPlayers := make(map[uint16]bool)
	ordinal := uint32(0)
	bindRef := func(ref sim.SavedGroupReference) (SnapshotSAVGroupReferenceBinding, error) {
		bound := SnapshotSAVGroupReferenceBinding{Key: ref.Key, Class: ref.Class, Owner: ref.Owner}
		if ref.Class != 0 {
			bound.ObjectIndex = byArchive[ref.Archive]
			if bound.ObjectIndex == 0 {
				return bound, fmt.Errorf("saved SAV Group lacks exact reference origin")
			}
		}
		return bound, nil
	}
	for _, player := range doc.Players {
		if player == 0 || seenPlayers[player] {
			continue
		}
		seenPlayers[player] = true
		// Allocate in distinct-root encounter order. These fresh native IDs
		// do not copy an archive tag, address, slot or document object index.
		playerID := uint32(len(players) + 1)
		slot, err := savedStructureValue(&doc.Objects[player-1], "Slot")
		if err != nil {
			return nil, err
		}
		players = append(players, sim.SavedGroupPlayer{ID: playerID, Slot: uint32(uint16(slot))})
		out.Players = append(out.Players, SnapshotSAVGroupPlayerBinding{ID: playerID, ObjectIndex: player})
		for inline := range doc.Objects[player-1].Groups {
			// ActorGraph assigns ordinals over distinct Player roots in this
			// exact order. Selector, owner slot and scalar address do not join.
			ordinal++
			g, exists := byID[ordinal]
			if !exists || g.Authored {
				return nil, fmt.Errorf("saved SAV Group lacks exact import ordinal %d", ordinal)
			}
			binding := SnapshotSAVGroupBinding{ID: g.ID, PlayerObject: player, InlineIndex: uint32(inline), ContainerID: playerID}
			containers = append(containers, sim.SavedGroupContainer{GroupID: g.ID, PlayerID: playerID})
			var err error
			if binding.Reference, err = bindRef(g.Reference); err != nil {
				return nil, err
			}
			if binding.Owner, err = bindRef(g.Owner); err != nil {
				return nil, err
			}
			out.Groups = append(out.Groups, binding)
			for _, m := range g.Members {
				if !m.Bound && m.Archive == 0 {
					continue
				}
				index := byArchive[m.Archive]
				if index == 0 {
					return nil, fmt.Errorf("saved SAV Group lacks exact member origin")
				}
				member := SnapshotSAVGroupMemberBinding{EntityID: m.Entity, Bound: m.Bound, ObjectIndex: index}
				if !m.Bound {
					member.UnresolvedHandle = m.Archive
				}
				out.Members = append(out.Members, member)
			}
		}
	}
	if int(ordinal) != len(groups) {
		return nil, fmt.Errorf("saved SAV document lacks imported Group containers")
	}
	slices.SortFunc(out.Members, savedGroupMemberCompare)
	checked, err := cloneSavedGroupBindings(out, doc, nil)
	if err != nil {
		return nil, err
	}
	if err := world.ImportSavedGroupPlayers(players, containers); err != nil {
		return nil, err
	}
	world.RebuildLoadedActorTraversal()
	if err := importSavedFormations(doc, world, checked); err != nil {
		return nil, err
	}
	return checked, nil
}
