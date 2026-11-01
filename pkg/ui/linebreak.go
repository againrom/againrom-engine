package ui

import (
	"strings"
	"unicode/utf8"

	"againrom/pkg/render/text"
)

// The line breaker. Every multi-line text in the UI is broken here: a surface
// is a lineRule row (what ends a line, what is trimmed, what an unbreakable
// word does, how paragraphs are cut) asked with a lineBox (the measure, the
// width, the indents, the line cap) over styled runs. No other file measures a
// word, cuts at a space or splits a text into words.

// breakRule is how a line ends.
type breakRule uint8

const (
	// breakWord ends a line at a break byte when one fits and cuts inside a
	// word only as the row's overflow says.
	breakWord breakRule = iota
	// breakChar ends a line wherever the next unit would not fit.
	breakChar
	// breakBalanced splits a text that does not fit one row into exactly two
	// rows of given widths at a space, the wider row as narrow as it can be.
	breakBalanced
)

type breakUnit uint8

const (
	unitByte breakUnit = iota
	unitRune
)

type breakParas uint8

const (
	// parasNone is one paragraph holding the whole text.
	parasNone breakParas = iota
	// parasLF splits at every "\n".
	parasLF
	// parasCRLF splits at every "\n" after "\r\n" became "\n".
	parasCRLF
	// parasOriginal splits at "\r\n", trims each paragraph and keeps one with a
	// bare "\r" in it whole.
	parasOriginal
)

type breakCollapse uint8

const (
	collapseNone breakCollapse = iota
	// collapseFields makes every run of white space one space.
	collapseFields
	// collapseSpaceTab makes every run of spaces and tabs one space.
	collapseSpaceTab
)

type breakOverflow uint8

const (
	// overflowCut breaks a word that is wider than a line where it stops fitting.
	overflowCut breakOverflow = iota
	// overflowKeep leaves such a word alone on its line.
	overflowKeep
	// overflowDump leaves such a word alone on its line, and puts the rest of a
	// paragraph whose first word fits alone but not with the space after it on
	// one line.
	overflowDump
)

// lineRule is one surface's row.
type lineRule struct {
	Rule     breakRule
	Unit     breakUnit
	Breaks   string // the bytes a word break may fall on
	Paras    breakParas
	Collapse breakCollapse
	Overflow breakOverflow

	// RawWhole emits a paragraph that fits whole as it stands, before it is
	// collapsed.
	RawWhole bool
	// Hyphens reads "letter-, space, letter" as a discretionary hyphen: a place
	// a line may end, drawing the hyphen, and nothing when it does not.
	Hyphens bool
	// RunBreaks lets a line end where one run meets the next.
	RunBreaks bool
	// FoldNewlines turns every "\n" into a space before breaking.
	FoldNewlines bool
	// TrimCR drops the "\r" that ends a paragraph.
	TrimCR bool
	// Scrub makes every byte that is not UTF-8 one U+FFFD before breaking.
	Scrub bool

	// Strict asks a line to measure under the width, not up to it.
	Strict bool
	// Edge ends a line at a break byte that is the first unit not to fit.
	Edge bool
	// Lossless keeps the break byte at the end of the line it ends.
	Lossless bool
	// Trim drops the break bytes that end a line cut before the text's end;
	// TrimLast drops them from the last line.
	Trim, TrimLast bool
	// SkipHead and SkipLead drop break bytes from the start of a paragraph and
	// from the start of every line after a cut.
	SkipHead, SkipLead bool
	// SkipBlank drops a paragraph with nothing in it; otherwise it is an empty line.
	SkipBlank bool
	// EmptyLine answers a text with no line by one empty line.
	EmptyLine bool
	// NeedWidth makes a width of no pixels or less no room at all.
	NeedWidth bool
	// Whole answers no measure, or no room, by the text unbroken.
	Whole bool
	// Close is what a line that ends its paragraph by fitting carries after its text.
	Close string
}

