package ui

import "strconv"

// GroupDigits formats n in decimal with a comma before every three digits
// counted from the right: the original's grouping routine (TOWN-469,
// SHOP-052). A leading sign is kept in front with no comma after it.
func GroupDigits(n int64) string {
	return groupDecimal(strconv.FormatInt(n, 10))
}

// GroupSigned is the generator's `%+d` text through the same routine: a
// non-negative value carries `+` (TOWN-469).
func GroupSigned(n int64) string {
	s := strconv.FormatInt(n, 10)
	if n >= 0 {
		s = "+" + s
	}
	return groupDecimal(s)
}

func groupDecimal(s string) string {
	sign := ""
	if s != "" && (s[0] == '-' || s[0] == '+') {
		sign, s = s[:1], s[1:]
	}
	out := make([]byte, 0, len(s)+len(s)/3)
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	return sign + string(out)
}
