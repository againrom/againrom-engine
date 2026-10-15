package game

import (
	"againrom/pkg/sim"
	"encoding/binary"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
)

func projectileRecordsCheck(t *testing.T, want projectile1157Source, f *FrontEnd) {
	t.Helper()
	if f.live == nil {
		t.Fatal("missing mission driver")
	}
	if diff := want.worldDifferences(f.live.world.SavedProjectiles()); len(diff) != 0 {
		t.Fatalf("raw source vs World: %v", diff)
	}
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if diff := want.documentDifferences(snapshot.SavedDocument); len(diff) != 0 {
		t.Fatalf("raw source vs retained Document: %v", diff)
	}
}

// Independent current-state oracle: saved source leaves plus frozen import
// parameters; it never reads the production projectile after stepping it.
func projectile1162Next(src projectile1157Source, bindings map[uint16]sim.SavedProjectileDriver, w *sim.World) projectile1157Source {
	out := src
	out.dirs = maps.Clone(src.dirs)
	out.values = maps.Clone(src.values)
	out.ids = slices.Clone(src.ids)
	out.items = nil
	for _, item := range src.items {
		p := item.fields
		d, armed := bindings[item.id]
		root := fmt.Sprintf("/Prj%d", item.id)
		if armed && p[14] == 0 {
			out.ids = slices.DeleteFunc(out.ids, func(id uint16) bool { return id == item.id })
			delete(out.dirs, root)
			for path := range out.values {
				if strings.HasPrefix(path, root+"/") {
					delete(out.values, path)
				}
			}
			continue
		}
		if armed {
			if d.HasTarget {
				for _, e := range w.Entities() {
					if e.ID != d.Target {
						continue
					}
					x, y := e.X*256+128, e.Y*256+128
					if fx, fy, ok := w.ActorFinePosition(e.ID); ok {
						x, y = e.X*256+int32(fx), e.Y*256+int32(fy)
					} else if e.Stride.Present && e.Transit > 0 {
						paid := int32(e.TransitTotal - e.Transit)
						x, y = e.Stride.FromX*256+128+paid*int32(e.Stride.StepX), e.Stride.FromY*256+128+paid*int32(e.Stride.StepY)
					}
					p[10], p[11] = x, y
					break
				}
			}
			p[0] += (p[10] - p[0]) / p[14]
			p[1] += (p[11] - p[1]) / p[14]
			p[2] += (p[12] - p[2]) / p[14]
			p[13]++
			p[5] = 0
			if d.Phases > 0 {
				p[5] = (p[13] / 2) % int32(d.Phases)
			}
			p[6] = p[7]
			p[14]--
		}
		item.fields = p
		out.items = append(out.items, item)
		for i, name := range projectile1157Names {
			key := root + "/" + name
			r := out.values[key]
			r.word = uint32(p[i])
			out.values[key] = r
		}
	}
	if !slices.Equal(out.ids, src.ids) {
		v := out.values["/Projectiles/IDs"]
		v.kind, v.word = 6, 0
		v.data = make([]byte, 4*len(out.ids))
		for i, id := range out.ids {
			binary.LittleEndian.PutUint32(v.data[i*4:], uint32(id))
		}
		out.values["/Projectiles/IDs"] = v
	}
	return out
}