// lineBox is what one asking measures with.
type lineBox struct {
	Measure     func(string) int // nil: no measure
	Width       int
	FirstIndent int // narrows the first line of every paragraph
	Indent      int // narrows, and marks, every line after a paragraph's first
	MaxLines    int // 0: no cap
	Rows        []int
}

// brokenLine is one line: its ink runs, its indent, where it stands in its paragraph and
// the span of the asked text it covers.
type brokenLine struct {
	Runs     []inkRun
	Indent   int
	First    bool // the first line of its paragraph
	Closed   bool // the line that ended its paragraph by fitting
	Last     bool // the last line of its paragraph, by fitting or not
	From, To int  // byte offsets into the asked text (rules that rewrite no text)
}

// Text is the line's characters.
func (l brokenLine) Text() string {
	if len(l.Runs) == 1 {
		return l.Runs[0].Text
	}
	var b strings.Builder
	for _, r := range l.Runs {
		b.WriteString(r.Text)
	}
	return b.String()
}

// Measures. A nil font gives a nil measure, which a rule reads as no room.

func advanceOf(f *text.Font) func(string) int {
	if f == nil {
		return nil
	}
	return f.Advance
}

// inkedAdvanceOf is advanceOf for a font that has records.
func inkedAdvanceOf(f *text.Font) func(string) int {
	if f == nil || len(f.Glyphs) == 0 {
		return nil
	}
	return f.Advance
}

func boxOf(f *text.Font) func(string) int {
	return func(s string) int { w, _ := f.Measure(s); return w }
}

func reachOf(f *text.Font) func(string) int {
	return func(s string) int { w, _ := f.Measure(s); return max(w, f.Advance(s)) }
}

func runeCount(s string) int { return len([]rune(s)) }

// The rows.
var (
	// shopTipRule: shop tips, objectives and the like, a word never cut.
	shopTipRule = lineRule{Breaks: " ", Paras: parasCRLF, Collapse: collapseFields, Overflow: overflowKeep,
		Edge: true, SkipHead: true, SkipLead: true}
	// documentRule: the same, a blank paragraph dropped.
	documentRule = lineRule{Breaks: " ", Paras: parasCRLF, Collapse: collapseFields, Overflow: overflowKeep,
		Edge: true, SkipHead: true, SkipLead: true, SkipBlank: true}
	// gameMenuRule: counted in characters.
	gameMenuRule = lineRule{Unit: unitRune, Breaks: " ", Collapse: collapseFields, Overflow: overflowKeep,
		Edge: true, SkipHead: true, SkipLead: true}
	// editorRule: nothing lost, a space may start a line.
	editorRule = lineRule{Breaks: " ", FoldNewlines: true, Lossless: true, Edge: true, EmptyLine: true, Whole: true}
	// messageRule: the map message line.
	messageRule = lineRule{Breaks: " ", Strict: true, Edge: true, Overflow: overflowKeep, NeedWidth: true, Whole: true}
	// modScreenRule: mod screens.
	modScreenRule = lineRule{Unit: unitRune, Breaks: " ", Collapse: collapseFields, Edge: true, SkipHead: true, SkipLead: true}
	// noticeRule: notices, a word wider than the line cut.
	noticeRule = lineRule{Breaks: " \t\r\n", Edge: true, Trim: true, SkipHead: true, SkipLead: true, NeedWidth: true}
	// dialogueRule: the original's dialogue paragraphs.
	dialogueRule = lineRule{Breaks: " ", Paras: parasOriginal, Collapse: collapseSpaceTab, RawWhole: true, Strict: true,
		Lossless: true, Overflow: overflowDump, Close: " \r", NeedWidth: true}
	// tipTextRule: the same with discretionary hyphens.
	tipTextRule = lineRule{Breaks: " ", Paras: parasOriginal, Collapse: collapseSpaceTab, RawWhole: true, Strict: true,
		Lossless: true, Overflow: overflowDump, Close: " \r", NeedWidth: true, Hyphens: true}
	// saveRule: a path, every character kept, cut where it stops fitting.
	saveRule = lineRule{Rule: breakChar, Unit: unitRune, Paras: parasLF, Scrub: true}
	// tooltipRule: hover help.
	tooltipRule = lineRule{Breaks: " ", Trim: true, TrimLast: true, SkipLead: true, EmptyLine: true, NeedWidth: true}
	// noticeRowRule: the notice window's rows, in ink runs, continuation lines indented.
	noticeRowRule = lineRule{Breaks: " ", RunBreaks: true, Edge: true, Trim: true, TrimLast: true, SkipHead: true,
		SkipLead: true, EmptyLine: true}
	// textSpanRule: the editor's text input, as spans of the text.
	textSpanRule = lineRule{Unit: unitRune, Scrub: true, Breaks: " ", Paras: parasLF, TrimCR: true, Lossless: true}
	// cardNameRule: a character card's name in two rows.
	cardNameRule = lineRule{Rule: breakBalanced, Breaks: " "}
)

