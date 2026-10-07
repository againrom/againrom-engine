package game

import (
	"encoding/binary"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// nativeCityPlayerState constructs the one fresh human-owned campaign Player.
// Source-backed city/world Players retain their own fields instead. Registration
// is separate from construction: SAV-664 supplies the mask for the authored
// slot/participant pair; AI-FORM-037 supplies the complete new settings block.
func nativeCityPlayerState(heroIdentity uint32) (fixed, settings []byte) {
	fixed = make([]byte, 51)
	const slot = uint16(1)
	binary.LittleEndian.PutUint16(fixed[0:2], slot)
	// The campaign's own roster ordinal is also one; it is not an archive ID.
	binary.LittleEndian.PutUint32(fixed[2:6], 1)
	// Participant +28 is zero for this human-owned Player. Registration writes
	// +2c = 1 << (slot mod16), rather than retaining the unregistered zero.
	binary.LittleEndian.PutUint16(fixed[19:21], uint16(1)<<uint(slot%16))
	// PAL-SHADE-013: +44 stays at its fresh constructor zero. The ALM-copy
	// writer later supplies a map colour; registration does not write it.
	// SAV-665/SAV-666: +50 starts at zero and the mana reserve starts at95.
	binary.LittleEndian.PutUint32(fixed[39:43], nativeCityManaReserve)
	binary.LittleEndian.PutUint32(fixed[43:47], heroIdentity)
	binary.LittleEndian.PutUint32(fixed[47:51], nativeCityPlayerIdentity)
	settings = make([]byte, 32)
	settings[31] = 2 // AI-FORM-037: conditional formation; other31 bytes zero.
	return fixed, settings
}

// The current canonical Player byte and the application setting index are
// distinct. A queued UI command may have updated only the latter. With no
// persisted setting or World, a fresh native Player starts in Auto.
func nativeCityPlayerFormation(data *sav.CityData, app *SnapshotApplicationState, world *sim.World) error {
	mode, setting := uint8(2), int32(1)
	if app != nil {
		if err := validateApplicationState(app); err != nil {
			return originalCityUnsupportedf("native Player formation: %v", err)
		}
		setting = app.Original.Formation
		// Same authored-index mapping as sim.remapFormationParameter and the
		// existing command seam: AI-FORM-037, including its default arm.
		switch setting {
		case 0:
			mode = 0
		case 2:
			mode = 1
		default:
			mode = 2
		}
	}
	if world != nil {
		var found bool
		mode, found = world.CommandFormationMode(sim.SelfSlot)
		if !found {
			return originalCityUnsupportedf("native Player formation has no current command receiver")
		}
		if app == nil {
			switch mode {
			case 0:
				setting = 0
			case 2:
				setting = 1
			default:
				setting = 2
			}
		}
	}
	if data == nil || len(data.Players) != 1 || data.Players[0] == 0 || int(data.Players[0]) > len(data.Objects) {
		return originalCityUnsupportedf("native Player formation lacks its single owned receiver")
	}
	player := data.Objects[data.Players[0]-1].Player
	if player == nil || len(player.Raw32) != 32 {
		return originalCityUnsupportedf("native Player formation has no complete settings block")
	}
	for i := range data.State.ValueRecords {
		r := &data.State.ValueRecords[i]
		if r.Path == "/GameOptions/Formation" && r.Value.Kind == 2 {
			player.Raw32[31] = mode
			r.Value.Int32 = setting
			return nil
		}
	}
	return originalCityUnsupportedf("native Player formation has no application setting leaf")
}
