package game

import (
	"errors"
	"fmt"
	"hash/fnv"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/mod"
	"againrom/pkg/namefold"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// bodiesAssets is the folder of the mod's body sheets under assets.
const bodiesAssets = "bodies"

// The bodies page's adders: a body from a picture, and a body for a weapon.
const (
	bodiesAddKey    = "newbody"
	bodiesWeaponKey = "newweapon"
)

// bodiesLook is what the bodies page reads of the install once: its body
// list, the bodies the game draws with under both directories and the
// registry class a new body starts from.
type bodiesLook struct {
	list     data.BodyList
	game     []data.HeroBody
	class    *terrain.UnitClass
	problems map[string]string
}

// bodiesLook reads the install's body list and unit class on first ask;
// without an install the parts are empty.
func (x *MapInspector) bodiesLook() *bodiesLook {
	if x.bodyArt != nil {
		return x.bodyArt
	}
	look := &bodiesLook{problems: map[string]string{}}
	x.bodyArt = look
	if x.units != nil {
		look.class = x.units.Classes[data.HeroUnmatchedClass]
	}
	if a := x.archives; a != nil {
		src := a.artSource()
		if list, ok := ReadBodyList(src); ok {
			look.list = list
			for i := 0; i < list.Len(); i++ {
				b := list.Entry(i)
				if b == "" || strings.HasSuffix(string(b), string(data.HeroShieldSuffix)) || slices.Contains(look.game, b) || !shippedBodyInBoth(src, b) {
					continue
				}
				look.game = append(look.game, b)
			}
		}
	}
	return look
}

// starter is the body a new sheet starts from: the geometry and timing of
// the game's own fighter body where the install gives one.
func (l *bodiesLook) starter() mod.BodySheet {
	action := func(frames, ticks int) mod.BodyAction {
		a := mod.BodyAction{Frames: frames}
		for range frames {
			a.Ticks = append(a.Ticks, ticks)
		}
		return a
	}
	b := mod.BodySheet{Width: 64, Height: 80, OriginX: 32, OriginY: 70, Directions: 8, Move: action(6, 2), Attack: action(5, 2)}
	c := l.class
	if c == nil {
		return b
	}
	if c.Width > 0 && c.Height > 0 && c.Width <= 512 && c.Height <= 512 && c.CenterX >= 0 && c.CenterX < c.Width && c.CenterY >= 0 && c.CenterY < c.Height {
		b.Width, b.Height, b.OriginX, b.OriginY = c.Width, c.Height, c.CenterX, c.CenterY
	}
	if c.Anim.D == 5 {
		b.Directions = 5
	}
	each := func(frames int, track []int) (int, int) {
		if frames < 1 || frames > 64 {
			return 0, 0
		}
		return frames, max(1, min(255, len(track)/frames))
	}
	if n, t := each(c.Anim.MoveSlot-c.Anim.MoveWind, c.Anim.MoveTrack); n > 0 {
		b.Move = action(n, t)
	}
	if n, t := each(c.Anim.AttackSlot, c.Anim.AttackTrack); n > 0 {
		b.Attack = action(n, t)
	}
	if n, t := each(c.Anim.IdleSlot, c.Anim.IdleTrack); n > 0 {
		b.Idle = action(n, t)
	}
	return b
}

// bodyFiles are the body sheets file and the weapon bodies file the project
// d holds: the long name of the second when that is the one it holds.
func bodyFiles(d *editorModDocument) []string {
	weapons := mod.WeaponBodiesFile
	if _, short := d.Files[mod.WeaponBodiesFile]; !short {
		if _, long := d.Files[mod.WeaponBodiesLongFile]; long {
			weapons = mod.WeaponBodiesLongFile
		}
	}
	return []string{mod.BodiesFile, weapons}
}

// bodyChoices are the bodies a weapon may be drawn with: the game's, then
// the mod's, each by name; extra, a body neither holds, stands last.
func (x *MapInspector) bodyChoices(bodies []mod.BodySheet, extra string) []string {
	var out []string
	for _, b := range x.bodiesLook().game {
		out = append(out, string(b))
	}
	for _, b := range bodies {
		if !b.Off && !slices.Contains(out, b.Name) {
			out = append(out, b.Name)
		}
	}
	if extra != "" && !slices.Contains(out, extra) {
		out = append(out, extra)
	}
	return out
}

// weaponRowOf is the weapon table row w names, 0 when the table has none.
func (x *MapInspector) weaponRowOf(w mod.WeaponBody) int {
	if x.table == nil || x.table.Weapons == nil {
		return 0
	}
	row, err := modWeaponRow(x.table.Weapons, w)
	if err != nil {
		return 0
	}
	return row
}

// weaponName is the weapon table's name of row, "" when it has none.
func (x *MapInspector) weaponName(row int) string {
	if x.table == nil || x.table.Weapons == nil {
		return ""
	}
	return weaponRowName(x.table.Weapons, row)
}

// weaponLabel is the heading of a weapon's body in the list.
func (x *MapInspector) weaponLabel(w mod.WeaponBody) string {
	name := w.Weapon
	if name == "" {
		name = x.weaponName(w.Row)
	}
	if name == "" {
		name = fmt.Sprintf(x.word("editor.bodies.weapon.row"), w.Row)
	}
	return fmt.Sprintf(x.word("editor.bodies.weapon.group"), name)
}

// bodiesFreeRows are the weapon rows the game draws with a body that the
// mod's weapon bodies leave alone, ascending.
func (x *MapInspector) bodiesFreeRows(weapons []mod.WeaponBody) []int {
	look := x.bodiesLook()
	var out []int
	if x.table == nil || x.table.Weapons == nil {
		return nil
	}
	for row := 1; row < x.table.Weapons.Len() && row <= mod.MaxWeaponRow; row++ {
		if x.weaponName(row) == "" || look.list.RowEntry(row) == "" {
			continue
		}
		if !slices.ContainsFunc(weapons, func(w mod.WeaponBody) bool { return x.weaponRowOf(w) == row }) {
			out = append(out, row)
		}
	}
	return out
}

// bodiesFields is W's bodies page: the button that adds a body from a
// picture; once the mod holds a body or a weapon body, the list of weapons
// to give a body, then each body's pictures, frames and timing and each
// weapon's body. A mod with none shows the one button.
func (s *mapInspectionEdits) bodiesFields() []ui.InspectionField {
	x := s.inspector
	if !s.modPageHolds() {
		return nil
	}
	project := s.workspace.Project
	if project == nil {
		project = x.mod.document()
	}
	held, err := s.heldRows(project, nil, bodyFiles(project)...)
	if err != nil {
		return []ui.InspectionField{x.pageLine("error", "", err.Error())}
	}
	out := []ui.InspectionField{{Key: bodiesAddKey, Label: x.word("editor.bodies.add"), Meaning: x.word("editor.bodies.add.help"), Action: true, Column: 1, Pick: []string{"png"}}}
	bodies, weapons := held.Bodies.Bodies, held.Bodies.Weapons
	if rows := x.bodiesFreeRows(weapons); len(rows) > 0 {
		pick := ui.InspectionField{Key: bodiesWeaponKey, Label: x.word("editor.bodies.weapon.add"), Meaning: x.word("editor.bodies.weapon.add.help"),
			Choices: []ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.bodies.weapon.choose"))}}}
		for _, row := range rows {
			pick.Choices = append(pick.Choices, ui.InspectionChoice{Value: int64(row), Label: x.inspectionUTF8(x.weaponName(row))})
		}
		out = append(out, pick)
	}
	num := func(key, label string, v int) ui.InspectionField {
		f := workspaceText(key, x.word("editor.bodies."+label), strconv.Itoa(v))
		f.Meaning = x.word("editor.bodies." + label + ".help")
		return f
	}
	for i, b := range bodies {
		key := "body." + strconv.Itoa(i) + "."
		start := len(out)
		name := workspaceText(key+"name", x.word("editor.bodies.name"), b.Name)
		name.Meaning = x.word("editor.bodies.name.help")
		out = append(out, name)
		if b.Off {
			out = append(out, x.pageLine(key+"state", "", x.word("editor.bodies.isoff")))
		} else {
			for _, p := range []struct{ field, label, path string }{{"picture", "picture", b.Heroes}, {"picturelight", "picturelight", b.HeroesLight}} {
				f := workspaceText(key+p.field, x.word("editor.bodies."+p.label), p.path)
				f.Meaning, f.Pick = x.word("editor.bodies."+p.label+".help"), []string{"png"}
				out = append(out, f)
			}
			if problem := s.bodyProblem(project, b); problem != "" {
				out = append(out, x.pageLine(key+"state", "", fmt.Sprintf(x.word("editor.bodies.state"), problem)))
			}
			dirs := ui.InspectionField{Key: key + "directions", Label: x.word("editor.bodies.directions"), Meaning: x.word("editor.bodies.directions.help"), Value: int64(b.Directions),
				Choices: []ui.InspectionChoice{{Value: 8, Label: x.inspectionUTF8(x.word("editor.bodies.directions.8"))}, {Value: 5, Label: x.inspectionUTF8(x.word("editor.bodies.directions.5"))}}}
			layer := int64(1)
			if b.WeaponLast {
				layer = 2
			}
			last := ui.InspectionField{Key: key + "weaponlast", Label: x.word("editor.bodies.weaponlast"), Meaning: x.word("editor.bodies.weaponlast.help"), Value: layer,
				Choices: []ui.InspectionChoice{{Value: 1, Label: x.inspectionUTF8(x.word("editor.bodies.weaponlast.no"))}, {Value: 2, Label: x.inspectionUTF8(x.word("editor.bodies.weaponlast.yes"))}}}
			out = append(out, num(key+"width", "width", b.Width), num(key+"height", "height", b.Height),
				num(key+"originx", "originx", b.OriginX), num(key+"originy", "originy", b.OriginY), dirs,
				num(key+"moveframes", "moveframes", b.Move.Frames), num(key+"movetime", "movetime", firstTick(b.Move)),
				num(key+"attackframes", "attackframes", b.Attack.Frames), num(key+"attacktime", "attacktime", firstTick(b.Attack)),
				num(key+"idleframes", "idleframes", b.Idle.Frames))
			if b.Idle.Frames > 0 {
				out = append(out, num(key+"idletime", "idletime", firstTick(b.Idle)))
			}
			out = append(out, last)
		}
		out = append(out, ui.InspectionField{Key: key + "remove", Label: x.word("editor.bodies.remove"), Meaning: x.word("editor.bodies.remove.help"), Action: true, Column: 1})
		for j := start; j < len(out); j++ {
			out[j].Group = b.Name
		}
	}
	for i, w := range weapons {
		key := "weapon." + strconv.Itoa(i) + "."
		start := len(out)
		names := x.bodyChoices(bodies, w.Body)
		f := ui.InspectionField{Key: key + "body", Label: x.word("editor.bodies.weapon.body"), Meaning: x.word("editor.bodies.weapon.body.help"), Value: int64(slices.Index(names, w.Body) + 1)}
		for n, name := range names {
			f.Choices = append(f.Choices, ui.InspectionChoice{Value: int64(n + 1), Label: x.inspectionUTF8(name)})
		}
		out = append(out, f, ui.InspectionField{Key: key + "remove", Label: x.word("editor.bodies.weapon.remove"), Meaning: x.word("editor.bodies.weapon.remove.help"), Action: true, Column: 1})
		for j := start; j < len(out); j++ {
			out[j].Group = x.weaponLabel(w)
		}
	}
	return out
}

// firstTick is the ticks each frame of a is held for, the first frame's when
// the frames differ.
func firstTick(a mod.BodyAction) int {
	if len(a.Ticks) == 0 {
		return 0
	}
	return a.Ticks[0]
}

// heldBytes is the bytes of the mod file rel in the project d, else in the
// folder; nil when there are none.
func (s *mapInspectionEdits) heldBytes(d *editorModDocument, rel string) []byte {
	if b, ok := d.Files[rel]; ok {
		return b
	}
	for name, b := range d.Files {
		if namefold.ModTree(name) == namefold.ModTree(rel) {
			return b
		}
	}
	if b, err := modDataFile(s.inspector.mod.dir, rel); err == nil {
		return b
	}
	return nil
}

// bodyProblem is why a game cannot cut the pictures of b into its frames, in
// the game's words, or "" when it can. The answer is kept for the same sheet
// and geometry.
func (s *mapInspectionEdits) bodyProblem(d *editorModDocument, b mod.BodySheet) string {
	look := s.inspector.bodiesLook()
	for _, rel := range []string{b.Heroes, b.HeroesLight} {
		raw := s.heldBytes(d, rel)
		if raw == nil {
			return fmt.Sprintf(s.inspector.word("editor.bodies.err.nopicture"), rel)
		}
		sum := fnv.New64a()
		sum.Write(raw)
		key := fmt.Sprintf("%d %d %d %d %d %v %v %v %d %x", b.Width, b.Height, b.OriginX, b.OriginY, b.Directions, b.Move.Ticks, b.Attack.Ticks, b.Idle.Ticks, len(raw), sum.Sum64())
		if got, ok := look.problems[key]; ok {
			if got != "" {
				return got
			}
			continue
		}
		problem := ""
		img, err := decodeBodySheet(raw)
		if err == nil {
			_, err = cutBodySheet(img, b)
		}
		if err != nil {
			problem = err.Error()
		}
		if len(look.problems) > 64 {
			clear(look.problems)
		}
		look.problems[key] = problem
		if problem != "" {
			return problem
		}
	}
	return ""
}

// editBodies is one draft step of the body sheets and weapon bodies files in
// the project w through editRows.
func (s *mapInspectionEdits) editBodies(w *editorWorkspaceState, edit func(*mod.BodyData) error) error {
	return kindFiles{s: s, w: w}.editRows(nil, bodyFiles(w.Project), func(d *mod.Data) error { return edit(&d.Bodies) })
}

// bodyName is the name a new body takes from the picture file: lower case
// letters, digits and _, cut to 32, and free among the mod's bodies and the
// bodies the game ships.
func (s *mapInspectionEdits) bodyName(bodies []mod.BodySheet, file string) string {
	var b strings.Builder
	for _, r := range namefold.Host(strings.TrimSuffix(file, filepath.Ext(file))) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	name := strings.Trim(b.String(), "_")
	if name == "" || name[0] < 'a' || name[0] > 'z' {
		name = "body_" + name
	}
	name = strings.TrimRight(name[:min(len(name), 28)], "_")
	free := func(n string) bool {
		return mod.ValidBodyName(n) && !slices.ContainsFunc(bodies, func(o mod.BodySheet) bool { return o.Name == n }) && !s.shippedBodyName(n)
	}
	for n, candidate := 2, name; ; n++ {
		if free(candidate) {
			return candidate
		}
		candidate = name + "_" + strconv.Itoa(n)
	}
}

// shippedBodyName reports whether the install ships a body of that name.
func (s *mapInspectionEdits) shippedBodyName(name string) bool {
	a := s.inspector.archives
	if a == nil {
		return false
	}
	return shippedBody(a.artSource(), data.HeroBody(name))
}

// addBody imports the host picture source as a new body's two sheets and
// adds the body in one draft step: the game's own body gives its frames and
// timing, and the one picture serves both armour directories until the
// light armour picture is replaced.
func (s *mapInspectionEdits) addBody(source string, replace bool) (*ui.InspectionDocument, error) {
	x := s.inspector
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	if source == "" {
		return nil, errors.New(x.word("editor.bodies.err.nofile"))
	}
	rel, picture, err := s.modAssetDraft(bodiesAssets, source, modAssetName(source))
	if err != nil {
		return nil, err
	}
	_, d, err := s.importModFile(rel, picture, replace, func() (*ui.InspectionDocument, error) {
		return s.addBody(source, true)
	}, func(w *editorWorkspaceState, rel string) error {
		return s.editBodies(w, func(b *mod.BodyData) error {
			sheet := x.bodiesLook().starter()
			sheet.Name, sheet.Heroes, sheet.HeroesLight = s.bodyName(b.Bodies, filepath.Base(source)), rel, rel
			b.Bodies = append(b.Bodies, sheet)
			return nil
		})
	})
	return d, err
}

// importBodyPicture imports the host picture source as the plain or the light
// armour sheet of body i in one draft step; the sheet it replaces leaves the
// mod when no other body names it.
func (s *mapInspectionEdits) importBodyPicture(i int, light bool, source string, replace bool) (*ui.InspectionDocument, error) {
	x := s.inspector
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	rel, picture, err := s.modAssetDraft(bodiesAssets, source, modAssetName(source))
	if err != nil {
		return nil, err
	}
	_, d, err := s.importModFile(rel, picture, replace, func() (*ui.InspectionDocument, error) {
		return s.importBodyPicture(i, light, source, true)
	}, func(w *editorWorkspaceState, rel string) error {
		var left []string
		err := s.editBodies(w, func(b *mod.BodyData) error {
			if i < 0 || i >= len(b.Bodies) || b.Bodies[i].Off {
				return x.modGone("editor.bodies.err.gone", strconv.Itoa(i))
			}
			path := &b.Bodies[i].Heroes
			if light {
				path = &b.Bodies[i].HeroesLight
			}
			old := *path
			*path = rel
			left = unusedSheets(b.Bodies, old)
			return nil
		})
		if err != nil {
			return err
		}
		return dropSheets(kindFiles{s: s, w: w}, left, rel)
	})
	return d, err
}

// unusedSheets are the pictures of old that no body of bodies names.
func unusedSheets(bodies []mod.BodySheet, old ...string) []string {
	var out []string
	for _, p := range old {
		if p != "" && !slices.ContainsFunc(bodies, func(b mod.BodySheet) bool { return b.Heroes == p || b.HeroesLight == p }) {
			out = append(out, p)
		}
	}
	return out
}

// dropSheets takes the pictures of left that sit under the bodies folder out
// of the project, but for keep.
func dropSheets(k kindFiles, left []string, keep string) error {
	for _, p := range left {
		if p == keep || !strings.HasPrefix(namefold.ModTree(p), mod.AssetsFolder+"/"+bodiesAssets+"/") {
			continue
		}
		if err := k.drop(p); err != nil {
			return err
		}
	}
	return nil
}

// setBodiesField makes one edit of the bodies page a draft step; a value the
// reader refuses is refused in the page's words.
func (s *mapInspectionEdits) setBodiesField(key, value string) (*ui.InspectionDocument, error) {
	return s.modPageEdit(key, value, s.bodiesEdit)
}

func (s *mapInspectionEdits) bodiesEdit(key, value string) (*ui.InspectionDocument, error) {
	if key == bodiesAddKey {
		return s.addBody(value, false)
	}
	gone := s.inspector.modGone("editor.bodies.err.gone", key)
	kind, i, field, ok := modRowKey(key)
	if key != bodiesWeaponKey && (!ok || kind != "body" && kind != "weapon") {
		return nil, gone
	}
	if kind == "body" && (field == "picture" || field == "picturelight") {
		return s.importBodyPicture(i, field == "picturelight", value, false)
	}
	return s.editKind(func(k kindFiles) error {
		var left []string
		err := s.editBodies(k.w, func(b *mod.BodyData) error {
			var err error
			switch {
			case key == bodiesWeaponKey:
				err = s.giveWeaponBody(b, value, gone)
			case kind == "body" && i < len(b.Bodies):
				left, err = s.editBody(b, i, field, value)
			case kind == "weapon" && i < len(b.Weapons):
				err = s.editWeaponBody(b, i, field, value, gone)
			default:
				err = gone
			}
			return err
		})
		if err == nil {
			err = dropSheets(k, left, "")
		}
		return err
	})
}

// giveWeaponBody adds a weapon body for the weapon row value names, drawn as
// the game draws it until the modder chooses another body.
func (s *mapInspectionEdits) giveWeaponBody(b *mod.BodyData, value string, gone error) error {
	x := s.inspector
	row, err := strconv.Atoi(value)
	if err != nil {
		return gone
	}
	if row == 0 {
		return nil
	}
	if !slices.Contains(x.bodiesFreeRows(b.Weapons), row) {
		return gone
	}
	body := string(x.bodiesLook().list.RowEntry(row))
	if names := x.bodyChoices(b.Bodies, ""); !slices.Contains(names, body) && len(names) > 0 {
		body = names[0]
	}
	b.Weapons = append(b.Weapons, mod.WeaponBody{Row: row, Body: body})
	return nil
}

// editWeaponBody makes the edit of field of weapon body i.
func (s *mapInspectionEdits) editWeaponBody(b *mod.BodyData, i int, field, value string, gone error) error {
	switch field {
	case "remove":
		b.Weapons = slices.Delete(b.Weapons, i, i+1)
	case "body":
		n, err := strconv.Atoi(value)
		names := s.inspector.bodyChoices(b.Bodies, b.Weapons[i].Body)
		if err != nil || n < 1 || n > len(names) {
			return gone
		}
		b.Weapons[i].Body = names[n-1]
	default:
		return gone
	}
	return nil
}

// editBody makes the edit of field of body i and answers the pictures a
// removal leaves unused.
func (s *mapInspectionEdits) editBody(b *mod.BodyData, i int, field, value string) ([]string, error) {
	x := s.inspector
	body := &b.Bodies[i]
	gone := x.modGone("editor.bodies.err.gone", field)
	if field == "remove" {
		var users []string
		for _, w := range b.Weapons {
			if w.Body == body.Name {
				users = append(users, x.weaponLabel(w))
			}
		}
		if len(users) > 0 {
			return nil, fmt.Errorf(x.word("editor.bodies.err.used"), body.Name, strings.Join(users, ", "))
		}
		old := []string{body.Heroes, body.HeroesLight}
		b.Bodies = slices.Delete(b.Bodies, i, i+1)
		return unusedSheets(b.Bodies, old...), nil
	}
	if field == "name" {
		if !mod.ValidBodyName(value) {
			return nil, errors.New(x.word("editor.bodies.err.name"))
		}
		if value != body.Name && (slices.ContainsFunc(b.Bodies, func(o mod.BodySheet) bool { return o.Name == value }) || s.shippedBodyName(value)) {
			return nil, fmt.Errorf(x.word("editor.bodies.err.taken"), value)
		}
		for j := range b.Weapons {
			if b.Weapons[j].Body == body.Name {
				b.Weapons[j].Body = value
			}
		}
		body.Name = value
		return nil, nil
	}
	if body.Off {
		return nil, gone
	}
	number := func(lo, hi int) (int, error) {
		n, err := strconv.Atoi(value)
		if err != nil || n < lo || n > hi {
			return 0, fmt.Errorf(x.word("editor.bodies.err.number"), lo, hi)
		}
		return n, nil
	}
	frames := func(a *mod.BodyAction, lo int) error {
		n, err := number(lo, 64)
		if err != nil {
			return err
		}
		ticks := 4
		if len(a.Ticks) > 0 {
			ticks = a.Ticks[len(a.Ticks)-1]
		} else if lo == 0 {
			ticks = max(1, firstTick(body.Move))
		}
		for len(a.Ticks) < n {
			a.Ticks = append(a.Ticks, ticks)
		}
		a.Ticks, a.Frames = a.Ticks[:n], n
		if n == 0 {
			*a = mod.BodyAction{}
		}
		return nil
	}
	time := func(a *mod.BodyAction) error {
		n, err := number(1, 255)
		if err != nil {
			return err
		}
		for j := range a.Ticks {
			a.Ticks[j] = n
		}
		return nil
	}
	var err error
	switch field {
	case "width", "height", "originx", "originy":
		w, h, ox, oy := body.Width, body.Height, body.OriginX, body.OriginY
		var n int
		if n, err = number(0, 512); err != nil {
			return nil, err
		}
		switch field {
		case "width":
			w = n
		case "height":
			h = n
		case "originx":
			ox = n
		default:
			oy = n
		}
		if w < 1 || h < 1 || ox >= w || oy >= h {
			return nil, errors.New(x.word("editor.bodies.err.frame"))
		}
		body.Width, body.Height, body.OriginX, body.OriginY = w, h, ox, oy
		if s := body.Selection; body.HasSelection && (s[2] > w || s[3] > h) {
			body.HasSelection, body.Selection = false, [4]int{}
		}
	case "directions":
		n, convErr := strconv.Atoi(value)
		if convErr != nil || n != 8 && n != 5 {
			return nil, gone
		}
		body.Directions = n
	case "weaponlast":
		n, convErr := strconv.Atoi(value)
		if convErr != nil || n != 1 && n != 2 {
			return nil, gone
		}
		body.WeaponLast = n == 2
	case "moveframes":
		err = frames(&body.Move, 1)
	case "movetime":
		err = time(&body.Move)
	case "attackframes":
		err = frames(&body.Attack, 1)
	case "attacktime":
		err = time(&body.Attack)
	case "idleframes":
		err = frames(&body.Idle, 0)
	case "idletime":
		err = time(&body.Idle)
	default:
		err = gone
	}
	return nil, err
}
