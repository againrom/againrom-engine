package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type playerByteSection struct {
	typeID          uint32
	header, payload []byte
}

func playerByteSections(t *testing.T, raw []byte) ([]playerByteSection, []byte) {
	t.Helper()
	if len(raw) < 20 {
		t.Fatal("short map header")
	}
	var out []playerByteSection
	at := 20
	for n := binary.LittleEndian.Uint32(raw[12:]); n > 0; n-- {
		if at+20 > len(raw) {
			t.Fatal("short section header")
		}
		h := raw[at : at+20]
		size := uint64(binary.LittleEndian.Uint32(h[8:]))
		if size > uint64(len(raw)-at-20) {
			t.Fatal("section exceeds map")
		}
		end := at + 20 + int(size)
		out = append(out, playerByteSection{binary.LittleEndian.Uint32(h[12:]), h, raw[at+20 : end]})
		at = end
	}
	return out, raw[at:]
}

func TestReleaseMapEditorPlayerEditsChangeOnlyRosterAndFraming(t *testing.T) {
	x := installedInspector(t)
	e := x.Editor()
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	installed := s.model.Bytes()
	baseline, err := s.model.SaveBytes()
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(installed)
	if err != nil {
		t.Fatal(err)
	}
	if d, err := s.AddPlayer(); err != nil || d == nil {
		t.Fatal("add player", err)
	}
	if d, err := s.SetField("Player", len(m.Groups), "relation0", 0); err != nil || d == nil {
		t.Fatal("new player's relation", err)
	}
	target := filepath.Join(t.TempDir(), "players.alm")
	if _, err := s.SaveAs(target); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	a, atail := playerByteSections(t, baseline)
	b, btail := playerByteSections(t, saved)
	if len(a) != len(b) || !bytes.Equal(atail, btail) || !bytes.Equal(baseline[:20], saved[:20]) {
		t.Fatal("roster edit changed outer framing or trailer")
	}
	for i, before := range a {
		after := b[i]
		if before.typeID != after.typeID {
			t.Fatal("section order changed")
		}
		h := bytes.Clone(after.header)
		if before.typeID == 5 {
			copy(h[8:12], before.header[8:12])
		}
		if !bytes.Equal(before.header, h) {
			t.Fatalf("type %d changed unrelated framing bytes", before.typeID)
		}
		p := bytes.Clone(after.payload)
		switch before.typeID {
		case 5:
			if len(p) != len(before.payload)+76 {
				t.Fatalf("roster grew by %d, want 76", len(p)-len(before.payload))
			}
			continue
		case 0:
			if binary.LittleEndian.Uint32(p[0x1c:]) != uint32(len(m.Groups)+1) {
				t.Fatal("roster count did not grow")
			}
			copy(p[0x1c:0x20], before.payload[0x1c:0x20])
		}
		if !bytes.Equal(before.payload, p) {
			t.Fatalf("roster edit changed type %d outside its count word", before.typeID)
		}
	}
	var saveOffsets []int
	if len(installed) != len(baseline) {
		t.Fatal("existing save form changed installed length")
	}
	for i := range installed {
		if installed[i] != baseline[i] {
			saveOffsets = append(saveOffsets, i)
		}
	}
	t.Logf("%s scenario/10.alm: installed %d bytes, saved %d; existing SaveForm differing byte offsets %v; only type-5 payload/length and type-0 +0x1c differ from baseline SaveForm", filepath.Base(os.Getenv("AGAINROM_ASSETS")), len(installed), len(saved), saveOffsets)
	writePlayerPanelWitness(t, x, target)
}

