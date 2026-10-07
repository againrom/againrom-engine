package sim

import "encoding/binary"

// AI-ACTIVITY-324: the nonzero dispatch gate is a rebuilt member count.
// The single-player world uses SelfSlot for its human-controlled actors;
// its current living, on-map population supplies the spatial sources.
func (w *World) refreshSavedGroupActivity() {
	var covered [32][32]bool
	for _, e := range w.entities {
		if e.Owner != SelfSlot || !e.Alive() || e.OffMap {
			continue
		}
		x, y := int(e.X)>>3, int(e.Y)>>3
		for dx := -2; dx <= 2; dx++ {
			for dy := -2; dy <= 2; dy++ {
				if (dx == -2 || dx == 2) && (dy == -2 || dy == 2) {
					continue
				}
				cx, cy := x+dx, y+dy
				if cx >= 0 && cx < 32 && cy >= 0 && cy < 32 {
					covered[cx][cy] = true
				}
			}
		}
	}
	for gi := range w.savedGroups.Groups {
		g := &w.savedGroups.Groups[gi]
		// Explicit native command groups keep their existing command lifetime.
		// This refresh repairs retained original groups (DIV-351).
		if g.Authored {
			continue
		}
		forced := binary.LittleEndian.Uint32(g.AI[0x48:]) != 0
		var count uint8
		represented := false
		for _, member := range g.Members {
			if !member.Bound {
				continue
			}
			i := indexOfEntity(w.entities, member.Entity)
			if i < 0 {
				continue
			}
			e := w.entities[i]
			if e.Owner == SelfSlot || !e.Alive() || e.OffMap {
				continue
			}
			represented = true
			x, y := int(e.X)>>3, int(e.Y)>>3
			if forced || x >= 0 && x < 32 && y >= 0 && y < 32 && covered[x][y] {
				count++
			}
		}
		if represented {
			g.AI[0x45] = count
		}
	}
}
