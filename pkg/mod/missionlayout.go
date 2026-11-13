package mod

import (
	"path"
	"slices"
	"strings"

	"againrom/pkg/locale"
	"againrom/pkg/namefold"
)

// FileMove writes the file at From again at To.
type FileMove struct{ From, To string }

// LayoutChange is a mod's mission rows moved into the tree: the rows as they
// read after, the files written at a new path, and the old files no row
// reads any more.
type LayoutChange struct {
	Missions []Mission
	Moves    []FileMove
	Removed  []string
	// Places are the declared map and texts of each moved group of rows,
	// before and after.
	Places []PlaceChange
}

// PlaceChange is one moved group's declared map and texts, before and after.
type PlaceChange struct{ Map, Texts, NewMap, NewTexts string }

// Rebase is rows, read from another state of the same mod, with each row
// that declares a moved group's map and texts placed as the group is: the
// change applied to rows it was not computed from.
func (c LayoutChange) Rebase(rows []Mission) []Mission {
	out := slices.Clone(rows)
	for i := range out {
		m := &out[i]
		if m.Off || m.Map == "" || m.byName {
			continue
		}
		t, tt := m.declared(m.Map, m.mapTemplate), m.declared(m.Texts, m.textsTemplate)
		for _, p := range c.Places {
			if namefold.ModTree(p.Map) == namefold.ModTree(t) && p.Texts == tt {
				m.place(p.NewMap, p.NewTexts)
				break
			}
		}
	}
	return out
}

// place sets the row's map and texts to the declarations newT and newTexts,
// read in the row's language.
func (m *Mission) place(newT, newTexts string) {
	m.mapTemplate = ""
	if strings.Contains(newT, langToken) {
		m.mapTemplate = newT
	}
	m.Map = path.Clean(missionPath(newT, m.language))
	m.Texts, m.textsTemplate = "", ""
	if newTexts != "" {
		m.Texts, m.textsTemplate = path.Clean(missionPath(newTexts, m.language)), newTexts
	}
}

// TreeLayout moves each mission row whose map or texts folder lies outside
// the tree into the place rule (MissionPlace), with the files it reads. The
// map, its labels and the pages beside it keep their names under the new
// stem. The pages of the texts folder, and of its parent when its tail is a
// language, join the pages beside the map unless a page the reader takes
// first holds the name. A row whose key extended names takes the .almx
// extension. Rows that share a map but not a texts folder, and a row whose
// new file a held file already takes, are left. held is every mod file.
func TreeLayout(rows []Mission, held []string, extended map[string]bool) LayoutChange {
	out := LayoutChange{Missions: slices.Clone(rows)}
	h := newHeldFiles(held)
	type group struct {
		mapT, textsT string
		rows         []int
		mixed        bool
	}
	var groups []*group
	byMap := map[string]*group{}
	for i, m := range rows {
		if m.Off || m.Map == "" || m.byName {
			continue
		}
		t, tt := m.declared(m.Map, m.mapTemplate), m.declared(m.Texts, m.textsTemplate)
		g := byMap[namefold.ModTree(t)]
		if g == nil {
			g = &group{mapT: t, textsT: tt}
			byMap[namefold.ModTree(t)] = g
			groups = append(groups, g)
		}
		g.mixed = g.mixed || g.textsT != tt
		g.rows = append(g.rows, i)
	}
	var claimed []string
	moved := map[string]FileMove{}
	changed := map[int]bool{}
	for _, g := range groups {
		if g.mixed {
			continue
		}
		almx := slices.ContainsFunc(g.rows, func(i int) bool { return extended[rows[i].Key] })
		newT, newTexts, moves, ok := h.layout(g.mapT, g.textsT, almx, claimed)
		if !ok || newT == g.mapT && newTexts == g.textsT {
			continue
		}
		if slices.ContainsFunc(moves, func(mv FileMove) bool {
			prior, taken := moved[namefold.ModTree(mv.To)]
			return taken && prior.From != mv.From
		}) {
			continue
		}
		for _, l := range layoutLanguages() {
			claimed = append(claimed, missionPath(newT, l))
		}
		for _, mv := range moves {
			moved[namefold.ModTree(mv.To)] = mv
		}
		out.Places = append(out.Places, PlaceChange{Map: g.mapT, Texts: g.textsT, NewMap: newT, NewTexts: newTexts})
		for _, i := range g.rows {
			changed[i] = true
			out.Missions[i].place(newT, newTexts)
		}
	}
	keys := make([]string, 0, len(moved))
	for k := range moved {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		out.Moves = append(out.Moves, moved[k])
	}
	removed := map[string]bool{}
	for _, mv := range out.Moves {
		from := namefold.ModTree(mv.From)
		if _, rewritten := moved[from]; rewritten || removed[from] {
			continue
		}
		stays := false
		for i, m := range rows {
			stays = stays || !changed[i] && !m.Off && h.reads(m, mv.From)
		}
		if !stays {
			removed[from] = true
			out.Removed = append(out.Removed, mv.From)
		}
	}
	slices.Sort(out.Removed)
	return out
}

