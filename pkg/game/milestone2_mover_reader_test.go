package game

import (
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
)

// SAV-TOKENPOS-074, SAV-UNITPROG-156 and SAV-WLIST-040 own every width
// below. Only decompression, structural starts and the identity-only
// archive/DTO permutation come from production. All three lists are walked
// from their own raw count, including escaped counts, without Record.Raw,
// Document values, ActorGraph or an importer-selected population.
type mover1160Record struct {
	archive  uint16
	class    string
	off      int
	position [12]byte
	mover    [180]byte
	order    [148]byte
	routes   [3][]uint16 // U15C static, U178 dynamic, U158_90 order path
	stage    byte
	hp       int16
}

type mover1160Source struct {
	records []mover1160Record
	origins map[uint16]uint16
}

var mover1160Lists = [...]string{"U15C", "U178", "U158_90"}

func mover1160Expected(f *sav.File, source []byte) (mover1160Source, error) {
	locs, err := f.DocumentActorLocations()
	if err != nil {
		return mover1160Source{}, err
	}
	objects, err := f.DocumentObjectLocations()
	if err != nil {
		return mover1160Source{}, err
	}
	_, origins, err := sav.DecodeDocumentDataWithOrigins(source)
	if err != nil {
		return mover1160Source{}, err
	}
	return mover1160Read(f.Body, locs, objects, origins)
}

func mover1160Read(body []byte, locs []sav.DocumentActorLocation, objects []sav.DocumentObjectLocation, origins []sav.DocumentObjectOrigin) (mover1160Source, error) {
	out := mover1160Source{origins: map[uint16]uint16{}}
	// Reuse 1158's raw tagged-actor/framing and bijective origin proof. Its
	// diagnostic stage/HP are also read from Body, never from imported actors.
	// Its decoded combat blocks do not supply any movement expectation.
	population, err := unit1158Read(body, locs, objects, origins)
	if err != nil {
		return out, err
	}
	out.origins = population.origins
	byArchive := map[uint16]unitCombatRecord{}
	for _, r := range population.records {
		byArchive[r.archive] = r
	}
	for _, loc := range locs {
		p := byArchive[loc.ArchiveIndex]
		r := mover1160Record{archive: p.archive, class: p.class, off: p.off, stage: p.stage, hp: p.hp}
		if loc.RoutesOff < loc.Off+41 || loc.RoutesOff > loc.RawBlocksOff || loc.RawBlocksOff > len(body)-462 {
			return out, fmt.Errorf("archive %d invalid route/block starts", r.archive)
		}
		at := loc.RoutesOff
		for i := range 2 {
			r.routes[i], at, err = mover1160Words(body, at, loc.RawBlocksOff)
			if err != nil {
				return out, fmt.Errorf("archive %d %s: %w", r.archive, mover1160Lists[i], err)
			}
		}
		if at != loc.RawBlocksOff {
			return out, fmt.Errorf("archive %d embedded lists end %d, block start %d", r.archive, at, loc.RawBlocksOff)
		}
		copy(r.position[:], body[loc.Off:loc.Off+12])
		// Six consecutive raw writes: 24+22+24+64, then mover180, order148.
		at += 24 + 22 + 24 + 64
		copy(r.mover[:], body[at:at+180])
		at += 180
		copy(r.order[:], body[at:at+148])
		at += 148
		r.routes[2], at, err = mover1160Words(body, at, loc.ControlOff)
		if err != nil {
			return out, fmt.Errorf("archive %d order path: %w", r.archive, err)
		}
		if at != loc.ControlOff {
			return out, fmt.Errorf("archive %d order path end %d, control start %d", r.archive, at, loc.ControlOff)
		}
		out.records = append(out.records, r)
	}
	slices.SortFunc(out.records, func(a, b mover1160Record) int { return int(a.archive) - int(b.archive) })
	return out, nil
}

func mover1160Words(body []byte, at, end int) ([]uint16, int, error) {
	if at < 0 || end > len(body) || at > end-2 {
		return nil, at, fmt.Errorf("truncated u16 count")
	}
	n := uint32(binary.LittleEndian.Uint16(body[at:]))
	at += 2
	if n == 0xffff {
		if at > end-4 {
			return nil, at, fmt.Errorf("truncated extended count")
		}
		n = binary.LittleEndian.Uint32(body[at:])
		at += 4
	}
	// The remaining bounded span, not an installed/decoded count, limits allocation.
	if uint64(n)*2 > uint64(end-at) {
		return nil, at, fmt.Errorf("count %d exceeds list span", n)
	}
	out := make([]uint16, int(n))
	for i := range out {
		out[i] = binary.LittleEndian.Uint16(body[at:])
		at += 2
	}
	return out, at, nil
}

func (r mover1160Record) blocks() []sav.DocumentRawData {
	out := []sav.DocumentRawData{{Name: "Block12", Bytes: r.position[:]}, {Name: "U154", Bytes: r.mover[:]}, {Name: "U158", Bytes: r.order[:]}}
	for i, name := range mover1160Lists {
		b := make([]byte, 0, len(r.routes[i])*2)
		for _, v := range r.routes[i] {
			b = binary.LittleEndian.AppendUint16(b, v)
		}
		out = append(out, sav.DocumentRawData{Name: name, Bytes: b})
	}
	return out
}