func writePlayerPanelWitness(t *testing.T, x *MapInspector, source string) {
	t.Helper()
	dir := os.Getenv("AGAINROM_EDITOR_PLAYERS_WITNESS_DIR")
	if dir == "" {
		return
	}
	if !filepath.IsAbs(dir) {
		t.Fatal("AGAINROM_EDITOR_PLAYERS_WITNESS_DIR must be absolute")
	}
	if err := refuseOriginalWriteTarget(dir, os.Getenv("AGAINROM_ASSETS")); err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	if err := e.Open(source); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	if err := e.Key("players"); err != nil {
		t.Fatal(err)
	}
	groups := windowPlayers(t, e)
	e.Select(recordIndex(t, e, "Player", len(groups)-1), false)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"panel", "name"} {
		if name == "name" {
			if err := e.Key("field"); err != nil {
				t.Fatal(err)
			}
		}
		path := filepath.Join(dir, "players-"+name+"-"+filepath.Base(os.Getenv("AGAINROM_ASSETS"))+".png")
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		encodeErr, closeErr := png.Encode(f, e.PanelImage(800)), f.Close()
		if encodeErr != nil || closeErr != nil {
			t.Fatal("player panel witness", encodeErr, closeErr)
		}
		t.Logf("headless player %s: %s", name, path)
	}
}

func playerColourClass(t *testing.T, x *MapInspector) int16 {
	t.Helper()
	raw, err := x.archives.Containers.ReadFile("scenario/10.alm")
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range m.Units {
		c := x.units.Classes[int32(u.ClassID)]
		if c != nil && c.OwnerShaded && mapload.Resolve(alm.Unit{ClassID: u.ClassID}, x.table).Found() {
			return u.ClassID
		}
	}
	t.Fatal("mission 10 offers no resolvable owner-shaded unit class")
	return 0
}