// layout is the tree place of a map declared as mapT with texts textsT: the
// new map and texts declarations and the file moves, per language. ok is
// false when a move would write over a held file.
func (h heldFiles) layout(mapT, textsT string, almx bool, claimed []string) (newT, newTexts string, moves []FileMove, ok bool) {
	oldPlace := MissionPlace{Map: path.Clean(mapT)}
	cleanTexts := ""
	if textsT != "" {
		cleanTexts = path.Clean(textsT)
	}
	textsOld := cleanTexts != "" && !oldPlace.holdsPages(cleanTexts)
	ext := path.Ext(mapT)
	if almx {
		ext = ExtendedMapExt
	}
	newT = strings.TrimSuffix(mapT, path.Ext(mapT)) + ext
	if !oldPlace.InTree() {
		stem := MissionStem(mapT)
		p := PlaceMission(stem, false)
		if strings.Contains(mapT, langToken) {
			p = PlaceLanguageMission(langToken, stem, false)
		}
		p = p.free(func(s string) bool {
			return slices.ContainsFunc(layoutLanguages(), func(l string) bool {
				return stemTaken(h.list, missionPath(s, l)) || stemTaken(claimed, missionPath(s, l))
			})
		})
		newT = strings.TrimSuffix(p.Map, path.Ext(p.Map)) + ext
	}
	newPlace := MissionPlace{Map: path.Clean(newT)}
	switch {
	case textsOld && strings.Contains(textsT, langToken) && strings.Contains(newT, langToken):
		newTexts = newPlace.Pages(langToken)
	case textsOld:
	case newPlace.Pages("") == oldPlace.Pages(""):
		newTexts = textsT
	case cleanTexts != "":
		newTexts = newPlace.Pages("") + strings.TrimPrefix(cleanTexts, oldPlace.Pages(""))
		if newTexts == newPlace.Pages("") || newTexts == newPlace.Pages(langToken) {
			newTexts = ""
		}
	}
	ok = true
	seen := map[string]bool{}
	move := func(from, to string) {
		if namefold.ModTree(from) == namefold.ModTree(to) || seen[from+"\x00"+to] {
			return
		}
		seen[from+"\x00"+to] = true
		ok = ok && !h.has(to)
		moves = append(moves, FileMove{From: from, To: to})
	}
	for _, l := range layoutLanguages() {
		op := MissionPlace{Map: path.Clean(missionPath(mapT, l))}
		np := MissionPlace{Map: path.Clean(missionPath(newT, l))}
		if f, held := h.actual(op.Map); held {
			move(f, np.Map)
		}
		if f, held := h.actual(op.Labels()); held {
			move(f, np.Labels())
		}
		for _, f := range h.under(op.Pages(""), false) {
			move(f[0], np.Pages("")+"/"+f[1])
		}
		if !textsOld {
			continue
		}
		texts := path.Clean(missionPath(textsT, l))
		perLanguage := strings.Contains(textsT, langToken)
		target := np.Pages("")
		if perLanguage {
			target = np.Pages(l)
		}
		beside, besideLang, own := h.pageNames(op.Pages("")), h.pageNames(op.Pages(l)), h.pageNames(texts)
		for _, f := range h.under(texts, true) {
			if name := f[1]; missionTextName.MatchString(name) && !beside[name] && !(perLanguage && besideLang[name]) {
				move(f[0], target+"/"+name)
			}
		}
		if locale.Known(path.Base(texts)) {
			for _, f := range h.under(path.Dir(texts), true) {
				if name := f[1]; missionTextName.MatchString(name) && !beside[name] && !(!perLanguage && own[name]) {
					move(f[0], np.Pages("")+"/"+name)
				}
			}
		}
	}
	return newT, newTexts, moves, ok
}

