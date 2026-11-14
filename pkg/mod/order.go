package mod

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"againrom/pkg/base"
)

// SupportedAPI is the mod interface version this build runs.
const SupportedAPI = 1

// Applies reports whether a manifest's applies-to list names the active base:
// "common" names every base, and any other entry a name the base answers to,
// its id or its edition's family word (base.AppliesTo).
func Applies(appliesTo []string, id string) bool {
	names := base.AppliesTo(id)
	for _, a := range appliesTo {
		if a == base.AllBases || slices.Contains(names, a) {
			return true
		}
	}
	return false
}

// Dep is one requires entry: a mod id and an optional version constraint.
type Dep struct {
	ID      string
	Op      string // "", ">=", "<=", ">", "<" or "="
	Version string
}

// ParseDep reads "id", "id>=1.0" and the other comparison forms.
func ParseDep(text string) (Dep, error) {
	text = strings.TrimSpace(text)
	i := strings.IndexAny(text, "<>=")
	if i < 0 {
		if !ValidID(text) {
			return Dep{}, fmt.Errorf("%q is not a mod id", text)
		}
		return Dep{ID: text}, nil
	}
	d := Dep{ID: strings.TrimSpace(text[:i])}
	j := i
	for j < len(text) && strings.IndexByte("<>=", text[j]) >= 0 {
		j++
	}
	d.Op, d.Version = text[i:j], strings.TrimSpace(text[j:])
	if d.Op == "==" {
		d.Op = "="
	}
	switch d.Op {
	case ">=", "<=", ">", "<", "=":
	default:
		return Dep{}, fmt.Errorf("%q: comparison %q is not one of >= <= > < =", text, d.Op)
	}
	if !ValidID(d.ID) {
		return Dep{}, fmt.Errorf("%q is not a mod id", d.ID)
	}
	if _, err := versionParts(d.Version); err != nil {
		return Dep{}, fmt.Errorf("%q: %w", text, err)
	}
	return d, nil
}

func versionParts(v string) ([]int, error) {
	if v == "" {
		return nil, fmt.Errorf("version is empty")
	}
	var out []int
	for _, p := range strings.Split(v, ".") {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("version %q is not dotted numbers", v)
		}
		out = append(out, n)
	}
	return out, nil
}

// CompareVersions orders two dotted numeric versions; a missing part is zero.
func CompareVersions(a, b string) (int, error) {
	x, err := versionParts(a)
	if err != nil {
		return 0, err
	}
	y, err := versionParts(b)
	if err != nil {
		return 0, err
	}
	for i := 0; i < len(x) || i < len(y); i++ {
		var p, q int
		if i < len(x) {
			p = x[i]
		}
		if i < len(y) {
			q = y[i]
		}
		if p != q {
			if p < q {
				return -1, nil
			}
			return 1, nil
		}
	}
	return 0, nil
}

func (d Dep) satisfiedBy(version string) (bool, error) {
	if d.Op == "" {
		return true, nil
	}
	c, err := CompareVersions(version, d.Version)
	if err != nil {
		return false, err
	}
	switch d.Op {
	case ">=":
		return c >= 0, nil
	case "<=":
		return c <= 0, nil
	case ">":
		return c > 0, nil
	case "<":
		return c < 0, nil
	}
	return c == 0, nil
}

// RefusalKind names why Order refuses a mod.
type RefusalKind int

const (
	// RefusedAPI: the mod's api is not the one this build runs.
	RefusedAPI RefusalKind = iota + 1
	// RefusedBase: the mod's applies-to does not name the active base.
	RefusedBase
	// RefusedConflict: the mod's conflicts names an enabled mod.
	RefusedConflict
	// RefusedRequirementUnreadable: a requires entry, or the version of the
	// enabled mod it names, cannot be read.
	RefusedRequirementUnreadable
	// RefusedRequirementMissing: a mod the mod requires is not enabled.
	RefusedRequirementMissing
	// RefusedRequirementVersion: the enabled required mod's version does not
	// meet the entry.
	RefusedRequirementVersion
	// RefusedCycle: the mods require or load after each other in a cycle.
	RefusedCycle
)

// Refusal is one reason Order refuses a mod set, as data: the mod it stands on
// and what it names. Error is the game's text for it.
type Refusal struct {
	Kind RefusalKind
	// Mod is the refused mod; for a cycle the first of Cycle.
	Mod string
	// Other is the mod a conflict or a requirement names.
	Other string
	// Requirement is the requires entry as the manifest writes it.
	Requirement string
	// Version is the enabled version of Other when it misses Requirement.
	Version string
	// API is the refused mod's api.
	API int
	// Base is the active base and AppliesTo the refused mod's list.
	Base      string
	AppliesTo []string
	// Cycle lists by id every mod a cycle leaves unplaced; the refusal
	// stands on each of them.
	Cycle []string
	err   error
}

func (r Refusal) Error() string { return r.err.Error() }

// Unwrap is the reading error of an unreadable requirement, else nil.
func (r Refusal) Unwrap() error { return errors.Unwrap(r.err) }

