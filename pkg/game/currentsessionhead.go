package game

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// An absent native head uses the ordinary constructor or retained bytes on the
// wire. Only that absence is policy; a changed ordinary head becomes current.
func captureAbsentSessionHead(doc *sav.DocumentData, world *sim.World) []byte {
	if doc == nil || doc.World == nil || world == nil || world.RawSessionHead() != ([48]byte{}) {
		return nil
	}
	digest := sha256.Sum256(doc.World.Session.Raw08[:])
	return digest[:]
}

func validateAbsentSessionHead(anchor []byte, doc *sav.DocumentData) error {
	if len(anchor) != 0 && (len(anchor) != sha256.Size || doc == nil || doc.World == nil) {
		return fmt.Errorf("invalid absent session-head policy")
	}
	return nil
}

func restoreAbsentSessionHead(world *sim.World, doc *sav.DocumentData, anchor []byte) error {
	if err := validateAbsentSessionHead(anchor, doc); err != nil {
		return err
	}
	if len(anchor) == 0 {
		return nil
	}
	if world == nil {
		return fmt.Errorf("absent session-head policy lacks its world")
	}
	digest := sha256.Sum256(doc.World.Session.Raw08[:])
	if bytes.Equal(digest[:], anchor) {
		world.SetRawSessionHead([48]byte{})
	}
	return nil
}
