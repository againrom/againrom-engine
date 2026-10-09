package pal

import "fmt"

// OwnerTableCount is the number of consecutive colour tables in the shared
// human-owner palette. Each table has TableSize's ordinary [B, G, R, reserved]
// layout; unlike Decode's BMP shape, the first one begins at byte zero.
const OwnerTableCount = 16

// OwnerSize is the complete byte size of the shared human-owner palette.
const OwnerSize = OwnerTableCount * TableSize

// DecodeOwnerTables decodes the shared human-owner palette's sixteen
// consecutive tables.
//
// The stream is exactly OwnerSize bytes. There is no header, offset or trailer
// to ignore: accepting a prefix or suffix would make a second undocumented
// shape look like this one. Every fourth byte is reserved and dropped, just as
// Decode drops it in the ordinary BMP-backed table.
func DecodeOwnerTables(data []byte) ([OwnerTableCount]Table, error) {
	if len(data) != OwnerSize {
		return [OwnerTableCount]Table{}, fmt.Errorf("pal: owner stream is %d byte(s), want exactly %d", len(data), OwnerSize)
	}

	var out [OwnerTableCount]Table
	for table := range out {
		copy(out[table][:], Entries(data[table*TableSize:(table+1)*TableSize]))
	}
	return out, nil
}
