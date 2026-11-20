package mod

import (
	"bytes"
	"fmt"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"
)

type missionPair struct{ key, value string }

func missionPairs(m Mission) []missionPair {
	p := []missionPair{{"key", strconv.Quote(m.Key)}, {"number", strconv.Itoa(m.Number)}, {"map", strconv.Quote(m.declared(m.Map, m.mapTemplate))}}
	if m.Kind == MissionReplace {
		p = append([]missionPair{{"kind", strconv.Quote(m.Kind)}}, p...)
	}
	if m.Kind == MissionCampaign {
		p = append([]missionPair{{"kind", strconv.Quote(m.Kind)}}, p...)
		if m.Map == "" {
			p = p[:len(p)-1]
		}
	}
	if m.AutoGetMission != nil {
		p = append(p, missionPair{"AutoGetMission", strconv.Itoa(*m.AutoGetMission)})
	}
	if m.Payment != nil {
		p = append(p, missionPair{"Payment", strconv.Itoa(*m.Payment)})
	}
	if m.Off {
		return []missionPair{{"number", strconv.Itoa(m.Number)}, {"off", "true"}, {"kind", strconv.Quote(m.Kind)}}
	}
	for _, v := range []missionPair{{"texts", m.declared(m.Texts, m.textsTemplate)}, {"title", m.TitleKey}, {"briefing", m.BriefKey}} {
		if v.value != "" {
			p = append(p, missionPair{v.key, writeString(v.value)})
		}
	}
	if !m.Campaign() || m.GuestParty() || m.MainParty {
		if !m.Campaign() {
			p = append(p, missionPair{"after", strconv.Itoa(m.After)})
		}
		for _, v := range []missionPair{{"class", m.Class}, {"sex", m.Sex}} {
			if v.value != "" {
				p = append(p, missionPair{v.key, strconv.Quote(v.value)})
			}
		}
		if m.MainParty {
			p = append(p, missionPair{"party", strconv.Quote("main")})
		} else if len(m.Party) != 0 {
			var members []string
			for _, g := range m.Party {
				fields := []string{"character = " + strconv.Quote(g.Character)}
				if g.Hero {
					fields = []string{"hero = true"}
				}
				if g.Level != 0 {
					fields = append(fields, "level = "+strconv.Itoa(g.Level))
				}
				for _, f := range []struct {
					key   string
					value *string
				}{{"weapon", g.Weapon}, {"shield", g.Shield}} {
					if f.value != nil {
						fields = append(fields, f.key+" = "+strconv.Quote(*f.value))
					}
				}
				if g.Armour != nil {
					var a []string
					for _, item := range g.Armour {
						a = append(a, strconv.Quote(item))
					}
					fields = append(fields, "armour = ["+strings.Join(a, ", ")+"]")
				}
				if g.Skills != 0 {
					fields = append(fields, "skills = "+strconv.Itoa(g.Skills))
				}
				if g.Spells != "" {
					fields = append(fields, "spells = "+strconv.Quote(g.Spells))
				}
				members = append(members, "{ "+strings.Join(fields, ", ")+" }")
			}
			p = append(p, missionPair{"party", "[" + strings.Join(members, ", ") + "]"})
		} else {
			p = append(p, missionPair{"guest", strconv.Itoa(m.Guest)}, missionPair{"chapter", strconv.Itoa(m.Chapter)})
			if m.Carry != "" && m.Carry != CarryJoin {
				p = append(p, missionPair{"carry", strconv.Quote(m.Carry)})
			}
		}
		if m.Gold != 0 {
			p = append(p, missionPair{"gold", strconv.Itoa(m.Gold)})
		}
	}
	return p
}

// declared is the path a row writes for value: its declared template when
// the template still reads as value in the row's language, else value.
func (m Mission) declared(value, template string) string {
	if template != "" && path.Clean(missionPath(template, m.language)) == value {
		return template
	}
	return value
}

func missionIdentity(m Mission) string {
	if m.Off {
		return fmt.Sprintf("off:%d", m.Number)
	}
	return m.Key
}

// WriteMissions is the missions file source with the rows before made the
// rows after: a kept row keeps its lines but for the changed pairs, a new
// row is appended. A mission the file form declares is its map file and
// takes no row, so it is left out of both.
func WriteMissions(source []byte, before, after []Mission) ([]byte, error) {
	before, after = missionRows(before), missionRows(after)
	sourced := func(list []Mission) []sourceRow {
		out := make([]sourceRow, len(list))
		for i, m := range list {
			out[i] = sourceRow{table: "mission", identity: missionIdentity(m), pairs: missionPairs(m)}
		}
		return out
	}
	return writeRows(source, sourced(before), sourced(after), func(key string) string {
		if strings.Contains("|key|number|map|texts|after|guest|chapter|", "|"+key+"|") {
			key += strings.Repeat(" ", 7-len(key))
		}
		return key
	})
}

