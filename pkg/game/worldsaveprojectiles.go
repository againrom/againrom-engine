package game

import (
	"encoding/binary"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// SAV-PROJSTORE-428/SAV-PROJLOAD-429 define the sixteen typed leaves and the
// independent allocator/ordered IDs. This constructor consumes complete live
// records, never a presentation cast event or an omitted constructor default.
func constructProjectileLeaves(doc *sav.DocumentData, current sim.SavedProjectiles, bound []uint16, found map[string]bool) error {
	dirs := map[string]bool{}
	paths := map[string]bool{}
	for _, r := range doc.State.DirectoryRecords {
		dirs[foldStatePath(r.Path)] = true
	}
	for _, r := range doc.State.ValueRecords {
		paths[foldStatePath(r.Path)] = true
	}
	addDir := func(path string) {
		if !dirs[foldStatePath(path)] {
			doc.State.DirectoryRecords = append(doc.State.DirectoryRecords, sav.CityStateDirectoryData{Path: path, Kind: 1})
			dirs[foldStatePath(path)] = true
		}
	}
	addDir("/Projectiles")
	if !paths["/projectiles/freeindex"] {
		doc.State.ValueRecords = append(doc.State.ValueRecords, sav.CityStateRecordData{Path: "/Projectiles/FreeIndex", Value: sav.CityStateValueData{Kind: 2, Int32: int32(current.FreeIndex)}})
	}
	if !paths["/projectiles/ids"] {
		data := make([]byte, 4*len(current.IDs))
		for i, id := range current.IDs {
			binary.LittleEndian.PutUint32(data[4*i:], uint32(id))
		}
		doc.State.ValueRecords = append(doc.State.ValueRecords, sav.CityStateRecordData{Path: "/Projectiles/IDs", Value: sav.CityStateValueData{Kind: 6, Bytes: data}})
	}
	for _, p := range current.Items {
		if !slices.Contains(bound, p.ID) {
			continue
		}
		root := "/Prj" + strconv.Itoa(int(p.ID))
		count := 0
		for _, name := range projectileFieldNames {
			if found[foldStatePath(root+"/"+name)] {
				count++
			}
		}
		if count == len(projectileFieldNames) {
			continue
		}
		if count != 0 || dirs[foldStatePath(root)] {
			return fmt.Errorf("current projectile %d has a partial existing section", p.ID)
		}
		addDir(root)
		fields := [...]int32{p.X, p.Y, p.Z, p.Picture, p.Dir, p.Phase, p.LastAction, p.Action, p.ActionDir, p.ActionTarget, p.ActionX, p.ActionY, p.ActionZ, p.ActionPhase, p.ActionSegments, p.ActionSpell}
		for i, name := range projectileFieldNames {
			doc.State.ValueRecords = append(doc.State.ValueRecords, sav.CityStateRecordData{Path: root + "/" + name, Value: sav.CityStateValueData{Kind: 2, Int32: fields[i]}})
		}
	}
	slices.SortFunc(doc.State.DirectoryRecords, func(a, b sav.CityStateDirectoryData) int { return strings.Compare(a.Path, b.Path) })
	slices.SortFunc(doc.State.ValueRecords, func(a, b sav.CityStateRecordData) int { return strings.Compare(a.Path, b.Path) })
	return nil
}