// lines breaks runs by the rule. The runs' ink is kept: a line holds the slices
// of the runs it covers. A rule that rewrites the text (collapse, folding,
// scrubbing) takes one run.
func (r lineRule) lines(box lineBox, runs ...inkRun) []brokenLine {
	var src strings.Builder
	var ends []int
	for i, run := range runs {
		src.WriteString(run.Text)
		if i < len(runs)-1 && run.Text != "" {
			ends = append(ends, src.Len())
		}
	}
	txt := src.String()
	if r.Rule == breakBalanced {
		return r.balanced(box, txt)
	}
	if box.Measure == nil || r.NeedWidth && box.Width <= 0 {
		if r.Whole && (txt != "" || r.EmptyLine) {
			return []brokenLine{{Runs: nonEmptyRuns(runs), To: len(txt)}}
		}
		return nil
	}
	ink := false
	if len(runs) > 0 {
		ink = runs[0].Alert
	}
	if r.FoldNewlines {
		txt = strings.ReplaceAll(txt, "\n", " ")
	}
	if r.Paras == parasCRLF {
		txt = strings.ReplaceAll(txt, "\r\n", "\n")
	}
	if r.Scrub {
		txt = scrubUTF8(txt)
	}
	gated := r.Hyphens && softGate(txt)
	var out []brokenLine
	for _, p := range r.paragraphs(txt, gated) {
		out = append(out, r.paragraph(box, p, gated, runs, ends, ink)...)
	}
	if len(out) == 0 && r.EmptyLine {
		out = []brokenLine{{First: true}}
	}
	if box.MaxLines > 0 && len(out) > box.MaxLines {
		out = out[:box.MaxLines]
	}
	return out
}

// plain is the lines of one unstyled text, as strings.
func (r lineRule) plain(box lineBox, s string) []string {
	lines := r.lines(box, inkRun{Text: s})
	if lines == nil {
		return nil
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.Text()
	}
	return out
}

func nonEmptyRuns(runs []inkRun) []inkRun {
	var out []inkRun
	for _, r := range runs {
		if r.Text != "" {
			out = append(out, r)
		}
	}
	return out
}

func scrubUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		b.WriteRune(r)
	}
	return b.String()
}

const breakTrimBytes = " \t\r\n\v\f"

type breakPara struct {
	text    string
	at      int // offset of text in the asked text, when it is a slice of it
	literal bool
}

// paragraphs cuts txt. rawFirst keeps the first paragraph of a text with
// several as it stands, leading white space included.
func (r lineRule) paragraphs(txt string, rawFirst bool) []breakPara {
	switch r.Paras {
	case parasLF, parasCRLF:
		var out []breakPara
		at := 0
		for _, part := range strings.Split(txt, "\n") {
			p := breakPara{text: part, at: at}
			if r.TrimCR {
				p.text = strings.TrimSuffix(p.text, "\r")
			}
			out = append(out, p)
			at += len(part) + 1
		}
		return out
	case parasOriginal:
		var out []breakPara
		for rest := txt; rest != ""; {
			end := strings.Index(rest, "\r\n")
			if end < 0 {
				out = append(out, breakPara{text: rest})
				break
			}
			piece := rest[:end]
			if !(rawFirst && len(out) == 0) {
				piece = strings.TrimLeft(piece, breakTrimBytes)
			}
			out = append(out, breakPara{text: piece})
			rest = strings.TrimLeft(rest[end+2:], breakTrimBytes)
		}
		return out
	}
	return []breakPara{{text: txt}}
}

