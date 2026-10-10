package mod

import (
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
		if a == "common" || slices.Contains(names, a) {
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

// Order checks a set of resolved mods against the active base and returns it in
// load order: mods whose applies-to is "common" first, then the others, each
// group by id, except that a mod always follows the mods it requires and the
// enabled mods it names in load-after. Every refusal names the mods involved:
// an api this build does not run, a mod that does not apply to the base, a
// conflict, a missing or too old requirement, and a cycle in the order.
func Order(mods []Entry, base string) ([]Entry, error) {
	byID := map[string]Entry{}
	for _, m := range mods {
		byID[m.Manifest.ID] = m
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	after := map[string][]string{}
	for _, id := range ids {
		m := byID[id].Manifest
		if m.API != SupportedAPI {
			return nil, fmt.Errorf("mod %q: api %d is not supported (this game runs api %d)", id, m.API, SupportedAPI)
		}
		if !Applies(m.AppliesTo, base) {
			return nil, fmt.Errorf("mod %q: applies-to %s does not include the active base %s", id, strings.Join(m.AppliesTo, ","), base)
		}
		for _, c := range m.Conflicts {
			if _, ok := byID[c]; ok {
				return nil, fmt.Errorf("mod %q conflicts with the enabled mod %q", id, c)
			}
		}
		for _, r := range m.Requires {
			d, err := ParseDep(r)
			if err != nil {
				return nil, fmt.Errorf("mod %q: requires: %w", id, err)
			}
			dep, ok := byID[d.ID]
			if !ok {
				return nil, fmt.Errorf("mod %q requires %q, which is not enabled", id, r)
			}
			ok, err = d.satisfiedBy(dep.Manifest.Version)
			if err != nil {
				return nil, fmt.Errorf("mod %q: requires %q: %w", id, r, err)
			}
			if !ok {
				return nil, fmt.Errorf("mod %q requires %q but %q %s is enabled", id, r, d.ID, dep.Manifest.Version)
			}
			after[id] = append(after[id], d.ID)
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
	out := make([]Entry, 0, len(ids))
	for len(out) < len(ids) {
		best := ""
		for _, id := range ids {
			if placed[id] {
				continue
			}
			ready := true
			for _, d := range after[id] {
				if !placed[d] {
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
			return nil, fmt.Errorf("mods %s require or load after each other in a cycle", strings.Join(left, ", "))
		}
		placed[best] = true
		out = append(out, byID[best])
	}
	return out, nil
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