func layoutLanguages() []string {
	var codes []string
	for _, l := range locale.All() {
		codes = append(codes, l.Code)
	}
	return codes
}

// heldFiles is a mod's file list, also keyed by the folded path.
type heldFiles struct {
	list []string
	fold map[string]string
}

func newHeldFiles(held []string) heldFiles {
	h := heldFiles{list: held, fold: map[string]string{}}
	for _, f := range held {
		h.fold[namefold.ModTree(f)] = f
	}
	return h
}

func (h heldFiles) has(f string) bool { _, ok := h.fold[namefold.ModTree(f)]; return ok }

// actual is the held file f as the mod spells it.
func (h heldFiles) actual(f string) (string, bool) {
	held, ok := h.fold[namefold.ModTree(f)]
	return held, ok
}

// under is the held files below folder, or directly in it when direct is
// set, each with its path below folder, sorted.
func (h heldFiles) under(folder string, direct bool) [][2]string {
	prefix := namefold.ModTree(folder) + "/"
	if folder == "." {
		prefix = ""
	}
	var out [][2]string
	for k, f := range h.fold {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		parts := strings.Split(f, "/")
		rest := strings.Join(parts[len(strings.Split(prefix, "/"))-1:], "/")
		if !direct || !strings.Contains(rest, "/") {
			out = append(out, [2]string{f, rest})
		}
	}
	slices.SortFunc(out, func(a, b [2]string) int { return strings.Compare(a[0], b[0]) })
	return out
}

// pageNames is the page names held directly in folder.
func (h heldFiles) pageNames(folder string) map[string]bool {
	names := map[string]bool{}
	for _, f := range h.under(folder, true) {
		if missionTextName.MatchString(f[1]) {
			names[f[1]] = true
		}
	}
	return names
}

// reads reports that row m, as declared, reads file f: its map, labels or a
// page beside it, or a page of its texts folder or that folder's parent.
func (h heldFiles) reads(m Mission, f string) bool {
	f = namefold.ModTree(f)
	in := func(folder string, direct bool) bool {
		prefix := namefold.ModTree(folder) + "/"
		if folder == "." {
			prefix = ""
		}
		rest, ok := strings.CutPrefix(f, prefix)
		return ok && (!direct || !strings.Contains(rest, "/"))
	}
	t, tt := m.declared(m.Map, m.mapTemplate), m.declared(m.Texts, m.textsTemplate)
	for _, l := range layoutLanguages() {
		if m.Map != "" {
			p := MissionPlace{Map: path.Clean(missionPath(t, l))}
			if f == namefold.ModTree(p.Map) || f == namefold.ModTree(p.Labels()) || in(p.Pages(""), false) {
				return true
			}
		}
		if tt != "" {
			texts := path.Clean(missionPath(tt, l))
			if in(texts, true) || locale.Known(path.Base(texts)) && in(path.Dir(texts), true) {
				return true
			}
		}
	}
	return false
}