// sourceRow is one table of a row file as the row writer sees it: its
// table name, the identity that names it across an edit and the pairs it
// writes, in order. A plain row is a table of its own, [name], not a row of
// the array [[name]].
type sourceRow struct {
	table, identity string
	plain           bool
	pairs           []missionPair
}

// writeRows is the one row writer of the data files: source with the rows
// before made the rows after. A kept row keeps its lines but for the
// changed pairs, a row after does not hold leaves with its own lines, and a new
// row is appended with its keys as pad writes them. Every table of source
// is a row of before, in order.
func writeRows(source []byte, before, after []sourceRow, pad func(string) string) ([]byte, error) {
	tables, err := parseTOMLWith(source, tomlOptions{arrays: true})
	if err != nil {
		return nil, err
	}
	var records []tomlTable
	for _, t := range tables {
		if t.name != "" {
			records = append(records, t)
		}
	}
	if len(records) != len(before) {
		return nil, fmt.Errorf("row source has %d records, expected %d", len(records), len(before))
	}
	lines := strings.SplitAfter(string(source), "\n")
	prefix := string(source)
	if len(records) != 0 {
		prefix = strings.Join(lines[:records[0].line-1], "")
	}
	// A record owns its header to its last line that is neither blank nor
	// a comment; the blank and comment lines after it head the next record
	// and stay when the record leaves.
	own, tails := make([]int, len(records)), make([]string, len(records))
	for i, t := range records {
		end := len(lines)
		if i+1 < len(records) {
			end = records[i+1].line - 1
		}
		own[i] = end
		for own[i] > t.line {
			s := strings.TrimSpace(lines[own[i]-1])
			if s != "" && !strings.HasPrefix(s, "#") {
				break
			}
			own[i]--
		}
		tails[i] = strings.Join(lines[own[i]:end], "")
	}
	kept, lastKept := make([]bool, len(before)), -1
	for j, m := range after {
		if i := slices.IndexFunc(before, func(old sourceRow) bool { return old.identity == m.identity }); i >= 0 {
			kept[i], lastKept = true, j
		}
	}
	var out strings.Builder
	out.WriteString(prefix)
	tailed := make([]bool, len(before))
	leftTails := func(upTo int) {
		for i := 0; i < upTo; i++ {
			if !kept[i] && !tailed[i] {
				tailed[i] = true
				out.WriteString(tails[i])
			}
		}
	}
	for j, m := range after {
		index := -1
		for i, old := range before {
			if old.identity == m.identity {
				index = i
				break
			}
		}
		if index < 0 {
			if j > lastKept {
				leftTails(len(before))
			}
			if out.Len() != 0 && !strings.HasSuffix(out.String(), "\n") {
				out.WriteByte('\n')
			}
			header := "[[%s]]\n"
			if m.plain {
				header = "[%s]\n"
			}
			fmt.Fprintf(&out, header, m.table)
			for _, p := range m.pairs {
				fmt.Fprintf(&out, "%s = %s\n", pad(p.key), p.value)
			}
			continue
		}
		leftTails(index)
		t := records[index]
		tailed[index] = true
		block := append([]string(nil), lines[t.line-1:own[index]]...)
		oldPairs, newPairs := before[index].pairs, m.pairs
		oldValues, newValues := map[string]string{}, map[string]string{}
		for _, p := range oldPairs {
			oldValues[p.key] = p.value
		}
		for _, p := range newPairs {
			newValues[p.key] = p.value
		}
		seen := map[string]bool{}
		for _, p := range t.pairs {
			seen[p.key] = true
			if oldValues[p.key] == newValues[p.key] {
				continue
			}
			line := p.line - t.line
			if value, ok := newValues[p.key]; ok {
				block[line], err = rewritePair(block[line], value)
			} else {
				block[line] = ""
			}
			if err != nil {
				return nil, err
			}
		}
		text := strings.Join(block, "") + tails[index]
		for _, p := range newPairs {
			if !seen[p.key] && oldValues[p.key] != p.value {
				if !strings.HasSuffix(text, "\n") {
					text += "\n"
				}
				text += p.key + " = " + p.value + "\n"
			}
		}
		out.WriteString(text)
		if len(after) > 1 && !strings.HasSuffix(text, "\n") {
			out.WriteByte('\n')
		}
	}
	leftTails(len(before))
	return []byte(out.String()), nil
}

