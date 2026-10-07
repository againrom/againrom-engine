package game

import "encoding/binary"

func savedSessionHead(current, retained [48]byte) [48]byte {
	if current != ([48]byte{}) {
		return current
	}
	if retained != ([48]byte{}) {
		return retained
	}
	binary.LittleEndian.PutUint64(current[16:24], 10000000)
	binary.LittleEndian.PutUint32(current[32:36], 10000)
	return current
}
