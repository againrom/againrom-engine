# 1011 — spec (as-built)

## Subject

The shop's tip widget: what loads it, where it draws, how its text wraps, and how it shares its
own rectangle with the pre-existing message strip.

## Load

`ReadShopTip` (`pkg/game/shoptip.go`) reads one addressed file whole through `entrySource` and
reports whether it was found. It never splits the payload into lines: `MISSION-DOC-021` decodes the
reader the tip goes through, `R1293`, as opening a node, reading its whole length and
appending one NUL — no CR-consuming line grammar of the kind `installtext.go`'s `TextTable`
reproduces for `main.txt`/`dialogs.txt`. A missing file, or a nil source, answers `false`.

`townScreen.Choose` (`pkg/game/townscreen.go`), on entering the shop room (`case roomShop:`), reads
`main/text/tips/shop1.txt` (`ShopTip1Path`) into `t.shopTip`. This runs on every entry, with no
cache and no cleared-on-exit rule of its own: `t.shopTip` behaves exactly as every other
`townScreen` field the shop screen projects — always current for whichever room the caller is
currently showing, and stale (harmlessly, since nothing reads it) once the caller leaves the shop.
`ShopScreen()` (`pkg/game/shopview.go`) carries the value out as `ShopScreenView.Tip`.

`main/text/tips/shop2.txt` (`ShopTip2Path`) is named but never read by this build. `SHOP-TIP-045`
decodes a second arm, `R1760`, that replaces the widget's text with `shop2.txt` at most once,
behind a latch, reached from the shop view's own `0x402` message arm — but not what user action
raises that message. Implementing a guessed trigger risked wiring an arbitrary action to the wrong
effect, so the swap is not built; a player sees `shop1.txt` for the life of the widget on every
visit. Recorded as `DIV-132`.

## Gate

`SHOP-TIP-045` states the widget's whole construction is gated on global `[L03631]`; the
global's setter is not read (the claim's own stated Unknown). This build treats the gate as always
open: `t.shopTip` is set on every shop entry regardless of any toggle. An install shipping
`shop1.txt` shows it on every visit; one shipping none draws nothing. Recorded as `DIV-132`.

## Placement and wrap

`shopTipRect` (`pkg/ui/shopscreen.go`) is panel-relative (0,162,312,298) inside the merchant panel
and so view-relative (164,162,476,298) on the composed frame — `SHOP-TIP-045`'s own decoded
rectangle, 312 by 136 pixels. `drawShopTip` is called in `ComposeShopScreen` right after the
merchant panel's own picture blit, before the shelf, table and pack cell loops — the same layer as
every other piece of room furniture, painted once per frame with no gate of its own.

`wrapShopTip` breaks the text greedily on whitespace, measuring each candidate line against
`v.Font.Measure` exactly as the pre-existing `drawShopMessage` already measures its own single
line. A run with no whitespace at all, or a single word wider than the rect, is returned as one
line rather than cut mid-glyph — no hyphenation rule is reproduced or authored. `"\r\n"` in the
payload is folded to `"\n"` and treated as a paragraph break, the same CRLF fold
`pkg/game/frontend.go`'s own `npcnames.txt` reader already applies to a different install text
file. `TEXT-API-007` establishes the original's own wrap pass, `R0727`, is built on the
font's own measurer and states no exact break rule; the greedy word-wrap above is authored against
that gap. `DLG-WRAP-009`'s one concrete number, the wrapping window's own default line pitch (font
height plus 2), is the pitch `drawShopTip` uses; no other constant from that claim is taken.
Recorded as `DIV-133`.

## Layering against the message strip

`shopMessageRect` (169,283)-(476,298) is a pre-existing widget — the answer to the last shop
press — already documented, before this story, as sitting at the tip rectangle's own bottom 15
rows (the comment cites `SHOP-TIP-045` and predates this story's own code). `drawShopTip` stops
before `shopMessageRect.Min.Y` rather than at `shopTipRect.Max.Y`: the tip paints in 121 of its 136
decoded rows, never the bottom 15.

This was tried the other way first, painting the full 136 rows and relying on `drawShopMessage`
being called later in `ComposeShopScreen` to overwrite the strip. It produced visible bleed-through:
the message text is short and horizontally centred inside a 307-pixel-wide strip, so only the
pixels under its own glyphs were overwritten, leaving tip ink showing through the wide margins
around it — worse than the plain floor that stood there before this story. Clipping before the
strip instead keeps the message affordance pixel-identical to the build before this story, whether
or not a tip is showing above it. No claim states how ROM1's own message-line affordance, if any,
shares this space with the tip widget; the layering rule is authored. Recorded as `DIV-133`.

## Mutation evidence

`pkg/ui/shopscreen_test.go`'s `TestDrawShopTipStopsBeforeTheMessageStrip` and
`TestTheAnswerMessageIsUnaffectedByTheTip` both assert against this clip bound. Reverting
`drawShopTip`'s clip condition from `y+font.Height() > shopMessageRect.Min.Y` to
`y+font.Height() > shopTipRect.Max.Y` (the naive full-height version tried first) turns both tests
red: the first because a long tip then paints past where it asserts it stops, the second because a
tip long enough to reach the strip then shows through the message's own unpainted margins. Reverted
back after confirming the failure.

## Cut list

- `SC-1`: `shop2.txt` is never read. The swap's own trigger (`0x402`) is not traced by any claim
  read for this story (`DIV-132`).
- `SC-2`: the widget's construction gate (`[L03631]`) is not reproduced; the gate is treated as
  always open (`DIV-132`).
- `SC-3`: the widget's own bottom 15 of 136 decoded rows are never painted, to avoid contending
  with the pre-existing message strip for the same pixels (`DIV-133`).
- `SC-4`: the wrap break rule is authored, not decoded (`DIV-133`).

## Divergence disposition

`DIV-132` and `DIV-133` are both spent, added to "Authored where research is silent" as `UNKNOWN`.
`DIV-018` moves to `DIVERGENCES-CLOSED.md`: both of its named subjects (the shelf animation, drawn
statically by 1009; the tip widget, drawn by this story) are now implemented. The shelf's own
residual frame-cadence gap continues under its own successor row, `DIV-127`.