func (r lineRule) collapse(s string) string {
	switch r.Collapse {
	case collapseFields:
		return strings.Join(strings.Fields(s), " ")
	case collapseSpaceTab:
		return strings.Join(strings.FieldsFunc(s, func(c rune) bool { return c == ' ' || c == '\t' }), " ")
	}
	return s
}

// paragraph breaks one paragraph.
func (r lineRule) paragraph(box lineBox, p breakPara, gated bool, runs []inkRun, ends []int, ink bool) []brokenLine {
	single := func(s string, closed bool) brokenLine {
		l := brokenLine{First: true, Closed: closed, From: p.at, To: p.at + len(s)}
		if closed {
			s += r.Close
		}
		if s != "" {
			l.Runs = []inkRun{{Text: s, Alert: ink}}
		}
		return l
	}
	t := p.text
	var hyph []int
	virtual := false
	raw := ""
	soft := false
	if r.Paras == parasOriginal {
		trimmed := strings.Trim(t, breakTrimBytes)
		if gated {
			var words []string
			words, hyph = softWords(trimmed)
			if soft = len(hyph) > 0; soft {
				t = strings.Join(words, " ")
			}
		}
		if !soft {
			// A tip text with discretionary hyphens takes a paragraph of white
			// space for blank before it looks for a carriage return.
			if strings.Contains(t, "\r") && !(gated && trimmed == "") {
				return []brokenLine{single(t, false)}
			}
			t = trimmed
		}
	}
	if !soft {
		if r.RawWhole && t != "" {
			if !r.over(box.Measure(t), box.Width) {
				return []brokenLine{single(t, true)}
			}
			raw = t
			t = r.collapse(t) + " "
			virtual = true
		} else if r.Collapse != collapseNone {
			t = r.collapse(t)
		}
	}
	if t == "" && r.Paras != parasNone {
		if r.SkipBlank {
			return nil
		}
		return []brokenLine{single("", false)}
	}
	sc := scan{r: r, box: box, t: t, hyph: hyph, virtual: virtual, raw: raw}
	if r.RunBreaks {
		sc.bounds = ends
	}
	spans := sc.run()
	out := make([]brokenLine, len(spans))
	for i, sp := range spans {
		l := brokenLine{Indent: sp.indent, First: sp.first, Closed: sp.closed, Last: sp.last, From: p.at + sp.a, To: p.at + sp.b}
		if len(runs) > 1 {
			l.Runs = sliceRuns(runs, sp.a, sp.b)
		} else {
			s := t[sp.a:sp.b] + sp.tail
			if sp.text != "" {
				s = sp.text
			}
			if sp.closed {
				s += r.Close
			}
			if s != "" {
				l.Runs = []inkRun{{Text: s, Alert: ink}}
			}
		}
		out[i] = l
	}
	return out
}

// sliceRuns is the part of runs that covers bytes a..b of their text, equal
// neighbours joined.
func sliceRuns(runs []inkRun, a, b int) []inkRun {
	var out []inkRun
	at := 0
	for _, run := range runs {
		lo, hi := max(a, at), min(b, at+len(run.Text))
		at += len(run.Text)
		if lo >= hi {
			continue
		}
		piece := run.Text[lo-(at-len(run.Text)) : hi-(at-len(run.Text))]
		if n := len(out); n > 0 && out[n-1].Alert == run.Alert {
			out[n-1].Text += piece
		} else {
			out = append(out, inkRun{Text: piece, Alert: run.Alert})
		}
	}
	return out
}

