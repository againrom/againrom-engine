package ui

import (
	"image"
	"strings"

	"againrom/pkg/render/text"
)

type tipWord struct {
	text   string
	breaks []int
}

func tipLetter(b byte) bool {
	return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= 0x80 && b <= 0xaf || b >= 0xe0 && b <= 0xef
}

func tipParagraphWords(s string) []tipWord {
	fields := strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == '\t' })
	words := make([]tipWord, 0, len(fields))
	for _, field := range fields {
		if len(words) > 0 {
			last := &words[len(words)-1]
			n := len(last.text)
			if n > 1 && last.text[n-1] == '-' && tipLetter(last.text[n-2]) && tipLetter(field[0]) {
				last.text = last.text[:n-1] + field
				last.breaks = append(last.breaks, n-1)
				continue
			}
		}
		words = append(words, tipWord{text: field})
	}
	return words
}

func tipWordsText(words []tipWord) string {
	parts := make([]string, len(words))
	for i, w := range words {
		parts[i] = w.text
	}
	return strings.Join(parts, " ")
}

func tipWordBreak(f *text.Font, prefix string, word tipWord, width int) int {
	best := 0
	for _, cut := range word.breaks {
		if f.Advance(prefix+word.text[:cut]+"- ") < width {
			best = cut
		}
	}
	return best
}

func tipWordTail(word tipWord, cut int) tipWord {
	out := tipWord{text: word.text[cut:]}
	for _, at := range word.breaks {
		if at > cut {
			out.breaks = append(out.breaks, at-cut)
		}
	}
	return out
}

func tipDiscretionary(s string) bool {
	for i := 1; i+2 < len(s); i++ {
		if s[i] != '-' || !tipLetter(s[i-1]) || s[i+1] != ' ' && s[i+1] != '\t' {
			continue
		}
		j := i + 1
		for j < len(s) && (s[j] == ' ' || s[j] == '\t') {
			j++
		}
		if j < len(s) && tipLetter(s[j]) {
			return true
		}
	}
	return false
}

func tipTextLines(f *text.Font, s string, width int) []dialogueLine {
	if f == nil || len(f.Glyphs) == 0 || width <= 0 || s == "" {
		return nil
	}
	if !tipDiscretionary(s) {
		return dialogueWrap(f, s, width)
	}
	var out []dialogueLine
	appendLine := func(s string, end bool) {
		out = append(out, dialogueLine{text: s, paragraphEnd: end})
	}
	for remaining := s; remaining != ""; {
		end := strings.Index(remaining, "\r\n")
		para := remaining
		remaining = ""
		if end >= 0 {
			para, remaining = para[:end], strings.TrimLeft(para[end+2:], dialogueTrimBytes)
		}
		words := tipParagraphWords(strings.Trim(para, dialogueTrimBytes))
		discretionary := false
		for _, word := range words {
			discretionary = discretionary || len(word.breaks) != 0
		}
		if !discretionary {
			if strings.Trim(para, dialogueTrimBytes) == "" {
				appendLine("", false)
				continue
			}
			for _, ln := range dialogueWrap(f, para, width) {
				appendLine(ln.text, ln.paragraphEnd)
			}
			continue
		}
		for len(words) > 0 {
			rest := tipWordsText(words)
			if f.Advance(rest) < width {
				appendLine(rest+" \r", true)
				break
			}
			n := 0
			for n < len(words) && f.Advance(tipWordsText(words[:n+1])+" ") < width {
				n++
			}
			prefix := ""
			if n > 0 {
				prefix = tipWordsText(words[:n]) + " "
			}
			if n < len(words) {
				if cut := tipWordBreak(f, prefix, words[n], width); cut > 0 {
					appendLine(prefix+words[n].text[:cut]+"- ", false)
					words[n] = tipWordTail(words[n], cut)
					words = words[n:]
					continue
				}
			}
			if n == 0 {
				if f.Advance(words[0].text) > width {
					appendLine(words[0].text, false)
					words = words[1:]
					continue
				}
				appendLine(rest, false)
				break
			}
			appendLine(prefix, false)
			words = words[n:]
		}
	}
	for i := range out {
		out[i].indent = i == 0 || out[i-1].paragraphEnd
		out[i].justify = i < len(out)-1 && !out[i].paragraphEnd
	}
	return out
}

func drawTipText(dst *image.RGBA, f *text.Font, lines []dialogueLine, r image.Rectangle) {
	if f == nil {
		return
	}
	pitch := f.Height() + 2
	indent := 0
	if glyph := f.GlyphFor('@'); glyph != nil {
		indent = glyph.Advance
	}
	for i, ln := range lines {
		y := r.Min.Y + i*pitch
		if pitch <= 0 || y+f.Height() > r.Max.Y {
			break
		}
		x, width := r.Min.X, r.Dx()
		if ln.indent {
			x, width = x+indent, width-indent
		}
		drawDialogueLine(dst, f, ln, 0, x, y, width, DialogueArithmetic{}, dialogueButtonInk)
	}
}
