package game

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type messageTag struct {
	Open, Close, End int
	Number           int
	Tag, Body        string
}

func (tag messageTag) text() (prefix, body, suffix string) {
	body = strings.TrimPrefix(strings.TrimPrefix(tag.Body, "\r\n"), "\n")
	prefix = tag.Body[:len(tag.Body)-len(body)]
	if strings.HasSuffix(body, "\r\n") {
		body, suffix = strings.TrimSuffix(body, "\r\n"), "\r\n"
	} else if strings.HasSuffix(body, "\r") {
		body, suffix = strings.TrimSuffix(body, "\r"), "\r"
	}
	return prefix, body, suffix
}

type messageMarkupDocument struct {
	Raw  string
	Tags []messageTag
}

func writeMessageTag(out *strings.Builder, tag, body string) {
	out.WriteByte('<')
	out.WriteString(tag)
	out.WriteByte('>')
	out.WriteString(body)
}

func decodeMessageMarkup(raw string) messageMarkupDocument {
	d := messageMarkupDocument{Raw: raw}
	for i := 0; i < len(raw); {
		open := strings.IndexByte(raw[i:], '<')
		if open < 0 {
			break
		}
		open += i
		close := strings.IndexByte(raw[open:], '>')
		if close < 0 {
			break
		}
		close += open
		end := len(raw)
		if next := strings.IndexByte(raw[close+1:], '<'); next >= 0 {
			end = close + 1 + next
		}
		tag := raw[open+1 : close]
		number := -1
		if start, end, ok := messageAttribute(tag, partMark); ok && end > start {
			if n, err := strconv.Atoi(tag[start:end]); err == nil {
				number = n
			}
		}
		d.Tags = append(d.Tags, messageTag{open, close, end, number, tag, raw[close+1 : end]})
		i = close + 1
	}
	return d
}

func messageAttribute(tag, key string) (start, end int, ok bool) {
	i := indexFold(tag, key)
	if i < 0 {
		return 0, 0, false
	}
	start = i + len(key)
	end = start
	for end < len(tag) && tag[end] >= '0' && tag[end] <= '9' {
		end++
	}
	return start, end, true
}

// setMessageAttribute is tag with key set to value, added after a comma when
// absent; a negative value cuts key, its digits and one separator beside them.
func setMessageAttribute(tag, key string, value int) string {
	start, end, ok := messageAttribute(tag, key)
	if ok {
		if value < 0 {
			return cutTagWord(tag, start-len(key), end-start+len(key))
		}
		return tag[:start] + strconv.Itoa(value) + tag[end:]
	}
	if value >= 0 {
		return tag + "," + key + strconv.Itoa(value)
	}
	return tag
}

func (d messageMarkupDocument) encode() string {
	var out strings.Builder
	at := 0
	for _, tag := range d.Tags {
		out.WriteString(d.Raw[at:tag.Open])
		writeMessageTag(&out, tag.Tag, tag.Body)
		at = tag.End
	}
	out.WriteString(d.Raw[at:])
	return out.String()
}

func messagePlainText(raw string) string {
	d := decodeMessageMarkup(raw)
	var out strings.Builder
	at := 0
	for _, tag := range d.Tags {
		out.WriteString(raw[at:tag.Open])
		at = tag.Close + 1
	}
	out.WriteString(raw[at:])
	return out.String()
}

// messagePageEdit is one part tag the writer writes: its source tag, or a
// new one for Source -1. Tag, when set, is the tag's text in place of its
// source's, or a new tag's whose number the writer sets; Separator is a new
// tag's line break before its text, "\n" when empty.
type messagePageEdit struct {
	Source, Number int
	Body           string
	Speaker        int64
	Tag, Separator string
}

// setHeroArms is tag naming exactly the hero literals lits among the four
// of eventArms: one it names and lits lacks is cut, with one separator
// beside it; one lits names and it lacks is added after the tag's own
// separator, a comma unless the tag parts its words by spaces alone.
func setHeroArms(tag string, lits ...string) string {
	for _, arm := range eventArms[:4] {
		for !slices.Contains(lits, arm.lit) {
			i := indexFold(tag, arm.lit)
			if i < 0 {
				break
			}
			tag = cutTagWord(tag, i, len(arm.lit))
		}
	}
	sep := ","
	if !strings.Contains(tag, ",") && strings.Contains(tag, " ") {
		sep = " "
	}
	for _, lit := range lits {
		if indexFold(tag, lit) < 0 {
			tag += sep + lit
		}
	}
	return tag
}

// cutTagWord is tag with its n bytes at i cut, and one comma or space beside
// them.
func cutTagWord(tag string, i, n int) string {
	switch {
	case i > 0 && (tag[i-1] == ',' || tag[i-1] == ' '):
		i, n = i-1, n+1
	case i+n < len(tag) && (tag[i+n] == ',' || tag[i+n] == ' '):
		n++
	}
	return tag[:i] + tag[i+n:]
}

func (d messageMarkupDocument) pages() []int {
	var out []int
	for i, tag := range d.Tags {
		if tag.Number >= 0 {
			out = append(out, i)
		}
	}
	return out
}

func (d messageMarkupDocument) write(pages []messagePageEdit, renumber bool) (string, error) {
	indices := d.pages()
	var out strings.Builder
	prefix := ""
	suffixes := map[int]string{}
	kept := map[int]bool{}
	for _, page := range pages {
		kept[page.Source] = true
	}
	if len(indices) > 0 {
		prefix = d.Raw[:d.Tags[indices[0]].Open]
	}
	for i, index := range indices {
		end := len(d.Raw)
		if i+1 < len(indices) {
			end = d.Tags[indices[i+1]].Open
		}
		suffixes[index] = d.Raw[d.Tags[index].End:end]
	}
	previous := -1
	for _, index := range indices {
		if kept[index] {
			previous = index
		} else if previous >= 0 {
			suffixes[previous] += suffixes[index]
		} else {
			prefix += suffixes[index]
		}
	}
	out.WriteString(prefix)
	numbers := map[int]int{}
	seen := map[int]bool{}
	for i, page := range pages {
		number := page.Number
		if renumber {
			if _, ok := numbers[number]; !ok {
				numbers[number] = len(numbers) + 1
			}
			number = numbers[number]
		}
		tag, separator, suffix, ending := "part="+strconv.Itoa(number), "\n", "", ""
		if page.Source < 0 && page.Tag != "" {
			tag = setMessageAttribute(page.Tag, partMark, number)
		}
		if page.Source < 0 && page.Separator != "" {
			separator = page.Separator
		}
		if page.Source >= 0 {
			if page.Source >= len(d.Tags) || d.Tags[page.Source].Number < 0 || seen[page.Source] {
				return "", fmt.Errorf("invalid message page %d", page.Source)
			}
			seen[page.Source] = true
			original := d.Tags[page.Source]
			tag = original.Tag
			if page.Tag != "" {
				tag = page.Tag
			}
			separator, _, ending = original.text()
			suffix = suffixes[page.Source]
			if renumber {
				tag = setMessageAttribute(tag, partMark, number)
			}
		}
		oldSpeaker := int64(-1)
		if speaker, named := tagSpeaker(tag); named {
			oldSpeaker = int64(speaker)
		}
		if oldSpeaker != page.Speaker {
			tag = setMessageAttribute(tag, speakerTag, int(page.Speaker))
		}
		if renumber && i+1 < len(pages) && ending == "" {
			ending = "\r\n"
		}
		writeMessageTag(&out, tag, separator+page.Body+ending)
		out.WriteString(suffix)
	}
	return out.String(), nil
}