func playerMissionMap(t *testing.T, x *MapInspector, class int16, edited bool) string {
	t.Helper()
	root := newMapMod(t)
	if err := x.SetMod(filepath.Join(root, newMapModID)); err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	if err := e.Open(newMapSource(32, 32)); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	apply := func(d *ui.InspectionDocument, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	apply(s.SetDropCell(image.Pt(16, 16)))
	apply(s.AddUnit(image.Pt(19, 16), int64(class)))
	apply(s.SetField("Player", 0, "relation1", 0))
	apply(s.SetField("Player", 1, "relation0", 0))
	if edited {
		apply(s.AddPlayer())
		apply(s.SetField("Player", 0, "relation2", 0))
		apply(s.SetTextField("Player", 2, "name", "Copper"))
		apply(s.SetField("Player", 2, "colour", 9))
		apply(s.SetField("Player", 2, "control", 1))
		apply(s.SetField("Player", 2, "relation0", 1))
		apply(s.SetField("Unit", 0, "owner", 3))
	}
	if err := e.Key("saveas"); err != nil || e.Dirty() {
		t.Fatalf("save into mod: %v, dirty %v", err, e.Dirty())
	}
	return root
}

func playerMissionActor(t *testing.T, f *FrontEnd) sim.Entity {
	t.Helper()
	for _, e := range f.live.world.Entities() {
		if e.MapUnitID == 1 {
			return e
		}
	}
	t.Fatal("authored unit is absent")
	return sim.Entity{}
}

func assertPlayerColour(t *testing.T, f *FrontEnd, e sim.Entity, shade uint32) {
	t.Helper()
	palettes := releaseOwnerTables(t, f)
	for _, draw := range f.live.entityDraws() {
		if draw.ID != uint32(e.ID) {
			continue
		}
		if draw.Art == nil || !draw.Art.OwnerShaded || draw.Frame == nil || draw.Frame.Palette != palettes[shade] {
			t.Fatal("authored unit did not select its player's installed palette")
		}
		for i, px := range draw.Frame.Pixels {
			if !px.Opaque || palettes[shade][px.Index] == palettes[e.Owner&15][px.Index] {
				continue
			}
			x, y := i%draw.Frame.Width, i/draw.Frame.Width
			want := palettes[shade][px.Index]
			if got := draw.Frame.RGBALit([3]uint8{}, 8).RGBAAt(x, y); got != want {
				t.Fatal("production lit body pixel", got, "want", want)
			}
			t.Logf("owner %d shade %d body pixel (%d,%d), palette index %d = %v; ordinal loss control %v", e.Owner, shade, x, y, px.Index, want, palettes[e.Owner&15][px.Index])
			return
		}
		t.Fatal("body offers no visible pixel differing from the ordinal palette")
	}
	t.Fatal("authored unit has no production draw")
}

func TestReleaseMapEditorPlayersPlayWithAuthoredColoursAndRelations(t *testing.T) {
	x := installedInspector(t)
	class := playerColourClass(t, x)
	plain, edited := playerMissionMap(t, x, class, false), playerMissionMap(t, x, class, true)
	before, _ := startArmsMission(t, plain)
	after, _ := startArmsMission(t, edited)
	old, actor := playerMissionActor(t, before), playerMissionActor(t, after)
	if old.Owner != 2 || actor.Owner != 3 || len(before.live.mission.state.Map.Groups) != 2 {
		t.Fatal("player ownership loss control", old.Owner, actor.Owner)
	}
	m := after.live.mission.state.Map
	if len(m.Groups) != 3 || m.Groups[2].Name != "Copper" || m.Groups[2].Color != 9 || m.Groups[2].Participant != 1 {
		t.Fatal("saved roster did not reach production mod mission", m.Groups)
	}
	players, present := after.live.world.CurrentPlayers()
	controls, controlPresent := after.live.world.PlayerParticipants()
	computer := false
	for i, p := range players {
		if p.Slot == 3 && i < len(controls) {
			computer = controls[i].PlayerID == p.ID && controls[i].Value == 1
		}
	}
	if !present || !controlPresent || !computer {
		t.Fatal("computer control is absent from the current Player carrier", players, controls)
	}
	assertPlayerColour(t, after, actor, 9)
	br, ar := before.live.world.Relations(), after.live.world.Relations()
	if br.Hostile(2, 1) || br.Hostile(3, 1) || !ar.Hostile(3, 1) || ar.Hostile(1, 3) {
		t.Fatalf("directed hostility differs from saved rows: control 2->1=%d 3->1=%d, edited 3->1=%d 1->3=%d", br.Byte(2, 1), br.Byte(3, 1), ar.Byte(3, 1), ar.Byte(1, 3))
	}
	snapshot, label, err := after.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	checkLoadedPlayers := func(f *FrontEnd, route string) {
		t.Helper()
		loadedActor := playerMissionActor(t, f)
		if loadedActor.Owner != actor.Owner {
			t.Fatal(route, "lost authored unit owner", loadedActor.Owner, actor.Owner)
		}
		assertPlayerColour(t, f, loadedActor, 9)
		if got, present := f.live.world.PlayerParticipants(); !present || !slices.Equal(got, controls) {
			t.Fatal(route, "lost computer control", got, controls)
		}
		if rel := f.live.world.Relations(); !rel.Hostile(3, 1) || rel.Hostile(1, 3) {
			t.Fatal(route, "lost directed hostility")
		}
		t.Logf("%s %s retained owner %d, colour 9, computer controls and directed hostility", filepath.Base(os.Getenv("AGAINROM_ASSETS")), route, loadedActor.Owner)
	}
	raw, err := after.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	cold := guestFront(t, edited, newMapModID)
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("legacy AGS cold mod LOAD", town, err)
	}
	if err := cold.App("player colours after LOAD").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	checkLoadedPlayers(cold, "legacy AGS cold LOAD")
	raw, err = after.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal("production SAV export", err)
	}
	coldSAV := guestFront(t, edited, newMapModID)
	open, town, err = coldSAV.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("production SAV cold mod LOAD", town, err)
	}
	if err := coldSAV.App("player colours after SAV LOAD").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	checkLoadedPlayers(coldSAV, "production SAV cold LOAD")
	acquired := false
	for tick := 0; tick < 160; tick++ {
		sim.Step(before.live.world, nil)
		sim.Step(after.live.world, nil)
		plainActor, editedActor := playerMissionActor(t, before), playerMissionActor(t, after)
		if plainActor.HasAttackTarget {
			t.Fatal("loss control: the neutral unit acquired a target")
		}
		if editedActor.HasAttackTarget {
			victim, ok := after.live.world.Entity(editedActor.AttackTarget)
			if ok && victim.Owner == 1 {
				acquired = true
				t.Logf("%s production mod mission: computer owner %d acquired owner %d at tick %d; neutral control held no target", filepath.Base(os.Getenv("AGAINROM_ASSETS")), actor.Owner, victim.Owner, tick+1)
				break
			}
		}
	}
	if !acquired {
		t.Fatalf("hostile authored owner acquired no first-player unit in 160 ticks; class %d", class)
	}
}
