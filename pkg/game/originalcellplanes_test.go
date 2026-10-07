package game

import (
	"bytes"
	"errors"
	"io/fs"
	"testing"

	"againrom/internal/synth"
)

type cellPlaneSource1115 struct {
	raw     []byte
	err     error
	address string
}

func (s *cellPlaneSource1115) ReadFile(address string) ([]byte, error) {
	s.address = address
	return s.raw, s.err
}

func TestOriginalCellPlanes1115ParameterFailuresDoNotPartlyAdopt(t *testing.T) {
	for _, name := range []string{"missing", "read-error", "malformed", "uninitialized-pair", "missing-map"} {
		t.Run(name, func(t *testing.T) {
			f, _, _ := crossingCellOriginalDoor1115(t, false, false)
			ms := *f.live.mission.state
			source := &cellPlaneSource1115{raw: synth.Reg(0x11, nil)}
			wantError := true
			switch name {
			case "missing":
				source.err, wantError = fs.ErrNotExist, false
			case "read-error":
				source.err = errors.New("test read failure")
			case "malformed":
				source.raw = []byte("not a registry")
			case "uninitialized-pair":
				m := *ms.Map
				m.Tiles = append([]uint16(nil), m.Tiles...)
				m.Tiles[0] = 13 << 6
				ms.Map = &m
			case "missing-map":
				ms.Map = nil
			}
			before, err := ms.World.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			err = importOriginalCellPlanes(&ms, source)
			if (err != nil) != wantError || source.address != "world/data/map.reg" {
				t.Fatalf("read/admission: address=%q err=%v", source.address, err)
			}
			after, marshalErr := ms.World.MarshalBinary()
			planes, present := ms.World.SavedCellPlanes()
			if marshalErr != nil || !bytes.Equal(before, after) || present || planes != nil {
				t.Fatal("failed or unavailable parameters changed native state", marshalErr)
			}
		})
	}
}