type span struct {
	a, b                int
	indent              int
	first, closed, last bool
	tail                string
	text                string // when not t[a:b]+tail
}

// scan is the walk over one paragraph.
type scan struct {
	r       lineRule
	box     lineBox
	t       string
	bounds  []int  // run joins, ascending
	hyph    []int  // discretionary hyphen places in t
	virtual bool   // t ends in a space the text does not have
	raw     string // the paragraph before it was collapsed, when virtual
}

func (r lineRule) over(measure, limit int) bool {
	if r.Strict {
		return measure >= limit
	}
	return measure > limit
}

func (r lineRule) isBreak(b byte) bool {
	return r.Rule != breakChar && strings.IndexByte(r.Breaks, b) >= 0
}

func (s *scan) step(j int) int {
	if s.r.Unit == unitRune {
		_, n := utf8.DecodeRuneInString(s.t[j:])
		return j + n
	}
	return j + 1
}

func (s *scan) skip(i int) int {
	for i < len(s.t) && s.r.isBreak(s.t[i]) {
		i++
	}
	return i
}

func (s *scan) isBound(at int) bool {
	for _, b := range s.bounds {
		if b == at {
			return true
		}
	}
	return false
}

func (s *scan) run() []span {
	r := s.r
	var out []span
	i := 0
	if r.SkipHead {
		i = s.skip(0)
	}
	for first := true; i < len(s.t); first = false {
		t := s.t
		limit, indent := s.box.Width, 0
		if first {
			if s.box.FirstIndent > 0 {
				limit = max(0, limit-s.box.FirstIndent)
			}
		} else {
			limit -= s.box.Indent
			indent = s.box.Indent
		}
		// f ends the longest prefix that fits, one unit at least.
		// A rule that cuts words takes one unit at least, or it would not advance.
		forced := r.Rule == breakChar || r.Overflow == overflowCut
		f := i
		for j := s.step(i); ; j = s.step(j) {
			if (f > i || !forced) && r.over(s.box.Measure(t[i:j]), limit) {
				break
			}
			f = j
			if j >= len(t) {
				break
			}
		}
		if f >= len(t) {
			end := len(t)
			if s.virtual {
				out = append(out, span{a: i, b: end, indent: indent, first: first})
				break
			}
			if r.TrimLast {
				for end > i && r.isBreak(t[end-1]) {
					end--
				}
			}
			out = append(out, span{a: i, b: end, indent: indent, first: first, closed: true})
			break
		}
		// The last place a line may end inside what fits.
		k, b := -1, -1
		if r.Rule == breakWord {
			for j := f - 1; j >= i; j-- {
				if r.isBreak(t[j]) {
					k = j
					break
				}
			}
			for _, e := range s.bounds {
				if e > i && e < f {
					b = e
				}
			}
		}
		if len(s.hyph) > 0 {
			ws := i
			if k >= i {
				ws = k + 1
			}
			we := s.wordEnd(ws)
			best := -1
			for _, h := range s.hyph {
				if h > ws && h < we && !r.over(s.box.Measure(t[i:h]+"- "), limit) {
					best = h
				}
			}
			if best > 0 {
				out = append(out, span{a: i, b: best, indent: indent, first: first, tail: "- "})
				i = best
				s.virtual = false
				continue
			}
		}
		var end, next int
		override := ""
		dropped := false
		switch {
		case r.Rule == breakWord && r.Edge && (r.isBreak(t[f]) || s.isBound(f)):
			end, next = f, f
			if r.isBreak(t[f]) && !r.Lossless {
				next, dropped = f+1, true
			}
		case k >= i || b > i:
			if b > k {
				end, next = b, b
			} else {
				end, next, dropped = k, k+1, !r.Lossless
				if r.Lossless {
					end = k + 1
				}
			}
		case r.Rule == breakChar || r.Overflow == overflowCut:
			end, next = f, f
		default:
			e := s.wordEnd(f)
			end, next, dropped = e, e+1, e < len(t)
			if r.Overflow == overflowDump && s.box.Measure(t[i:e]) <= limit {
				end = len(t)
				if s.virtual {
					end--
					override = s.raw
				}
				next = len(t) + 1
			}
		}
		if r.Trim {
			for end > i && r.isBreak(t[end-1]) {
				end--
			}
		}
		out = append(out, span{a: i, b: end, indent: indent, first: first, text: override})
		i = next
		// A break byte that ended the text leaves one empty word after it.
		if dropped && i == len(t) && !r.SkipLead && !r.Lossless {
			out = append(out, span{a: i, b: i, indent: s.box.Indent, closed: true})
			break
		}
		if r.SkipLead && i < len(t) {
			i = s.skip(i)
		}
		if s.virtual {
			s.virtual = false
			s.t = s.t[:len(s.t)-1]
		}
	}
	if n := len(out); n > 0 {
		out[n-1].last = true
	}
	return out
}