func rewritePair(line, value string) (string, error) {
	eq := strings.IndexByte(line, '=')
	if eq < 0 {
		return "", fmt.Errorf("missing assignment")
	}
	start := eq + 1
	for start < len(line) && (line[start] == ' ' || line[start] == '\t') {
		start++
	}
	_, rest, err := parseTOMLValue(strings.TrimSpace(line[start:]))
	if err != nil {
		return "", err
	}
	end := len(strings.TrimRight(line, "\r\n"))
	if rest != "" {
		end = start + len(strings.TrimSpace(line[start:])) - len(rest)
	}
	for end > start && (line[end-1] == ' ' || line[end-1] == '\t') {
		end--
	}
	return line[:start] + value + line[end:], nil
}

// ValueEdit is the string value that the pair at Line of a row file takes.
type ValueEdit struct {
	Line  int
	Value string
}

// RewriteValues is source with the string value of the pair at each edit's
// line made the edit's value, by the row writer's own pair rewrite; every
// other byte stays. The result is read back as a row file.
func RewriteValues(source []byte, edits []ValueEdit) ([]byte, error) {
	lines := strings.SplitAfter(string(source), "\n")
	for _, e := range edits {
		if e.Line < 1 || e.Line > len(lines) {
			return nil, fmt.Errorf("line %d is not in the file", e.Line)
		}
		line, err := rewritePair(lines[e.Line-1], writeString(e.Value))
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", e.Line, err)
		}
		lines[e.Line-1] = line
	}
	out := []byte(strings.Join(lines, ""))
	if _, err := parseTOMLWith(out, tomlOptions{arrays: true, quotedKeys: true}); err != nil {
		return nil, err
	}
	return out, nil
}

func WriteManifestTitle(source []byte, title string) ([]byte, error) {
	out, err := WriteManifest(source, map[string]any{"title": title})
	if err == nil {
		_, err = ParseManifest(out)
	}
	return out, err
}

func CloneMission(m Mission) Mission {
	if m.AutoGetMission != nil {
		v := *m.AutoGetMission
		m.AutoGetMission = &v
	}
	if m.Payment != nil {
		v := *m.Payment
		m.Payment = &v
	}
	if m.Party != nil {
		m.Party = append([]GuestMember{}, m.Party...)
		for i := range m.Party {
			g := &m.Party[i]
			if g.Weapon != nil {
				v := *g.Weapon
				g.Weapon = &v
			}
			if g.Shield != nil {
				v := *g.Shield
				g.Shield = &v
			}
			if g.Armour != nil {
				g.Armour = append([]string{}, g.Armour...)
			}
		}
	}
	m.MapBytes = bytes.Clone(m.MapBytes)
	if m.TextFiles != nil {
		files := map[string][]byte{}
		for k, v := range m.TextFiles {
			files[k] = bytes.Clone(v)
		}
		m.TextFiles = files
	}
	return m
}

// numberRows completes the identity of rows whose records may repeat: a row
// that came from a source line is named by that line, so a delete of an
// earlier repeat leaves the later one its own; a new row (line 0) takes the
// next repeat number of its identity.
func numberRows(rows []sourceRow, lines []int) {
	seen := map[string]int{}
	for i := range rows {
		if lines[i] > 0 {
			rows[i].identity += "\x00@" + strconv.Itoa(lines[i])
			continue
		}
		seen[rows[i].identity]++
		rows[i].identity += "\x00" + strconv.Itoa(seen[rows[i].identity])
	}
}

func pairMap(r sourceRow) map[string]string {
	m := map[string]string{}
	for _, p := range r.pairs {
		m[p.key] = p.value
	}
	return m
}

// sameRows reports whether a and b are the same rows in the same order by
// table and pairs; the identity is of one reading of the source and is left out.
func sameRows(a, b []sourceRow) bool {
	return slices.EqualFunc(a, b, func(x, y sourceRow) bool {
		return x.table == y.table && maps.Equal(pairMap(x), pairMap(y))
	})
}

// DataBefore is the records a writer diffs its after records against: the
// missions and the offer rows the file held when it was opened.
type DataBefore struct {
	Missions []Mission
	Offers   []OfferRecord
	Towns    []TownRecord
}

// writeMissionRows is the missions kind's writer over d.Before.
func writeMissionRows(_ string, source []byte, d Data) ([]byte, error) {
	return WriteMissions(source, d.Before.Missions, d.Missions.Missions)
}

// writeTownRows is the towns kind's writer over d.Before.
func writeTownRows(_ string, source []byte, d Data) ([]byte, error) {
	return WriteTownData(source, TownData{Records: d.Before.Towns, Offers: d.Before.Offers}, d.Towns)
}
