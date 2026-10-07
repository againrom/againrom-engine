package game

import (
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// releaseOwnerTables reads the release resource independently of the
// production palette decoder. The expected table and channel order below come
// straight from the raw [B, G, R, reserved] bytes, so a shared decoder defect
// cannot make the loader and this witness agree.
func releaseOwnerTables(t *testing.T, f *FrontEnd) [16][256]color.RGBA {
	t.Helper()
	raw, err := f.Archives.Containers.ReadFile(UnitOwnerPalette)
	if err != nil {
		t.Fatalf("read %s: %v", UnitOwnerPalette, err)
	}
	if len(raw) != 16*256*4 {
		t.Fatalf("%s has %d bytes, want 16384", UnitOwnerPalette, len(raw))
	}
	var out [16][256]color.RGBA
	for table := range out {
		for i := range out[table] {
			e := raw[(table*256+i)*4:]
			out[table][i] = color.RGBA{R: e[2], G: e[1], B: e[0], A: 0xff}
		}
	}
	return out
}

func releaseBaseFrame(c *terrain.UnitClass, drawn *terrain.StaticFrame) *terrain.StaticFrame {
	if c == nil || drawn == nil {
		return nil
	}
	for _, frame := range c.Frames {
		if frame == drawn {
			return frame
		}
		if frame == nil || frame.Width != drawn.Width || frame.Height != drawn.Height ||
			len(frame.Pixels) != len(drawn.Pixels) {
			continue
		}
		if len(frame.Pixels) == 0 || &frame.Pixels[0] == &drawn.Pixels[0] {
			return frame
		}
	}
	return nil
}

func TestReleaseCampaignMapHumanoidsUseTheirOwnerShade(t *testing.T) {
	f := releaseFront(t)
	wantTables := releaseOwnerTables(t, f)
	if !f.Units.HasOwnerPalettes {
		t.Fatal("front end did not load the shared owner palette")
	}

	registryBytes, err := f.Archives.Containers.ReadFile(UnitRegistry)
	if err != nil {
		t.Fatalf("read %s: %v", UnitRegistry, err)
	}
	registry, err := reg.Parse(registryBytes)
	if err != nil {
		t.Fatalf("parse %s: %v", UnitRegistry, err)
	}
	registryClasses, err := data.LoadUnitClasses(registry)
	if err != nil {
		t.Fatalf("load %s: %v", UnitRegistry, err)
	}
	ownerClasses := 0
	for _, row := range registryClasses.All() {
		want := row.Palette == 0
		class := f.Units.Classes[row.ID]
		if class == nil {
			t.Errorf("registry class %d has no loaded render class", row.ID)
			continue
		}
		if class.OwnerShaded != want {
			t.Errorf("registry class %d Palette=%d: OwnerShaded=%t, want %t",
				row.ID, row.Palette, class.OwnerShaded, want)
		}
		if want {
			ownerClasses++
		}
	}
	if ownerClasses != 18 {
		t.Fatalf("unit bundle marks %d owner-shaded classes, want the shipped 18", ownerClasses)
	}

	missions := make([]int, 0, len(f.Campaign.Value().Chapters))
	for n := range f.Campaign.Value().Chapters {
		missions = append(missions, n)
	}
	sort.Ints(missions)

	placements, eligible, visible, hidden := 0, 0, 0, 0
	var slots [16]int
	mission20Witness := false
	for _, n := range missions {
		m := releaseMissionMap(t, f, n)
		mapload.WithdrawBorderPlacements(m)
		if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpenerWith(n, nil)(); err != nil {
			t.Fatalf("open mission %d: %v", n, err)
		}
		if f.live == nil || f.live.world == nil {
			t.Fatalf("mission %d opened without a live world", n)
		}

		entities := make(map[sim.EntityID]sim.Entity, len(m.Units))
		for _, entity := range f.live.world.Entities() {
			entities[entity.ID] = entity
		}
		draws := make(map[uint32]struct {
			art   *terrain.UnitClass
			frame *terrain.StaticFrame
		}, len(m.Units))
		for _, draw := range f.live.entityDraws() {
			draws[draw.ID] = struct {
				art   *terrain.UnitClass
				frame *terrain.StaticFrame
			}{draw.Art, draw.Frame}
		}

		placements += len(m.Units)
		for i := range m.Units {
			id := sim.EntityID(i)
			entity, ok := entities[id]
			if !ok {
				t.Errorf("mission %d map placement %d has no world entity", n, i)
				continue
			}
			class := f.Units.Classes[entity.Class]
			if class == nil || !class.OwnerShaded {
				continue
			}
			eligible++
			shade := int(entity.Owner & 0x0f)
			slots[shade]++
			draw, ok := draws[uint32(id)]
			if !ok {
				hidden++
				continue
			}
			visible++
			if draw.art == nil || !draw.art.OwnerShaded {
				t.Errorf("mission %d entity %d class %d selected non-owner art", n, id, entity.Class)
				continue
			}
			if draw.frame == nil {
				t.Errorf("mission %d entity %d class %d selected no frame", n, id, entity.Class)
				continue
			}
			if draw.frame.Palette != wantTables[shade] {
				t.Errorf("mission %d entity %d class %d owner %d has a palette other than table %d",
					n, id, entity.Class, entity.Owner, shade)
				continue
			}

			base := releaseBaseFrame(draw.art, draw.frame)
			if base == nil || n != 20 || mission20Witness {
				continue
			}
			for pixel, px := range draw.frame.Pixels {
				if !px.Opaque || base.Palette[px.Index] == wantTables[shade][px.Index] {
					continue
				}
				x, y := pixel%draw.frame.Width, pixel/draw.frame.Width
				want := wantTables[shade][px.Index]
				if got := draw.frame.RGBA().RGBAAt(x, y); got != want {
					t.Fatalf("mission 20 entity %d unlit pixel %d = %+v, want %+v", id, pixel, got, want)
				}
				// Sprite row 8 at zero tint is the raw palette. This runs the
				// production lit walk while retaining an independently writable
				// colour expectation.
				if got := draw.frame.RGBALit([3]uint8{}, 8).RGBAAt(x, y); got != want {
					t.Fatalf("mission 20 entity %d lit pixel %d = %+v, want %+v", id, pixel, got, want)
				}
				t.Logf("mission 20 witness: entity=%d class=%d owner=%d shade=%d pixel=%d index=%d base=%+v selected=%+v",
					id, entity.Class, entity.Owner, shade, pixel, px.Index, base.Palette[px.Index], want)
				mission20Witness = true
				break
			}
		}
	}

	if len(missions) == 0 || placements == 0 || eligible == 0 || visible == 0 {
		t.Fatalf("campaign census is empty: missions=%d placements=%d eligible=%d visible=%d",
			len(missions), placements, eligible, visible)
	}
	if !mission20Witness {
		t.Fatal("mission 20 has no visible owner-shaded pixel which differs from its base palette")
	}
	t.Logf("%s campaign owner-shade census: missions=%d placements=%d eligible=%d visible=%d hidden=%d slots=%v",
		filepath.Base(os.Getenv("AGAINROM_ASSETS")), len(missions), placements, eligible, visible, hidden, slots)
}
