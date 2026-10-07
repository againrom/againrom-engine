package sim

type rom2GroupKey struct{ owner, group uint32 }
type rom2Activity struct{ active map[rom2GroupKey]bool }

func (w *World) rom2ActivityMask() *rom2Activity {
	if w.rom2 == nil {
		return nil
	}
	var covered [32][32]bool
	for _, e := range w.entities {
		if e.Owner != SelfSlot || !e.Alive() || e.OffMap {
			continue
		}
		x, y := int(e.X)>>3, int(e.Y)>>3
		for dx := -2; dx <= 2; dx++ {
			for dy := -2; dy <= 2; dy++ {
				cx, cy := x+dx, y+dy
				if cx >= 0 && cx < 32 && cy >= 0 && cy < 32 {
					covered[cx][cy] = true
				}
			}
		}
	}
	a := &rom2Activity{active: make(map[rom2GroupKey]bool)}
	for _, row := range w.rom2.Groups {
		if row.Forced || row.Owner == SelfSlot {
			a.active[rom2GroupKey{row.Owner, row.Group}] = true
		}
	}
	for _, e := range w.entities {
		if !e.Alive() || e.OffMap {
			continue
		}
		x, y := int(e.X)>>3, int(e.Y)>>3
		if e.Owner == SelfSlot || e.HP < e.MaxHP || x >= 0 && x < 32 && y >= 0 && y < 32 && covered[x][y] {
			a.active[rom2GroupKey{e.Owner, effectiveGroup(e)}] = true
		}
	}
	return a
}

func (a *rom2Activity) groupActive(owner, group uint32) bool {
	return a == nil || a.active[rom2GroupKey{owner, group}]
}
func (a *rom2Activity) actorActive(e Entity) bool {
	return e.CommandGroup != 0 || a.groupActive(e.Owner, effectiveGroup(e))
}

func (w *World) refreshROM2SavedGroupActivity(a *rom2Activity) {
	for gi := range w.savedGroups.Groups {
		g := &w.savedGroups.Groups[gi]
		if g.Authored {
			continue
		}
		represented, active := false, false
		for _, m := range g.Members {
			if !m.Bound {
				continue
			}
			i := indexOfEntity(w.entities, m.Entity)
			if i < 0 {
				continue
			}
			represented = true
			e := w.entities[i]
			active = active || a.groupActive(e.Owner, effectiveGroup(e))
		}
		if represented {
			g.AI[0x45] = 0
			if active {
				g.AI[0x45] = 1
			}
		}
	}
}
