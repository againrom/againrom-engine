package game

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func beforeAttackNotices(t *testing.T, world []byte) []byte {
	t.Helper()
	b := bytes.Clone(world)
	b = beforeSpellDelivery1183(t, b)
	if len(b) > 0 && b[0] == 94 {
		b = b[:len(b)-8]
		b[0] = 93
	}
	if len(b) > 0 && b[0] == 93 {
		n := uint64(binary.LittleEndian.Uint32(b[len(b)-4:]))
		if n%7 != 0 || n > uint64(len(b)-38) {
			t.Fatal("invalid attack-notice fixture suffix")
		}
		b = b[:len(b)-4-int(n)]
		b[0] = 92
	}
	return b
}
