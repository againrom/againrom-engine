package sav

import "fmt"

// ActorPools is the narrow pool projection of one Unit-derived object reached
// through the complete Player list. It is not the corpse list or the head scan.
// Words retain their wire bits; a consumer must interpret health as signed when
// deciding whether the actor is living (amended SAV-DEATH-051).
type ActorPools struct {
	Off                      int
	MapUnitID                uint16
	Cell                     uint16
	Stage                    uint8
	HP, MaxHP, Mana, MaxMana uint16
}

// ActorPools walks every Player and every group actor reference, in archive
// order (SAV-ROSTER-024, SAV-DOC-053). Null slots add no actor and repeated
// references name the same object, not another copy. The four statistic words
// are SAV-UNITFLD-049's named fields; no opaque runtime block is interpreted.
func (f *File) ActorPools(current ...uint16) ([]ActorPools, error) {
	actors, err := f.playerActors(current)
	if err != nil {
		return nil, err
	}
	var out []ActorPools
	for _, actor := range actors {
		p, err := actorPool(actor)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// playerRecords advances the shared archive counter through the exact bounded
// list. GroundSacks uses the same traversal before reaching the later lists.
func (w *walker) playerRecords(n uint32) ([]*Record, error) {
	if w.p < 0 || w.p > len(w.b) || n > maxListElements || uint64(n) > uint64((len(w.b)-w.p)/2) {
		return nil, fmt.Errorf("sav: Players count %d exceeds bounded remaining data", n)
	}
	out := make([]*Record, 0, int(n))
	seen := make(map[*Record]bool)
	for i := uint32(0); i < n; i++ {
		player, err := w.object(0)
		if err != nil {
			return nil, fmt.Errorf("sav: Player %d: %w", i, err)
		}
		if player == nil {
			continue
		}
		if player.Class != "Player" {
			return nil, fmt.Errorf("sav: Player %d is %s, not a Player", i, player.Class)
		}
		if !seen[player] {
			out = append(out, player)
			seen[player] = true
		}
	}
	return out, nil
}
