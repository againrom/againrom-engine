// Package storydoc checks a story document against docs/STORY-FORMAT.md.
package storydoc

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	MaxBytes     = 6144
	MaxCodeRows  = 8
	MaxProofRows = 12
	MaxLineRunes = 120
	MaxParaLines = 5
)

var Sections = []string{"Result", "Authority", "Rules", "Code", "Proof", "Open debt"}

var (
	ruleLine = regexp.MustCompile(`^(\d+)\. \S`)
	number   = regexp.MustCompile(`\d+`)
)

func Check(text string) []string {
	var bad []string
	add := func(f string, a ...any) { bad = append(bad, fmt.Sprintf(f, a...)) }

	if n := len(text); n > MaxBytes {
		add("%d bytes, over %d", n, MaxBytes)
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")

	var heads []string
	body := map[string][]string{}
	cur, title, fence, para := "", false, false, 0
	endPara := func(at int) {
		if para > MaxParaLines {
			add("line %d: a paragraph of %d lines, over %d", at, para, MaxParaLines)
		}
		para = 0
	}
	for i, l := range lines {
		n := i + 1
		if strings.HasPrefix(l, "```") {
			fence = !fence
			endPara(n)
			continue
		}
		if fence {
			continue
		}
		if !title && strings.TrimSpace(l) != "" {
			title = true
			if !strings.HasPrefix(l, "# ") {
				add("line %d: the first line is not a `# ` title", n)
			}
			continue
		}
		if strings.HasPrefix(l, "## ") {
			endPara(n)
			cur = strings.TrimSpace(l[3:])
			heads = append(heads, cur)
			continue
		}
		if strings.HasPrefix(l, "#") {
			add("line %d: a heading other than the six sections", n)
		}
		if cur != "" {
			body[cur] = append(body[cur], l)
		}
		t := strings.TrimSpace(l)
		table := strings.HasPrefix(t, "|")
		if !table && utf8.RuneCountInString(l) > MaxLineRunes {
			add("line %d: %d characters, over %d", n, utf8.RuneCountInString(l), MaxLineRunes)
		}
		switch {
		case t == "" || table:
			endPara(n)
		case strings.HasPrefix(t, "- ") || ruleLine.MatchString(t):
			endPara(n)
		case strings.HasPrefix(l, "  "):
		default:
			para++
		}
	}
	endPara(len(lines))

	if strings.Join(heads, "|") != strings.Join(Sections, "|") {
		add("sections %q, not %q", heads, Sections)
	}
	for _, l := range body["Result"] {
		if strings.Contains(l, "`") {
			add("Result holds a backtick: %q", strings.TrimSpace(l))
			break
		}
	}

	rules := 0
	for _, l := range body["Rules"] {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		m := ruleLine.FindStringSubmatch(t)
		if m == nil {
			add("Rules: not a numbered one-line rule: %q", t)
			continue
		}
		rules++
		if k, _ := strconv.Atoi(m[1]); k != rules {
			add("Rules: rule %d is numbered %d", rules, k)
		}
	}
	if rules == 0 {
		add("Rules holds no rule")
	}

	if n := len(tableRows(body["Code"])); n > MaxCodeRows {
		add("Code has %d rows, over %d", n, MaxCodeRows)
	}
	proof := tableRows(body["Proof"])
	if n := len(proof); n > MaxProofRows {
		add("Proof has %d rows, over %d", n, MaxProofRows)
	}
	proved := map[int]bool{}
	for _, cells := range proof {
		if len(cells) < 2 {
			continue
		}
		for _, d := range number.FindAllString(cells[len(cells)-1], -1) {
			k, _ := strconv.Atoi(d)
			proved[k] = true
		}
	}
	for k := 1; k <= rules; k++ {
		if !proved[k] {
			add("rule %d has no Proof row", k)
		}
	}
	return bad
}

func tableRows(lines []string) [][]string {
	var rows [][]string
	inTable := false
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if !strings.HasPrefix(t, "|") {
			inTable = false
			continue
		}
		if !inTable {
			inTable = true
			continue
		}
		if strings.Trim(t, "|-: ") == "" {
			continue
		}
		var cells []string
		for _, c := range strings.Split(strings.Trim(t, "|"), "|") {
			cells = append(cells, strings.TrimSpace(c))
		}
		rows = append(rows, cells)
	}
	return rows
}