// wordEnd is where the word that holds or follows at ends.
func (s *scan) wordEnd(at int) int {
	for at < len(s.t) && !s.r.isBreak(s.t[at]) {
		at++
	}
	return at
}

// balanced splits one text into two rows, box.Rows wide, at a space; none when
// it fits box.Width whole or no split fits.
func (r lineRule) balanced(box lineBox, t string) []brokenLine {
	if box.Measure == nil || len(box.Rows) < 2 || box.Measure(t) <= box.Width {
		return nil
	}
	best, found, at := 0, false, 0
	for i := 1; i < len(t)-1; i++ {
		if !r.isBreak(t[i]) {
			continue
		}
		w1, w2 := box.Measure(t[:i]), box.Measure(t[i+1:])
		if w1 > box.Rows[0] || w2 > box.Rows[1] {
			continue
		}
		if wide := max(w1, w2); !found || wide < best {
			best, found, at = wide, true, i
		}
	}
	if !found {
		return nil
	}
	return []brokenLine{
		{Runs: []inkRun{{Text: t[:at]}}, First: true, From: 0, To: at},
		{Runs: []inkRun{{Text: t[at+1:]}}, From: at + 1, To: len(t)},
	}
}

// lineWords are the words of one line: its break-separated pieces.
func lineWords(s string) []string { return strings.Fields(s) }

// gapWords is the words of one drawn line, split at spaces and tabs only.
func gapWords(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == '\t' })
}

// collapseGaps makes every run of white space one space and trims the ends.
func collapseGaps(s string) string { return strings.Join(strings.Fields(s), " ") }

// Discretionary hyphens.

func softLetter(b byte) bool {
	return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= 0x80 && b <= 0xaf || b >= 0xe0 && b <= 0xef
}

// softWords are a paragraph's words with every "letter-, space, letter" joined
// into one word, and the places in the joined words where the hyphen stood.
func softWords(s string) (words []string, breaks []int) {
	start := 0 // where the last word stands in the joined words
	for _, field := range strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == '\t' }) {
		if n := len(words); n > 0 {
			last := words[n-1]
			if m := len(last); m > 1 && last[m-1] == '-' && softLetter(last[m-2]) && softLetter(field[0]) {
				words[n-1] = last[:m-1] + field
				breaks = append(breaks, start+m-1)
				continue
			}
			start += len(last) + 1
		}
		words = append(words, field)
	}
	return words, breaks
}

// softGate reports whether s has a discretionary hyphen at all.
func softGate(s string) bool {
	for i := 1; i+2 < len(s); i++ {
		if s[i] != '-' || !softLetter(s[i-1]) || s[i+1] != ' ' && s[i+1] != '\t' {
			continue
		}
		j := i + 1
		for j < len(s) && (s[j] == ' ' || s[j] == '\t') {
			j++
		}
		if j < len(s) && softLetter(s[j]) {
			return true
		}
	}
	return false
}