// Names reports whether the refusal stands on mod id: the refused mod, or any
// mod of a cycle.
func (r Refusal) Names(id string) bool {
	return r.Mod == id || r.Kind == RefusedCycle && slices.Contains(r.Cycle, id)
}

// Ordering is what Order makes of a set of mods on a base: the mods in load
// order and every refusal, in the order Order checks them.
type Ordering struct {
	Mods     []Entry
	Refusals []Refusal
}

// Err is the first refusal, nil when the set loads.
func (o Ordering) Err() error {
	if len(o.Refusals) == 0 {
		return nil
	}
	return o.Refusals[0]
}

// Of lists the refusals that stand on mod id.
func (o Ordering) Of(id string) []Refusal {
	var out []Refusal
	for _, r := range o.Refusals {
		if r.Names(id) {
			out = append(out, r)
		}
	}
	return out
}

// Fit lists the refusals a mod raises on base by itself, whatever else is
// enabled: an api this build does not run, then an applies-to that does not
// name the base. Order raises them first for each mod.
func Fit(m Manifest, base string) []Refusal {
	var out []Refusal
	if m.API != SupportedAPI {
		out = append(out, Refusal{Kind: RefusedAPI, Mod: m.ID, API: m.API,
			err: fmt.Errorf("mod %q: api %d is not supported (this game runs api %d)", m.ID, m.API, SupportedAPI)})
	}
	if !Applies(m.AppliesTo, base) {
		out = append(out, Refusal{Kind: RefusedBase, Mod: m.ID, Base: base, AppliesTo: m.AppliesTo,
			err: fmt.Errorf("mod %q: applies-to %s does not include the active base %s", m.ID, strings.Join(m.AppliesTo, ","), base)})
	}
	return out
}

// Order checks a set of resolved mods against the active base and puts it in
// load order: mods whose applies-to is "common" first, then the others, each
// group by id, except that a mod always follows the mods it requires and the
// enabled mods it names in load-after. It checks every mod by id: Fit, then
// each conflict, then each requirement; a cycle comes last, and its mods
// follow the others by group and id. Every refusal names the mods involved,
// and the set loads only when there is none.
func Order(mods []Entry, base string) Ordering {
	byID := map[string]Entry{}
	for _, m := range mods {
		byID[m.Manifest.ID] = m
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var refused []Refusal
	after := map[string][]string{}
	for _, id := range ids {
		m := byID[id].Manifest
		refused = append(refused, Fit(m, base)...)
		for _, c := range m.Conflicts {
			if _, ok := byID[c]; ok {
				refused = append(refused, Refusal{Kind: RefusedConflict, Mod: id, Other: c,
					err: fmt.Errorf("mod %q conflicts with the enabled mod %q", id, c)})
			}
		}
		for _, r := range m.Requires {
			d, err := ParseDep(r)
			if err != nil {
				refused = append(refused, Refusal{Kind: RefusedRequirementUnreadable, Mod: id, Requirement: r,
					err: fmt.Errorf("mod %q: requires: %w", id, err)})
				continue
			}
			dep, ok := byID[d.ID]
			if !ok {
				refused = append(refused, Refusal{Kind: RefusedRequirementMissing, Mod: id, Other: d.ID, Requirement: r,
					err: fmt.Errorf("mod %q requires %q, which is not enabled", id, r)})
				continue
			}
			after[id] = append(after[id], d.ID)
			ok, err = d.satisfiedBy(dep.Manifest.Version)
			switch {
			case err != nil:
				refused = append(refused, Refusal{Kind: RefusedRequirementUnreadable, Mod: id, Other: d.ID, Requirement: r,
					err: fmt.Errorf("mod %q: requires %q: %w", id, r, err)})
			case !ok:
				refused = append(refused, Refusal{Kind: RefusedRequirementVersion, Mod: id, Other: d.ID, Requirement: r, Version: dep.Manifest.Version,
					err: fmt.Errorf("mod %q requires %q but %q %s is enabled", id, r, d.ID, dep.Manifest.Version)})
			}
		}
		for _, l := range m.LoadAfter {
			if _, ok := byID[l]; ok {
				after[id] = append(after[id], l)
			}
		}
	}
	group := func(id string) int {
		if containsString(byID[id].Manifest.AppliesTo, "common") {
			return 0
		}
		return 1
	}
	placed := map[string]bool{}
	cycle := false
	out := make([]Entry, 0, len(ids))
	for len(out) < len(ids) {
		best := ""
		for _, id := range ids {
			if placed[id] {
				continue
			}
			ready := true
			for _, d := range after[id] {
				if !placed[d] && !cycle {
					ready = false
					break
				}
			}
			if !ready {
				continue
			}
			if best == "" || group(id) < group(best) || group(id) == group(best) && id < best {
				best = id
			}
		}
		if best == "" {
			var left []string
			for _, id := range ids {
				if !placed[id] {
					left = append(left, id)
				}
			}
			refused = append(refused, Refusal{Kind: RefusedCycle, Mod: left[0], Cycle: left,
				err: fmt.Errorf("mods %s require or load after each other in a cycle", strings.Join(left, ", "))})
			cycle = true
			continue
		}
		placed[best] = true
		out = append(out, byID[best])
	}
	return Ordering{Mods: out, Refusals: refused}
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
