package game

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// assembledText is the name of an installed mission text a mod mission takes:
// its event and tip pages. Title and briefing come from the mod's own words.
var assembledText = regexp.MustCompile(`^(event|tips)[0-9]{2}\.txt$`)

// AssembleModMission copies the installed map of mission and its event and
// tip texts out of the install at root into mission/<lang>/map.alm and
// mission/<lang>/text/ of the mod folder out, where a data/missions.toml
// names them. out must be a mod folder (it holds mod.toml) outside the
// install; nothing is written anywhere else. It returns the files written,
// relative to out.
func AssembleModMission(root, lang string, mission int, out string) ([]string, error) {
	if lang == "" || strings.ContainsAny(lang, `/\.:`) {
		return nil, fmt.Errorf("language %q is not a language code", lang)
	}
	address, ok := MissionMap(mission)
	if !ok {
		return nil, fmt.Errorf("mission %d is not a mission number", mission)
	}
	inside, err := pathWithin(root, out)
	if err != nil {
		return nil, err
	}
	if inside {
		return nil, fmt.Errorf("-out %s is inside the install %s; the tool never writes into an install", out, root)
	}
	if st, err := os.Stat(filepath.Join(out, "mod.toml")); err != nil || !st.Mode().IsRegular() {
		return nil, fmt.Errorf("-out %s is not a mod folder: it holds no mod.toml", out)
	}
	a, err := OpenArchives(root)
	if err != nil {
		return nil, err
	}
	m, err := a.Containers.ReadFile(address)
	if err != nil {
		return nil, fmt.Errorf("the install holds no map for mission %d: %w", mission, err)
	}
	files := map[string][]byte{"mission/" + lang + "/map.alm": m}
	prefix := mainPrefix + "text/battle/m" + strconv.Itoa(mission) + "/"
	for _, e := range a.Containers.Entries() {
		name, ok := strings.CutPrefix(strings.ToLower(e.Address), prefix)
		if !ok || !assembledText.MatchString(name) {
			continue
		}
		b, err := a.Containers.ReadFile(e.Address)
		if err != nil {
			return nil, err
		}
		files["mission/"+lang+"/text/"+name] = b
	}
	var written []string
	for rel, b := range files {
		p := filepath.Join(out, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return written, err
		}
		if err := os.WriteFile(p, b, 0o644); err != nil {
			return written, err
		}
		written = append(written, rel)
	}
	return written, nil
}

// pathWithin reports whether path is dir or lies beneath it, after both are
// made absolute and their links resolved as far as they exist.
func pathWithin(dir, path string) (bool, error) {
	resolve := func(p string) (string, error) {
		abs, err := filepath.Abs(p)
		if err != nil {
			return "", err
		}
		if r, err := filepath.EvalSymlinks(abs); err == nil {
			return r, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		return abs, nil
	}
	d, err := resolve(dir)
	if err != nil {
		return false, err
	}
	p, err := resolve(path)
	if err != nil {
		return false, err
	}
	rel, err := filepath.Rel(d, p)
	if err != nil {
		return false, nil
	}
	return rel == "." || !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel), nil
}
