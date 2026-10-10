package game

// The shop's own tip widget (1011 spec; `SHOP-TIP-045`).
//
// `SHOP-TIP-045` decodes the load and the placement: activation constructs a
// widget of a different class than the merchant panel's own children, at
// merchant-panel-relative (0,162,312,298) — view-relative
// (164,162,476,298) — holding the whole of `main/text/tips/shop1.txt`, and
// hands it to the merchant panel as a child. Both texts are the shop's tip in
// the ROM1 description.

// ReadShopTip reads one tip file whole and reports whether it is there.
//
// THE FILE IS READ WHOLE, NOT SPLIT INTO LINES. Unlike `TextTable` (
// installtext.go), which reproduces a DIFFERENT loader's own CR-consuming
// line walk for `main.txt`/`dialogs.txt`, this format carries no per-line
// structure of its own — wrapping it into a widget's own lines is a layout
// pass over the whole string, not a split on a shipped delimiter
// (`DIV-133`).
//
// A missing file answers false and draws nothing, matching every other
// install text reader in this tree: a widget with nothing to show is a widget
// that is not there.
func ReadShopTip(src entrySource, addr string) (string, bool) {
	if src == nil {
		return "", false
	}
	b, err := src.ReadFile(addr)
	if err != nil {
		return "", false
	}
	return string(b), true
}
